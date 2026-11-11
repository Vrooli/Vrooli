package scanners

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunExternalToolCapturesOutputAndAppliesExitPolicy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixtures use a POSIX script")
	}
	tool := writeExecutable(t, "#!/bin/sh\nprintf 'findings\\n'\nprintf 'diagnostic\\n' >&2\nexit 3\n")

	stdout, stderr, err := runExternalTool(tool, nil, time.Second, func(err error) bool {
		return strings.Contains(err.Error(), "exit status 3")
	}, true)
	if err != nil {
		t.Fatalf("runExternalTool() error = %v", err)
	}
	if stdout != "findings\n" || stderr != "diagnostic\n" {
		t.Fatalf("captured output = (%q, %q), want findings and diagnostic", stdout, stderr)
	}

	_, _, err = runExternalTool(tool, nil, time.Second, nil, false)
	if err == nil || !strings.Contains(err.Error(), "exit status 3") || !strings.Contains(err.Error(), "diagnostic") {
		t.Fatalf("strict exit policy error = %v, want exit status and stderr", err)
	}
}

func TestCountExternalToolFilesPreservesScannerScopes(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "main.go"), "package main\nfunc main() {}\n")
	writeFile(t, filepath.Join(root, "main_test.go"), "package main\n")
	writeFile(t, filepath.Join(root, "notes.md"), "one\ntwo\n")
	writeFile(t, filepath.Join(root, "vendor", "dep.go"), "package dep\n")
	writeFile(t, filepath.Join(root, "vendor", "secret.txt"), "secret\n")

	goFiles, goLines := countExternalToolFiles(root, func(name string) bool {
		return name == "vendor"
	}, func(path string) bool {
		return strings.HasSuffix(path, ".go") && !strings.Contains(path, "_test.go")
	})
	if goFiles != 1 || goLines != 2 {
		t.Fatalf("Go count = (%d files, %d lines), want (1, 2)", goFiles, goLines)
	}

	textFiles, textLines := countExternalToolFiles(root, func(name string) bool {
		return name == "vendor"
	}, func(path string) bool {
		switch filepath.Ext(path) {
		case ".go", ".md", ".txt":
			return true
		default:
			return false
		}
	})
	if textFiles != 3 || textLines != 5 {
		t.Fatalf("text count = (%d files, %d lines), want (3, 5)", textFiles, textLines)
	}
}

func TestCommandScannersKeepToolSpecificScanBehavior(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixtures use a POSIX script")
	}
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "main.go"), "package main\nfunc main() {}\n")
	writeFile(t, filepath.Join(root, "main_test.go"), "package main\n")
	writeFile(t, filepath.Join(root, "notes.md"), "one\ntwo\n")
	logger := testLogger{}

	gitleaksPath := writeExecutable(t, "#!/bin/sh\nif [ \"$1\" = version ]; then echo v8.0.0; exit 0; fi\nreport=''\nwhile [ \"$#\" -gt 0 ]; do\n  if [ \"$1\" = --report-path ]; then report=\"$2\"; shift 2; else shift; fi\ndone\nprintf '[]' > \"$report\"\nexit 1\n")
	gitleaks := NewGitleaksScanner(logger)
	gitleaks.binPath = gitleaksPath
	gitleaksResult, err := gitleaks.Scan(ScanOptions{Path: root, ScanType: "quick"})
	if err != nil {
		t.Fatalf("Gitleaks Scan() error = %v", err)
	}
	if gitleaksResult.ToolVersion != "8.0.0" || gitleaksResult.FilesScanned != 3 || gitleaksResult.LinesScanned != 5 {
		t.Fatalf("Gitleaks result version/files/lines = %q/%d/%d, want 8.0.0/3/5", gitleaksResult.ToolVersion, gitleaksResult.FilesScanned, gitleaksResult.LinesScanned)
	}

	gosecPath := writeExecutable(t, "#!/bin/sh\nif [ \"$1\" = -version ]; then echo 'Version: 2.21.0'; exit 0; fi\nprintf '{\"Issues\":[],\"Stats\":{}}'\nexit 3\n")
	gosec := NewGosecScanner(logger)
	gosec.binPath = gosecPath
	gosecResult, err := gosec.Scan(ScanOptions{Path: root, ScanType: "quick"})
	if err != nil {
		t.Fatalf("Gosec Scan() error = %v", err)
	}
	if gosecResult.ToolVersion != "2.21.0" || gosecResult.FilesScanned != 1 || gosecResult.LinesScanned != 2 {
		t.Fatalf("Gosec result version/files/lines = %q/%d/%d, want 2.21.0/1/2", gosecResult.ToolVersion, gosecResult.FilesScanned, gosecResult.LinesScanned)
	}
}

func TestGosecRuleMetadataHasOneEntryForEachDefaultCheck(t *testing.T) {
	scanner := &GosecScanner{}
	for _, rule := range scanner.GetDefaultRules() {
		metadata, ok := gosecRules[rule.ID]
		if !ok || metadata.title == "" || metadata.category == "" || metadata.recommendation == "" {
			t.Errorf("default rule %s has incomplete shared metadata: %+v", rule.ID, metadata)
		}
	}

	if got := scanner.getRuleTitle("G502"); got != "Blacklisted Import: crypto/des" {
		t.Errorf("G502 title = %q", got)
	}
	if got := scanner.mapToOWASP("G502"); got != "" {
		t.Errorf("G502 OWASP mapping = %q, want empty", got)
	}
	if got := scanner.getRuleCategory("unknown"); got != "Security" {
		t.Errorf("unknown category = %q", got)
	}
	if got := scanner.getRuleRecommendation("unknown"); got != "Review and fix the security issue according to best practices." {
		t.Errorf("unknown recommendation = %q", got)
	}
}

type testLogger struct{}

func (testLogger) Info(string, ...any)  {}
func (testLogger) Warn(string, ...any)  {}
func (testLogger) Error(string, ...any) {}
func (testLogger) Debug(string, ...any) {}

func writeExecutable(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tool")
	if err := os.WriteFile(path, []byte(contents), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
