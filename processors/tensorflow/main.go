package tensorflow

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"github.com/owulveryck/cortical"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"io/ioutil"
	"log"
)

var (
	config                configuration
	modelfile, labelsfile string
	session               *tf.Session
	graph                 *tf.Graph
	c                     chan []byte
)

type configuration struct {
	TFModelDir string `default:"/tmp/modeldir"`
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

// Tensorflow is filling the cortical.Cortex interface
type Tensorflow struct{}

// NewCortex is filling the  ...
func (t *tensorflow) NewCortex(ctx context.Context) (cortical.GetInfoFromCortexFunc, cortical.SendInfoToCortex) {
	c := make(chan []byte)
	class := &classifier{
		c: c,
	}
	return class.Send, class.Receive
}

type classifier struct {
	c chan []byte
}

// Receive is the receiver of event
func (t *classifier) Receive(ctx context.Context, b *[]byte) {
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
		t.c <- []byte(fmt.Sprintf("%v (%2.0f%%)", label.Label, label.Probability*100.0))
	}
}

// Send to the websocket
func (t *classifier) Send(ctx context.Context) chan []byte {
	return t.c
}

func init() {

	// Default values
	err := envconfig.Process("SOCKETCAM", &config)
	if err != nil {
		log.Fatal(err.Error())
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
	//defer session.Close()
	c = make(chan []byte)

}
