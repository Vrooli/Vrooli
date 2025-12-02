package execute

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"test-genie/cli/execute/report"
)

// SSE event types (must match server)
const (
	SSEEventPhaseStart  = "phase_start"
	SSEEventPhaseEnd    = "phase_end"
	SSEEventProgress    = "progress"
	SSEEventObservation = "observation"
	SSEEventComplete    = "complete"
	SSEEventError       = "error"
	SSEEventHeartbeat   = "heartbeat"
)

// SSEPhaseEndEvent matches the server's PhaseEndEvent
type SSEPhaseEndEvent struct {
	Phase    string  `json:"phase"`
	Status   string  `json:"status"`
	Duration float64 `json:"durationSeconds"`
	Error    string  `json:"error,omitempty"`
}

// SSECompleteEvent is the final event with full results
type SSECompleteEvent struct {
	Success       bool         `json:"success"`
	ExecutionID   string       `json:"executionId"`
	PresetUsed    string       `json:"presetUsed"`
	StartedAt     string       `json:"startedAt"`
	CompletedAt   string       `json:"completedAt"`
	PhaseSummary  PhaseSummary `json:"phaseSummary"`
	Phases        []Phase      `json:"phases"`
	TotalDuration float64      `json:"totalDuration"`
}

// SSEErrorEvent represents an error from the stream
type SSEErrorEvent struct {
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// RunWithSSE executes the suite using SSE streaming for real-time output
func (c *Client) RunWithSSE(req Request, printer *report.Printer, phaseNames []string) (Response, error) {
	baseURL := c.resolveBaseURL()
	if baseURL == "" {
		return Response{}, fmt.Errorf("api base URL not configured")
	}

	// Marshal request body
	payload, err := json.Marshal(req)
	if err != nil {
		return Response{}, fmt.Errorf("encode request: %w", err)
	}

	// Create request with extended timeout context
	ctx, cancel := context.WithTimeout(context.Background(), defaultExecutionTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/executions/stream", bytes.NewReader(payload))
	if err != nil {
		return Response{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	// Use a client without timeout (context handles it)
	streamClient := &http.Client{}
	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return Response{}, fmt.Errorf("api error (%d): %s", resp.StatusCode, extractErrorMessage(body))
	}

	// Process SSE stream
	return c.processSSEStream(resp.Body, printer, phaseNames)
}

func (c *Client) processSSEStream(r io.Reader, printer *report.Printer, phaseNames []string) (Response, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 1MB max line size

	var currentEvent string
	var dataBuffer strings.Builder
	var result Response
	var completedPhases []Phase

	// Track phases for progress display
	startTime := time.Now()
	color := report.NewColor(os.Stdout)
	_ = phaseNames // Available for future use

	for scanner.Scan() {
		line := scanner.Text()

		// Parse SSE format
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			dataBuffer.WriteString(strings.TrimPrefix(line, "data: "))
			continue
		}

		// Empty line marks end of event
		if line == "" && currentEvent != "" && dataBuffer.Len() > 0 {
			data := dataBuffer.String()
			dataBuffer.Reset()

			switch currentEvent {
			case SSEEventPhaseStart:
				var event struct {
					Phase string `json:"phase"`
					Index int    `json:"index"`
					Total int    `json:"total"`
				}
				if err := json.Unmarshal([]byte(data), &event); err == nil {
					elapsed := time.Since(startTime).Truncate(time.Second)
					// Print phase start marker
					fmt.Fprintf(os.Stdout, "%s\n", color.Cyan(fmt.Sprintf(
						"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━",
					)))
					fmt.Fprintf(os.Stdout, "%s Running phase %s [%d/%d] • elapsed %s\n",
						color.Yellow("⟳"),
						color.Bold(event.Phase),
						event.Index, event.Total,
						elapsed,
					)
				}

			case SSEEventPhaseEnd:
				var event SSEPhaseEndEvent
				if err := json.Unmarshal([]byte(data), &event); err == nil {
					phase := Phase{
						Name:            event.Phase,
						Status:          event.Status,
						DurationSeconds: event.Duration,
						Error:           event.Error,
					}
					completedPhases = append(completedPhases, phase)

					// Print phase completion
					statusIcon := color.Green("✅")
					if event.Status != "passed" {
						statusIcon = color.Red("❌")
					}
					fmt.Fprintf(os.Stdout, "%s Phase %s %s • %.1fs\n",
						statusIcon,
						color.Bold(event.Phase),
						event.Status,
						event.Duration,
					)
					if event.Error != "" {
						fmt.Fprintf(os.Stdout, "   %s %s\n", color.Red("Error:"), event.Error)
					}
				}

			case SSEEventProgress:
				var event struct {
					Phase   string  `json:"phase"`
					Elapsed float64 `json:"elapsedSeconds"`
					Message string  `json:"message"`
				}
				if err := json.Unmarshal([]byte(data), &event); err == nil && event.Message != "" {
					fmt.Fprintf(os.Stdout, "   %s %s\n", color.Cyan("→"), event.Message)
				}

			case SSEEventObservation:
				var event struct {
					Message string `json:"message"`
				}
				if err := json.Unmarshal([]byte(data), &event); err == nil {
					fmt.Fprintf(os.Stdout, "   %s %s\n", color.Cyan("•"), event.Message)
				}

			case SSEEventComplete:
				var event SSECompleteEvent
				if err := json.Unmarshal([]byte(data), &event); err == nil {
					result = Response{
						Success:      event.Success,
						ExecutionID:  event.ExecutionID,
						PresetUsed:   event.PresetUsed,
						StartedAt:    event.StartedAt,
						CompletedAt:  event.CompletedAt,
						PhaseSummary: event.PhaseSummary,
						Phases:       event.Phases,
					}
					// Final separator
					fmt.Fprintln(os.Stdout, color.Cyan("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
				}

			case SSEEventError:
				var event SSEErrorEvent
				if err := json.Unmarshal([]byte(data), &event); err == nil {
					return Response{}, fmt.Errorf("execution failed: %s", event.Message)
				}

			case SSEEventHeartbeat:
				// Heartbeats keep the connection alive, no output needed
			}

			currentEvent = ""
		}
	}

	if err := scanner.Err(); err != nil {
		return Response{}, fmt.Errorf("stream read error: %w", err)
	}

	// If we didn't get a complete event, build response from collected phases
	if result.Phases == nil && len(completedPhases) > 0 {
		result.Phases = completedPhases
		passed := 0
		failed := 0
		var totalDuration float64
		for _, p := range completedPhases {
			totalDuration += p.DurationSeconds
			if p.Status == "passed" {
				passed++
			} else {
				failed++
			}
		}
		result.PhaseSummary = PhaseSummary{
			Total:           len(completedPhases),
			Passed:          passed,
			Failed:          failed,
			DurationSeconds: int(totalDuration),
		}
		result.Success = failed == 0
	}

	return result, nil
}
