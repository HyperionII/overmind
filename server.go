package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Server struct {
	messages       []string
	clients        map[int]*Client
	addClientCh    chan *Client
	removeClientCh chan *Client
	errorCh        chan error
	broadcastCh    chan []byte
	upgrader       websocket.Upgrader
	cacheClient    *RedisClient
}

func NewServer() *Server {
	cacheClient := NewRedisClient()

	if err := cacheClient.Ping(); err != nil {
		log.Fatalln("could not ping cache database:", err)
		return nil
	}

	return &Server{
		messages:       []string{},
		clients:        make(map[int]*Client),
		addClientCh:    make(chan *Client),
		removeClientCh: make(chan *Client),
		errorCh:        make(chan error),
		broadcastCh:    make(chan []byte),
		cacheClient:    NewRedisClient(),

		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

func (s *Server) AddClient(client *Client) {
	s.addClientCh <- client
}

func (s *Server) RemoveClient(client *Client) {
	s.removeClientCh <- client
}

func (s *Server) BroadcastMessage(message []byte) {
	s.broadcastCh <- message
}

func (s *Server) LogError(err error) {
	s.errorCh <- err
}

func (s *Server) OnConnect(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)

	if err != nil {
		s.LogError(err)
		return
	}

	client := NewClient(conn, s)

	s.AddClient(client)
	client.Listen()
	s.RemoveClient(client)
}

func (s *Server) Listen() {
	for {
		select {
		case client := <-s.addClientCh:
			s.addClient(client)
			s.sendAllCachedMessages(client, "defaultChannel")

		case client := <-s.removeClientCh:
			s.removeClient(client)

		case msg := <-s.broadcastCh:
			s.broadcastMessage(msg)

		case err := <-s.errorCh:
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
		go client.Write(message)
	}

	s.cacheClient.SaveMessage("defaultChannel", string(message))
}

func (s *Server) sendAllCachedMessages(client *Client, channel string) {
	messages, err := s.cacheClient.GetAllMessages(channel)

	if err != nil {
		log.Println("send all cached messages:", err)
		return
	}

	go client.WriteMany(messages)
}
