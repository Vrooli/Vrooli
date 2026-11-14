package session

import (
	"context"
	"encoding/json"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
	"net/http"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/internal/testutil/fakedriver"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

func TestManager_ApplyDefaults(t *testing.T) {
	m := &Manager{defaultViewport: Viewport{Width: 1920, Height: 1080}}
	for _, tc := range []struct {
		name           string
		spec           Spec
		width          int
		height         int
		frameStreaming *FrameStreamingConfig
	}{
		{name: "viewport defaults", spec: Spec{Mode: ModeRecording}, width: 1920, height: 1080},
		{name: "explicit viewport preserved", spec: Spec{Mode: ModeRecording, ViewportWidth: 800, ViewportHeight: 600}, width: 800, height: 600},
		{
			name:  "frame streaming defaults",
			spec:  Spec{Mode: ModeRecording, ViewportWidth: 1280, ViewportHeight: 720, FrameStreaming: &FrameStreamingConfig{}},
			width: 1280, height: 720, frameStreaming: &FrameStreamingConfig{Quality: 55, FPS: 6, Scale: "css"},
		},
		{
			name:  "explicit frame streaming preserved",
			spec:  Spec{Mode: ModeRecording, ViewportWidth: 1280, ViewportHeight: 720, FrameStreaming: &FrameStreamingConfig{Quality: 80, FPS: 12, Scale: "device"}},
			width: 1280, height: 720, frameStreaming: &FrameStreamingConfig{Quality: 80, FPS: 12, Scale: "device"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := m.applyDefaults(tc.spec)
			require.Equal(t, tc.width, got.ViewportWidth)
			require.Equal(t, tc.height, got.ViewportHeight)
			require.Equal(t, tc.frameStreaming, got.FrameStreaming)
		})
	}
}

func TestManager_Get_ExistingSession(t *testing.T) {
	t.Parallel()

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	sessionID := "test-session-get"
	srv := fakedriver.StartSessionServer(t, sessionID)

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	m := NewManagerWithClient(client, WithLogger(log))
	ctx := context.Background()

	// Create a session
	session, err := m.Create(ctx, Spec{
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Mode:           ModeRecording,
		ViewportWidth:  1280,
		ViewportHeight: 720,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get the session
	retrieved, ok := m.Get(session.ID())
	if !ok {
		t.Error("expected session to be found")
	}
	if retrieved == nil {
		t.Fatal("expected non-nil session")
	}
	if retrieved.ID() != session.ID() {
		t.Errorf("expected session ID %s, got %s", session.ID(), retrieved.ID())
	}
}

func TestManager_Get_NotFound(t *testing.T) {
	t.Parallel()

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	srv := fakedriver.StartSessionServer(t, "test-session")

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	m := NewManagerWithClient(client, WithLogger(log))

	// Try to get a non-existent session
	_, ok := m.Get("non-existent-session")
	if ok {
		t.Error("expected session to not be found")
	}
}

func TestManager_Close_RemovesSession(t *testing.T) {
	t.Parallel()

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	sessionID := "test-session-close"
	srv := fakedriver.StartSessionServer(t, sessionID)

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	m := NewManagerWithClient(client, WithLogger(log))
	ctx := context.Background()

	// Create a session
	session, err := m.Create(ctx, Spec{
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Mode:           ModeRecording,
		ViewportWidth:  1280,
		ViewportHeight: 720,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify session exists
	if _, ok := m.Get(session.ID()); !ok {
		t.Fatal("expected session to exist before close")
	}

	// Close the session
	err = m.Close(ctx, session.ID())
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify session is removed
	if _, ok := m.Get(session.ID()); ok {
		t.Error("expected session to be removed after close")
	}

	// Verify active count is 0
	if m.ActiveCount() != 0 {
		t.Errorf("expected 0 active sessions, got %d", m.ActiveCount())
	}
}

func TestManager_DirectSessionCloseDeregistersSession(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/session/start", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session_id":      "direct-close-session",
			"actual_viewport": map[string]any{"width": 1280, "height": 720, "source": "requested"},
		})
	})
	mux.HandleFunc("/session/direct-close-session/close", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})
	srv := testutil.StartHTTPServer(t, mux)

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	m := NewManagerWithClient(client)
	sess, err := m.Create(context.Background(), Spec{ExecutionID: uuid.New(), WorkflowID: uuid.New(), Mode: ModeExecution})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := sess.Close(context.Background()); err != nil {
		t.Fatalf("direct session close: %v", err)
	}
	if got := m.ActiveCount(); got != 0 {
		t.Fatalf("active sessions after direct close = %d, want 0", got)
	}
}

