package wsdispatcher

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"time"

	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

var (
	testServer *httptest.Server
	tsURL      *url.URL
	okURL      *url.URL
	koURL      *url.URL
)

func init() {
	router := mux.NewRouter().StrictSlash(true)

	handler := gowmb.CreateHandler(createMessage(), newTag(), "tag")
	router.
		Methods("GET").
		Path("/serveWs/{tag}").
		Name("WebSocket").
		HandlerFunc(handler)
	router.
		Methods("GET").
		Path("/badtag/{tagg}").
		Name("WebSocket").
		HandlerFunc(handler)

	testServer = httptest.NewServer(router) //Creating new server with the user handlers
	tsURL, _ = url.Parse(testServer.URL)
	okURL = &url.URL{Scheme: "ws", Host: tsURL.Host, Path: "/serveWs/1234"}
	koURL = &url.URL{Scheme: "ws", Host: tsURL.Host, Path: "/badtag/1234"}
}

func TestPingPong(t *testing.T) {
	wsURL := url.URL{Scheme: "ws", Host: tsURL.Host, Path: "/serveWs/1234"}
	c, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		t.Errorf("Cannot connect to the websocket %v", err)

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
func TestBadTag(t *testing.T) {

}
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
