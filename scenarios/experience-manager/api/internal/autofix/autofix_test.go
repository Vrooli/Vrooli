package autofix

import "testing"

func TestNewRegistryHasNoPhaseOneCandidates(t *testing.T) {
	reg := NewRegistry()
	resp, err := reg.PreviewFixResponse("demo", t.TempDir(), nil)
	if err != nil {
		t.Fatalf("PreviewFixResponse: %v", err)
	}
	if resp.GetScenario() != "demo" {
		t.Fatalf("scenario = %q", resp.GetScenario())
	}
	if len(resp.GetCandidates()) != 0 {
		t.Fatalf("expected no candidates, got %d", len(resp.GetCandidates()))
	}
}
