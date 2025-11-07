package websocket

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// HubInterface defines the interface for WebSocket hub operations
// This interface allows for easier testing by enabling mock implementations
type HubInterface interface {
	// ServeWS handles WebSocket requests from clients
	// It creates a new client, registers it with the hub, and starts message pumps
	ServeWS(conn *websocket.Conn, executionID *uuid.UUID)

	// BroadcastUpdate sends an update to all connected clients
	// If clients are subscribed to specific executions, updates are filtered accordingly
	BroadcastUpdate(update ExecutionUpdate)

	// GetClientCount returns the number of currently connected clients
	GetClientCount() int

	// Run starts the hub's main event loop
	// This should be called in a goroutine when the hub is created
	Run()
}
