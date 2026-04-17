package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/driver"
)

// mockHTTPHandler creates a test HTTP server that responds to session lifecycle requests.
func mockHTTPHandler(t *testing.T, sessionID string) http.Handler {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/session/start", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"session_id": sessionID,
			"actual_viewport": map[string]any{
				"width":  1280,
				"height": 720,
				"source": "requested",
				"reason": "UI-requested dimensions used",
			},
		})
	})

	mux.HandleFunc("/session/"+sessionID+"/close", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
		})
	})

	return mux
}

func TestManager_ApplyDefaults_Viewport(t *testing.T) {
	t.Parallel()

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	srv := httptest.NewServer(mockHTTPHandler(t, "test-session-1"))
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	m := NewManagerWithClient(client,
		WithLogger(log),
		WithDefaultViewport(1920, 1080),
	)

	// Test with zero viewport dimensions
	spec := Spec{
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Mode:           ModeRecording,
		ViewportWidth:  0, // Should use default
		ViewportHeight: 0, // Should use default
	}

	applied := m.applyDefaults(spec)

	if applied.ViewportWidth != 1920 {
		t.Errorf("expected viewport width 1920, got %d", applied.ViewportWidth)
	}
	if applied.ViewportHeight != 1080 {
		t.Errorf("expected viewport height 1080, got %d", applied.ViewportHeight)
	}

	// Test with explicit viewport dimensions
	spec2 := Spec{
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Mode:           ModeRecording,
		ViewportWidth:  800, // Should be preserved
		ViewportHeight: 600, // Should be preserved
	}

	applied2 := m.applyDefaults(spec2)

	if applied2.ViewportWidth != 800 {
		t.Errorf("expected viewport width 800, got %d", applied2.ViewportWidth)
	}
	if applied2.ViewportHeight != 600 {
		t.Errorf("expected viewport height 600, got %d", applied2.ViewportHeight)
	}
}

func TestManager_ApplyDefaults_FrameStreaming(t *testing.T) {
	t.Parallel()

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	srv := httptest.NewServer(mockHTTPHandler(t, "test-session-2"))
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	m := NewManagerWithClient(client, WithLogger(log))

	// Test frame streaming defaults
	spec := Spec{
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Mode:           ModeRecording,
		ViewportWidth:  1280,
		ViewportHeight: 720,
		FrameStreaming: &FrameStreamingConfig{
			Quality: 0,  // Should use default 55
			FPS:     0,  // Should use default 6
			Scale:   "", // Should use default "css"
		},
	}

	applied := m.applyDefaults(spec)

	if applied.FrameStreaming == nil {
		t.Fatal("expected FrameStreaming to be non-nil")
	}
	if applied.FrameStreaming.Quality != 55 {
		t.Errorf("expected Quality 55, got %d", applied.FrameStreaming.Quality)
	}
	if applied.FrameStreaming.FPS != 6 {
		t.Errorf("expected FPS 6, got %d", applied.FrameStreaming.FPS)
	}
	if applied.FrameStreaming.Scale != "css" {
		t.Errorf("expected Scale 'css', got '%s'", applied.FrameStreaming.Scale)
	}

	// Test with explicit values
	spec2 := Spec{
		ExecutionID:    uuid.New(),
		WorkflowID:     uuid.New(),
		Mode:           ModeRecording,
		ViewportWidth:  1280,
		ViewportHeight: 720,
		FrameStreaming: &FrameStreamingConfig{
			Quality: 80,       // Should be preserved
			FPS:     12,       // Should be preserved
			Scale:   "device", // Should be preserved
		},
	}

	applied2 := m.applyDefaults(spec2)

	if applied2.FrameStreaming.Quality != 80 {
		t.Errorf("expected Quality 80, got %d", applied2.FrameStreaming.Quality)
	}
	if applied2.FrameStreaming.FPS != 12 {
		t.Errorf("expected FPS 12, got %d", applied2.FrameStreaming.FPS)
	}
	if applied2.FrameStreaming.Scale != "device" {
		t.Errorf("expected Scale 'device', got '%s'", applied2.FrameStreaming.Scale)
	}
}

