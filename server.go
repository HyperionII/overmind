package main

import (
	"io"

	"golang.org/x/net/websocket"
)

var (
	mainChannel = make(chan string)
)

type Server struct {
	messages         []string
	clients          map[int]string
	addClientChan    chan int
	removeClientChan chan int
	errorChan        chan error
}

func (s *Server) EchoHandler(ws *websocket.Conn) {
	io.Copy(ws, ws)
}
