package executions

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"browser-automation-studio/cli/internal/appctx"
	"browser-automation-studio/cli/internal/output"

	"github.com/gorilla/websocket"
)

type heartbeatState struct {
	lastHeartbeat time.Time
	initialized   bool
	state         string
	lastStep      string
}

func runWatch(ctx *appctx.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("execution ID required")
	}
	executionID := args[0]

	fmt.Printf("Watching execution: %s\n", executionID)
	fmt.Println("Press Ctrl+C to stop watching")
	fmt.Println("")

	stop := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		streamExecution(ctx, executionID, stop)
	}()

	statusDone := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		pollExecution(ctx, executionID, stop, statusDone)
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sig:
		close(stop)
	case <-statusDone:
		close(stop)
	}

	wg.Wait()

	if err := printTimelineSummary(ctx, executionID); err != nil {
		fmt.Printf("\n%s\n", err.Error())
	}

	return nil
}

func streamExecution(ctx *appctx.Context, executionID string, stop <-chan struct{}) {
	wsURL := buildWebSocketURL(ctx.APIRoot(), executionID)
	if wsURL == "" {
		fmt.Println("[stream] WebSocket base unavailable")
		return
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		fmt.Printf("[stream] WebSocket error: %s\n", err.Error())
		return
	}
	defer conn.Close()

	state := &heartbeatState{lastHeartbeat: time.Now(), state: "init"}
	fmt.Printf("[stream] Connected to execution telemetry (%s)\n", executionID)

	heartbeatTicker := time.NewTicker(2 * time.Second)
	defer heartbeatTicker.Stop()

	go func() {
		for {
			select {
			case <-stop:
				return
			case <-heartbeatTicker.C:
				checkHeartbeat(state)
			}
		}
	}()

	for {
		select {
		case <-stop:
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1000, ""))
			return
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("[stream] Telemetry stream closed\n")
				return
			}
			handleStreamMessage(state, message)
		}
	}
}

func buildWebSocketURL(apiRoot, executionID string) string {
	apiRoot = strings.TrimRight(strings.TrimSpace(apiRoot), "/")
	if apiRoot == "" {
		return ""
	}
	parsed, err := url.Parse(apiRoot)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}

	wsScheme := "ws"
	if parsed.Scheme == "https" {
		wsScheme = "wss"
	}
	return fmt.Sprintf("%s://%s/ws?execution_id=%s", wsScheme, parsed.Host, executionID)
}

func handleStreamMessage(state *heartbeatState, message []byte) {
	var payload map[string]any
	if err := json.Unmarshal(message, &payload); err != nil {
		text := strings.TrimSpace(string(message))
		if text != "" {
			fmt.Printf("[stream] %s\n", text)
		}
		return
	}

	payloadType, _ := payload["type"].(string)
	if payloadType == "event" {
		data, _ := payload["data"].(map[string]any)
		if data == nil {
			return
		}
		eventType, _ := data["type"].(string)
		label := labelForEvent(data)
		switch eventType {
		case "execution.started":
			fmt.Println("[stream] execution started")
		case "execution.completed":
			fmt.Println("[stream] execution completed")
		case "execution.failed":
			fmt.Printf("[stream] execution failed%s\n", messageSuffix(data, "message"))
		case "execution.cancelled":
			fmt.Println("[stream] execution cancelled")
		case "step.started":
			fmt.Printf("[stream] %s started\n", label)
		case "step.completed":
			fmt.Printf("[stream] %s completed %s\n", label, fmtDuration(data["payload"]))
		case "step.failed":
			fmt.Printf("[stream] %s failed%s\n", label, messageSuffix(data, "message"))
		case "step.heartbeat":
			state.lastHeartbeat = time.Now()
			state.lastStep = label
			if !state.initialized {
				state.initialized = true
				if state.state == "awaiting_warn" || state.state == "awaiting_stalled" {
					fmt.Printf("[stream] heartbeat received (%s)\n", label)
				}
			} else if state.state == "delayed" || state.state == "stalled" {
				fmt.Printf("[stream] heartbeat recovered (%s)\n", label)
			}
			state.state = "healthy"
		case "step.screenshot":
			fmt.Printf("[stream] %s screenshot\n", label)
		case "step.telemetry":
			summarizeTelemetry(data["payload"], label)
		}
		return
	}

	if payloadType == "progress" {
		progress, _ := payload["progress"].(float64)
		step, _ := payload["current_step"].(string)
		label := strings.TrimSpace(step)
		if label != "" {
			fmt.Printf("[stream] progress %.0f%% - %s\n", progress, label)
			return
		}
		fmt.Printf("[stream] progress %.0f%%\n", progress)
		return
	}

	if payloadType == "completed" {
		fmt.Println("[stream] execution completed")
		return
	}

	if payloadType == "failed" {
		fmt.Printf("[stream] execution failed%s\n", messageSuffix(payload, "message"))
		return
	}
}

