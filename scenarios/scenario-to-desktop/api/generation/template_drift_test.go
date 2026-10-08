package generation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareGeneratedArtifactDetectsDrift(t *testing.T) {
	dir := t.TempDir()
	rendered := filepath.Join(dir, "rendered.ts")
	checkedIn := filepath.Join(dir, "generated.ts")
	if err := os.WriteFile(rendered, []byte("auth-manager"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(checkedIn, []byte("auth-manager"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CompareGeneratedArtifact(rendered, checkedIn); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(checkedIn, []byte("stale-auth-manager"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CompareGeneratedArtifact(rendered, checkedIn); err == nil || !strings.Contains(err.Error(), "generated artifact drift") {
		t.Fatalf("drift error = %v", err)
	}
}

func TestBrowserAutomationStudioGeneratedShellMatchesTemplate(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	root := filepath.Clean(filepath.Join(workingDir, "..", "..", "..", ".."))
	templatePath := filepath.Join(root, "scenarios", "scenario-to-desktop", "templates", "vanilla", "main.ts")
	checkedInPath := filepath.Join(root, "scenarios", "browser-automation-studio", "platforms", "electron", "src", "main.ts")
	if err := CompareGeneratedTemplate(templatePath, checkedInPath); err != nil {
		t.Fatal(err)
	}
}