func TestManager_Get_ExistingSession(t *testing.T) {
	t.Parallel()

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	sessionID := "test-session-get"
	srv := httptest.NewServer(mockHTTPHandler(t, sessionID))
	defer srv.Close()

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

	srv := httptest.NewServer(mockHTTPHandler(t, "test-session"))
	defer srv.Close()

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
	srv := httptest.NewServer(mockHTTPHandler(t, sessionID))
	defer srv.Close()

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

	srv := httptest.NewServer(mux)
	defer srv.Close()

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

	srv := httptest.NewServer(mux)
	defer srv.Close()

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

	srv := httptest.NewServer(mux)
	defer srv.Close()

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

	srv := httptest.NewServer(mockHTTPHandler(t, "test-session"))
	defer srv.Close()

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

// =============================================================================
// buildArtifactPaths Tests
// =============================================================================

func TestManager_BuildArtifactPaths_NilCapabilities(t *testing.T) {
	t.Parallel()

	m := &Manager{
		executionArtifactsRoot: "/data/artifacts",
	}

	spec := Spec{
		ExecutionID: uuid.New(),
	}

	// Should return nil when capabilities is nil
	paths := m.buildArtifactPaths(spec, nil)
	if paths != nil {
		t.Error("expected nil paths when capabilities is nil")
	}
}

func TestManager_BuildArtifactPaths_EmptyRoot(t *testing.T) {
	t.Parallel()

	m := &Manager{
		executionArtifactsRoot: "", // Empty root
	}

	spec := Spec{
		ExecutionID: uuid.New(),
	}
	caps := &driver.CapabilityRequest{
		Video: true,
	}

	// Should return nil when root is empty
	paths := m.buildArtifactPaths(spec, caps)
	if paths != nil {
		t.Error("expected nil paths when root is empty")
	}
}

func TestManager_BuildArtifactPaths_WhitespaceRoot(t *testing.T) {
	t.Parallel()

	m := &Manager{
		executionArtifactsRoot: "   ", // Whitespace only
	}

	spec := Spec{
		ExecutionID: uuid.New(),
	}
	caps := &driver.CapabilityRequest{
		Video: true,
	}

	// Should return nil when root is whitespace only
	paths := m.buildArtifactPaths(spec, caps)
	if paths != nil {
		t.Error("expected nil paths when root is whitespace")
	}
}

func TestManager_BuildArtifactPaths_NoArtifactsRequested(t *testing.T) {
	t.Parallel()

	m := &Manager{
		executionArtifactsRoot: "/data/artifacts",
	}

	spec := Spec{
		ExecutionID: uuid.New(),
	}
	caps := &driver.CapabilityRequest{
		Video:   false,
		HAR:     false,
		Tracing: false,
	}

	// Should return nil when no artifacts are requested
	paths := m.buildArtifactPaths(spec, caps)
	if paths != nil {
		t.Error("expected nil paths when no artifacts are requested")
	}
}

func TestManager_BuildArtifactPaths_VideoOnly(t *testing.T) {
	t.Parallel()

	m := &Manager{
		executionArtifactsRoot: "/data/artifacts",
	}

	execID := uuid.New()
	spec := Spec{
		ExecutionID: execID,
	}
	caps := &driver.CapabilityRequest{
		Video: true,
	}

	paths := m.buildArtifactPaths(spec, caps)
	if paths == nil {
		t.Fatal("expected non-nil paths")
	}

	expectedRoot := "/data/artifacts/" + execID.String() + "/artifacts"
	if paths.Root != expectedRoot {
		t.Errorf("expected Root %q, got %q", expectedRoot, paths.Root)
	}

	expectedVideoDir := expectedRoot + "/videos"
	if paths.VideoDir != expectedVideoDir {
		t.Errorf("expected VideoDir %q, got %q", expectedVideoDir, paths.VideoDir)
	}

	if paths.HARPath != "" {
		t.Errorf("expected empty HARPath, got %q", paths.HARPath)
	}

	if paths.TracePath != "" {
		t.Errorf("expected empty TracePath, got %q", paths.TracePath)
	}
}

func TestManager_BuildArtifactPaths_HAROnly(t *testing.T) {
	t.Parallel()

	m := &Manager{
		executionArtifactsRoot: "/data/artifacts",
	}

	execID := uuid.New()
	spec := Spec{
		ExecutionID: execID,
	}
	caps := &driver.CapabilityRequest{
		HAR: true,
	}

	paths := m.buildArtifactPaths(spec, caps)
	if paths == nil {
		t.Fatal("expected non-nil paths")
	}

	expectedHARPath := "/data/artifacts/" + execID.String() + "/artifacts/har/execution-" + execID.String() + ".har"
	if paths.HARPath != expectedHARPath {
		t.Errorf("expected HARPath %q, got %q", expectedHARPath, paths.HARPath)
	}

	if paths.VideoDir != "" {
		t.Errorf("expected empty VideoDir, got %q", paths.VideoDir)
	}
}

func TestManager_BuildArtifactPaths_TracingOnly(t *testing.T) {
	t.Parallel()

	m := &Manager{
		executionArtifactsRoot: "/data/artifacts",
	}

	execID := uuid.New()
	spec := Spec{
		ExecutionID: execID,
	}
	caps := &driver.CapabilityRequest{
		Tracing: true,
	}

	paths := m.buildArtifactPaths(spec, caps)
	if paths == nil {
		t.Fatal("expected non-nil paths")
	}

	expectedTracePath := "/data/artifacts/" + execID.String() + "/artifacts/traces/execution-" + execID.String() + ".zip"
	if paths.TracePath != expectedTracePath {
		t.Errorf("expected TracePath %q, got %q", expectedTracePath, paths.TracePath)
	}
}

func TestManager_BuildArtifactPaths_AllArtifacts(t *testing.T) {
	t.Parallel()

	m := &Manager{
		executionArtifactsRoot: "/data/artifacts",
	}

	execID := uuid.New()
	spec := Spec{
		ExecutionID: execID,
	}
	caps := &driver.CapabilityRequest{
		Video:   true,
		HAR:     true,
		Tracing: true,
	}

	paths := m.buildArtifactPaths(spec, caps)
	if paths == nil {
		t.Fatal("expected non-nil paths")
	}

	if paths.VideoDir == "" {
		t.Error("expected non-empty VideoDir")
	}
	if paths.HARPath == "" {
		t.Error("expected non-empty HARPath")
	}
	if paths.TracePath == "" {
		t.Error("expected non-empty TracePath")
	}
}

// =============================================================================
// buildFrameCallbackURL Tests
// =============================================================================

func TestManager_BuildFrameCallbackURL_RecordingMode(t *testing.T) {
	t.Parallel()

	m := &Manager{
		apiHost: "127.0.0.1",
		apiPort: "8080",
	}

	execID := uuid.New()
	spec := Spec{
		ExecutionID: execID,
		Mode:        ModeRecording,
	}

	url := m.buildFrameCallbackURL(spec)

	expected := "http://127.0.0.1:8080/api/v1/recordings/live/" + execID.String() + "/frame"
	if url != expected {
		t.Errorf("expected URL %q, got %q", expected, url)
	}
}

func TestManager_BuildFrameCallbackURL_ExecutionMode(t *testing.T) {
	t.Parallel()

	m := &Manager{
		apiHost: "127.0.0.1",
		apiPort: "8080",
	}

	execID := uuid.New()
	spec := Spec{
		ExecutionID: execID,
		Mode:        ModeExecution,
	}

	url := m.buildFrameCallbackURL(spec)

	expected := "http://127.0.0.1:8080/api/v1/executions/" + execID.String() + "/frames"
	if url != expected {
		t.Errorf("expected URL %q, got %q", expected, url)
	}
}

func TestManager_BuildFrameCallbackURL_HybridMode(t *testing.T) {
	t.Parallel()

	m := &Manager{
		apiHost: "127.0.0.1",
		apiPort: "8080",
	}

	execID := uuid.New()
	spec := Spec{
		ExecutionID: execID,
		Mode:        ModeHybrid, // Not recording mode
	}

	url := m.buildFrameCallbackURL(spec)

	// Hybrid mode uses execution endpoint (not recording endpoint)
	expected := "http://127.0.0.1:8080/api/v1/executions/" + execID.String() + "/frames"
	if url != expected {
		t.Errorf("expected URL %q, got %q", expected, url)
	}
}

func TestManager_BuildFrameCallbackURL_CustomHostPort(t *testing.T) {
	t.Parallel()

	m := &Manager{
		apiHost: "192.168.1.100",
		apiPort: "9090",
	}

	execID := uuid.New()
	spec := Spec{
		ExecutionID: execID,
		Mode:        ModeRecording,
	}

	url := m.buildFrameCallbackURL(spec)

	expected := "http://192.168.1.100:9090/api/v1/recordings/live/" + execID.String() + "/frame"
	if url != expected {
		t.Errorf("expected URL %q, got %q", expected, url)
	}
}

// =============================================================================
// WithAPIEndpoint Option Tests
// =============================================================================

func TestManager_WithAPIEndpoint(t *testing.T) {
	t.Parallel()

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	srv := httptest.NewServer(mockHTTPHandler(t, "test-session"))
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	m := NewManagerWithClient(client,
		WithLogger(log),
		WithAPIEndpoint("custom.host", "3000"),
	)

	if m.apiHost != "custom.host" {
		t.Errorf("expected apiHost 'custom.host', got '%s'", m.apiHost)
	}
	if m.apiPort != "3000" {
		t.Errorf("expected apiPort '3000', got '%s'", m.apiPort)
	}
}

// =============================================================================
// WithExecutionArtifactsRoot Option Tests
// =============================================================================

func TestManager_WithExecutionArtifactsRoot(t *testing.T) {
	t.Parallel()

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	srv := httptest.NewServer(mockHTTPHandler(t, "test-session"))
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	m := NewManagerWithClient(client,
		WithLogger(log),
		WithExecutionArtifactsRoot("/custom/artifacts"),
	)

	if m.executionArtifactsRoot != "/custom/artifacts" {
		t.Errorf("expected executionArtifactsRoot '/custom/artifacts', got '%s'", m.executionArtifactsRoot)
	}
}

func TestManager_WithExecutionArtifactsRoot_TrimsWhitespace(t *testing.T) {
	t.Parallel()

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	srv := httptest.NewServer(mockHTTPHandler(t, "test-session"))
	defer srv.Close()

	client, err := driver.NewClientWithURL(srv.URL, driver.WithoutCircuitBreaker())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	m := NewManagerWithClient(client,
		WithLogger(log),
		WithExecutionArtifactsRoot("  /artifacts/path  "),
	)

	if m.executionArtifactsRoot != "/artifacts/path" {
		t.Errorf("expected whitespace to be trimmed, got '%s'", m.executionArtifactsRoot)
	}
}
