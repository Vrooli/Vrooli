package websocket

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// ExecutionUpdate represents a real-time update for workflow execution
type ExecutionUpdate struct {
	Type        string    `json:"type"` // "progress", "log", "screenshot", "completed", "failed"
	ExecutionID uuid.UUID `json:"execution_id"`
	Progress    int       `json:"progress,omitempty"`
	CurrentStep string    `json:"current_step,omitempty"`
	Status      string    `json:"status,omitempty"`
	Message     string    `json:"message,omitempty"`
	Data        any       `json:"data,omitempty"`
	Timestamp   string    `json:"timestamp"`
}

// Client represents a WebSocket client
type Client struct {
	ID          uuid.UUID
	Conn        *websocket.Conn
	Send        chan ExecutionUpdate
	Hub         *Hub
	ExecutionID *uuid.UUID // Optional: client can subscribe to specific execution
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan ExecutionUpdate
	register   chan *Client
	unregister chan *Client
	log        *logrus.Logger
	mu         sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub(log *logrus.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan ExecutionUpdate),
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
			case client.Send <- ExecutionUpdate{
				Type:      "connected",
				Message:   "Connected to Browser Automation Studio",
				Timestamp: getCurrentTimestamp(),
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
			for client := range h.clients {
				// If client is subscribed to a specific execution, filter updates
				if client.ExecutionID != nil && *client.ExecutionID != update.ExecutionID {
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
func (h *Hub) BroadcastUpdate(update ExecutionUpdate) {
	update.Timestamp = getCurrentTimestamp()
	h.broadcast <- update
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ServeWS handles WebSocket requests from clients
func (h *Hub) ServeWS(conn *websocket.Conn, executionID *uuid.UUID) {
	client := &Client{
		ID:          uuid.New(),
		Conn:        conn,
		Send:        make(chan ExecutionUpdate, 256),
		Hub:         h,
		ExecutionID: executionID,
	}

	// Register the client
	client.Hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the websocket connection to the hub
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
			}
		}
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	defer c.Conn.Close()

	for {
		select {
		case update, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
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
