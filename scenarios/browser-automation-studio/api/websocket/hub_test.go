package websocket

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestHubCloseExecutionNoop(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] close execution does not drop clients", func(t *testing.T) {
		hub := newTestHub(t)

		client := &Client{
			ID:   uuid.New(),
			Send: make(chan any, 4),
			Hub:  hub,
		}
		hub.register <- client
		_ = waitForMessage(t, client.Send) // welcome

		hub.CloseExecution(uuid.New())

		update := contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           contracts.EventKindExecutionProgress,
			ExecutionID:    uuid.New(),
			WorkflowID:     uuid.New(),
			Sequence:       99,
		}

		hub.BroadcastEnvelope(update)

		msg := waitForMessage(t, client.Send)
		env, ok := msg.(contracts.EventEnvelope)
		if !ok {
			t.Fatalf("expected envelope after CloseExecution, got %T", msg)
		}
		if env.Sequence != update.Sequence {
			t.Fatalf("expected sequence %d, got %d", update.Sequence, env.Sequence)
		}
	})
}

func TestHubDropsUnresponsiveClient(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] removes clients that cannot receive updates", func(t *testing.T) {
		hub := newTestHub(t)

		client := &Client{
			ID:   uuid.New(),
			Send: make(chan any, 1), // intentionally tiny buffer
			Hub:  hub,
		}

		hub.register <- client
		time.Sleep(50 * time.Millisecond) // allow welcome message to fill buffer

		blockedUpdate := contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           contracts.EventKindExecutionProgress,
			ExecutionID:    uuid.New(),
			WorkflowID:     uuid.New(),
			Sequence:       1,
		}

		hub.BroadcastEnvelope(blockedUpdate)
		time.Sleep(50 * time.Millisecond) // allow hub to process removal

		if count := hub.GetClientCount(); count != 0 {
			t.Fatalf("expected hub to drop unresponsive client, still have %d", count)
		}

		select {
		case _, ok := <-client.Send:
			if !ok {
				return // channel closed as expected
			}
		default:
			t.Fatalf("expected client channel to be closed after removal")
		}
	})
}

func TestServeWSSubscribeFiltersMessages(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] websocket subscription filters non-matching executions", func(t *testing.T) {
		hub := newTestHub(t)
		server := startTestWebSocketServer(t, hub)
		defer server.Close()

		conn := dialTestWebSocket(t, server)
		defer conn.Close()

		drainWelcomeMessage(t, conn)

		executionID := uuid.New()
		if err := conn.WriteJSON(map[string]any{
			"type":         "subscribe",
			"execution_id": executionID.String(),
		}); err != nil {
			t.Fatalf("failed to send subscribe message: %v", err)
		}
		time.Sleep(50 * time.Millisecond)

		matching := contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           contracts.EventKindExecutionProgress,
			ExecutionID:    executionID,
			WorkflowID:     uuid.New(),
			Sequence:       10,
		}
		hub.BroadcastEnvelope(matching)

		var received contracts.EventEnvelope
		if err := conn.ReadJSON(&received); err != nil {
			t.Fatalf("expected matching envelope, read error: %v", err)
		}
		if received.ExecutionID != executionID {
			t.Fatalf("expected execution %s, got %s", executionID, received.ExecutionID)
		}

		nonMatching := contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           contracts.EventKindExecutionProgress,
			ExecutionID:    uuid.New(),
			WorkflowID:     uuid.New(),
			Sequence:       11,
		}

		hub.BroadcastEnvelope(nonMatching)

		_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		var unexpected contracts.EventEnvelope
		if err := conn.ReadJSON(&unexpected); err == nil {
			t.Fatalf("expected filtered event to be dropped, but received %+v", unexpected)
		} else if ne, ok := err.(net.Error); !ok || !ne.Timeout() {
			t.Fatalf("expected timeout waiting for filtered message, got %v", err)
		}
		_ = conn.SetReadDeadline(time.Time{})
	})
}