func TestManager_CloseAll(t *testing.T) {
	t.Parallel()

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	// Create a handler that responds to multiple session IDs
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/session/start", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session_id": "session-" + string(rune('a'+callCount-1)),
			"actual_viewport": map[string]any{
				"width":  1280,
				"height": 720,
				"source": "requested",
			},
		})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})

	srv := testutil.StartHTTPServer(t, mux)

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	m := NewManagerWithClient(client, WithLogger(log))
	ctx := context.Background()

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		_, err := m.Create(ctx, Spec{
			ExecutionID:    uuid.New(),
			WorkflowID:     uuid.New(),
			Mode:           ModeRecording,
			ViewportWidth:  1280,
			ViewportHeight: 720,
		})
		if err != nil {
			t.Fatalf("Create %d failed: %v", i, err)
		}
	}

	// Verify we have 3 sessions
	if m.ActiveCount() != 3 {
		t.Errorf("expected 3 active sessions, got %d", m.ActiveCount())
	}

	// Close all sessions
	m.CloseAll(ctx)

	// Verify all sessions are closed
	if m.ActiveCount() != 0 {
		t.Errorf("expected 0 active sessions after CloseAll, got %d", m.ActiveCount())
	}
}

func TestManager_ActiveCount(t *testing.T) {
	t.Parallel()

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	// Create a handler that responds to multiple session IDs
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/session/start", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session_id": "session-" + string(rune('a'+callCount-1)),
			"actual_viewport": map[string]any{
				"width":  1280,
				"height": 720,
				"source": "requested",
			},
		})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})

	srv := testutil.StartHTTPServer(t, mux)

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	m := NewManagerWithClient(client, WithLogger(log))
	ctx := context.Background()

	// Initial count should be 0
	if m.ActiveCount() != 0 {
		t.Errorf("expected 0 initial active sessions, got %d", m.ActiveCount())
	}

	// Create sessions and verify count
	var sessions []*Session
	for i := 0; i < 5; i++ {
		session, err := m.Create(ctx, Spec{
			ExecutionID:    uuid.New(),
			WorkflowID:     uuid.New(),
			Mode:           ModeRecording,
			ViewportWidth:  1280,
			ViewportHeight: 720,
		})
		if err != nil {
			t.Fatalf("Create %d failed: %v", i, err)
		}
		sessions = append(sessions, session)

		if m.ActiveCount() != i+1 {
			t.Errorf("expected %d active sessions, got %d", i+1, m.ActiveCount())
		}
	}

	// Close some sessions and verify count decreases
	for i := 0; i < 3; i++ {
		err := m.Close(ctx, sessions[i].ID())
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}
		expected := 5 - i - 1
		if m.ActiveCount() != expected {
			t.Errorf("expected %d active sessions after closing %d, got %d", expected, i+1, m.ActiveCount())
		}
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	// Create a handler that tracks concurrent requests
	var mu sync.Mutex
	sessionCounter := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/session/start", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		sessionCounter++
		id := sessionCounter
		mu.Unlock()
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session_id": "concurrent-session-" + string(rune('a'+id-1)),
			"actual_viewport": map[string]any{
				"width":  1280,
				"height": 720,
				"source": "requested",
			},
		})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})

	srv := testutil.StartHTTPServer(t, mux)

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	m := NewManagerWithClient(client, WithLogger(log))
	ctx := context.Background()

	// Run concurrent operations
	var wg sync.WaitGroup
	errChan := make(chan error, 100)

	// Concurrent creates
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := m.Create(ctx, Spec{
				ExecutionID:    uuid.New(),
				WorkflowID:     uuid.New(),
				Mode:           ModeRecording,
				ViewportWidth:  1280,
				ViewportHeight: 720,
			})
			if err != nil {
				errChan <- err
			}
		}()
	}

	// Concurrent reads
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = m.ActiveCount()
		}()
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("concurrent operation error: %v", err)
	}

	// All creates should have succeeded
	if m.ActiveCount() != 10 {
		t.Errorf("expected 10 sessions after concurrent creates, got %d", m.ActiveCount())
	}

	// Cleanup
	m.CloseAll(ctx)
}

