package main

import "testing"

func TestBootstrapScriptCandidatesPreferDeclaredRoots(t *testing.T) {
	got := bootstrapScriptCandidates(
		"/tmp/api",
		"/srv/vrooli-bridge",
		"/srv/legacy-scenario",
		"/srv/vrooli",
		"/tmp/bin/vrooli-bridge-api",
	)
	if len(got) == 0 || got[0] != "/srv/vrooli-bridge/bootstrap/bootstrap.sh" {
		t.Fatalf("first bootstrap candidate = %v, want scenario root", got)
	}
	if got[0] == "/tmp/api/bootstrap/bootstrap.sh" {
		t.Fatal("working-directory candidate must not precede declared roots")
	}
}

func TestCanonicalControlPlaneEndpointPrecedence(t *testing.T) {
	t.Setenv("BRIDGE_CONTROL_PLANE_URL", "https://configured.example.test")
	t.Setenv("BRIDGE_TUNNEL_URL", "https://tunnel.example.test")
	if got, source := canonicalControlPlaneEndpoint(); got != "https://configured.example.test" || source != "configured" {
		t.Fatalf("configured endpoint = %q (%s)", got, source)
	}
	t.Setenv("BRIDGE_CONTROL_PLANE_URL", "")
	if got, source := canonicalControlPlaneEndpoint(); got != "https://tunnel.example.test" || source != "tunnel" {
		t.Fatalf("tunnel endpoint = %q (%s)", got, source)
	}
}
