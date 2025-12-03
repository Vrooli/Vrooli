package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
)

// SSE event types for streaming execution progress
const (
	SSEEventPhaseStart    = "phase_start"
	SSEEventPhaseEnd      = "phase_end"
	SSEEventProgress      = "progress"
	SSEEventObservation   = "observation"
	SSEEventComplete      = "complete"
	SSEEventError         = "error"
	SSEEventHeartbeat     = "heartbeat"
)

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// PhaseStartEvent is sent when a phase begins execution
type PhaseStartEvent struct {
	Phase     string `json:"phase"`
	Index     int    `json:"index"`
	Total     int    `json:"total"`
	Timestamp string `json:"timestamp"`
}

// PhaseEndEvent is sent when a phase completes
type PhaseEndEvent struct {
	Phase    string `json:"phase"`
	Status   string `json:"status"`
	Duration int    `json:"durationSeconds"`
	Error    string `json:"error,omitempty"`
}

// ProgressEvent provides periodic progress updates
type ProgressEvent struct {
	Phase   string  `json:"phase"`
	Elapsed float64 `json:"elapsedSeconds"`
	Message string  `json:"message,omitempty"`
}

// streamingExecutor wraps the execution service to emit SSE events
type streamingExecutor struct {
	svc         suiteExecutor
	eventWriter func(SSEEvent) error
}

func (s *Server) handleExecuteSuiteStream(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Check if the client supports flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	defer r.Body.Close()
	var payload suiteExecutionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeSSEError(w, flusher, "invalid JSON payload")
		return
	}

	scenario := strings.TrimSpace(payload.ScenarioName)
	if scenario == "" {
		s.writeSSEError(w, flusher, "scenarioName is required")
		return
	}

	execRequest := orchestrator.SuiteExecutionRequest{
		ScenarioName: scenario,
		Preset:       strings.TrimSpace(payload.Preset),
		Phases:       payload.Phases,
		Skip:         payload.Skip,
		FailFast:     payload.FailFast,
	}

	var suiteRequestID *uuid.UUID
	if id := strings.TrimSpace(payload.SuiteRequestID); id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			s.writeSSEError(w, flusher, "suiteRequestId must be a valid UUID")
			return
		}
		suiteRequestID = &parsed
	}

	if s.executionSvc == nil {
		s.writeSSEError(w, flusher, "execution service unavailable")
		return
	}

	// Create a context that respects client disconnection
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Start a heartbeat goroutine to keep the connection alive
	heartbeatDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.writeSSE(w, flusher, SSEEvent{
					Event: SSEEventHeartbeat,
					Data: map[string]interface{}{
						"timestamp": time.Now().Format(time.RFC3339),
					},
				})
			case <-ctx.Done():
				close(heartbeatDone)
				return
			}
		}
	}()

	// Execute the suite with real-time event streaming
	startTime := time.Now()
	result, err := s.executionSvc.ExecuteWithEvents(ctx, execution.SuiteExecutionInput{
		Request:        execRequest,
		SuiteRequestID: suiteRequestID,
	}, func(event orchestrator.ExecutionEvent) {
		// Convert orchestrator events to SSE events
		switch event.Type {
		case orchestrator.EventPhaseStart:
			s.writeSSE(w, flusher, SSEEvent{
				Event: SSEEventPhaseStart,
				Data: PhaseStartEvent{
					Phase:     event.Phase,
					Index:     event.PhaseIndex,
					Total:     event.PhaseTotal,
					Timestamp: event.Timestamp.Format(time.RFC3339),
				},
			})
		case orchestrator.EventPhaseEnd:
			s.writeSSE(w, flusher, SSEEvent{
				Event: SSEEventPhaseEnd,
				Data: PhaseEndEvent{
					Phase:    event.Phase,
					Status:   event.Status,
					Duration: event.DurationSeconds,
					Error:    event.Error,
				},
			})
		case orchestrator.EventObservation:
			s.writeSSE(w, flusher, SSEEvent{
				Event: SSEEventObservation,
				Data: map[string]interface{}{
					"phase":     event.Phase,
					"message":   event.Message,
					"timestamp": event.Timestamp.Format(time.RFC3339),
				},
			})
		case orchestrator.EventProgress:
			s.writeSSE(w, flusher, SSEEvent{
				Event: SSEEventProgress,
				Data: ProgressEvent{
					Phase:   event.Phase,
					Message: event.Message,
				},
			})
		case orchestrator.EventComplete:
			// Complete event is handled after the function returns
		}
	})

	// Stop heartbeat
	cancel()
	<-heartbeatDone

	if err != nil {
		s.writeSSEError(w, flusher, fmt.Sprintf("execution failed: %v", err))
		return
	}

	// Send completion event with full result
	s.writeSSE(w, flusher, SSEEvent{
		Event: SSEEventComplete,
		Data: map[string]interface{}{
			"success":       result.Success,
			"executionId":   result.ExecutionID,
			"presetUsed":    result.PresetUsed,
			"startedAt":     result.StartedAt,
			"completedAt":   result.CompletedAt,
			"phaseSummary":  result.PhaseSummary,
			"phases":        result.Phases,
			"totalDuration": time.Since(startTime).Seconds(),
		},
	})
}

func (s *Server) writeSSE(w http.ResponseWriter, flusher http.Flusher, event SSEEvent) {
	data, err := json.Marshal(event.Data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\n", event.Event)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func (s *Server) writeSSEError(w http.ResponseWriter, flusher http.Flusher, message string) {
	s.writeSSE(w, flusher, SSEEvent{
		Event: SSEEventError,
		Data: map[string]interface{}{
			"message":   message,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	})
}
