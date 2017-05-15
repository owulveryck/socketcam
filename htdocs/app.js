'use strict';
window.addEventListener("load", function(evt) {
  var output = document.getElementById("output");
  var input = document.getElementById("input");
  var ws;

  var video = document.querySelector('video');
  var canvas;

  var print = function(message) {
    var d = document.createElement("div");
    d.innerHTML = message;
    output.appendChild(d);
  };

  var loc = window.location, new_uri;
  if (loc.protocol === "https:") {
    new_uri = "wss:";
  } else {
    new_uri = "ws:";
  }
  new_uri += "//" + loc.host;
  new_uri += loc.pathname + "ws";
  ws = new WebSocket(new_uri);
  ws.onopen = function(evt) {
    print("OPEN");
    takeSnapshot();
  }
  ws.onclose = function(evt) {
    print("CLOSE");
    ws = null;
  }
  ws.onmessage = function(evt) {
    print("RESPONSE: " + evt.data);
  }
  ws.onerror = function(evt) {
    print("ERROR: " + evt.data);
  }

  /**
   *  generates a still frame image from the stream in the <video>
   *  appends the image to the <body>
   */
  function takeSnapshot() {
    var img = document.querySelector('img') || document.createElement('img');
    var context;
    var width = video.offsetWidth
      , height = video.offsetHeight;

    canvas = canvas || document.createElement('canvas');
    canvas.width = width;
    canvas.height = height;

    context = canvas.getContext('2d');
    context.drawImage(video, 0, 0, width, height);

    img.src = canvas.toDataURL('image/png');
    //ws.send(canvas.toDataURL('image/png'));
    ws.send(img.src);

    document.body.appendChild(img);
  }

  // use MediaDevices API
  // docs: https://developer.mozilla.org/en-US/docs/Web/API/MediaDevices/getUserMedia
  if (navigator.mediaDevices) {
    // access the web cam
    navigator.mediaDevices.getUserMedia({video: true})
    // permission granted:
      .then(function(stream) {
        video.src = window.URL.createObjectURL(stream);
        video.addEventListener('click', takeSnapshot);
      })
    // permission denied:
      .catch(function(error) {
        document.body.textContent = 'Could not access the camera. Error: ' + error.name;
      });
  }

});
