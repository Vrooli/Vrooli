package websocket

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Manager handles WebSocket connections and broadcasting
type Manager struct {
	upgrader     websocket.Upgrader
	clients      map[*websocket.Conn]bool
	clientsMutex sync.RWMutex
	broadcast    chan interface{}
}

// NewManager creates a new WebSocket manager
func NewManager() *Manager {
	manager := &Manager{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan interface{}),
	}

	// Start the broadcaster goroutine
	go manager.startBroadcaster()

	return manager
}

// GetBroadcastChannel returns the broadcast channel for sending updates
func (m *Manager) GetBroadcastChannel() chan<- interface{} {
	return m.broadcast
}

// HandleWebSocket handles new WebSocket connections
func (m *Manager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Register client
	m.clientsMutex.Lock()
	m.clients[conn] = true
	clientCount := len(m.clients)
	m.clientsMutex.Unlock()

	log.Printf("WebSocket client connected. Total clients: %d", clientCount)

	// Send initial state
	conn.WriteJSON(map[string]interface{}{
		"type":      "connected",
		"message":   "Connected to Ecosystem Manager",
		"timestamp": time.Now().Unix(),
	})

	// Cleanup on disconnect
	defer func() {
		m.clientsMutex.Lock()
		delete(m.clients, conn)
		clientCount := len(m.clients)
		m.clientsMutex.Unlock()
		log.Printf("WebSocket client disconnected. Total clients: %d", clientCount)
	}()

	// Keep connection alive and handle incoming messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		// For now, we don't process incoming messages from clients
		// This is just to keep the connection alive
	}
}

// startBroadcaster runs the main broadcast loop
func (m *Manager) startBroadcaster() {
	for {
		update := <-m.broadcast
		m.broadcastToAll(update)
	}
}

// broadcastToAll sends an update to all connected clients
func (m *Manager) broadcastToAll(update interface{}) {
	m.clientsMutex.RLock()
	defer m.clientsMutex.RUnlock()

	for client := range m.clients {
		err := client.WriteJSON(update)
		if err != nil {
			log.Printf("WebSocket write error: %v", err)
			client.Close()
			// The cleanup will happen in the HandleWebSocket defer
		}
	}
}

// BroadcastUpdate sends a typed update to all clients
func (m *Manager) BroadcastUpdate(updateType string, data interface{}) {
	update := map[string]interface{}{
		"type":      updateType,
		"data":      data,
		"timestamp": time.Now().Unix(),
	}

	// Non-blocking send
	select {
	case m.broadcast <- update:
	default:
		log.Printf("Warning: WebSocket broadcast channel full, dropping update")
	}
}

// GetClientCount returns the current number of connected clients
func (m *Manager) GetClientCount() int {
	m.clientsMutex.RLock()
	defer m.clientsMutex.RUnlock()
	return len(m.clients)
}
