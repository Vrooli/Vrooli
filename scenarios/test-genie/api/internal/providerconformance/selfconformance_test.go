package providerconformance

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
)

// repoRootFromTest resolves the repository root from this test file's location so
// the self-conformance check does not depend on the working directory.
func repoRootFromTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// .../scenarios/test-genie/api/internal/providerconformance/<file>
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
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
	for _, code := range []string{CodeNorthStarMissing, CodeLadderIncomplete, CodeDocsSkeletonIncomplete} {
		requireNoCode(t, report, code)
	}
}
