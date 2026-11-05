package aisearch

import (
	"strings"
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
)

func TestComposeBacklogText_Full(t *testing.T) {
	archived := "2026-04-01T00:00:00Z"
	item := backlog.BacklogItem{
		Name:        "retry-semantics",
		Title:       "Retry Semantics for Jobs",
		Description: "Make job retries idempotent and observable.",
		Tags:        []string{"reliability", "observability"},
		Kind:        backlog.KindExecute,
		Status:      backlog.StatusReady,
		Milestone:   "observability-core",
		Effort:      "M",
		DependsOn:   []string{"idea/tracing", "fix/metrics-leak"},
		Note:        "Coordinated with SRE; blocked on metrics pipeline refactor.",
		ArchivedAt:  &archived,
	}

	got := composeBacklogText(item)
	wantSubstrings := []string{
		"Retry Semantics for Jobs",
		"Make job retries idempotent and observable.",
		"Tags: reliability, observability",
		"Kind: execute",
		"Status: ready",
		"Milestone: observability-core",
		"Effort: M",
		"Dependencies: idea/tracing, fix/metrics-leak",
		"Note: Coordinated with SRE",
	}
	for _, s := range wantSubstrings {
		if !strings.Contains(got, s) {
			t.Errorf("expected composition to contain %q, got:\n%s", s, got)
		}
	}
}

func TestComposeBacklogText_EmptyFields(t *testing.T) {
	item := backlog.BacklogItem{Title: "Just a title"}
	got := composeBacklogText(item)
	if got != "Just a title" {
		t.Errorf("expected only title, got %q", got)
	}
}

func TestComposeBacklogText_TruncatesLongNote(t *testing.T) {
	item := backlog.BacklogItem{
		Title: "T",
		Note:  strings.Repeat("x", 3000),
	}
	got := composeBacklogText(item)
	if len(got) > 2200 {
		t.Errorf("expected note truncated, total length %d", len(got))
	}
	if !strings.Contains(got, "...") {
		t.Error("expected truncated note to contain ellipsis")
	}
}

func TestComposeGoalText_Full(t *testing.T) {
	archived := "2026-04-01T00:00:00Z"
	goal := goals.Goal{
		Name:        "observability-core",
		Title:       "Observability Core",
		Description: "Foundational tracing, metrics, logging.",
		Status:      "active",
		Targets:     []string{"execute/retry-semantics", "fix/metrics-leak"},
		ArchivedAt:  &archived,
	}
	got := composeGoalText(goal)
	wantSubstrings := []string{
		"Observability Core",
		"Description: Foundational tracing, metrics, logging.",
		"Status: active",
		"Targets: execute/retry-semantics, fix/metrics-leak",
	}
	for _, s := range wantSubstrings {
		if !strings.Contains(got, s) {
			t.Errorf("expected composition to contain %q, got:\n%s", s, got)
		}
	}
}

func TestBuildBacklogPayload(t *testing.T) {
	archived := "2026-04-01T00:00:00Z"
	item := backlog.BacklogItem{
		Name:       "retry-semantics",
		Title:      "Retry",
		Kind:       backlog.KindExecute,
		Status:     backlog.StatusReady,
		Priority:   7,
		Tags:       []string{"reliability"},
		Milestone:  "observability-core",
		Effort:     "M",
		ArchivedAt: &archived,
	}
	p := buildBacklogPayload(item, "")
	if p["name"] != "retry-semantics" {
		t.Errorf("name: %v", p["name"])
	}
	if p["kind"] != "execute" {
		t.Errorf("kind: %v", p["kind"])
	}
	if p["status"] != "ready" {
		t.Errorf("status: %v", p["status"])
	}
	if p["priority"] != 7 {
		t.Errorf("priority: %v", p["priority"])
	}
	if p["archived"] != true {
		t.Errorf("archived: %v", p["archived"])
	}
	tags, ok := p["tags"].([]string)
	if !ok || len(tags) != 1 || tags[0] != "reliability" {
		t.Errorf("tags: %v", p["tags"])
	}
}

