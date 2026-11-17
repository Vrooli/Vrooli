package nodejsruntime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/tools"
)

func TestRuntimeCatalogPinsDefaultAndAlternateAcrossSupportedTargets(t *testing.T) {
	platforms := []struct{ os, arch string }{
		{"linux", "amd64"}, {"linux", "arm64"},
		{"darwin", "amd64"}, {"darwin", "arm64"},
		{"windows", "amd64"}, {"windows", "arm64"},
	}
	toolData, err := tools.Manifests.ReadFile("node/tool.json")
	if err != nil {
		t.Fatal(err)
	}
	var defaultManifest toolManifest
	if err := json.Unmarshal(toolData, &defaultManifest); err != nil {
		t.Fatal(err)
	}
	catalogData, err := tools.Manifests.ReadFile("node/versions.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalog versionCatalog
	if err := json.Unmarshal(catalogData, &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Versions) == 0 {
		t.Fatal("Node version catalog has no scenario-selectable releases")
	}
	validate := func(version string, acquisition binaryfetch.Acquisition) {
		if err := acquisition.Validate(); err != nil {
			t.Fatalf("Node %s acquisition: %v", version, err)
		}
		for _, platform := range platforms {
			target, err := acquisition.Resolve(binaryfetch.Facts{"os": platform.os, "arch": platform.arch})
			if err != nil || len(target.SHA256) != 64 || target.Layout != "dir" || target.BinPath == "" {
				t.Errorf("Node %s target %s/%s = %#v, want checksum-pinned archive", version, platform.os, platform.arch, target)
			}
		}
	}
	if defaultManifest.Acquisition == nil {
		t.Fatal("default Node manifest has no acquisition")
	}
	if defaultCatalog, ok := catalog.Versions[defaultManifest.Version]; !ok {
		t.Fatalf("host default Node %s must remain in the scenario version catalog", defaultManifest.Version)
	} else if !reflect.DeepEqual(*defaultManifest.Acquisition, defaultCatalog) {
		t.Fatalf("default Node %s acquisition differs from the selectable catalog entry", defaultManifest.Version)
	}
	validate(defaultManifest.Version, *defaultManifest.Acquisition)
	for version, acquisition := range catalog.Versions {
		if !exactVersion.MatchString(version) {
			t.Errorf("catalog version %q is not exact x.y.z", version)
		}
		validate(version, acquisition)
	}
}

func TestRuntimeManagerRejectsUncataloguedVersionBeforeAcquiring(t *testing.T) {
	manager := RuntimeManager{
		Home: t.TempDir(), OS: "linux", Arch: "amd64",
		FetchDir: func(context.Context, binaryfetch.Target, string, binaryfetch.ProgressFunc) (string, error) {
			t.Fatal("uncatalogued Node version must not be downloaded")
			return "", nil
		},
	}
	if _, err := manager.Ensure(context.Background(), "23.0.0"); err == nil {
		t.Fatal("Ensure accepted an uncatalogued Node release")
	}
}

func TestRuntimeManagerSharesInstalledVersionCacheAcrossCallers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture executable uses a POSIX shebang")
	}
	home := t.TempDir()
	fetches := 0
	records := 0
	manager := RuntimeManager{
		Home: home, OS: "linux", Arch: "amd64",
		AcquireLock: func(context.Context, string) (func(), error) { return func() {}, nil },
		RecordArtifacts: func(_ string, entries ...cliinstall.InstallEntry) error {
			records += len(entries)
			return nil
		},
		FetchDir: func(_ context.Context, spec binaryfetch.Target, dest string, _ binaryfetch.ProgressFunc) (string, error) {
			fetches++
			entry := filepath.Join(dest, filepath.FromSlash(spec.BinPath))
			if err := os.MkdirAll(filepath.Dir(entry), 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(entry, []byte("#!/bin/sh\nprintf 'v22.23.5\\n'\n"), 0o755); err != nil {
				return "", err
			}
			return entry, nil
		},
	}
	first, err := manager.Ensure(context.Background(), "22.23.5")
	if err != nil {
		t.Fatalf("Ensure first caller: %v", err)
	}
	second, err := manager.Ensure(context.Background(), "22.23.5")
	if err != nil {
		t.Fatalf("Ensure second caller: %v", err)
	}
	if first != second {
		t.Fatalf("runtime dirs differ: %q vs %q", first, second)
	}
	if fetches != 1 {
		t.Fatalf("fetch count = %d, want one shared install", fetches)
	}
	if records != 1 {
		t.Fatalf("recorded artifact count = %d, want one tracked runtime install", records)
	}
}
