'use strict';

// *************************************************
// Definitions
// *************************************************
var recognition = new webkitSpeechRecognition();
var video = document.querySelector('video');
var canvas;
var listening=false;
var recognizing=true;
var ws
// Connecting the websocket
var loc = window.location, new_uri;
if (loc.protocol === "https:") {
  new_uri = "wss:";
} else {
  new_uri = "ws:";
}
new_uri += "//" + loc.host + "/ws";
ws = new WebSocket(new_uri);

function Message(trigger, desc, messageType, data){
  this.action = trigger; // may be: ear, eye, skin
  this.desc = desc;
  this.messageType = messageType;
  this.data = data;
}

var print = function(message) {
  document.getElementById("result").innerHTML= message;
};

// *************************************************
// Defining the behaviour on event of the websocket
// *************************************************
ws.onmessage = function(event) {
  console.log("Received:" + event.data);
  console.log("Now speaking, stopping recognition.");
  print(event.data);
  recognizing=false;
  var msg = new SpeechSynthesisUtterance(event.data);
  window.speechSynthesis.speak(msg);
  console.log("Starting recognition again.");
  recognizing=true;
};


// *************************************************
// Ear processing
// *************************************************
recognition.continuous = true;
recognition.interimResults = false;
recognition.onresult = function(event) { 
  var final_transcript="";
  if (recognizing == true ) {
    for (var i = event.resultIndex; i < event.results.length; ++i) {
      if (event.results[i].isFinal) {
        final_transcript += event.results[i][0].transcript;
        console.log("FINAL TRANSCRIPTION:")
        console.log(final_transcript);
        if (listening == true){
          ws.send(final_transcript);
          listening=false;
        }
        else {
          if (final_transcript.includes("Jarvis"))
          {
            console.log("Keyword Jarvis detected");
            listening=true;
            ws.send(final_transcript);  
          }
        }
      }
    }
  }
};

// *************************************************
// Eye processing
// *************************************************
function takeSnapshot() {
  var context;
  var width = video.offsetWidth
    , height = video.offsetHeight;

  canvas = canvas || document.createElement('canvas');
  canvas.width = width;
  canvas.height = height;

  context = canvas.getContext('2d');
  context.drawImage(video, 0, 0, width, height);

  var dataURI = canvas.toDataURL('image/jpeg')
  var byteString = dataURI.split(',')[1];

  // separate out the mime component
  var mimeString = dataURI.split(',')[0].split(':')[1].split(';')[0]
  //
  var message = {"dataURI":{}};
  message.dataURI.contentType = mimeString;
  message.dataURI.content = byteString;
  var json = JSON.stringify(message);
  ws.send(json);
  console.log("message sent");
}

// *************************************************
// Activate the senses
// *************************************************
// use MediaDevices API
// docs: https://developer.mozilla.org/en-US/docs/Web/API/MediaDevices/getUserMedia
if (navigator.mediaDevices) {
  // access the web cam
  var front = false;
  document.getElementById('flip-button').onclick = function() { front = !front; };

  var constraints = { video: { facingMode: (front? "user" : "environment") } };
  navigator.mediaDevices.getUserMedia(constraints)
  // permission granted:
    .then(function(stream) {
      video.src = window.URL.createObjectURL(stream);
      video.addEventListener('click', takeSnapshot);
      // setInterval(takeSnapshot,3000);
    })
  // permission denied:
    .catch(function(error) {
      document.body.textContent = 'Could not access the camera. Error: ' + error.name;
    });
}
recognition.start();
