package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/navigaid/pretty"
)

func NewHeader() *Header {
	id := uuid.New().String()
	rich := fmt.Sprintf("http://localhost/v/%s", id)
	plain := fmt.Sprintf("http://localhost/p/%s", id)
	return &Header{
		Id:        id,
		RichText:  rich,
		PlainText: plain,
	}
}

type Header struct {
	Id        string `json:"id"`
	RichText  string `json:"rich_text"`
	PlainText string `json:"plain_text"`
}

func (h *Header) String() string {
	return pretty.JSONString(h)
}

func front() {
	log.Println("listening on port http://0.0.0.0:8000")
	log.Fatalln(http.ListenAndServe(":8000", nil))
}

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
		log.Println("connected:", conn.RemoteAddr(), header.Id)
		go io.Copy(os.Stdout, conn)
		io.WriteString(conn, header.String())
	}
}
