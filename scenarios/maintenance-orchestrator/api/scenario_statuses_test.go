package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// stubCLIRunner backs the typed CLI client with fixed output/error so handler
// tests are deterministic instead of shelling the real `vrooli` binary.
type stubCLIRunner struct {
	out []byte
	err error
}

func (s stubCLIRunner) Run(context.Context, string, ...string) ([]byte, error) {
	return s.out, s.err
}

func (s stubCLIRunner) RunCombined(context.Context, string, ...string) ([]byte, error) {
	return s.out, s.err
}

// useStubCLI swaps the package cliClient for a stub-backed one for the test.
func useStubCLI(t *testing.T, out []byte, err error) {
	t.Helper()
	prev := cliClient
	cliClient = vroolicli.New(vroolicli.WithRunner(stubCLIRunner{out: out, err: err}))
	t.Cleanup(func() { cliClient = prev })
}

func TestHandleGetScenarioStatuses_TypedMapping(t *testing.T) {
	useStubCLI(t, []byte(`{"success":true,"scenarios":[
		{"name":"alpha","status":"running","processes":3},
		{"name":"beta","status":"ERROR","processes":0},
		{"name":"gamma","status":"stopped","processes":1},
		{"name":"","status":"running","processes":9}
	]}`), nil)

	w := httptest.NewRecorder()
	handleGetScenarioStatuses()(w, httptest.NewRequest("GET", "/api/v1/scenario-statuses", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Statuses map[string]struct {
			Status       string `json:"status"`
			ProcessCount int    `json:"processCount"`
		} `json:"statuses"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if len(resp.Statuses) != 3 {
		t.Fatalf("expected 3 statuses (blank name dropped), got %d: %+v", len(resp.Statuses), resp.Statuses)
	}
	if resp.Statuses["alpha"].Status != "running" || resp.Statuses["alpha"].ProcessCount != 3 {
		t.Errorf("alpha = %+v", resp.Statuses["alpha"])
	}
	if resp.Statuses["beta"].Status != "error" { // ERROR normalized to error
		t.Errorf("beta status = %q, want error", resp.Statuses["beta"].Status)
	}
	if resp.Statuses["gamma"].Status != "stopped" || resp.Statuses["gamma"].ProcessCount != 1 {
		t.Errorf("gamma = %+v", resp.Statuses["gamma"])
	}
}

// TestHandleGetScenarioStatuses_CLIError verifies a CLI failure surfaces as a
// 500 instead of a silent success — the typed client never degrades to empty.
func TestHandleGetScenarioStatuses_CLIError(t *testing.T) {
	useStubCLI(t, nil, errors.New("boom"))

	w := httptest.NewRecorder()
	handleGetScenarioStatuses()(w, httptest.NewRequest("GET", "/api/v1/scenario-statuses", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}
