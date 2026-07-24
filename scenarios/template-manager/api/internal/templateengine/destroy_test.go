package templateengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

// destroyRepo lays down a scenario plus its full six-output proto footprint.
func destroyRepo(t *testing.T, scenario string, withScenarioDir bool) string {
	t.Helper()
	root := t.TempDir()
	python := strings.ReplaceAll(scenario, "-", "_")

	dirs := []string{
		filepath.Join("packages", "proto", "schemas", scenario, "v1", "health"),
		filepath.Join("packages", "proto", "gen", "go", scenario, "v1"),
		filepath.Join("packages", "proto", "gen", "typescript", scenario, "v1"),
		filepath.Join("packages", "proto", "gen", "typescript", "js", scenario, "v1"),
		filepath.Join("packages", "proto", "gen", "python", python, "v1"),
	}
	if withScenarioDir {
		dirs = append(dirs, filepath.Join("scenarios", scenario, "api"))
	}
	for _, dir := range dirs {
		full := filepath.Join(root, dir)
		if err := os.MkdirAll(full, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(full, "file.txt"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manifests := filepath.Join(root, "packages", "proto", "gen", "manifests")
	if err := os.MkdirAll(manifests, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifests, scenario+".lock.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func destroyDeps(root string) HandlerDeps[struct{}] {
	return HandlerDeps[struct{}]{Root: func(struct{}) string { return root }}
}

func runDestroyIn(t *testing.T, root string, req templatecontracts.DestroyRequest) templatecontracts.DestroyResult {
	t.Helper()
	result, err := runDestroy(destroyDeps(root), struct{}{}, req)
	if err != nil {
		t.Fatalf("runDestroy: %v", err)
	}
	return result
}

// The whole point of the command: a plain rm of the scenario dir strands the
// shared proto outputs, so destroy must remove all of them together.
func TestDestroyRemovesScenarioAndEveryProtoOutput(t *testing.T) {
	root := destroyRepo(t, "throwaway-probe", true)

	result := runDestroyIn(t, root, templatecontracts.DestroyRequest{Name: "throwaway-probe", Force: true})

	for _, rel := range []string{
		"scenarios/throwaway-probe",
		"packages/proto/schemas/throwaway-probe",
		"packages/proto/gen/go/throwaway-probe",
		"packages/proto/gen/typescript/throwaway-probe",
		"packages/proto/gen/typescript/js/throwaway-probe",
		"packages/proto/gen/python/throwaway_probe",
		"packages/proto/gen/manifests/throwaway-probe.lock.json",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Errorf("%s still exists after destroy", rel)
		}
	}
	if !result.NeedsProtoGenerate {
		t.Error("destroy should require a codegen re-run")
	}
}

// Destroying a live scenario must be deliberate.
func TestDestroyRefusesLiveScenarioWithoutForce(t *testing.T) {
	root := destroyRepo(t, "live-scenario", true)

	_, err := runDestroy(destroyDeps(root), struct{}{}, templatecontracts.DestroyRequest{Name: "live-scenario"})
	if err == nil {
		t.Fatal("expected an error without --force")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("error should name the flag that unblocks it, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "scenarios", "live-scenario")); statErr != nil {
		t.Fatal("refused destroy must not delete anything")
	}
}

// The already-deleted case: reap stranded codegen without needing --force and
// without touching a scenario directory that no longer exists.
func TestDestroyProtoOnlyReapsStrandedCodegen(t *testing.T) {
	root := destroyRepo(t, "orphan-surface", false)

	result := runDestroyIn(t, root, templatecontracts.DestroyRequest{Name: "orphan-surface", ProtoOnly: true})

	if _, err := os.Stat(filepath.Join(root, "packages", "proto", "schemas", "orphan-surface")); !os.IsNotExist(err) {
		t.Error("proto schemas survived a proto-only destroy")
	}
	for _, rel := range result.PathsRemoved {
		if strings.HasPrefix(rel, "scenarios/") {
			t.Errorf("proto-only destroy must not touch %s", rel)
		}
	}
}

// proto-only on a scenario that still exists must leave the scenario alone.
func TestDestroyProtoOnlyLeavesLiveScenarioDirectory(t *testing.T) {
	root := destroyRepo(t, "live-scenario", true)

	runDestroyIn(t, root, templatecontracts.DestroyRequest{Name: "live-scenario", ProtoOnly: true})

	if _, err := os.Stat(filepath.Join(root, "scenarios", "live-scenario")); err != nil {
		t.Fatal("proto-only destroy deleted the scenario directory")
	}
}

func TestDestroyDryRunDeletesNothingButListsEverything(t *testing.T) {
	root := destroyRepo(t, "throwaway-probe", true)

	result := runDestroyIn(t, root, templatecontracts.DestroyRequest{Name: "throwaway-probe", DryRun: true, Force: true})

	if len(result.PathsRemoved) < 7 {
		t.Fatalf("dry run should list the whole footprint, got %v", result.PathsRemoved)
	}
	for _, rel := range result.PathsRemoved {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Errorf("dry run deleted %s", rel)
		}
	}
	if !strings.Contains(result.Message, "would remove") {
		t.Errorf("dry-run message should be conditional, got %q", result.Message)
	}
}

// A partially-removed footprint must not look like a complete one.
func TestDestroyReportsAbsentPathsSeparately(t *testing.T) {
	root := destroyRepo(t, "partial-surface", false)
	if err := os.RemoveAll(filepath.Join(root, "packages", "proto", "gen", "python", "partial_surface")); err != nil {
		t.Fatal(err)
	}

	result := runDestroyIn(t, root, templatecontracts.DestroyRequest{Name: "partial-surface", ProtoOnly: true})

	var sawAbsent bool
	for _, rel := range result.PathsAbsent {
		if strings.Contains(rel, "python/partial_surface") {
			sawAbsent = true
		}
	}
	if !sawAbsent {
		t.Fatalf("missing python output should be reported absent, got %v", result.PathsAbsent)
	}
}

// A path in the scenario id would let a caller escape the repo layout.
func TestDestroyRejectsPathLikeScenarioIDs(t *testing.T) {
	root := destroyRepo(t, "throwaway-probe", false)

	for _, bad := range []string{"../etc", "nested/id", ".", ""} {
		if _, err := runDestroy(destroyDeps(root), struct{}{}, templatecontracts.DestroyRequest{Name: bad, Force: true}); err == nil {
			t.Errorf("scenario id %q must be rejected", bad)
		}
	}
}
