package engine

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

func newPlaywrightEngineForServer(serverURL string, client *http.Client, log *logrus.Logger) (*PlaywrightEngine, error) {
	if client == nil {
		client = http.DefaultClient
	}
	return NewPlaywrightEngineWithHTTPClient(serverURL, client, log)
}

func TestPlaywrightEngine_Run_DecodesScreenshotAndDOM(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/session/start", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]string{"session_id": "sess-123"})
	})
	handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		w.WriteHeader(http.StatusOK)
	})
	handler.HandleFunc("/session/sess-123/run", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		resp := map[string]any{
			"schema_version":        contracts.StepOutcomeSchemaVersion,
			"payload_version":       contracts.PayloadVersion,
			"step_index":            1,
			"step_type":             "navigate",
			"node_id":               "node-1",
			"success":               true,
			"final_url":             "https://example.com",
			"screenshot_base64":     "ZmFrZS1kYXRh",
			"screenshot_media_type": "image/png",
			"screenshot_width":      800,
			"screenshot_height":     600,
			"dom_html":              "<html></html>",
			"dom_preview":           "<html>",
			"video_path":            "/tmp/video.webm",
			"trace_path":            "/tmp/trace.zip",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	handler.HandleFunc("/session/sess-123/reset", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
	})
	handler.HandleFunc("/session/sess-123/close", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	log := logrus.New()
	engine, err := newPlaywrightEngineForServer(server.URL, server.Client(), log)
	if err != nil {
		t.Fatalf("engine init: %v", err)
	}

	session, err := engine.StartSession(context.Background(), SessionSpec{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	outcome, err := session.Run(context.Background(), contracts.CompiledInstruction{
		Index:  1,
		NodeID: "node-1",
		Action: &basactions.ActionDefinition{
			Type: basactions.ActionType_ACTION_TYPE_NAVIGATE,
		},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if outcome.Screenshot == nil || string(outcome.Screenshot.Data) != "fake-data" {
		t.Fatalf("expected screenshot data decoded, got %+v", outcome.Screenshot)
	}
	if outcome.DOMSnapshot == nil || outcome.DOMSnapshot.HTML != "<html></html>" {
		t.Fatalf("expected DOM snapshot, got %+v", outcome.DOMSnapshot)
	}
	if err := session.Reset(context.Background()); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := session.Close(context.Background()); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestResolvePlaywrightDriverURL_Default(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	got := resolvePlaywrightDriverURL(log, true)
	if got == "" {
		t.Fatalf("expected default URL")
	}
	if got != "http://127.0.0.1:39400" {
		t.Fatalf("unexpected default URL: %s", got)
	}
}

func TestPlaywrightEngine_Capabilities(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	engine, err := newPlaywrightEngineForServer(server.URL, server.Client(), nil)
	if err != nil {
		t.Fatalf("engine init: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	caps, err := engine.Capabilities(ctx)
	if err != nil {
		t.Fatalf("caps: %v", err)
	}
	if caps.Engine != "playwright" {
		t.Fatalf("unexpected engine name: %s", caps.Engine)
	}
}

// mockHTTPClient implements HTTPDoer for testing
type mockHTTPClient struct {
	doFn func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.doFn != nil {
		return m.doFn(req)
	}
	return nil, errors.New("mock not configured")
}

func TestPlaywrightDriverError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *PlaywrightDriverError
		contains []string
	}{
		{
			name: "basic error with message",
			err: &PlaywrightDriverError{
				Op:      "health",
				Message: "connection refused",
			},
			contains: []string{"health", "failed", "connection refused"},
		},
		{
			name: "error with URL",
			err: &PlaywrightDriverError{
				Op:      "start_session",
				URL:     "http://localhost:39400",
				Message: "timeout",
			},
			contains: []string{"start_session", "localhost:39400", "timeout"},
		},
		{
			name: "error with cause",
			err: &PlaywrightDriverError{
				Op:      "run",
				URL:     "http://localhost:39400",
				Message: "request failed",
				Cause:   errors.New("EOF"),
			},
			contains: []string{"run", "request failed", "EOF"},
		},
		{
			name: "error with hint",
			err: &PlaywrightDriverError{
				Op:      "health",
				URL:     "http://localhost:39400",
				Message: "not responding",
				Hint:    "ensure playwright-driver is running",
			},
			contains: []string{"health", "not responding", "hint", "playwright-driver"},
		},
		{
			name: "full error with all fields",
			err: &PlaywrightDriverError{
				Op:      "start_session",
				URL:     "http://localhost:39400",
				Message: "session limit reached",
				Cause:   errors.New("max concurrent sessions: 2"),
				Hint:    "wait for other executions to complete",
			},
			contains: []string{
				"start_session",
				"localhost:39400",
				"session limit reached",
				"max concurrent sessions",
				"hint",
				"wait for other executions",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, substr := range tt.contains {
				if !strings.Contains(errStr, substr) {
					t.Errorf("expected error string to contain %q, got: %s", substr, errStr)
				}
			}
		})
	}
}

func TestPlaywrightDriverError_Unwrap(t *testing.T) {
	cause := errors.New("underlying cause")
	err := &PlaywrightDriverError{
		Op:      "test",
		Message: "wrapper",
		Cause:   cause,
	}

	unwrapped := errors.Unwrap(err)
	if unwrapped != cause {
		t.Errorf("expected Unwrap to return cause, got %v", unwrapped)
	}

	// Test errors.Is works through the wrapper
	if !errors.Is(err, cause) {
		t.Error("expected errors.Is(err, cause) to be true")
	}
}

func TestNewPlaywrightEngineWithHTTPClient(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("creates engine with valid inputs", func(t *testing.T) {
		mock := &mockHTTPClient{}
		engine, err := NewPlaywrightEngineWithHTTPClient("http://localhost:39400", mock, log)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if engine == nil {
			t.Fatal("expected engine to be non-nil")
		}
		if engine.Name() != "playwright" {
			t.Errorf("expected name 'playwright', got %s", engine.Name())
		}
	})

	t.Run("strips trailing slash from URL", func(t *testing.T) {
		mock := &mockHTTPClient{}
		engine, err := NewPlaywrightEngineWithHTTPClient("http://localhost:39400/", mock, log)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if engine == nil {
			t.Fatal("expected engine to be non-nil")
		}
	})

	t.Run("fails with empty URL", func(t *testing.T) {
		mock := &mockHTTPClient{}
		_, err := NewPlaywrightEngineWithHTTPClient("", mock, log)
		if err == nil {
			t.Fatal("expected error for empty URL")
		}
		if !strings.Contains(err.Error(), "driver URL is required") {
			t.Errorf("expected URL error, got: %v", err)
		}
	})

	t.Run("fails with whitespace-only URL", func(t *testing.T) {
		mock := &mockHTTPClient{}
		_, err := NewPlaywrightEngineWithHTTPClient("   ", mock, log)
		if err == nil {
			t.Fatal("expected error for whitespace URL")
		}
	})

	t.Run("fails with nil HTTP client", func(t *testing.T) {
		_, err := NewPlaywrightEngineWithHTTPClient("http://localhost:39400", nil, log)
		if err == nil {
			t.Fatal("expected error for nil HTTP client")
		}
		if !strings.Contains(err.Error(), "httpClient") {
			t.Errorf("expected httpClient error, got: %v", err)
		}
	})
}

func TestPlaywrightEngine_Health_ErrorCases(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("returns error when driver unreachable", func(t *testing.T) {
		mock := &mockHTTPClient{
			doFn: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("connection refused")
			},
		}
		engine, _ := NewPlaywrightEngineWithHTTPClient("http://localhost:39400", mock, log)
		_, err := engine.Capabilities(context.Background())
		if err == nil {
			t.Fatal("expected error when driver unreachable")
		}
		var driverErr *PlaywrightDriverError
		if !errors.As(err, &driverErr) {
			t.Fatalf("expected PlaywrightDriverError, got %T", err)
		}
		if driverErr.Op != "health" {
			t.Errorf("expected op 'health', got %s", driverErr.Op)
		}
	})

	t.Run("returns error for non-200 health response", func(t *testing.T) {
		mock := &mockHTTPClient{
			doFn: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusServiceUnavailable,
					Body:       io.NopCloser(strings.NewReader("Service temporarily unavailable")),
				}, nil
			},
		}
		engine, _ := NewPlaywrightEngineWithHTTPClient("http://localhost:39400", mock, log)
		_, err := engine.Capabilities(context.Background())
		if err == nil {
			t.Fatal("expected error for non-200 response")
		}
		var driverErr *PlaywrightDriverError
		if !errors.As(err, &driverErr) {
			t.Fatalf("expected PlaywrightDriverError, got %T", err)
		}
		if !strings.Contains(driverErr.Error(), "status 503") {
			t.Errorf("expected status code in error, got: %s", driverErr.Error())
		}
	})

	t.Run("returns error when engine not configured", func(t *testing.T) {
		var engine *PlaywrightEngine
		_, err := engine.Capabilities(context.Background())
		if err == nil {
			t.Fatal("expected error for nil engine")
		}
		if !strings.Contains(err.Error(), "engine not configured") {
			t.Fatalf("expected engine not configured error, got: %v", err)
		}
	})
}

func TestPlaywrightEngine_StartSession_ErrorCases(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("returns error when connection fails", func(t *testing.T) {
		mock := &mockHTTPClient{
			doFn: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("dial tcp: connection refused")
			},
		}
		engine, _ := NewPlaywrightEngineWithHTTPClient("http://localhost:39400", mock, log)
		_, err := engine.StartSession(context.Background(), SessionSpec{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
		})
		if err == nil {
			t.Fatal("expected error when connection fails")
		}
		var driverErr *PlaywrightDriverError
		if !errors.As(err, &driverErr) {
			t.Fatalf("expected PlaywrightDriverError, got %T", err)
		}
		if driverErr.Op != "POST /session/start" {
			t.Errorf("expected op 'POST /session/start', got %s", driverErr.Op)
		}
	})

	t.Run("returns error for non-200 response", func(t *testing.T) {
		mock := &mockHTTPClient{
			doFn: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusTooManyRequests,
					Body:       io.NopCloser(strings.NewReader("Maximum concurrent sessions reached")),
				}, nil
			},
		}
		engine, _ := NewPlaywrightEngineWithHTTPClient("http://localhost:39400", mock, log)
		_, err := engine.StartSession(context.Background(), SessionSpec{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
		})
		if err == nil {
			t.Fatal("expected error for non-200 response")
		}
		var driverErr *PlaywrightDriverError
		if !errors.As(err, &driverErr) {
			t.Fatalf("expected PlaywrightDriverError, got %T", err)
		}
		if !strings.Contains(driverErr.Error(), "status 429") {
			t.Errorf("expected status code in error, got: %s", driverErr.Error())
		}
		if !strings.Contains(driverErr.Hint, "concurrent sessions") {
			t.Errorf("expected helpful hint about sessions, got: %s", driverErr.Hint)
		}
	})

	t.Run("returns error for invalid JSON response", func(t *testing.T) {
		mock := &mockHTTPClient{
			doFn: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("not json")),
				}, nil
			},
		}
		engine, _ := NewPlaywrightEngineWithHTTPClient("http://localhost:39400", mock, log)
		_, err := engine.StartSession(context.Background(), SessionSpec{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
		})
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
		if !strings.Contains(err.Error(), "parse response") {
			t.Fatalf("expected parse response error, got %v", err)
		}
	})

	t.Run("returns error for empty session_id", func(t *testing.T) {
		mock := &mockHTTPClient{
			doFn: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"session_id":""}`)),
				}, nil
			},
		}
		engine, _ := NewPlaywrightEngineWithHTTPClient("http://localhost:39400", mock, log)
		_, err := engine.StartSession(context.Background(), SessionSpec{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
		})
		if err == nil {
			t.Fatal("expected error for empty session_id")
		}
		var driverErr *PlaywrightDriverError
		if !errors.As(err, &driverErr) {
			t.Fatalf("expected PlaywrightDriverError, got %T", err)
		}
		if !strings.Contains(driverErr.Message, "empty session_id") {
			t.Errorf("expected empty session_id message, got: %s", driverErr.Message)
		}
	})

	t.Run("includes browser launch hint when relevant", func(t *testing.T) {
		mock := &mockHTTPClient{
			doFn: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("browser failed to launch: cannot find chromium")),
				}, nil
			},
		}
		engine, _ := NewPlaywrightEngineWithHTTPClient("http://localhost:39400", mock, log)
		_, err := engine.StartSession(context.Background(), SessionSpec{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
		})
		if err == nil {
			t.Fatal("expected error")
		}
		var driverErr *PlaywrightDriverError
		if !errors.As(err, &driverErr) {
			t.Fatalf("expected PlaywrightDriverError, got %T", err)
		}
		if !strings.Contains(driverErr.Hint, "browser") {
			t.Errorf("expected browser hint, got: %s", driverErr.Hint)
		}
	})
}

func TestPlaywrightSession_Run_ErrorCases(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("returns error when connection fails", func(t *testing.T) {
		requestCount := 0
		mock := &mockHTTPClient{
			doFn: func(req *http.Request) (*http.Response, error) {
				requestCount++
				// First request is StartSession
				if requestCount == 1 {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"session_id":"sess-123"}`)),
					}, nil
				}
				// Second request (Run) fails
				return nil, errors.New("connection reset by peer")
			},
		}
		engine, _ := NewPlaywrightEngineWithHTTPClient("http://localhost:39400", mock, log)
		session, _ := engine.StartSession(context.Background(), SessionSpec{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
		})

		_, err := session.Run(context.Background(), contracts.CompiledInstruction{
			Index:  0,
			NodeID: "node-1",
			Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE},
		})
		if err == nil {
			t.Fatal("expected error when run fails")
		}
		if !strings.Contains(err.Error(), "connection reset") {
			t.Errorf("expected connection error, got: %v", err)
		}
	})

	t.Run("returns error for non-200 run response", func(t *testing.T) {
		requestCount := 0
		mock := &mockHTTPClient{
			doFn: func(req *http.Request) (*http.Response, error) {
				requestCount++
				if requestCount == 1 {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"session_id":"sess-123"}`)),
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader("Invalid instruction type")),
				}, nil
			},
		}
		engine, _ := NewPlaywrightEngineWithHTTPClient("http://localhost:39400", mock, log)
		session, _ := engine.StartSession(context.Background(), SessionSpec{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
		})

		_, err := session.Run(context.Background(), contracts.CompiledInstruction{
			Index:  0,
			NodeID: "node-1",
			Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_UNSPECIFIED},
		})
		if err == nil {
			t.Fatal("expected error for bad request")
		}
		if !strings.Contains(err.Error(), "Invalid instruction type") {
			t.Errorf("expected error message from response, got: %v", err)
		}
	})

	t.Run("returns error for invalid JSON outcome", func(t *testing.T) {
		requestCount := 0
		mock := &mockHTTPClient{
			doFn: func(req *http.Request) (*http.Response, error) {
				requestCount++
				if requestCount == 1 {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"session_id":"sess-123"}`)),
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("not valid json")),
				}, nil
			},
		}
		engine, _ := NewPlaywrightEngineWithHTTPClient("http://localhost:39400", mock, log)
		session, _ := engine.StartSession(context.Background(), SessionSpec{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
		})

		_, err := session.Run(context.Background(), contracts.CompiledInstruction{
			Index:  0,
			NodeID: "node-1",
			Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE},
		})
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
		if !strings.Contains(err.Error(), "decode driver outcome") {
			t.Errorf("expected decode error, got: %v", err)
		}
	})

	t.Run("returns error for invalid base64 screenshot", func(t *testing.T) {
		requestCount := 0
		mock := &mockHTTPClient{
			doFn: func(req *http.Request) (*http.Response, error) {
				requestCount++
				if requestCount == 1 {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"session_id":"sess-123"}`)),
					}, nil
				}
				// Return invalid base64 for screenshot
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success":true,"screenshot_base64":"!!!invalid!!!"}`)),
				}, nil
			},
		}
		engine, _ := NewPlaywrightEngineWithHTTPClient("http://localhost:39400", mock, log)
		session, _ := engine.StartSession(context.Background(), SessionSpec{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
		})

		_, err := session.Run(context.Background(), contracts.CompiledInstruction{
			Index:  0,
			NodeID: "node-1",
			Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_SCREENSHOT},
		})
		if err == nil {
			t.Fatal("expected error for invalid base64")
		}
		if !strings.Contains(err.Error(), "decode screenshot") {
			t.Errorf("expected screenshot decode error, got: %v", err)
		}
	})
}