func fmtDuration(payload any) string {
	if payload == nil {
		return ""
	}
	payloadMap, ok := payload.(map[string]any)
	if !ok {
		return ""
	}
	if value, ok := payloadMap["duration_ms"].(float64); ok {
		return fmt.Sprintf("%dms", int(value))
	}
	if value, ok := payloadMap["durationMs"].(float64); ok {
		return fmt.Sprintf("%dms", int(value))
	}
	return ""
}

func labelForEvent(event map[string]any) string {
	if value, ok := event["step_node_id"].(string); ok && strings.TrimSpace(value) != "" {
		return value
	}
	if value, ok := event["step_type"].(string); ok && strings.TrimSpace(value) != "" {
		return value
	}
	if value, ok := event["step_index"].(float64); ok {
		return fmt.Sprintf("step %d", int(value)+1)
	}
	return "step"
}

func messageSuffix(payload map[string]any, key string) string {
	if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
		return " - " + value
	}
	return ""
}

func summarizeTelemetry(payload any, label string) {
	payloadMap, ok := payload.(map[string]any)
	if !ok {
		return
	}
	consoleLogs := intFromAny(payloadMap["console_logs"])
	networkEvents := intFromAny(payloadMap["network_events"])
	if consoleLogs == 0 && networkEvents == 0 {
		return
	}
	fmt.Printf("[stream] %s: console %d, network %d\n", label, consoleLogs, networkEvents)
}

func intFromAny(value any) int {
	switch typed := value.(type) {
	case []any:
		return len(typed)
	case float64:
		return int(typed)
	case int:
		return typed
	default:
		return 0
	}
}

func checkHeartbeat(state *heartbeatState) {
	delta := time.Since(state.lastHeartbeat)
	warn := 8 * time.Second
	stall := 15 * time.Second

	if !state.initialized {
		if delta >= stall && state.state != "awaiting_stalled" {
			state.state = "awaiting_stalled"
			fmt.Printf("[stream] WARN no heartbeat yet (%ds)\n", int(delta.Seconds()))
			return
		}
		if delta >= warn && state.state != "awaiting_warn" {
			state.state = "awaiting_warn"
			fmt.Printf("[stream] waiting for first heartbeat (%ds)\n", int(delta.Seconds()))
			return
		}
		return
	}

	if delta >= stall && state.state != "stalled" {
		state.state = "stalled"
		fmt.Printf("[stream] WARN heartbeat stalled (%ds)\n", int(delta.Seconds()))
		return
	}
	if delta >= warn && state.state != "delayed" {
		state.state = "delayed"
		fmt.Printf("[stream] heartbeat delayed (%ds)\n", int(delta.Seconds()))
		return
	}
	state.state = "healthy"
}

func pollExecution(ctx *appctx.Context, executionID string, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	for {
		select {
		case <-stop:
			fmt.Println("")
			return
		default:
		}
		detail, _, err := getExecution(ctx, executionID)
		if err != nil {
			fmt.Printf("\rStatus: unknown | Progress: ? | Step: ?")
			time.Sleep(2 * time.Second)
			continue
		}
		fmt.Printf("\rStatus: %s | Progress: %d%% | Step: %s", detail.Status, detail.Progress, detail.CurrentStep)
		if detail.Status == "completed" || detail.Status == "failed" {
			fmt.Println("")
			fmt.Printf("OK: Execution %s\n", detail.Status)
			return
		}
		time.Sleep(2 * time.Second)
	}
}

func printTimelineSummary(ctx *appctx.Context, executionID string) error {
	body, err := ctx.Core.APIClient.Get(ctx.APIPath("/executions/"+executionID+"/timeline"), nil)
	if err != nil {
		return fmt.Errorf("No timeline response received from API")
	}

	lines, err := output.SummarizeTimeline(body)
	if err != nil {
		return fmt.Errorf("Unable to parse execution timeline response")
	}
	if len(lines) == 0 {
		fmt.Println("")
		fmt.Println("No timeline artifacts available for this execution")
		return nil
	}

	fmt.Println("")
	fmt.Println("Timeline Summary")
	for _, line := range lines {
		fmt.Println(line)
	}
	return nil
}
