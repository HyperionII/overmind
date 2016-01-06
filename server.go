package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Server struct {
	messages         []string
	clients          map[int]*Client
	addClientChan    chan *Client
	removeClientChan chan *Client
	errorChan        chan error
	broadcastChannel chan []byte
	upgrader         websocket.Upgrader
	cacheClient      *RedisClient
}

func NewServer() *Server {
	cacheClient := NewRedisClient()

	if err := cacheClient.Ping(); err != nil {
		log.Fatalln("could not ping cache database:", err)
		return nil
	}

	return &Server{
		messages:         []string{},
		clients:          make(map[int]*Client),
		addClientChan:    make(chan *Client),
		removeClientChan: make(chan *Client),
		errorChan:        make(chan error),
		broadcastChannel: make(chan []byte),
		cacheClient:      NewRedisClient(),

		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

func (s *Server) AddClient(client *Client) {
	s.addClientChan <- client
}

func (s *Server) RemoveClient(client *Client) {
	s.removeClientChan <- client
}

func (s *Server) BroadcastMessage(message []byte) {
	s.broadcastChannel <- message
}

func (s *Server) OnConnect(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println(err)
		return
	}

	client := NewClient(conn, s)

	s.AddClient(client)
	client.Listen()

	if err := conn.Close(); err != nil {
		log.Println(err)
	}
}

func (s *Server) Listen() {
	for {
		select {
		case client := <-s.addClientChan:
			s.addClient(client)
			s.sendAllCachedMessages(client, "defaultChannel")

		case client := <-s.removeClientChan:
			s.removeClient(client)

		case msg := <-s.broadcastChannel:
			s.broadcastMessage(msg)

		case err := <-s.errorChan:
			log.Println("server error channel:", err.Error())
		}
	}
}

func (s *Server) addClient(client *Client) {
	s.clients[client.Id] = client

	log.Println(client.Name, "has joined the channel!")
	log.Println("Currently", len(s.clients), "clients connected!")
}

func (s *Server) removeClient(client *Client) {
	delete(s.clients, client.Id)

	log.Println(client.Name, "has left the channel!")
	log.Println("Currently", len(s.clients), "clients connected!")
}

func (s *Server) broadcastMessage(message []byte) {
	for _, client := range s.clients {
		client.Write(message)
	}

	s.cacheClient.SaveMessage("defaultChannel", string(message))
}

func (s *Server) sendAllCachedMessages(client *Client, channel string) {
	messages, err := s.cacheClient.GetAllMessages(channel)

	if err != nil {
		log.Println("send all cached messages:", err)
		return
	}

	for _, message := range messages {
		select {
		case client.msgCh <- []byte(message):

		case <-time.After(10 * time.Second):
			close(client.msgCh)
			delete(s.clients, client.Id)
			return
		}
	}
}
