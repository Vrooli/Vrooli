package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/runmanager"

	"github.com/gorilla/mux"
)

// fakeStreamExecutor emits a single phase_start, then stays quiet for quietFor
// (no further events) before returning, so the quiet-phase heartbeat must fire.
type fakeStreamExecutor struct {
	phase    string
	quietFor time.Duration
	result   *orchestrator.SuiteExecutionResult
}

func (f *fakeStreamExecutor) Execute(ctx context.Context, input execution.SuiteExecutionInput) (*orchestrator.SuiteExecutionResult, error) {
	return f.result, nil
}

func (f *fakeStreamExecutor) ExecuteWithEvents(ctx context.Context, input execution.SuiteExecutionInput, emit orchestrator.ExecutionEventCallback) (*orchestrator.SuiteExecutionResult, error) {
	emit(orchestrator.ExecutionEvent{
		Type:       orchestrator.EventPhaseStart,
		Phase:      f.phase,
		PhaseIndex: 1,
		PhaseTotal: 1,
		Timestamp:  time.Now(),
	})
	// Stay quiet so the heartbeat goroutine observes a quiet active phase.
	select {
	case <-time.After(f.quietFor):
	case <-ctx.Done():
	}
	return f.result, nil
}

// parseSSEFrames splits an SSE body into (event, data) pairs.
func parseSSEFrames(t *testing.T, body string) []struct{ Event, Data string } {
	t.Helper()
	var frames []struct{ Event, Data string }
	var cur struct{ Event, Data string }
	for _, line := range strings.Split(body, "\n") {
		switch {
		case strings.HasPrefix(line, "event: "):
			cur.Event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			cur.Data = strings.TrimPrefix(line, "data: ")
		case line == "" && cur.Event != "":
			frames = append(frames, cur)
			cur = struct{ Event, Data string }{}
		}
	}
	return frames
}

// TestStream_QuietPhaseEmitsPhaseAwareHeartbeat verifies that when a phase runs
// without emitting events for longer than the heartbeat interval, the stream
// emits a heartbeat carrying the active phase — the §9 "heartbeat fires after
// the interval with no output" case.
func TestStream_QuietPhaseEmitsPhaseAwareHeartbeat(t *testing.T) {
	t.Setenv("TEST_GENIE_HEARTBEAT_SECONDS", "5") // floor; keeps the test ~6s

	exec := &fakeStreamExecutor{
		phase:    "standards",
		quietFor: 6 * time.Second,
		result: &orchestrator.SuiteExecutionResult{
			ScenarioName: "demo",
			RunID:        "20251208-151044-abcd1234",
			ArtifactDir:  "/x/coverage/runs/20251208-151044-abcd1234",
			Success:      true,
			Verdict:      "PASS",
		},
	}
	srv := &Server{
		config:     Config{Port: "0", ServiceName: "Test Genie API"},
		router:     mux.NewRouter(),
		runManager: runmanager.New(exec, ""),
		logger:     log.New(io.Discard, "", 0),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/stream",
		strings.NewReader(`{"scenarioName":"demo"}`))
	w := httptest.NewRecorder()

	srv.handleExecuteSuiteStream(w, req)

	frames := parseSSEFrames(t, w.Body.String())

	var sawHeartbeat, sawComplete bool
	for _, f := range frames {
		switch f.Event {
		case SSEEventHeartbeat:
			sawHeartbeat = true
			var hb struct {
				Phase string `json:"phase"`
			}
			if err := json.Unmarshal([]byte(f.Data), &hb); err != nil {
				t.Fatalf("heartbeat data not JSON: %v", err)
			}
			if hb.Phase != "standards" {
				t.Errorf("heartbeat phase = %q, want standards", hb.Phase)
			}
		case SSEEventComplete:
			sawComplete = true
			var c struct {
				RunID       string `json:"runId"`
				ArtifactDir string `json:"artifactDir"`
			}
			if err := json.Unmarshal([]byte(f.Data), &c); err != nil {
				t.Fatalf("complete data not JSON: %v", err)
			}
			if c.RunID == "" || c.ArtifactDir == "" {
				t.Errorf("complete event missing runId/artifactDir: %+v", c)
			}
		}
	}
	if !sawHeartbeat {
		t.Error("expected a phase-aware heartbeat during the quiet phase")
	}
	if !sawComplete {
		t.Error("expected a complete event")
	}
}
