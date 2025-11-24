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

func newTestHub(t *testing.T) *Hub {
	t.Helper()

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	hub := NewHub(logger)
	go hub.Run()
	return hub
}

func waitForUpdate(t *testing.T, ch <-chan ExecutionUpdate) ExecutionUpdate {
	t.Helper()

	select {
	case update := <-ch:
		return update
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for update")
	}
	return ExecutionUpdate{}
}

type mockHub struct{}

func (m *mockHub) ServeWS(conn *websocket.Conn, executionID *uuid.UUID) {}
func (m *mockHub) BroadcastEnvelope(event any)                          {}
func (m *mockHub) BroadcastUpdate(update ExecutionUpdate)               {}
func (m *mockHub) GetClientCount() int                                  { return 0 }
func (m *mockHub) Run()                                                 {}
func (m *mockHub) CloseExecution(executionID uuid.UUID)                 {}

func TestHubBroadcastsToRegisteredClients(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] broadcasts execution updates to all connected clients", func(t *testing.T) {
		hub := newTestHub(t)

		client := &Client{
			ID:   uuid.New(),
			Send: make(chan ExecutionUpdate, 4),
			Hub:  hub,
		}

		hub.register <- client

		// Drain welcome message to avoid interfering with assertions.
		_ = waitForUpdate(t, client.Send)

		update := ExecutionUpdate{
			Type:        "progress",
			ExecutionID: uuid.New(),
			Progress:    42,
		}

		hub.BroadcastUpdate(update)

		received := waitForUpdate(t, client.Send)
		if received.Type != update.Type || received.Progress != update.Progress {
			t.Fatalf("expected %+v, got %+v", update, received)
		}
	})
}

func TestHubFiltersByExecutionID(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] filters updates by execution ID for subscribed clients", func(t *testing.T) {
		hub := newTestHub(t)

		executionID := uuid.New()

		broadcastClient := &Client{
			ID:   uuid.New(),
			Send: make(chan ExecutionUpdate, 4),
			Hub:  hub,
		}

		filteredClient := &Client{
			ID:          uuid.New(),
			Send:        make(chan ExecutionUpdate, 4),
			Hub:         hub,
			ExecutionID: &executionID,
		}

		hub.register <- broadcastClient
		hub.register <- filteredClient

		// Drain welcome updates
		_ = waitForUpdate(t, broadcastClient.Send)
		_ = waitForUpdate(t, filteredClient.Send)

		matching := ExecutionUpdate{
			Type:        "log",
			ExecutionID: executionID,
			Message:     "hello",
		}
		different := ExecutionUpdate{
			Type:        "log",
			ExecutionID: uuid.New(),
			Message:     "ignored",
		}

		hub.BroadcastUpdate(matching)
		hub.BroadcastUpdate(different)

		receivedBroadcast := waitForUpdate(t, broadcastClient.Send)
		if receivedBroadcast.Message != matching.Message {
			t.Fatalf("expected broadcast client to receive matching update, got %+v", receivedBroadcast)
		}

		receivedFiltered := waitForUpdate(t, filteredClient.Send)
		if receivedFiltered.ExecutionID != executionID {
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

func TestBroadcastEnvelopeDeliversEnvelopePayload(t *testing.T) {
	hub := newTestHub(t)

	client := &Client{
		ID:   uuid.New(),
		Send: make(chan ExecutionUpdate, 4),
		Hub:  hub,
	}
	hub.register <- client
	_ = waitForUpdate(t, client.Send) // welcome

	env := contracts.EventEnvelope{
		SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		Kind:           contracts.EventKindStepCompleted,
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Sequence:       5,
		Timestamp:      time.Now().UTC(),
		Payload:        map[string]any{"note": "ok"},
	}

	hub.BroadcastEnvelope(env)

	received := waitForUpdate(t, client.Send)
	payload, ok := received.Data.(contracts.EventEnvelope)
	if !ok {
		t.Fatalf("expected envelope payload, got %T", received.Data)
	}
	if payload.Kind != env.Kind || payload.Sequence != env.Sequence {
		t.Fatalf("payload mismatch: %+v", payload)
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
			Send: make(chan ExecutionUpdate, 4),
			Hub:  hub,
		}
		client2 := &Client{
			ID:   uuid.New(),
			Send: make(chan ExecutionUpdate, 4),
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
			Send: make(chan ExecutionUpdate, 4),
			Hub:  hub,
		}

		hub.register <- client

		welcome := waitForUpdate(t, client.Send)
		if welcome.Type != "connected" {
			t.Errorf("expected welcome message type 'connected', got %s", welcome.Type)
		}
		if welcome.Message != "Connected to Browser Automation Studio" {
			t.Errorf("expected welcome message, got %s", welcome.Message)
		}
	})
}

// TestHubHandlesMultipleBroadcasts verifies hub can handle rapid sequential broadcasts
func TestHubHandlesMultipleBroadcasts(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] handles multiple rapid broadcasts", func(t *testing.T) {
		hub := newTestHub(t)

		client := &Client{
			ID:   uuid.New(),
			Send: make(chan ExecutionUpdate, 10),
			Hub:  hub,
		}

		hub.register <- client
		_ = waitForUpdate(t, client.Send) // Drain welcome message

		executionID := uuid.New()

		// Send multiple updates rapidly
		for i := 0; i < 5; i++ {
			hub.BroadcastUpdate(ExecutionUpdate{
				Type:        "progress",
				ExecutionID: executionID,
				Progress:    i * 20,
			})
		}

		// Verify all updates were received
		receivedProgress := make([]int, 0, 5)
		for i := 0; i < 5; i++ {
			update := waitForUpdate(t, client.Send)
			receivedProgress = append(receivedProgress, update.Progress)
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
			Send: make(chan ExecutionUpdate, 4),
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
