package providerconformance

import (
	"context"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

// repoRootFromTest resolves through the repository contract so `-trimpath`
// cannot turn runtime source paths into package-relative false roots.
func repoRootFromTest(t *testing.T) string {
	t.Helper()
	root, err := repocontract.FindRepoRootFromCWD()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}

// TestTestGenieOwnDescriptorPassesContract is the recursion guard: Test Genie's
// own provider-conformance descriptor must satisfy the Phase Capability Contract
// it enforces on every other phase — a first-class North Star on each ladder, a
// next_unlock on every non-top rung, and a skeleton-conformant remediation doc.
// The live probe is skipped for the self target, so only the descriptor + doc
// checks run.
func TestTestGenieOwnDescriptorPassesContract(t *testing.T) {
	repoRoot := repoRootFromTest(t)
	report, err := New(repoRoot).ValidateScenario(context.Background(), "test-genie", "")
	if err != nil {
		t.Fatalf("ValidateScenario(test-genie): %v", err)
	}
	for _, code := range []string{CodeNorthStarMissing, CodeLadderIncomplete, CodeDocsSkeletonIncomplete, CodeRungUngated} {
		requireNoCode(t, report, code)
	}
}

// TestAgentManagerDescriptorHasGatedAgentConformanceLadder prevents the
// provider-owned Agent Manager phase from regressing to advisory maturity
// rungs after its fleet conformance gate has been activated.
func TestAgentManagerDescriptorHasGatedAgentConformanceLadder(t *testing.T) {
	repoRoot := repoRootFromTest(t)
	report, err := New(repoRoot).ValidateScenario(context.Background(), "agent-manager", "")
	if err != nil {
		t.Fatalf("ValidateScenario(agent-manager): %v", err)
	}
	requireNoCode(t, report, CodeRungUngated)
}
