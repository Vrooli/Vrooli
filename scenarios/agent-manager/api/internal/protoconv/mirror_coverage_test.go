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

func TestDiffResultMirrorCoversSource(t *testing.T) {
	assertMirrorCoversSource(t, sandbox.DiffResult{}, protoconv.DiffResult{})
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
		SupportsMessages:         true,
		SupportsToolEvents:       true,
		SupportsCostTracking:     true,
		SupportsStreaming:        true,
		SupportsCancellation:     true,
		SupportsContinuation:     true,
		SupportsWarmIteration:    true,
		SupportsImageAttachments: true,
		SupportsToolRestriction:  true,
		ToolRestrictionMappings:  map[string]string{"read": "Read"},
		SupportsEffort:           true,
		EffortMappings:           map[string]string{"high": "xhigh"},
		EffortModelSpecific:      true,
		MaxTurns:                 17,
		SupportedModels:          []string{"provider/model"},
		SupportsRunnerDefault:    true,
		DynamicModelPrefixes:     []string{"ollama/"},
		SupportedFeatures:        []string{"EnableBrowser"},
		AllowedExtraFlags:        []string{"--verbose"},
	}

	status := protoconv.OrchestratorRunnerStatusToProto(&protoconv.OrchestratorRunnerStatus{
		Type: domain.RunnerTypeCodex, Capabilities: want,
	})
	got := protoconv.RunnerCapabilitiesFromProto(status.Capabilities, status.SupportedModels)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("capability round trip mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}
