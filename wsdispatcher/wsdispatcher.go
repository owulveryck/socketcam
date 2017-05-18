// Package wsdispatcher is a utility that handles websocket connexions and dispatch
// all the []byte received to the consumers
// It also get all the informations of the producers and sends them back to the websocket
package wsdispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
)

// WSDispatch specifies how to upgrade an HTTP connection to a Websocket connection
// as well as the action to be performed on receive a []byte
type WSDispatch struct {
	Upgrader  websocket.Upgrader
	Senders   []func(ctx context.Context) []byte
	Receivers []func(context.Context, *[]byte)
}

type httpErr struct {
	Msg  string `json:"msg"`
	Code int    `json:"code"`
}

// Sender must implement the send method
type Sender interface {
	Send(stop chan struct{}) chan []byte
}

// Receiver must implement the receive method
type Receiver interface {
	Receive(msg <-chan []byte, stop chan struct{})
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

// ContextKeyType is the type of the key of the context
type ContextKeyType string

// ContextKey is the key name where the session is stored
const ContextKey = "uuid"

// ServeWS is the dispacher function
func (wsd *WSDispatch) ServeWS(w http.ResponseWriter, r *http.Request) {
	ctx := context.WithValue(r.Context(), ContextKeyType(ContextKey), uuid.New().String())
	conn, err := wsd.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		handleErr(w, err, http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	rcvsNum := len(wsd.Receivers)
	sndrsNum := len(wsd.Senders)
	var stop []chan struct{}
	for i := 0; i < sndrsNum+rcvsNum; i++ {
		s := make(chan struct{})
		stop = append(stop, s)
	}
	rcv := make(chan []byte, 1)
	senders := make([]<-chan []byte, sndrsNum)
	chans := fanOut(rcv, rcvsNum, 1)
	for i := 0; i < sndrsNum; i++ {
		senders[i] = send(ctx, stop[i], wsd.Senders[i])
	}
	for i := range chans {
		receive(ctx, chans[i], stop[i+sndrsNum], wsd.Receivers[i])
	}
	done := make(chan struct{}, 1)
	send := merge(done, senders...)
	closed := make(chan struct{}, 2)
	go func() {
		for {
			p := <-send
			err := conn.WriteMessage(websocket.TextMessage, p)
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
					closed <- struct{}{}
					return
				}
				if err == websocket.ErrCloseSent {
					closed <- struct{}{}
					return
				}
				handleErr(w, err, http.StatusInternalServerError)
				continue
			}
		}
	}()
	go func() {
		for {
			MessageType, p, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
					closed <- struct{}{}
					return
				}
				if err == websocket.ErrCloseSent {
					closed <- struct{}{}
					return
				}
				handleErr(w, err, http.StatusInternalServerError)
				continue
			}
			if MessageType != websocket.TextMessage {
				handleErr(w, errors.New("Only text []byte are supported"), http.StatusNotImplemented)
				continue
			}
			rcv <- p
		}
	}()
	<-closed
	done <- struct{}{}
	for i := 0; i < sndrsNum+rcvsNum; i++ {
		stop[i] <- struct{}{}
	}
}

func fanOut(ch <-chan []byte, size, lag int) []chan []byte {
	cs := make([]chan []byte, size)
	for i := range cs {
		// The size of the channels buffer controls how far behind the recievers
		// of the fanOut channels can lag the other channels.
		cs[i] = make(chan []byte, lag)
	}
	go func() {
		for msg := range ch {
			for _, c := range cs {
				c <- msg
			}
		}
		for _, c := range cs {
			// close all our fanOut channels when the input channel is exhausted.
			close(c)
		}
	}()
	return cs
}

func merge(done <-chan struct{}, cs ...<-chan []byte) <-chan []byte {
	var wg sync.WaitGroup
	out := make(chan []byte)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c or done is closed, then calls
	// wg.Done.
	output := func(c <-chan []byte) {
		defer wg.Done()
		for n := range c {
			select {
			case out <- n:
			case <-done:
				return
			}
		}
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func send(ctx context.Context, stop chan struct{}, f func(context.Context) []byte) chan []byte {
	c := make(chan []byte)
	go func() {
		for {
			select {
			case <-stop:
				close(c)
				return
			case c <- f(ctx):
			}
		}
	}()
	return c
}

func receive(ctx context.Context, msg <-chan []byte, stop chan struct{}, f func(context.Context, *[]byte)) {
	go func() {
		for {
			select {
			case b := <-msg:
				f(ctx, &b)
			case <-stop:
				return
			}
		}
	}()
}
