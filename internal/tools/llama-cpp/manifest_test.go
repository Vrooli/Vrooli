package conformance

import (
	"encoding/json"
	"io/fs"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	alltools "github.com/vrooli/vrooli/internal/tools"
)

func TestManifestPinsInspectedLinuxCaptionBundle(t *testing.T) {
	data, err := fs.ReadFile(alltools.Manifests, "llama-cpp/tool.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest hostreqkit.ToolManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	var support struct {
		PlatformStatus map[string]struct {
			Status   string `json:"status"`
			Evidence string `json:"evidence"`
		} `json:"platform_status"`
	}
	if err := json.Unmarshal(data, &support); err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "llama-cpp" || manifest.Manual {
		t.Fatalf("managed tool identity/manual flag = %q / %v", manifest.Name, manifest.Manual)
	}
	if len(manifest.Commands) != 2 || manifest.Commands[0] != "llama-mtmd-cli" || manifest.Commands[1] != "llama-cli" {
		t.Fatalf("caption runtime command contract = %#v", manifest.Commands)
	}
	if manifest.Acquisition == nil {
		t.Fatal("expected a checksum-pinned official acquisition")
	}
	target, ok := hostreqkit.TargetFor(manifest.Acquisition, "linux", "amd64")
	if !ok {
		t.Fatal("expected the inspected Linux amd64 release target")
	}
	if target.URL != "https://github.com/ggml-org/llama.cpp/releases/download/b12224/llama-b12224-bin-ubuntu-x64.tar.gz" ||
		target.SHA256 != "e3f9e11e61b8bcecccc4b7522d131cc6d3ec18fc3f31f6caa58ddda63801bbd5" ||
		target.Archive != "tar.gz" || target.Layout != "dir" || target.BinPath != "llama-b12224/llama-mtmd-cli" {
		t.Fatalf("release target differs from the inspected official archive: %+v", target)
	}
	if support.PlatformStatus["linux"].Status != "experimental" {
		t.Fatalf("Linux platform claim = %+v, want experimental pending caption inference", support.PlatformStatus["linux"])
	}
	for _, platform := range []string{"macos", "windows"} {
		if support.PlatformStatus[platform].Status != "unqualified" {
			t.Fatalf("%s platform claim = %+v, want unqualified", platform, support.PlatformStatus[platform])
		}
	}
	for _, key := range []struct{ os, arch string }{{"linux", "arm64"}, {"darwin", "amd64"}, {"darwin", "arm64"}, {"windows", "amd64"}} {
		if _, ok := hostreqkit.TargetFor(manifest.Acquisition, key.os, key.arch); ok {
			t.Errorf("unexpected uninspected download target for %s/%s", key.os, key.arch)
		}
		if reason, ok := hostreqkit.UnsupportedFor(manifest.Acquisition, key.os, key.arch); !ok || !strings.Contains(reason, "unqualified") {
			t.Errorf("%s/%s should explain its unqualified status, got %q / %v", key.os, key.arch, reason, ok)
		}
	}
}
