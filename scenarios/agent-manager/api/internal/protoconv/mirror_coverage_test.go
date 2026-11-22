package protoconv_test

import (
	"reflect"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/protoconv"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
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
		SupportsToolEvents:        false,
		SupportsCostTracking:      true,
		SupportsStreaming:         false,
		SupportsCancellation:      true,
		SupportsContinuation:      false,
		SupportsWarmIteration:     true,
		SupportsSessionCompaction: false,
		SupportsImageAttachments:  true,
		SupportsToolRestriction:   false,
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
	if status.GetCapabilities().GetSupportsMessages() != true || status.GetCapabilities().GetSupportsToolEvents() != false || status.GetCapabilities().GetSupportsCostTracking() != true || status.GetCapabilities().GetSupportsStreaming() != false || status.GetCapabilities().GetSupportsCancellation() != true || status.GetCapabilities().GetSupportsContinuation() != false || status.GetCapabilities().GetSupportsWarmIteration() != true || status.GetCapabilities().GetSupportsSessionCompaction() != false || status.GetCapabilities().GetSupportsImageAttachments() != true || status.GetCapabilities().GetSupportsToolRestriction() != false {
		t.Fatalf("proto runner capability flags do not match independently specified values: %+v", status.GetCapabilities())
	}
}

func TestRunnerCapabilitiesToProtoOwnsMutableFields(t *testing.T) {
	caps := protoconv.RunnerCapabilities{
		SpawnCapabilities:       []protoconv.SpawnCapability{{ExecutionMode: "interactive", SandboxModes: []string{"protected"}}},
		EffortMappings:          map[string]string{"high": "xhigh"},
		ToolRestrictionMappings: map[string]string{"read": "Read"},
		SupportedModels:         []string{"provider/model"}, DynamicModelPrefixes: []string{"ollama/"},
		SupportedFeatures: []string{"EnableBrowser"}, AllowedExtraFlags: []string{"--verbose"},
	}
	status := protoconv.OrchestratorRunnerStatusToProto(&protoconv.OrchestratorRunnerStatus{Capabilities: caps})

	caps.SpawnCapabilities[0].SandboxModes[0] = "off"
	caps.EffortMappings["high"] = "low"
	caps.ToolRestrictionMappings["read"] = "Bash"
	caps.SupportedModels[0] = "changed"
	caps.DynamicModelPrefixes[0] = "changed/"
	caps.SupportedFeatures[0] = "changed"
	caps.AllowedExtraFlags[0] = "--changed"

	got := status.GetCapabilities()
	if got.GetEffortMappings()["high"] != "xhigh" || got.GetToolRestrictionMappings()["read"] != "Read" {
		t.Fatalf("proto mappings changed with source: effort=%v tool=%v", got.GetEffortMappings(), got.GetToolRestrictionMappings())
	}
	if len(got.GetSpawnCapabilities()) != 1 || len(status.GetSupportedModels()) != 1 || len(got.GetDynamicModelPrefixes()) != 1 || len(got.GetSupportedFeatures()) != 1 || len(got.GetAllowedExtraFlags()) != 1 {
		t.Fatalf("proto capability list lengths changed with source: %+v", status)
	}
	if len(got.GetSpawnCapabilities()[0].GetSandboxModes()) != 1 || got.GetSpawnCapabilities()[0].GetSandboxModes()[0] != "protected" || status.GetSupportedModels()[0] != "provider/model" || got.GetDynamicModelPrefixes()[0] != "ollama/" || got.GetSupportedFeatures()[0] != "EnableBrowser" || got.GetAllowedExtraFlags()[0] != "--verbose" {
		t.Fatalf("proto capability lists changed with source: %+v", got)
	}
}

func TestRunnerCapabilitiesFromProtoOwnsMutableFields(t *testing.T) {
	nilCapabilityModels := []string{"provider/default"}
	nilCapabilities := protoconv.RunnerCapabilitiesFromProto(nil, nilCapabilityModels)
	nilCapabilityModels[0] = "changed"
	if nilCapabilities.SupportedModels[0] != "provider/default" {
		t.Fatalf("nil-capability model list changed with input: %+v", nilCapabilities)
	}

	wire := &pb.RunnerCapabilities{
		SpawnCapabilities:       []*pb.SpawnCapability{{ExecutionMode: "interactive", SandboxModes: []string{"protected"}}},
		EffortMappings:          map[string]string{"high": "xhigh"},
		ToolRestrictionMappings: map[string]string{"read": "Read"},
		DynamicModelPrefixes:    []string{"ollama/"}, SupportedFeatures: []string{"EnableBrowser"},
		AllowedExtraFlags: []string{"--verbose"},
	}
	models := []string{"provider/model"}
	got := protoconv.RunnerCapabilitiesFromProto(wire, models)

	wire.SpawnCapabilities[0].SandboxModes[0] = "off"
	wire.EffortMappings["high"] = "low"
	wire.ToolRestrictionMappings["read"] = "Bash"
	wire.DynamicModelPrefixes[0] = "changed/"
	wire.SupportedFeatures[0] = "changed"
	wire.AllowedExtraFlags[0] = "--changed"
	models[0] = "changed"

	if got.EffortMappings["high"] != "xhigh" || got.ToolRestrictionMappings["read"] != "Read" {
		t.Fatalf("domain mappings changed with proto: effort=%v tool=%v", got.EffortMappings, got.ToolRestrictionMappings)
	}
	if len(got.SpawnCapabilities) != 1 || len(got.SpawnCapabilities[0].SandboxModes) != 1 || len(got.SupportedModels) != 1 || len(got.DynamicModelPrefixes) != 1 || len(got.SupportedFeatures) != 1 || len(got.AllowedExtraFlags) != 1 {
		t.Fatalf("domain capability list lengths changed with proto: %+v", got)
	}
	if got.SpawnCapabilities[0].SandboxModes[0] != "protected" || got.SupportedModels[0] != "provider/model" || got.DynamicModelPrefixes[0] != "ollama/" || got.SupportedFeatures[0] != "EnableBrowser" || got.AllowedExtraFlags[0] != "--verbose" {
		t.Fatalf("domain capability lists changed with proto: %+v", got)
	}
}
