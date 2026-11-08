package credentialauthority

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestRepositoryCredentialResolveGuard(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		t.Fatalf("load repo contract: %v", err)
	}
	var findings []collapseFinding
	err = contract.WalkSource(repoRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "templates" {
				return fs.SkipDir
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
