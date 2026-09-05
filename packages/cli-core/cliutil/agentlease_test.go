package cliutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type recordingLeaseRecorder struct {
	mu       sync.Mutex
	created  []EditorLeaseRecord
	stopped  []string
	holders  []ClaimHolder
	overlaps [][]string
}

func (r *recordingLeaseRecorder) Create(_ context.Context, record EditorLeaseRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.created = append(r.created, record)
	return nil
}

func (r *recordingLeaseRecorder) Heartbeat(context.Context, string) error { return nil }

func (r *recordingLeaseRecorder) Stop(_ context.Context, sessionID, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopped = append(r.stopped, sessionID+":"+reason)
	return nil
}

func (r *recordingLeaseRecorder) Overlaps(_ context.Context, paths []string) ([]ClaimHolder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.overlaps = append(r.overlaps, paths)
	return r.holders, nil
}

func withLeaseRecorder(t *testing.T, recorder EditorLeaseRecorder) {
	t.Helper()
	previous := DefaultEditorLeaseRecorder
	DefaultEditorLeaseRecorder = recorder
	t.Cleanup(func() { DefaultEditorLeaseRecorder = previous })
}

// [REQ:STORM-002] The attach body carries the tree and the scope, and the
// session's editor lease records both with the launcher's pid.
func TestLauncherSendsWorkingDirAndScope(t *testing.T) {
	t.Setenv(AgentManagerIdentityTokenEnv, "")
	var attachBody map[string]any
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/runs/attach" {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			mu.Lock()
			attachBody = body
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"run":{"id":"run-lease-1"},"identity_token":"safe-token"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	recorder := &recordingLeaseRecorder{}
	withLeaseRecorder(t, recorder)
	dir := t.TempDir()
	result, err := LaunchCodingAgentResult(context.Background(), AgentLaunchRequest{
		Agent:      "codex",
		APIBase:    server.URL,
		WorkingDir: dir,
		LookPath:   func(string) (string, error) { return "/safe/fixture/codex", nil },
		RunChild: func(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error {
			return nil
		},
		AttachTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("LaunchCodingAgentResult() error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if attachBody["working_dir"] != dir || !strings.HasPrefix(attachBody["scope"].(string), "vrooli-agent-") {
		t.Fatalf("attach body = %v, want working_dir and scope", attachBody)
	}
	if !result.LeaseRecorded || len(recorder.created) != 1 {
		t.Fatalf("lease not recorded: result=%+v created=%v", result, recorder.created)
	}
	lease := recorder.created[0]
	if lease.SessionID != "run-lease-1" || lease.WorkingDir != dir || lease.PID != os.Getpid() || lease.Scope != attachBody["scope"] || lease.Agent != "codex" {
		t.Fatalf("lease = %+v", lease)
	}
	if len(recorder.stopped) != 1 || recorder.stopped[0] != "run-lease-1:session ended" {
		t.Fatalf("lease stop = %v", recorder.stopped)
	}
}

// [REQ:STORM-002] An overlapping claim names its holder on stderr and the
// launch continues; claims are advisory in this release.
func TestClaimOverlapNamesHolderAndContinues(t *testing.T) {
	t.Setenv(AgentManagerIdentityTokenEnv, "")
	recorder := &recordingLeaseRecorder{holders: []ClaimHolder{{SessionID: "other", Agent: "claude", PID: 4242, WorkingDir: "/repo", HeldPath: "/repo/internal", Overlap: "existing_contains_new", Age: 90 * time.Second}}}
	withLeaseRecorder(t, recorder)
	var stderr bytes.Buffer
	var ran bool
	result, err := LaunchCodingAgentResult(context.Background(), AgentLaunchRequest{
		Agent:    "codex",
		APIBase:  "http://127.0.0.1:1",
		Claims:   []string{"/repo/internal/shell"},
		Stderr:   &stderr,
		LookPath: func(string) (string, error) { return "/safe/fixture/codex", nil },
		RunChild: func(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error {
			ran = true
			return nil
		},
		AttachTimeout: 50 * time.Millisecond,
	})
	if err != nil || !ran {
		t.Fatalf("launch did not continue: ran=%v err=%v", ran, err)
	}
	if len(result.ClaimOverlaps) != 1 || result.ClaimOverlaps[0].SessionID != "other" {
		t.Fatalf("overlaps = %+v", result.ClaimOverlaps)
	}
	message := stderr.String()
	for _, want := range []string{"/repo/internal", "claude", "session other", "pid 4242", "1m30s", "continuing"} {
		if !strings.Contains(message, want) {
			t.Fatalf("stderr %q lacks %q", message, want)
		}
	}
	if len(recorder.created) != 1 || recorder.created[0].Claims[0] != "/repo/internal/shell" {
		t.Fatalf("lease claims = %+v", recorder.created)
	}
}
