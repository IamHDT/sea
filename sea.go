package main

import (
	"github.com/google/uuid"
	"io"
	"log"
	"net"
)

func main() {
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
		io.WriteString(conn, uuid.New().String())
	}
}
