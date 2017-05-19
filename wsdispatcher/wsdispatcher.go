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
)

// WSDispatch specifies how to upgrade an HTTP connection to a Websocket connection
// as well as the action to be performed on receive a []byte
type WSDispatch struct {
	Upgrader  websocket.Upgrader
	Senders   []func(ctx context.Context) []byte
	Receivers []func(context.Context, *[]byte)
	Chatter   []func(context.Context) (func(ctx context.Context) []byte, func(context.Context, *[]byte))
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
	for _, chatter := range wsd.Chatter {
		snd, rcv := chatter(ctx)
		if snd != nil {
			wsd.Senders = append(wsd.Senders, snd)
		}
		if rcv != nil {
			wsd.Receivers = append(wsd.Receivers, rcv)
		}
	}
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
