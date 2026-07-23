package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/protoconv"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestHandleWebSocketSupportsTypedAndLegacyClientMessages(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc((&Handler{}).HandleWebSocket(hub)))
	t.Cleanup(server.Close)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	readMessage := func() *domainpb.AgentManagerWsMessage {
		t.Helper()
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read websocket message: %v", err)
		}
		var message domainpb.AgentManagerWsMessage
		if err := protoconv.UnmarshalJSON(data, &message); err != nil {
			t.Fatalf("decode websocket message: %v", err)
		}
		return &message
	}
	writeMessage := func(data []byte) {
		t.Helper()
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			t.Fatalf("write websocket message: %v", err)
		}
	}

	if welcome := readMessage(); welcome.Type != domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_CONNECTED || welcome.GetConnected() == nil {
		t.Fatalf("expected connected welcome, got %+v", welcome)
	}

	typedPing, err := protoconv.MarshalJSON(&domainpb.AgentManagerWsClientMessage{Type: domainpb.AgentManagerWsClientMessageType_AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_PING})
	if err != nil {
		t.Fatalf("marshal typed ping: %v", err)
	}
	writeMessage(typedPing)
	if pong := readMessage(); pong.Type != domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_PONG || pong.GetPong() == nil {
		t.Fatalf("expected typed pong, got %+v", pong)
	}

	runID := uuid.New()
	subscribe, err := protoconv.MarshalJSON(&domainpb.AgentManagerWsClientMessage{
		Type:    domainpb.AgentManagerWsClientMessageType_AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_SUBSCRIBE,
		Payload: &domainpb.AgentManagerWsClientMessage_RunSubscription{RunSubscription: &domainpb.RunSubscription{RunId: runID.String()}},
	})
	if err != nil {
		t.Fatalf("marshal subscription: %v", err)
	}
	writeMessage(subscribe)
	hub.BroadcastProgress(runID, domain.RunPhaseExecuting, 45, "checking durable state")
	if progress := readMessage(); progress.Type != domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_PROGRESS || progress.GetRunId() != runID.String() || progress.GetRunProgress().GetPercentComplete() != 45 {
		t.Fatalf("expected subscribed progress update, got %+v", progress)
	}

	writeMessage([]byte(`{"type":"ping"}`))
	if pong := readMessage(); pong.Type != domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_PONG {
		t.Fatalf("expected legacy pong, got %+v", pong)
	}
}

// =============================================================================
// HUB LIFECYCLE TESTS
// =============================================================================

func TestNewWebSocketHub(t *testing.T) {
	hub := NewWebSocketHub()
	if hub == nil {
		t.Fatal("expected non-nil hub")
	}
	if hub.clients == nil {
		t.Fatal("expected clients map to be initialized")
	}
	if cap(hub.broadcast) != 256 {
		t.Errorf("expected broadcast channel capacity 256, got %d", cap(hub.broadcast))
	}
}

// =============================================================================
// BROADCAST FILTERING TESTS
// =============================================================================

