// This is a gohperjs example

// go:generate gopherjs build main.go -o app.js -m
// +build ignore

package main

import "github.com/gopherjs/gopherjs/js"

func main() {
	// Say hello
	utterance := js.Global.Get("SpeechSynthesisUtterance").New()
	utterance.Set("lang", "fr-FR")
	utterance.Set("text", "salut comment ça va?")
	js.Global.Get("window").Get("speechSynthesis").Call("speak", utterance)
	// Listening
	ear := js.Global.Get("webkitSpeechRecognition").New()
	ear.Set("lang", "fr-FR")
	ear.Set("continuous", true)
	ear.Set("interimResults", true)
	ear.Set("onresult", func(event *js.Object) {
		go func() {
			for i := event.Get("resultIndex").Int(); i < event.Get("results").Get("length").Int(); i++ {
				//fmt.Println(event.Get("results").Index(i).Index(0).Get("transcript"))
			}
		}()
	})
	ear.Call("start")
	// turn on the webcam
	video := js.Global.Get("document").Call("querySelector", "video")
	type ctrs struct {
		*js.Object
		Video interface{} `js:"video"`
		Audio bool        `js:"audio"`
	}
	type mediaTrackConstraints struct {
		*js.Object
		Width      int    `js:"width"`
		Height     int    `js:"height"`
		FacingMode string `js:"facingMode"`
	}

	constraints := &ctrs{Object: js.Global.Get("Object").New()}
	videoParams := &mediaTrackConstraints{Object: js.Global.Get("Object").New()}
	//videoParams.Width = 1024
	//videoParams.Height = 720
	videoParams.FacingMode = "environment"

	constraints.Video = videoParams

	senses := js.Global.Get("navigator").Get("mediaDevices").Call("getUserMedia", constraints)
	senses.Call("then", func(stream *js.Object) {
		go func() {
			strObj := js.Global.Get("window").Get("URL").Call("createObjectURL", stream)
			video.Set("src", strObj)
		}()
	})
	senses.Call("catch", func(error *js.Object) {
		go func() {
		}()
	})
}