func TestPlaywrightSession_Reset_ErrorCases(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("returns error when reset fails", func(t *testing.T) {
		requestCount := 0
		mock := &mockHTTPClient{
			doFn: func(req *http.Request) (*http.Response, error) {
				requestCount++
				if requestCount == 1 {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"session_id":"sess-123"}`)),
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("Session state corrupted")),
				}, nil
			},
		}
		engine, _ := NewPlaywrightEngineWithHTTPClient("http://localhost:39400", mock, log)
		session, _ := engine.StartSession(context.Background(), SessionSpec{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
		})

		err := session.Reset(context.Background())
		if err == nil {
			t.Fatal("expected error when reset fails")
		}
		if !strings.Contains(err.Error(), "reset failed") {
			t.Errorf("expected reset failed error, got: %v", err)
		}
	})

	t.Run("returns error when connection drops", func(t *testing.T) {
		requestCount := 0
		mock := &mockHTTPClient{
			doFn: func(req *http.Request) (*http.Response, error) {
				requestCount++
				if requestCount == 1 {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"session_id":"sess-123"}`)),
					}, nil
				}
				return nil, errors.New("broken pipe")
			},
		}
		engine, _ := NewPlaywrightEngineWithHTTPClient("http://localhost:39400", mock, log)
		session, _ := engine.StartSession(context.Background(), SessionSpec{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
		})

		err := session.Reset(context.Background())
		if err == nil {
			t.Fatal("expected error when connection drops")
		}
	})
}

