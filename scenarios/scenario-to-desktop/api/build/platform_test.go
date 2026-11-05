package build

import (
	"os"
	"path/filepath"
	"testing"
)

type recordingPackageFinder struct {
	platform string
	path     string
}

func (f *recordingPackageFinder) FindBuiltPackage(_ string, platform string) (string, error) {
	f.platform = platform
	return f.path, nil
}

func TestElectronBuilderPlatformAcceptsConcretePipelineTargets(t *testing.T) {
	for input, want := range map[string]string{"linux-amd64": "linux", "linux-arm64": "linux", "macos-arm64": "mac", "windows-amd64": "win", "win": "win"} {
		got, ok := electronBuilderPlatform(input)
		if !ok || got != want {
			t.Fatalf("electronBuilderPlatform(%q) = %q, %t; want %q, true", input, got, ok, want)
		}
	}
	if _, ok := electronBuilderPlatform("plan9-amd64"); ok {
		t.Fatal("unknown target was accepted")
	}
}

func TestRecordBuiltPackageNormalizesConcretePipelineTarget(t *testing.T) {
	dist := t.TempDir()
	artifact := filepath.Join(dist, "app.AppImage")
	if err := os.WriteFile(artifact, []byte("package"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore()
	store.Save(&Status{BuildID: "build", PlatformResults: map[string]*PlatformResult{"linux-amd64": {Platform: "linux-amd64"}}, Artifacts: map[string]string{}})
	finder := &recordingPackageFinder{path: artifact}
	service := NewService(WithStore(store), WithPackageFinder(finder))

	service.recordBuiltPackage("build", "fixture", dist, "linux-amd64")

	if finder.platform != "linux" {
		t.Fatalf("package finder platform = %q, want linux", finder.platform)
	}
	status, ok := store.Get("build")
	if !ok || status.Artifacts["linux-amd64"] != artifact {
		t.Fatalf("artifact result = %#v", status)
	}
}
