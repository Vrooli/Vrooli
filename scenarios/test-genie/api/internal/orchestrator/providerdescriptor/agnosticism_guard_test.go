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
	violations, err := providerScenarioLiteralViolations(repoRoot, sourceRoot, map[string]string{
		// agent-manager is the control-plane remediation integration, not a
		// test-genie phase provider. Keep its literal explicit and reviewable.
		filepath.ToSlash(filepath.Join(sourceRoot, "agentmanager", "client.go")) + "::agent-manager": "control-plane remediation client",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("provider scenario literals found in non-test test-genie source: %s", strings.Join(violations, ", "))
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
