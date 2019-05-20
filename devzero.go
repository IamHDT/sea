package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"gopkg.in/fsnotify.v1"
)

func DevZero() (chan byte, chan struct{}) {
	zeros := make(chan byte)
	buf := make([]byte, 4)
	devzero, err := os.Open("file")
	if err != nil {
		panic(err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan struct{})

	fileName := "file"

	err = watcher.Add(fileName)
	if err != nil {
		log.Fatal(err)
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
				log.Println("reached EOF, waiting for update", "read:", n)
				select {
				case <-done:
					println("goroutine: close")
					return
				case event := <-watcher.Events:
					if event.Op&fsnotify.Write == fsnotify.Write {
						log.Println("modified file:", event.Name)

					}
				}
			}
			log.Println("offset:", offset, "read", n)
			for i := range buf[:n] {
				println(i)
				zeros <- buf[i]
			}
			offset += n
		}
	}()
	return zeros, done
}

func main() {
	zeros, done := DevZero()
	for i := 0; ; i++ {
		b := <-zeros
		if b == 0x31 {
			close(done)
			break
		}
		fmt.Printf("[%d] %x\n", i, b)
	}
}
