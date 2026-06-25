package graph

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Broker manages WebSocket client connections and broadcasts graph events.
// Adapted from ecosystem-manager's websocket.Manager pattern.
type Broker struct {
	clients   map[*websocket.Conn]bool
	mu        sync.RWMutex
	broadcast chan WSMessage
	done      chan struct{}
}

// NewBroker creates and starts a Broker.
func NewBroker() *Broker {
	b := &Broker{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan WSMessage, 64),
		done:      make(chan struct{}),
	}
	go b.startBroadcaster()
	go b.startHeartbeat()
	return b
}

// AddClient registers a WebSocket connection.
func (b *Broker) AddClient(conn *websocket.Conn) {
	b.mu.Lock()
	b.clients[conn] = true
	b.mu.Unlock()
}

// RemoveClient unregisters a WebSocket connection.
func (b *Broker) RemoveClient(conn *websocket.Conn) {
	b.mu.Lock()
	delete(b.clients, conn)
	b.mu.Unlock()
}

// BroadcastUpdate wraps data in a WSMessage envelope and queues it for broadcast.
// Non-blocking — drops the message if the broadcast channel is full.
func (b *Broker) BroadcastUpdate(event string, payload any) {
	msg := NewWSMessage(event, payload)
	select {
	case b.broadcast <- msg:
	default:
		slog.Warn("broadcast channel full, dropping event", "event", event)
	}
}

// Stop shuts down the broker goroutines.
func (b *Broker) Stop() {
	close(b.done)
}

// startBroadcaster reads from the broadcast channel and sends to all clients.
func (b *Broker) startBroadcaster() {
	for {
		select {
		case msg := <-b.broadcast:
			b.broadcastToAll(msg)
		case <-b.done:
			return
		}
	}
}

// startHeartbeat sends a heartbeat message every 30 seconds.
func (b *Broker) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			b.BroadcastUpdate(WSHeartbeat, struct{}{})
		case <-b.done:
			return
		}
	}
}

// broadcastToAll sends a message to every connected client. Removes clients on write error.
func (b *Broker) broadcastToAll(msg WSMessage) {
	b.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(b.clients))
	for c := range b.clients {
		clients = append(clients, c)
	}
	b.mu.RUnlock()

	for _, conn := range clients {
		if err := conn.WriteJSON(msg); err != nil {
			slog.Warn("write error, removing client", "error", err)
			b.RemoveClient(conn)
			if closeErr := conn.Close(); closeErr != nil {
				slog.Debug("graph: close client conn failed", "err", closeErr)
			}
		}
	}
}

// ClientCount returns the number of connected clients.
func (b *Broker) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}
