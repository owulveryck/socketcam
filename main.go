package main

import (
	rice "github.com/GeertJohan/go.rice"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/kelseyhightower/envconfig"
	"github.com/owulveryck/cortical"
	//"github.com/owulveryck/socketcam/processors/tensorflow"
	"log"
	"net/http"

	"github.com/owulveryck/socketcam/processors/rekognition"
	"github.com/phyber/negroni-gzip/gzip"
	"github.com/urfave/negroni"
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
	//d1 := dummy.New()
	//d2 := dummy.New()
	//d3 := dummy.New()
	//d4 := dummy.New()
	brain := &cortical.Cortical{
		Upgrader: websocket.Upgrader{},
		Cortexes: []cortical.Cortex{&rekognition.Rekognition{}},
	}

	router := mux.NewRouter().StrictSlash(true)

	router.
		Methods("GET").
		Path("/ws").
		Name("Communication Channel").
		HandlerFunc(brain.ServeWS)

	router.
		Methods("GET").
		PathPrefix("/").
		Name("Static").
		//Handler(http.FileServer(http.Dir("./htdocs")))
		Handler(http.FileServer(rice.MustFindBox("htdocs").HTTPBox()))
	n := negroni.Classic()
	n.Use(gzip.Gzip(gzip.DefaultCompression))

	n.UseHandler(router)
	if config.Scheme == "https" {
		log.Fatal(http.ListenAndServeTLS(config.ListenAddress, config.Certificate, config.PrivateKey, n))

	} else {
		log.Fatal(http.ListenAndServe(config.ListenAddress, n))

	}
}
