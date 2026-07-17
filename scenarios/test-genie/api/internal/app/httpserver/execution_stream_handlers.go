package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/runmanager"
	"test-genie/internal/shared"
)

// SSE event types for streaming execution progress. This is the browser-facing
// streaming REST exception (the test-genie UI's live-progress view): a thin
// gateway that starts a run on the run manager and proxies its canonical event
// stream as Server-Sent Events. All durability/decoupling lives in the run
// manager; this handler only adapts the transport, so a browser disconnect
// detaches the viewer without aborting the run.
const (
	SSEEventPhaseStart  = "phase_start"
	SSEEventPhaseEnd    = "phase_end"
	SSEEventProgress    = "progress"
	SSEEventObservation = "observation"
	SSEEventComplete    = "complete"
	SSEEventError       = "error"
	SSEEventHeartbeat   = "heartbeat"
)

// SSEEvent represents a Server-Sent Event.
type SSEEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// PhaseStartEvent is sent when a phase begins execution.
type PhaseStartEvent struct {
	Phase     string `json:"phase"`
	Index     int    `json:"index"`
	Total     int    `json:"total"`
	Timestamp string `json:"timestamp"`
}

// PhaseEndEvent is sent when a phase completes.
type PhaseEndEvent struct {
	Phase    string `json:"phase"`
	Status   string `json:"status"`
	Duration int    `json:"durationSeconds"`
	Error    string `json:"error,omitempty"`
}

func (s *Server) handleExecuteSuiteStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	input, err := decodeSuiteExecutionInput(r)
	if err != nil {
		s.writeSSEError(w, flusher, err.Error())
		return
	}
	if s.runManager == nil {
		s.writeSSEError(w, flusher, "execution service unavailable")
		return
	}
	if runID := s.runManager.CoalescedRunID(input.Request); runID != "" {
		replay, ch, followErr := s.runManager.Follow(r.Context(), input.Request.ScenarioName, runID)
		if followErr != nil {
			s.writeSSEError(w, flusher, followErr.Error())
			return
		}
		startTime := time.Now()
		for _, ev := range replay {
			s.writeCanonicalSSE(w, flusher, ev, startTime)
		}
		for ev := range ch {
			s.writeCanonicalSSE(w, flusher, ev, startTime)
		}
		return
	}
	caller := admissionCaller(r)
	releasePreview, err := s.runManager.TryAcquirePreviewFor(caller)
	if err != nil {
		w.Header().Set("Retry-After", "5")
		s.writeSSEError(w, flusher, err.Error())
		return
	}
	defer releasePreview()

	// Synchronous plan validation + ETA. A malformed request (bad preset/phase)
	// is reported as an SSE error before the run starts.
	preview, eta, err := s.previewExecutionPlan(r.Context(), input.Request)
	if err != nil {
		s.writeSSEError(w, flusher, err.Error())
		return
	}
	applyPreviewPhaseSelection(&input, preview)

	res, err := s.runManager.Start(runmanager.StartOptions{Input: input, Caller: caller, EstimatedTotalSeconds: eta})
	if err != nil {
		var saturated *runmanager.SaturatedError
		if errors.As(err, &saturated) {
			w.Header().Set("Retry-After", "5")
		}
		s.writeSSEError(w, flusher, err.Error())
		return
	}
	runID := res.RunID

	replay, ch, err := s.runManager.Follow(r.Context(), input.Request.ScenarioName, runID)
	if err != nil {
		s.writeSSEError(w, flusher, err.Error())
		return
	}

	startTime := time.Now()
	for _, ev := range replay {
		s.writeCanonicalSSE(w, flusher, ev, startTime)
	}
	for ev := range ch {
		s.writeCanonicalSSE(w, flusher, ev, startTime)
	}
}

// previewExecutionPlan resolves the summed plan estimate and surfaces plan
// validation errors. A non-fatal preview error yields ETA 0 and no mutation.
func (s *Server) previewExecutionPlan(ctx context.Context, req orchestrator.SuiteExecutionRequest) (*execution.ExecutionPlanPreview, int, error) {
	if s.executionPlanner == nil {
		return nil, 0, nil
	}
	preview, err := s.executionPlanner.Preview(ctx, req)
	if err != nil {
		var vErr shared.ValidationError
		if errors.As(err, &vErr) {
			return nil, 0, vErr
		}
		return nil, 0, nil
	}
	if preview == nil {
		return nil, 0, nil
	}
	return preview, preview.Summary.EstimatedDurationSeconds, nil
}

func applyPreviewPhaseSelection(input *execution.SuiteExecutionInput, preview *execution.ExecutionPlanPreview) {
	if input == nil || preview == nil || preview.Profile == nil || len(input.Request.Phases) > 0 {
		return
	}
	if len(preview.Phases) == 0 {
		return
	}
	names := make([]string, 0, len(preview.Phases))
	for _, phase := range preview.Phases {
		if phase.Name != "" {
			names = append(names, phase.Name)
		}
	}
	if len(names) > 0 {
		input.Request.Phases = names
	}
}

// writeCanonicalSSE maps one canonical run event onto the browser's SSE wire
// vocabulary. run_started is internal-only (the browser already knows the run
// it requested) and is not forwarded.
func (s *Server) writeCanonicalSSE(w http.ResponseWriter, flusher http.Flusher, ev runmanager.Event, startTime time.Time) {
	switch ev.Kind {
	case runmanager.EventPhaseStarted:
		s.writeSSE(w, flusher, SSEEvent{Event: SSEEventPhaseStart, Data: PhaseStartEvent{
			Phase: ev.Phase, Index: ev.PhaseIndex, Total: ev.PhaseTotal, Timestamp: time.Now().Format(time.RFC3339),
		}})
	case runmanager.EventPhaseProgress:
		s.writeSSE(w, flusher, SSEEvent{Event: SSEEventProgress, Data: map[string]interface{}{
			"phase": ev.Phase, "message": ev.Message, "elapsedSeconds": ev.ElapsedSeconds,
		}})
	case runmanager.EventPhaseHeartbeat:
		s.writeSSE(w, flusher, SSEEvent{Event: SSEEventHeartbeat, Data: map[string]interface{}{
			"phase": ev.Phase, "elapsedSeconds": ev.ElapsedSeconds, "quietSeconds": ev.QuietSeconds,
			"timestamp": time.Now().Format(time.RFC3339),
		}})
	case runmanager.EventPhaseCompleted, runmanager.EventPhaseFailed:
		s.writeSSE(w, flusher, SSEEvent{Event: SSEEventPhaseEnd, Data: PhaseEndEvent{
			Phase: ev.Phase, Status: ev.Status, Duration: ev.DurationSeconds, Error: ev.Error,
		}})
	case runmanager.EventRunCompleted:
		if ev.Error != "" {
			s.writeSSEError(w, flusher, ev.Error)
			return
		}
		data := map[string]interface{}{
			"success":       ev.Success,
			"verdict":       ev.Verdict,
			"runId":         ev.RunID,
			"artifactDir":   ev.ArtifactDir,
			"totalDuration": time.Since(startTime).Seconds(),
		}
		s.writeSSE(w, flusher, SSEEvent{Event: SSEEventComplete, Data: data})
	}
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
