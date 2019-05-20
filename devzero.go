package main

import (
	"io"
	"log"
	"os"

	"gopkg.in/fsnotify.v1"
)

func Dev(file string) (chan []byte, chan struct{}) {
	zeros := make(chan []byte)
	done := make(chan struct{})

	buf := make([]byte, 65536)
	devzero, err := os.Open(file)
	if err != nil {
		panic(err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	err = watcher.Add(file)
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
			zeros <- buf[:n]
			offset += n
		}
	}()
	return zeros, done
}

/*
func testDev() {
	zeros, done := Dev("file")
	for i := 0; ; i++ {
		b := <-zeros
		if b == 0x31 {
			close(done)
			break
		}
		fmt.Printf("[%d] %x\n", i, b)
	}
}
*/
