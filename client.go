package main

import (
	"fmt"
	"io"
	"strconv"

	"golang.org/x/net/websocket"
)

var (
	maxClientId = 0
)

type Client struct {
	Id   int
	Name string

	ws       *websocket.Conn
	server   *Server
	msgChan  chan string
	doneChan chan bool
}

func NewClient(ws *websocket.Conn, server *Server) *Client {
	maxClientId++

	return &Client{
		Id:       maxClientId,
		Name:     "Client" + strconv.Itoa(maxClientId),
		ws:       ws,
		server:   server,
		msgChan:  make(chan string),
		doneChan: make(chan bool),
	}
}

func (c *Client) Close() {
	c.doneChan <- true
}

func (c *Client) Listen() {
	go c.listenWrite()
	c.listenRead()
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

func (c *Client) listenWrite() {
	for {
		select {
		case <-c.doneChan:
			c.server.RemoveClient(c)

			// For listenRead method
			c.doneChan <- true
			return
		case msg := <-c.msgChan:
			_, err := c.ws.Write([]byte(msg))

			if err != nil {
				c.server.errorChan <- err
			}
		}
	}
}

func (c *Client) listenRead() {
	var bytesRead int
	var err error
	data := make([]byte, 1024)

	for {
		select {
		case <-c.doneChan:
			c.server.RemoveClient(c)

			// For listenWrite method
			c.doneChan <- true
			return
		default:
			bytesRead, err = c.ws.Read(data)

			if err == io.EOF {
				c.doneChan <- true
				continue
			} else if err != nil {
				c.server.errorChan <- err
				continue
			}

			msg := string(data[:bytesRead])
			c.server.BroadcastMessage(msg)
		}
	}
}
