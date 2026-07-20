package orchestration

import (
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

func TestValidateToolRestriction(t *testing.T) {
	registry := runner.NewRegistry()
	unsupported := runner.NewMockRunner(domain.RunnerTypeCodex)
	if err := registry.Register(unsupported); err != nil {
		t.Fatal(err)
	}
	enforcing := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	caps := enforcing.Capabilities()
	caps.SupportsToolRestriction = true
	enforcing.SetCapabilities(caps)
	if err := registry.Register(enforcing); err != nil {
		t.Fatal(err)
	}
	o := New(nil, nil, nil, WithRunners(registry))

	if err := o.validateToolRestriction(&domain.RunConfig{RunnerType: domain.RunnerTypeCodex, AllowedTools: []string{"read"}}); err == nil {
		t.Fatal("unsupported enforced restriction was accepted")
	}
	if err := o.validateToolRestriction(&domain.RunConfig{RunnerType: domain.RunnerTypeCodex, AllowedTools: []string{"read"}, ToolRestrictionPolicy: domain.ToolRestrictionPolicyAdvisory}); err != nil {
		t.Fatalf("advisory restriction rejected: %v", err)
	}
	if err := o.validateToolRestriction(&domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode, AllowedTools: []string{"read"}}); err != nil {
		t.Fatalf("enforcing runner rejected: %v", err)
	}
}
