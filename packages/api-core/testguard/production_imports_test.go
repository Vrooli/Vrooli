package testguard_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/testguard"
)

func TestProductionTestutilImportsScansWholeAPIAndIgnoresTestSupport(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example/api\n")
	write("handlers/production.go", "package handlers\nimport _ \"example/api/internal/testutil/mocks\"\n")
	write("internal/testutil/helpers.go", "package testutil\nimport _ \"example/api/internal/testutil/mocks\"\n")
	write("internal/testutil/mocks/fake.go", "package mocks\nimport _ \"example/api/internal/testutil\"\n")
	write("internal/generated/bridge.go", "package generated\nimport _ \"example/api/internal/testutil\"\n")
	write("handlers/production_test.go", "package handlers\nimport _ \"example/api/internal/testutil\"\n")
	write("handlers/near-miss.go", "package handlers\nimport _ \"example/api/internal/testutilities\"\n")

	violations, err := testguard.ProductionTestutilImports(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 1 || violations[0].File != "handlers/production.go" || violations[0].Import != "example/api/internal/testutil/mocks" {
		t.Fatalf("violations = %#v, want only production handler import", violations)
	}
}

func TestProductionTestutilImportsRejectsMalformedSource(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example/api\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "broken.go"), []byte("package broken\nimport \"unterminated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := testguard.ProductionTestutilImports(root); err == nil {
		t.Fatal("expected parse error for malformed production source")
	}
}

func TestAssertNoForbiddenReferencesMatchesAliasesAndScopesToProductionFiles(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"domain.go":      "package domain\nimport (clock \"time\"; web \"net/http\")\nvar _ = clock.Duration(0)\nvar _ = clock.Now\nvar _ = web.DefaultClient\n",
		"domain_test.go": "package domain\nimport \"time\"\nvar _ = time.Now\n",
		"near.go":        "package domain\nimport \"net/http/httptest\"\nvar _ = httptest.NewRecorder\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	violations := &assertionCapture{TB: t}
	testguard.AssertNoForbiddenReferences(violations, root, map[string]string{
		"net/http": "transport belongs in handlers",
		"time.Now": "inject the clock",
	})
	joined := strings.Join(violations.failures, "\n")
	if len(violations.failures) != 3 || !strings.Contains(joined, "domain.go references time.Now") || !strings.Contains(joined, "domain.go imports net/http") {
		t.Fatalf("failures = %#v, want the production-only package and aliased member violations", violations.failures)
	}
}

type assertionCapture struct {
	testing.TB
	failures []string
}

func (c *assertionCapture) Helper() {}
func (c *assertionCapture) Errorf(format string, args ...any) {
	c.failures = append(c.failures, fmt.Sprintf(format, args...))
}
func (c *assertionCapture) Fatalf(format string, args ...any) {
	c.failures = append(c.failures, fmt.Sprintf(format, args...))
}
