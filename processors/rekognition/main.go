package rekognition

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/kelseyhightower/envconfig"
	"github.com/owulveryck/cortical"
	"io/ioutil"
	"log"
)

type configuration struct {
	Me map[string]string `required:"true" default:"Olivier Wulveryck:/tmp/me.jpg"`
}

var (
	c      chan []byte
	sess   *session.Session
	svc    *rekognition.Rekognition
	config configuration
)

type message struct {
	Action  string      `json:"action"`
	Message interface{} `json:"message"`
	DataURI struct {
		ContentType string `json:"contentType"`
		Content     []byte `json:"content"`
	} `json:"dataURI"`
}

//Rekognition is implementing the cortical.Cortex interface
type Rekognition struct{}

// NewCortex is filling the  ...
func (r *Rekognition) NewCortex(ctx context.Context) (cortical.GetInfoFromCortexFunc, cortical.SendInfoToCortex) {
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
		//results := make(map[string]float32)
		log.Println("Rekognition")
		for k, v := range config.Me {
			log.Println("%v: %v", k, v)
			// Opening my face
			me, err := ioutil.ReadFile(v)
			if err != nil {
				log.Fatal(err)
			}
			data := m.DataURI.Content
			params := &rekognition.CompareFacesInput{
				SourceImage: &rekognition.Image{ // Required
					Bytes: data,
				},
				TargetImage: &rekognition.Image{ // Required
					Bytes: me,
				},
				SimilarityThreshold: aws.Float64(1.0),
			}
			log.Println("DEBUG: sending params ", params)
			resp, err := svc.CompareFaces(params)

			if err != nil {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
				return
			}

			// Pretty-print the response data.
			for _, face := range resp.FaceMatches {
				if *face.Similarity > 92.0 {
					t.c <- []byte("Salut Olivier Wulveryck")
				} else {
					t.c <- []byte("Bonjour à vous")

				}
			}
			fmt.Println(k)
			fmt.Println(resp)
			//t.c <- []byte(fmt.Sprintf("%v (%2.0f%%)", label.Label, label.Probability*100.0))
		}
	}
}

// Send to the websocket
func (t *classifier) Send(ctx context.Context) chan []byte {
	return t.c
}

func init() {
	err := envconfig.Process("REKOGNITION", &config)
	if err != nil {
		log.Fatal(err.Error())
	}
	sess, err = session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	svc = rekognition.New(sess)

	if err != nil {
		fmt.Println("failed to create session,", err)
		return
	}
	//defer session.Close()
	c = make(chan []byte)

}
