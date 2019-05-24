package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/google/uuid"
	"github.com/navigaid/pretty"
	"golang.org/x/net/websocket"
	"gopkg.in/fsnotify.v1"
)

var Headers = make(map[string]*Header)

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

func NewHeader(conn net.Conn) *Header {
	ip := conn.RemoteAddr().String()
	id := uuid.New().String()
	rich := fmt.Sprintf("http://localhost:8000/v/%s", id)
	plain := fmt.Sprintf("http://localhost:8000/p/%s", id)
	file, err := os.Create("/tmp/" + id)
	if err != nil {
		log.Fatalln(err)
	}
	doneTCP := make(chan struct{})
	header := &Header{
		IP:        ip,
		Id:        id,
		RichText:  rich,
		PlainText: plain,
		file:      file,
		doneTCP:   doneTCP,
	}
	Headers[id] = header
	return header
}

type Header struct {
	IP        string   `json:"ip"`
	Id        string   `json:"id"`
	RichText  string   `json:"v"`
	PlainText string   `json:"p"`
	file      *os.File //`json:"-"`
	doneTCP   chan struct{}
}

func (h *Header) String() string {
	return pretty.JSONString(h)
}

func serveWS() {
	exe, err := os.Executable()
	if err != nil {
		log.Fatalln(err)
	}
	http.Handle("/", http.FileServer(http.Dir(path.Join(path.Dir(exe), "seashells.io"))))
	http.HandleFunc("/p/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.RequestURI, "/p/")
		if _, ok := Headers[id]; !ok {
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, http.StatusText(http.StatusNotFound))
			return
		}
		log.Println("dumping buffer for", id)
		//io.Copy(w, Headers[id].File)
		file, err := os.Open(Headers[id].file.Name())
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
	})
	http.Handle("/ws", websocket.Handler(func(wsconn *websocket.Conn) {
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

		tail, doneWS := Dev("/tmp/"+id, Headers[id].doneTCP)

		for b := range tail {
			_, err := wsconn.Write(b)
			if err != nil {
				log.Println(err)
				//println("closing connection")
				close(doneWS)
				return
			}
		}
	}))
	log.Println("listening on port http://0.0.0.0:8000")
	log.Fatalln(http.ListenAndServe(":8000", nil))
}

func serveTCP() {
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
		header := NewHeader(conn)
		log.Print("connected:", header)
		go func() {
			io.Copy(header.file, conn)
			println("tcp client disconnected")
			close(header.doneTCP)
		}()
		io.WriteString(conn, header.String())
	}
}

func Dev(file string, doneTCP chan struct{}) (chan []byte, chan struct{}) {
	data := make(chan []byte)
	done := make(chan struct{})

	buf := make([]byte, 65536)
	devzero, err := os.Open(file)
	if err != nil {
		log.Fatalln(err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalln(err)
	}

	err = watcher.Add(file)
	if err != nil {
		log.Fatalln(err)
	}

	offset := 0
	go func() {
		for {
			n, err := devzero.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Println(err)
					break
				}
				//log.Println("reached EOF, waiting for update", "read:", n)
				select {
				case <-doneTCP:
					return
				case <-done:
					println("ws client disconnected")
					return
				case event := <-watcher.Events:
					if event.Op&fsnotify.Write == fsnotify.Write {
						//log.Println("modified file:", event.Name)
					}
				}
			}
			// log.Println("offset:", offset, "read", n)
			data <- buf[:n]
			offset += n
		}
	}()
	return data, done
}

func main() {
	go serveWS()
	go serveTCP()
	select {}
}
