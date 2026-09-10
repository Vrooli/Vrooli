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
