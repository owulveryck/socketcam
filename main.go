package main

import (
	"bytes"
	"cloud.google.com/go/vision"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/kelseyhightower/envconfig"
	"github.com/phyber/negroni-gzip/gzip"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/urfave/negroni"
	"github.com/vincent-petithory/dataurl"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

var (
	config                configuration
	upgrader              = websocket.Upgrader{} // use default options
	client                *vision.Client
	modelfile, labelsfile string
	session               *tf.Session
	graph                 *tf.Graph
)

const (
	senseVisison = "vision"
	senseHearing = "hearing"
	senseRading  = "reading"
)

type configuration struct {
	Debug         bool   `default:"true"`
	Scheme        string `default:"http"`
	ListenAddress string `default:":8080"`
	PrivateKey    string `default:"ssl/server.key"`
	Certificate   string `default:"ssl/server.pem"`
	TFModelDir    string `default:"/tmp/modeldir"`
}

type httpErr struct {
	Msg  string `json:"msg"`
	Code int    `json:"code"`
}

type message struct {
	Action  string      `json:"action"`
	Message interface{} `json:"message"`
	DataURI []byte      `json:"data_uri"`
}

func handleErr(w http.ResponseWriter, err error, status int) {
	msg, err := json.Marshal(&httpErr{
		Msg:  err.Error(),
		Code: status,
	})
	if err != nil {
		msg = []byte(err.Error())
	}
	http.Error(w, string(msg), status)
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		handleErr(w, err, http.StatusInternalServerError)
		return
	}
	defer c.Close()
	for {
		mt, r, err := c.NextReader()
		if err != nil {
			handleErr(w, err, http.StatusInternalServerError)
			continue
		}
		if mt != websocket.TextMessage {
			handleErr(w, errors.New("Only text message are supported"), http.StatusNotImplemented)
			continue
		}
		rd, err := process(r)
		if err != nil {
			msg, _ := json.Marshal(&httpErr{
				Msg:  err.Error(),
				Code: http.StatusInternalServerError,
			})
			c.WriteMessage(websocket.TextMessage, msg)
			continue
		}
		cw, err := c.NextWriter(mt)
		if err != nil {
			handleErr(w, err, http.StatusInternalServerError)
			return
		}
		if _, err := io.Copy(cw, rd); err != nil {
			handleErr(w, err, http.StatusInternalServerError)
			return
		}
		if err := cw.Close(); err != nil {
			handleErr(w, err, http.StatusInternalServerError)
			return
		}
	}
}

func process(r io.Reader) (io.Reader, error) {
	var err error
	dataURL, err := dataurl.Decode(r)
	if err != nil {
		return r, err
	}
	if dataURL.ContentType() == "image/jpeg" {

		// Run inference on *imageFile.
		// For multiple images, session.Run() can be called in a loop (and
		// concurrently). Alternatively, images can be batched since the model
		// accepts batches of image data as input.
		tensor, err := makeTensorFromImage(dataURL.Data)
		if err != nil {
			log.Fatal(err)
		}
		output, err := session.Run(
			map[tf.Output]*tf.Tensor{
				graph.Operation("input").Output(0): tensor,
			},
			[]tf.Output{
				graph.Operation("output").Output(0),
			},
			nil)
		if err != nil {
			log.Fatal(err)
		}
		// output[0].Value() is a vector containing probabilities of
		// labels for each image in the "batch". The batch size was 1.
		// Find the most probably label index.
		probabilities := output[0].Value().([][]float32)[0]
		return bytes.NewReader([]byte(printBestLabel(probabilities, labelsfile))), nil

		// For now, only use tensorflow
		if false {
			log.Println("Querying the vision API")
			img, err := vision.NewImageFromReader(ioutil.NopCloser(bytes.NewReader(dataURL.Data)))
			if err != nil {
				log.Println(err)
				return r, err
			}
			ctx := context.Background()
			client, err := vision.NewClient(ctx)
			if err != nil {
				log.Println(err)
				return r, err
			}
			defer client.Close()

			annsSlice, err := client.Annotate(ctx, &vision.AnnotateRequest{
				Image:      img,
				MaxLogos:   100,
				MaxTexts:   100,
				Web:        true,
				SafeSearch: true,
			})
			if err != nil {
				log.Println(err)
				return r, err
			}
			for _, anns := range annsSlice {
				if anns.Web != nil {
					for _, i := range anns.Web.FullMatchingImages {
						log.Println(i.URL)
					}
					for _, i := range anns.Web.PartialMatchingImages {
						log.Println(i.URL)
					}
					for _, i := range anns.Web.PagesWithMatchingImages {
						log.Println(i.URL)
					}
				}
				if anns.Logos != nil {
					fmt.Println("Logos", anns.Logos)
					for _, logo := range anns.Logos {
						log.Println(logo)
					}
				}
				if anns.Texts != nil {
					fmt.Println("Texts", anns.Texts)
				}
				if anns.FullText != nil {
					fmt.Println(anns.FullText.Text)
					return bytes.NewReader([]byte(anns.FullText.Text)), nil
				}
				if anns.Error != nil {
					fmt.Printf("at least one of the features failed: %v", anns.Error)
				}
			}
		}
	}
	return r, nil
}

func main() {

	// Default values
	err := envconfig.Process("SOCKETCAM", &config)
	if err != nil {
		log.Fatal(err.Error())
	}
	if config.Debug {
		log.Printf("==> SCHEME: %v", config.Scheme)
		log.Printf("==> ADDRESS: %v", config.ListenAddress)
		log.Printf("==> PRIVATEKEY: %v", config.PrivateKey)
		log.Printf("==> CERTIFICATE: %v", config.Certificate)
	}

	// Initializing tensorflow
	// Load the serialized GraphDef from a file.
	modelfile, labelsfile, err = modelFiles(config.TFModelDir)
	if err != nil {
		log.Fatal(err)
	}
	model, err := ioutil.ReadFile(modelfile)
	if err != nil {
		log.Fatal(err)
	}

	// Construct an in-memory graph from the serialized form.
	graph = tf.NewGraph()
	if err := graph.Import(model, ""); err != nil {
		log.Fatal(err)
	}

	// Create a session for inference over graph.
	session, err = tf.NewSession(graph, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	router := newRouter()
	n := negroni.Classic()
	n.Use(gzip.Gzip(gzip.DefaultCompression))

	n.UseHandler(router)
	if config.Scheme == "https" {
		log.Fatal(http.ListenAndServeTLS(config.ListenAddress, config.Certificate, config.PrivateKey, n))

	} else {
		log.Fatal(http.ListenAndServe(config.ListenAddress, n))

	}
}

// NewRouter is the constructor for all my routes
func newRouter() *mux.Router {

	router := mux.NewRouter().StrictSlash(true)

	router.
		Methods("GET").
		Path("/ws").
		Name("Communication Channel").
		HandlerFunc(serveWs)

	router.
		Methods("GET").
		PathPrefix("/").
		Name("Static").
		Handler(http.FileServer(http.Dir("./htdocs")))
	return router
}
