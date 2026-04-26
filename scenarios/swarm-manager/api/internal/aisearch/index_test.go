package aisearch

import (
	"strings"
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
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
		Initiative:  "observability-core",
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
		"Initiative: observability-core",
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

func TestComposeInitiativeText_Full(t *testing.T) {
	archived := "2026-04-01T00:00:00Z"
	init := initiatives.Initiative{
		Name:        "observability-core",
		Title:       "Observability Core",
		Description: "Foundational tracing, metrics, logging.",
		Status:      "active",
		DependsOn:   []string{"data-platform"},
		Items:       []string{"execute/retry-semantics", "fix/metrics-leak"},
		Note:        "Quarterly priority.",
		ArchivedAt:  &archived,
	}
	got := composeInitiativeText(init)
	wantSubstrings := []string{
		"Observability Core",
		"Description: Foundational tracing, metrics, logging.",
		"Status: active",
		"Dependencies: data-platform",
		"Items: execute/retry-semantics, fix/metrics-leak",
		"Note: Quarterly priority.",
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
		Initiative: "observability-core",
		Effort:     "M",
		ArchivedAt: &archived,
	}
	p := buildBacklogPayload(item)
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
	p := buildBacklogPayload(item)
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
	p := buildBacklogPayload(item)
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

func TestBuildInitiativePayload(t *testing.T) {
	init := initiatives.Initiative{
		Name:     "obs-core",
		Title:    "Observability Core",
		Status:   "active",
		Priority: 3,
	}
	p := buildInitiativePayload(init)
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

func TestInitiativePointID_DiffersFromBacklog(t *testing.T) {
	a := initiativePointID("x")
	b := backlogPointID(backlog.KindExecute, "x")
	if a == b {
		t.Error("expected initiative and backlog namespaces to differ")
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
