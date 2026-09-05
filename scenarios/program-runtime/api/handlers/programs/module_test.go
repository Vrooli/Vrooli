package programs

import "testing"

func TestEndpointsAreDeclared(t *testing.T) {
	if len(Endpoints) != 10 {
		t.Fatalf("endpoints=%d", len(Endpoints))
	}
}

func TestDiscoveryEvalEndpointIsDeclared(t *testing.T) {
	for _, endpoint := range Endpoints {
		if endpoint.ID == "programs_discovery_eval" {
			return
		}
	}
	t.Fatal("programs_discovery_eval is not declared; the CLI binding would bypass endpoint parity")
}

// TestWaitEndpointIsDeclared pins the block-once primitive to the declared
// surface. It is named separately from the count so a future change that
// removes it fails with a reason rather than an arithmetic mismatch.
func TestWaitEndpointIsDeclared(t *testing.T) {
	for _, endpoint := range Endpoints {
		if endpoint.ID == "programs_wait" {
			return
		}
	}
	t.Fatal("programs_wait is not declared; callers would fall back to polling GetProgram")
}
