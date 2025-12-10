package httpserver

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketManager handles WebSocket connections and broadcasting for agent updates
type WebSocketManager struct {
	upgrader     websocket.Upgrader
	clients      map[*websocket.Conn]bool
	clientsMutex sync.RWMutex
	broadcast    chan any
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager() *WebSocketManager {
	manager := &WebSocketManager{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan any, 100), // Buffered channel to prevent blocking
	}

	// Start the broadcaster goroutine
	go manager.startBroadcaster()

	return manager
}

// HandleWebSocket handles new WebSocket connections
func (m *WebSocketManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Register client
	m.clientsMutex.Lock()
	m.clients[conn] = true
	clientCount := len(m.clients)
	m.clientsMutex.Unlock()

	log.Printf("[WebSocket] Client connected. Total clients: %d", clientCount)

	// Send initial connection message
	if err := conn.WriteJSON(map[string]any{
		"type":      "connected",
		"message":   "Connected to Test Genie",
		"timestamp": time.Now().Unix(),
	}); err != nil {
		log.Printf("[WebSocket] Failed to send initial message: %v", err)
	}

	// Cleanup on disconnect
	defer func() {
		m.clientsMutex.Lock()
		delete(m.clients, conn)
		clientCount := len(m.clients)
		m.clientsMutex.Unlock()
		log.Printf("[WebSocket] Client disconnected. Total clients: %d", clientCount)
	}()

	// Keep connection alive and handle incoming messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket] Error: %v", err)
			}
			break
		}
		// We don't process incoming messages for now - this just keeps the connection alive
	}
}

// startBroadcaster runs the main broadcast loop
func (m *WebSocketManager) startBroadcaster() {
	for update := range m.broadcast {
		m.broadcastToAll(update)
	}
}

// broadcastToAll sends an update to all connected clients
func (m *WebSocketManager) broadcastToAll(update any) {
	m.clientsMutex.RLock()
	defer m.clientsMutex.RUnlock()

	for client := range m.clients {
		if err := client.WriteJSON(update); err != nil {
			log.Printf("[WebSocket] Write error: %v", err)
			client.Close()
			// The cleanup will happen in the HandleWebSocket defer
		}
	}
}

// BroadcastUpdate sends a typed update to all clients
func (m *WebSocketManager) BroadcastUpdate(updateType string, data any) {
	update := map[string]any{
		"type":      updateType,
		"data":      data,
		"timestamp": time.Now().Unix(),
	}

	// Non-blocking send to prevent blocking the caller
	select {
	case m.broadcast <- update:
	default:
		log.Printf("[WebSocket] Warning: broadcast channel full, dropping update")
	}
}

// BroadcastAgentUpdate broadcasts an agent status change
func (m *WebSocketManager) BroadcastAgentUpdate(agent *ActiveAgent) {
	m.BroadcastUpdate("agent_updated", map[string]any{
		"id":          agent.ID,
		"sessionId":   agent.SessionID,
		"scenario":    agent.Scenario,
		"scope":       agent.Scope,
		"phases":      agent.Phases,
		"model":       agent.Model,
		"status":      string(agent.Status),
		"startedAt":   agent.StartedAt.Format(time.RFC3339),
		"completedAt": formatCompletedAt(agent.CompletedAt),
		"promptHash":  agent.PromptHash,
		"promptIndex": agent.PromptIndex,
		"output":      agent.Output,
		"error":       agent.Error,
	})
}

// BroadcastAgentOutput broadcasts incremental output from a running agent
func (m *WebSocketManager) BroadcastAgentOutput(agentID string, output string, sequence int64) {
	m.BroadcastUpdate("agent_output", map[string]any{
		"agentId":  agentID,
		"output":   output,
		"sequence": sequence,
	})
}

// BroadcastAgentStopped broadcasts when an agent is stopped
func (m *WebSocketManager) BroadcastAgentStopped(agentID string) {
	m.BroadcastUpdate("agent_stopped", map[string]any{
		"agentId": agentID,
	})
}

// BroadcastAgentsStoppedAll broadcasts when all agents are stopped
func (m *WebSocketManager) BroadcastAgentsStoppedAll(stoppedCount int) {
	m.BroadcastUpdate("agents_stopped_all", map[string]any{
		"stoppedCount": stoppedCount,
	})
}

// GetClientCount returns the current number of connected clients
func (m *WebSocketManager) GetClientCount() int {
	m.clientsMutex.RLock()
	defer m.clientsMutex.RUnlock()
	return len(m.clients)
}

func formatCompletedAt(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