func TestPlaywrightSession_Close_ErrorCases(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("returns error when close fails", func(t *testing.T) {
		requestCount := 0
		mock := &mockHTTPClient{
			doFn: func(req *http.Request) (*http.Response, error) {
				requestCount++
				if requestCount == 1 {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"session_id":"sess-123"}`)),
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("Session not found")),
				}, nil
			},
		}
		engine, _ := NewPlaywrightEngineWithHTTPClient("http://localhost:39400", mock, log)
		session, _ := engine.StartSession(context.Background(), SessionSpec{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
		})

		err := session.Close(context.Background())
		if err == nil {
			t.Fatal("expected error when close fails")
		}
		if !strings.Contains(err.Error(), "Session not found") {
			t.Errorf("expected close error to include response body, got: %v", err)
		}
	})
}

func TestPlaywrightEngine_Run_VideoAndTracePaths(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/session/start", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]string{"session_id": "sess-123"})
	})
	handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		w.WriteHeader(http.StatusOK)
	})
	handler.HandleFunc("/session/sess-123/run", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		resp := map[string]any{
			"schema_version":  contracts.StepOutcomeSchemaVersion,
			"payload_version": contracts.PayloadVersion,
			"step_index":      0,
			"step_type":       "navigate",
			"node_id":         "node-1",
			"success":         true,
			"video_path":      "/tmp/recording-123.webm",
			"trace_path":      "/tmp/trace-123.zip",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	log := logrus.New()
	engine, err := newPlaywrightEngineForServer(server.URL, server.Client(), log)
	if err != nil {
		t.Fatalf("engine init: %v", err)
	}

	session, err := engine.StartSession(context.Background(), SessionSpec{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	outcome, err := session.Run(context.Background(), contracts.CompiledInstruction{
		Index:  0,
		NodeID: "node-1",
		Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if outcome.Notes == nil {
		t.Fatal("expected Notes to be populated with video/trace paths")
	}
	if outcome.Notes["video_path"] != "/tmp/recording-123.webm" {
		t.Errorf("expected video_path, got %v", outcome.Notes["video_path"])
	}
	if outcome.Notes["trace_path"] != "/tmp/trace-123.zip" {
		t.Errorf("expected trace_path, got %v", outcome.Notes["trace_path"])
	}
}

func TestPlaywrightEngine_Run_ContextCancellation(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)

	t.Run("honors context cancellation during run", func(t *testing.T) {
		mock := &mockHTTPClient{
			doFn: func(req *http.Request) (*http.Response, error) {
				if strings.Contains(req.URL.Path, "/start") {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"session_id":"sess-123"}`)),
					}, nil
				}
				// Check if context is cancelled
				select {
				case <-req.Context().Done():
					return nil, req.Context().Err()
				default:
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"success":true}`)),
					}, nil
				}
			},
		}
		engine, _ := NewPlaywrightEngineWithHTTPClient("http://localhost:39400", mock, log)
		session, _ := engine.StartSession(context.Background(), SessionSpec{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
		})

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := session.Run(ctx, contracts.CompiledInstruction{
			Index:  0,
			NodeID: "node-1",
			Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE},
		})
		if err == nil {
			t.Fatal("expected error for cancelled context")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled error, got: %v", err)
		}
	})
}

func TestPlaywrightEngine_Run_SetsSchemaVersions(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/session/start", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]string{"session_id": "sess-123"})
	})
	handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		w.WriteHeader(http.StatusOK)
	})
	handler.HandleFunc("/session/sess-123/run", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":    true,
			"step_index": 0,
			"node_id":    "node-1",
			"step_type":  "click",
		})
	})
	handler.HandleFunc("/session/sess-123/close", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	engine, err := newPlaywrightEngineForServer(server.URL, server.Client(), nil)
	if err != nil {
		t.Fatalf("engine init: %v", err)
	}

	session, err := engine.StartSession(context.Background(), SessionSpec{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	outcome, err := session.Run(context.Background(), contracts.CompiledInstruction{
		Index:  0,
		NodeID: "node-1",
		Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CLICK},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if outcome.SchemaVersion != contracts.StepOutcomeSchemaVersion {
		t.Errorf("expected SchemaVersion %s, got %s", contracts.StepOutcomeSchemaVersion, outcome.SchemaVersion)
	}
	if outcome.PayloadVersion != contracts.PayloadVersion {
		t.Errorf("expected PayloadVersion %s, got %s", contracts.PayloadVersion, outcome.PayloadVersion)
	}
}

func TestPlaywrightEngine_Capabilities_AllFieldsPopulated(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	engine, err := newPlaywrightEngineForServer(server.URL, server.Client(), nil)
	if err != nil {
		t.Fatalf("engine init: %v", err)
	}

	caps, err := engine.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("caps: %v", err)
	}

	// Verify all capability fields are properly set
	if caps.SchemaVersion != contracts.CapabilitiesSchemaVersion {
		t.Errorf("expected SchemaVersion %s, got %s", contracts.CapabilitiesSchemaVersion, caps.SchemaVersion)
	}
	if caps.Engine != "playwright" {
		t.Errorf("expected Engine 'playwright', got %s", caps.Engine)
	}
	if caps.Version == "" {
		t.Error("expected Version to be set")
	}
	if caps.MaxConcurrentSessions <= 0 {
		t.Errorf("expected positive MaxConcurrentSessions, got %d", caps.MaxConcurrentSessions)
	}
	if caps.MaxViewportWidth <= 0 || caps.MaxViewportHeight <= 0 {
		t.Errorf("expected positive viewport dimensions, got %dx%d", caps.MaxViewportWidth, caps.MaxViewportHeight)
	}

	// Verify feature flags
	if !caps.SupportsHAR {
		t.Error("expected SupportsHAR to be true")
	}
	if !caps.SupportsVideo {
		t.Error("expected SupportsVideo to be true")
	}
	if !caps.SupportsIframes {
		t.Error("expected SupportsIframes to be true")
	}
	if !caps.SupportsTracing {
		t.Error("expected SupportsTracing to be true")
	}
}
