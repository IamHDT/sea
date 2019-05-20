package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"text/template"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hpcloud/tail"
	"github.com/navigaid/pretty"
	ws "golang.org/x/net/websocket"
)

var vtemplate = `
<!doctype html>
<html>
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Seashells - {{.Id}}</title>
    <link rel="stylesheet" href="../static/vendor/xterm.css">
    <link rel="stylesheet" href="../static/terminal.css">
    <script src="../static/vendor/xterm.js"></script>
    <script src="../static/vendor/fit.js"></script>
    <script src="../static/vendor/encoding-indexes.js"></script>
    <script src="../static/vendor/encoding.js"></script>
    <script>
      (function(i,s,o,g,r,a,m){i['GoogleAnalyticsObject']=r;i[r]=i[r]||function(){
      (i[r].q=i[r].q||[]).push(arguments)},i[r].l=1*new Date();a=s.createElement(o),
      m=s.getElementsByTagName(o)[0];a.async=1;a.src=g;m.parentNode.insertBefore(a,m)
      })(window,document,'script','https://www.google-analytics.com/analytics.js','ga');
      ga('create', 'UA-140472724-1', 'auto');
      ga('send', 'pageview');
    </script>
  </head>
  <body>
    <div id="terminal"></div>
    <script>
      sessionId = "{{.Id}}";
    </script>
    <script src="../static/terminal.js"></script>
  </body>
</html>
`

func NewHeader() *Header {
	id := uuid.New().String()
	rich := fmt.Sprintf("http://localhost:8000/v/%s", id)
	plain := fmt.Sprintf("http://localhost:8000/p/%s", id)
	wS := fmt.Sprintf("http://localhost:8000/%s", id)
	file, err := os.Create("/tmp/" + id)
	if err != nil {
		log.Fatalln(err)
	}
	header := &Header{
		Id:        id,
		RichText:  rich,
		PlainText: plain,
		WS:        wS,
		File:      file,
	}
	Headers[id] = header
	return header
}

type Header struct {
	Id        string   `json:"id"`
	RichText  string   `json:"rich_text"`
	PlainText string   `json:"plain_text"`
	WS        string   `json:"WS"`
	File      *os.File `json:-`
}

func (h *Header) String() string {
	return pretty.JSONString(map[string]interface{}{
		"id":         h.Id,
		"rich_text":  h.RichText,
		"plain_text": h.PlainText,
		"ws":         h.WS,
	})
}

var indexhtml = `
<!doctype html>
  <html>
    <head>
      <link rel="stylesheet" href="node_modules/xterm/dist/xterm.css" />
      <script src="node_modules/xterm/dist/xterm.js"></script>
    </head>
    <body>
      <div id="terminal"></div>
      <script>

  var term = new Terminal({
    convertEol: true,
    scrollback: 10000,
    disableStdin: true,
    cursorBlink: true,
  });

        //var term = new Terminal();
        term.open(document.getElementById('terminal'));
        term.write('Hello from \x1B[1;3;31mxterm.js\x1B[0m $ ')

        var socket = new WebSocket("ws://localhost:8000/ws/%s");

        socket.onopen = function () {
                term.write("Status: Connected\n");
        };

        socket.onmessage = function (e) {
                term.write(e.data);
        };

      </script>
    </body>
  </html>
`

var wshtml = `
<input id="input" type="text" />
<button onclick="send()">Send</button>
<pre id="output"></pre>
<script>
        var input = document.getElementById("input");
        var output = document.getElementById("output");
        var socket = new WebSocket("ws://localhost:8000/ws/%s");

        socket.onopen = function () {
                output.innerHTML += "Status: Connected\n";
        };

        socket.onmessage = function (e) {
                output.innerHTML += "Server: " + e.data + "\n";
        };

        function send() {
                socket.send(input.value);
                input.value = "";
        }
</script>
`

