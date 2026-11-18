//go:build linux

package cliinstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveHostPackageSupportsInstalledAPKAndScoopPackages(t *testing.T) {
	binDir := t.TempDir()
	capture := filepath.Join(t.TempDir(), "args")
	t.Setenv("PATH", binDir)
	t.Setenv("VROOLI_TEST_PACKAGE_ARGS", capture)
	for _, manager := range []string{"apk", "scoop"} {
		script := "#!/bin/sh\nprintf '%s\\n' \"$*\" > \"$VROOLI_TEST_PACKAGE_ARGS\"\n"
		if err := os.WriteFile(filepath.Join(binDir, manager), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	for _, tc := range []struct {
		manager string
		want    string
	}{
		{manager: "apk", want: "del vrooli-test-package"},
		{manager: "scoop", want: "uninstall vrooli-test-package"},
	} {
		t.Run(tc.manager, func(t *testing.T) {
			entry := InstallEntry{
				Kind: EntryPackage,
				Path: "vrooli-test-package",
				Provenance: InstallProvenance{
					PackageManager: tc.manager,
					PackageName:    "vrooli-test-package",
				},
			}
			if err := removeHostPackage(entry); err != nil {
				t.Fatalf("removeHostPackage: %v", err)
			}
			args, err := os.ReadFile(capture)
			if err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(string(args)) != tc.want {
				t.Fatalf("arguments = %q, want %q", strings.TrimSpace(string(args)), tc.want)
			}
		})
	}
}
