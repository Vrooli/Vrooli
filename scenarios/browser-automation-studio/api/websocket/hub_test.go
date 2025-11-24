package websocket

import (
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

func newTestHubBase(t *testing.T) *Hub {
	t.Helper()

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	hub := NewHub(logger)
	go hub.Run()
	return hub
}

func newTestHub(t *testing.T) *Hub {
	return newTestHubBase(t)
}

func waitForMessage(t *testing.T, ch <-chan any) any {
	t.Helper()

	select {
	case update := <-ch:
		return update
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for update")
	}
	return nil
}

type mockHub struct{}

func (m *mockHub) ServeWS(conn *websocket.Conn, executionID *uuid.UUID) {}
func (m *mockHub) BroadcastEnvelope(event any)                          {}
func (m *mockHub) GetClientCount() int                                  { return 0 }
func (m *mockHub) Run()                                                 {}
func (m *mockHub) CloseExecution(executionID uuid.UUID)                 {}

func TestHubBroadcastsToRegisteredClients(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] broadcasts execution updates to all connected clients", func(t *testing.T) {
		hub := newTestHub(t)

		client := &Client{
			ID:   uuid.New(),
			Send: make(chan any, 4),
			Hub:  hub,
		}

		hub.register <- client

		// Drain welcome message to avoid interfering with assertions.
		_ = waitForMessage(t, client.Send)

		update := contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           contracts.EventKindExecutionProgress,
			ExecutionID:    uuid.New(),
			WorkflowID:     uuid.New(),
			Sequence:       1,
		}

		hub.BroadcastEnvelope(update)

		msg := waitForMessage(t, client.Send)
		env, ok := msg.(contracts.EventEnvelope)
		if !ok {
			t.Fatalf("expected envelope, got %T", msg)
		}
		if env.Kind != update.Kind || env.ExecutionID != update.ExecutionID {
			t.Fatalf("expected %+v, got %+v", update, env)
		}
	})
}

func TestHubFiltersByExecutionID(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] filters updates by execution ID for subscribed clients", func(t *testing.T) {
		hub := newTestHub(t)

		executionID := uuid.New()

		broadcastClient := &Client{
			ID:   uuid.New(),
			Send: make(chan any, 4),
			Hub:  hub,
		}

		filteredClient := &Client{
			ID:          uuid.New(),
			Send:        make(chan any, 4),
			Hub:         hub,
			ExecutionID: &executionID,
		}

		hub.register <- broadcastClient
		hub.register <- filteredClient

		// Drain welcome updates
		_ = waitForMessage(t, broadcastClient.Send)
		_ = waitForMessage(t, filteredClient.Send)

		matching := contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           contracts.EventKindExecutionProgress,
			ExecutionID:    executionID,
			WorkflowID:     uuid.New(),
			Sequence:       1,
		}
		different := contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           contracts.EventKindExecutionProgress,
			ExecutionID:    uuid.New(),
			WorkflowID:     uuid.New(),
			Sequence:       2,
		}

		hub.BroadcastEnvelope(matching)
		hub.BroadcastEnvelope(different)

		receivedBroadcast := waitForMessage(t, broadcastClient.Send)
		envBroadcast, ok := receivedBroadcast.(contracts.EventEnvelope)
		if !ok || envBroadcast.ExecutionID != matching.ExecutionID {
			t.Fatalf("expected broadcast client to receive matching envelope, got %+v", receivedBroadcast)
		}

		receivedFiltered := waitForMessage(t, filteredClient.Send)
		envFiltered, ok := receivedFiltered.(contracts.EventEnvelope)
		if !ok || envFiltered.ExecutionID != executionID {
			t.Fatalf("filtered client received wrong execution: %+v", receivedFiltered)
		}

		select {
		case extra := <-filteredClient.Send:
			t.Fatalf("filtered client should not receive non-matching updates, got %+v", extra)
		case <-time.After(200 * time.Millisecond):
			// expected
		}
	})
}

func TestBroadcastEnvelopeRawWhenLegacyDisabled(t *testing.T) {
	hub := newTestHub(t)

	client := &Client{
		ID:   uuid.New(),
		Send: make(chan any, 4),
		Hub:  hub,
	}
	hub.register <- client
	_ = waitForMessage(t, client.Send) // welcome

	env := contracts.EventEnvelope{
		SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		Kind:           contracts.EventKindExecutionStarted,
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Sequence:       1,
		Timestamp:      time.Now().UTC(),
		Payload:        map[string]any{"note": "raw"},
	}

	hub.BroadcastEnvelope(env)

	msg := waitForMessage(t, client.Send)
	payload, ok := msg.(contracts.EventEnvelope)
	if !ok {
		t.Fatalf("expected raw envelope, got %T", msg)
	}
	if payload.Kind != env.Kind || payload.ExecutionID != env.ExecutionID {
		t.Fatalf("envelope payload mismatch: %+v", payload)
	}
}

