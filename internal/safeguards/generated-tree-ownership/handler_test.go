package generatedtreeownership

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func newFixture(t *testing.T, owned []string) (h hostreqkit.Handler, commands *[][]string) {
	t.Helper()
	previousRoot, previousOwned, previousIDs, previousRun := repoRoot, rootOwned, invokingIDs, runPrivileged
	t.Cleanup(func() {
		repoRoot, rootOwned, invokingIDs, runPrivileged = previousRoot, previousOwned, previousIDs, previousRun
	})
	repoRoot = func() string { return t.TempDir() }
	rootOwned = func(string) ([]string, error) { return owned, nil }
	invokingIDs = func() (int, int) { return 1000, 1001 }
	recorded := &[][]string{}
	runPrivileged = func(_ string, name string, args []string, _ hostreqkit.EnsureOptions) error {
		*recorded = append(*recorded, append([]string{name}, args...))
		return nil
	}
	return NewHandler(hostreqkit.SafeguardManifest{Name: "generated_tree_ownership"}), recorded
}

func inspect(h hostreqkit.Handler) hostreqkit.ItemStatus {
	return h.Inspect(hostreqkit.Host{OS: "linux"}, hostreqspec.ResolvedRequirement{Name: "generated_tree_ownership", Kind: hostreqspec.KindSafeguard})
}

func TestRootOwnedTreeIsPendingThenHandedBackRecursively(t *testing.T) {
	tree := "/repo/packages/proto/gen.previous"
	h, commands := newFixture(t, []string{tree})
	status := inspect(h)
	if status.Applied {
		t.Fatalf("a root-owned tree must leave the safeguard unapplied: %+v", status)
	}
	applied, err := h.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{})
	if err != nil || !applied.Applied || applied.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("Apply() = %+v, %v", applied, err)
	}
	want := fmt.Sprint([][]string{{"chown", "-hR", "1000:1001", tree}})
	if got := fmt.Sprint(*commands); got != want {
		t.Fatalf("commands = %s, want %s", got, want)
	}
}

func TestNothingRootOwnedRunsNothing(t *testing.T) {
	h, commands := newFixture(t, nil)
	status := inspect(h)
	if !status.Applied || status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("status = %+v", status)
	}
	if _, err := h.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{}); err != nil || len(*commands) != 0 {
		t.Fatalf("Apply ran %v, err %v", *commands, err)
	}
}

func TestDryRunChangesNothing(t *testing.T) {
	h, commands := newFixture(t, []string{"/repo/packages/proto/gen"})
	got, err := h.Apply(hostreqkit.Host{OS: "linux"}, inspect(h), hostreqkit.EnsureOptions{DryRun: true})
	if err != nil || got.ExecutionState != hostreqkit.ExecutionWouldApply || len(*commands) != 0 {
		t.Fatalf("dry run = %+v, %v, commands %v", got, err, *commands)
	}
}

func TestRefusesToHandFilesToRoot(t *testing.T) {
	h, commands := newFixture(t, []string{"/repo/packages/proto/gen"})
	invokingIDs = func() (int, int) { return 0, 0 }
	got, err := h.Apply(hostreqkit.Host{OS: "linux"}, inspect(h), hostreqkit.EnsureOptions{})
	if err != nil || got.ExecutionState != hostreqkit.ExecutionFailed || len(*commands) != 0 {
		t.Fatalf("got %+v, %v, commands %v", got, err, *commands)
	}
}

// The scan must report only trees that hold a root-owned entry, and must not
// fail when a generated tree does not exist. An unprivileged test cannot create
// a root-owned file, so this pins the absent and operator-owned cases.
func TestRootOwnedTreesIgnoresAbsentAndOperatorOwnedTrees(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "packages/proto/gen/go"), 0o755); err != nil {
		t.Fatal(err)
	}
	if os.Geteuid() == 0 {
		t.Skip("a root test run owns every fixture file")
	}
	trees, err := rootOwnedTrees(root)
	if err != nil || len(trees) != 0 {
		t.Fatalf("rootOwnedTrees = %v, %v", trees, err)
	}
}
