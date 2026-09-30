package credentialauthority

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRepositoryCredentialResolveGuard(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
	var findings []collapseFinding
	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if path != repoRoot && (info.Name() == ".git" || info.Name() == "vendor" || info.Name() == "templates") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		fileFindings, scanErr := scanCredentialCollapses(path, source)
		if scanErr != nil {
			return scanErr
		}
		findings = append(findings, fileFindings...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) > 0 {
		lines := make([]string, len(findings))
		for i, finding := range findings {
			lines[i] = finding.String()
		}
		t.Fatalf("credential authority Resolve collapse detected:\n%s", strings.Join(lines, "\n"))
	}
}
