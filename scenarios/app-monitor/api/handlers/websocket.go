package handlers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	upgrader *websocket.Upgrader
	redis    *redis.Client
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(upgrader *websocket.Upgrader, redis *redis.Client) *WebSocketHandler {
	return &WebSocketHandler{
		upgrader: upgrader,
		redis:    redis,
	}
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// HandleWebSocket handles WebSocket upgrade and connection
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	// Send initial connection message
	if err := conn.WriteJSON(WebSocketMessage{
		Type:    "connection",
		Payload: map[string]string{"status": "connected"},
	}); err != nil {
		log.Printf("Failed to send connection message: %v", err)
		return
	}

	// If Redis is available, subscribe to events
	if h.redis != nil {
		ctx := context.Background()
		pubsub := h.redis.Subscribe(ctx, "app-events")
		defer pubsub.Close()

		ch := pubsub.Channel()

		// Create channels for communication
		done := make(chan struct{})
		defer close(done)

		// Handle incoming messages from client
		go h.handleClientMessages(conn, done)

		// Send Redis events to client
		for {
			select {
			case msg := <-ch:
				var event map[string]interface{}
				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					continue
				}

				message := WebSocketMessage{
					Type:    "app_update",
					Payload: event,
				}

				if err := conn.WriteJSON(message); err != nil {
					log.Printf("Failed to write WebSocket message: %v", err)
					return
				}
			case <-done:
				return
			}
		}
	} else {
		// No Redis, just handle client messages
		h.handleClientMessages(conn, nil)
	}
}

// handleClientMessages handles incoming messages from the client
func (h *WebSocketHandler) handleClientMessages(conn *websocket.Conn, done chan struct{}) {
	defer func() {
		if done != nil {
			done <- struct{}{}
		}
	}()

	// Set read deadline to detect disconnected clients
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	for {
		var msg WebSocketMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle different message types
		switch msg.Type {
		case "ping":
			// Respond with pong
			if err := conn.WriteJSON(WebSocketMessage{
				Type:    "pong",
				Payload: map[string]string{"timestamp": time.Now().Format(time.RFC3339)},
			}); err != nil {
				log.Printf("Failed to send pong: %v", err)
				break
			}

		case "subscribe":
			// Handle subscription requests
			if appID, ok := msg.Payload.(map[string]interface{})["appId"].(string); ok {
				log.Printf("Client subscribed to app: %s", appID)
				// In the future, implement app-specific subscriptions
			}

		case "unsubscribe":
			// Handle unsubscription requests
			if appID, ok := msg.Payload.(map[string]interface{})["appId"].(string); ok {
				log.Printf("Client unsubscribed from app: %s", appID)
				// In the future, implement app-specific unsubscriptions
			}

		case "command":
			// Handle command execution requests
			if cmd, ok := msg.Payload.(map[string]interface{})["command"].(string); ok {
				log.Printf("Command received: %s", cmd)
				// Send response
				if err := conn.WriteJSON(WebSocketMessage{
					Type: "command_response",
					Payload: map[string]string{
						"command": cmd,
						"result":  "Command execution not implemented",
					},
				}); err != nil {
					log.Printf("Failed to send command response: %v", err)
				}
			}

		default:
			log.Printf("Unknown message type: %s", msg.Type)
		}
	}
}