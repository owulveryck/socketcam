// This is a gohperjs example

// go:generate gopherjs build main.go -o app.js -m
// +build ignore

package main

import (
	"fmt"

	"github.com/gopherjs/gopherjs/js"
)

func main() {
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
	videoParams.Width = 1024
	videoParams.Height = 720
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
			fmt.Println("Could not access camera")
		}()
	})
}
