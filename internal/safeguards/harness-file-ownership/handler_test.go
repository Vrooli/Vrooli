package harnessfileownership

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
	home := t.TempDir()
	previousHome, previousList, previousIDs, previousRun := homeDir, rootOwnedList, invokingIDs, runPrivileged
	t.Cleanup(func() {
		homeDir, rootOwnedList, invokingIDs, runPrivileged = previousHome, previousList, previousIDs, previousRun
	})
	homeDir = func() (string, error) { return home, nil }
	rootOwnedList = func(string) ([]string, error) { return owned, nil }
	invokingIDs = func() (int, int) { return 1000, 1001 }
	recorded := &[][]string{}
	runPrivileged = func(_ string, name string, args []string, _ hostreqkit.EnsureOptions) error {
		*recorded = append(*recorded, append([]string{name}, args...))
		return nil
	}
	return NewHandler(hostreqkit.SafeguardManifest{Name: "harness_file_ownership"}), recorded
}

func linuxHost() hostreqkit.Host { return hostreqkit.Host{OS: "linux"} }

func TestRootOwnedFileIsPendingThenHandedBackWithoutFollowingLinks(t *testing.T) {
	path := filepath.Join(os.TempDir(), "agy")
	h, commands := newFixture(t, []string{path})
	status := h.Inspect(linuxHost(), hostreqspec.ResolvedRequirement{Name: "harness_file_ownership", Kind: hostreqspec.KindSafeguard})
	if status.Applied {
		t.Fatalf("a root-owned file must leave the safeguard unapplied: %+v", status)
	}
	applied, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
	if err != nil || !applied.Applied || applied.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("Apply() = %+v, %v", applied, err)
	}
	want := fmt.Sprint([][]string{{"chown", "-h", "1000:1001", path}})
	if got := fmt.Sprint(*commands); got != want {
		t.Fatalf("commands = %s, want %s", got, want)
	}
}

func TestNothingRootOwnedIsAlreadyPresentAndRunsNothing(t *testing.T) {
	h, commands := newFixture(t, nil)
	status := h.Inspect(linuxHost(), hostreqspec.ResolvedRequirement{Name: "harness_file_ownership", Kind: hostreqspec.KindSafeguard})
	if !status.Applied || status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("status = %+v", status)
	}
	if _, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{}); err != nil || len(*commands) != 0 {
		t.Fatalf("Apply ran %v, err %v", *commands, err)
	}
}

func TestDryRunChangesNothing(t *testing.T) {
	h, commands := newFixture(t, []string{"/home/x/.local/bin/agy"})
	status := h.Inspect(linuxHost(), hostreqspec.ResolvedRequirement{Name: "harness_file_ownership", Kind: hostreqspec.KindSafeguard})
	got, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil || got.ExecutionState != hostreqkit.ExecutionWouldApply || len(*commands) != 0 {
		t.Fatalf("dry run = %+v, %v, commands %v", got, err, *commands)
	}
}

func TestRefusesToHandFilesToRoot(t *testing.T) {
	h, commands := newFixture(t, []string{"/home/x/.local/bin/agy"})
	invokingIDs = func() (int, int) { return 0, 0 }
	status := h.Inspect(linuxHost(), hostreqspec.ResolvedRequirement{Name: "harness_file_ownership", Kind: hostreqspec.KindSafeguard})
	got, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
	if err != nil || got.ExecutionState != hostreqkit.ExecutionFailed || len(*commands) != 0 {
		t.Fatalf("got %+v, %v, commands %v", got, err, *commands)
	}
}
