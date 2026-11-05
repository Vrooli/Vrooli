package planmodel

import (
	"reflect"
	"testing"
)

func TestChangeBoundaryNormalized(t *testing.T) {
	b := ChangeBoundary{
		AcceptanceAllow:    []string{"  scenarios/plan-manager/**  ", `packages\proto\**`, "scenarios/plan-manager/**", "./docs/**", ""},
		AcceptanceDeny:     []string{"scenarios/swarm-manager/**", "scenarios/swarm-manager/**"},
		OperatorOnlyReason: "  ",
	}
	got := b.Normalized()
	wantAllow := []string{"docs/**", "packages/proto/**", "scenarios/plan-manager/**"}
	if !reflect.DeepEqual(got.AcceptanceAllow, wantAllow) {
		t.Fatalf("allow normalized = %v, want %v", got.AcceptanceAllow, wantAllow)
	}
	wantDeny := []string{"scenarios/swarm-manager/**"}
	if !reflect.DeepEqual(got.AcceptanceDeny, wantDeny) {
		t.Fatalf("deny normalized = %v, want %v", got.AcceptanceDeny, wantDeny)
	}
	if got.OperatorOnlyReason != "" {
		t.Fatalf("operator reason = %q, want empty", got.OperatorOnlyReason)
	}
}

func TestChangeBoundaryAffectedScenariosAndRepoPaths(t *testing.T) {
	cases := []struct {
		name      string
		allow     []string
		scenarios []string
		repo      []string
	}{
		{
			name:      "single scenario",
			allow:     []string{"scenarios/plan-manager/**"},
			scenarios: []string{"plan-manager"},
			repo:      nil,
		},
		{
			name:      "multi scenario sorted",
			allow:     []string{"scenarios/swarm-manager/**", "scenarios/plan-manager/api/**"},
			scenarios: []string{"plan-manager", "swarm-manager"},
			repo:      nil,
		},
		{
			name:      "mixed scenario and repo paths",
			allow:     []string{"scenarios/plan-manager/**", "packages/proto/**", "docs/**"},
			scenarios: []string{"plan-manager"},
			repo:      []string{"docs/**", "packages/proto/**"},
		},
		{
			name:      "non-scenario only",
			allow:     []string{"packages/proto/**", "Makefile"},
			scenarios: nil,
			repo:      []string{"Makefile", "packages/proto/**"},
		},
		{
			name:      "bare scenarios glob is not a single scenario",
			allow:     []string{"scenarios/**"},
			scenarios: nil,
			repo:      []string{"scenarios/**"},
		},
		{
			name:      "exact scenario dir",
			allow:     []string{"scenarios/plan-manager"},
			scenarios: []string{"plan-manager"},
			repo:      nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := ChangeBoundary{AcceptanceAllow: tc.allow}
			if got := b.AffectedScenarios(); !reflect.DeepEqual(got, tc.scenarios) {
				t.Errorf("AffectedScenarios() = %v, want %v", got, tc.scenarios)
			}
			if got := b.RepoPaths(); !reflect.DeepEqual(got, tc.repo) {
				t.Errorf("RepoPaths() = %v, want %v", got, tc.repo)
			}
		})
	}
}

func TestValidateBoundaryPlaceholders(t *testing.T) {
	b := ChangeBoundary{
		AcceptanceAllow: []string{"scenarios/<scenario>/**", "packages/proto/**"},
		AcceptanceDeny:  []string{"<allowed path>"},
	}
	problems := ValidateBoundary(b, true)
	if len(problems) != 2 {
		t.Fatalf("expected 2 placeholder problems, got %d: %v", len(problems), problems)
	}
}

func TestValidateBoundaryRequireAllow(t *testing.T) {
	// Empty boundary with requireAllow and no operator reason is invalid.
	if got := ValidateBoundary(ChangeBoundary{}, true); len(got) == 0 {
		t.Fatal("expected require-allow violation for empty boundary")
	}
	// Operator-only reason satisfies the allow requirement.
	if got := ValidateBoundary(ChangeBoundary{OperatorOnlyReason: "docs-only operator decision"}, true); len(got) != 0 {
		t.Fatalf("operator-only reason should satisfy allow requirement, got %v", got)
	}
	// Non-finalizing reader (requireAllow=false) tolerates empty boundary.
	if got := ValidateBoundary(ChangeBoundary{}, false); len(got) != 0 {
		t.Fatalf("non-finalizing validate should tolerate empty boundary, got %v", got)
	}
}

func TestValidateBoundaryDenyOverlap(t *testing.T) {
	b := ChangeBoundary{
		AcceptanceAllow: []string{"scenarios/plan-manager/**"},
		AcceptanceDeny:  []string{"scenarios/plan-manager/**"},
	}
	problems := ValidateBoundary(b, true)
	found := false
	for _, p := range problems {
		if contains(p, "also appears in acceptance_allow") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected deny/allow overlap problem, got %v", problems)
	}
}

