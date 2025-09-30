package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		// Parent scenario enforces origin restrictions; allow here to simplify embedding.
		return true
	},
}

func handleWebSocketStream(w http.ResponseWriter, r *http.Request, session *session, metrics *metricsRegistry) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		metrics.httpUpgradesFail.Add(1)
		log.Printf("websocket upgrade failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := newWSClient(session, conn)
	session.addClient(client)
	client.start()
}

type wsClient struct {
	session *session
	conn    *websocket.Conn
	send    chan websocketEnvelope
	closeMu sync.Mutex
	closed  bool
}

func newWSClient(session *session, conn *websocket.Conn) *wsClient {
	return &wsClient{
		session: session,
		conn:    conn,
		send:    make(chan websocketEnvelope, 64),
	}
}

func (c *wsClient) start() {
	go c.writeLoop()
	go c.readLoop()
	// Emit initial status snapshot
	c.enqueue(websocketEnvelope{
		Type: "status",
		Payload: mustJSON(statusPayload{
			Status:    "connected",
			Timestamp: time.Now().UTC(),
		}),
	})
}

func (c *wsClient) enqueue(msg websocketEnvelope) {
	select {
	case c.send <- msg:
	default:
		// Drop oldest to prevent slow consumer blocking entire session
		select {
		case <-c.send:
		default:
		}
		c.send <- msg
	}
}

func (c *wsClient) close() {
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return
	}
	c.closed = true
	c.closeMu.Unlock()

	close(c.send)
	_ = c.conn.Close()
	c.session.removeClient(c)
}

func (c *wsClient) writeLoop() {
	for msg := range c.send {
		if err := c.conn.WriteJSON(msg); err != nil {
			c.close()
			return
		}
	}
}

func (c *wsClient) readLoop() {
	defer c.close()
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				// Unexpected error; session continues but client disconnects
			}
			return
		}

		var envelope websocketEnvelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			continue
		}

		switch envelope.Type {
		case "input":
			var payload inputPayload
			if len(envelope.Payload) == 0 {
				continue
			}
			if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
				continue
			}

			bytes, err := decodePayload(payload)
			if err != nil {
				continue
			}
			_ = c.session.handleInput(bytes)

		case "resize":
			var payload resizePayload
			if len(envelope.Payload) == 0 {
				continue
			}
			if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
				continue
			}
			if err := c.session.resize(payload.Cols, payload.Rows); err != nil {
				continue
			}

		case "heartbeat":
			c.session.touch()
			c.enqueue(websocketEnvelope{
				Type:    "heartbeat",
				Payload: mustJSON(heartbeatPayload{Timestamp: time.Now().UTC()}),
			})

		default:
			// ignore unsupported types
		}
	}
}

func decodePayload(payload inputPayload) ([]byte, error) {
	switch payload.Encoding {
	case "", "utf-8", "utf8":
		return []byte(payload.Data), nil
	case "base64":
		return base64.StdEncoding.DecodeString(payload.Data)
	default:
		return nil, errors.New("unsupported encoding")
	}
}
