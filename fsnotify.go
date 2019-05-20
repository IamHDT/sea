package main

import (
	"log"
	"os"

	"gopkg.in/fsnotify.v1"
)

func main() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	//done := make(chan bool)

	fileName := "file"

	os.Create(fileName)

	err = watcher.Add(fileName)
	if err != nil {
		log.Fatal(err)
	}
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("modified file:", event.Name)

			}
		}
	}
}