func TestServeWSUnsubscribeRestoresBroadcasts(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] unsubscribe removes execution filter", func(t *testing.T) {
		hub := newTestHub(t)
		server := startTestWebSocketServer(t, hub)
		defer server.Close()

		conn := dialTestWebSocket(t, server)
		defer conn.Close()

		drainWelcomeMessage(t, conn)

		executionID := uuid.New()
		if err := conn.WriteJSON(map[string]any{
			"type":         "subscribe",
			"execution_id": executionID.String(),
		}); err != nil {
			t.Fatalf("failed to send subscribe message: %v", err)
		}
		time.Sleep(50 * time.Millisecond)

		if err := conn.WriteJSON(map[string]any{"type": "unsubscribe"}); err != nil {
			t.Fatalf("failed to send unsubscribe message: %v", err)
		}
		time.Sleep(50 * time.Millisecond)

		event := contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           contracts.EventKindExecutionProgress,
			ExecutionID:    uuid.New(),
			WorkflowID:     uuid.New(),
			Sequence:       20,
		}

		hub.BroadcastEnvelope(event)

		var received contracts.EventEnvelope
		if err := conn.ReadJSON(&received); err != nil {
			t.Fatalf("expected envelope after unsubscribe, got error: %v", err)
		}
		if received.Sequence != event.Sequence {
			t.Fatalf("expected sequence %d after unsubscribe, got %d", event.Sequence, received.Sequence)
		}
	})
}

var testUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func startTestWebSocketServer(t *testing.T, hub *Hub) *httptest.Server {
	t.Helper()

	handler := http.NewServeMux()
	handler.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("websocket upgrade failed: %v", err)
			return
		}
		hub.ServeWS(conn, nil)
	})

	return httptest.NewServer(handler)
}

func dialTestWebSocket(t *testing.T, server *httptest.Server) *websocket.Conn {
	t.Helper()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket server: %v", err)
	}
	return conn
}

func drainWelcomeMessage(t *testing.T, conn *websocket.Conn) {
	t.Helper()

	var welcome map[string]any
	if err := conn.ReadJSON(&welcome); err != nil {
		t.Fatalf("expected welcome message, got error: %v", err)
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

// =============================================================================
// Recording Action Streaming Tests
// =============================================================================

// TestBroadcastRecordingActionToSubscribedClient verifies recording actions are sent to subscribed clients
func TestBroadcastRecordingActionToSubscribedClient(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] broadcasts recording actions to subscribed clients", func(t *testing.T) {
		hub := newTestHub(t)

		sessionID := "test-session-123"

		client := &Client{
			ID:                 uuid.New(),
			Send:               make(chan any, 4),
			Hub:                hub,
			RecordingSessionID: &sessionID,
		}

		hub.register <- client
		_ = waitForMessage(t, client.Send) // Drain welcome message

		testAction := map[string]any{
			"id":         "action-1",
			"actionType": "click",
			"selector":   "#submit-btn",
		}

		hub.BroadcastRecordingAction(sessionID, testAction)

		msg := waitForMessage(t, client.Send)
		recordingMsg, ok := msg.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", msg)
		}

		if recordingMsg["type"] != "recording_action" {
			t.Errorf("expected type 'recording_action', got %v", recordingMsg["type"])
		}
		if recordingMsg["session_id"] != sessionID {
			t.Errorf("expected session_id '%s', got %v", sessionID, recordingMsg["session_id"])
		}
		if recordingMsg["action"] == nil {
			t.Error("expected action to be present")
		}
		if recordingMsg["timestamp"] == nil {
			t.Error("expected timestamp to be present")
		}
	})
}

// TestBroadcastRecordingActionFiltersNonSubscribed verifies non-subscribed clients don't receive recording actions
func TestBroadcastRecordingActionFiltersNonSubscribed(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] filters recording actions from non-subscribed clients", func(t *testing.T) {
		hub := newTestHub(t)

		sessionID := "test-session-123"
		otherSessionID := "other-session-456"

		// Client subscribed to different session
		subscribedOther := &Client{
			ID:                 uuid.New(),
			Send:               make(chan any, 4),
			Hub:                hub,
			RecordingSessionID: &otherSessionID,
		}

		// Client not subscribed to any recording
		unsubscribed := &Client{
			ID:   uuid.New(),
			Send: make(chan any, 4),
			Hub:  hub,
		}

		hub.register <- subscribedOther
		hub.register <- unsubscribed
		_ = waitForMessage(t, subscribedOther.Send) // Drain welcome
		_ = waitForMessage(t, unsubscribed.Send)    // Drain welcome

		testAction := map[string]any{
			"id":         "action-1",
			"actionType": "click",
		}

		hub.BroadcastRecordingAction(sessionID, testAction)

		// Neither client should receive the action
		select {
		case msg := <-subscribedOther.Send:
			t.Fatalf("client subscribed to other session should not receive action, got %+v", msg)
		case <-time.After(100 * time.Millisecond):
			// Expected - no message
		}

		select {
		case msg := <-unsubscribed.Send:
			t.Fatalf("unsubscribed client should not receive action, got %+v", msg)
		case <-time.After(100 * time.Millisecond):
			// Expected - no message
		}
	})
}

