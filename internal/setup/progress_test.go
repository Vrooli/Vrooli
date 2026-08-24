package setup

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSetupPhaseCatalogIsStableAndOrdered(t *testing.T) {
	if len(setupPhases) != 15 {
		t.Fatalf("phase count = %d, want 15", len(setupPhases))
	}
	seen := map[SetupPhase]bool{}
	for i, phase := range setupPhases {
		if phase.ID == "" || phase.Label == "" {
			t.Fatalf("phase %d is incomplete: %+v", i, phase)
		}
		if seen[phase.ID] {
			t.Fatalf("duplicate phase %q", phase.ID)
		}
		seen[phase.ID] = true
	}
	if setupPhases[0].ID != PhaseValidation || setupPhases[len(setupPhases)-1].ID != PhaseCompletion {
		t.Fatalf("catalog boundaries changed: %+v", setupPhases)
	}
}

func TestRenderSetupProgressPlainOutputIsStable(t *testing.T) {
	got := renderSetupProgress(ProgressEvent{Kind: EventPhaseStarted, Phase: PhaseBootstrap, PhaseLabel: "Bootstrap tools", PhaseIndex: 5, PhaseCount: 15})
	want := "SETUP  · 5/15 · Bootstrap tools\n"
	if got != want {
		t.Fatalf("rendered phase = %q, want %q", got, want)
	}
	heartbeat := renderSetupProgress(ProgressEvent{Kind: EventHeartbeat, Operation: "Checking bootstrap tools", Elapsed: 11 * time.Second})
	if !strings.Contains(heartbeat, "still working: Checking bootstrap tools") || !strings.Contains(heartbeat, "11s") {
		t.Fatalf("heartbeat = %q", heartbeat)
	}
}

func TestProgressWriterJSONIsNewlineDelimitedAndDoesNotExposeErrors(t *testing.T) {
	var out bytes.Buffer
	sink := &writerSink{w: &out, json: true}
	sink.Publish(ProgressEvent{Kind: EventPhaseFailed, Phase: PhaseRequirements, At: time.Unix(1, 0)})
	line := strings.TrimSpace(out.String())
	if strings.Contains(line, "password") || strings.Contains(line, "secret") {
		t.Fatalf("sensitive text leaked: %q", line)
	}
	if !strings.HasSuffix(out.String(), "\n") {
		t.Fatalf("structured output is not newline-delimited: %q", out.String())
	}
}

func TestProgressCompletionStopsHeartbeats(t *testing.T) {
	var out bytes.Buffer
	p := newProgressCoordinator(&out, progressOptions{FirstHeartbeat: time.Hour, HeartbeatEvery: time.Hour})
	p.Start()
	p.StartPhase(PhaseValidation)
	p.Operation("checking host")
	p.CompletePhase()
	p.Finish(nil)
	text := out.String()
	if !strings.Contains(text, "Starting project setup") || !strings.Contains(text, "Complete") {
		t.Fatalf("output = %q", text)
	}
	if strings.Contains(text, "still working") {
		t.Fatalf("unexpected heartbeat after completion: %q", text)
	}
}

func TestProgressStateIsAtomicAndTerminal(t *testing.T) {
	path := t.TempDir() + "/state/active-setup.json"
	var out bytes.Buffer
	p := newProgressCoordinator(&out, progressOptions{StatePath: path, FirstHeartbeat: time.Hour, HeartbeatEvery: time.Hour})
	p.Start()
	p.StartPhase(PhaseProject)
	p.Operation("loading project")
	p.CompletePhase()
	p.Finish(nil)
	state, err := readActiveSetupState(path)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if state.Version != "v1" || state.Status != "completed" || state.Phase != PhaseProject {
		t.Fatalf("state = %+v", state)
	}
	if strings.Contains(out.String(), path) {
		t.Fatalf("state path leaked to output: %q", out.String())
	}
}

func TestHeartbeatDueUsesThresholdThenInterval(t *testing.T) {
	started := time.Unix(100, 0)
	if heartbeatDue(started.Add(9*time.Second), started, time.Time{}, 10*time.Second, 30*time.Second) {
		t.Fatal("heartbeat fired before threshold")
	}
	first := started.Add(10 * time.Second)
	if !heartbeatDue(first, started, time.Time{}, 10*time.Second, 30*time.Second) {
		t.Fatal("heartbeat did not fire at threshold")
	}
	if heartbeatDue(first.Add(29*time.Second), started, first, 10*time.Second, 30*time.Second) {
		t.Fatal("heartbeat fired before interval")
	}
	if !heartbeatDue(first.Add(30*time.Second), started, first, 10*time.Second, 30*time.Second) {
		t.Fatal("heartbeat did not fire at interval")
	}
}

func TestProgressSinkPanicIsNonFatal(t *testing.T) {
	p := newProgressCoordinator(nil, progressOptions{FirstHeartbeat: time.Hour, HeartbeatEvery: time.Hour})
	p.sink = progressSinkFunc(func(ProgressEvent) { panic("broken sink") })
	p.Start()
	p.StartPhase(PhaseValidation)
	p.Operation("safe operation")
	p.CompletePhase()
	p.Finish(nil)
}

func TestProgressFormatEnvironmentSelectsStructuredAndQuietModes(t *testing.T) {
	t.Setenv("VROOLI_SETUP_PROGRESS_FORMAT", "json")
	var structured bytes.Buffer
	p := newProgressCoordinator(&structured, progressOptions{})
	p.Start()
	if !strings.HasPrefix(strings.TrimSpace(structured.String()), "{") || strings.Contains(structured.String(), "\x1b[") {
		t.Fatalf("structured output = %q", structured.String())
	}

	t.Setenv("VROOLI_SETUP_PROGRESS_FORMAT", "quiet")
	var quiet bytes.Buffer
	q := newProgressCoordinator(&quiet, progressOptions{})
	q.Start()
	if quiet.Len() != 0 {
		t.Fatalf("quiet output = %q", quiet.String())
	}
}

func TestActiveSetupStateRequiresProcessIdentityBeforeStaleLabel(t *testing.T) {
	path := t.TempDir() + "/active.json"
	state := activeSetupState{Version: "v1", RunID: "old", Status: "running", PID: 999999, Host: "this-host-is-not-real", UpdatedAt: time.Now().Add(-10 * time.Minute), PhaseLabel: "Apply host requirements"}
	b, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	renderActiveSetupState(&out, path, time.Now())
	if !strings.Contains(out.String(), "possibly stale") {
		t.Fatalf("diagnosis = %q", out.String())
	}
}

func TestProgressRedactsSensitiveOperationValues(t *testing.T) {
	var out bytes.Buffer
	sink := &writerSink{w: &out}
	sink.Publish(ProgressEvent{Kind: EventOperationChanged, Operation: "configure token=super-secret password:abc123"})
	text := out.String()
	if strings.Contains(text, "super-secret") || strings.Contains(text, "abc123") || !strings.Contains(text, "[redacted]") {
		t.Fatalf("redacted output = %q", text)
	}
}

type progressSinkFunc func(ProgressEvent)

func (f progressSinkFunc) Publish(ev ProgressEvent) { f(ev) }
