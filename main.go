package main

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/kelseyhightower/envconfig"
	"github.com/owulveryck/socketcam/wsdispatcher"
	"github.com/phyber/negroni-gzip/gzip"
	"github.com/urfave/negroni"
	"log"
	"net/http"
	"time"
)

var (
	config configuration
)

const (
	senseVisison = "vision"
	senseHearing = "hearing"
	senseRading  = "reading"
)

type configuration struct {
	Debug         bool   `default:"true"`
	Scheme        string `default:"http"`
	ListenAddress string `default:":8080"`
	PrivateKey    string `default:"ssl/server.key"`
	Certificate   string `default:"ssl/server.pem"`
}

func main() {

	// Default values
	err := envconfig.Process("SOCKETCAM", &config)
	if err != nil {
		log.Fatal(err.Error())
	}
	if config.Debug {
		log.Printf("==> SCHEME: %v", config.Scheme)
		log.Printf("==> ADDRESS: %v", config.ListenAddress)
		log.Printf("==> PRIVATEKEY: %v", config.PrivateKey)
		log.Printf("==> CERTIFICATE: %v", config.Certificate)
	}
	wsDsptch := &wsdispatcher.WSDispatch{
		Upgrader: websocket.Upgrader{},
		Processors: []func(<-chan wsdispatcher.Message) chan wsdispatcher.Message{
			func(msg <-chan wsdispatcher.Message) chan wsdispatcher.Message {
				c := make(chan wsdispatcher.Message)
				// Write a ping to the websocket every second
				go func() {
					for {
						log.Println("ping 1.5")
						time.Sleep(1500 * time.Millisecond)
						c <- []byte("ping 1.5s")
					}
				}()
				return c
			},
			func(msg <-chan wsdispatcher.Message) chan wsdispatcher.Message {
				c := make(chan wsdispatcher.Message)
				// Write a ping to the websocket every second
				go func() {
					for {
						<-msg
						log.Println("Received a message")
						c <- []byte("message processed")

					}
				}()
				return c
			},
			func(msg <-chan wsdispatcher.Message) chan wsdispatcher.Message {
				c := make(chan wsdispatcher.Message)
				// Write a ping to the websocket every second
				go func() {
					for {
						<-msg
						log.Println("Also Received the message")
						c <- []byte("message processed too")
					}
				}()
				return c
			},
			func(msg <-chan wsdispatcher.Message) chan wsdispatcher.Message {
				c := make(chan wsdispatcher.Message)
				// Write a ping to the websocket every second
				go func() {
					for {
						log.Println("ping")
						time.Sleep(1 * time.Second)
						c <- []byte("ping 1s")
					}
				}()
				return c
			},
		},
	}

	router := mux.NewRouter().StrictSlash(true)

	router.
		Methods("GET").
		Path("/ws").
		Name("Communication Channel").
		HandlerFunc(wsDsptch.ServeWS)

	router.
		Methods("GET").
		PathPrefix("/").
		Name("Static").
		Handler(http.FileServer(http.Dir("./htdocs")))
	n := negroni.Classic()
	n.Use(gzip.Gzip(gzip.DefaultCompression))

	n.UseHandler(router)
	if config.Scheme == "https" {
		log.Fatal(http.ListenAndServeTLS(config.ListenAddress, config.Certificate, config.PrivateKey, n))

	} else {
		log.Fatal(http.ListenAndServe(config.ListenAddress, n))

	}
}
