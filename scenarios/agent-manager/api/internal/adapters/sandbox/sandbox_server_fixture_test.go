// Responsibility: retain sandbox launcher test declarations within their original package.
package sandbox

import (
	"agent-manager/internal/adapters/runner"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// sandboxTestServer is an httptest-backed simulator of the workspace-sandbox
// /processes endpoints. It models a single process at a time. Tests can
// drive the lifecycle by calling appendStdout / appendStderr / markExited
// to feed bytes into the SSE channels and wind down the streams cleanly.
type sandboxTestServer struct {
	mu sync.Mutex

	procPID     int
	procRunning atomic.Bool
	procStarted atomic.Bool

	// hostMergedDir is the value returned from GET /api/v1/sandboxes/{id};
	// the launcher uses it to translate host workingDir / env values.
	// Empty means "no translation expected" (translateHostPathToNamespace
	// becomes identity in that case).
	hostMergedDir string

	// homeOverlayState is what the mock GET returns. Empty defaults to
	// "present" so existing happy-path tests don't need to set it.
	homeOverlayState string

	// identityLayout, when true, makes the mock report the identity layout
	// (workspacePath == mergedDir, pathIllusion=false — copy driver / macOS).
	// The default (false) reports the bwrap path-illusion layout
	// (workspacePath="/workspace", pathIllusion=true) so existing
	// translation tests keep their contract.
	identityLayout       bool
	policyFilesSupported bool

	// Recorded request state for assertions.
	startProcessBody  map[string]any
	startProcessSeen  atomic.Bool
	killSeen          atomic.Bool
	stdinBody         []byte
	stdinClose        atomic.Bool
	stdinSeen         atomic.Bool
	startProcessCode  int
	startProcessReply string

	// Per-stream subscriber registry. Each /logs/stream connection adds a
	// channel that receives chunks; markExited closes them after sending
	// the exit frame.
	stdoutSubs []chan sseChunk
	stderrSubs []chan sseChunk
	subsMu     sync.Mutex

	// Buffered exit info; sent on exitFrame as JSON.
	exitInfo *remoteExitInfo
	// listProcessExit simulates provider-side terminal evidence available
	// after the SSE transport itself was interrupted.
	listProcessExit *remoteExitInfo
}

type sseChunk struct {
	event string
	data  []byte
}

func newSandboxTestServer(initialPID int) *sandboxTestServer {
	return &sandboxTestServer{procPID: initialPID}
}

// startServer wires the routes and returns the running test server.
func (m *sandboxTestServer) startServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/sandboxes/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == "POST" && strings.HasSuffix(path, "/processes"):
			m.handleStartProcess(w, r)
		case r.Method == "POST" && strings.Contains(path, "/processes/") && strings.HasSuffix(path, "/stdin"):
			m.handleStdin(w, r)
		case r.Method == "GET" && strings.Contains(path, "/processes/") && strings.HasSuffix(path, "/logs/stream"):
			m.handleStreamLogs(w, r)
		case r.Method == "GET" && strings.HasSuffix(path, "/processes"):
			m.handleListProcesses(w, r)
		case r.Method == "DELETE" && strings.Contains(path, "/processes/"):
			m.handleKillProcess(w, r)
		case r.Method == "GET" && !strings.Contains(path, "/processes"):
			m.handleGetSandbox(w, r)
		default:
			t.Logf("sandboxTestServer: unhandled %s %s", r.Method, path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	return httptest.NewServer(mux)
}

func (m *sandboxTestServer) handleListProcesses(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	exit := m.listProcessExit
	m.mu.Unlock()
	processes := []map[string]any{}
	if exit != nil {
		processes = append(processes, map[string]any{
			"pid":       m.procPID,
			"exitCode":  exit.ExitCode,
			"signal":    exit.Signal,
			"oomKilled": exit.OOMKilled,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"processes": processes, "total": len(processes), "running": 0})
}

func (m *sandboxTestServer) handleGetSandbox(w http.ResponseWriter, r *http.Request) {
	// Extract sandbox id from /api/v1/sandboxes/<id>.
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/sandboxes/"), "/")
	id := parts[0]
	state := m.homeOverlayState
	if state == "" {
		// Mock default = Present so existing happy-path tests don't have
		// to set the field. The Phase F regression tests explicitly set
		// HomeOverlayAbsent / HomeOverlayUnsupported when needed.
		state = string(HomeOverlayPresent)
	}
	// Report the negotiated workspace layout. Default = bwrap path-illusion
	// ("/workspace", true); identityLayout flips to the copy/seatbelt layout
	// where the agent-visible path equals the host merged dir.
	workspacePath := "/workspace"
	pathIllusion := true
	containment := map[string]any{
		"level":        "required",
		"backend":      "bwrap",
		"enforcements": []string{"filesystem-write-containment", "network-deny", "pid-namespace", "path-illusion"},
	}
	if m.policyFilesSupported {
		containment["enforcements"] = append(containment["enforcements"].([]string), runner.EnforcementPolicyFiles)
	}
	if m.identityLayout {
		workspacePath = m.hostMergedDir
		pathIllusion = false
		containment = map[string]any{
			"level":        "none",
			"backend":      "none",
			"enforcements": []string{},
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":               id,
		"scopePath":        "/scope",
		"projectRoot":      "/scope",
		"status":           "active",
		"mergedDir":        m.hostMergedDir,
		"workspacePath":    workspacePath,
		"pathIllusion":     pathIllusion,
		"containment":      containment,
		"homeOverlayState": state,
	})
}

func (m *sandboxTestServer) handleStartProcess(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var parsed map[string]any
	_ = json.Unmarshal(body, &parsed)

	m.mu.Lock()
	m.startProcessBody = parsed
	override := m.startProcessCode
	overrideBody := m.startProcessReply
	m.mu.Unlock()
	m.startProcessSeen.Store(true)

	if override != 0 && override != http.StatusCreated && override != http.StatusOK {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(override)
		_, _ = w.Write([]byte(overrideBody))
		return
	}

	m.procRunning.Store(true)
	m.procStarted.Store(true)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"pid":       m.procPID,
		"sandboxId": uuid.New(),
		"command":   parsed["command"],
		"withStdin": parsed["withStdin"],
	})
}

