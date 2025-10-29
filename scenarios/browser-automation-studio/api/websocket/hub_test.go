package websocket

import (
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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

func TestHubBroadcastsToRegisteredClients(t *testing.T) {
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
}

func TestHubFiltersByExecutionID(t *testing.T) {
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
}

func TestGetClientCount(t *testing.T) {
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
}
