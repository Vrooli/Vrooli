package domain

import (
	"encoding/json"
	"testing"
)

func TestManifestDependenciesParsesProgramBindingPeers(t *testing.T) {
	var manifest CloudManifest
	if err := json.Unmarshal([]byte(`{"version":"1","environment":"test","target":{"type":"vps"},"scenario":{"id":"demo"},"dependencies":{"program_binding_peers":["vrooli-memory"]},"bundle":{},"ports":{},"edge":{}}`), &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Dependencies.ProgramBindingPeers) != 1 || manifest.Dependencies.ProgramBindingPeers[0] != "vrooli-memory" {
		t.Fatalf("program binding peers = %#v", manifest.Dependencies.ProgramBindingPeers)
	}
}

func TestTargetRefFromManifestCarriesComputeManagerIdentity(t *testing.T) {
	manifest := CloudManifest{Target: ManifestTarget{Type: "vps", VPS: &ManifestVPS{
		Host: "127.0.0.1", Port: 2222, User: "vrooli", Workdir: "/srv/vrooli",
		ComputeManagerInstanceID: "qemu-abc123",
	}}}
	target := TargetRefFromManifest(manifest)
	if target.ComputeManagerID != "qemu-abc123" {
		t.Fatalf("manager identity = %q", target.ComputeManagerID)
	}
	if target.Locator.Host != "127.0.0.1" || target.Locator.Port != 2222 || target.Locator.User != "vrooli" || target.Locator.Workdir != "/srv/vrooli" {
		t.Fatalf("SSH locator = %+v", target.Locator)
	}
}