func (m *sandboxTestServer) handleStdin(w http.ResponseWriter, r *http.Request) {
	m.stdinSeen.Store(true)
	body, _ := io.ReadAll(r.Body)
	m.mu.Lock()
	m.stdinBody = append(m.stdinBody, body...)
	m.mu.Unlock()
	if r.URL.Query().Get("close") == "true" {
		m.stdinClose.Store(true)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"pid":          m.procPID,
		"bytesWritten": len(body),
		"closed":       r.URL.Query().Get("close") == "true",
	})
}

func (m *sandboxTestServer) handleStreamLogs(w http.ResponseWriter, r *http.Request) {
	stream := r.URL.Query().Get("stream")
	if stream != "stdout" && stream != "stderr" {
		http.Error(w, "missing stream", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "no flusher", http.StatusInternalServerError)
		return
	}
	flusher.Flush()

	ch := make(chan sseChunk, 32)
	m.subsMu.Lock()
	if stream == "stdout" {
		m.stdoutSubs = append(m.stdoutSubs, ch)
	} else {
		m.stderrSubs = append(m.stderrSubs, ch)
	}
	m.subsMu.Unlock()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case chunk, open := <-ch:
			if !open {
				_, _ = fmt.Fprintf(w, "event: end\ndata: stream closed\n\n")
				flusher.Flush()
				return
			}
			if chunk.event == "exit" {
				_, _ = fmt.Fprintf(w, "event: exit\ndata: %s\n\n", string(chunk.data))
			} else {
				_, _ = fmt.Fprintf(w, "data: %s\n\n", string(chunk.data))
			}
			flusher.Flush()
		}
	}
}

func (m *sandboxTestServer) handleKillProcess(w http.ResponseWriter, r *http.Request) {
	m.killSeen.Store(true)
	m.procRunning.Store(false)
	// Closing all subscribers terminates their SSE goroutines.
	m.subsMu.Lock()
	for _, ch := range m.stdoutSubs {
		close(ch)
	}
	for _, ch := range m.stderrSubs {
		close(ch)
	}
	m.stdoutSubs = nil
	m.stderrSubs = nil
	m.subsMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// appendStdout pushes a chunk to all stdout subscribers.
func (m *sandboxTestServer) appendStdout(b []byte) {
	m.subsMu.Lock()
	defer m.subsMu.Unlock()
	for _, ch := range m.stdoutSubs {
		ch <- sseChunk{data: append([]byte(nil), b...)}
	}
}

// appendStderr pushes a chunk to all stderr subscribers.
func (m *sandboxTestServer) appendStderr(b []byte) {
	m.subsMu.Lock()
	defer m.subsMu.Unlock()
	for _, ch := range m.stderrSubs {
		ch <- sseChunk{data: append([]byte(nil), b...)}
	}
}

// markExited sends the exit frame on both streams and closes them.
func (m *sandboxTestServer) markExited(info remoteExitInfo) {
	m.procRunning.Store(false)
	m.exitInfo = &info
	payload, _ := json.Marshal(info)
	m.subsMu.Lock()
	for _, ch := range m.stdoutSubs {
		ch <- sseChunk{event: "exit", data: payload}
		close(ch)
	}
	for _, ch := range m.stderrSubs {
		ch <- sseChunk{event: "exit", data: payload}
		close(ch)
	}
	m.stdoutSubs = nil
	m.stderrSubs = nil
	m.subsMu.Unlock()
}

// =============================================================================
// Tests
// =============================================================================

// nsPath is the agent-visible workspace path under the bwrap path-illusion
// layout — the value workspace-sandbox reports as workspacePath and binds
// the merged dir onto. Test-local (the production constant is server-owned).
const nsPath = "/workspace"

// waitForStreams makes fixture lifecycle events depend on registered streams,
// rather than elapsed scheduler time. The timeout bounds a broken fixture.
func (m *sandboxTestServer) waitForStreams(t *testing.T) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		m.subsMu.Lock()
		ready := len(m.stdoutSubs) > 0 && len(m.stderrSubs) > 0
		m.subsMu.Unlock()
		if ready {
			return
		}
		select {
		case <-tick.C:
		case <-deadline.C:
			t.Fatal("fixture stdout/stderr streams did not register")
		}
	}
}
