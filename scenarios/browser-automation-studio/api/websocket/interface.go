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

	// BroadcastEnvelope sends a normalized automation event envelope to all connected clients.
	BroadcastEnvelope(event any)

	// GetClientCount returns the number of currently connected clients
	GetClientCount() int

	// Run starts the hub's main event loop
	// This should be called in a goroutine when the hub is created
	Run()

	// CloseExecution drains and tears down per-execution queues when the run
	// finishes. Optional; hubs may no-op.
	CloseExecution(executionID uuid.UUID)
}
