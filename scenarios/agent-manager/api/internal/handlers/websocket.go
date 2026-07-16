package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/protoconv"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/types/known/timestamppb"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// WebSocket upgrader with permissive origin check for development
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// In production, this should validate against allowed origins
		return true
	},
}

// WebSocketHub manages WebSocket connections and event broadcasting
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan *domainpb.AgentManagerWsMessage
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mu         sync.RWMutex
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	hub           *WebSocketHub
	conn          *websocket.Conn
	send          chan []byte
	mu            sync.RWMutex
	subscriptions map[string]bool // runID -> subscribed
	allEvents     bool            // subscribe to all events
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan *domainpb.AgentManagerWsMessage, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
	}
}

// Run starts the hub's main loop
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			count := len(h.clients)
			h.mu.Unlock()
			log.Printf("[ws] Client connected. Total: %d", count)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			count := len(h.clients)
			h.mu.Unlock()
			log.Printf("[ws] Client disconnected. Total: %d", count)

		case message := <-h.broadcast:
			data, err := protoconv.MarshalJSON(message)
			if err != nil {
				log.Printf("[ws] Failed to marshal message: %v", err)
				continue
			}

			var stale []*WebSocketClient
			h.mu.RLock()
			for client := range h.clients {
				if !client.shouldReceive(message) {
					continue
				}
				select {
				case client.send <- data:
				default:
					// Client buffer full, schedule for removal
					stale = append(stale, client)
				}
			}
			h.mu.RUnlock()

			// Remove stale clients under write lock
			if len(stale) > 0 {
				h.mu.Lock()
				for _, client := range stale {
					if _, ok := h.clients[client]; ok {
						close(client.send)
						delete(h.clients, client)
					}
				}
				h.mu.Unlock()
			}
		}
	}
}

// BroadcastEvent broadcasts a run event to subscribed clients
func (h *WebSocketHub) BroadcastEvent(event *domain.RunEvent) {
	runID := event.RunID.String()
	h.broadcast <- &domainpb.AgentManagerWsMessage{
		Type:  domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT,
		RunId: &runID,
		Payload: &domainpb.AgentManagerWsMessage_RunEvent{
			RunEvent: protoconv.RunEventToProto(event),
		},
	}
}

// BroadcastRunStatus broadcasts a run status change
func (h *WebSocketHub) BroadcastRunStatus(run *domain.Run) {
	runID := run.ID.String()
	selectionStatus := domainpb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_UNSPECIFIED
	selectionRule := ""
	if run.Result != nil {
		selectionStatus = protoconv.FinalOutputSelectionStatusToProto(run.Result.Selection.Status)
		selectionRule = run.Result.Selection.Rule
	}
	h.broadcast <- &domainpb.AgentManagerWsMessage{
		Type:  domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS,
		RunId: &runID,
		Payload: &domainpb.AgentManagerWsMessage_RunStatus{
			RunStatus: &domainpb.RunStatusUpdate{
				RunId:                 runID,
				Status:                protoconv.RunStatusToProto(run.Status),
				TaskId:                run.TaskID.String(),
				PromptPreview:         run.PromptPreview,
				ResultSelectionStatus: selectionStatus,
				ResultSelectionRule:   selectionRule,
			},
		},
	}
}

// BroadcastTaskStatus broadcasts a task status change
func (h *WebSocketHub) BroadcastTaskStatus(task *domain.Task) {
	h.broadcast <- &domainpb.AgentManagerWsMessage{
		Type: domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_TASK_STATUS,
		Payload: &domainpb.AgentManagerWsMessage_TaskStatus{
			TaskStatus: &domainpb.TaskStatusUpdate{
				TaskId: task.ID.String(),
				Status: protoconv.TaskStatusToProto(task.Status),
			},
		},
	}
}

// BroadcastMessage broadcasts a generic message
func (h *WebSocketHub) BroadcastMessage(msgType string, payload interface{}) {
	log.Printf("[ws] BroadcastMessage is deprecated, msgType=%s", msgType)
}

// BroadcastProgress broadcasts a progress update for a run.
// This implements the orchestration.EventBroadcaster interface.
func (h *WebSocketHub) BroadcastProgress(runID uuid.UUID, phase domain.RunPhase, percent int, action string) {
	runIDStr := runID.String()
	h.broadcast <- &domainpb.AgentManagerWsMessage{
		Type:  domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_PROGRESS,
		RunId: &runIDStr,
		Payload: &domainpb.AgentManagerWsMessage_RunProgress{
			RunProgress: &domainpb.ProgressEventData{
				Phase:           protoconv.RunPhaseToProto(phase),
				PercentComplete: int32(percent),
				CurrentAction:   action,
			},
		},
	}
}

func (h *WebSocketHub) BroadcastWorkflowLifecycle(event *domain.WorkflowLifecycleEvent) {
	if event == nil {
		return
	}
	update := &domainpb.WorkflowLifecycleUpdate{ExecutionId: event.ExecutionID.String(), DefinitionDigest: event.DefinitionDigest, Status: string(event.Status), NodeId: event.NodeID, Strategy: string(event.Strategy), ProfileIdentity: event.ProfileIdentity, ConversationId: event.ConversationID, JournalSequence: event.JournalSequence, JournalKind: string(event.JournalKind), JournalPayloadDigest: event.JournalPayloadDigest, BudgetUsage: &domainpb.WorkflowBudgetUsage{Turns: int32(event.BudgetUsage.Turns), Tokens: int32(event.BudgetUsage.Tokens), CostUsd: event.BudgetUsage.CostUSD, NodeAttempts: int32(event.BudgetUsage.NodeAttempts), Children: int32(event.BudgetUsage.Children), Retries: int32(event.BudgetUsage.Retries)}}
	if event.RunID != nil {
		update.RunId = event.RunID.String()
	}
	if event.SourceAttemptID != nil {
		update.SourceAttemptId = event.SourceAttemptID.String()
	}
	if event.TerminalReason != nil {
		update.TerminalReason = &domainpb.WorkflowTerminalReason{Code: event.TerminalReason.Code, Message: event.TerminalReason.Message, Retryable: event.TerminalReason.Retryable, BudgetName: event.TerminalReason.BudgetName}
	}
	h.broadcast <- &domainpb.AgentManagerWsMessage{Type: domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_WORKFLOW_LIFECYCLE, Payload: &domainpb.AgentManagerWsMessage_WorkflowLifecycle{WorkflowLifecycle: update}}
}