func TestManager_ReuseMode_Default(t *testing.T) {
	t.Parallel()

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	srv := fakedriver.StartSessionServer(t, "test-session")

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	m := NewManagerWithClient(client, WithLogger(log))

	// Test with empty reuse mode
	spec := Spec{
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Mode:           ModeRecording,
		ViewportWidth:  1280,
		ViewportHeight: 720,
		ReuseMode:      "", // Should use default "reuse"
	}

	applied := m.applyDefaults(spec)

	if applied.ReuseMode != "reuse" {
		t.Errorf("expected ReuseMode 'reuse', got '%s'", applied.ReuseMode)
	}

	// Test with explicit value
	spec2 := Spec{
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Mode:           ModeRecording,
		ViewportWidth:  1280,
		ViewportHeight: 720,
		ReuseMode:      "fresh",
	}

	applied2 := m.applyDefaults(spec2)

	if applied2.ReuseMode != "fresh" {
		t.Errorf("expected ReuseMode 'fresh', got '%s'", applied2.ReuseMode)
	}
}

func TestManager_BuildRequestCarriesSessionProfileVersion(t *testing.T) {
	m := &Manager{}
	request := m.buildRequest(Spec{
		ExecutionID:           uuid.New(),
		WorkflowID:            uuid.New(),
		Mode:                  ModeExecution,
		SessionProfileVersion: "opaque-profile-context-version",
	})

	require.Equal(t, "opaque-profile-context-version", request.Options.SessionProfileVersion)
}

func TestManager_BuildRequestSerializesResolvedOptionsWithAdmission(t *testing.T) {
	m := &Manager{}
	request := m.buildRequest(Spec{
		ExecutionID:    uuid.MustParse("f3a99b79-a67f-4bf9-bbbf-2c61f2353ff1"),
		WorkflowID:     uuid.MustParse("0a9ef5f1-c9a7-4ff0-9dde-edcdf328347d"),
		Mode:           ModeExecution,
		ViewportWidth:  1024,
		ViewportHeight: 768,
		ReuseMode:      "clean",
		BaseURL:        "https://example.test",
		FrameStreaming: &FrameStreamingConfig{URL: "ws://127.0.0.1:39000/frames", Quality: 55, FPS: 6, Scale: "device"},
	})

	payload, err := json.Marshal(request)
	require.NoError(t, err)
	var wire map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(payload, &wire))
	require.Contains(t, wire, "execution_id")
	require.Contains(t, wire, "workflow_id")
	require.NotContains(t, wire, "viewport")
	require.NotContains(t, wire, "reuse_mode")

	var options map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(wire["session_options"], &options))
	require.JSONEq(t, `{"width":1024,"height":768}`, string(options["viewport"]))
	require.JSONEq(t, `"clean"`, string(options["reuse_mode"]))
	require.JSONEq(t, `"device"`, string(options["frame_scale"]))
	require.JSONEq(t, `"https://example.test"`, string(options["base_url"]))
	var frameStreaming map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(options["frame_streaming"], &frameStreaming))
	require.NotContains(t, frameStreaming, "scale")
}

