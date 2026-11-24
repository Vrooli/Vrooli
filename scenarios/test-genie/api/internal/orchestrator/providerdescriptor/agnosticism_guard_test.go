package providerdescriptor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// providerScenarioLiteralViolations is intentionally source-level rather than
// a provider registry. Adding a new descriptor-only provider must not require
// adding a test-genie exception or switch case. Control-plane integrations are
// the only allowed literals and must be justified here.
func providerScenarioLiteralViolations(repoRoot, sourceRoot string, allow map[string]string) ([]string, error) {
	providers, err := descriptorScenarioNames(repoRoot)
	if err != nil {
		return nil, err
	}
	var violations []string
	err = filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}
			if _, isProvider := providers[value]; !isProvider {
				return true
			}
			key := filepath.ToSlash(path) + "::" + value
			if reason := allow[key]; reason != "" {
				return true
			}
			violations = append(violations, key)
			return true
		})
		return nil
	})
	return violations, err
}

func descriptorScenarioNames(repoRoot string) (map[string]struct{}, error) {
	entries, err := os.ReadDir(filepath.Join(repoRoot, "scenarios"))
	if err != nil {
		return nil, err
	}
	providers := map[string]struct{}{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == "test-genie" {
			// The harness's own identity is an application identity, not a
			// delegated provider and therefore is not an agnosticism coupling.
			continue
		}
		matches, err := filepath.Glob(filepath.Join(repoRoot, "scenarios", entry.Name(), ".vrooli", "test-genie*.json"))
		if err != nil {
			return nil, err
		}
		if len(matches) > 0 {
			providers[entry.Name()] = struct{}{}
		}
	}
	return providers, nil
}

func TestTestGenieSourceIsProviderAgnostic(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", "..", "..", "..", ".."))
	sourceRoot := filepath.Join(repoRoot, "scenarios", "test-genie", "api")
	violations, err := providerScenarioLiteralViolations(repoRoot, sourceRoot, providerAgnosticSourceAllowlist(sourceRoot))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("provider scenario literals found in non-test test-genie source: %s", strings.Join(violations, ", "))
	}
}

func providerAgnosticSourceAllowlist(sourceRoot string) map[string]string {
	return map[string]string{
		// agent-manager is the control-plane remediation integration, not a
		// test-genie phase provider. Keep its literal explicit and reviewable.
		filepath.ToSlash(filepath.Join(sourceRoot, "agentmanager", "client.go")) + "::agent-manager": "control-plane remediation client",
		// The descriptor loader reads Prompt Manager's active skill-pack index
		// and indexes scenario-owned skills; this is catalog validation input,
		// not provider phase dispatch. Keep the owner integration exact-file scoped.
		filepath.ToSlash(filepath.Join(sourceRoot, "internal", "orchestrator", "providerdescriptor", "descriptor.go")) + "::prompt-manager": "active skill-catalog owner integration for recommended-skill validation",
		// The following exact-target checks and request-scoped grant forwarding
		// implement Audio Tools' Test Genie-owned lease/browser-grant contract.
		// They are service integrations, not phase routing or provider catalog
		// selection. Keep each exception limited to its owning boundary.
		filepath.ToSlash(filepath.Join(sourceRoot, "internal", "app", "httpserver", "demand_lease_receipts.go")) + "::audio-tools":               "target-scoped Test Genie demand-lease receipt endpoint for the browser-grant integration",
		filepath.ToSlash(filepath.Join(sourceRoot, "internal", "app", "runs", "service.go")) + "::audio-tools":                                   "defense-in-depth target scope for retained demand-lease receipt verification",
		filepath.ToSlash(filepath.Join(sourceRoot, "internal", "orchestrator", "phases", "phase_validationprovider.go")) + "::audio-tools":       "attach the run-scoped lease receipt only to the browser-grant soak phase",
		filepath.ToSlash(filepath.Join(sourceRoot, "internal", "orchestrator", "phases", "validationprovider", "provider.go")) + "::audio-tools": "forward request-scoped browser and lease grants only to the exact provider phase",
	}
}

func TestProviderAgnosticismAllowlistIsExactFileScoped(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "scenarios", "audio-tools", ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "scenarios", "audio-tools", ".vrooli", "test-genie.json"), []byte(`{"scenario":"audio-tools"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	sourceRoot := filepath.Join(repoRoot, "api")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "unapproved.go"), []byte(`package api; const provider = "audio-tools"`), 0o644); err != nil {
		t.Fatal(err)
	}
	allow := map[string]string{
		filepath.ToSlash(filepath.Join(sourceRoot, "known", "integration.go")) + "::audio-tools": "reviewed exact integration",
	}
	violations, err := providerScenarioLiteralViolations(repoRoot, sourceRoot, allow)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 || !strings.Contains(violations[0], "unapproved.go::audio-tools") {
		t.Fatalf("violations = %v, want the unapproved audio-tools literal", violations)
	}
}

func TestProviderAgnosticismGuardRejectsNewLiteral(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "scenarios", "new-provider", ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "scenarios", "new-provider", ".vrooli", "test-genie.json"), []byte(`{"scenario":"new-provider"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	sourceRoot := filepath.Join(repoRoot, "api")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "bad.go"), []byte(`package bad; const provider = "new-provider"`), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := providerScenarioLiteralViolations(repoRoot, sourceRoot, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 || !strings.Contains(violations[0], "new-provider") {
		t.Fatalf("violations = %v, want new-provider literal", violations)
	}
}