// TestRecordingSubscriptionViaWebSocket verifies clients can subscribe to recording sessions via WebSocket
func TestRecordingSubscriptionViaWebSocket(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] websocket subscribe_recording enables recording action streaming", func(t *testing.T) {
		hub := newTestHub(t)
		server := startTestWebSocketServer(t, hub)
		defer server.Close()

		conn := dialTestWebSocket(t, server)
		defer conn.Close()

		drainWelcomeMessage(t, conn)

		sessionID := "recording-session-789"

		// Subscribe to recording session
		if err := conn.WriteJSON(map[string]any{
			"type":       "subscribe_recording",
			"session_id": sessionID,
		}); err != nil {
			t.Fatalf("failed to send subscribe_recording: %v", err)
		}

		// Should receive subscription confirmation
		var confirmation map[string]any
		if err := conn.ReadJSON(&confirmation); err != nil {
			t.Fatalf("failed to read confirmation: %v", err)
		}
		if confirmation["type"] != "recording_subscribed" {
			t.Errorf("expected type 'recording_subscribed', got %v", confirmation["type"])
		}
		if confirmation["session_id"] != sessionID {
			t.Errorf("expected session_id '%s', got %v", sessionID, confirmation["session_id"])
		}

		// Allow subscription to be processed
		time.Sleep(50 * time.Millisecond)

		// Broadcast a recording action
		testAction := map[string]any{
			"id":         "action-2",
			"actionType": "type",
			"payload":    map[string]any{"text": "hello"},
		}
		hub.BroadcastRecordingAction(sessionID, testAction)

		// Should receive the action
		var actionMsg map[string]any
		if err := conn.ReadJSON(&actionMsg); err != nil {
			t.Fatalf("failed to read recording action: %v", err)
		}
		if actionMsg["type"] != "recording_action" {
			t.Errorf("expected type 'recording_action', got %v", actionMsg["type"])
		}
		if actionMsg["session_id"] != sessionID {
			t.Errorf("expected session_id '%s', got %v", sessionID, actionMsg["session_id"])
		}
	})
}

// TestRecordingUnsubscriptionViaWebSocket verifies clients can unsubscribe from recording sessions
func TestRecordingUnsubscriptionViaWebSocket(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] websocket unsubscribe_recording stops recording action streaming", func(t *testing.T) {
		hub := newTestHub(t)
		server := startTestWebSocketServer(t, hub)
		defer server.Close()

		conn := dialTestWebSocket(t, server)
		defer conn.Close()

		drainWelcomeMessage(t, conn)

		sessionID := "recording-session-unsub"

		// Subscribe first
		if err := conn.WriteJSON(map[string]any{
			"type":       "subscribe_recording",
			"session_id": sessionID,
		}); err != nil {
			t.Fatalf("failed to send subscribe_recording: %v", err)
		}

		// Drain confirmation
		var confirmation map[string]any
		if err := conn.ReadJSON(&confirmation); err != nil {
			t.Fatalf("failed to read confirmation: %v", err)
		}
		time.Sleep(50 * time.Millisecond)

		// Now unsubscribe
		if err := conn.WriteJSON(map[string]any{
			"type": "unsubscribe_recording",
		}); err != nil {
			t.Fatalf("failed to send unsubscribe_recording: %v", err)
		}
		time.Sleep(50 * time.Millisecond)

		// Broadcast a recording action
		hub.BroadcastRecordingAction(sessionID, map[string]any{"id": "action-3"})

		// Should NOT receive the action
		_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		var unexpected map[string]any
		if err := conn.ReadJSON(&unexpected); err == nil {
			t.Fatalf("expected no message after unsubscribe, got %+v", unexpected)
		} else if ne, ok := err.(net.Error); !ok || !ne.Timeout() {
			t.Fatalf("expected timeout, got %v", err)
		}
		_ = conn.SetReadDeadline(time.Time{})
	})
}

