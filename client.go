package main

import (
	"fmt"
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

func (c *Client) Write(msg []byte) bool {
	select {
	case c.msgCh <- msg:
		// Message successfully sent.
		return true

	case <-time.After(writeWait):
		// If client queue is full and timed out, then send a close channel
		// signal.
		select {
		case c.closeCh <- true:
			// Close signal sent successfully.
		default:
			// If close signal can't be sent, it means there's already
			// one signal in queue already. Ignore send.
		}

		return false
	}
}

func (c *Client) WriteMany(msgs []string) bool {
	var ok bool

	for _, msg := range msgs {
		ok = c.Write([]byte(msg))

		if !ok {
			return false
		}
	}

	return true
}

func (c *Client) write(messageType int, msg []byte) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(messageType, msg)
}

func (c *Client) onWrite() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()

		// End onRead() goroutine.
		c.closeCh <- true
	}()

	for {
		select {
		case msg := <-c.msgCh:
			err := c.write(websocket.TextMessage, msg)

			if err != nil {
				c.server.LogError(err)
				return
			}

		case <-ticker.C:
			err := c.write(websocket.PingMessage, []byte{})

			if err != nil {
				err = fmt.Errorf("client[%d] ping timeout: %s", c.Id, err.Error())
				c.server.LogError(err)
				return
			}

		case <-c.closeCh:
			c.write(websocket.CloseMessage, []byte{})
			return
		}
	}
}

func (c *Client) onRead() {
	defer func() {
		// End onWrite() goroutine.
		c.closeCh <- true
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
