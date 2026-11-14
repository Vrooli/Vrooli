package installgateway

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/tools"
)

func systemToolFixture(t *testing.T, name string) string {
	t.Helper()
	root := t.TempDir()
	raw, err := tools.Manifests.ReadFile(name + "/tool.json")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "internal", "tools", name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tool.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestSystemToolUsesDeclaredOwnerAndRefusesChangedResolution(t *testing.T) {
	root := systemToolFixture(t, "tesseract")
	r, err := Resolve(root, "tesseract", "host-tool", "system", "tesseract", "")
	if err != nil {
		t.Fatal(err)
	}
	if r.Command() != "vrooli host install tesseract --json" || r.HostManifestSHA256 == "" || r.Profile.LifecycleMode != "system-host-tool" || r.Profile.FrozenLockfile {
		t.Fatalf("incorrect system policy: %+v", r)
	}
	for _, args := range [][3]string{{"../tesseract", "tesseract", ""}, {"tesseract", "curl", ""}, {"tesseract", "tesseract", "5.3.4"}} {
		if _, err := Resolve(root, args[0], "host-tool", "system", args[1], args[2]); err == nil {
			t.Fatalf("unowned system route admitted: %v", args)
		}
	}
	changed := r
	changed.Argv = []string{"sh", "injected"}
	if _, err := (ExecInstaller{}).Install(context.Background(), changed); err == nil {
		t.Fatal("changed executable reached host owner")
	}
	if err := os.WriteFile(r.ManifestPath, []byte(`{"name":"tesseract","defaultPackage":"other"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (ExecInstaller{}).Install(context.Background(), r); err == nil {
		t.Fatal("changed package identity reached host owner")
	}
}

func TestSystemToolRefusesManualAndReleaseOwnedTools(t *testing.T) {
	for _, name := range []string{"sd", "uv"} {
		root := systemToolFixture(t, name)
		if _, err := Resolve(root, name, "host-tool", "system", name, ""); err == nil {
			t.Fatalf("%s escaped its native acquisition owner", name)
		}
	}
}