// HandleWebSocket handles WebSocket connections
func (h *Handler) HandleWebSocket(hub *WebSocketHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[ws] Upgrade error: %v", err)
			return
		}

		client := &WebSocketClient{
			hub:           hub,
			conn:          conn,
			send:          make(chan []byte, 256),
			subscriptions: make(map[string]bool),
			allEvents:     false,
		}

		hub.register <- client

		// Send welcome message
		connected := &domainpb.AgentManagerWsMessage{
			Type: domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_CONNECTED,
			Payload: &domainpb.AgentManagerWsMessage_Connected{
				Connected: &domainpb.WsConnected{
					Message:   "Connected to agent-manager WebSocket",
					Timestamp: timestamppb.New(time.Now().UTC()),
				},
			},
		}
		welcomeData, _ := protoconv.MarshalJSON(connected)
		client.send <- welcomeData

		// Start read and write goroutines
		go client.writePump()
		go client.readPump()
	}
}

// readPump handles incoming messages from a client
func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB max message size
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[ws] Read error: %v", err)
			}
			break
		}

		var msg domainpb.AgentManagerWsClientMessage
		if err := protoconv.UnmarshalJSON(message, &msg); err != nil {
			// Fallback to legacy JSON parsing for backward compatibility.
			var legacy struct {
				Type    string `json:"type"`
				Payload struct {
					RunID string `json:"runId,omitempty"`
				} `json:"payload,omitempty"`
			}
			if err := json.Unmarshal(message, &legacy); err != nil {
				log.Printf("[ws] Failed to parse message: %v", err)
				continue
			}
			switch legacy.Type {
			case "ping":
				c.send <- buildPongPayload()
			case "subscribe":
				updateSubscription(c, legacy.Payload.RunID, true)
			case "unsubscribe":
				updateSubscription(c, legacy.Payload.RunID, false)
			case "subscribe_all":
				updateAllEventsSubscription(c, true)
				log.Printf("[ws] Client subscribed to all events")
			case "unsubscribe_all":
				updateAllEventsSubscription(c, false)
				log.Printf("[ws] Client unsubscribed from all events")
			}
			continue
		}

		switch msg.Type {
		case domainpb.AgentManagerWsClientMessageType_AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_PING:
			c.send <- buildPongPayload()

		case domainpb.AgentManagerWsClientMessageType_AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_SUBSCRIBE:
			updateSubscription(c, msg.GetRunSubscription().GetRunId(), true)

		case domainpb.AgentManagerWsClientMessageType_AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_UNSUBSCRIBE:
			updateSubscription(c, msg.GetRunSubscription().GetRunId(), false)

		case domainpb.AgentManagerWsClientMessageType_AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_SUBSCRIBE_ALL:
			updateAllEventsSubscription(c, true)
			log.Printf("[ws] Client subscribed to all events")

		case domainpb.AgentManagerWsClientMessageType_AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_UNSUBSCRIBE_ALL:
			updateAllEventsSubscription(c, false)
			log.Printf("[ws] Client unsubscribed from all events")
		}
	}
}

func buildPongPayload() []byte {
	pong := &domainpb.AgentManagerWsMessage{
		Type: domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_PONG,
		Payload: &domainpb.AgentManagerWsMessage_Pong{
			Pong: &domainpb.WsPong{
				Timestamp: timestamppb.New(time.Now().UTC()),
			},
		},
	}
	data, _ := protoconv.MarshalJSON(pong)
	return data
}

func updateSubscription(c *WebSocketClient, runID string, subscribe bool) {
	if runID == "" {
		return
	}
	if _, err := uuid.Parse(runID); err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if subscribe {
		c.subscriptions[runID] = true
		log.Printf("[ws] Client subscribed to run: %s", runID)
		return
	}
	delete(c.subscriptions, runID)
	log.Printf("[ws] Client unsubscribed from run: %s", runID)
}

func updateAllEventsSubscription(c *WebSocketClient, subscribe bool) {
	c.mu.Lock()
	c.allEvents = subscribe
	c.mu.Unlock()
}

func (c *WebSocketClient) shouldReceive(message *domainpb.AgentManagerWsMessage) bool {
	runID := message.GetRunId()
	if runID == "" {
		return true
	}

	// Status updates are small list-level invalidation messages. They are
	// intentionally global so the UI can keep run lists fresh without receiving
	// every run event body.
	if message.Type == domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS {
		return true
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.allEvents || c.subscriptions[runID]
}

// writePump handles outgoing messages to a client
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub closed the channel
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send each message as its own WebSocket frame to preserve JSON validity
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

			// Drain any queued messages, each as a separate frame
			n := len(c.send)
			for i := 0; i < n; i++ {
				if err := c.conn.WriteMessage(websocket.TextMessage, <-c.send); err != nil {
					return
				}
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
