package runner_test

import (
	"sort"
	"strings"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/rolepolicy"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// TestRunnerTypeTriSourceConformance is the RunnerType drift gate required by
// the grok-runner plan (Phase 3). The set of valid runner types is declared in
// THREE places that must never diverge:
//
//  1. Go      — domain.ValidRunnerTypes()
//  2. proto   — the generated RunnerType enum (domainpb.RunnerType_value)
//  3. JSON    — runner candidates in config/role-policy-catalog.json
//
// Adding a runner (e.g. grok) must touch all three; this test fails loudly if
// one is forgotten, making the Phase-4 enum addition self-checking.
//
// The proto enum names (RUNNER_TYPE_CLAUDE_CODE) map to the runner value
// strings ("claude-code") by stripping the RUNNER_TYPE_ prefix, lowercasing,
// and converting underscores to hyphens. RUNNER_TYPE_UNSPECIFIED (the zero
// value) is not a real runner and is excluded.
func TestRunnerTypeTriSourceConformance(t *testing.T) {
	goSet := make(map[string]struct{})
	for _, rt := range domain.ValidRunnerTypes() {
		goSet[string(rt)] = struct{}{}
	}

	protoSet := make(map[string]struct{})
	for name, value := range domainpb.RunnerType_value {
		if value == 0 {
			continue // RUNNER_TYPE_UNSPECIFIED — not a real runner
		}
		protoSet[protoEnumToRunnerValue(name)] = struct{}{}
	}

	revision, err := rolepolicy.Load(rolepolicy.ResolvePath())
	if err != nil {
		t.Fatalf("load role policy catalog: %v", err)
	}
	jsonSet := make(map[string]struct{})
	for _, role := range revision.Catalog().Roles {
		for _, candidate := range role.Candidates {
			jsonSet[string(candidate.Runner)] = struct{}{}
		}
	}

	if diff := setDiff(goSet, protoSet); diff != "" {
		t.Errorf("Go vs proto RunnerType sets diverge:\n%s", diff)
	}
	if diff := setDiff(goSet, jsonSet); diff != "" {
		t.Errorf("Go vs role-policy-catalog.json RunnerType sets diverge:\n%s", diff)
	}
}

// protoEnumToRunnerValue converts RUNNER_TYPE_CLAUDE_CODE → "claude-code".
func protoEnumToRunnerValue(name string) string {
	trimmed := strings.TrimPrefix(name, "RUNNER_TYPE_")
	return strings.ReplaceAll(strings.ToLower(trimmed), "_", "-")
}

// setDiff returns a human-readable description of the symmetric difference of
// two string sets, or "" when they are identical.
func setDiff(a, b map[string]struct{}) string {
	var onlyA, onlyB []string
	for k := range a {
		if _, ok := b[k]; !ok {
			onlyA = append(onlyA, k)
		}
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			onlyB = append(onlyB, k)
		}
	}
	if len(onlyA) == 0 && len(onlyB) == 0 {
		return ""
	}
	sort.Strings(onlyA)
	sort.Strings(onlyB)
	var sb strings.Builder
	if len(onlyA) > 0 {
		sb.WriteString("  only in first: " + strings.Join(onlyA, ", ") + "\n")
	}
	if len(onlyB) > 0 {
		sb.WriteString("  only in second: " + strings.Join(onlyB, ", ") + "\n")
	}
	return sb.String()
}
