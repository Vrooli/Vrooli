package runtime

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
)

func newSilentLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logger.SetLevel(logrus.ErrorLevel)
	return logger
}

func TestSessionExecuteInstructionSuccess(t *testing.T) {
	t.Parallel()

	var (
		mu       sync.Mutex
		requests []map[string]any
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != browserlessFunctionPath {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request payload: %v", err)
		}

		mu.Lock()
		requests = append(requests, payload)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"success":true,"steps":[{"success":true,"nodeId":"node-1","type":"navigate","index":0,"durationMs":180,"finalUrl":"https://example.com"}],"totalDurationMs":180}`)
	}))
	defer server.Close()

	session := NewSession(server.URL, server.Client(), newSilentLogger())
	instruction := Instruction{
		Index:  0,
		NodeID: "node-1",
		Type:   "navigate",
		Params: InstructionParam{
			URL: "https://example.com",
		},
	}

	response, err := session.ExecuteInstruction(context.Background(), instruction)
	if err != nil {
		t.Fatalf("ExecuteInstruction returned error: %v", err)
	}
	if response == nil {
		t.Fatalf("expected response, got nil")
	}
	if !response.Success {
		t.Fatalf("expected success response, got %+v", response)
	}
	if len(response.Steps) != 1 {
		t.Fatalf("expected one step result, got %d", len(response.Steps))
	}
	step := response.Steps[0]
	if step.NodeID != "node-1" {
		t.Errorf("expected nodeId node-1, got %s", step.NodeID)
	}
	if step.DurationMs != 180 {
		t.Errorf("expected duration 180ms, got %d", step.DurationMs)
	}
	if step.FinalURL != "https://example.com" {
		t.Errorf("expected final URL to match request, got %s", step.FinalURL)
	}

	mu.Lock()
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, captured %d", len(requests))
	}
	captured := requests[0]
	mu.Unlock()

	code, ok := captured["code"].(string)
	if !ok || strings.TrimSpace(code) == "" {
		t.Fatalf("expected non-empty script code in payload, got %#v", captured["code"])
	}

	ctxPayload, ok := captured["context"].(map[string]any)
	if !ok {
		t.Fatalf("expected context map in payload, got %#v", captured["context"])
	}
	sessionID, ok := ctxPayload["sessionId"].(string)
	if !ok || sessionID == "" {
		t.Fatalf("expected sessionId in context payload, got %#v", ctxPayload["sessionId"])
	}
	if sessionID != session.SessionID() {
		t.Fatalf("expected sessionId %s, got %s", session.SessionID(), sessionID)
	}
}

func TestSessionExecuteInstructionHTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		io.WriteString(w, "upstream failure")
	}))
	defer server.Close()

	session := NewSession(server.URL, server.Client(), newSilentLogger())
	instruction := Instruction{
		Index:  0,
		NodeID: "node-err",
		Type:   "navigate",
		Params: InstructionParam{URL: "https://example.com"},
	}

	_, err := session.ExecuteInstruction(context.Background(), instruction)
	if err == nil {
		t.Fatalf("expected error when Browserless returns non-2xx status")
	}
	if !strings.Contains(err.Error(), "status") {
		t.Errorf("expected error to mention status code, got %v", err)
	}
}

func TestSessionExecuteInstructionInvalidJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, "not-json")
	}))
	defer server.Close()

	session := NewSession(server.URL, server.Client(), newSilentLogger())
	instruction := Instruction{
		Index:  0,
		NodeID: "node-json",
		Type:   "navigate",
		Params: InstructionParam{URL: "https://example.com"},
	}

	_, err := session.ExecuteInstruction(context.Background(), instruction)
	if err == nil {
		t.Fatalf("expected JSON decode error, got nil")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("expected decode error, got %v", err)
	}
}

func TestSessionHTMLCache(t *testing.T) {
	t.Parallel()

	session := NewSession("http://localhost", nil, newSilentLogger())
	if session.LastHTML() != "" {
		t.Fatalf("expected empty HTML cache by default")
	}

	session.SetLastHTML("<html>cached</html>")
	if session.LastHTML() != "<html>cached</html>" {
		t.Fatalf("expected LastHTML to return cached content")
	}
}
