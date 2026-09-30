package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

// streamEvents connects to WebSocket and streams events for a specific run.
func (a *App) streamEvents(runID string) error {
	// Resolve through APIClient even when no HTTP request has initialized the
	// underlying client yet (for example, a first-command --follow).
	baseURL := strings.TrimRight(a.core.APIClient.BaseURL(), "/")
	wsURL, err := httpToWSURL(baseURL + "/api/v1/ws")
	if err != nil {
		return fmt.Errorf("failed to build WebSocket URL: %w", err)
	}

	fmt.Printf("Connecting to %s...\n", wsURL)

	// Reuse the shared authentication and request-provenance hooks for the
	// upgrade. Gorilla dials directly, outside the normal HTTP transport.
	handshake, err := http.NewRequest(http.MethodGet, wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to build WebSocket request: %w", err)
	}
	a.core.HTTPClient.ApplyRequestHeaders(handshake)
	for name, value := range a.core.APIClient.AuthHeaders() {
		handshake.Header.Set(name, value)
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, handshake.Header)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	defer conn.Close()

	// Subscribe to the specific run
	subscribeMsg, err := protojson.Marshal(&domainpb.AgentManagerWsClientMessage{
		Type: domainpb.AgentManagerWsClientMessageType_AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_SUBSCRIBE,
		Payload: &domainpb.AgentManagerWsClientMessage_RunSubscription{
			RunSubscription: &domainpb.RunSubscription{RunId: runID},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to encode subscription: %w", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, subscribeMsg); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	fmt.Printf("Streaming events for run %s (Ctrl+C to stop)...\n\n", runID)

	// Handle interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	// Channel to receive messages
	messages := make(chan *domainpb.AgentManagerWsMessage)
	errChan := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	readerDone := make(chan struct{})

	// Read messages in a goroutine
	go func() {
		defer close(readerDone)
		errChan <- readWSMessages(ctx, conn.ReadMessage, messages)
	}()
	defer func() {
		cancel()
		_ = conn.Close() // Unblock a socket read as well as message delivery.
		<-readerDone
	}()

	// Process messages until interrupt or error
	for {
		select {
		case <-interrupt:
			fmt.Println("\nDisconnecting...")
			// Send close message
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
			return nil

		case err := <-errChan:
			return err

		case msg := <-messages:
			if err := a.handleWSMessage(msg, runID); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
		}
	}
}

// readWSMessages preserves protobuf enum, oneof and int64 semantics. Delivery
// must be cancellable because the terminal may stop consuming mid-batch.
func readWSMessages(ctx context.Context, read func() (int, []byte, error), messages chan<- *domainpb.AgentManagerWsMessage) error {
	for {
		_, data, err := read()
		if err != nil {
			if ctx.Err() != nil || websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				return nil
			}
			return fmt.Errorf("connection closed: %w", err)
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var msg domainpb.AgentManagerWsMessage
			if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal([]byte(line), &msg); err != nil {
				return fmt.Errorf("failed to decode WebSocket message: %w", err)
			}
			select {
			case messages <- &msg:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

// handleWSMessage processes a WebSocket message and prints it.
func (a *App) handleWSMessage(msg *domainpb.AgentManagerWsMessage, targetRunID string) error {
	// Filter to only show messages for our run (or general messages)
	if msg.GetRunId() != "" && msg.GetRunId() != targetRunID {
		return nil
	}

	timestamp := time.Now().Format("15:04:05")

	switch msg.GetType() {
	case domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_CONNECTED:
		// Already printed connection message
		return nil

	case domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_PONG:
		// Ignore pong messages
		return nil

	case domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT:
		payload := msg.GetRunEvent()
		if payload == nil {
			return fmt.Errorf("run_event message is missing runEvent")
		}
		if payload.RunId != "" && payload.RunId != targetRunID {
			return nil
		}
		dataStr := runEventDataString(payload)
		if len(dataStr) > 80 {
			dataStr = dataStr[:77] + "..."
		}
		fmt.Printf("[%s] EVENT #%d %-12s %s\n", timestamp, payload.Sequence, formatEnumValue(payload.EventType, "RUN_EVENT_TYPE_", "_"), dataStr)

	case domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_PROGRESS:
		payload := msg.GetRunProgress()
		if payload == nil {
			return fmt.Errorf("run_progress message is missing runProgress")
		}
		fmt.Printf("[%s] PROGRESS %d%% | Phase: %s | %s\n", timestamp, payload.PercentComplete, formatEnumValue(payload.Phase, "RUN_PHASE_", "_"), payload.CurrentAction)

	case domainpb.AgentManagerWsMessageType_AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS:
		payload := msg.GetRunStatus()
		if payload == nil {
			return fmt.Errorf("run_status message is missing runStatus")
		}
		if payload.RunId != "" && payload.RunId != targetRunID {
			return nil
		}
		fmt.Printf("[%s] STATUS: %s\n", timestamp, formatEnumValue(payload.Status, "RUN_STATUS_", "_"))

	default:
		// Print unknown message types as JSON for debugging
		payloadStr := marshalProtoJSON(msg)
		if len(payloadStr) > 100 {
			payloadStr = payloadStr[:97] + "..."
		}
		fmt.Printf("[%s] %s: %s\n", timestamp, msg.GetType(), payloadStr)
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