func TestBuildBacklogPayload_NilTagsNormalized(t *testing.T) {
	item := backlog.BacklogItem{Name: "n", Kind: backlog.KindIdea}
	p := buildBacklogPayload(item, "")
	tags, ok := p["tags"].([]string)
	if !ok {
		t.Fatalf("expected tags as []string, got %T", p["tags"])
	}
	if len(tags) != 0 {
		t.Errorf("expected empty tags slice, got %v", tags)
	}
	if p["archived"] != false {
		t.Errorf("archived: %v", p["archived"])
	}
	scenarios, ok := p["target_scenarios"].([]string)
	if !ok {
		t.Fatalf("expected target_scenarios as []string, got %T", p["target_scenarios"])
	}
	if len(scenarios) != 0 {
		t.Errorf("expected empty target_scenarios for item without acceptance globs, got %v", scenarios)
	}
}

func TestBuildBacklogPayload_TargetScenariosFromAcceptanceAllow(t *testing.T) {
	item := backlog.BacklogItem{
		Name:            "fix-foo",
		Kind:            backlog.KindFix,
		AcceptanceAllow: []string{"scenarios/web-console/**", "scenarios/command-center/**"},
	}
	p := buildBacklogPayload(item, "")
	scenarios, ok := p["target_scenarios"].([]string)
	if !ok {
		t.Fatalf("expected target_scenarios as []string, got %T", p["target_scenarios"])
	}
	if len(scenarios) != 2 {
		t.Fatalf("expected 2 target scenarios, got %v", scenarios)
	}
	want := map[string]bool{"web-console": true, "command-center": true}
	for _, s := range scenarios {
		if !want[s] {
			t.Errorf("unexpected scenario in payload: %s", s)
		}
	}
}

func TestBuildGoalPayload(t *testing.T) {
	goal := goals.Goal{
		Name:     "obs-core",
		Title:    "Observability Core",
		Status:   "active",
		Priority: 3,
	}
	p := buildGoalPayload(goal, "")
	if p["name"] != "obs-core" || p["title"] != "Observability Core" {
		t.Errorf("unexpected payload: %+v", p)
	}
	if p["priority"] != 3 {
		t.Errorf("priority: %v", p["priority"])
	}
	if p["archived"] != false {
		t.Errorf("archived: %v", p["archived"])
	}
}

func TestBacklogPointID_Deterministic(t *testing.T) {
	a := backlogPointID(backlog.KindExecute, "retry-semantics")
	b := backlogPointID(backlog.KindExecute, "retry-semantics")
	if a != b {
		t.Errorf("expected deterministic UUIDv5, got %s vs %s", a, b)
	}
	if len(a) != 36 {
		t.Errorf("expected 36-char UUID, got %s (len=%d)", a, len(a))
	}
}

func TestBacklogPointID_DiffersByKind(t *testing.T) {
	a := backlogPointID(backlog.KindExecute, "x")
	b := backlogPointID(backlog.KindIdea, "x")
	if a == b {
		t.Error("expected point IDs to differ by kind")
	}
}

func TestGoalPointID_DiffersFromBacklog(t *testing.T) {
	a := goalPointID("x")
	b := backlogPointID(backlog.KindExecute, "x")
	if a == b {
		t.Error("expected goal and backlog namespaces to differ")
	}
}

func TestUUIDv5_KnownVector(t *testing.T) {
	// Sanity check against a second call producing the same value.
	got1 := uuidV5(qdrantNamespace, "swarm-manager:execute/alpha")
	got2 := uuidV5(qdrantNamespace, "swarm-manager:execute/alpha")
	if got1 != got2 {
		t.Fatalf("expected determinism, got %s vs %s", got1, got2)
	}
	// Version nibble should be 5 (position 14 in hex, 0-indexed).
	if got1[14] != '5' {
		t.Errorf("expected UUIDv5 version nibble '5', got %q in %s", got1[14], got1)
	}
}

