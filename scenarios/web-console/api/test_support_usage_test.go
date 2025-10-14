package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type handlerInvocation struct {
	bodyChecked bool
}

func TestHandlerTestSuitePatterns(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/proxy"):
			if r.Header.Get("X-Forwarded-For") == "" || r.Header.Get("X-Forwarded-Proto") == "" {
				writeJSONError(w, http.StatusForbidden, "proxy headers required")
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		case strings.HasPrefix(r.URL.Path, "/teapot"):
			writeJSONError(w, http.StatusTeapot, "short and stout")
			return
		}

		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		defer r.Body.Close()
		payload, _ := io.ReadAll(r.Body)
		if len(payload) == 0 {
			writeJSONError(w, http.StatusBadRequest, "body required")
			return
		}

		var decoded map[string]any
		if err := json.Unmarshal(payload, &decoded); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid payload")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	suite := HandlerTestSuite{
		HandlerName: "DemoHandler",
		Handler:     handler,
		BaseURL:     "/",
	}

	customPattern := ErrorTestPattern{
		Name:           "CustomTeapot",
		ExpectedStatus: http.StatusTeapot,
		Execute: func(t *testing.T, setupData interface{}) (*httptest.ResponseRecorder, *http.Request) {
			req := makeHTTPRequest(HTTPTestRequest{
				Method: http.MethodGet,
				Path:   "/teapot",
			})
			w := httptest.NewRecorder()
			return w, req
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, _ interface{}) {
			assertErrorResponse(t, w, http.StatusTeapot, "short and stout")
		},
	}

	patterns := NewTestScenarioBuilder().
		AddInvalidJSON(http.MethodPost, "/resource").
		AddEmptyBody(http.MethodPost, "/resource").
		AddMethodNotAllowed(http.MethodGet, "/resource").
		AddMissingProxyHeaders(http.MethodPost, "/proxy").
		AddCustom(customPattern).
		Build()

	suite.RunErrorTests(t, patterns)

	TemplateComprehensiveHandlerTest(t, "DemoHandler", handler)
}

func TestRunPerformanceTest(t *testing.T) {
	var cleaned atomic.Bool

	pattern := PerformanceTestPattern{
		Name:        "FastOperation",
		Description: "should finish well within budget",
		MaxDuration: 25 * time.Millisecond,
		Setup: func(t *testing.T) interface{} {
			return "ok"
		},
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			if setupData.(string) != "ok" {
				t.Fatalf("unexpected setup data: %v", setupData)
			}
			start := time.Now()
			time.Sleep(5 * time.Millisecond)
			return time.Since(start)
		},
		Validate: func(t *testing.T, duration time.Duration, setupData interface{}) {
			if duration <= 0 {
				t.Fatalf("expected positive duration, got %v", duration)
			}
			if setupData.(string) != "ok" {
				t.Fatalf("unexpected setup data during validate: %v", setupData)
			}
		},
		Cleanup: func(interface{}) {
			cleaned.Store(true)
		},
	}

	RunPerformanceTest(t, pattern)

	if !cleaned.Load() {
		t.Fatal("expected cleanup to run")
	}
}

func TestRunConcurrencyTest(t *testing.T) {
	var executions atomic.Int32
	var cleaned atomic.Bool

	pattern := ConcurrencyTestPattern{
		Name:        "ConcurrentNoop",
		Description: "runs several iterations concurrently",
		Concurrency: 4,
		Iterations:  8,
		Setup: func(t *testing.T) interface{} {
			return "marker"
		},
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			if setupData.(string) != "marker" {
				t.Fatalf("unexpected setup data in execute: %v", setupData)
			}
			executions.Add(1)
			if iteration%3 == 0 {
				time.Sleep(1 * time.Millisecond)
			}
			return nil
		},
		Validate: func(t *testing.T, setupData interface{}, results []error) {
			if setupData.(string) != "marker" {
				t.Fatalf("unexpected setup data in validate: %v", setupData)
			}
			if len(results) != 8 {
				t.Fatalf("expected 8 results, got %d", len(results))
			}
			for i, err := range results {
				if err != nil {
					t.Fatalf("expected nil error at %d, got %v", i, err)
				}
			}
		},
		Cleanup: func(interface{}) {
			cleaned.Store(true)
		},
	}

	RunConcurrencyTest(t, pattern)

	if executions.Load() != 8 {
		t.Fatalf("expected 8 executions, got %d", executions.Load())
	}
	if !cleaned.Load() {
		t.Fatal("expected cleanup to run")
	}
}

func TestDecodePayloadVariants(t *testing.T) {
	cases := []struct {
		name     string
		payload  inputPayload
		wantData string
		wantErr  bool
	}{
		{
			name:     "PlainUTF8",
			payload:  inputPayload{Data: "hello", Encoding: "utf-8"},
			wantData: "hello",
		},
		{
			name:     "DefaultEncoding",
			payload:  inputPayload{Data: "world"},
			wantData: "world",
		},
		{
			name:     "Base64",
			payload:  inputPayload{Data: base64.StdEncoding.EncodeToString([]byte("binary")), Encoding: "base64"},
			wantData: "binary",
		},
		{
			name:    "InvalidBase64",
			payload: inputPayload{Data: "!not base64!", Encoding: "base64"},
			wantErr: true,
		},
		{
			name:    "UnsupportedEncoding",
			payload: inputPayload{Data: "noop", Encoding: "utf-16"},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := decodePayload(tc.payload)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(data) != tc.wantData {
				t.Fatalf("unexpected payload: got %q want %q", string(data), tc.wantData)
			}
		})
	}
}

func TestWSClientEnqueueDropsOldest(t *testing.T) {
	client := &wsClient{send: make(chan websocketEnvelope, 1)}

	client.enqueue(websocketEnvelope{Type: "first"})
	client.enqueue(websocketEnvelope{Type: "second"})

	select {
	case msg := <-client.send:
		if msg.Type != "second" {
			t.Fatalf("expected 'second', got %q", msg.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestNewWSClientInitialisesDefaults(t *testing.T) {
	s := &session{}
	client := newWSClient(s, nil)

	if client.session != s {
		t.Fatal("expected session to be assigned")
	}
	if client.conn != nil {
		t.Fatal("expected conn to be nil in test setup")
	}
	if len(client.send) != 0 {
		t.Fatalf("expected empty send channel, got len=%d", len(client.send))
	}
	if client.id == "" {
		t.Fatal("expected client id to be populated")
	}
}
