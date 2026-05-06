package protocgenconnectgo

import (
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

var testManifest = hostreqkit.ToolManifest{
	Name:        "protoc-gen-connect-go",
	Description: "Go Connect protoc plugin",
	Commands:    []string{"protoc-gen-connect-go"},
	VersionArgs: []string{"--version"},
	Handler:     "protoc_gen_connect_go",
	Version:     "v1.19.2",
	Platforms:   []string{"linux", "macos", "windows"},
}

func restoreStubs(t *testing.T) func() {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origUserHome := UserHomeDirFn
	return func() {
		hostreqkit.LookPathFn = origLookPath
		UserHomeDirFn = origUserHome
	}
}

func TestInspectGoPresentEnablesInstall(t *testing.T) {
	defer restoreStubs(t)()
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "go" {
			return "/usr/bin/go", nil
		}
		return "", os.ErrNotExist
	}
	status := NewHandler(testManifest).Inspect(hostreqkit.Host{OS: "linux"}, hostreqspec.ResolvedRequirement{Name: "protoc-gen-connect-go", Kind: hostreqspec.KindTool})
	if !status.InstallSupported {
		t.Fatal("InstallSupported should be true with Go available")
	}
	if !strings.Contains(status.PackageName, "@v1.19.2") {
		t.Fatalf("PackageName = %q", status.PackageName)
	}
}

func TestApplyDryRunMentionsGoInstall(t *testing.T) {
	defer restoreStubs(t)()
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "go" {
			return "/usr/bin/go", nil
		}
		return "", os.ErrNotExist
	}
	h := NewHandler(testManifest)
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, hostreqspec.ResolvedRequirement{Name: "protoc-gen-connect-go", Kind: hostreqspec.KindTool})
	out, err := h.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionWouldInstall {
		t.Fatalf("ExecutionState = %q", out.ExecutionState)
	}
	if !containsNote(out.Notes, "go install connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.19.2") {
		t.Fatalf("notes = %v", out.Notes)
	}
}

func TestPinnedVersionMatchesManifest(t *testing.T) {
	if testManifest.Version != defaultVersion {
		t.Fatalf("manifest version %q drifted from defaultVersion %q", testManifest.Version, defaultVersion)
	}
}

func containsNote(notes []string, needle string) bool {
	for _, note := range notes {
		if strings.Contains(note, needle) {
			return true
		}
	}
	return false
}
