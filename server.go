package main

import (
	"io"
	"log"
	"net/http"

	"golang.org/x/net/websocket"
)

func EchoHandler(ws *websocket.Conn) {
	io.Copy(ws, ws)
}

func main() {
	log.Println("Starting server at port...")

	http.Handle("/echo", websocket.Handler(EchoHandler))

	err := http.ListenAndServe(":2222", nil)

	if err != nil {
		log.Fatalln("main: listen and serve:", err)
	}
}
