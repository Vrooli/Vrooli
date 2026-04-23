package backlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/testutil"
)

func TestCreate_Success(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	payload := map[string]any{
		"name":        "New Test Idea",
		"title":       "New Test Idea",
		"description": "A new test idea",
		"priority":    3,
		"tags":        []string{"new", "test"},
		"kind":        "idea",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	result := resp.Item

	if result.Name != "new-test-idea" {
		t.Errorf("expected sanitized name 'new-test-idea', got '%s'", result.Name)
	}
	if result.Status != StatusBacklog {
		t.Errorf("expected status 'backlog', got '%s'", result.Status)
	}
	if result.Kind != KindIdea {
		t.Errorf("expected kind 'idea', got '%s'", result.Kind)
	}

	specPath := filepath.Join(rootDir, "ideas", "new-test-idea", "spec.json")
	testutil.AssertFileExists(t, specPath)
}

func TestCreate_RejectsUnknownField(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	req := httptest.NewRequest("POST", "/api/v1/backlog", strings.NewReader(`{
		"name": "new-test-idea",
		"title": "New Test Idea",
		"kind": "idea",
		"scope": "scenarios/swarm-manager"
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)

	testutil.AssertStatusBadRequest(t, w)
	if !strings.Contains(w.Body.String(), "invalid request body") {
		t.Fatalf("expected invalid request body error, got: %s", w.Body.String())
	}

	testutil.AssertFileNotExists(t, filepath.Join(rootDir, "ideas", "new-test-idea", "spec.json"))
}

// ---------------------------------------------------------------------------
// Auto-initialize workshop on Create
// ---------------------------------------------------------------------------

func TestCreate_AutoInitializesWorkshop(t *testing.T) {
	spawned := make(chan struct{})
	agent := &mockAgentService{
		result:   agentmanager.RunResult{RunID: "run-auto", TaskID: "task-auto"},
		spawnedC: spawned,
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	// Re-enable auto-initialize for this test.
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "config", "settings.json"), map[string]any{
		"theme":                    "dark",
		"default_mode":             "manual",
		"max_auto_rounds":          10,
		"auto_initialize_workshop": true,
		"auto_advance_workshop":    true,
		"auto_cascade_workshop":    true,
		"agent_max_turns":          600,
		"agent_timeout_seconds":    900,
		"agent_requires_approval":  true,
		"search_debounce_ms":       300,
		"toast_duration_ms":        5000,
		"delete_confirmation":      map[string]any{"backlog": "simple", "initiative": "strong", "capture": "none"},
	})

	payload := map[string]any{
		"name":  "auto-init-test",
		"title": "Auto Init Test",
		"kind":  "idea",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Create(w, req)

	testutil.AssertStatusCreated(t, w)

	// Wait for the background goroutine to call SpawnBacklog.
	select {
	case <-spawned:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for auto-init spawn")
	}

	if agent.lastReq == nil {
		t.Fatal("expected agent spawn to be called")
	}
	if agent.lastReq.Name != "auto-init-test" {
		t.Errorf("expected spawn for 'auto-init-test', got %q", agent.lastReq.Name)
	}
}

func TestCreate_AutoInitializeDisabledViaSetting(t *testing.T) {
	agent := &mockAgentService{
		result: agentmanager.RunResult{RunID: "run-x", TaskID: "task-x"},
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	// Disable auto-initialize via settings.
	t.Setenv("SCENARIO_ROOT", rootDir)
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "config", "settings.json"), map[string]any{
		"theme":                    "dark",
		"default_mode":             "manual",
		"max_auto_rounds":          10,
		"auto_initialize_workshop": false,
		"auto_advance_workshop":    true,
		"auto_cascade_workshop":    true,
		"agent_max_turns":          600,
		"agent_timeout_seconds":    900,
		"agent_requires_approval":  true,
		"search_debounce_ms":       300,
		"toast_duration_ms":        5000,
		"delete_confirmation":      map[string]any{"backlog": "simple", "initiative": "strong", "capture": "none"},
	})

	payload := map[string]any{
		"name":  "no-auto-test",
		"title": "No Auto Test",
		"kind":  "idea",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Create(w, req)

	testutil.AssertStatusCreated(t, w)

	// Give a brief window for any goroutine to fire (it shouldn't).
	time.Sleep(100 * time.Millisecond)

	if agent.lastReq != nil {
		t.Error("expected NO agent spawn when auto_initialize_workshop is false")
	}
}

func TestCreate_AutoInit_AgentDown_StillCreates(t *testing.T) {
	spawned := make(chan struct{})
	agent := &mockAgentService{
		err:      fmt.Errorf("agent down"),
		spawnedC: spawned,
	}
	h, rootDir := setupTestHandlerWithAgent(t, agent)

	// Re-enable auto-initialize to test agent-down resilience.
	testutil.WriteJSONFile(t, filepath.Join(rootDir, "config", "settings.json"), map[string]any{
		"theme":                    "dark",
		"default_mode":             "manual",
		"max_auto_rounds":          10,
		"auto_initialize_workshop": true,
		"auto_advance_workshop":    true,
		"auto_cascade_workshop":    true,
		"agent_max_turns":          600,
		"agent_timeout_seconds":    900,
		"agent_requires_approval":  true,
		"search_debounce_ms":       300,
		"toast_duration_ms":        5000,
		"delete_confirmation":      map[string]any{"backlog": "simple", "initiative": "strong", "capture": "none"},
	})

	payload := map[string]any{
		"name":  "agent-down-test",
		"title": "Agent Down Test",
		"kind":  "fix",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Create(w, req)

	// Item should be created successfully regardless of agent error.
	testutil.AssertStatusCreated(t, w)

	specPath := filepath.Join(rootDir, "fix", "agent-down-test", "spec.json")
	testutil.AssertFileExists(t, specPath)

	// Wait for the goroutine to attempt spawn (and fail).
	select {
	case <-spawned:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for auto-init attempt")
	}
}

func TestCreate_WithEffort(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	payload := map[string]any{
		"name":     "effort-test",
		"title":    "Effort Test",
		"kind":     "idea",
		"effort":   "L",
		"priority": 3,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if resp.Item.Effort != "L" {
		t.Errorf("expected effort 'L', got %q", resp.Item.Effort)
	}

	// Verify persisted to disk.
	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "ideas", "effort-test", "spec.json"))
	if saved.Effort != "L" {
		t.Errorf("expected saved effort 'L', got %q", saved.Effort)
	}
}

func TestCreate_EffortNormalizesCase(t *testing.T) {
	h, _ := setupTestHandler(t)

	payload := map[string]any{
		"name":   "effort-case-test",
		"title":  "Effort Case Test",
		"kind":   "fix",
		"effort": "xl",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if resp.Item.Effort != "XL" {
		t.Errorf("expected effort 'XL', got %q", resp.Item.Effort)
	}
}

func TestCreate_InvalidEffort(t *testing.T) {
	h, _ := setupTestHandler(t)

	payload := map[string]any{
		"name":   "bad-effort",
		"title":  "Bad Effort",
		"kind":   "idea",
		"effort": "HUGE",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	testutil.AssertStatusBadRequest(t, w)
}

func TestCreate_EffortOptional(t *testing.T) {
	h, _ := setupTestHandler(t)

	payload := map[string]any{
		"name":  "no-effort",
		"title": "No Effort",
		"kind":  "idea",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if resp.Item.Effort != "" {
		t.Errorf("expected empty effort, got %q", resp.Item.Effort)
	}
}

func TestCreate_WithAcceptanceGlobs(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	payload := map[string]any{
		"name":             "globs-test",
		"title":            "Globs Test",
		"kind":             "fix",
		"acceptance_allow": []string{"api/**", "*.go"},
		"acceptance_deny":  []string{"vendor/**"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if len(resp.Item.AcceptanceAllow) != 2 {
		t.Errorf("expected 2 allow globs, got %d", len(resp.Item.AcceptanceAllow))
	}
	if len(resp.Item.AcceptanceDeny) != 1 {
		t.Errorf("expected 1 deny glob, got %d", len(resp.Item.AcceptanceDeny))
	}

	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "fix", "globs-test", "spec.json"))
	if len(saved.AcceptanceAllow) != 2 {
		t.Errorf("expected 2 saved allow globs, got %d", len(saved.AcceptanceAllow))
	}
}

func TestCreate_WithSpawnedFrom(t *testing.T) {
	h, rootDir := setupTestHandler(t)

	payload := map[string]any{
		"name":         "spawned-item",
		"title":        "Spawned Item",
		"kind":         "execute",
		"spawned_from": "research/agent-identity-standard",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/backlog", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, req)
	testutil.AssertStatusCreated(t, w)

	resp := testutil.DecodeJSON[backlogItemResponse](t, w)
	if resp.Item.SpawnedFrom != "research/agent-identity-standard" {
		t.Errorf("expected spawned_from 'research/agent-identity-standard', got %q", resp.Item.SpawnedFrom)
	}

	saved := testutil.ReadJSONFile[BacklogItem](t, filepath.Join(rootDir, "execute", "spawned-item", "spec.json"))
	if saved.SpawnedFrom != "research/agent-identity-standard" {
		t.Errorf("expected saved spawned_from 'research/agent-identity-standard', got %q", saved.SpawnedFrom)
	}
}
