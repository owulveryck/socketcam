'use strict';

// *************************************************
// Definitions
// *************************************************
var recognition = new webkitSpeechRecognition();
recognition.lang = "fr-FR";
var video = document.querySelector('video');
var canvas;
var listening=false;
var recognizing=true;
var localstream
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
  //recognizing=false;
  var msg = new SpeechSynthesisUtterance('I see a '+ event.data);
  window.speechSynthesis.speak(msg);
  console.log("Starting recognition again.");
  recognizing=true;
};


function talk(message) {
  var utterance = new SpeechSynthesisUtterance(message);
  utterance.lang = 'fr-FR';
  var voices = window.speechSynthesis.getVoices();
  console.log(voices);
  utterance.voice = voices[1];
  utterance.voiceURI = 'native';
  window.speechSynthesis.speak(utterance);
}

// *************************************************
// Ear processing
// *************************************************
recognition.continuous = true;
recognition.interimResults = true;
recognition.onresult = function(event) { 
  console.log(event);
  var final_transcript="";
  if (recognizing == true ) {
    for (var i = event.resultIndex; i < event.results.length; ++i) {
      if (event.results[i].isFinal) {
        final_transcript += event.results[i][0].transcript;
        console.log("FINAL TRANSCRIPTION:")
        console.log(final_transcript);
        ws.send(final_transcript);
        if (final_transcript.includes("ouvre les yeux")){
          var front = false;
          //document.getElementById('flip-button').onclick = function() { front = !front; };

          var constraints = { video: { facingMode: (front? "user" : "environment") } };
          navigator.mediaDevices.getUserMedia(constraints)
          // permission granted:
            .then(function(stream) {
              localstream = stream
              video.src = window.URL.createObjectURL(stream);
              video.addEventListener('click', takeSnapshot);
              // setInterval(takeSnapshot,3000);
            })
          // permission denied:
            .catch(function(error) {
              document.body.textContent = 'Could not access the camera. Error: ' + error.name;
            });
        }
        if (final_transcript.includes("Salut")){
          talk("salut!");
        }
        if (final_transcript.includes("ferme les yeux")){
          //clearInterval(theDrawLoop);
          //  //ExtensionData.vidStatus = 'off';
          video.pause();
          video.src = "";
          localstream.getTracks()[0].stop();
          console.log("Vid off");
        }
        if (final_transcript.includes("que vois-tu")){
          takeSnapshot();
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
  //document.getElementById('flip-button').onclick = function() { front = !front; };

  var constraints = { video: { facingMode: (front? "user" : "environment") } };
  navigator.mediaDevices.getUserMedia(constraints)
  // permission granted:
  //  .then(function(stream) {
  //    video.src = window.URL.createObjectURL(stream);
  //    video.addEventListener('click', takeSnapshot);
      // setInterval(takeSnapshot,3000);
  //  })
  // permission denied:
    .catch(function(error) {
      document.body.textContent = 'Could not access the camera. Error: ' + error.name;
    });
}
console.log('Listening');
recognition.start();
