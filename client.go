package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

var (
	maxClientId = 0
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type Client struct {
	Id   int
	Name string

	conn    *websocket.Conn
	server  *Server
	msgCh   chan []byte
	closeCh chan bool
}

func NewClient(conn *websocket.Conn, server *Server) *Client {
	maxClientId++

	return &Client{
		Id:      maxClientId,
		Name:    "Client" + strconv.Itoa(maxClientId),
		conn:    conn,
		server:  server,
		msgCh:   make(chan []byte, 256),
		closeCh: make(chan bool, 1),
	}
}

func (c *Client) Close() {
	close(c.msgCh)
}

func (c *Client) Listen() {
	go c.onWrite()
	c.onRead()
}

func (c *Client) Write(msg []byte) {
	select {
	case c.msgCh <- msg:
	case <-time.After(writeWait):
		c.closeCh <- true
	}
}

func (c *Client) write(messageType int, msg []byte) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(messageType, msg)
}

func (c *Client) onWrite() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		log.Println("Closing client", c.Id, "")
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.msgCh:
			if !ok {
				c.write(websocket.CloseMessage, []byte{})
				return
			}

			err := c.write(websocket.TextMessage, []byte(msg))

			if err != nil {
				c.server.errorChan <- err
				return
			}

		case <-ticker.C:
			err := c.write(websocket.PingMessage, []byte{})

			if err != nil {
				c.server.errorChan <- fmt.Errorf("client[%d] ping timeout: %s", c.Id, err.Error())
				return
			}
		case <-c.closeCh:
			c.closeCh <- true // Close onRead goroutine.
			return
		}
	}
}

func (c *Client) onRead() {
	defer func() {
		c.server.removeClient(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-c.closeCh:
			c.closeCh <- true // End onWrite goroutine.
			return
		default:
			_, message, err := c.conn.ReadMessage()

			if err != nil {
				return
			}

			c.server.BroadcastMessage(message)
		}
	}
}
