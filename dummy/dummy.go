package dummy

import (
	"fmt"
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
func (r *ReceiverSender) Send() []byte {
	log.Printf("[%v] Sleeping...", r.Ret)
	time.Sleep(time.Duration(r.Ret) * time.Millisecond)
	return []byte(fmt.Sprintf("%v", r.Ret))
}

// Receive ...
func (r *ReceiverSender) Receive(b *[]byte) {
	log.Printf("[%v] received %v", r.Ret, *b)
}
