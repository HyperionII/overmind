package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Server contains all information to host the websocket server.
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

// NewServer initializes a new Client struct.
func NewServer() *Server {
	return &Server{
		messages:       []string{},
		clients:        make(map[int]*Client),
		addClientCh:    make(chan *Client),
		removeClientCh: make(chan *Client),
		errorCh:        make(chan error),
		broadcastCh:    make(chan []byte, 64),

		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// AddClient adds a new client to the server's client list. AddClient is a
// blocking function.
func (s *Server) AddClient(client *Client) {
	s.addClientCh <- client
}

// RemoveClient removes an existing client from the server's client list.
// RemoveClient is a blocking function.
func (s *Server) RemoveClient(client *Client) {
	s.removeClientCh <- client
}

// BroadcastMessage is used to send all client's a message. BroadcastMessage is
// a blocking function.
func (s *Server) BroadcastMessage(c *Client, message []byte) {
	log.Printf("%s says: %s", c.Name, message)
	s.broadcastCh <- message
}

// LogError lets other goroutines to log errors though the server. In the
// future, error processing might be done in this function so it is better
// to call this function when an error happened. This function is a blocking
// function.
func (s *Server) LogError(err error) {
	s.errorCh <- err
}

// OnConnect 'upgrades' a normal HTTP request to a websocket connection.
func (s *Server) OnConnect(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)

	if err != nil {
		s.LogError(err)
		conn.Close()
		return
	}

	client := NewClient(conn, s)
	err = client.ReadAndSetClientName()

	if err != nil {
		s.LogError(err)
		conn.Close()
		return
	}

	s.AddClient(client)
	client.Listen()
	s.RemoveClient(client)
}

// Listen is the main server loop. This function awaits for incoming messages
// from all channels.
func (s *Server) Listen() {
	for {
		select {
		case client := <-s.addClientCh:
			s.addClient(client)

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
	s.clients[client.ID] = client

	log.Println(client.Name, "has joined the channel!")
	log.Println("Currently", len(s.clients), "clients connected!")
}

func (s *Server) removeClient(client *Client) {
	delete(s.clients, client.ID)

	log.Println(client.Name, "has left the channel!")
	log.Println("Currently", len(s.clients), "clients connected!")
}

func (s *Server) broadcastMessage(message []byte) {
	for _, client := range s.clients {
		client.Write(message)
	}
}

func (s *Server) sendAllCachedMessages(client *Client, channel string) {
	messages, err := s.cacheClient.GetAllMessages(channel)

	if err != nil {
		log.Println("send all cached messages:", err)
		return
	}

	go client.WriteMany(messages)
}
