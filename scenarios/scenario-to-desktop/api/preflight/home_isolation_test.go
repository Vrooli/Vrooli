package preflight

import (
	runtimeenv "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/env"
	bundlemanifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"os"
	"path/filepath"
	"testing"
)

func TestPreflightIsolatesServiceHomeAndPeerRegistry(t *testing.T) {
	operator := t.TempDir()
	t.Setenv("HOME", operator)
	t.Setenv("XDG_DATA_HOME", filepath.Join(operator, "private-data"))
	appData := t.TempDir()
	options, err := isolatedPreflightOptions(&bundlemanifest.Manifest{}, t.TempDir(), appData)
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(appData, "home")
	if options.PeerHomeDir != home {
		t.Fatalf("peer registry uses %q", options.PeerHomeDir)
	}
	environment, err := runtimeenv.NewRenderer(appData, options.BundlePath, nil, options.EnvReader).RenderEnvMap(bundlemanifest.Service{ID: "api"}, bundlemanifest.Binary{})
	if err != nil {
		t.Fatal(err)
	}
	if environment["HOME"] != home {
		t.Fatalf("service HOME = %q", environment["HOME"])
	}
	if _, err := os.Stat(home); err != nil {
		t.Fatal(err)
	}
	if got := options.EnvReader.Getenv("XDG_DATA_HOME"); got != filepath.Join(home, ".local", "share") {
		t.Fatalf("data home = %q", got)
	}
	if os.Getenv("HOME") != operator {
		t.Fatal("preflight mutated process-wide HOME")
	}
}
