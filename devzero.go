package main

import (
	"fmt"
	"log"
	"os"
)

func DevZero() chan byte {
	zeros := make(chan byte)
	buf := make([]byte, 1)
	devzero, err := os.Open("/dev/zero")
	if err != nil {
		panic(err)
	}
	offset := 0
	go func() {
		for {
			n, err := devzero.Read(buf)
			if err != nil {
				log.Println(err)
			}
			log.Println(offset, n, err)
			zeros <- buf[0]
			offset += n
		}
	}()
	return zeros
}

func main() {
	zeros := DevZero()
	for {
		fmt.Println(<-zeros)
	}
}
