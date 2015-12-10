package main

import (
	"fmt"
	"io"
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

	default:
		c.server.RemoveClient(c)
		err := fmt.Errorf("client is disconnected: attempt to send message to client [%d] failed", c.Id)
		c.server.errorChan <- err
	}
}

func (c *Client) onWrite() {
	for {
		select {
		case <-c.doneChan:
			c.server.RemoveClient(c)

			// For onRead method
			c.doneChan <- true
			return
		case msg := <-c.msgChan:
			err := c.conn.WriteMessage(websocket.TextMessage, []byte(msg))

			if err != nil {
				c.server.errorChan <- err
			}
		}
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
			fmt.Println("Received message with type", messageType)

			if err == io.EOF {
				fmt.Println("EOF found :D")
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
