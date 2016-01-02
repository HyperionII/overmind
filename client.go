package main

import (
	"fmt"
	"io"
	"log"
	"strconv"

	"github.com/gorilla/websocket"
)

var (
	maxClientId = 0
)

type Client struct {
	Id   int
	Name string

	conn     *websocket.Conn
	server   *Server
	msgChan  chan string
	doneChan chan bool
}

func NewClient(conn *websocket.Conn, server *Server) *Client {
	maxClientId++

	return &Client{
		Id:       maxClientId,
		Name:     "Client" + strconv.Itoa(maxClientId),
		conn:     conn,
		server:   server,
		msgChan:  make(chan string),
		doneChan: make(chan bool),
	}
}

func (c *Client) Close() {
	c.doneChan <- true
}

func (c *Client) Listen() {
	go c.onWrite()
	c.onRead()
}

func (c *Client) Write(msg string) {
	select {
	case c.msgChan <- msg:
		log.Println("Client::Write->", msg)

	default:
		c.server.RemoveClient(c)
		err := fmt.Errorf("attempt to send message to client [%d] failed: client is disconnected: ", c.Id)
		log.Println(err)
	}
}

func (c *Client) onWrite() {
	for {
		log.Println("Client::Write -> Waiting for event...")

		select {
		case <-c.doneChan:
			c.server.RemoveClient(c)

			// For onRead method
			c.doneChan <- true
			return
		case msg := <-c.msgChan:
			log.Println("Client chan received message:", msg)

			err := c.conn.WriteMessage(websocket.TextMessage, []byte(msg))

			if err != nil {
				log.Println(err)
				c.server.errorChan <- err
			}

			log.Println("Message sent!")
		}

		log.Println("Client::Write -> Event served!")
	}
}

func (c *Client) onRead() {
	for {
		select {
		case <-c.doneChan:
			c.server.RemoveClient(c)

			// For onWrite method
			c.doneChan <- true
			return
		default:
			messageType, data, err := c.conn.ReadMessage()
			log.Println("Received message with type", messageType)

			if err == io.EOF {
				log.Println("EOF found :D")
				c.doneChan <- true
				continue
			} else if err != nil {
				c.server.errorChan <- err
				continue
			}

			msg := string(data)
			c.server.BroadcastMessage(msg)
		}
	}
}
