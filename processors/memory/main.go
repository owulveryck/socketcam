package memory

import (
	"context"
	"encoding/json"
	"log"

	"github.com/kelseyhightower/envconfig"
	"github.com/owulveryck/cortical"
)

type configuration struct {
	Path string `default:"/tmp/training"`
}

var (
	c      chan []byte
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

//Memory is implementing the cortical.Cortex interface
type Memory struct{}

// NewCortex is filling the  ...
func (r *Memory) NewCortex(ctx context.Context) (cortical.GetInfoFromCortexFunc, cortical.SendInfoToCortex) {
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
	}
}

// Send to the websocket
func (t *classifier) Send(ctx context.Context) chan []byte {
	return t.c
}

func init() {
	err := envconfig.Process("MEMORY_CORTEX", &config)
	if err != nil {
		log.Fatal(err.Error())
	}
	//defer session.Close()
	c = make(chan []byte)

}
