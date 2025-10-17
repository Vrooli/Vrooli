package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketManager manages WebSocket connections for real-time updates
type WebSocketManager struct {
	clients    map[string]*WebSocketClient
	broadcast  chan []byte
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mutex      sync.RWMutex
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	ID     string
	UserID string
	Conn   *websocket.Conn
	Send   chan []byte
}

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type      string      `json:"type"`
	UserID    string      `json:"user_id,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp string      `json:"timestamp"`
}

// Real-time update types
const (
	MessageTypePostScheduled = "post_scheduled"
	MessageTypePostPosted    = "post_posted"
	MessageTypePostFailed    = "post_failed"
	MessageTypePostUpdated   = "post_updated"
	MessageTypePostDeleted   = "post_deleted"
	MessageTypeAccountConnected = "account_connected"
	MessageTypeAccountDisconnected = "account_disconnected"
	MessageTypeQueueStatus   = "queue_status"
	MessageTypeAnalyticsUpdate = "analytics_update"
)

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager() *WebSocketManager {
	manager := &WebSocketManager{
		clients:    make(map[string]*WebSocketClient),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
	}

	// Start the message hub
	go manager.run()

	return manager
}

// run starts the WebSocket message hub
func (wsm *WebSocketManager) run() {
	for {
		select {
		case client := <-wsm.register:
			wsm.mutex.Lock()
			wsm.clients[client.ID] = client
			wsm.mutex.Unlock()
			log.Printf("📡 WebSocket client %s connected (User: %s)", client.ID, client.UserID)
			
			// Send welcome message
			welcome := WebSocketMessage{
				Type:      "connected",
				Data:      map[string]string{"message": "Connected to Social Media Scheduler"},
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			client.sendMessage(welcome)

		case client := <-wsm.unregister:
			wsm.mutex.Lock()
			if _, ok := wsm.clients[client.ID]; ok {
				delete(wsm.clients, client.ID)
				close(client.Send)
				log.Printf("📡 WebSocket client %s disconnected", client.ID)
			}
			wsm.mutex.Unlock()

		case message := <-wsm.broadcast:
			wsm.mutex.RLock()
			for _, client := range wsm.clients {
				select {
				case client.Send <- message:
				default:
					// Client buffer full, close connection
					delete(wsm.clients, client.ID)
					close(client.Send)
				}
			}
			wsm.mutex.RUnlock()
		}
	}
}

// Register adds a new WebSocket client
func (wsm *WebSocketManager) Register(client *WebSocketClient) {
	wsm.register <- client
}

// Unregister removes a WebSocket client
func (wsm *WebSocketManager) Unregister(client *WebSocketClient) {
	wsm.unregister <- client
}

// BroadcastToUser sends a message to all connections for a specific user
func (wsm *WebSocketManager) BroadcastToUser(userID string, message WebSocketMessage) {
	message.UserID = userID
	message.Timestamp = time.Now().UTC().Format(time.RFC3339)

	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Printf("⚠️  Error marshaling WebSocket message: %v", err)
		return
	}

	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()

	for _, client := range wsm.clients {
		if client.UserID == userID {
			select {
			case client.Send <- jsonData:
			default:
				// Client buffer full, skip this client
				log.Printf("⚠️  WebSocket client %s buffer full, skipping message", client.ID)
			}
		}
	}
}

// BroadcastToAll sends a message to all connected clients
func (wsm *WebSocketManager) BroadcastToAll(message WebSocketMessage) {
	message.Timestamp = time.Now().UTC().Format(time.RFC3339)

	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Printf("⚠️  Error marshaling WebSocket message: %v", err)
		return
	}

	wsm.broadcast <- jsonData
}

// NotifyPostScheduled sends a notification when a post is scheduled
func (wsm *WebSocketManager) NotifyPostScheduled(userID string, postData interface{}) {
	message := WebSocketMessage{
		Type: MessageTypePostScheduled,
		Data: postData,
	}
	wsm.BroadcastToUser(userID, message)
}

// NotifyPostPosted sends a notification when a post is successfully posted
func (wsm *WebSocketManager) NotifyPostPosted(userID string, postData interface{}) {
	message := WebSocketMessage{
		Type: MessageTypePostPosted,
		Data: postData,
	}
	wsm.BroadcastToUser(userID, message)
}

// NotifyPostFailed sends a notification when a post fails
func (wsm *WebSocketManager) NotifyPostFailed(userID string, postData interface{}) {
	message := WebSocketMessage{
		Type: MessageTypePostFailed,
		Data: postData,
	}
	wsm.BroadcastToUser(userID, message)
}

// NotifyPostUpdated sends a notification when a post is updated
func (wsm *WebSocketManager) NotifyPostUpdated(userID string, postData interface{}) {
	message := WebSocketMessage{
		Type: MessageTypePostUpdated,
		Data: postData,
	}
	wsm.BroadcastToUser(userID, message)
}

// NotifyPostDeleted sends a notification when a post is deleted
func (wsm *WebSocketManager) NotifyPostDeleted(userID string, postData interface{}) {
	message := WebSocketMessage{
		Type: MessageTypePostDeleted,
		Data: postData,
	}
	wsm.BroadcastToUser(userID, message)
}

// NotifyAccountConnected sends a notification when a social account is connected
func (wsm *WebSocketManager) NotifyAccountConnected(userID string, accountData interface{}) {
	message := WebSocketMessage{
		Type: MessageTypeAccountConnected,
		Data: accountData,
	}
	wsm.BroadcastToUser(userID, message)
}

// NotifyAccountDisconnected sends a notification when a social account is disconnected
func (wsm *WebSocketManager) NotifyAccountDisconnected(userID string, accountData interface{}) {
	message := WebSocketMessage{
		Type: MessageTypeAccountDisconnected,
		Data: accountData,
	}
	wsm.BroadcastToUser(userID, message)
}

// NotifyQueueStatus sends queue status updates
func (wsm *WebSocketManager) NotifyQueueStatus(queueData interface{}) {
	message := WebSocketMessage{
		Type: MessageTypeQueueStatus,
		Data: queueData,
	}
	wsm.BroadcastToAll(message)
}

// NotifyAnalyticsUpdate sends analytics updates to users
func (wsm *WebSocketManager) NotifyAnalyticsUpdate(userID string, analyticsData interface{}) {
	message := WebSocketMessage{
		Type: MessageTypeAnalyticsUpdate,
		Data: analyticsData,
	}
	wsm.BroadcastToUser(userID, message)
}

// Close closes the WebSocket manager and all connections
func (wsm *WebSocketManager) Close() {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()

	for _, client := range wsm.clients {
		client.Conn.Close()
		close(client.Send)
	}
	
	close(wsm.broadcast)
	close(wsm.register)
	close(wsm.unregister)

	log.Println("📡 WebSocket manager closed")
}

// GetConnectedClients returns the number of connected clients
func (wsm *WebSocketManager) GetConnectedClients() int {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()
	return len(wsm.clients)
}

// GetUserConnections returns the number of connections for a specific user
func (wsm *WebSocketManager) GetUserConnections(userID string) int {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()

	count := 0
	for _, client := range wsm.clients {
		if client.UserID == userID {
			count++
		}
	}
	return count
}

// WebSocket client methods
func (c *WebSocketClient) sendMessage(message WebSocketMessage) {
	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Printf("⚠️  Error marshaling message: %v", err)
		return
	}

	select {
	case c.Send <- jsonData:
	default:
		log.Printf("⚠️  Client %s send buffer full", c.ID)
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// The hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps messages from the websocket connection to the hub
func (c *WebSocketClient) readPump(wsm *WebSocketManager) {
	defer func() {
		wsm.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("📡 WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages from client
		var incomingMsg struct {
			Type string      `json:"type"`
			Data interface{} `json:"data"`
		}

		if err := json.Unmarshal(message, &incomingMsg); err != nil {
			log.Printf("⚠️  Error unmarshaling client message: %v", err)
			continue
		}

		// Handle different message types from client
		switch incomingMsg.Type {
		case "ping":
			// Respond with pong
			pongMsg := WebSocketMessage{
				Type: "pong",
				Data: map[string]string{"message": "pong"},
			}
			c.sendMessage(pongMsg)

		case "subscribe":
			// Handle subscription requests (e.g., subscribe to specific post updates)
			log.Printf("📡 Client %s subscribed to: %v", c.ID, incomingMsg.Data)

		case "unsubscribe":
			// Handle unsubscription requests
			log.Printf("📡 Client %s unsubscribed from: %v", c.ID, incomingMsg.Data)

		default:
			log.Printf("📡 Unknown message type from client %s: %s", c.ID, incomingMsg.Type)
		}
	}
}