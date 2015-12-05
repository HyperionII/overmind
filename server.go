package main

import (
	"log"

	"golang.org/x/net/websocket"
)

type Server struct {
	messages         []string
	clients          map[int]*Client
	addClientChan    chan *Client
	removeClientChan chan *Client
	errorChan        chan error
	broadcastChannel chan string
}

func NewServer() *Server {
	return &Server{
		messages:         []string{},
		clients:          make(map[int]*Client),
		addClientChan:    make(chan *Client),
		removeClientChan: make(chan *Client),
		errorChan:        make(chan error),
		broadcastChannel: make(chan string),
	}
}

func (s *Server) AddClient(client *Client) {
	s.addClientChan <- client
}

func (s *Server) RemoveClient(client *Client) {
	s.removeClientChan <- client
}

func (s *Server) BroadcastMessage(message string) {
	s.broadcastChannel <- message
}

func (s *Server) OnConnect(ws *websocket.Conn) {
	client := NewClient(ws, s)

	s.AddClient(client)
	client.Listen()

	if err := ws.Close(); err != nil {
		log.Println(err)
	}
}

func (s *Server) Listen() {
	for {
		select {

		case client := <-s.addClientChan:
			s.clients[client.Id] = client

			log.Println(client.Name, "has joined the channel!")
			log.Println("Currently", len(s.clients), "clients connected!")

		case client := <-s.removeClientChan:
			delete(s.clients, client.Id)

			log.Println(client.Name, "has left the channel!")
			log.Println("Currently", len(s.clients), "clients connected!")

		case msg := <-s.broadcastChannel:
			for _, client := range s.clients {
				client.Write(msg)
			}

		case err := <-s.errorChan:
			log.Println("error:", err.Error())
		}
	}
}
