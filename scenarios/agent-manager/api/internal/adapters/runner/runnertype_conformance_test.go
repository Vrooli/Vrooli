package runner_test

import (
	"sort"
	"strings"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/rolepolicy"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// TestRunnerTypeTriSourceConformance keeps the Go/proto runner vocabulary in
// agreement and checks that operator policy selects only supported runners.
// A policy is a selected subset, not a registry: requiring every supported
// runner would force unwanted cross-provider fallbacks into operator policy.
// Catalog.Load additionally validates unknown runner identities fail closed.
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
	for _, role := range revision.Catalog().Roles {
		for _, candidate := range role.Candidates {
			if _, known := goSet[string(candidate.Runner)]; !known {
				t.Errorf("role policy selects unsupported runner %q", candidate.Runner)
			}
		}
	}

	// Selecting a subset must not relax validation of unknown candidates.
	invalid := revision.Catalog().Clone()
	role := invalid.Roles[invalid.DefaultRole]
	role.Candidates[0].Runner = domain.RunnerType("not-a-runner")
	invalid.Roles[invalid.DefaultRole] = role
	if err := invalid.Validate(); err == nil {
		t.Fatal("unknown runner passed policy validation")
	}

	if diff := setDiff(goSet, protoSet); diff != "" {
		t.Errorf("Go vs proto RunnerType sets diverge:\n%s", diff)
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
