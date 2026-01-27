package dochealing

import "testing"

func TestHealRequestNormalize(t *testing.T) {
	req := HealRequest{
		ScenarioName:   "  alpha ",
		Issues:         []string{"  fix docs  ", " ", "\t"},
		TimeoutSeconds: 0,
	}
	if err := req.normalize(); err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if req.ScenarioName != "alpha" {
		t.Fatalf("expected scenario alpha, got %q", req.ScenarioName)
	}
	if len(req.Issues) != 1 || req.Issues[0] != "fix docs" {
		t.Fatalf("unexpected issues: %+v", req.Issues)
	}
	if req.TimeoutSeconds == 0 {
		t.Fatalf("expected default timeout to be set")
	}
}

func TestHealRequestNormalizeRequiresScenario(t *testing.T) {
	req := HealRequest{}
	if err := req.normalize(); err == nil {
		t.Fatalf("expected error for missing scenario")
	}
}
