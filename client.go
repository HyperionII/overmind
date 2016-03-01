package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = 54 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 512

	// Client's buffered channel size.
	messageChannelSize = 16
)

// Client contains all information associated with a websocket client conn.
type Client struct {
	ID   int
	Name string

	conn    *websocket.Conn
	server  *Server
	msgCh   chan []byte
	closeCh chan bool
}

// NewClient initializes a new Client struct, sets the default read limits and
// deadlines and creates a Pong Handler for the connection.
func NewClient(conn *websocket.Conn, server *Server) *Client {
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	return &Client{
		ID:      maxClientID,
		conn:    conn,
		server:  server,
		msgCh:   make(chan []byte, messageChannelSize),
		closeCh: make(chan bool, 1),
	}
}

// CloseAndWait sends a signal to the close channel to terminate the
// Read() and Write() client's goroutines. It waits for the message
// to go through before returning.
func (c *Client) CloseAndWait() {
	c.closeCh <- true
}

// CloseOrIgnore sends a signal to the close channel to terminate the
// Read() and Write() client's goroutines. It attempts to send the signal
// to the channel or ignore it right away if it can't go through.
func (c *Client) CloseOrIgnore() {
	select {
	case c.closeCh <- true:
		// Close signal sent successfully.
	default:
		// If close signal can't be sent, it means there's already
		// one signal in queue. Ignore send.
	}
}

// Listen will spawn a new goroutine to write to the client and will make a
// call to onRead() for when the client sends in data.
func (c *Client) Listen() {
	go c.onWrite()
	c.onRead()
}

// Write attempts to send a message to the client. If the client
// channel's buffer is full, then we spawn a new goroutine with a call
// to client.writeAndWait which will wait waitTime duration. This is done in
// order to prevent blocking the caller for too long.
func (c *Client) Write(msg []byte) {
	select {
	case c.msgCh <- msg:
		// Message sent.

	default:
		// Client's queue is full, so spawn a new goroutine to avoid
		// blocking the caller for long.
		go c.writeAndWait(msg)

	}
}

// writeAndWait attempts to send the message to the client's channel. If it
// takes too long to respond, we disconnect the client.
func (c *Client) writeAndWait(msg []byte) bool {
	select {
	case c.msgCh <- msg:
		// Message successfully sent.
		return true

	case <-time.After(writeWait):
		// If client queue is full and timed out, then send a close channel
		// signal.
		c.CloseOrIgnore()

		return false
	}
}

// WriteMany takes an array of strings and calls the Write() function
// to send each message in sequence to the client's channel. If one of them
// fails, we return.
func (c *Client) WriteMany(msgs []string) bool {
	var ok bool

	for _, msg := range msgs {
		ok = c.writeAndWait([]byte(msg))

		if !ok {
			return false
		}
	}

	return true
}

// ReadAndSetClientName awaits for the user to input his name and then sets it
// to the current client instance.
func (c *Client) ReadAndSetClientName() error {
	var credentials struct {
		Name string `json:"name"`
	}

	err := c.conn.ReadJSON(credentials)

	if err != nil {
		return err
	} else if credentials.Name == "" {
		return errors.New("read and set client name: empty name received")
	}

	c.Name = credentials.Name
	c.conn.SetReadDeadline(time.Now().Add(pongWait))

	return nil
}

// write is a wrapper around gorilla's Connection.WriteMessage function
// that sets a write deadline to timeout a write call and the actual
// WriteMessage call to send the message.
func (c *Client) write(messageType int, msg []byte) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(messageType, msg)
}

// onWrite function writes to the client all messages. Messages include server
// custom messages, ping messages and close connection messages. Must be run
// on a different goroutine from onRead().
func (c *Client) onWrite() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()

		// End onRead() goroutine.
		c.CloseOrIgnore()
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
				err = fmt.Errorf("client[%d] ping timeout: %s", c.ID, err.Error())
				c.server.LogError(err)
				return
			}

		case <-c.closeCh:
			c.write(websocket.CloseMessage, []byte{})
			return
		}
	}
}

// onRead function reads all messages from client and makes sure to send them
// to the server. Client messages include the pong messages and custom client
// messages. Must be run on a different goroutine frmo onWrite().
func (c *Client) onRead() {
	defer func() {
		c.conn.Close()

		// End onWrite() goroutine.
		c.CloseOrIgnore()
	}()

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