func TestWebSocketHub_BroadcastFiltering_AllEvents(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	// Register a client subscribed to all events
	client := &WebSocketClient{
		hub:           hub,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
		allEvents:     true,
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond) // let registration process

	// Broadcast a run event
	runID := uuid.New().String()
	hub.broadcast <- &domainpb.AgentManagerWsMessage{
		Type:  domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS,
		RunId: &runID,
		Payload: &domainpb.AgentManagerWsMessage_RunStatus{
			RunStatus: &domainpb.RunStatusUpdate{
				RunId:  runID,
				Status: domainpb.RunStatus_RUN_STATUS_COMPLETE,
			},
		},
	}

	select {
	case msg := <-client.send:
		if len(msg) == 0 {
			t.Error("expected non-empty message")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast message")
	}
}

func TestWebSocketHub_BroadcastFiltering_SubscribedRun(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	subscribedRunID := uuid.New().String()
	otherRunID := uuid.New().String()

	client := &WebSocketClient{
		hub:           hub,
		send:          make(chan []byte, 256),
		subscriptions: map[string]bool{subscribedRunID: true},
		allEvents:     false,
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Broadcast for unsubscribed run — should NOT be received
	hub.broadcast <- &domainpb.AgentManagerWsMessage{
		Type:  domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT,
		RunId: &otherRunID,
		Payload: &domainpb.AgentManagerWsMessage_RunEvent{
			RunEvent: &domainpb.RunEvent{
				Id:    uuid.New().String(),
				RunId: otherRunID,
			},
		},
	}

	// Broadcast for subscribed run — SHOULD be received
	hub.broadcast <- &domainpb.AgentManagerWsMessage{
		Type:  domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT,
		RunId: &subscribedRunID,
		Payload: &domainpb.AgentManagerWsMessage_RunEvent{
			RunEvent: &domainpb.RunEvent{
				Id:    uuid.New().String(),
				RunId: subscribedRunID,
			},
		},
	}

	// Wait for both broadcasts to process
	time.Sleep(50 * time.Millisecond)

	// Should have exactly one message (the subscribed run)
	if len(client.send) != 1 {
		t.Fatalf("expected 1 message, got %d", len(client.send))
	}
	msg := <-client.send
	var parsed domainpb.AgentManagerWsMessage
	if err := protoconv.UnmarshalJSON(msg, &parsed); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}
	if parsed.GetRunId() != subscribedRunID {
		t.Errorf("expected run_id %s, got %s", subscribedRunID, parsed.GetRunId())
	}
}

func TestWebSocketHub_RunStatusBroadcastsToUnsubscribedClients(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	client := &WebSocketClient{
		hub:           hub,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
		allEvents:     false,
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	runID := uuid.New().String()
	hub.broadcast <- &domainpb.AgentManagerWsMessage{
		Type:  domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS,
		RunId: &runID,
		Payload: &domainpb.AgentManagerWsMessage_RunStatus{
			RunStatus: &domainpb.RunStatusUpdate{
				RunId:  runID,
				Status: domainpb.RunStatus_RUN_STATUS_RUNNING,
			},
		},
	}

	select {
	case msg := <-client.send:
		var parsed domainpb.AgentManagerWsMessage
		if err := protoconv.UnmarshalJSON(msg, &parsed); err != nil {
			t.Fatalf("failed to unmarshal message: %v", err)
		}
		if parsed.Type != domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS {
			t.Errorf("expected RUN_STATUS type, got %v", parsed.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for global run_status broadcast")
	}
}

// =============================================================================
// STALE CLIENT REMOVAL (Bug 3 fix validation)
// =============================================================================

func TestWebSocketHub_StaleClientRemoval_NoRace(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	// Create a client with a tiny buffer that will fill up
	staleClient := &WebSocketClient{
		hub:           hub,
		send:          make(chan []byte, 1), // tiny buffer
		subscriptions: make(map[string]bool),
		allEvents:     true,
	}
	hub.register <- staleClient
	time.Sleep(10 * time.Millisecond)

	// Fill the buffer
	staleClient.send <- []byte("filler")

	runID := uuid.New().String()

	// Broadcast — should trigger stale client removal under write lock
	hub.broadcast <- &domainpb.AgentManagerWsMessage{
		Type:  domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS,
		RunId: &runID,
		Payload: &domainpb.AgentManagerWsMessage_RunStatus{
			RunStatus: &domainpb.RunStatusUpdate{
				RunId:  runID,
				Status: domainpb.RunStatus_RUN_STATUS_COMPLETE,
			},
		},
	}

	time.Sleep(50 * time.Millisecond)

	// Verify client was removed
	hub.mu.RLock()
	_, exists := hub.clients[staleClient]
	hub.mu.RUnlock()
	if exists {
		t.Error("expected stale client to be removed")
	}
}

func TestWebSocketHub_StaleClientRemoval_ConcurrentSafety(t *testing.T) {
	// Stress test: many goroutines broadcasting simultaneously while clients
	// are being registered/unregistered. This catches the data race that
	// existed before the RLock fix (Bug 3).
	hub := NewWebSocketHub()
	go hub.Run()

	const numClients = 10
	const numBroadcasts = 50
	var wg sync.WaitGroup

	// Register clients with varying buffer sizes
	for i := 0; i < numClients; i++ {
		client := &WebSocketClient{
			hub:           hub,
			send:          make(chan []byte, 2),
			subscriptions: make(map[string]bool),
			allEvents:     true,
		}
		hub.register <- client
	}
	time.Sleep(20 * time.Millisecond)

	// Hammer broadcasts from multiple goroutines
	for i := 0; i < numBroadcasts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runID := uuid.New().String()
			hub.broadcast <- &domainpb.AgentManagerWsMessage{
				Type:  domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT,
				RunId: &runID,
				Payload: &domainpb.AgentManagerWsMessage_RunEvent{
					RunEvent: &domainpb.RunEvent{
						Id:    uuid.New().String(),
						RunId: runID,
					},
				},
			}
		}()
	}

	wg.Wait()
	// If we get here without a data race panic, the fix works.
	// Run with -race flag to verify: go test -race -run TestWebSocketHub_StaleClient
}

func TestWebSocketHub_SubscriptionMutationConcurrentSafety(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	runID := uuid.New().String()
	client := &WebSocketClient{
		hub:           hub,
		send:          make(chan []byte, 4096),
		subscriptions: make(map[string]bool),
		allEvents:     false,
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			updateSubscription(client, runID, i%2 == 0)
			updateAllEventsSubscription(client, i%3 == 0)
		}(i)
		go func() {
			defer wg.Done()
			hub.broadcast <- &domainpb.AgentManagerWsMessage{
				Type:  domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT,
				RunId: &runID,
				Payload: &domainpb.AgentManagerWsMessage_RunEvent{
					RunEvent: &domainpb.RunEvent{
						Id:    uuid.New().String(),
						RunId: runID,
					},
				},
			}
		}()
	}
	wg.Wait()
}

// =============================================================================
// BROADCAST RUN STATUS WITH DISPLAY FIELDS (Bug 2 fix validation)
// =============================================================================

func TestBroadcastRunStatus_IncludesDisplayFields(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	client := &WebSocketClient{
		hub:           hub,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
		allEvents:     true,
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	run := &domain.Run{
		ID:            uuid.New(),
		TaskID:        uuid.New(),
		Status:        domain.RunStatusComplete,
		PromptPreview: "Fix the authentication bug in login flow",
		Result: &domain.RunResult{Selection: domain.FinalOutputSelection{
			Status: domain.FinalOutputSelectionSelected,
			Rule:   "unique_terminal_main_assistant",
		}},
	}

	hub.BroadcastRunStatus(run)

	select {
	case msg := <-client.send:
		var parsed domainpb.AgentManagerWsMessage
		if err := protoconv.UnmarshalJSON(msg, &parsed); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		status := parsed.GetRunStatus()
		if status == nil {
			t.Fatal("expected run_status payload")
		}
		if status.RunId != run.ID.String() {
			t.Errorf("expected run_id %s, got %s", run.ID, status.RunId)
		}
		if status.TaskId != run.TaskID.String() {
			t.Errorf("expected task_id %s, got %s", run.TaskID, status.TaskId)
		}
		if status.PromptPreview != run.PromptPreview {
			t.Errorf("expected prompt_preview %q, got %q", run.PromptPreview, status.PromptPreview)
		}
		if status.ResultSelectionStatus != domainpb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_SELECTED {
			t.Errorf("expected selected result status, got %v", status.ResultSelectionStatus)
		}
		if status.ResultSelectionRule != "unique_terminal_main_assistant" {
			t.Errorf("unexpected result selection rule %q", status.ResultSelectionRule)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}
}

// =============================================================================
// MESSAGE FRAME INTEGRITY (Bug 4 fix validation)
// =============================================================================

func TestWebSocketHub_EachMessageIsSeparateFrame(t *testing.T) {
	// Verify that when multiple messages are queued, they arrive as separate
	// valid JSON objects (not concatenated with newlines).
	hub := NewWebSocketHub()
	go hub.Run()

	client := &WebSocketClient{
		hub:           hub,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
		allEvents:     true,
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Send two broadcasts rapidly
	for i := 0; i < 2; i++ {
		runID := uuid.New().String()
		hub.broadcast <- &domainpb.AgentManagerWsMessage{
			Type:  domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS,
			RunId: &runID,
			Payload: &domainpb.AgentManagerWsMessage_RunStatus{
				RunStatus: &domainpb.RunStatusUpdate{
					RunId:  runID,
					Status: domainpb.RunStatus_RUN_STATUS_RUNNING,
				},
			},
		}
	}

	time.Sleep(50 * time.Millisecond)

	// Each message in the send channel should be independently valid JSON
	count := len(client.send)
	if count < 2 {
		t.Fatalf("expected at least 2 messages in channel, got %d", count)
	}
	for i := 0; i < count; i++ {
		msg := <-client.send
		var raw map[string]interface{}
		if err := json.Unmarshal(msg, &raw); err != nil {
			t.Errorf("message %d is not valid JSON: %v\ncontent: %s", i, err, string(msg))
		}
	}
}

// =============================================================================
// BROADCAST RUN STATUS WITH EMPTY PROMPT PREVIEW (Bug fix validation)
// =============================================================================

func TestBroadcastRunStatus_EmptyPromptPreview(t *testing.T) {
	// Before the fix, PromptPreview was never populated during execution,
	// causing the UI to show "Unknown Task". This test validates that
	// BroadcastRunStatus faithfully transmits whatever PromptPreview is set.
	hub := NewWebSocketHub()
	go hub.Run()

	client := &WebSocketClient{
		hub:           hub,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
		allEvents:     true,
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	// Case 1: Empty PromptPreview (the bug scenario)
	runEmpty := &domain.Run{
		ID:     uuid.New(),
		TaskID: uuid.New(),
		Status: domain.RunStatusComplete,
		// PromptPreview intentionally empty
	}
	hub.BroadcastRunStatus(runEmpty)

	select {
	case msg := <-client.send:
		var parsed domainpb.AgentManagerWsMessage
		if err := protoconv.UnmarshalJSON(msg, &parsed); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		status := parsed.GetRunStatus()
		if status == nil {
			t.Fatal("expected run_status payload")
		}
		if status.PromptPreview != "" {
			t.Errorf("expected empty prompt_preview for unpopulated run, got %q", status.PromptPreview)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}

	// Case 2: Populated PromptPreview (after fix)
	runPopulated := &domain.Run{
		ID:            uuid.New(),
		TaskID:        uuid.New(),
		Status:        domain.RunStatusRunning,
		PromptPreview: "Implement user authentication",
	}
	hub.BroadcastRunStatus(runPopulated)

	select {
	case msg := <-client.send:
		var parsed domainpb.AgentManagerWsMessage
		if err := protoconv.UnmarshalJSON(msg, &parsed); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		status := parsed.GetRunStatus()
		if status == nil {
			t.Fatal("expected run_status payload")
		}
		if status.PromptPreview != "Implement user authentication" {
			t.Errorf("expected prompt_preview %q, got %q", "Implement user authentication", status.PromptPreview)
		}
		// Also verify non-terminal status is correctly transmitted
		if status.Status != domainpb.RunStatus_RUN_STATUS_RUNNING {
			t.Errorf("expected RUNNING status, got %v", status.Status)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}
}

func TestBroadcastWorkflowLifecycleContainsMetadataOnly(t *testing.T) {
	hub := NewWebSocketHub()
	executionID := uuid.New()
	runID := uuid.New()
	hub.BroadcastWorkflowLifecycle(&domain.WorkflowLifecycleEvent{ExecutionID: executionID, DefinitionDigest: "sha256:def", Status: domain.WorkflowExecutionRunning, NodeID: "review", Strategy: domain.WorkflowAttemptFreshRun, ProfileIdentity: "reviewer", RunID: &runID, JournalSequence: 4, JournalKind: domain.WorkflowJournalAttempt, JournalPayloadDigest: "sha256:payload", BudgetUsage: domain.WorkflowBudgetUsage{NodeAttempts: 1}})

	message := <-hub.broadcast
	update := message.GetWorkflowLifecycle()
	if update == nil || update.ExecutionId != executionID.String() || update.RunId != runID.String() || update.JournalSequence != 4 {
		t.Fatalf("unexpected workflow lifecycle update: %+v", update)
	}
	data, err := protoconv.MarshalJSON(message)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"promptSnapshot", "\"result\":", "\"payload\":"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("workflow lifecycle broadcast leaked content field %q: %s", forbidden, data)
		}
	}
}

func TestBroadcastTaskStatus_NoRunIdFiltering(t *testing.T) {
	// BroadcastTaskStatus does not set RunId, which means the hub sends it
	// to ALL clients regardless of subscription state. This test documents
	// that behavior.
	hub := NewWebSocketHub()
	go hub.Run()

	specificRunID := uuid.New().String()
	client := &WebSocketClient{
		hub:           hub,
		send:          make(chan []byte, 256),
		subscriptions: map[string]bool{specificRunID: true},
		allEvents:     false,
	}
	hub.register <- client
	time.Sleep(10 * time.Millisecond)

	task := &domain.Task{
		ID:     uuid.New(),
		Status: domain.TaskStatusRunning,
	}
	hub.BroadcastTaskStatus(task)

	// Even though client is only subscribed to a specific run,
	// task_status messages (no RunId) are sent to ALL clients.
	select {
	case msg := <-client.send:
		var parsed domainpb.AgentManagerWsMessage
		if err := protoconv.UnmarshalJSON(msg, &parsed); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if parsed.Type != domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_TASK_STATUS {
			t.Errorf("expected TASK_STATUS type, got %v", parsed.Type)
		}
		// Verify no RunId is set
		if parsed.RunId != nil {
			t.Errorf("expected nil RunId for task_status, got %q", *parsed.RunId)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout: task_status should reach all clients")
	}
}