func TestGetClientCount(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] tracks connected client count", func(t *testing.T) {
		hub := newTestHub(t)

		if count := hub.GetClientCount(); count != 0 {
			t.Fatalf("expected 0 clients, got %d", count)
		}

		client1 := &Client{
			ID:   uuid.New(),
			Send: make(chan any, 4),
			Hub:  hub,
		}
		client2 := &Client{
			ID:   uuid.New(),
			Send: make(chan any, 4),
			Hub:  hub,
		}

		hub.register <- client1
		time.Sleep(50 * time.Millisecond)

		if count := hub.GetClientCount(); count != 1 {
			t.Fatalf("expected 1 client after registration, got %d", count)
		}

		hub.register <- client2
		time.Sleep(50 * time.Millisecond)

		if count := hub.GetClientCount(); count != 2 {
			t.Fatalf("expected 2 clients after second registration, got %d", count)
		}

		hub.unregister <- client1
		time.Sleep(50 * time.Millisecond)

		if count := hub.GetClientCount(); count != 1 {
			t.Fatalf("expected 1 client after unregister, got %d", count)
		}
	})
}

// TestHubSendsWelcomeMessage verifies that new clients receive a welcome message upon connection
func TestHubSendsWelcomeMessage(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] sends welcome message to newly connected clients", func(t *testing.T) {
		hub := newTestHub(t)

		client := &Client{
			ID:   uuid.New(),
			Send: make(chan any, 4),
			Hub:  hub,
		}

		hub.register <- client

		msg := waitForMessage(t, client.Send)
		welcome, ok := msg.(map[string]any)
		if !ok {
			t.Fatalf("expected welcome map, got %T", msg)
		}
		if welcome["type"] != "connected" {
			t.Errorf("expected welcome message type 'connected', got %v", welcome["type"])
		}
		if welcome["message"] != "Connected to Browser Automation Studio" {
			t.Errorf("expected welcome message, got %v", welcome["message"])
		}
	})
}

// TestHubHandlesMultipleBroadcasts verifies hub can handle rapid sequential broadcasts
func TestHubHandlesMultipleBroadcasts(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] handles multiple rapid broadcasts", func(t *testing.T) {
		hub := newTestHub(t)

		client := &Client{
			ID:   uuid.New(),
			Send: make(chan any, 10),
			Hub:  hub,
		}

		hub.register <- client
		_ = waitForMessage(t, client.Send) // Drain welcome message

		executionID := uuid.New()

		// Send multiple updates rapidly
		for i := 0; i < 5; i++ {
			hub.BroadcastEnvelope(contracts.EventEnvelope{
				SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
				PayloadVersion: contracts.PayloadVersion,
				Kind:           contracts.EventKindExecutionProgress,
				ExecutionID:    executionID,
				WorkflowID:     uuid.New(),
				Sequence:       uint64(i + 1),
				Payload:        map[string]any{"progress": i * 20},
			})
		}

		// Verify all updates were received
		receivedProgress := make([]int, 0, 5)
		for i := 0; i < 5; i++ {
			msg := waitForMessage(t, client.Send)
			env, ok := msg.(contracts.EventEnvelope)
			if !ok {
				t.Fatalf("expected envelope, got %T", msg)
			}
			progress := 0
			if m, ok := env.Payload.(map[string]any); ok {
				if p, ok := m["progress"].(int); ok {
					progress = p
				} else if pFloat, ok := m["progress"].(float64); ok {
					progress = int(pFloat)
				}
			}
			receivedProgress = append(receivedProgress, progress)
		}

		if len(receivedProgress) != 5 {
			t.Errorf("expected 5 progress updates, got %d", len(receivedProgress))
		}
	})
}

// TestHubCleanupOnClientDisconnect verifies proper cleanup when client disconnects
func TestHubCleanupOnClientDisconnect(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] cleans up disconnected clients", func(t *testing.T) {
		hub := newTestHub(t)

		client := &Client{
			ID:   uuid.New(),
			Send: make(chan any, 4),
			Hub:  hub,
		}

		hub.register <- client
		time.Sleep(50 * time.Millisecond)

		initialCount := hub.GetClientCount()
		if initialCount != 1 {
			t.Fatalf("expected 1 client after registration, got %d", initialCount)
		}

		hub.unregister <- client
		time.Sleep(50 * time.Millisecond)

		finalCount := hub.GetClientCount()
		if finalCount != 0 {
			t.Errorf("expected 0 clients after unregister, got %d", finalCount)
		}

		// Verify the client is removed from the hub - channel closing is implementation detail
		// that's handled asynchronously, so we just verify client count decreased
	})
}