// =============================================================================
// buildArtifactPaths Tests
// =============================================================================

func TestManager_BuildArtifactPaths(t *testing.T) {
	execID := uuid.New()
	const root = "/data/artifacts"
	artifactRoot := filepath.Join(root, execID.String(), "artifacts")
	allPaths := &driver.ArtifactPaths{
		Root:             artifactRoot,
		VideoDir:         filepath.Join(artifactRoot, "videos"),
		HARPath:          filepath.Join(artifactRoot, "har", "execution-"+execID.String()+".har"),
		TracePath:        filepath.Join(artifactRoot, "traces", "execution-"+execID.String()+".zip"),
		PerfDir:          filepath.Join(artifactRoot, "performance"),
		AccessibilityDir: filepath.Join(artifactRoot, "accessibility"),
	}
	for _, tc := range []struct {
		name string
		root string
		caps *driver.CapabilityRequest
		want *driver.ArtifactPaths
	}{
		{name: "nil capabilities", root: root},
		{name: "empty root", caps: &driver.CapabilityRequest{Video: true}},
		{name: "whitespace root", root: "   ", caps: &driver.CapabilityRequest{Video: true}},
		{name: "no artifacts", root: root, caps: &driver.CapabilityRequest{}},
		{
			name: "video only", root: root, caps: &driver.CapabilityRequest{Video: true},
			want: &driver.ArtifactPaths{Root: artifactRoot, VideoDir: allPaths.VideoDir},
		},
		{
			name: "HAR only", root: root, caps: &driver.CapabilityRequest{HAR: true},
			want: &driver.ArtifactPaths{Root: artifactRoot, HARPath: allPaths.HARPath},
		},
		{
			name: "tracing only", root: root, caps: &driver.CapabilityRequest{Tracing: true},
			want: &driver.ArtifactPaths{Root: artifactRoot, TracePath: allPaths.TracePath},
		},
		{
			name: "performance trace only", root: root, caps: &driver.CapabilityRequest{PerfTrace: true},
			want: &driver.ArtifactPaths{Root: artifactRoot, PerfDir: allPaths.PerfDir},
		},
		{
			name: "accessibility only", root: root, caps: &driver.CapabilityRequest{Accessibility: true},
			want: &driver.ArtifactPaths{Root: artifactRoot, AccessibilityDir: allPaths.AccessibilityDir},
		},
		{
			name: "all artifacts", root: root,
			caps: &driver.CapabilityRequest{Video: true, HAR: true, Tracing: true, PerfTrace: true, Accessibility: true},
			want: allPaths,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := &Manager{executionArtifactsRoot: tc.root}
			got := m.buildArtifactPaths(Spec{ExecutionID: execID}, tc.caps)
			require.Equal(t, tc.want, got)
		})
	}
}

// =============================================================================
// buildFrameStreamURL Tests
// =============================================================================

func TestManager_BuildFrameStreamURL(t *testing.T) {
	for _, tc := range []struct {
		name string
		host string
		port string
	}{
		{name: "loopback", host: "127.0.0.1", port: "8080"},
		{name: "custom host and port", host: "192.168.1.100", port: "9090"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := &Manager{apiHost: tc.host, apiPort: tc.port}
			got := m.buildFrameStreamURL()
			want := "ws://" + tc.host + ":" + tc.port + "/ws/frames"
			require.Equal(t, want, got)
		})
	}
}

// =============================================================================
// Manager Option Tests
// =============================================================================

