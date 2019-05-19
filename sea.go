package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/navigaid/pretty"
)

func NewHeader() *Header {
	id := uuid.New().String()
	rich := fmt.Sprintf("http://localhost:8000/v/%s", id)
	plain := fmt.Sprintf("http://localhost:8000/p/%s", id)
	buf := bytes.NewBuffer(make([]byte, 0))
	header := &Header{
		Id:        id,
		RichText:  rich,
		PlainText: plain,
		Buffer:    buf,
	}
	Headers[id] = header
	return header
}

type Header struct {
	Id        string        `json:"id"`
	RichText  string        `json:"rich_text"`
	PlainText string        `json:"plain_text"`
	Buffer    *bytes.Buffer `json:-`
}

func (h *Header) String() string {
	return pretty.JSONString(h)
}

func front() {
	log.Println("listening on port http://0.0.0.0:8000")
	http.HandleFunc("/p/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.RequestURI, "/p/")
		if _, ok := Headers[id]; !ok {
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, http.StatusText(http.StatusNotFound))
			return
		}
		log.Println("dumping buffer for", id)
		//go io.Copy(w, Headers[id].Buffer)
		w.Write(Headers[id].Buffer.Bytes())
	})
	http.HandleFunc("/v/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.RequestURI, "/v/")
		if _, ok := Headers[id]; !ok {
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, http.StatusText(http.StatusNotFound))
			return
		}
		buffer := Headers[id].Buffer
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
		for {
			io.Copy(chunkedConn, buffer)
			time.Sleep(time.Second / 10)
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
		log.Println("connected:", conn.RemoteAddr(), header.Id, header.PlainText, header.RichText)
		go io.Copy(io.MultiWriter(os.Stdout, header.Buffer), conn)
		io.WriteString(conn, header.String())
	}
}
