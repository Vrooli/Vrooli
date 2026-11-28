package catalog

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestCensusReason(t *testing.T) {
	tests := []struct {
		name                             string
		inContract, enabled, declaresCLI bool
		installed                        bool
		want                             string
	}{
		{"absent", false, false, true, true, "absent_from_contract"},
		{"disabled", true, false, true, true, "resource_disabled"},
		{"not declared", true, true, false, false, "cli_not_declared"},
		{"declared missing", true, true, true, false, "declared_not_installed"},
		{"installed", true, true, true, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := censusReason(tt.inContract, tt.enabled, tt.declaresCLI, tt.installed); got != tt.want {
				t.Fatalf("censusReason() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCensusSkipsSharedCodeDirectories(t *testing.T) {
	dir := t.TempDir()
	if declared, err := isResourceDir(dir + "/internal/resource.json"); err != nil || declared {
		t.Fatal("a directory without resource.json counted as a resource")
	}
	manifest := dir + "/resource.json"
	if err := os.WriteFile(manifest, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if declared, err := isResourceDir(manifest); err != nil || !declared {
		t.Fatal("a declared resource was skipped")
	}
}

func TestCensusManifestInspectionFailureIsNotAnAbsentResource(t *testing.T) {
	dir := t.TempDir()
	loop := filepath.Join(dir, "loop")
	if err := os.Symlink("loop", loop); err != nil {
		t.Fatal(err)
	}
	declared, err := isResourceDir(filepath.Join(loop, "resource.json"))
	if declared || !errors.Is(err, syscall.ELOOP) {
		t.Fatalf("unreadable inventory must fail, got declared=%v error=%v", declared, err)
	}
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	declared, err = isResourceDir(filepath.Join(file, "resource.json"))
	if declared || !errors.Is(err, syscall.ENOTDIR) {
		t.Fatalf("invalid inventory parent must fail, got declared=%v error=%v", declared, err)
	}
}
