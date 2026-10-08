//nolint:goconst // test data deliberately reuses stable manifest fixtures.
package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/tools"
)

// TestGoToolchainContract keeps the exact release pin in one authority:
// internal/tools/go/tool.json. The root module requests that same toolchain
// and both GitHub workflows resolve their setup-go input from the manifest at
// runtime, so a security bump cannot silently leave local and CI builds apart.
func TestGoToolchainContract(t *testing.T) {
	data, err := tools.Manifests.ReadFile("go/tool.json")
	if err != nil {
		t.Fatalf("read Go tool manifest: %v", err)
	}
	var manifest hostreqkit.ToolManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse Go tool manifest: %v", err)
	}
	if manifest.Version == "" {
		t.Fatal("Go tool manifest must declare an exact version")
	}
	if manifest.SourceType() != "url" || manifest.Acquisition == nil {
		t.Fatalf("Go tool acquisition = %#v; want checksum-verified URL targets", manifest.Acquisition)
	}
	for _, targetName := range []string{"linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64"} {
		parts := strings.SplitN(targetName, "/", 2)
		target, ok := hostreqkit.TargetFor(manifest.Acquisition, parts[0], parts[1])
		if !ok || target.Layout != "dir" || target.Archive == "" || target.BinPath == "" || len(target.SHA256) != 64 || target.RuntimeEnv["GOROOT"] != "go" {
			t.Errorf("Go target %s = %#v; want a complete verified directory release", targetName, target)
		}
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate repository root")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "../.."))
	goMod, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		t.Fatalf("read root go.mod: %v", err)
	}
	if !strings.Contains(string(goMod), "toolchain go"+manifest.Version) {
		t.Errorf("root go.mod must contain toolchain go%s (manifest version)", manifest.Version)
	}

	for _, workflow := range []string{".github/workflows/test.yml", ".github/workflows/vrooli-release.yml"} {
		contents, err := os.ReadFile(filepath.Join(repoRoot, workflow))
		if err != nil {
			t.Errorf("read %s: %v", workflow, err)
			continue
		}
		text := string(contents)
		if !strings.Contains(text, "id: go-version") ||
			!strings.Contains(text, "jq -r '.version' internal/tools/go/tool.json") ||
			!strings.Contains(text, "go-version: ${{ steps.go-version.outputs.value }}") {
			t.Errorf("%s must resolve actions/setup-go's version from internal/tools/go/tool.json", workflow)
		}
	}
}
