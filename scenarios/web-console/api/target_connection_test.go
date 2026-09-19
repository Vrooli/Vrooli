package main

import (
	"testing"

	"github.com/vrooli/api-core/targetmodel"
)

// TestTargetByIDKeepsTheConnection pins the defect that made every remote
// machine unusable: the lookup returned the selected Target without its
// Bridge URL and owner credential, so sessions opened with no Authorization
// header and installs failed with "Bridge URL is not configured".
func TestTargetByIDKeepsTheConnection(t *testing.T) {
	remote := targetConnection{
		Target:          targetmodel.Target{ID: "bridge-node:n1", NodeID: "n1", Available: true},
		BaseURL:         "http://bridge.test",
		BaseURLExplicit: true,
		OwnerToken:      "LocalSession owner",
		ReauthToken:     "reauth",
	}
	srv := &Server{remoteTargetCatalog: func() []targetConnection { return []targetConnection{remote} }}

	got, ok := srv.targetByID("bridge-node:n1")
	if !ok {
		t.Fatal("target not found")
	}
	if got.BaseURL != remote.BaseURL || !got.BaseURLExplicit || got.OwnerToken != remote.OwnerToken || got.ReauthToken != remote.ReauthToken {
		t.Fatalf("targetByID dropped the connection: %+v", got)
	}
	if got.NodeID != "n1" {
		t.Fatalf("NodeID = %q, want n1", got.NodeID)
	}
}

// A node capability that is not a coding agent must not reach the browser
// under the agent prefix, or the launcher offers to "install" it.
func TestNodeCapabilitiesAreNotPresentedAsAgents(t *testing.T) {
	for identity, want := range map[string]string{
		"capability:opencode":           "capability:opencode",
		"capability:claude":             "capability:claude",
		"capability:bridge-provisioner": NodeCapabilityPrefix + "bridge-provisioner",
		// Machine health readings are their own group, never "missing features".
		"capability:node-health.disk":                NodeHealthPrefix + "disk",
		"capability:node-health.bootstrap-artifacts": NodeHealthPrefix + "bootstrap-artifacts",
		"heartbeat": "heartbeat",
	} {
		if got := browserFactKey(identity); got != want {
			t.Errorf("browserFactKey(%q) = %q, want %q", identity, got, want)
		}
	}
}
