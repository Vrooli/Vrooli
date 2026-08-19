package codecs

import (
	"slices"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

func TestAntigravityBuildArgsUsesRunnerDefaultSymbol(t *testing.T) {
	c := NewAntigravityForTest()
	args := c.BuildArgs(c.NewState(), runner.ExecuteRequest{
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeAntigravity, Model: "default"},
		Prompt:         "safe fixture",
	})
	if !slices.Equal(args, []string{"--print", "safe fixture"}) {
		t.Fatalf("default model args = %q, want no explicit model flag", args)
	}

	args = c.BuildArgs(c.NewState(), runner.ExecuteRequest{
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeAntigravity, Model: "gemini-test"},
		Prompt:         "safe fixture",
	})
	if !slices.Equal(args, []string{"--print", "safe fixture", "--model", "gemini-test"}) {
		t.Fatalf("explicit model args = %q", args)
	}
}

func TestAntigravityCapabilitiesAreHonestForPrintMode(t *testing.T) {
	caps := NewAntigravityForTest().Capabilities()
	if !caps.SupportsMessages || !caps.SupportsStreaming || !caps.SupportsContinuation || !caps.SupportsCancellation || !caps.SupportsRunnerDefault {
		t.Fatalf("basic capabilities = %+v", caps)
	}
	if caps.SupportsToolEvents || caps.SupportsCostTracking || caps.SupportsImageAttachments || caps.SupportsToolRestriction || caps.SupportsEffort {
		t.Fatalf("unstable print capabilities claimed: %+v", caps)
	}
}