func TestManagerOptions(t *testing.T) {
	for _, tc := range []struct {
		name      string
		options   []Option
		host      string
		port      string
		root      string
		checkAPI  bool
		checkRoot bool
	}{
		{
			name:    "API endpoint",
			options: []Option{WithAPIEndpoint("custom.host", "3000")},
			host:    "custom.host", port: "3000", checkAPI: true,
		},
		{
			name:    "artifact root",
			options: []Option{WithExecutionArtifactsRoot("/custom/artifacts")},
			root:    "/custom/artifacts", checkRoot: true,
		},
		{
			name:    "artifact root trims whitespace",
			options: []Option{WithExecutionArtifactsRoot("  /artifacts/path  ")},
			root:    "/artifacts/path", checkRoot: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := NewManagerWithClient(nil, tc.options...)
			if tc.checkAPI {
				require.Equal(t, tc.host, m.apiHost)
				require.Equal(t, tc.port, m.apiPort)
			}
			if tc.checkRoot {
				require.Equal(t, tc.root, m.executionArtifactsRoot)
			}
		})
	}
}

func TestRepeatedStartPreservesTransportSequenceAcrossLiveHandles(t *testing.T) {
	var mu sync.Mutex
	highWater := float64(17)
	var sequences []float64
	server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if request.URL.Path == "/session/start" {
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"session_id": "same-session", "lease_id": "same-lease", "last_instruction_sequence": highWater}))
			return
		}
		var body map[string]any
		require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
		sequence, _ := body["operation_sequence"].(float64)
		sequences = append(sequences, sequence)
		if sequence > highWater {
			highWater = sequence
		}
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	manager := NewManagerWithClient(client)
	spec := Spec{ExecutionID: uuid.New(), WorkflowID: uuid.New(), Mode: ModeExecution}
	first, err := manager.Create(context.Background(), spec)
	require.NoError(t, err)
	instruction := contracts.CompiledInstruction{NodeID: "node", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CLICK}}
	_, err = first.Run(context.Background(), instruction)
	require.NoError(t, err)
	second, err := manager.Create(context.Background(), spec)
	require.NoError(t, err)
	_, err = first.Run(context.Background(), instruction)
	require.NoError(t, err)
	_, err = second.Run(context.Background(), instruction)
	require.NoError(t, err)
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []float64{18, 19, 20}, sequences, "start retries cannot fork/reset transport ownership")
}

func TestCreateForDrillIsBrokerOwnedAndClosesWithItsLease(t *testing.T) {
	var startBody map[string]any
	var closeBody map[string]any
	server := testutil.StartHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/session/start":
			require.Equal(t, "scoped-drill-token", request.Header.Get("X-Playwright-Drill-Token"))
			require.NoError(t, json.NewDecoder(request.Body).Decode(&startBody))
			_, _ = w.Write([]byte(`{"session_id":"drill-session","lease_id":"drill-lease","active_page_id":"page-1"}`))
		case "/session/drill-session/close":
			require.NoError(t, json.NewDecoder(request.Body).Decode(&closeBody))
			_, _ = w.Write([]byte(`{"success":true}`))
		default:
			http.NotFound(w, request)
		}
	}))
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	manager := NewManagerWithClient(client)
	req := &driver.CreateSessionRequest{ExecutionID: "drill-execution", WorkflowID: "failure-drill", Options: driver.SessionOptions{Viewport: driver.Viewport{Width: 100, Height: 100}, ReuseMode: "fresh"}}

	owned, err := manager.CreateForDrill(context.Background(), req, "scoped-drill-token")
	require.NoError(t, err)
	require.Equal(t, "drill-session", owned.ID())
	_, ok := manager.Get(owned.ID())
	require.True(t, ok, "drill lease must be registered by the broker")
	_, err = owned.CloseWithArtifacts(context.Background())
	require.NoError(t, err)
	_, ok = manager.Get(owned.ID())
	require.False(t, ok, "successful close must release the broker's lease owner")
	require.Equal(t, "drill-execution", startBody["execution_id"])
	require.Equal(t, "drill-execution", closeBody["execution_id"])
	require.Equal(t, "drill-lease", closeBody["lease_id"])
}

func TestRouteSessionRequestRequiresManagerOwnedSession(t *testing.T) {
	manager := NewManagerWithClient(nil)
	_, err := manager.RouteSessionRequest(context.Background(), "unowned", http.MethodGet, "/record/debug", nil)
	require.Error(t, err)
}
