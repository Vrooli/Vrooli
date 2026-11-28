package phases

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	catalog "test-genie/internal/orchestrator/phases"
)

// TestAllowedPhasesMatchCatalog is the anti-drift guard: the CLI's allowed phase
// set must equal the catalog phase set. If a phase is added/removed in the
// catalog, this fails until the CLI tracks it (which it now does automatically).
func TestAllowedPhasesMatchCatalog(t *testing.T) {
	want := make(map[string]struct{})
	for _, n := range catalog.ValidPhaseNames() {
		want[n] = struct{}{}
	}
	got := make(map[string]struct{})
	for _, n := range allowedPhaseNames() {
		got[n] = struct{}{}
	}
	if len(got) != len(want) {
		t.Fatalf("CLI allowed phase count = %d, catalog count = %d", len(got), len(want))
	}
	for n := range want {
		if _, ok := got[n]; !ok {
			t.Errorf("catalog phase %q missing from CLI allowed set", n)
		}
	}
	for n := range got {
		if _, ok := want[n]; !ok {
			t.Errorf("CLI allowed phase %q is not a catalog phase", n)
		}
	}
}

// TestE2EAliasStillValidates ensures the e2e→workflow alias still resolves even
// though "e2e" is not a catalog phase name.
func TestE2EAliasStillValidates(t *testing.T) {
	got, err := NormalizeSelection([]string{"e2e"})
	if err != nil {
		t.Fatalf("e2e should validate via alias: %v", err)
	}
	if len(got) != 1 || got[0] != "workflow" {
		t.Fatalf("e2e should normalize to [workflow], got %v", got)
	}
}

// TestPreviouslyMissingPhasesValidate locks in the bug fix: experience-contract,
// contracts, and performance were absent from the hand-maintained list and could be
// named via the CLI. They must now validate.
func TestPreviouslyMissingPhasesValidate(t *testing.T) {
	for _, p := range []string{"experience", "experience-contract", "contracts", "performance"} {
		got, err := NormalizeSelection([]string{p})
		if err != nil {
			t.Errorf("phase %q should validate via CLI: %v", p, err)
			continue
		}
		if len(got) != 1 || got[0] != p {
			t.Errorf("phase %q should normalize to [%s], got %v", p, p, got)
		}
	}
}

func TestRetiredUIHealthPhaseIsRejected(t *testing.T) {
	if _, err := NormalizeSelection([]string{"ui-health"}); err == nil {
		t.Fatal("retired ui-health selector must be rejected; use experience")
	}
}

// This guards executable selectors across the phase producers and first-party
// consumers. Historical
// findings and provider/service identifiers remain valid data; they are not phase
// selections.
func TestOwnedExecutableConsumersDoNotSelectRetiredUIHealthPhase(t *testing.T) {
	root := repositoryRoot(t)
	selectors := regexp.MustCompile(`--(?:phases|skip)(?:=|\s+)[^\n]*ui-health|Phases\s*:\s*\[\]string\s*\{[^}]*"ui-health"|(?:phases|skip)\s*[:=]\s*\[\s*"[^\]]*ui-health`)
	sourceRoots := []string{
		"scenarios/test-genie/api",
		"scenarios/test-genie/cli",
		"scenarios/git-control-tower/api",
		"scenarios/ui-health/api/handlers/validation",
		"scenarios/experience-manager/api",
		"scenarios/workflow-health/api",
		"scenarios/performance-health/api",
		"scenarios/browser-automation-studio/api/services/readiness",
		"scenarios/browser-automation-studio/api/services/uxmetrics",
		"packages/maturity-go",
	}
	for _, relative := range sourceRoots {
		base := filepath.Join(root, filepath.FromSlash(relative))
		err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == ".git" || entry.Name() == "node_modules" || entry.Name() == "dist" || entry.Name() == "testdata" || entry.Name() == "data" || entry.Name() == "runs" || entry.Name() == "coverage" || entry.Name() == "evidence" {
					return filepath.SkipDir
				}
				return nil
			}
			switch filepath.Ext(path) {
			case ".go", ".py", ".sh", ".ts", ".tsx", ".js", ".mjs", ".json", ".yaml", ".yml":
			default:
				return nil
			}
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if selectors.Match(body) {
				t.Errorf("retired phase selector in %s", filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator))))
			}
			if filepath.Ext(path) == ".json" {
				var value any
				if err := json.Unmarshal(body, &value); err == nil && hasRetiredPhaseSelection(value) {
					t.Errorf("retired phase selection in %s", filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator))))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", relative, err)
		}
	}
	configs, err := filepath.Glob(filepath.Join(root, "scenarios", "*", ".vrooli", "testing.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range configs {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var value any
		if err := json.Unmarshal(body, &value); err != nil {
			t.Errorf("invalid testing config %s: %v", path, err)
			continue
		}
		if hasRetiredPhaseSelection(value) {
			t.Errorf("retired phase selection in %s", filepath.ToSlash(strings.TrimPrefix(path, root+string(filepath.Separator))))
		}
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "scenarios", "test-genie", ".vrooli", "test-genie.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repository root")
		}
		dir = parent
	}
}

func hasRetiredPhaseSelection(value any) bool {
	switch node := value.(type) {
	case map[string]any:
		for key, child := range node {
			if strings.EqualFold(key, "phases") {
				if phases, ok := child.(map[string]any); ok {
					if _, found := phases["ui-health"]; found {
						return true
					}
				}
				if phases, ok := child.([]any); ok {
					for _, phase := range phases {
						if phase == "ui-health" {
							return true
						}
					}
				}
			}
			if hasRetiredPhaseSelection(child) {
				return true
			}
		}
	case []any:
		for _, child := range node {
			if hasRetiredPhaseSelection(child) {
				return true
			}
		}
	}
	return false
}
