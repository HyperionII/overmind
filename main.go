package main

import (
	"log"
	"net/http"

	"golang.org/x/net/websocket"
)

func main() {
	log.Println("Starting server at port...")

	s := NewServer()

	http.Handle("/echo", websocket.Handler(s.OnConnect))

	log.Println("Listening...")
	go s.Listen()
	err := http.ListenAndServe(":2222", nil)

	if err != nil {
		log.Fatalln("main: listen and serve:", err)
	}
}
