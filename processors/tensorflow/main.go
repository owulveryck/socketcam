package tensorflow

import (
	"cloud.google.com/go/vision"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/kelseyhightower/envconfig"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"io/ioutil"
	"log"
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
	GVision       bool   `default:"false"`
	TFVision      bool   `default:"true"`
	Scheme        string `default:"http"`
	ListenAddress string `default:":8080"`
	PrivateKey    string `default:"ssl/server.key"`
	Certificate   string `default:"ssl/server.pem"`
	TFModelDir    string `default:"/tmp/modeldir"`
}
type tfLabel struct {
	Label       string
	Probability float32
}

type message struct {
	Action  string      `json:"action"`
	Message interface{} `json:"message"`
	DataURI struct {
		ContentType string `json:"contentType"`
		Content     []byte `json:"content"`
	} `json:"dataURI"`
}

// Receive is the receiver of event
func Receive(ctx context.Context, b *[]byte) {
	var m message
	err := json.Unmarshal(*b, &m)
	if err != nil {
		return
	}
	if m.DataURI.ContentType == "image/jpeg" {

		data := m.DataURI.Content
		// Run inference on *imageFile.
		// For multiple images, session.Run() can be called in a loop (and
		// concurrently). Alternatively, images can be batched since the model
		// accepts batches of image data as input.
		tensor, err := makeTensorFromImage(data)
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
		label := printBestLabel(probabilities, labelsfile)
		buff.Write([]byte(fmt.Sprintf("%v (%2.0f%%)", label.Label, label.Probability*100.0)))
	}

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

}