func front() {
	log.Println("listening on port http://0.0.0.0:8000")
	//http.Handle("/echo.html", http.FileServer(http.Dir(".")))
	http.Handle("/", http.FileServer(http.Dir("./seashells.io")))
	http.HandleFunc("/hehehehehe", func(w http.ResponseWriter, r *http.Request) {
		//http.ServeFile(w, r, "index.html")
		//http.FileServer(http.Dir(".")).ServeHTTP(w, r)
		w.Header().Set("Content-Type", "text/html")
		id := strings.TrimPrefix(r.RequestURI, "/")
		w.Write([]byte(fmt.Sprintf(indexhtml, id)))
	})
	http.HandleFunc("/wshtml/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.RequestURI, "/wshtml/")
		if _, ok := Headers[id]; !ok {
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, http.StatusText(http.StatusNotFound))
			return
		}
		//io.WriteString(w, )
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(fmt.Sprintf(wshtml, id)))
	})
	http.HandleFunc("/ws/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.RequestURI, "/ws/")

		if _, ok := Headers[id]; !ok {
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, http.StatusText(http.StatusNotFound))
			return
		}
		tail, err := tail.TailFile(Headers[id].File.Name(), tail.Config{Follow: true})
		if err != nil {
			log.Println(err)
			return
		}
		ws.Handler(func(wsconn *ws.Conn) {
			for line := range tail.Lines {
				// io.Copy(chunkedConn, buffer)
				//io.Copy(wsconn, buffer)
				io.WriteString(wsconn, line.Text+"\n")
				//time.Sleep(time.Second / 10)
			}
		}).ServeHTTP(w, r)
	})
	http.Handle("/ws", ws.Handler(func(wsconn *ws.Conn) {
		buf, err := ioutil.ReadAll(io.LimitReader(wsconn, 36))
		if err != nil {
			log.Println(err)
			return
		}
		id := string(buf)
		log.Println("sessionId", id)

		if _, ok := Headers[id]; !ok {
			//w.WriteHeader(http.StatusNotFound)
			io.WriteString(wsconn, http.StatusText(http.StatusNotFound))
			return
		}
		tail, err := tail.TailFile(Headers[id].File.Name(), tail.Config{Follow: true})
		if err != nil {
			log.Println(err)
			return
		}

		for line := range tail.Lines {
			io.WriteString(wsconn, line.Text+"\n")
		}
	}))
	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		var upgrader = websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}
		conn, _ := upgrader.Upgrade(w, r, nil) // error ignored for sake of simplicity

		for {
			// Read message from browser
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			// Print the message to the console
			fmt.Printf("%s sent: %s\n", conn.RemoteAddr(), string(msg))

			// Write message back to browser
			if err = conn.WriteMessage(msgType, msg); err != nil {
				return
			}
		}
	})
	http.HandleFunc("/p/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.RequestURI, "/p/")
		if _, ok := Headers[id]; !ok {
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, http.StatusText(http.StatusNotFound))
			return
		}
		log.Println("dumping buffer for", id)
		//io.Copy(w, Headers[id].File)
		file, err := os.Open(Headers[id].File.Name())
		if err != nil {
			log.Println(err)
			return
		}
		buf, err := ioutil.ReadAll(file)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, err.Error())
			return
		}

		if _, err := w.Write(buf); err != nil {
			log.Println(err)
		}
	})
	http.HandleFunc("/v/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.RequestURI, "/v/")
		header, ok := Headers[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, http.StatusText(http.StatusNotFound))
			return
		}
		tail, err := tail.TailFile(Headers[id].File.Name(), tail.Config{Follow: true})
		if err != nil {
			log.Println(err)
			return
		}

		tmpl, err := template.New(id).Parse(vtemplate)
		if err != nil {
			panic(err)
		}
		rendered := bytes.NewBuffer(make([]byte, 0))
		err = tmpl.Execute(rendered, header)
		if err != nil {
			panic(err)
		}
		w.Write(rendered.Bytes())
		return

		// buffer := Headers[id].Buffer
		// w.Write(Headers[id].Buffer.Bytes())
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			log.Println(err)
			return
		}
		chunkedConn := httputil.NewChunkedWriter(conn)
		defer chunkedConn.Close()
		log.Println("streaming buffer for", id)
		io.WriteString(conn, "HTTP/1.1 200 OK\r\n")
		io.WriteString(conn, "Content-Type: text/plain; charset=utf-8\r\n")
		io.WriteString(conn, "Transfer-Encoding: chunked\r\n")
		io.WriteString(conn, "\r\n")
		for line := range tail.Lines {
			//io.Copy(chunkedConn, buffer)
			io.WriteString(chunkedConn, line.Text+"\n")
		}
	})
	log.Fatalln(http.ListenAndServe(":8000", nil))
}

var Headers = make(map[string]*Header)

func main() {
	go front()
	log.Println("listening on port tcp://0.0.0.0:1337")
	ln, err := net.Listen("tcp", ":1337")
	if err != nil {
		log.Fatalln(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		header := NewHeader()
		log.Println("connected:", conn.RemoteAddr(), header.Id, header.PlainText, header.RichText, header.WS)
		go io.Copy(io.MultiWriter(os.Stdout, header.File), conn)
		io.WriteString(conn, header.String())
	}
}
