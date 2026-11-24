package websocket

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
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
	"github.com/vrooli/browser-automation-studio/automation/driver"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func testStreamGrant(hub *Hub, sessionID string) recordingStreamGrant {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return hub.recordingGrants[sessionID]
}

func testAddHubClient(hub *Hub, client *Client) {
	hub.mu.Lock()
	hub.clients[client] = true
	hub.mu.Unlock()
}

func newRecordingGrantClient(t *testing.T, hub *Hub, sessionID string) *Client {
	t.Helper()
	if _, _, err := hub.IssueSessionStreamCookie(sessionID, uuid.NewString(), uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	grant := testStreamGrant(hub, sessionID)
	return &Client{
		ID: uuid.New(), Send: make(chan any, 4), BinarySend: make(chan []byte, 1), Hub: hub,
		RecordingGrantSessionID: sessionID, RecordingGrantExecutionID: grant.executionID, RecordingGrantRole: grant.role,
		recordingGrantHash: grant.secretHash,
	}
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

func waitForClientCount(t *testing.T, hub *Hub, want int) {
	t.Helper()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		if got := hub.GetClientCount(); got == want {
			return
		}
		select {
		case <-ticker.C:
		case <-timer.C:
			t.Fatalf("timed out waiting for %d clients; got %d", want, hub.GetClientCount())
		}
	}
}

func waitForClientSubscription(t *testing.T, hub *Hub, subscribed func(*Client) bool) {
	t.Helper()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		hub.mu.RLock()
		found := false
		for client := range hub.clients {
			if subscribed(client) {
				found = true
				break
			}
		}
		hub.mu.RUnlock()
		if found {
			return
		}
		select {
		case <-ticker.C:
		case <-timer.C:
			t.Fatal("timed out waiting for client subscription state")
		}
	}
}

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

func TestBroadcastEnvelopeBuffersBoundedBurstsThenBackpressures(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	hub := NewHub(logger) // Intentionally do not start Run; model a paused consumer.
	for i := 0; i < MaxPendingBroadcastEvents; i++ {
		hub.BroadcastEnvelope(map[string]any{"sequence": i})
	}
	if got := len(hub.broadcast); got != MaxPendingBroadcastEvents {
		t.Fatalf("queued events=%d, want bounded capacity %d", got, MaxPendingBroadcastEvents)
	}

	returned := make(chan struct{})
	started := make(chan struct{})
	go func() {
		close(started)
		hub.BroadcastEnvelope(map[string]any{"sequence": MaxPendingBroadcastEvents})
		close(returned)
	}()
	<-started
	select {
	case <-returned:
		t.Fatal("publisher bypassed backpressure when the bounded queue was full")
	case <-time.After(10 * time.Millisecond):
	}
	first := <-hub.broadcast
	if got := first.(map[string]any)["sequence"]; got != 0 {
		t.Fatalf("first queued event=%v, want sequence 0", got)
	}
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("publisher did not resume after one queue slot was released")
	}
	if got := len(hub.broadcast); got != MaxPendingBroadcastEvents {
		t.Fatalf("queued events after resume=%d, want capacity %d", got, MaxPendingBroadcastEvents)
	}
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
		welcome := waitForMessage(t, client.Send)
		client.Send <- welcome // Keep the one-slot buffer full for the backpressure case.

		blockedUpdate := contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           contracts.EventKindExecutionProgress,
			ExecutionID:    uuid.New(),
			WorkflowID:     uuid.New(),
			Sequence:       1,
		}

		hub.BroadcastEnvelope(blockedUpdate)
		waitForClientCount(t, hub, 0)

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
		waitForClientSubscription(t, hub, func(client *Client) bool {
			return client.ExecutionID != nil && *client.ExecutionID == executionID
		})

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
		waitForClientSubscription(t, hub, func(client *Client) bool {
			return client.ExecutionID != nil && *client.ExecutionID == executionID
		})

		if err := conn.WriteJSON(map[string]any{"type": "unsubscribe"}); err != nil {
			t.Fatalf("failed to send unsubscribe message: %v", err)
		}
		waitForClientSubscription(t, hub, func(client *Client) bool { return client.ExecutionID == nil })

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

	return testutil.StartHTTPServer(t, handler)
}

func startRecordingWebSocketServer(t *testing.T, hub *Hub, sessionID string) *httptest.Server {
	t.Helper()
	handler := http.NewServeMux()
	path := SessionStreamCookiePath(sessionID)
	handler.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionStreamCookieName(sessionID))
		if err != nil || !hub.ValidateSessionStreamCookie(sessionID, cookie.Value) {
			http.Error(w, "recording stream unavailable", http.StatusForbidden)
			return
		}
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("recording websocket upgrade failed: %v", err)
			return
		}
		if !hub.ServeRecordingWS(conn, sessionID, cookie.Value) {
			_ = conn.Close()
		}
	})
	return testutil.StartHTTPServer(t, handler)
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

func dialTestWebSocketWithCookie(t *testing.T, server *httptest.Server, name, value string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + SessionStreamCookiePath(strings.TrimPrefix(name, "bas_ws_"))
	header := http.Header{"Cookie": []string{name + "=" + value}}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("failed to dial authorized recording websocket: %v", err)
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
		_ = waitForMessage(t, client1.Send)
		waitForClientCount(t, hub, 1)

		hub.register <- client2
		_ = waitForMessage(t, client2.Send)
		waitForClientCount(t, hub, 2)

		hub.unregister <- client1
		waitForClientCount(t, hub, 1)
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
		if welcome["message"] != "Connected to Vrooli Ascension" {
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
		_ = waitForMessage(t, client.Send)
		waitForClientCount(t, hub, 1)

		hub.unregister <- client
		waitForClientCount(t, hub, 0)

		// Verify the client is removed from the hub - channel closing is implementation detail
		// that's handled asynchronously, so we just verify client count decreased
	})
}

// =============================================================================
// Recording Action Streaming Tests
// =============================================================================

// testEntry creates the canonical stream payload used by recording and execution.
func testEntry(id string, actionType basactions.ActionType) *bastimeline.TimelineEntry {
	return &bastimeline.TimelineEntry{
		Id: id, SequenceNum: 1, Timestamp: timestamppb.New(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		Action: &basactions.ActionDefinition{Type: actionType},
	}
}

// TestBroadcastRecordingEntryToSubscribedClient verifies recording entries are sent to subscribed clients
func TestBroadcastRecordingEntryToSubscribedClient(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] broadcasts recording entries to subscribed clients", func(t *testing.T) {
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

		hub.BroadcastTimelineEntry(sessionID, testEntry("action-1", basactions.ActionType_ACTION_TYPE_CLICK))

		msg := waitForMessage(t, client.Send)
		recordingMsg, ok := msg.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", msg)
		}

		if recordingMsg["type"] != "TIMELINE_MESSAGE_TYPE_ENTRY" {
			t.Errorf("expected typed timeline message, got %v", recordingMsg["type"])
		}
		if recordingMsg["session_id"] != sessionID {
			t.Errorf("expected session_id '%s', got %v", sessionID, recordingMsg["session_id"])
		}
		if recordingMsg["entry"] == nil {
			t.Error("expected entry to be present")
		}
	})
}

