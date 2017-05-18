package dummy

import (
	"context"
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
func (r *ReceiverSender) Send(ctx context.Context) []byte {
	log.Printf("[%v] Sleeping... (%v)", r.Ret, ctx)
	time.Sleep(time.Duration(r.Ret) * time.Millisecond)
	return []byte(fmt.Sprintf("%v", r.Ret))
}

// Receive ...
func (r *ReceiverSender) Receive(ctx context.Context, b *[]byte) {
	log.Printf("[%v] received (%v)", r.Ret, ctx)
}
