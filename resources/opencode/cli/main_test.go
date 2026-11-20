package main

import (
	"encoding/json"
	"os"
	"testing"
)

// TestUpstreamPinnedVersionMatchesManifest prevents the resource CLI's
// compatibility constant from drifting away from the manifest-owned baseline.
func TestUpstreamPinnedVersionMatchesManifest(t *testing.T) {
	data, err := os.ReadFile("../resource.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Acquisition struct {
			Targets []struct {
				Channel string `json:"channel"`
				Package string `json:"package"`
				Version string `json:"version"`
			} `json:"targets"`
		} `json:"acquisition"`
		UpstreamCLI struct {
			VersionPinned string `json:"version_pinned"`
		} `json:"upstream_cli"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.UpstreamCLI.VersionPinned != upstreamPinnedVersion {
		t.Fatalf("upstream_cli.version_pinned = %q, CLI constant = %q", manifest.UpstreamCLI.VersionPinned, upstreamPinnedVersion)
	}
	var sawDefault, sawV2 bool
	for _, target := range manifest.Acquisition.Targets {
		if target.Channel == "" {
			sawDefault = true
			if target.Package != "opencode-ai" {
				t.Fatalf("default channel package = %q, want opencode-ai", target.Package)
			}
			if target.Version != upstreamPinnedVersion {
				t.Fatalf("default channel pin = %q, want %q", target.Version, upstreamPinnedVersion)
			}
		}
		if target.Channel == "v2" {
			sawV2 = true
			if target.Package != "@opencode/cli" {
				t.Fatalf("v2 channel package = %q, want @opencode/cli", target.Package)
			}
		}
	}
	if !sawDefault || !sawV2 {
		t.Fatalf("manifest must declare both the default and v2 channels (default=%v v2=%v)", sawDefault, sawV2)
	}
}
