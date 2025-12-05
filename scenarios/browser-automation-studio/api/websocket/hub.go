package websocket

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

// Client represents a WebSocket client
type Client struct {
	ID                 uuid.UUID
	Conn               *websocket.Conn
	Send               chan any
	Hub                *Hub
	ExecutionID        *uuid.UUID // Optional: client can subscribe to specific execution
	RecordingSessionID *string    // Optional: client can subscribe to recording session updates
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan any
	register   chan *Client
	unregister chan *Client
	log        *logrus.Logger
	mu         sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub(log *logrus.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan any),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		log:        log,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

			h.log.WithField("client_id", client.ID).Info("Client connected to WebSocket hub")

			// Send a welcome message
			select {
			case client.Send <- map[string]any{
				"type":      "connected",
				"message":   "Connected to Browser Automation Studio",
				"timestamp": getCurrentTimestamp(),
			}:
			default:
				close(client.Send)
				h.mu.Lock()
				delete(h.clients, client)
				h.mu.Unlock()
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				h.log.WithField("client_id", client.ID).Info("Client disconnected from WebSocket hub")
			}
			h.mu.Unlock()

		case update := <-h.broadcast:
			h.mu.RLock()
			execID := extractExecutionID(update)
			for client := range h.clients {
				// If client is subscribed to a specific execution, filter updates
				if client.ExecutionID != nil && execID != nil && *client.ExecutionID != *execID {
					continue
				}

				select {
				case client.Send <- update:
				default:
					delete(h.clients, client)
					close(client.Send)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastUpdate sends an update to all connected clients
// BroadcastEnvelope pushes an automation event envelope directly to clients.
func (h *Hub) BroadcastEnvelope(event any) {
	h.broadcast <- event
}

// BroadcastRecordingAction sends a recording action to clients subscribed to a specific session.
func (h *Hub) BroadcastRecordingAction(sessionID string, action any) {
	message := map[string]any{
		"type":       "recording_action",
		"session_id": sessionID,
		"action":     action,
		"timestamp":  getCurrentTimestamp(),
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		// Only send to clients subscribed to this recording session
		if client.RecordingSessionID != nil && *client.RecordingSessionID == sessionID {
			select {
			case client.Send <- message:
			default:
				// Client buffer full, skip
			}
		}
	}
}

// GetClientCount returns the number of connected clients.
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// CloseExecution is a no-op for the hub; it satisfies the interface used by sinks.
func (h *Hub) CloseExecution(executionID uuid.UUID) {
	_ = executionID
}

// ServeWS handles WebSocket requests from clients.
func (h *Hub) ServeWS(conn *websocket.Conn, executionID *uuid.UUID) {
	client := &Client{
		ID:          uuid.New(),
		Conn:        conn,
		Send:        make(chan any, 256),
		Hub:         h,
		ExecutionID: executionID,
	}

	// Register the client
	client.Hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	// Set read deadline and pong handler for keepalive
	c.Conn.SetReadLimit(512)
	c.Conn.SetPongHandler(func(string) error {
		return nil
	})

	for {
		// Read message from client
		var msg map[string]any
		if err := c.Conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.log.WithError(err).Error("WebSocket error")
			}
			break
		}

		// Handle client messages (e.g., subscription changes)
		if msgType, ok := msg["type"].(string); ok {
			switch msgType {
			case "subscribe":
				if execID, ok := msg["execution_id"].(string); ok {
					if id, err := uuid.Parse(execID); err == nil {
						c.ExecutionID = &id
						c.Hub.log.WithFields(logrus.Fields{
							"client_id":    c.ID,
							"execution_id": id,
						}).Info("Client subscribed to execution updates")
					}
				}
			case "unsubscribe":
				c.ExecutionID = nil
				c.Hub.log.WithField("client_id", c.ID).Info("Client unsubscribed from execution updates")
			case "subscribe_recording":
				if sessionID, ok := msg["session_id"].(string); ok && sessionID != "" {
					c.RecordingSessionID = &sessionID
					c.Hub.log.WithFields(logrus.Fields{
						"client_id":  c.ID,
						"session_id": sessionID,
					}).Info("Client subscribed to recording updates")
					// Send confirmation
					select {
					case c.Send <- map[string]any{
						"type":       "recording_subscribed",
						"session_id": sessionID,
						"timestamp":  getCurrentTimestamp(),
					}:
					default:
					}
				}
			case "unsubscribe_recording":
				c.RecordingSessionID = nil
				c.Hub.log.WithField("client_id", c.ID).Info("Client unsubscribed from recording updates")
			}
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	defer c.Conn.Close()

	for {
		select {
		case update, ok := <-c.Send:
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(update); err != nil {
				c.Hub.log.WithError(err).Error("Failed to write WebSocket message")
				return
			}
		}
	}
}

// getCurrentTimestamp returns the current timestamp as ISO8601 string
func getCurrentTimestamp() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}

func extractExecutionID(msg any) *uuid.UUID {
	switch v := msg.(type) {
	case contracts.EventEnvelope:
		return &v.ExecutionID
	case *contracts.EventEnvelope:
		return &v.ExecutionID
	case map[string]any:
		if env, ok := v["data"].(contracts.EventEnvelope); ok {
			return &env.ExecutionID
		}
		if raw, ok := v["execution_id"]; ok {
			if s, ok := raw.(string); ok {
				if id, err := uuid.Parse(s); err == nil {
					return &id
				}
			}
		}
		if raw, ok := v["executionId"]; ok {
			if s, ok := raw.(string); ok {
				if id, err := uuid.Parse(s); err == nil {
					return &id
				}
			}
		}
	default:
		return nil
	}
	return nil
}