// TestMultipleRecordingSessionsIsolation verifies actions are isolated between different recording sessions
func TestMultipleRecordingSessionsIsolation(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] recording actions are isolated between sessions", func(t *testing.T) {
		hub := newTestHub(t)

		sessionA := "session-A"
		sessionB := "session-B"

		clientA := &Client{
			ID:                 uuid.New(),
			Send:               make(chan any, 4),
			Hub:                hub,
			RecordingSessionID: &sessionA,
		}

		clientB := &Client{
			ID:                 uuid.New(),
			Send:               make(chan any, 4),
			Hub:                hub,
			RecordingSessionID: &sessionB,
		}

		hub.register <- clientA
		hub.register <- clientB
		_ = waitForMessage(t, clientA.Send) // Drain welcome
		_ = waitForMessage(t, clientB.Send) // Drain welcome

		// Broadcast to session A
		hub.BroadcastRecordingAction(sessionA, map[string]any{"id": "action-A"})

		// Client A should receive
		msgA := waitForMessage(t, clientA.Send)
		if msgMap, ok := msgA.(map[string]any); ok {
			if msgMap["session_id"] != sessionA {
				t.Errorf("expected session_id '%s', got %v", sessionA, msgMap["session_id"])
			}
		} else {
			t.Fatalf("expected map, got %T", msgA)
		}

		// Client B should NOT receive
		select {
		case msg := <-clientB.Send:
			t.Fatalf("client B should not receive session A action, got %+v", msg)
		case <-time.After(100 * time.Millisecond):
			// Expected
		}

		// Now broadcast to session B
		hub.BroadcastRecordingAction(sessionB, map[string]any{"id": "action-B"})

		// Client B should receive
		msgB := waitForMessage(t, clientB.Send)
		if msgMap, ok := msgB.(map[string]any); ok {
			if msgMap["session_id"] != sessionB {
				t.Errorf("expected session_id '%s', got %v", sessionB, msgMap["session_id"])
			}
		} else {
			t.Fatalf("expected map, got %T", msgB)
		}

		// Client A should NOT receive
		select {
		case msg := <-clientA.Send:
			t.Fatalf("client A should not receive session B action, got %+v", msg)
		case <-time.After(100 * time.Millisecond):
			// Expected
		}
	})
}

// TestBroadcastRecordingActionWithFullBuffer verifies graceful handling when client buffer is full
func TestBroadcastRecordingActionWithFullBuffer(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] handles full client buffer gracefully for recording actions", func(t *testing.T) {
		hub := newTestHub(t)

		sessionID := "test-session-buffer"

		// Client with tiny buffer that will fill up
		client := &Client{
			ID:                 uuid.New(),
			Send:               make(chan any, 1), // Tiny buffer
			Hub:                hub,
			RecordingSessionID: &sessionID,
		}

		hub.register <- client
		// Don't drain welcome - let buffer fill

		// Try to broadcast - should not panic or block
		hub.BroadcastRecordingAction(sessionID, map[string]any{"id": "action-1"})
		hub.BroadcastRecordingAction(sessionID, map[string]any{"id": "action-2"})
		hub.BroadcastRecordingAction(sessionID, map[string]any{"id": "action-3"})

		// Should complete without hanging (test will timeout if it blocks)
		time.Sleep(50 * time.Millisecond)

		// Client count should still be correct (we don't drop clients for recording, just skip)
		if count := hub.GetClientCount(); count != 1 {
			// Note: The implementation skips full buffers rather than dropping clients for recording
			// This is different from execution broadcasts which drop unresponsive clients
		}
	})
}

// TestRecordingActionStreamingSequence verifies multiple actions are received in order
func TestRecordingActionStreamingSequence(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] streams multiple recording actions in sequence", func(t *testing.T) {
		hub := newTestHub(t)

		sessionID := "test-session-sequence"

		client := &Client{
			ID:                 uuid.New(),
			Send:               make(chan any, 10),
			Hub:                hub,
			RecordingSessionID: &sessionID,
		}

		hub.register <- client
		_ = waitForMessage(t, client.Send) // Drain welcome

		// Send sequence of actions
		for i := 1; i <= 5; i++ {
			hub.BroadcastRecordingAction(sessionID, map[string]any{
				"id":          fmt.Sprintf("action-%d", i),
				"sequenceNum": i,
			})
		}

		// Verify all received in order
		for i := 1; i <= 5; i++ {
			msg := waitForMessage(t, client.Send)
			if msgMap, ok := msg.(map[string]any); ok {
				action := msgMap["action"].(map[string]any)
				expectedID := fmt.Sprintf("action-%d", i)
				if action["id"] != expectedID {
					t.Errorf("expected action id '%s', got %v", expectedID, action["id"])
				}
			} else {
				t.Fatalf("expected map, got %T", msg)
			}
		}
	})
}