func TestBoundaryAnchorCommands(t *testing.T) {
	b := ChangeBoundary{
		AcceptanceAllow: []string{"scenarios/plan-manager/**", "packages/proto/**", "docs/**"},
	}
	commands, informational := BoundaryAnchorCommands(b, "plan-manager-baseline", "abc123")
	// One scenario => 2 oracle commands; one repo diff (informational).
	if len(commands) != 3 {
		t.Fatalf("expected 3 commands, got %d: %v", len(commands), commands)
	}
	if len(informational) != 1 {
		t.Fatalf("expected 1 informational command, got %d: %v", len(informational), informational)
	}
	wantDiff := "git diff --stat abc123 -- docs/** packages/proto/**"
	if informational[0] != wantDiff {
		t.Fatalf("informational diff = %q, want %q", informational[0], wantDiff)
	}
}

func TestBoundaryAnchorCommandsNoBaselineName(t *testing.T) {
	b := ChangeBoundary{AcceptanceAllow: []string{"scenarios/plan-manager/**"}}
	// No safe baseline name => no fabricated oracle command.
	commands, informational := BoundaryAnchorCommands(b, "", "")
	if len(commands) != 0 || len(informational) != 0 {
		t.Fatalf("expected no commands without a baseline name, got %v / %v", commands, informational)
	}
}

func TestUnresolvedPlaceholders(t *testing.T) {
	got := UnresolvedPlaceholders("scenarios/<scenario>/** and <branch> and <scenario>")
	want := []string{"<scenario>", "<branch>"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("UnresolvedPlaceholders = %v, want %v", got, want)
	}
	if UnresolvedPlaceholders("scenarios/plan-manager/**") != nil {
		t.Fatal("expected nil for clean path")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

func TestBoundaryFromLegacyAnchor(t *testing.T) {
	// scenario_baseline anchor => allow scenarios/<scenario>/**.
	b := BoundaryFromLegacyAnchor(RegressionAnchor{Strategy: AnchorStrategyScenarioBaseline, Scenario: "plan-manager"})
	if !reflect.DeepEqual(b.AcceptanceAllow, []string{"scenarios/plan-manager/**"}) {
		t.Fatalf("scenario anchor => %v", b.AcceptanceAllow)
	}

	// head_sha_allowlist anchor => allowlist verbatim, bare scenario dir expanded.
	b = BoundaryFromLegacyAnchor(RegressionAnchor{
		Strategy:       AnchorStrategyHeadShaAllowlist,
		AllowlistPaths: []string{"scenarios/foo", "packages/proto/**"},
	})
	if !reflect.DeepEqual(b.AcceptanceAllow, []string{"packages/proto/**", "scenarios/foo/**"}) {
		t.Fatalf("allowlist anchor => %v", b.AcceptanceAllow)
	}

	// Unstructured/legacy-prose anchor => zero boundary (nothing safe to derive).
	if got := BoundaryFromLegacyAnchor(RegressionAnchor{Strategy: AnchorStrategyLegacyProse, BaselineName: "some prose"}); !got.IsZero() {
		t.Fatalf("legacy-prose anchor should yield zero boundary, got %v", got)
	}

	// A placeholder scenario is not derived.
	if got := BoundaryFromLegacyAnchor(RegressionAnchor{Scenario: "<scenario>"}); !got.IsZero() {
		t.Fatalf("placeholder scenario must not derive a boundary, got %v", got)
	}
}

// TestLegacyMarkdownImportDerivesBoundary proves a pre-cutover plan with a
// scenario-baseline anchor and no Change Boundary section gains a derived boundary
// on import.
func TestLegacyMarkdownImportDerivesBoundary(t *testing.T) {
	md := "# Legacy plan\n\n## Purpose\n\nDo a thing.\n\n## Regression Anchor\n\n" +
		"- Strategy: scenario_baseline\n- Scenario baseline: `plan-manager` (name `legacy-baseline`)\n\n" +
		"### Phase 1 — Work\n- Intent: work\n"
	p, err := ParsePlanMarkdown(md)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !reflect.DeepEqual(p.ChangeBoundary.AcceptanceAllow, []string{"scenarios/plan-manager/**"}) {
		t.Fatalf("imported boundary = %v, want [scenarios/plan-manager/**]", p.ChangeBoundary.AcceptanceAllow)
	}
}

// TestSwarmManagerBoundaryMapsDirectly proves a Swarm Manager-style
// acceptance_allow / acceptance_deny set maps directly into a Plan Manager
// boundary with identical affected-scenario derivation (the consumer-inversion
// contract — no vocabulary translation).
func TestSwarmManagerBoundaryMapsDirectly(t *testing.T) {
	// What a swarm-manager backlog item carries.
	swarmAllow := []string{"scenarios/image-tools/**", "packages/ai-go/**"}
	swarmDeny := []string{"scenarios/swarm-manager/**"}

	b := ChangeBoundary{AcceptanceAllow: swarmAllow, AcceptanceDeny: swarmDeny}.Normalized()
	if !reflect.DeepEqual(b.AffectedScenarios(), []string{"image-tools"}) {
		t.Fatalf("affected scenarios = %v, want [image-tools]", b.AffectedScenarios())
	}
	if !reflect.DeepEqual(b.RepoPaths(), []string{"packages/ai-go/**"}) {
		t.Fatalf("repo paths = %v", b.RepoPaths())
	}
	if !reflect.DeepEqual(b.AcceptanceDeny, swarmDeny) {
		t.Fatalf("deny not preserved verbatim: %v", b.AcceptanceDeny)
	}
}
