package hostreqkit

import (
	"encoding/json"
	"testing"
)

func TestToolManifestAcquisitionRoundTrips(t *testing.T) {
	raw := `{
		"name": "sd",
		"description": "stable-diffusion.cpp",
		"commands": ["sd"],
		"versionArgs": ["--help"],
		"acquisition": {
			"kind": "url",
			"targets": [{
				"when": {"os": "linux", "arch": "amd64"},
				"url": "https://example.com/sd-linux.tar.gz",
				"sha256": "abc123",
				"archive": "tar.gz",
				"bin_path": "build/bin/sd",
				"mode": "0755"
			}]
		},
		"requires": {
			"gpu": true,
			"minVramGb": 4,
			"arch": ["amd64", "arm64"],
			"minRamGb": 8
		}
	}`

	var manifest ToolManifest
	if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if manifest.SourceType() != "url" {
		t.Fatalf("SourceType = %q, want url", manifest.SourceType())
	}
	target, ok := TargetFor(manifest.Acquisition, "linux", "amd64")
	if !ok {
		t.Fatalf("TargetFor(linux, amd64) not found")
	}
	if target.URL != "https://example.com/sd-linux.tar.gz" || target.SHA256 != "abc123" {
		t.Fatalf("unexpected target: %+v", target)
	}
	if target.Archive != "tar.gz" || target.BinPath != "build/bin/sd" || target.Mode != "0755" {
		t.Fatalf("unexpected archive/binPath/mode: %+v", target)
	}
	if manifest.Requires == nil || manifest.Requires.GPU == nil || !*manifest.Requires.GPU {
		t.Fatalf("requires.gpu not parsed: %+v", manifest.Requires)
	}
	if manifest.Requires.MinVRAMGb != 4 || manifest.Requires.MinRAMGb != 8 {
		t.Fatalf("requires vram/ram not parsed: %+v", manifest.Requires)
	}

	// Round-trip back to JSON and parse again to confirm tags are stable.
	out, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var again ToolManifest
	if err := json.Unmarshal(out, &again); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if again.SourceType() != "url" {
		t.Fatalf("round-trip SourceType = %q", again.SourceType())
	}
}

func TestToolManifestAcquisitionKindDefaultsToPackage(t *testing.T) {
	var manifest ToolManifest
	if err := json.Unmarshal([]byte(`{"name":"jq","description":"x","commands":["jq"],"versionArgs":["--version"],"defaultPackage":"jq"}`), &manifest); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if manifest.SourceType() != "package" {
		t.Fatalf("SourceType = %q, want package (absent source)", manifest.SourceType())
	}
	if _, ok := TargetFor(manifest.Acquisition, "linux", "amd64"); ok {
		t.Fatalf("TargetFor on nil source should report false")
	}
}

func TestToolSourceTargetMisses(t *testing.T) {
	source := &ToolSource{
		Kind:    "url",
		Targets: []ToolSourceTarget{{When: map[string]string{"os": "linux", "arch": "amd64"}, URL: "u", SHA256: "s"}},
	}
	if _, ok := TargetFor(source, "darwin", "arm64"); ok {
		t.Fatalf("TargetFor(darwin, arm64) should miss")
	}
}
