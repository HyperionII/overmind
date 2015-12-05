package main

import (
	"log"
	"net/http"

	"golang.org/x/net/websocket"
)

func main() {
	log.Println("Starting server at port...")

	s := &Server{}

	http.Handle("/echo", websocket.Handler(s.EchoHandler))

	err := http.ListenAndServe(":2222", nil)

	if err != nil {
		log.Fatalln("main: listen and serve:", err)
	}
}
