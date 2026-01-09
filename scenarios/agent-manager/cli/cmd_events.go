package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketMessage represents a message from the WebSocket server.
type WebSocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	RunID   string          `json:"runId,omitempty"`
}

// RunEventPayload represents a run event from the WebSocket.
type RunEventPayload struct {
	ID        string `json:"id"`
	RunID     string `json:"runId"`
	Sequence  int64  `json:"sequence"`
	EventType string `json:"eventType"`
	Timestamp string `json:"timestamp"`
	Data      any    `json:"data,omitempty"`
}

// RunProgressPayload represents a progress update from the WebSocket.
type RunProgressPayload struct {
	RunID           string `json:"runId"`
	Phase           string `json:"phase"`
	PercentComplete int    `json:"percentComplete"`
	CurrentAction   string `json:"currentAction"`
}

// RunStatusPayload represents a status change from the WebSocket.
type RunStatusPayload struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// streamEvents connects to WebSocket and streams events for a specific run.
func (a *App) streamEvents(runID string) error {
	// Get API base and convert to WebSocket URL
	baseURL := a.core.HTTPClient.BaseURL()
	wsURL, err := httpToWSURL(baseURL + "/api/v1/ws")
	if err != nil {
		return fmt.Errorf("failed to build WebSocket URL: %w", err)
	}

	fmt.Printf("Connecting to %s...\n", wsURL)

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	defer conn.Close()

	// Subscribe to the specific run
	subscribeMsg := map[string]any{
		"type": "subscribe",
		"payload": map[string]string{
			"runId": runID,
		},
	}
	if err := conn.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	fmt.Printf("Streaming events for run %s (Ctrl+C to stop)...\n\n", runID)

	// Handle interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	// Channel to receive messages
	messages := make(chan WebSocketMessage)
	errChan := make(chan error, 1)

	// Read messages in a goroutine
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					errChan <- fmt.Errorf("connection closed: %w", err)
				}
				close(messages)
				return
			}

			// Handle batched messages (separated by newlines)
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				var msg WebSocketMessage
				if err := json.Unmarshal([]byte(line), &msg); err != nil {
					continue // Skip malformed messages
				}
				messages <- msg
			}
		}
	}()

	// Process messages until interrupt or error
	for {
		select {
		case <-interrupt:
			fmt.Println("\nDisconnecting...")
			// Send close message
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				return nil // Ignore close errors
			}
			return nil

		case err := <-errChan:
			return err

		case msg, ok := <-messages:
			if !ok {
				return nil // Channel closed
			}
			if err := a.handleWSMessage(msg, runID); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
		}
	}
}

// handleWSMessage processes a WebSocket message and prints it.
func (a *App) handleWSMessage(msg WebSocketMessage, targetRunID string) error {
	// Filter to only show messages for our run (or general messages)
	if msg.RunID != "" && msg.RunID != targetRunID {
		return nil
	}

	timestamp := time.Now().Format("15:04:05")

	switch msg.Type {
	case "connected":
		// Already printed connection message
		return nil

	case "pong":
		// Ignore pong messages
		return nil

	case "run_event":
		var payload RunEventPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("failed to parse run_event: %w", err)
		}
		dataStr := ""
		if payload.Data != nil {
			dataBytes, _ := json.Marshal(payload.Data)
			dataStr = string(dataBytes)
			if len(dataStr) > 80 {
				dataStr = dataStr[:77] + "..."
			}
		}
		fmt.Printf("[%s] EVENT #%d %-12s %s\n", timestamp, payload.Sequence, payload.EventType, dataStr)

	case "run_progress":
		var payload RunProgressPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("failed to parse run_progress: %w", err)
		}
		fmt.Printf("[%s] PROGRESS %d%% | Phase: %s | %s\n", timestamp, payload.PercentComplete, payload.Phase, payload.CurrentAction)

	case "run_status":
		var payload RunStatusPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("failed to parse run_status: %w", err)
		}
		fmt.Printf("[%s] STATUS: %s\n", timestamp, payload.Status)
		// If run completed, we can exit
		if payload.Status == "completed" || payload.Status == "failed" || payload.Status == "cancelled" {
			fmt.Printf("\nRun %s\n", payload.Status)
			return nil
		}

	default:
		// Print unknown message types as JSON for debugging
		payloadStr := string(msg.Payload)
		if len(payloadStr) > 100 {
			payloadStr = payloadStr[:97] + "..."
		}
		fmt.Printf("[%s] %s: %s\n", timestamp, msg.Type, payloadStr)
	}

	return nil
}

// httpToWSURL converts an HTTP URL to a WebSocket URL.
func httpToWSURL(httpURL string) (string, error) {
	parsed, err := url.Parse(httpURL)
	if err != nil {
		return "", err
	}

	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	case "ws", "wss":
		// Already a WebSocket URL
	default:
		return "", fmt.Errorf("unsupported scheme: %s", parsed.Scheme)
	}

	return parsed.String(), nil
}
