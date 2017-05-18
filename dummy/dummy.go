package dummy

import (
	"fmt"
	"github.com/owulveryck/socketcam/wsdispatcher"
	"log"
	"math/rand"
	"time"
)

// ReceiverSender is a dummy type that reads a message, wait for some time and sends ret back
type ReceiverSender struct {
	Ret     int
	message []byte
}

// New ...
func New() *ReceiverSender {
	max := 1500
	min := 1000
	rg := rand.Intn(max-min) + min
	return &ReceiverSender{
		Ret: rg,
	}

}

// Send ...
func (r *ReceiverSender) Send(stop chan struct{}) chan wsdispatcher.Message {
	c := make(chan wsdispatcher.Message)
	go func() {
		for {
			log.Printf("[%v] Sending...", r.Ret)
			c <- []byte(fmt.Sprintf("ping %v", r.Ret))
			log.Printf("[%v] Sleeping...", r.Ret)
			time.Sleep(time.Duration(r.Ret) * time.Millisecond)
		}
	}()
	log.Println("[%v] Send returns", r.Ret)
	return c
}

// Receive ...
func (r *ReceiverSender) Receive(msg <-chan wsdispatcher.Message, stop chan struct{}) {
	go func() {
		for {
			select {
			case <-msg:
				log.Printf("[%v] Received a message", r.Ret)
			case <-stop:
				return
			}
		}
	}()
}
