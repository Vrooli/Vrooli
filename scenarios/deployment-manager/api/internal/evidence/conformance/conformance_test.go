package conformance

import (
	"strings"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestReferenceProducerConformsForHostEmulatorAndBridgeTargets(t *testing.T) {
	// [REQ:DM-P0-038] Producer-shaped verdicts conform to the shared contract.
	producer := FakeProducer{Producer: "scenario-to-desktop"}
	tests := []struct {
		name   string
		target *commonv1.EvidenceTarget
	}{
		{name: "local host", target: &commonv1.EvidenceTarget{Ramp: "local", Platform: "linux", Os: "linux", DeviceKind: commonv1.DeviceKind_DEVICE_KIND_HOST}},
		{name: "emulator", target: &commonv1.EvidenceTarget{Ramp: "pre-release", Platform: "android", Os: "android", DeviceKind: commonv1.DeviceKind_DEVICE_KIND_EMULATOR}},
		{name: "bridge node", target: &commonv1.EvidenceTarget{Ramp: "release", Platform: "windows", Os: "windows", DeviceKind: commonv1.DeviceKind_DEVICE_KIND_PHYSICAL, BridgeNodeId: stringPtr("node-1")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verdict, err := producer.Verdict(tt.target, commonv1.Disposition_DISPOSITION_PASSED, "run-1")
			if err != nil {
				t.Fatal(err)
			}
			if violations := Validate(verdict); len(violations) != 0 {
				t.Fatalf("unexpected violations: %v", violations)
			}
		})
	}
}

func TestValidateRejectsArtifactPathAndInvalidFields(t *testing.T) {
	// [REQ:DM-P0-038] Reference-only evidence rejects artifact bytes and invalid references.
	verdict := &commonv1.TargetVerdict{Target: &commonv1.EvidenceTarget{Ramp: "local", Platform: "linux", Os: "linux", DeviceKind: commonv1.DeviceKind_DEVICE_KIND_EMULATOR}, RunId: "run"}
	verdict.Refs = []*commonv1.EvidenceRef{{Producer: "producer", ArtifactId: "/tmp/evidence.mp4", Kind: "video/mp4", Checksum: "sha256:x", SizeBytes: -1}}
	violations := Validate(verdict)
	if len(violations) < 2 {
		t.Fatalf("expected invalid timestamp and size violations, got %v", violations)
	}
	for _, violation := range violations {
		if strings.Contains(strings.ToLower(violation.Reason), "path") {
			t.Fatalf("path must never be accepted: %v", violation)
		}
	}
}

func stringPtr(value string) *string { return &value }
