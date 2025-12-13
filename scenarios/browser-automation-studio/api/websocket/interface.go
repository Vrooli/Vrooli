package websocket

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Re-export gorilla/websocket constants for use by handlers
const (
	BinaryMessage = websocket.BinaryMessage
	TextMessage   = websocket.TextMessage
	CloseMessage  = websocket.CloseMessage
)

// IsCloseError checks if the error is a WebSocket close error.
// Returns true for normal closure errors that don't need to be logged as errors.
func IsCloseError(err error) bool {
	return websocket.IsCloseError(err,
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseNoStatusReceived,
	)
}

// HubInterface defines the interface for WebSocket hub operations
// This interface allows for easier testing by enabling mock implementations
type HubInterface interface {
	// ServeWS handles WebSocket requests from clients
	// It creates a new client, registers it with the hub, and starts message pumps
	ServeWS(conn *websocket.Conn, executionID *uuid.UUID)

	// BroadcastEnvelope sends a normalized automation event envelope to all connected clients.
	BroadcastEnvelope(event any)

	// BroadcastRecordingAction sends a recording action to clients subscribed to a specific session.
	BroadcastRecordingAction(sessionID string, action any)

	// BroadcastRecordingActionWithTimeline sends a recording action with a unified TimelineEvent.
	// This is the V2 format that includes both the legacy action and the timeline_event field.
	BroadcastRecordingActionWithTimeline(sessionID string, action any, timelineEvent map[string]any)

	// BroadcastRecordingFrame sends a frame to clients subscribed to a specific recording session.
	// This eliminates the need for clients to poll for frames.
	BroadcastRecordingFrame(sessionID string, frame *RecordingFrame)

	// BroadcastBinaryFrame sends raw binary frame data (JPEG bytes) to clients subscribed to a recording session.
	// More efficient than BroadcastRecordingFrame as it avoids base64 encoding overhead.
	BroadcastBinaryFrame(sessionID string, jpegData []byte)

	// HasRecordingSubscribers returns true if any clients are subscribed to the given session.
	// Used by the frame push endpoint to avoid unnecessary work.
	HasRecordingSubscribers(sessionID string) bool

	// BroadcastPerfStats sends performance statistics to clients subscribed to a recording session.
	// Used by the debug performance mode to stream aggregated timing data.
	BroadcastPerfStats(sessionID string, stats any)

	// GetClientCount returns the number of currently connected clients
	GetClientCount() int

	// Run starts the hub's main event loop
	// This should be called in a goroutine when the hub is created
	Run()

	// CloseExecution drains and tears down per-execution queues when the run
	// finishes. Optional; hubs may no-op.
	CloseExecution(executionID uuid.UUID)
}
