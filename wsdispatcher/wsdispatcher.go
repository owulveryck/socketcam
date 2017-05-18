// Package wsdispatcher is a utility that handles websocket connexions and dispatch
// all the Message received to the consumers
// It also get all the informations of the producers and sends them back to the websocket
package wsdispatcher

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
)

// WSDispatch specifies how to upgrade an HTTP connection to a Websocket connection
// as well as the action to be performed on receive a Message
type WSDispatch struct {
	Upgrader websocket.Upgrader
	//receiver   []<-chan Message
	//sender     chan<- Message
	Processors []func(Message <-chan Message) chan Message
}

type httpErr struct {
	Msg  string `json:"msg"`
	Code int    `json:"code"`
}

// Message ...
type Message []byte

/*
type Message struct {
	Action  string      `json:"action"`
	Message interface{} `json:"Message"`
	DataURI struct {
		ContentType string `json:"contentType"`
		Content     []byte `json:"content"`
	} `json:"dataURI"`
}
*/

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

// ServeWS is the dispacher function
func (wsd *WSDispatch) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsd.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		handleErr(w, err, http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	l := len(wsd.Processors)
	rcv := make(chan Message, 1)
	senders := make([]<-chan Message, l)
	chans := fanOut(rcv, l, 1)
	for i := range chans {
		senders[i] = wsd.Processors[i](chans[i])
	}
	done := make(chan struct{}, 1)
	send := merge(done, senders...)
	closed := make(chan struct{}, 2)
	go func() {
		for {
			p := <-send
			err := conn.WriteMessage(websocket.TextMessage, p)
			if ce, ok := err.(*websocket.CloseError); ok {
				switch ce.Code {
				case websocket.CloseNormalClosure,
					websocket.CloseGoingAway,
					websocket.CloseNoStatusReceived:
					closed <- struct{}{}
					return
				default:
					handleErr(w, err, http.StatusInternalServerError)
					continue

				}
			}
		}
	}()
	go func() {
		for {
			MessageType, p, err := conn.ReadMessage()
			if ce, ok := err.(*websocket.CloseError); ok {
				switch ce.Code {
				case websocket.CloseNormalClosure,
					websocket.CloseGoingAway,
					websocket.CloseNoStatusReceived:
					closed <- struct{}{}
					return
				default:
					handleErr(w, err, http.StatusInternalServerError)
					continue

				}
			}
			if MessageType != websocket.TextMessage {
				handleErr(w, errors.New("Only text Message are supported"), http.StatusNotImplemented)
				continue
			}
			rcv <- p
		}
	}()
	<-closed
	done <- struct{}{}

}

func fanOut(ch <-chan Message, size, lag int) []chan Message {
	cs := make([]chan Message, size)
	for i := range cs {
		// The size of the channels buffer controls how far behind the recievers
		// of the fanOut channels can lag the other channels.
		cs[i] = make(chan Message, lag)
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

func merge(done <-chan struct{}, cs ...<-chan Message) <-chan Message {
	var wg sync.WaitGroup
	out := make(chan Message)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c or done is closed, then calls
	// wg.Done.
	output := func(c <-chan Message) {
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
