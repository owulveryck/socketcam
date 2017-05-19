package wsdispatcher

import (
	"github.com/gorilla/websocket"
	"time"

	"context"
	"github.com/gorilla/mux"
	"net/http/httptest"
	"net/url"
	"testing"
)

var (
	testServer *httptest.Server
	tsURL      *url.URL
)

// Echo is a dummy type that reads a message, wait for some time and sends ret back
type Echo struct {
	pong string
	c    chan []byte
}

// NewCortex is filling the  ...
func NewCortex(ctx context.Context) (GetInfoFromCortexFunc, SendInfoToCortex) {
	c := make(chan []byte)
	echo := &Echo{
		pong: "pong",
		c:    c,
	}
	return echo.Get, echo.Receive
}

// Get ...
func (e *Echo) Get(ctx context.Context) chan []byte {
	return e.c
}

// Receive ...
func (e *Echo) Receive(ctx context.Context, b *[]byte) {
	e.c <- []byte(e.pong)
}
func init() {
	router := mux.NewRouter().StrictSlash(true)
	wsDsptch := &WSDispatch{
		Upgrader: websocket.Upgrader{},
		Cortexs:  []func(context.Context) (GetInfoFromCortexFunc, SendInfoToCortex){NewCortex},
	}

	router.
		Methods("GET").
		Path("/ws").
		Name("WebSocket").
		HandlerFunc(wsDsptch.ServeWS)

	testServer = httptest.NewServer(router) //Creating new server with the user handlers
	tsURL, _ = url.Parse(testServer.URL)
}

func TestPingPong(t *testing.T) {
	wsURL := url.URL{Scheme: "ws", Host: tsURL.Host, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		t.Fatalf("Cannot connect to the websocket %v", err)

	}
	defer c.Close()
	if err := c.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(2*time.Second)); err != nil {
		t.Errorf("write close: %v", err)
	}
	err = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, ""))
	if err != nil {
		t.Errorf("write close: %v", err)
	}
}

/*
func TestServeWs(t *testing.T) {
	httpURL := url.URL{Scheme: tsURL.Scheme, Host: tsURL.Host, Path: "/serveWs/"}
	// Try to connect to a socket without an ID
	request, err := http.NewRequest("GET", httpURL.String(), nil)

	res, err := http.DefaultClient.Do(request)

	if err != nil {
		t.Error(err)
	}

	// We don't serve the baseurl, a tag is mandatory
	if res.StatusCode != 404 {
		t.Errorf("Success expected: %d", res.StatusCode)
	}

	//Try with a valid tag
	httpURL.Path = "/serveWs/1234'"
	request, err = http.NewRequest("GET", httpURL.String(), nil)

	res, err = http.DefaultClient.Do(request)

	if err != nil {
		t.Error(err)
	}

	// We shall get a bad request as we are expected a websocket
	if res.StatusCode != 200 {
		t.Errorf("Success expected: %d", res.StatusCode)
	}
	// Now test the websocket
	wsURL := url.URL{Scheme: "ws", Host: tsURL.Host, Path: "/serveWs/1234"}
	c, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		t.Errorf("Cannot connect to the websocket %v", err)
	}
	defer c.Close()

	done := make(chan bool)

	go func() {
		defer close(done)
		tm, message, err := c.ReadMessage()
		if err != nil {
			t.Errorf("Error in the message reception: %v (type %v)", err, tm)
		}
		t.Logf("Received message %s of type %v", message, tm)
		done <- true
	}()
	// Sending a message with a Set method that will return success
	type inputOK struct {
		ID int `json:"id"`
	}

	messageOK := &inputOK{ID: 0}
	b, err := json.Marshal(messageOK)
	if err != nil {
		t.Error(err)
	}
	err = c.WriteMessage(websocket.TextMessage, b)
	if err != nil {
		t.Errorf("Cannot write messageOK %v (%v) to websocket: %v", messageOK, b, err)
	}
	// Sending a message with a Set method that will return failure
	type inputKO struct {
		ID string `json:"id"`
	}

	messageKO := &inputKO{ID: "ko"}
	b, err = json.Marshal(messageKO)
	if err != nil {
		t.Error(err)
	}
	err = c.WriteMessage(websocket.TextMessage, b)
	if err != nil {
		t.Errorf("Cannot write messageKO %v (%v) to websocket: %v", messageKO, b, err)
	}
	<-done
	err = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, ""))
	if err != nil {
		t.Errorf("write close: %v", err)
	}
}
*/
