package protoconv_test

import (
	"reflect"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/protoconv"
)

func assertMirrorCoversSource(t *testing.T, source, mirror any) {
	t.Helper()
	sourceType := reflect.TypeOf(source)
	mirrorType := reflect.TypeOf(mirror)
	for i := 0; i < sourceType.NumField(); i++ {
		field := sourceType.Field(i)
		if _, ok := mirrorType.FieldByName(field.Name); !ok {
			t.Errorf("%s does not mirror %s.%s", mirrorType.Name(), sourceType.Name(), field.Name)
		}
	}
}

func TestStopAllResultMirrorCoversSource(t *testing.T) {
	assertMirrorCoversSource(t, orchestration.StopAllResult{}, protoconv.StopAllResult{})
}

func TestApproveResultMirrorCoversSource(t *testing.T) {
	assertMirrorCoversSource(t, orchestration.ApproveResult{}, protoconv.ApproveResult{})
}

func TestProbeResultMirrorCoversSource(t *testing.T) {
	assertMirrorCoversSource(t, orchestration.ProbeResult{}, protoconv.ProbeResult{})
}

func TestDiffResultMirrorCoversConversionFields(t *testing.T) {
	// PatchSHA256 and EvidenceComplete are sandbox-provider evidence used by
	// orchestration. They intentionally stop before this conversion mirror and
	// the public RunDiff wire projection; FileCountVerified is the public review
	// assurance bit copied by this conversion.
	source := reflect.TypeOf(sandbox.DiffResult{})
	mirror := reflect.TypeOf(protoconv.DiffResult{})
	intentionalProviderOnly := map[string]string{
		"PatchSHA256":      "exact provider patch evidence is not part of public RunDiff",
		"EvidenceComplete": "provider completeness proof is internal-only",
	}
	for i := 0; i < source.NumField(); i++ {
		field := source.Field(i)
		if _, ok := mirror.FieldByName(field.Name); ok {
			continue
		}
		if reason, ok := intentionalProviderOnly[field.Name]; ok && reason != "" {
			delete(intentionalProviderOnly, field.Name)
			continue
		}
		t.Errorf("%s does not mirror conversion field %s.%s", mirror.Name(), source.Name(), field.Name)
	}
	for field, reason := range intentionalProviderOnly {
		t.Errorf("documented provider-only field %s was removed or renamed (%s)", field, reason)
	}
}

func TestFileChangeMirrorCoversSource(t *testing.T) {
	assertMirrorCoversSource(t, sandbox.FileChange{}, protoconv.FileChange{})
}

func TestOrchestratorRunnerStatusMirrorCoversSource(t *testing.T) {
	assertMirrorCoversSource(t, orchestration.RunnerStatus{}, protoconv.OrchestratorRunnerStatus{})
}

func TestRunnerCapabilitiesMirrorCoversSource(t *testing.T) {
	assertMirrorCoversSource(t, runner.Capabilities{}, protoconv.RunnerCapabilities{})
}

func TestRunnerCapabilitiesProtoRoundTrip(t *testing.T) {
	want := protoconv.RunnerCapabilities{
		SpawnCapabilities: []protoconv.SpawnCapability{{
			ExecutionMode: "interactive", SandboxModes: []string{"tracking", "off"}, NativeObjective: true,
		}},
		SupportsMessages:          true,
		SupportsToolEvents:        true,
		SupportsCostTracking:      true,
		SupportsStreaming:         true,
		SupportsCancellation:      true,
		SupportsContinuation:      true,
		SupportsWarmIteration:     true,
		SupportsSessionCompaction: true,
		SupportsImageAttachments:  true,
		SupportsToolRestriction:   true,
		ToolRestrictionMappings:   map[string]string{"read": "Read"},
		SupportsEffort:            true,
		EffortMappings:            map[string]string{"high": "xhigh"},
		EffortModelSpecific:       true,
		MaxTurns:                  17,
		SupportedModels:           []string{"provider/model"},
		SupportsRunnerDefault:     true,
		DynamicModelPrefixes:      []string{"ollama/"},
		SupportedFeatures:         []string{"EnableBrowser"},
		AllowedExtraFlags:         []string{"--verbose"},
	}

	status := protoconv.OrchestratorRunnerStatusToProto(&protoconv.OrchestratorRunnerStatus{
		Type: domain.RunnerTypeCodex, Capabilities: want,
	})
	got := protoconv.RunnerCapabilitiesFromProto(status.Capabilities, status.SupportedModels)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("capability round trip mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}
