package cloudflared

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

var stubLookups = cloudflaredStubLookups

func cloudflaredStubLookups(t *testing.T) func() {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origReadFile := hostreqkit.ReadFileFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	origElevationFacts := hostreqkit.ElevationFactsFn
	origKeyDownload := KeyDownloadFn
	hostreqkit.ElevationFactsFn = func() hostreqkit.ElevationFacts {
		return hostreqkit.ElevationFacts{Platform: "linux", Elevated: true, CanElevate: true, Mechanism: "test"}
	}
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.ReadFileFn = origReadFile
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
		hostreqkit.ElevationFactsFn = origElevationFacts
		KeyDownloadFn = origKeyDownload
	}
}

var testManifest = hostreqkit.ToolManifest{
	Name:        "cloudflared",
	Description: "Cloudflare Tunnel client",
	Commands:    []string{"cloudflared"},
	VersionArgs: []string{"--version"},
	Handler:     "cloudflared",
	Packages:    map[string]string{"brew": "cloudflare/cloudflare/cloudflared", "winget": "Cloudflare.cloudflared"},
	InstallHint: "Install cloudflared",
	Platforms:   []string{"linux", "macos", "windows"},
}

var newTestHandler = cloudflaredTestHandler

func cloudflaredTestHandler() hostreqkit.Handler {
	return NewHandler(testManifest)
}

// --- Name and Kind ---

// --- Inspect tests ---

// --- Apply tests ---