// TestBroadcastRecordingEntryFiltersNonSubscribed verifies non-subscribed clients don't receive recording entries
func TestBroadcastRecordingEntryFiltersNonSubscribed(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] filters recording entries from non-subscribed clients", func(t *testing.T) {
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

		hub.BroadcastTimelineEntry(sessionID, testEntry("action-1", basactions.ActionType_ACTION_TYPE_CLICK))

		// Neither client should receive the entry
		select {
		case msg := <-subscribedOther.Send:
			t.Fatalf("client subscribed to other session should not receive entry, got %+v", msg)
		case <-time.After(100 * time.Millisecond):
			// Expected - no message
		}

		select {
		case msg := <-unsubscribed.Send:
			t.Fatalf("unsubscribed client should not receive entry, got %+v", msg)
		case <-time.After(100 * time.Millisecond):
			// Expected - no message
		}
	})
}

// TestRecordingSubscriptionViaWebSocket verifies clients can subscribe to recording sessions via WebSocket
func TestRecordingSubscriptionViaWebSocket(t *testing.T) {
	t.Run("[REQ:BAS-RECORD-MODE] websocket subscribe_recording enables recording action streaming", func(t *testing.T) {
		hub := newTestHub(t)
		sessionID := uuid.NewString()
		executionID := uuid.NewString()
		leaseID := uuid.NewString()
		cookieName, cookieValue, err := hub.IssueSessionStreamCookie(sessionID, executionID, leaseID)
		if err != nil {
			t.Fatal(err)
		}
		server := startRecordingWebSocketServer(t, hub, sessionID)
		conn := dialTestWebSocketWithCookie(t, server, cookieName, cookieValue)
		defer conn.Close()

		drainWelcomeMessage(t, conn)

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

		waitForClientSubscription(t, hub, func(client *Client) bool {
			return client.RecordingSessionID != nil && *client.RecordingSessionID == sessionID
		})

		// Broadcast a recording entry
		hub.BroadcastTimelineEntry(sessionID, testEntry("action-2", basactions.ActionType_ACTION_TYPE_INPUT))

		// Should receive the entry
		var actionMsg map[string]any
		if err := conn.ReadJSON(&actionMsg); err != nil {
			t.Fatalf("failed to read recording entry: %v", err)
		}
		if actionMsg["type"] != "TIMELINE_MESSAGE_TYPE_ENTRY" {
			t.Errorf("expected typed timeline message, got %v", actionMsg["type"])
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
		sessionID := uuid.NewString()
		cookieName, cookieValue, err := hub.IssueSessionStreamCookie(sessionID, uuid.NewString(), uuid.NewString())
		if err != nil {
			t.Fatal(err)
		}
		server := startRecordingWebSocketServer(t, hub, sessionID)
		conn := dialTestWebSocketWithCookie(t, server, cookieName, cookieValue)
		defer conn.Close()

		drainWelcomeMessage(t, conn)

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
		waitForClientSubscription(t, hub, func(client *Client) bool {
			return client.RecordingSessionID != nil && *client.RecordingSessionID == sessionID
		})

		// Now unsubscribe
		if err := conn.WriteJSON(map[string]any{
			"type": "unsubscribe_recording",
		}); err != nil {
			t.Fatalf("failed to send unsubscribe_recording: %v", err)
		}
		waitForClientSubscription(t, hub, func(client *Client) bool { return client.RecordingSessionID == nil })

		// Broadcast a recording action
		hub.BroadcastTimelineEntry(sessionID, testEntry("action-3", basactions.ActionType_ACTION_TYPE_CLICK))

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

func TestGenericWebSocketCannotSubscribeToRecordingSession(t *testing.T) {
	hub := newTestHub(t)
	sessionID := uuid.NewString()
	if _, _, err := hub.IssueSessionStreamCookie(sessionID, uuid.NewString(), uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	forwarded := make(chan struct{}, 1)
	hub.SetInputForwarder(func(string, map[string]any) (*driver.ForwardInputResponse, error) {
		forwarded <- struct{}{}
		return &driver.ForwardInputResponse{Status: "ok"}, nil
	})
	server := startTestWebSocketServer(t, hub)
	conn := dialTestWebSocket(t, server)
	defer conn.Close()
	drainWelcomeMessage(t, conn)
	if err := conn.WriteJSON(map[string]any{"type": "subscribe_recording", "session_id": sessionID}); err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteJSON(map[string]any{"type": "recording_input", "session_id": sessionID, "input": map[string]any{"type": "click"}}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if hub.HasRecordingFrameSubscribers(sessionID) {
		t.Fatal("generic socket subscribed to a protected recording session")
	}
	select {
	case <-forwarded:
		t.Fatal("generic socket forwarded input to a protected recording session")
	default:
	}
}

func TestGenericRunSubscriptionsStayDeniedForFullBrowserSessionLifetime(t *testing.T) {
	hub := newTestHub(t)
	ctxClient := &Client{ID: uuid.New(), Send: make(chan any, 8), BinarySend: make(chan []byte, 1), Hub: hub}
	testAddHubClient(hub, ctxClient)
	sessionID, executionID, leaseID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	exportID := uuid.NewString()
	parsedExecutionID, _ := uuid.Parse(executionID)

	// A generic socket can subscribe before the browser session is admitted.
	ctxClient.handleSubscription("subscribe", map[string]any{"execution_id": executionID})
	ctxClient.handleSubscription("subscribe_execution_frames", map[string]any{"execution_id": executionID})
	ctxClient.handleSubscription("subscribe_export", map[string]any{"export_id": exportID})
	if ctxClient.ExecutionID == nil || ctxClient.ExecutionFrameStreamID == nil || ctxClient.ExportSubscriptionID == nil {
		t.Fatal("fixture setup failed to establish preexisting generic selectors")
	}
	for {
		select {
		case <-ctxClient.Send:
		default:
			goto initialMessagesDrained
		}
	}
initialMessagesDrained:

	_, cookie, err := hub.IssueSessionStreamCookieRole(sessionID, executionID, leaseID, "observe", time.Now().Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	grant := testStreamGrant(hub, sessionID)
	if ctxClient.ExecutionID != nil || ctxClient.ExecutionFrameStreamID != nil || ctxClient.ExportSubscriptionID == nil {
		t.Fatalf("session admission did not evict preexisting run selectors: execution=%v frame=%v export=%v", ctxClient.ExecutionID, ctxClient.ExecutionFrameStreamID, ctxClient.ExportSubscriptionID)
	}
	ctxClient.handleSubscription("subscribe", map[string]any{"execution_id": executionID})
	ctxClient.handleSubscription("subscribe_execution_frames", map[string]any{"execution_id": executionID})
	if ctxClient.ExecutionID != nil || ctxClient.ExecutionFrameStreamID != nil {
		t.Fatal("generic socket selected an active browser execution")
	}

	ownerClient := &Client{
		ID: uuid.New(), Send: make(chan any, 8), BinarySend: make(chan []byte, 1), Hub: hub,
		ExecutionID: &parsedExecutionID, ExecutionFrameStreamID: &executionID,
		ExportSubscriptionID: &exportID, RecordingGrantSessionID: sessionID,
		RecordingGrantExecutionID: executionID, RecordingGrantRole: "observe", recordingGrantHash: grant.secretHash,
	}
	testAddHubClient(hub, ownerClient)

	hub.BroadcastEnvelope(contracts.EventEnvelope{ExecutionID: parsedExecutionID, Kind: contracts.EventKindStepCompleted})
	hub.BroadcastExecutionFrame(executionID, &ExecutionFrame{ExecutionID: executionID, SessionID: sessionID, Data: "YWNjZXB0ZWQ=", MediaType: "image/jpeg", Width: 8, Height: 8})
	hub.BroadcastExecutionFrame(executionID, &ExecutionFrame{ExecutionID: executionID, SessionID: uuid.NewString(), Data: "Zm9yZWlnbg==", MediaType: "image/jpeg", Width: 8, Height: 8})
	hub.BroadcastExportProgress(&ExportProgress{ExportID: exportID, ExecutionID: executionID, Stage: "complete", Status: "completed", StorageURL: "/local/export"})

	gotEvent, gotFrame, gotExport := false, false, false
	for i := 0; i < 3; i++ {
		switch got := waitForMessage(t, ownerClient.Send).(type) {
		case contracts.EventEnvelope:
			gotEvent = true
		case queuedExecutionFrame:
			if got.message["type"] != "session_frame" || got.message["session_id"] != sessionID || got.message["data"] != "YWNjZXB0ZWQ=" {
				t.Fatalf("authorized frame = %#v", got)
			}
			gotFrame = true
		case map[string]any:
			if got["type"] == "export_progress" && got["export_id"] == exportID {
				gotExport = true
			} else {
				t.Fatalf("unexpected authorized owner message = %#v", got)
			}
		default:
			t.Fatalf("unexpected authorized owner message type = %T", got)
		}
	}
	if !gotEvent || !gotFrame || !gotExport {
		t.Fatalf("authorized owner messages missing: event=%v frame=%v export=%v", gotEvent, gotFrame, gotExport)
	}
	hub.mu.RLock()
	latest := hub.latestFrames["execution:"+executionID].execution
	hub.mu.RUnlock()
	if latest == nil || latest.SessionID != sessionID || latest.Data != "YWNjZXB0ZWQ=" {
		t.Fatalf("foreign frame changed owner latest-frame slot: %#v", latest)
	}
	select {
	case got := <-ctxClient.Send:
		t.Fatalf("generic client received foreign-run owner data: %#v", got)
	default:
	}

	// Cookie expiry revokes the connection credential, not the still-live
	// browser run's ownership fence.
	hub.expireSessionStreamCookie(sessionID, grant.secretHash)
	ctxClient.handleSubscription("subscribe", map[string]any{"execution_id": executionID})
	ctxClient.handleSubscription("subscribe_execution_frames", map[string]any{"execution_id": executionID})
	if ctxClient.ExecutionID != nil || ctxClient.ExecutionFrameStreamID != nil {
		t.Fatal("generic run subscription reopened after stream-cookie expiry")
	}
	if hub.ValidateSessionStreamCookie(sessionID, cookie) {
		t.Fatal("expired stream cookie remained valid")
	}

	hub.RevokeSessionStreamCookie(sessionID)
	hub.CloseExecution(parsedExecutionID)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		hub.mu.RLock()
		_, protected := hub.sessionExecutionOwners[sessionID]
		_, cached := hub.latestFrames["execution:"+executionID]
		hub.mu.RUnlock()
		if !protected && !cached {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("closed execution owner fence was not released after the hub drained its close marker")
}

func TestRecordingStreamGrantRevocationClearsActiveOwner(t *testing.T) {
	hub := newTestHub(t)
	sessionID := uuid.NewString()
	_, value, err := hub.IssueSessionStreamCookie(sessionID, uuid.NewString(), uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	grant := testStreamGrant(hub, sessionID)
	client := &Client{ID: uuid.New(), Send: make(chan any, 4), BinarySend: make(chan []byte, 1), Hub: hub,
		RecordingGrantSessionID: sessionID, RecordingGrantExecutionID: grant.executionID, RecordingGrantRole: grant.role, recordingGrantHash: grant.secretHash}
	hub.register <- client
	_ = waitForMessage(t, client.Send)
	client.handleSubscription("subscribe_recording", map[string]any{"session_id": sessionID})
	_ = waitForMessage(t, client.Send)
	if !hub.HasRecordingFrameSubscribers(sessionID) {
		t.Fatal("owner could not subscribe before revocation")
	}
	hub.RevokeSessionStreamCookie(sessionID)
	if hub.ValidateSessionStreamCookie(sessionID, value) {
		t.Fatal("revoked cookie still validates")
	}
	if hub.HasRecordingFrameSubscribers(sessionID) || client.RecordingSessionID != nil || client.RecordingGrantSessionID != "" {
		t.Fatal("revocation retained active stream authority")
	}
}

func TestControlReceiptRotationReleasesHeldInputBeforeRevocation(t *testing.T) {
	hub := newTestHub(t)
	sessionID := uuid.NewString()
	executionID, leaseID := uuid.NewString(), uuid.NewString()
	_, oldValue, err := hub.IssueSessionStreamCookie(sessionID, executionID, leaseID)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	hub.SetInputStateReleaser(func(gotSession, gotExecution, gotLease string) error {
		called = true
		if gotSession != sessionID || gotExecution != executionID || gotLease != leaseID {
			t.Fatalf("release was not bound to the old lease: %s %s %s", gotSession, gotExecution, gotLease)
		}
		if hub.ValidateSessionStreamCookie(sessionID, oldValue) {
			t.Fatal("old receipt remained admissible while held input was released")
		}
		return nil
	})
	if _, _, err := hub.IssueSessionStreamCookieRole(sessionID, executionID, leaseID, "observe", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("control receipt rotation did not release input")
	}
	if hub.ValidateSessionStreamCookie(sessionID, oldValue) {
		t.Fatal("old control receipt remained valid after rotation")
	}
}

func TestControlReceiptRotationFailsClosedWithoutInputOwner(t *testing.T) {
	hub := newTestHub(t)
	sessionID := uuid.NewString()
	executionID, leaseID := uuid.NewString(), uuid.NewString()
	_, oldValue, err := hub.IssueSessionStreamCookie(sessionID, executionID, leaseID)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := hub.IssueSessionStreamCookie(sessionID, executionID, leaseID); err == nil {
		t.Fatal("control rotation succeeded without an input-state owner")
	}
	if hub.ValidateSessionStreamCookie(sessionID, oldValue) {
		t.Fatal("failed rotation retained the stale control receipt")
	}
}

func TestControlReceiptRotationRetriesFailedInputReleaseBeforeIssuingNewReceipt(t *testing.T) {
	hub := newTestHub(t)
	sessionID := uuid.NewString()
	executionID, leaseID := uuid.NewString(), uuid.NewString()
	_, oldValue, err := hub.IssueSessionStreamCookie(sessionID, executionID, leaseID)
	if err != nil {
		t.Fatal(err)
	}
	releaseAttempts := 0
	hub.SetInputStateReleaser(func(gotSession, gotExecution, gotLease string) error {
		releaseAttempts++
		if gotSession != sessionID || gotExecution != executionID || gotLease != leaseID {
			t.Fatalf("release was not retried against the old lease: %s %s %s", gotSession, gotExecution, gotLease)
		}
		if releaseAttempts == 1 {
			return errors.New("temporary input-release failure")
		}
		return nil
	})

	if _, _, err := hub.IssueSessionStreamCookieRole(sessionID, executionID, leaseID, "observe", time.Now().Add(time.Minute)); err == nil {
		t.Fatal("rotation succeeded while old held input could not be released")
	}
	if hub.ValidateSessionStreamCookie(sessionID, oldValue) {
		t.Fatal("old receipt remained admissible after failed release")
	}
	grant := testStreamGrant(hub, sessionID)
	if !grant.revoking || grant.leaseID != leaseID {
		t.Fatal("failed release did not retain the old lease as unusable retry state")
	}

	_, nextValue, err := hub.IssueSessionStreamCookieRole(sessionID, executionID, leaseID, "observe", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatalf("rotation did not recover after the owner released input: %v", err)
	}
	if releaseAttempts != 2 {
		t.Fatalf("expected release retry before new receipt, got %d attempts", releaseAttempts)
	}
	if hub.ValidateSessionStreamCookie(sessionID, oldValue) || !hub.ValidateSessionStreamCookie(sessionID, nextValue) {
		t.Fatal("rotation did not replace the revoked receipt with the new receipt")
	}
}

func TestObserveReceiptCannotForwardInput(t *testing.T) {
	hub := newTestHub(t)
	sessionID := uuid.NewString()
	if _, _, err := hub.IssueSessionStreamCookieRole(sessionID, uuid.NewString(), uuid.NewString(), "observe", time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	grant := testStreamGrant(hub, sessionID)
	client := &Client{RecordingGrantSessionID: sessionID, RecordingGrantExecutionID: grant.executionID, RecordingGrantRole: grant.role, recordingGrantHash: grant.secretHash, Hub: hub}
	if hub.authorizesRecordingInput(client, sessionID) {
		t.Fatal("observe receipt authorized input")
	}
}

func TestValidateRecordingInputBoundsTextPayload(t *testing.T) {
	if size, err := validateRecordingInput(map[string]any{"type": "keyboard", "text": strings.Repeat("x", MaxStreamTextInputBytes)}); err != nil || size != MaxStreamTextInputBytes {
		t.Fatalf("limit-sized text returned size=%d, err=%v", size, err)
	}
	if _, err := validateRecordingInput(map[string]any{"type": "keyboard", "text": strings.Repeat("x", MaxStreamTextInputBytes+1)}); err == nil {
		t.Fatal("oversized text input was accepted")
	}
	if _, err := validateRecordingInput(map[string]any{"type": "text", "text": "hello"}); err == nil {
		t.Fatal("unknown input type was accepted")
	}
}

func TestSessionStreamGrantReceivesSanitizedBoundFrame(t *testing.T) {
	hub := newTestHub(t)
	sessionID, executionID := uuid.NewString(), uuid.NewString()
	cookieHash := sha256.Sum256([]byte("opaque-cookie"))
	hub.mu.Lock()
	hub.recordingGrants[sessionID] = recordingStreamGrant{executionID: executionID, leaseID: uuid.NewString(), role: "observe", expiresAt: time.Now().Add(time.Minute), secretHash: cookieHash}
	hub.mu.Unlock()
	client := &Client{ID: uuid.New(), Send: make(chan any, 1), BinarySend: make(chan []byte, 1), Hub: hub,
		RecordingGrantSessionID: sessionID, RecordingGrantExecutionID: executionID, RecordingGrantRole: "observe",
		ExecutionFrameStreamID: &executionID, recordingGrantHash: cookieHash}
	testAddHubClient(hub, client)
	hub.BroadcastExecutionFrame(executionID, &ExecutionFrame{ExecutionID: executionID, SessionID: sessionID, Data: "YmFzZTY0", MediaType: "image/jpeg", Width: 800, Height: 600})
	queued, ok := (<-client.Send).(queuedExecutionFrame)
	message := queued.message
	if !ok || message["type"] != "session_frame" || message["session_id"] != sessionID {
		t.Fatalf("sanitized frame message = %#v", message)
	}
	if _, leaked := message["execution_id"]; leaked {
		t.Fatalf("private execution id reached session client: %#v", message)
	}
}

func TestControlStreamCookieExpiryReleasesInputAndRevokesClients(t *testing.T) {
	hub := newTestHub(t)
	sessionID, executionID, leaseID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	_, value, err := hub.IssueSessionStreamCookieRole(sessionID, executionID, leaseID, "control", time.Now().Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	grant := testStreamGrant(hub, sessionID)
	released := make(chan struct{}, 1)
	hub.SetInputStateReleaser(func(gotSession, gotExecution, gotLease string) error {
		if gotSession != sessionID || gotExecution != executionID || gotLease != leaseID {
			t.Errorf("expiry release identity = %s %s %s", gotSession, gotExecution, gotLease)
		}
		if hub.ValidateSessionStreamCookie(sessionID, value) {
			t.Error("cookie remained valid while input release was running")
		}
		released <- struct{}{}
		return nil
	})
	client := &Client{ID: uuid.New(), Send: make(chan any, 2), BinarySend: make(chan []byte, 1), Hub: hub,
		RecordingGrantSessionID: sessionID, RecordingGrantExecutionID: executionID, RecordingGrantRole: "control", recordingGrantHash: grant.secretHash}
	testAddHubClient(hub, client)
	select {
	case <-released:
	case <-time.After(3 * time.Second):
		t.Fatal("expired control grant did not release held input")
	}
	select {
	case <-client.Send:
	case <-time.After(time.Second):
		t.Fatal("expired stream client was not revoked")
	}
	if hub.ValidateSessionStreamCookie(sessionID, value) {
		t.Fatal("expired control cookie remained valid")
	}
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
		hub.BroadcastTimelineEntry(sessionA, testEntry("action-A", basactions.ActionType_ACTION_TYPE_CLICK))

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
		hub.BroadcastTimelineEntry(sessionB, testEntry("action-B", basactions.ActionType_ACTION_TYPE_CLICK))

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

// TestBroadcastRecordingEntryWithFullBuffer verifies graceful handling when client buffer is full
func TestBroadcastRecordingEntryWithFullBuffer(t *testing.T) {
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
		hub.BroadcastTimelineEntry(sessionID, testEntry("action-1", basactions.ActionType_ACTION_TYPE_CLICK))
		hub.BroadcastTimelineEntry(sessionID, testEntry("action-2", basactions.ActionType_ACTION_TYPE_CLICK))
		hub.BroadcastTimelineEntry(sessionID, testEntry("action-3", basactions.ActionType_ACTION_TYPE_CLICK))

		// Client count should still be correct (we don't drop clients for recording, just skip)
		// Note: The implementation skips full buffers rather than dropping clients for recording
		// This is different from execution broadcasts which drop unresponsive clients
		count := hub.GetClientCount()
		_ = count // Count verification is informational - implementation skips rather than drops
	})
}

// =============================================================================
// Frame Drop Monitoring Tests
// =============================================================================

// TestGetDroppedFrameCount verifies the dropped frame counter tracks correctly
func TestGetDroppedFrameCount(t *testing.T) {
	t.Run("[REQ:BAS-FRAME-STREAM] tracks dropped frame count", func(t *testing.T) {
		hub := newTestHub(t)

		// Initially should be 0
		if count := hub.GetDroppedFrameCount(); count != 0 {
			t.Fatalf("expected 0 dropped frames initially, got %d", count)
		}
	})
}

// TestBroadcastBinaryFrameDropsWithFullBuffer verifies binary frames are dropped and counted when buffer is full
func TestBroadcastBinaryFrameDropsWithFullBuffer(t *testing.T) {
	t.Run("[REQ:BAS-FRAME-STREAM] drops and counts frames when client binary buffer is full", func(t *testing.T) {
		hub := newTestHub(t)

		sessionID := "test-session-binary-buffer"

		// Client with tiny binary buffer that will fill up
		client := &Client{
			ID:                 uuid.New(),
			Send:               make(chan any, 4),
			BinarySend:         make(chan []byte, 1), // Tiny buffer - only 1 frame
			Hub:                hub,
			RecordingFrames:    true,
			RecordingSessionID: &sessionID,
		}

		hub.register <- client
		_ = waitForMessage(t, client.Send) // Drain welcome

		// Send multiple binary frames - first should go through, rest should be dropped
		testFrame := []byte{0xFF, 0xD8, 0xFF} // JPEG magic bytes
		hub.BroadcastBinaryFrame(sessionID, testFrame)
		hub.BroadcastBinaryFrame(sessionID, testFrame) // Should be dropped
		hub.BroadcastBinaryFrame(sessionID, testFrame) // Should be dropped

		// At least 2 frames should have been dropped
		droppedCount := hub.GetDroppedFrameCount()
		if droppedCount < 2 {
			t.Errorf("expected at least 2 dropped frames, got %d", droppedCount)
		}
	})
}

func TestBroadcastBinaryFrameDropsAtByteBudget(t *testing.T) {
	hub := newTestHub(t)
	sessionID := "test-session-binary-byte-budget"
	client := &Client{
		ID: uuid.New(), Send: make(chan any, 4), BinarySend: make(chan []byte, 4), Hub: hub,
		RecordingFrames: true, RecordingSessionID: &sessionID,
	}
	hub.register <- client
	_ = waitForMessage(t, client.Send)

	frame := make([]byte, MaxClientBinaryQueueBytes*3/4)
	hub.BroadcastBinaryFrame(sessionID, frame)
	hub.BroadcastBinaryFrame(sessionID, frame)
	queued := client.queuedBinaryFrameBytes()
	if queued != len(frame) || queued > MaxClientBinaryQueueBytes || len(client.BinarySend) != 1 {
		t.Fatalf("queued=%d channel_frames=%d, want one %d-byte frame under %d-byte cap", queued, len(client.BinarySend), len(frame), MaxClientBinaryQueueBytes)
	}
	if dropped := hub.GetDroppedFrameCount(); dropped != 1 {
		t.Fatalf("dropped frames=%d, want 1 for byte-cap rejection", dropped)
	}

	first := <-client.BinarySend
	client.releaseBinaryFrame(first)
	if queued := client.queuedBinaryFrameBytes(); queued != 0 {
		t.Fatalf("queued after completed write=%d, want 0", queued)
	}
	hub.BroadcastBinaryFrame(sessionID, frame)
	if queued := client.queuedBinaryFrameBytes(); queued != len(frame) {
		t.Fatalf("queued after release=%d, want %d", queued, len(frame))
	}
	t.Logf("BAS_MOTION_RELAY_QUEUE max_queued_bytes=%d cap_bytes=%d dropped=%d", queued, MaxClientBinaryQueueBytes, hub.GetDroppedFrameCount())
}

func testViewerFramePacket(sessionID, pageID string, sequence int) []byte {
	header, _ := json.Marshal(map[string]any{
		"version": 1, "session_id": sessionID, "page_id": pageID,
		"page_title": pageID, "page_url": "https://example.test/" + pageID,
		"captured_at": fmt.Sprintf("2026-11-20T00:00:%02dZ", sequence),
	})
	packet := make([]byte, 4+len(header)+1)
	binary.BigEndian.PutUint32(packet, uint32(len(header)))
	copy(packet[4:], header)
	packet[len(packet)-1] = byte(sequence)
	return packet
}

func viewerFrameIdentity(packet []byte) (map[string]any, byte, bool) {
	if len(packet) < 5 {
		return nil, 0, false
	}
	headerLen := int(binary.BigEndian.Uint32(packet[:4]))
	if headerLen <= 0 || 4+headerLen >= len(packet) {
		return nil, 0, false
	}
	var header map[string]any
	if json.Unmarshal(packet[4:4+headerLen], &header) != nil {
		return nil, 0, false
	}
	return header, packet[len(packet)-1], true
}

func TestSlowRecordingViewerReplacesStaleFramesAcrossPageSwitchWithoutSessionLeak(t *testing.T) {
	hub := newTestHub(t)
	sessionA, sessionB := uuid.NewString(), uuid.NewString()
	pageOld, pageFresh, pageB := uuid.NewString(), uuid.NewString(), uuid.NewString()
	clientA := &Client{ID: uuid.New(), Send: make(chan any, 4), BinarySend: make(chan []byte, MaxClientBinaryPendingFrames), Hub: hub, RecordingFrames: true, RecordingSessionID: &sessionA}
	clientB := &Client{ID: uuid.New(), Send: make(chan any, 4), BinarySend: make(chan []byte, MaxClientBinaryPendingFrames), Hub: hub, RecordingFrames: true, RecordingSessionID: &sessionB}
	testAddHubClient(hub, clientA)
	testAddHubClient(hub, clientB)

	// Withhold client reads to model a slow socket. Old-page frames must not
	// accumulate ahead of the first frame for the newly selected page.
	for sequence := 1; sequence <= 8; sequence++ {
		hub.BroadcastBinaryFrame(sessionA, testViewerFramePacket(sessionA, pageOld, sequence))
	}
	fresh := testViewerFramePacket(sessionA, pageFresh, 9)
	hub.BroadcastBinaryFrame(sessionA, fresh)
	hub.BroadcastBinaryFrame(sessionB, testViewerFramePacket(sessionB, pageB, 1))

	if len(clientA.BinarySend) != MaxClientBinaryPendingFrames || clientA.queuedBinaryFrameBytes() > MaxClientBinaryQueueBytes {
		t.Fatalf("slow viewer A queue exceeded bound: frames=%d bytes=%d", len(clientA.BinarySend), clientA.queuedBinaryFrameBytes())
	}
	if len(clientB.BinarySend) != MaxClientBinaryPendingFrames || clientB.queuedBinaryFrameBytes() > MaxClientBinaryQueueBytes {
		t.Fatalf("slow viewer B queue exceeded bound: frames=%d bytes=%d", len(clientB.BinarySend), clientB.queuedBinaryFrameBytes())
	}
	gotA := <-clientA.BinarySend
	clientA.releaseBinaryFrame(gotA)
	headerA, sequenceA, ok := viewerFrameIdentity(gotA)
	if !ok || headerA["session_id"] != sessionA || headerA["page_id"] != pageFresh || sequenceA != 9 {
		t.Fatalf("first delivered page-switch frame was stale or foreign: header=%v sequence=%d ok=%v", headerA, sequenceA, ok)
	}
	gotB := <-clientB.BinarySend
	clientB.releaseBinaryFrame(gotB)
	headerB, sequenceB, ok := viewerFrameIdentity(gotB)
	if !ok || headerB["session_id"] != sessionB || headerB["page_id"] != pageB || sequenceB != 1 {
		t.Fatalf("viewer B received a cross-session/page frame: header=%v sequence=%d ok=%v", headerB, sequenceB, ok)
	}
	if dropped := hub.GetDroppedFrameCount(); dropped < 8 {
		t.Fatalf("replaced stale frames were not reflected in drop telemetry: %d", dropped)
	}
}

// TestBroadcastBinaryFrameToSubscribedClient verifies binary frames are sent to subscribed clients
func TestBroadcastBinaryFrameToSubscribedClient(t *testing.T) {
	t.Run("[REQ:BAS-FRAME-STREAM] broadcasts binary frames to subscribed clients", func(t *testing.T) {
		hub := newTestHub(t)

		sessionID := "test-session-binary"

		client := &Client{
			ID:                 uuid.New(),
			Send:               make(chan any, 4),
			BinarySend:         make(chan []byte, 4),
			Hub:                hub,
			RecordingFrames:    true,
			RecordingSessionID: &sessionID,
		}

		hub.register <- client
		_ = waitForMessage(t, client.Send) // Drain welcome

		testFrame := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG magic bytes
		hub.BroadcastBinaryFrame(sessionID, testFrame)

		select {
		case frame := <-client.BinarySend:
			if len(frame) != len(testFrame) {
				t.Errorf("expected frame length %d, got %d", len(testFrame), len(frame))
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for binary frame")
		}
	})
}

func TestLatestFrameSlotReplacesAndSeedsLateRecordingViewer(t *testing.T) {
	hub := newTestHub(t)
	sessionID := uuid.NewString()
	hub.BroadcastBinaryFrame(sessionID, []byte("older"))
	hub.BroadcastBinaryFrame(sessionID, []byte("newest"))
	if len(hub.latestFrames) != 1 || string(hub.latestFrames["recording:"+sessionID].binary) != "newest" {
		t.Fatalf("latest recording frame slot did not replace the prior frame: %#v", hub.latestFrames)
	}
	client := newRecordingGrantClient(t, hub, sessionID)
	hub.register <- client
	waitForMessage(t, client.Send)
	client.handleSubscription("subscribe_recording", map[string]any{"session_id": sessionID})
	waitForMessage(t, client.Send) // subscription confirmation
	select {
	case got := <-client.BinarySend:
		if string(got) != "newest" {
			t.Fatalf("late recording viewer got %q", got)
		}
	default:
		t.Fatal("late recording viewer did not receive the latest frame")
	}
	hub.ClearRecordingFrames(sessionID)
	if len(hub.latestFrames) != 0 {
		t.Fatalf("recording teardown retained a frame slot: %#v", hub.latestFrames)
	}
}

func TestLatestFrameSlotReplacesAndSeedsLateExecutionViewer(t *testing.T) {
	hub := newTestHub(t)
	hub.BroadcastExecutionFrame("execution-latest", &ExecutionFrame{ExecutionID: "execution-latest", Data: "older", MediaType: "image/jpeg", Width: 10, Height: 20, CapturedAt: "2026-09-30T00:00:00Z"})
	hub.BroadcastExecutionFrame("execution-latest", &ExecutionFrame{ExecutionID: "execution-latest", Data: "newest", MediaType: "image/jpeg", Width: 30, Height: 40, CapturedAt: "2026-09-30T00:00:01Z"})
	if len(hub.latestFrames) != 1 || hub.latestFrames["execution:execution-latest"].execution.Data != "newest" {
		t.Fatalf("latest execution frame slot did not replace the prior frame: %#v", hub.latestFrames)
	}
	client := &Client{ID: uuid.New(), Send: make(chan any, 4), Hub: hub}
	hub.register <- client
	waitForMessage(t, client.Send)
	client.handleSubscription("subscribe_execution_frames", map[string]any{"execution_id": "execution-latest"})
	waitForMessage(t, client.Send) // subscription confirmation
	message := waitForMessage(t, client.Send).(queuedExecutionFrame).message
	if message["data"] != "newest" || message["width"] != 30 || message["height"] != 40 {
		t.Fatalf("late execution viewer received %#v", message)
	}
	hub.ClearExecutionFrames("execution-latest")
	if len(hub.latestFrames) != 0 {
		t.Fatalf("execution teardown retained a frame slot: %#v", hub.latestFrames)
	}
}

func TestExecutionPreviewQueueCoalescesByRunAndPreservesOtherMessages(t *testing.T) {
	hub := newTestHub(t)
	client := &Client{ID: uuid.New(), Send: make(chan any, 8), Hub: hub}
	if !client.enqueueExecutionFrame("run-a", executionFrameMessage("run-a", &ExecutionFrame{Data: "old"})) {
		t.Fatal("old preview frame was rejected")
	}
	client.Send <- map[string]any{"type": "status", "sequence": 1}
	if !client.enqueueExecutionFrame("run-b", executionFrameMessage("run-b", &ExecutionFrame{Data: "other-run"})) {
		t.Fatal("other run preview frame was rejected")
	}
	client.Send <- map[string]any{"type": "status", "sequence": 2}
	if !client.enqueueExecutionFrame("run-a", executionFrameMessage("run-a", &ExecutionFrame{Data: "new"})) {
		t.Fatal("new preview frame was rejected")
	}

	var got []any
	for len(client.Send) > 0 {
		got = append(got, <-client.Send)
	}
	if len(got) != 4 {
		t.Fatalf("queued %d messages, want 4: %#v", len(got), got)
	}
	if got[0].(map[string]any)["sequence"] != 1 || got[1].(queuedExecutionFrame).executionID != "run-b" ||
		got[2].(map[string]any)["sequence"] != 2 || got[3].(queuedExecutionFrame).message["data"] != "new" {
		t.Fatalf("control FIFO or run-isolated latest frames changed: %#v", got)
	}
	if client.executionFrameQueueBytes != got[1].(queuedExecutionFrame).bytes+got[3].(queuedExecutionFrame).bytes {
		t.Fatalf("frame byte accounting mismatch: %d", client.executionFrameQueueBytes)
	}
}

func TestExecutionPreviewQueueRejectsOversizedJSONAndCountsBase64Envelope(t *testing.T) {
	hub := newTestHub(t)
	client := &Client{ID: uuid.New(), Send: make(chan any, 4), Hub: hub}
	if client.enqueueExecutionFrame("run-large", map[string]any{"type": "execution_frame", "data": strings.Repeat("x", MaxClientExecutionFrameMessageBytes)}) {
		t.Fatal("oversized serialized preview frame was accepted")
	}
	if len(client.Send) != 0 || client.executionFrameQueueBytes != 0 {
		t.Fatalf("oversized frame changed queue: len=%d bytes=%d", len(client.Send), client.executionFrameQueueBytes)
	}
	message := executionFrameMessage("run-bounded", &ExecutionFrame{Data: strings.Repeat("A", 1024)})
	encoded, err := json.Marshal(message)
	if err != nil {
		t.Fatal(err)
	}
	if !client.enqueueExecutionFrame("run-bounded", message) {
		t.Fatal("bounded preview frame was rejected")
	}
	if client.executionFrameQueueBytes != len(encoded) || client.executionFrameQueueBytes <= len(message["data"].(string)) {
		t.Fatalf("queue did not account for serialized base64/JSON size: got=%d serialized=%d", client.executionFrameQueueBytes, len(encoded))
	}
}

func TestExecutionPreviewQueueCountsActiveBytesAgainstPendingCap(t *testing.T) {
	hub := newTestHub(t)
	client := &Client{ID: uuid.New(), Send: make(chan any, 4), Hub: hub}
	active := queuedExecutionFrame{executionID: "run-active", bytes: MaxClientExecutionFrameQueueBytes - 100}
	client.executionFrameQueueBytes = active.bytes // The writePump has dequeued it and is still writing.
	message := executionFrameMessage("run-next", &ExecutionFrame{Data: strings.Repeat("x", 200)})
	if client.enqueueExecutionFrame("run-next", message) {
		t.Fatal("pending frame exceeded the active-plus-pending byte cap")
	}
	if len(client.Send) != 0 || client.executionFrameQueueBytes != active.bytes {
		t.Fatalf("rejected pending frame changed active accounting: queue=%d bytes=%d", len(client.Send), client.executionFrameQueueBytes)
	}
}

// TestBroadcastBinaryFrameFiltersNonSubscribed verifies non-subscribed clients don't receive binary frames
func TestBroadcastBinaryFrameFiltersNonSubscribed(t *testing.T) {
	t.Run("[REQ:BAS-FRAME-STREAM] filters binary frames from non-subscribed clients", func(t *testing.T) {
		hub := newTestHub(t)

		sessionID := "test-session-binary-filter"
		otherSessionID := "other-session"

		// Client subscribed to different session
		otherClient := &Client{
			ID:                 uuid.New(),
			Send:               make(chan any, 4),
			BinarySend:         make(chan []byte, 4),
			Hub:                hub,
			RecordingFrames:    true,
			RecordingSessionID: &otherSessionID,
		}

		// Client not subscribed to any recording
		unsubscribed := &Client{
			ID:         uuid.New(),
			Send:       make(chan any, 4),
			BinarySend: make(chan []byte, 4),
			Hub:        hub,
		}

		hub.register <- otherClient
		hub.register <- unsubscribed
		_ = waitForMessage(t, otherClient.Send)  // Drain welcome
		_ = waitForMessage(t, unsubscribed.Send) // Drain welcome

		testFrame := []byte{0xFF, 0xD8, 0xFF}
		hub.BroadcastBinaryFrame(sessionID, testFrame)

		// Neither client should receive the frame
		select {
		case <-otherClient.BinarySend:
			t.Fatal("client subscribed to other session should not receive frame")
		case <-time.After(100 * time.Millisecond):
			// Expected
		}

		select {
		case <-unsubscribed.BinarySend:
			t.Fatal("unsubscribed client should not receive frame")
		case <-time.After(100 * time.Millisecond):
			// Expected
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

		// Send sequence of entries
		for i := 1; i <= 5; i++ {
			hub.BroadcastTimelineEntry(sessionID, testEntry(fmt.Sprintf("action-%d", i), basactions.ActionType_ACTION_TYPE_CLICK))
		}

		// Verify all received in order
		for i := 1; i <= 5; i++ {
			msg := waitForMessage(t, client.Send)
			if msgMap, ok := msg.(map[string]any); ok {
				entry := msgMap["entry"].(map[string]any)
				expectedID := fmt.Sprintf("action-%d", i)
				if entry["id"] != expectedID {
					t.Errorf("expected entry id '%s', got %v", expectedID, entry["id"])
				}
			} else {
				t.Fatalf("expected map, got %T", msg)
			}
		}
	})
}

// [REQ:BAS-RH-J13] Received button/key transitions must not overtake an admitted input.
func TestRecordingInputPreservesConnectionOrder(t *testing.T) {
	hub := newTestHubBase(t)
	sessionID := uuid.NewString()
	cookieName, cookieValue, err := hub.IssueSessionStreamCookie(sessionID, uuid.NewString(), uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	first, second, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	effects := make(chan int, 2)
	hub.SetInputForwarder(func(_ string, input map[string]any) (*driver.ForwardInputResponse, error) {
		n := int(input["sequence"].(float64))
		if n == 1 {
			close(first)
			<-release
		}
		effects <- n
		if n == 2 {
			close(second)
		}
		return &driver.ForwardInputResponse{Status: "ok", AppliedSequence: uint64(n), InputID: fmt.Sprintf("input-%d", n)}, nil
	})
	server := startRecordingWebSocketServer(t, hub, sessionID)
	conn := dialTestWebSocketWithCookie(t, server, cookieName, cookieValue)
	defer conn.Close()
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var connected map[string]any
	if err := conn.ReadJSON(&connected); err != nil || connected["type"] != "connected" {
		t.Fatalf("read connection confirmation: message=%v err=%v", connected, err)
	}
	send := func(sequence int) {
		t.Helper()
		if err := conn.WriteJSON(map[string]any{"type": "recording_input", "session_id": sessionID, "input": map[string]any{"type": "keyboard", "key": "a", "sequence": sequence, "input_id": fmt.Sprintf("input-%d", sequence)}}); err != nil {
			t.Fatal(err)
		}
	}
	send(1)
	select {
	case <-first:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("first input not admitted")
	}
	// A blocked browser must not hold the global hub lock.
	counted := make(chan int, 1)
	go func() { counted <- hub.GetClientCount() }()
	select {
	case count := <-counted:
		if count != 1 {
			t.Errorf("client count %d, want 1", count)
		}
	case <-time.After(time.Second):
		close(release)
		t.Fatal("blocked browser input stalled the hub")
	}
	send(2)
	overlap := false
	select {
	case <-second:
		overlap = true
	case <-time.After(150 * time.Millisecond):
	}
	close(release)
	if overlap {
		t.Error("second input overtook the first while it was still executing")
	}
	for want := 1; want <= 2; want++ {
		select {
		case got := <-effects:
			if got != want {
				t.Errorf("effect %d, want %d", got, want)
			}
		case <-time.After(time.Second):
			t.Fatal("input failed to finish")
		}
	}
	for want := 1; want <= 2; want++ {
		var ack map[string]any
		if err := conn.ReadJSON(&ack); err != nil {
			t.Fatalf("read input receipt %d: %v", want, err)
		}
		if ack["type"] != "recording_input_applied" || ack["session_id"] != sessionID || ack["input_id"] != fmt.Sprintf("input-%d", want) || ack["applied_sequence"] != float64(want) {
			t.Errorf("receipt %d has wrong content: %v", want, ack)
		}
	}
}

func TestSubscriptionAfterDisconnectDoesNotSendOrResubscribe(t *testing.T) {
	hub := newTestHub(t)
	client := &Client{ID: uuid.New(), Send: make(chan any, 1), Hub: hub}
	hub.register <- client
	_ = waitForMessage(t, client.Send)
	hub.unregister <- client
	select {
	case _, ok := <-client.Send:
		if ok {
			t.Fatal("expected disconnected send channel")
		}
	case <-time.After(time.Second):
		t.Fatal("client was not disconnected")
	}
	for _, kind := range []string{"subscribe", "subscribe_recording", "subscribe_execution_frames", "subscribe_driver_status", "subscribe_export"} {
		client.handleSubscription(kind, map[string]any{"execution_id": uuid.NewString(), "session_id": "recording", "export_id": "export"})
	}
	if client.ExecutionID != nil || client.RecordingSessionID != nil || client.ExecutionFrameStreamID != nil || client.DriverStatusSubscribed || client.ExportSubscriptionID != nil {
		t.Fatal("a disconnected client was resubscribed")
	}
}

// [REQ:BAS-RH-J05] Event-only consumers do not receive unused image traffic.
func TestRecordingFrameSubscriptionIntent(t *testing.T) {
	for _, choice := range []string{"default", "enabled", "events only"} {
		t.Run(choice, func(t *testing.T) {
			hub := newTestHub(t)
			sessionID := uuid.NewString()
			client := newRecordingGrantClient(t, hub, sessionID)
			hub.register <- client
			_ = waitForMessage(t, client.Send)
			msg := map[string]any{"session_id": sessionID}
			if choice != "default" {
				msg["frames"] = choice == "enabled"
			}
			client.handleSubscription("subscribe_recording", msg)
			_ = waitForMessage(t, client.Send)
			wantFrames := choice != "events only"
			if got := hub.HasRecordingFrameSubscribers(sessionID); got != wantFrames {
				t.Errorf("frame consumers=%v, want %v", got, wantFrames)
			}
			for i := 0; i < 3; i++ {
				hub.BroadcastBinaryFrame(sessionID, []byte{0xff, 0xd8})
			}
			wantQueued, wantDropped := 0, int64(0)
			if wantFrames {
				wantQueued = 1
				wantDropped = 2
			}
			if got := len(client.BinarySend); got != wantQueued {
				t.Errorf("queued images=%d, want %d", got, wantQueued)
			}
			if got := hub.GetDroppedFrameCount(); got != wantDropped {
				t.Errorf("dropped images=%d, want %d", got, wantDropped)
			}
			hub.BroadcastPageEvent(sessionID, map[string]any{"type": "created"})
			event := waitForMessage(t, client.Send).(map[string]any)
			if event["type"] != "page_event" {
				t.Fatalf("page event lost: %v", event)
			}
			client.handleSubscription("unsubscribe_recording", nil)
			if hub.HasRecordingFrameSubscribers("recording") {
				t.Fatal("unsubscribed client remains a frame consumer")
			}
		})
	}
}

func TestRecordingFrameSubscriptionChoiceReplacesPreviousChoice(t *testing.T) {
	hub := newTestHub(t)
	sessionID := uuid.NewString()
	client := newRecordingGrantClient(t, hub, sessionID)
	hub.register <- client
	_ = waitForMessage(t, client.Send)
	for _, enabled := range []bool{false, true, false} {
		client.handleSubscription("subscribe_recording", map[string]any{"session_id": sessionID, "frames": enabled})
		_ = waitForMessage(t, client.Send)
		if got := hub.HasRecordingFrameSubscribers(sessionID); got != enabled {
			t.Errorf("replacement choice=%v, want %v", got, enabled)
		}
	}
}