// ---- composePayloadHash and payload_hash field tests ----

func TestComposePayloadHash_Deterministic(t *testing.T) {
	payload := map[string]interface{}{"name": "x", "kind": "fix", "priority": 3}
	h1 := composePayloadHash("hello world", payload)
	h2 := composePayloadHash("hello world", payload)
	if h1 != h2 {
		t.Errorf("expected deterministic hash, got %q vs %q", h1, h2)
	}
}

func TestComposePayloadHash_TextChangeDetected(t *testing.T) {
	payload := map[string]interface{}{"name": "x"}
	h1 := composePayloadHash("alpha", payload)
	h2 := composePayloadHash("beta", payload)
	if h1 == h2 {
		t.Errorf("expected different hashes for different text, both got %q", h1)
	}
}

func TestComposePayloadHash_PayloadFieldChangeDetected(t *testing.T) {
	text := "stable text"
	h1 := composePayloadHash(text, map[string]interface{}{"archived": false})
	h2 := composePayloadHash(text, map[string]interface{}{"archived": true})
	if h1 == h2 {
		t.Errorf("expected different hashes for different archived flag, both got %q", h1)
	}
}

func TestComposePayloadHash_PrefixIsSha256(t *testing.T) {
	h := composePayloadHash("anything", map[string]interface{}{"a": 1})
	if !strings.HasPrefix(h, "sha256:") {
		t.Errorf("expected hash to start with 'sha256:', got %q", h)
	}
	// "sha256:" + 16 hex chars = 23 chars. Pin the contract so accidental width
	// changes (e.g. someone bumps sum[:8] to sum[:16]) trip a test, not a runtime
	// surprise during reconcile diffs.
	if len(h) != len("sha256:")+16 {
		t.Errorf("expected hash width %d, got %d (%q)", len("sha256:")+16, len(h), h)
	}
}

func TestComposePayloadHash_KeyOrderInsensitive(t *testing.T) {
	// json.Marshal sorts map keys, so the same logical payload built in
	// different insertion orders must produce the same hash. Pinning this here
	// because Go map iteration order is randomized — without sorted keys, this
	// would be intermittent.
	a := map[string]interface{}{"name": "x", "priority": 1, "archived": false}
	b := map[string]interface{}{"archived": false, "priority": 1, "name": "x"}
	if composePayloadHash("t", a) != composePayloadHash("t", b) {
		t.Error("expected key order to be irrelevant to payload hash")
	}
}

func TestBuildBacklogPayload_IncludesPayloadHash(t *testing.T) {
	item := backlog.BacklogItem{Kind: backlog.KindFix, Name: "x"}
	out := buildBacklogPayload(item, "sha256:deadbeef00000000")
	if out["payload_hash"] != "sha256:deadbeef00000000" {
		t.Errorf("expected payload_hash field set, got %v", out["payload_hash"])
	}
}

func TestBuildBacklogPayload_OmitsHashWhenEmpty(t *testing.T) {
	item := backlog.BacklogItem{Kind: backlog.KindFix, Name: "x"}
	out := buildBacklogPayload(item, "")
	if _, present := out["payload_hash"]; present {
		t.Error("expected payload_hash to be absent when empty (so composePayloadHash sees a clean payload)")
	}
}

func TestBuildGoalPayload_IncludesPayloadHash(t *testing.T) {
	goal := goals.Goal{Name: "obs"}
	out := buildGoalPayload(goal, "sha256:cafebabe00000000")
	if out["payload_hash"] != "sha256:cafebabe00000000" {
		t.Errorf("expected payload_hash field set, got %v", out["payload_hash"])
	}
}

func TestBuildGoalPayload_OmitsHashWhenEmpty(t *testing.T) {
	goal := goals.Goal{Name: "obs"}
	out := buildGoalPayload(goal, "")
	if _, present := out["payload_hash"]; present {
		t.Error("expected payload_hash to be absent when empty")
	}
}
