package main

import "testing"

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
