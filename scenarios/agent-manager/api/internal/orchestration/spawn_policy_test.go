package orchestration

import (
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

func TestResolveSpawnPolicyPrefersDeclaredFeasibleCombination(t *testing.T) {
	resolution, err := ResolveSpawnPolicy(&domain.SpawnPolicy{AxisOrder: []string{"sandboxMode", "executionMode"}, SandboxMode: domain.PreferenceAxis{Prefer: []string{"protected", "tracking", "off"}}, ExecutionMode: domain.PreferenceAxis{Prefer: []string{"codec_pipe", "interactive"}}}, runner.Capabilities{SpawnCapabilities: []runner.SpawnCapability{{ExecutionMode: "codec_pipe", SandboxModes: []string{"tracking"}}, {ExecutionMode: "interactive", SandboxModes: []string{"tracking"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.ExecutionMode != "codec_pipe" || resolution.SandboxMode != "tracking" {
		t.Fatalf("resolution=%+v", resolution)
	}
}

func TestResolveSpawnPolicyNamesUnsatisfiableRequirement(t *testing.T) {
	_, err := ResolveSpawnPolicy(&domain.SpawnPolicy{AxisOrder: []string{"sandboxMode", "executionMode"}, SandboxMode: domain.PreferenceAxis{Prefer: []string{"protected"}}, ExecutionMode: domain.PreferenceAxis{Prefer: []string{"interactive"}}, Require: []domain.SpawnCombination{{ExecutionMode: "interactive", SandboxMode: "protected"}}}, runner.Capabilities{SpawnCapabilities: []runner.SpawnCapability{{ExecutionMode: "interactive", SandboxModes: []string{"tracking"}}}})
	if err == nil {
		t.Fatal("expected named unsatisfiable requirement")
	}
}

func TestResolveSpawnPolicyFallsBackToClaudeInteractiveTracking(t *testing.T) {
	policy := &domain.SpawnPolicy{
		AxisOrder:     []string{"sandboxMode", "executionMode"},
		SandboxMode:   domain.PreferenceAxis{Prefer: []string{"protected", "tracking", "off"}},
		ExecutionMode: domain.PreferenceAxis{Prefer: []string{"codec_pipe", "interactive"}},
	}
	resolution, err := ResolveSpawnPolicy(policy, runner.Capabilities{SpawnCapabilities: []runner.SpawnCapability{{ExecutionMode: "interactive", SandboxModes: []string{"tracking", "off"}, NativeObjective: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.ExecutionMode != "interactive" || resolution.SandboxMode != "tracking" || !resolution.NativeObjective {
		t.Fatalf("resolution=%+v", resolution)
	}
}

func TestResolveSpawnPolicyRecordsDeclaredFallbackWhenNoPreferenceFeasible(t *testing.T) {
	policy := &domain.SpawnPolicy{
		AxisOrder:     []string{"executionMode", "sandboxMode"},
		ExecutionMode: domain.PreferenceAxis{Prefer: []string{"interactive"}},
		SandboxMode:   domain.PreferenceAxis{Prefer: []string{"tracking"}},
	}
	// The runner declares only codec_pipe/protected, so no preferred
	// combination is feasible; the declared capability is used and recorded.
	resolution, err := ResolveSpawnPolicy(policy, runner.Capabilities{SpawnCapabilities: []runner.SpawnCapability{{ExecutionMode: "codec_pipe", SandboxModes: []string{"protected"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.ExecutionMode != "codec_pipe" || resolution.SandboxMode != "protected" {
		t.Fatalf("resolution=%+v", resolution)
	}
	if resolution.Fallback != "codec_pipe/protected" {
		t.Fatalf("fallback not recorded: %q", resolution.Fallback)
	}
}

func TestResolveSpawnPolicyZeroCapabilitiesUsesCodecPipeDefault(t *testing.T) {
	policy := &domain.SpawnPolicy{
		AxisOrder:     []string{"sandboxMode", "executionMode"},
		SandboxMode:   domain.PreferenceAxis{Prefer: []string{"off"}},
		ExecutionMode: domain.PreferenceAxis{Prefer: []string{"interactive"}},
	}
	resolution, err := ResolveSpawnPolicy(policy, runner.Capabilities{})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.ExecutionMode != "codec_pipe" || resolution.SandboxMode != "off" {
		t.Fatalf("resolution=%+v", resolution)
	}
}

func TestResolveSpawnPolicyZeroCapabilitiesStillFailsRequiredMode(t *testing.T) {
	policy := &domain.SpawnPolicy{
		AxisOrder:     []string{"sandboxMode", "executionMode"},
		SandboxMode:   domain.PreferenceAxis{Prefer: []string{"tracking"}},
		ExecutionMode: domain.PreferenceAxis{Prefer: []string{"interactive"}},
		Require:       []domain.SpawnCombination{{ExecutionMode: "interactive", SandboxMode: "tracking"}},
	}
	if _, err := ResolveSpawnPolicy(policy, runner.Capabilities{}); err == nil {
		t.Fatal("expected required spawn mode to fail closed")
	}
}

func TestResolveSpawnPolicyHonorsExplicitRunAxesBeforeProfilePreferences(t *testing.T) {
	policy := &domain.SpawnPolicy{
		AxisOrder:     []string{"executionMode", "sandboxMode"},
		ExecutionMode: domain.PreferenceAxis{Prefer: []string{"interactive", "codec_pipe"}},
		SandboxMode:   domain.PreferenceAxis{Prefer: []string{"off", "tracking", "protected"}},
	}
	runMode := domain.RunModeSandboxed
	constraints := SpawnPolicyConstraints{ExecutionMode: "codec_pipe", SandboxMode: "protected", RunMode: &runMode}
	resolution, err := ResolveSpawnPolicyWithConstraints(policy, runner.Capabilities{SpawnCapabilities: []runner.SpawnCapability{
		{ExecutionMode: "interactive", SandboxModes: []string{"off", "tracking"}},
		{ExecutionMode: "codec_pipe", SandboxModes: []string{"protected"}},
	}}, constraints)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.ExecutionMode != "codec_pipe" || resolution.SandboxMode != "protected" {
		t.Fatalf("explicit caller axes were replaced by profile preference: %+v", resolution)
	}
}

func TestResolveSpawnPolicyFiltersPartialExplicitAxisBeforeRanking(t *testing.T) {
	policy := &domain.SpawnPolicy{
		AxisOrder:     []string{"executionMode", "sandboxMode"},
		ExecutionMode: domain.PreferenceAxis{Prefer: []string{"interactive", "codec_pipe"}},
		SandboxMode:   domain.PreferenceAxis{Prefer: []string{"protected", "off"}},
	}
	resolution, err := ResolveSpawnPolicyWithConstraints(policy, runner.Capabilities{SpawnCapabilities: []runner.SpawnCapability{
		{ExecutionMode: "interactive", SandboxModes: []string{"off"}},
		{ExecutionMode: "codec_pipe", SandboxModes: []string{"protected"}},
	}}, SpawnPolicyConstraints{SandboxMode: "protected"})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.ExecutionMode != "codec_pipe" || resolution.SandboxMode != "protected" {
		t.Fatalf("partial explicit sandbox selector was not applied before ranking: %+v", resolution)
	}
}

func TestResolveSpawnPolicyRejectsExplicitSelectorsWithoutCompatibleCapability(t *testing.T) {
	policy := &domain.SpawnPolicy{
		AxisOrder:     []string{"executionMode", "sandboxMode"},
		ExecutionMode: domain.PreferenceAxis{Prefer: []string{"interactive", "codec_pipe"}},
		SandboxMode:   domain.PreferenceAxis{Prefer: []string{"off", "protected"}},
	}
	_, err := ResolveSpawnPolicyWithConstraints(policy, runner.Capabilities{SpawnCapabilities: []runner.SpawnCapability{{ExecutionMode: "interactive", SandboxModes: []string{"off"}}}}, SpawnPolicyConstraints{ExecutionMode: "codec_pipe", SandboxMode: "protected"})
	if err == nil {
		t.Fatal("expected incompatible explicit caller selectors to be refused")
	}
}

func TestResolveSpawnPolicyKeepsRunModeAndForceInPlaceConstraints(t *testing.T) {
	policy := &domain.SpawnPolicy{
		AxisOrder:     []string{"sandboxMode", "executionMode"},
		SandboxMode:   domain.PreferenceAxis{Prefer: []string{"off", "protected"}},
		ExecutionMode: domain.PreferenceAxis{Prefer: []string{"interactive", "codec_pipe"}},
	}
	caps := runner.Capabilities{SpawnCapabilities: []runner.SpawnCapability{
		{ExecutionMode: "interactive", SandboxModes: []string{"off"}},
		{ExecutionMode: "codec_pipe", SandboxModes: []string{"protected"}},
	}}
	sandboxed := domain.RunModeSandboxed
	resolution, err := ResolveSpawnPolicyWithConstraints(policy, caps, SpawnPolicyConstraints{RunMode: &sandboxed})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.SandboxMode == string(domain.SandboxModeOff) {
		t.Fatalf("explicit sandboxed run mode was changed to in-place: %+v", resolution)
	}
	inPlace := domain.RunModeInPlace
	resolution, err = ResolveSpawnPolicyWithConstraints(policy, caps, SpawnPolicyConstraints{RunMode: &inPlace, ForceInPlace: true})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.SandboxMode != string(domain.SandboxModeOff) {
		t.Fatalf("explicit in-place mode was changed to sandboxed: %+v", resolution)
	}
	if _, err := ResolveSpawnPolicyWithConstraints(policy, caps, SpawnPolicyConstraints{RunMode: &sandboxed, ForceInPlace: true}); err == nil {
		t.Fatal("expected conflicting ForceInPlace and explicit sandboxed mode to fail")
	}
	if _, err := ResolveSpawnPolicyWithConstraints(policy, caps, SpawnPolicyConstraints{SandboxMode: "protected", ForceInPlace: true}); err == nil {
		t.Fatal("expected ForceInPlace and protected mode to fail")
	}
}

func TestSpawnPolicyConstraintsCaptureOnlyCallerSelectedAxes(t *testing.T) {
	runMode := domain.RunModeSandboxed
	constraints := spawnPolicyConstraintsForRequest(CreateRunRequest{
		ExecutionMode: "codec_pipe",
		RunMode:       &runMode,
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
	})
	if constraints.ExecutionMode != "codec_pipe" || constraints.SandboxMode != "protected" || constraints.RunMode == nil || *constraints.RunMode != runMode || constraints.ForceInPlace {
		t.Fatalf("caller selectors not captured: %+v", constraints)
	}
	defaults := spawnPolicyConstraintsForRequest(CreateRunRequest{})
	if defaults.ExecutionMode != "" || defaults.SandboxMode != "" || defaults.RunMode != nil || defaults.ForceInPlace {
		t.Fatalf("absent selectors were incorrectly treated as caller constraints: %+v", defaults)
	}
}

func TestResolveSpawnPolicyExplicitSelectorsCannotBypassRequire(t *testing.T) {
	policy := &domain.SpawnPolicy{
		AxisOrder:     []string{"executionMode", "sandboxMode"},
		ExecutionMode: domain.PreferenceAxis{Prefer: []string{"codec_pipe", "interactive"}},
		SandboxMode:   domain.PreferenceAxis{Prefer: []string{"protected", "tracking"}},
		Require:       []domain.SpawnCombination{{ExecutionMode: "interactive", SandboxMode: "tracking"}},
	}
	capabilities := runner.Capabilities{SpawnCapabilities: []runner.SpawnCapability{
		{ExecutionMode: "codec_pipe", SandboxModes: []string{"protected"}},
		{ExecutionMode: "interactive", SandboxModes: []string{"tracking"}},
	}}
	_, err := ResolveSpawnPolicyWithConstraints(policy, capabilities, SpawnPolicyConstraints{ExecutionMode: "codec_pipe", SandboxMode: "protected"})
	if err == nil {
		t.Fatal("explicit caller selectors must not bypass the profile's required combination")
	}
}
