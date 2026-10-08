package main

import (
	"strings"
	"testing"

	"scenario-to-cloud/bundle"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/edge"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/vps"
)

// TestEdgeSpecIdempotency verifies that the edge spec the executor hands
// the target owner is deterministic for equal inputs (same digest, same
// snippet) and changes only when the routed domain or upstream changes; the
// target owner compares digests to decide whether a reload is needed.
// [REQ:STC-IDEM-001] Edge route application is idempotent
// [REQ:STC-IDEM-002] Edge spec rendering is deterministic
func TestEdgeSpecIdempotency(t *testing.T) {
	manifest := domain.CloudManifest{Target: domain.ManifestTarget{VPS: &domain.ManifestVPS{Workdir: "/root/Vrooli"}}, Scenario: domain.ManifestScenario{ID: "app"}, Edge: domain.ManifestEdge{Domain: "example.com", Caddy: domain.ManifestCaddy{Enabled: true}}}
	cc := vps.CommandContext{DeploymentID: "dep-1", ScenarioID: "app", Manifest: manifest}
	first, err := vps.EdgeSpecFor(cc, map[string]string{"domain": "example.com", "upstream_port": "3000"})
	if err != nil {
		t.Fatal(err)
	}
	second, _ := vps.EdgeSpecFor(cc, map[string]string{"domain": "example.com", "upstream_port": "3000"})
	if first.Digest == "" || first.Digest != second.Digest || edge.RenderSnippet(first) != edge.RenderSnippet(second) {
		t.Fatalf("edge spec must be deterministic: %s vs %s", first.Digest, second.Digest)
	}
	for _, changed := range []map[string]string{{"domain": "old-domain.com", "upstream_port": "3000"}, {"domain": "example.com", "upstream_port": "3001"}} {
		other, err := vps.EdgeSpecFor(cc, changed)
		if err != nil {
			t.Fatal(err)
		}
		if other.Digest == first.Digest {
			t.Fatalf("a changed route must change the digest: %v", changed)
		}
	}
	if !strings.Contains(edge.RenderSnippet(first), "example.com {") || !strings.Contains(edge.RenderSnippet(first), "reverse_proxy 127.0.0.1:3000") {
		t.Fatalf("snippet = %q", edge.RenderSnippet(first))
	}
}

// TestScopedStopIsIdempotent verifies that the scoped lifecycle stop the
// executor dispatches is the same receipted invocation on every attempt: a
// replay under the same (operation, step) is a receipt replay, never a
// second kill.
// [REQ:STC-IDEM-004] Scenario stop is idempotent
func TestScopedStopIsIdempotent(t *testing.T) {
	cc := vps.CommandContext{DeploymentID: "dep", ScenarioID: "test-scenario", Identity: vps.Identity{OperationID: "op-1", Fence: 1}}
	action := execplan.Action{ID: execplan.OpWorkloadStop, OwnerOperation: execplan.OpWorkloadStop, Inputs: map[string]string{"workdir": "/root/Vrooli", "scenario": "test-scenario", "ports": "api=3001,ui=3000"}}
	first, err := vps.ActionCommands(action, cc)
	if err != nil {
		t.Fatal(err)
	}
	second, err := vps.ActionCommands(action, cc)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || strings.Join(first[0].Command.Argv(), " ") != strings.Join(second[0].Command.Argv(), " ") {
		t.Fatalf("stop invocation must be identical across attempts: %v vs %v", first, second)
	}
	if first[0].Step != execplan.OpWorkloadStop || !first[0].Command.Effectful {
		t.Fatalf("stop must be a receipted step: %+v", first[0])
	}
}

// TestDeleteBundleIdempotent verifies that DeleteBundle can be called
// multiple times on the same hash without error.
// [REQ:STC-IDEM-005] Bundle deletion is idempotent
func TestDeleteBundleIdempotent(t *testing.T) {
	// DeleteBundle already returns 0 bytes and nil error for missing files
	// This is tested implicitly in bundle.go via os.IsNotExist check

	// Test with empty dir (no bundles)
	tempDir := t.TempDir()

	// First delete - file doesn't exist
	freed1, err := bundle.DeleteBundle(tempDir, "abc123")
	if err != nil {
		t.Errorf("First DeleteBundle failed: %v", err)
	}
	if freed1 != 0 {
		t.Errorf("Expected 0 bytes freed, got %d", freed1)
	}

	// Second delete - still doesn't exist, should still succeed
	freed2, err := bundle.DeleteBundle(tempDir, "abc123")
	if err != nil {
		t.Errorf("Second DeleteBundle failed: %v", err)
	}
	if freed2 != 0 {
		t.Errorf("Expected 0 bytes freed, got %d", freed2)
	}
}
