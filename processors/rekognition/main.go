package rekognition

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/kelseyhightower/envconfig"
	"github.com/owulveryck/cortical"
)

type configuration struct {
	Me map[string]string `required:"true" default:"Olivier Wulveryck:/tmp/me.jpg"`
}

var (
	c      chan []byte
	sess   *session.Session
	svc    *rekognition.Rekognition
	svcS3  *s3.S3
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
		found := false
		for k, v := range config.Me {
			go func(k, v string) {
				log.Printf("%v: %v", k, v)
				// Is the face hosted on s3?
				var me []byte
				if strings.Contains(v, "s3//") {
					elements := strings.Split(v, "/")
					input := &s3.GetObjectInput{
						Bucket: aws.String(elements[2]),
						Key:    aws.String(filepath.Join(elements[3:]...)),
					}
					log.Println(input)
					result, err := svcS3.GetObject(input)
					if err != nil {
						if aerr, ok := err.(awserr.Error); ok {
							switch aerr.Code() {
							case s3.ErrCodeNoSuchKey:
								fmt.Println(s3.ErrCodeNoSuchKey, aerr.Error())
							default:
								fmt.Println(aerr.Error())
							}
						} else {
							// Print the error, cast err to awserr.Error to get the Code and
							// Message from an error.
							fmt.Println(err.Error())
						}
						return
					}
					me, err = ioutil.ReadAll(result.Body)
					result.Body.Close()
				} else {
					// Opening my face
					me, err = ioutil.ReadFile(v)
				}
				if err != nil {
					log.Println(err)
					return
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
				resp, err := svc.CompareFaces(params)

				if err != nil {
					// Print the error, cast err to awserr.Error to get the Code and
					// Message from an error.
					fmt.Println(err.Error())
					return
				}

				// Pretty-print the response data.
				for _, face := range resp.FaceMatches {
					if *face.Similarity > 70.0 {
						found = true
						t.c <- []byte("Salut " + k)
					}
				}
				fmt.Println(k)
				fmt.Println(resp)
				//t.c <- []byte(fmt.Sprintf("%v (%2.0f%%)", label.Label, label.Probability*100.0))
			}(k, v)
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
	svcS3 = s3.New(sess)

	if err != nil {
		fmt.Println("failed to create session,", err)
		return
	}
	//defer session.Close()
	c = make(chan []byte)

}
