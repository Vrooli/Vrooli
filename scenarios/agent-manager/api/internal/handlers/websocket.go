package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocket upgrader with permissive origin check for development
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// In production, this should validate against allowed origins
		return true
	},
}

// WebSocketHub manages WebSocket connections and event broadcasting
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan *WebSocketMessage
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mu         sync.RWMutex
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	hub           *WebSocketHub
	conn          *websocket.Conn
	send          chan []byte
	subscriptions map[string]bool // runID -> subscribed
	allEvents     bool            // subscribe to all events
}

// WebSocketMessage represents a message to broadcast
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	RunID   string      `json:"runId,omitempty"`
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan *WebSocketMessage, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
	}
}

// Run starts the hub's main loop
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			count := len(h.clients)
			h.mu.Unlock()
			log.Printf("[ws] Client connected. Total: %d", count)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			count := len(h.clients)
			h.mu.Unlock()
			log.Printf("[ws] Client disconnected. Total: %d", count)

		case message := <-h.broadcast:
			data, err := json.Marshal(message)
			if err != nil {
				log.Printf("[ws] Failed to marshal message: %v", err)
				continue
			}

			h.mu.RLock()
			for client := range h.clients {
				// Check if client should receive this message
				if message.RunID != "" && !client.allEvents {
					if !client.subscriptions[message.RunID] {
						continue
					}
				}

				select {
				case client.send <- data:
				default:
					// Client buffer full, close connection
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastEvent broadcasts a run event to subscribed clients
func (h *WebSocketHub) BroadcastEvent(event *domain.RunEvent) {
	h.broadcast <- &WebSocketMessage{
		Type:    "run_event",
		Payload: event,
		RunID:   event.RunID.String(),
	}
}

// BroadcastRunStatus broadcasts a run status change
func (h *WebSocketHub) BroadcastRunStatus(run *domain.Run) {
	h.broadcast <- &WebSocketMessage{
		Type: "run_status",
		Payload: map[string]interface{}{
			"id":     run.ID.String(),
			"status": string(run.Status),
		},
		RunID: run.ID.String(),
	}
}

// BroadcastTaskStatus broadcasts a task status change
func (h *WebSocketHub) BroadcastTaskStatus(task *domain.Task) {
	h.broadcast <- &WebSocketMessage{
		Type: "task_status",
		Payload: map[string]interface{}{
			"id":     task.ID.String(),
			"status": string(task.Status),
		},
	}
}

// BroadcastMessage broadcasts a generic message
func (h *WebSocketHub) BroadcastMessage(msgType string, payload interface{}) {
	h.broadcast <- &WebSocketMessage{
		Type:    msgType,
		Payload: payload,
	}
}

// BroadcastProgress broadcasts a progress update for a run.
// This implements the orchestration.EventBroadcaster interface.
func (h *WebSocketHub) BroadcastProgress(runID uuid.UUID, phase domain.RunPhase, percent int, action string) {
	h.broadcast <- &WebSocketMessage{
		Type: "run_progress",
		Payload: map[string]interface{}{
			"runId":           runID.String(),
			"phase":           string(phase),
			"percentComplete": percent,
			"currentAction":   action,
		},
		RunID: runID.String(),
	}
}

// HandleWebSocket handles WebSocket connections
func (h *Handler) HandleWebSocket(hub *WebSocketHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[ws] Upgrade error: %v", err)
			return
		}

		client := &WebSocketClient{
			hub:           hub,
			conn:          conn,
			send:          make(chan []byte, 256),
			subscriptions: make(map[string]bool),
			allEvents:     false,
		}

		hub.register <- client

		// Send welcome message
		welcome := WebSocketMessage{
			Type: "connected",
			Payload: map[string]interface{}{
				"message":   "Connected to agent-manager WebSocket",
				"timestamp": time.Now().Format(time.RFC3339),
			},
		}
		welcomeData, _ := json.Marshal(welcome)
		client.send <- welcomeData

		// Start read and write goroutines
		go client.writePump()
		go client.readPump()
	}
}

// readPump handles incoming messages from a client
func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB max message size
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[ws] Read error: %v", err)
			}
			break
		}

		// Parse incoming message
		var msg struct {
			Type    string `json:"type"`
			Payload struct {
				RunID string `json:"runId,omitempty"`
			} `json:"payload,omitempty"`
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("[ws] Failed to parse message: %v", err)
			continue
		}

		// Handle message types
		switch msg.Type {
		case "ping":
			// Respond with pong
			pong := WebSocketMessage{
				Type: "pong",
				Payload: map[string]interface{}{
					"timestamp": time.Now().Format(time.RFC3339),
				},
			}
			data, _ := json.Marshal(pong)
			c.send <- data

		case "subscribe":
			// Subscribe to a specific run
			if msg.Payload.RunID != "" {
				if _, err := uuid.Parse(msg.Payload.RunID); err == nil {
					c.subscriptions[msg.Payload.RunID] = true
					log.Printf("[ws] Client subscribed to run: %s", msg.Payload.RunID)
				}
			}

		case "unsubscribe":
			// Unsubscribe from a specific run
			if msg.Payload.RunID != "" {
				delete(c.subscriptions, msg.Payload.RunID)
				log.Printf("[ws] Client unsubscribed from run: %s", msg.Payload.RunID)
			}

		case "subscribe_all":
			// Subscribe to all events
			c.allEvents = true
			log.Printf("[ws] Client subscribed to all events")

		case "unsubscribe_all":
			// Unsubscribe from all events
			c.allEvents = false
			log.Printf("[ws] Client unsubscribed from all events")
		}
	}
}

// writePump handles outgoing messages to a client
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Batch any queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
