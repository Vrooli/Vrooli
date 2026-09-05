package agentmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gorilla/websocket"
)

type WebSocketEventSource struct {
	url    string
	dialer *websocket.Dialer
}

func NewWebSocketEventSource(baseURL string, dialer *websocket.Dialer) *WebSocketEventSource {
	wsURL, _ := WebSocketURL(baseURL)
	if dialer == nil {
		dialer = websocket.DefaultDialer
	}
	return &WebSocketEventSource{url: wsURL, dialer: dialer}
}

func (s *WebSocketEventSource) StreamRunEvents(ctx context.Context, runID string, emit func(ActivityEvent) error) error {
	if strings.TrimSpace(s.url) == "" {
		return ErrUnavailable
	}
	conn, _, err := s.dialer.DialContext(ctx, s.url, nil)
	if err != nil {
		return fmt.Errorf("%w: websocket connect: %v", ErrUnavailable, err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{
		"type": "subscribe",
		"payload": map[string]string{
			"runId": runID,
		},
	}); err != nil {
		return fmt.Errorf("subscribe to agent run %s: %w", runID, err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read agent event: %w", err)
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			events, err := DecodeWebSocketLine([]byte(line), runID)
			if err != nil {
				continue
			}
			for _, ev := range events {
				if err := emit(ev); err != nil {
					return err
				}
				if ev.Done {
					return nil
				}
			}
		}
	}
}

type wsMessage struct {
	Type             string          `json:"type"`
	Payload          json.RawMessage `json:"payload"`
	RunID            string          `json:"runId,omitempty"`
	RunIDSnake       string          `json:"run_id,omitempty"`
	RunEvent         json.RawMessage `json:"run_event,omitempty"`
	RunEventCamel    json.RawMessage `json:"runEvent,omitempty"`
	RunStatus        json.RawMessage `json:"run_status,omitempty"`
	RunStatusCamel   json.RawMessage `json:"runStatus,omitempty"`
	RunProgress      json.RawMessage `json:"run_progress,omitempty"`
	RunProgressCamel json.RawMessage `json:"runProgress,omitempty"`
}

type runEventPayload struct {
	ID             string          `json:"id"`
	RunID          string          `json:"runId"`
	RunIDSnake     string          `json:"run_id"`
	Sequence       int64           `json:"sequence"`
	EventType      string          `json:"eventType"`
	EventTypeSnake string          `json:"event_type"`
	Data           json.RawMessage `json:"data,omitempty"`
	Log            json.RawMessage `json:"log,omitempty"`
	Message        json.RawMessage `json:"message,omitempty"`
	ToolCall       json.RawMessage `json:"tool_call,omitempty"`
	ToolResult     json.RawMessage `json:"tool_result,omitempty"`
	Status         json.RawMessage `json:"status,omitempty"`
	Progress       json.RawMessage `json:"progress,omitempty"`
	Error          json.RawMessage `json:"error,omitempty"`
}

type runProgressPayload struct {
	RunID                string `json:"runId"`
	RunIDSnake           string `json:"run_id"`
	Phase                string `json:"phase"`
	PercentComplete      int    `json:"percentComplete"`
	PercentCompleteSnake int    `json:"percent_complete"`
	CurrentAction        string `json:"currentAction"`
	CurrentActionSnake   string `json:"current_action"`
}

type runStatusPayload struct {
	ID         string `json:"id"`
	RunID      string `json:"runId"`
	RunIDSnake string `json:"run_id"`
	Status     string `json:"status"`
}

func DecodeWebSocketLine(data []byte, targetRunID string) ([]ActivityEvent, error) {
	var msg wsMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	msgRunID := firstNonEmpty(msg.RunID, msg.RunIDSnake)
	if msgRunID != "" && targetRunID != "" && msgRunID != targetRunID {
		return nil, nil
	}
	switch normalizeWSType(msg.Type) {
	case "connected", "pong":
		return nil, nil
	case "run_event":
		var payload runEventPayload
		if err := json.Unmarshal(firstRaw(msg.Payload, msg.RunEvent, msg.RunEventCamel), &payload); err != nil {
			return nil, err
		}
		runID := firstNonEmpty(payload.RunID, payload.RunIDSnake)
		if runID != "" && targetRunID != "" && runID != targetRunID {
			return nil, nil
		}
		return []ActivityEvent{MapRunEvent(payload)}, nil
	case "run_progress":
		var payload runProgressPayload
		if err := json.Unmarshal(firstRaw(msg.Payload, msg.RunProgress, msg.RunProgressCamel), &payload); err != nil {
			return nil, err
		}
		return []ActivityEvent{{
			Kind:  EventKindProgress,
			RunID: firstNonEmpty(payload.RunID, payload.RunIDSnake, msgRunID),
			Text:  formatProgress(payload),
		}}, nil
	case "run_status":
		var payload runStatusPayload
		if err := json.Unmarshal(firstRaw(msg.Payload, msg.RunStatus, msg.RunStatusCamel), &payload); err != nil {
			return nil, err
		}
		status := normalizeRunStatus(firstNonEmpty(payload.Status, string(msg.Payload), string(msg.RunStatus), string(msg.RunStatusCamel)))
		return []ActivityEvent{{
			Kind:  statusKind(status),
			RunID: firstNonEmpty(payload.RunID, payload.RunIDSnake, payload.ID, msgRunID),
			Text:  "Agent run " + status,
			Done:  terminalStatus(status),
		}}, nil
	default:
		return []ActivityEvent{{
			Kind: EventKindLog,
			Text: strings.TrimSpace(msg.Type + ": " + string(msg.Payload)),
		}}, nil
	}
}

func MapRunEvent(payload runEventPayload) ActivityEvent {
	eventType := normalizeEventType(firstNonEmpty(payload.EventType, payload.EventTypeSnake))
	return ActivityEvent{
		Kind:     eventKind(eventType),
		RunID:    firstNonEmpty(payload.RunID, payload.RunIDSnake),
		Sequence: payload.Sequence,
		Text:     formatRunEvent(eventType, firstRaw(payload.Data, payload.Log, payload.Message, payload.ToolCall, payload.ToolResult, payload.Status, payload.Progress, payload.Error)),
		Done:     false,
	}
}

func formatRunEvent(eventType string, data json.RawMessage) string {
	var obj map[string]any
	if len(data) > 0 && json.Unmarshal(data, &obj) == nil {
		for _, key := range []string{"message", "content", "text", "currentAction", "status", "error"} {
			if value, ok := obj[key]; ok {
				return strings.TrimSpace(fmt.Sprintf("%s: %v", eventType, value))
			}
		}
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return eventType
	}
	return eventType + ": " + trimmed
}

func formatProgress(payload runProgressPayload) string {
	percent := payload.PercentComplete
	if percent == 0 {
		percent = payload.PercentCompleteSnake
	}
	action := firstNonEmpty(payload.CurrentAction, payload.CurrentActionSnake)
	parts := []string{fmt.Sprintf("Agent progress %d%%", percent)}
	if strings.TrimSpace(payload.Phase) != "" {
		parts = append(parts, "phase "+payload.Phase)
	}
	if action != "" {
		parts = append(parts, action)
	}
	return strings.Join(parts, " | ")
}

func eventKind(eventType string) EventKind {
	switch eventType {
	case "message":
		return EventKindMessage
	case "tool_call", "tool_result":
		return EventKindTool
	case "error":
		return EventKindError
	case "status":
		return EventKindStatus
	default:
		return EventKindLog
	}
}

func statusKind(status string) EventKind {
	if terminalStatus(status) {
		return EventKindDone
	}
	return EventKindStatus
}

func terminalStatus(status string) bool {
	switch normalizeRunStatus(status) {
	case "completed", "complete", "failed", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func normalizeEventType(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	trimmed = strings.TrimPrefix(trimmed, "run_event_type_")
	if trimmed == "" {
		return "log"
	}
	return trimmed
}

func normalizeWSType(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	trimmed = strings.TrimPrefix(trimmed, "agent_manager_ws_message_type_")
	return trimmed
}

func normalizeRunStatus(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	trimmed = strings.TrimPrefix(trimmed, "run_status_")
	return trimmed
}

func firstRaw(values ...json.RawMessage) json.RawMessage {
	for _, value := range values {
		if len(value) > 0 && strings.TrimSpace(string(value)) != "" {
			return value
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
