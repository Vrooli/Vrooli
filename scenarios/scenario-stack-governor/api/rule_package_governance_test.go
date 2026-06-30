package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPackageGovernanceRule_FiltersToScenarioPaths(t *testing.T) {
	root := setupRuleGoCliTestRepo(t)
	script := filepath.Join(t.TempDir(), "fake-vrooli")
	writeTestFile(t, script, `#!/usr/bin/env bash
set -e
cat <<'JSON'
{
  "success": true,
  "audit": {
    "validation": {"issues": []},
    "issues": [
      {
        "severity": "error",
        "code": "package-no-workspace-deps",
        "message": "real scenario \"alpha\" uses workspace:* for shared package adoption",
        "path": "`+filepath.Join(root, "scenarios", "alpha", "ui", "package.json")+`"
      },
      {
        "severity": "error",
        "code": "missing-package-manifest",
        "message": "package root is missing .vrooli/package.json",
        "path": "`+filepath.Join(root, "packages", "beta", ".vrooli", "package.json")+`"
      }
    ]
  }
}
JSON
`)
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatalf("chmod fake vrooli: %v", err)
	}
	t.Setenv("VROOLI_BIN", script)

	result := RunPackageGovernanceScenarioAdoption(t.Context(), root, "alpha")
	if result.Passed {
		t.Fatalf("expected failure, got pass: %+v", result)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(result.Findings), result.Findings)
	}
	if result.Findings[0].ScenarioName != "alpha" {
		t.Fatalf("finding scenario = %q", result.Findings[0].ScenarioName)
	}
	if !strings.Contains(result.Findings[0].Message, "workspace:*") {
		t.Fatalf("unexpected finding message: %q", result.Findings[0].Message)
	}
}

func TestPackageGovernanceRule_ScansAllScenariosWhenUnscoped(t *testing.T) {
	root := setupRuleGoCliTestRepo(t)
	script := filepath.Join(t.TempDir(), "fake-vrooli")
	writeTestFile(t, script, `#!/usr/bin/env bash
set -e
cat <<'JSON'
{
  "success": true,
  "audit": {
    "validation": {"issues": []},
    "issues": [
      {
        "severity": "warning",
        "code": "package-no-unauthorized-postinstall",
        "message": "consumer \"alpha\" still uses postinstall shared-package propagation",
        "path": "`+filepath.Join(root, "scenarios", "alpha", "ui", "package.json")+`"
      },
      {
        "severity": "error",
        "code": "package-adoption-supported",
        "message": "testkit-go is not scenario-adoptable",
        "path": "`+filepath.Join(root, "scenarios", "beta", "api", "go.mod")+`"
      }
    ]
  }
}
JSON
`)
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatalf("chmod fake vrooli: %v", err)
	}
	t.Setenv("VROOLI_BIN", script)

	result := RunPackageGovernanceScenarioAdoption(t.Context(), root, "")
	if len(result.Findings) != 2 {
		t.Fatalf("expected 2 findings, got %d: %+v", len(result.Findings), result.Findings)
	}
	if result.Findings[0].ScenarioName == result.Findings[1].ScenarioName {
		t.Fatalf("expected findings for different scenarios: %+v", result.Findings)
	}
}

func TestPackageGovernanceRule_ReportsCommandFailure(t *testing.T) {
	root := setupRuleGoCliTestRepo(t)
	script := filepath.Join(t.TempDir(), "fake-vrooli")
	writeTestFile(t, script, "#!/usr/bin/env bash\nset -e\nprintf 'boom\\n' >&2\nexit 12\n")
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatalf("chmod fake vrooli: %v", err)
	}
	t.Setenv("VROOLI_BIN", script)

	result := RunPackageGovernanceScenarioAdoption(t.Context(), root, "alpha")
	if result.Passed {
		t.Fatal("expected failure when package audit command fails")
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}
	if result.Findings[0].Level != "error" {
		t.Fatalf("finding level = %q", result.Findings[0].Level)
	}
}

func TestPackageGovernanceRule_ReportsBudgetExceededMetadata(t *testing.T) {
	root := setupRuleGoCliTestRepo(t)
	script := filepath.Join(t.TempDir(), "fake-vrooli")
	writeTestFile(t, script, `#!/usr/bin/env bash
set -e
cat <<'JSON'
{
  "success": true,
  "audit": {
    "validation": {"issues": []},
    "issues": [],
    "scan_stats": {
      "files_scanned": "10",
      "files_skipped": "2",
      "bytes_scanned": "2048",
      "skipped_by_reason": {"file-byte-budget": "1"},
      "budget_exceeded": true
    }
  }
}
JSON
`)
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatalf("chmod fake vrooli: %v", err)
	}
	t.Setenv("VROOLI_BIN", script)

	result := RunPackageGovernanceScenarioAdoption(t.Context(), root, "alpha")
	if result.Passed {
		t.Fatalf("expected warning to fail rule, got pass: %+v", result)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(result.Findings), result.Findings)
	}
	if result.Findings[0].Level != "warn" || !strings.Contains(result.Findings[0].Message, "scan budget") {
		t.Fatalf("unexpected budget finding: %+v", result.Findings[0])
	}
}

func TestPackageGovernanceRule_TimesOutAuditCommand(t *testing.T) {
	root := setupRuleGoCliTestRepo(t)
	script := filepath.Join(t.TempDir(), "fake-vrooli")
	writeTestFile(t, script, "#!/usr/bin/env bash\nsleep 5\n")
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatalf("chmod fake vrooli: %v", err)
	}
	t.Setenv("VROOLI_BIN", script)
	originalTimeout := packageAuditTimeout
	packageAuditTimeout = 10 * time.Millisecond
	t.Cleanup(func() { packageAuditTimeout = originalTimeout })

	result := RunPackageGovernanceScenarioAdoption(t.Context(), root, "alpha")
	if result.Passed {
		t.Fatal("expected failure when package audit times out")
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}
	if !strings.Contains(result.Findings[0].Message, "timed out") {
		t.Fatalf("unexpected timeout message: %q", result.Findings[0].Message)
	}
}
