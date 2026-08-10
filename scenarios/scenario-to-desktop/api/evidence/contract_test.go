package evidence

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func validManifest(profile deliveryramp.Profile, state deliveryramp.RunState) deliveryramp.Manifest {
	now := time.Date(2026, 8, 5, 18, 0, 0, 0, time.UTC)
	gates := make([]deliveryramp.GateResult, 0, len(deliveryramp.RequiredGates(profile)))
	for _, name := range deliveryramp.RequiredGates(profile) {
		gates = append(gates, deliveryramp.GateResult{Name: name, Disposition: deliveryramp.GatePassed, Required: true, StartedAt: now, CompletedAt: now.Add(time.Second)})
	}
	return deliveryramp.Manifest{
		SchemaVersion: deliveryramp.ManifestSchemaVersion, RunID: "run-1", Profile: profile, State: state,
		Target:     deliveryramp.EvidenceTarget{Ramp: "scenario-to-desktop", Platform: "linux", OS: "linux", DeviceKind: "host"},
		Runner:     deliveryramp.Runner{ID: "runner-1", Kind: "local", HostOS: "linux", TargetOS: "linux", Isolation: "xvfb"},
		Provenance: deliveryramp.Provenance{ArtifactDigest: "sha256:artifact", GitCommit: "commit", StartedAt: now, CompletedAt: now.Add(time.Minute)},
		Timeline:   deliveryramp.TimelineSummary{Version: "journey-evidence.v2", JourneyRef: "capture:journey-1", ChapterIDs: []string{"launch"}, EventCount: 1, Ordered: true, RedactionStatus: "verified"},
		Gates:      gates,
		Artifacts:  []deliveryramp.Artifact{{ImmutableRef: "capture:recording-1", LocalPath: "/tmp/recording.mp4", Kind: "recording", Checksum: "sha256:capture", SizeBytes: 42, Width: 1280, Height: 720, DurationMs: 3000, Container: "mp4", Codec: "h264", UsefulFrames: true, CreatedAt: now}},
	}
}

func TestEvidenceStateTransitionsRejectSkippedGates(t *testing.T) {
	if !deliveryramp.StateCreated.CanTransitionTo(deliveryramp.StateProtocolReady) || !deliveryramp.StateArtifactsPersisted.CanTransitionTo(deliveryramp.StateGovernanceReported) {
		t.Fatal("expected normal evidence transitions to be allowed")
	}
	for _, invalid := range [][2]deliveryramp.RunState{{deliveryramp.StateCreated, deliveryramp.StateJourneyPassed}, {deliveryramp.StateProtocolReady, deliveryramp.StatePassed}, {deliveryramp.StatePassed, deliveryramp.StateVisualLaunched}, {deliveryramp.StateFailed, deliveryramp.StatePassed}} {
		if invalid[0].CanTransitionTo(invalid[1]) {
			t.Errorf("unexpected transition %q -> %q", invalid[0], invalid[1])
		}
	}
}

func TestReleaseVisualManifestRoundTripsAndValidates(t *testing.T) {
	original := validManifest(deliveryramp.ProfileReleaseVisual, deliveryramp.StatePassed)
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded deliveryramp.Manifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := decoded.Validate(); err != nil {
		t.Fatalf("round-tripped manifest should validate: %v", err)
	}
	if !strings.Contains(string(data), `"artifact_digest":"sha256:artifact"`) || !strings.Contains(string(data), `"checksum":"sha256:capture"`) {
		t.Fatalf("serialized manifest lost identity fields: %s", data)
	}
}

func TestVisualManifestCannotPassWithProtocolOnly(t *testing.T) {
	manifest := validManifest(deliveryramp.ProfileVisual, deliveryramp.StatePassed)
	manifest.Gates = manifest.Gates[:1]
	if err := manifest.Validate(); err == nil || !strings.Contains(err.Error(), "required gate") {
		t.Fatalf("expected missing visual gates to fail, got %v", err)
	}
}

func TestManifestRejectsMalformedAndUnverifiedArtifacts(t *testing.T) {
	cases := []struct {
		name string
		edit func(*deliveryramp.Manifest)
		want string
	}{
		{"relative path", func(m *deliveryramp.Manifest) { m.Artifacts[0].LocalPath = "recording.mp4" }, "absolute"},
		{"empty media", func(m *deliveryramp.Manifest) { m.Artifacts[0].SizeBytes = 0 }, "positive size"},
		{"blank recording", func(m *deliveryramp.Manifest) { m.Artifacts[0].UsefulFrames = false }, "useful frames"},
		{"duplicate gate", func(m *deliveryramp.Manifest) { m.Gates = append(m.Gates, m.Gates[0]) }, "duplicate gate"},
		{"non-terminal", func(m *deliveryramp.Manifest) { m.State = deliveryramp.StateVisualLaunched }, "not terminal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manifest := validManifest(deliveryramp.ProfileVisual, deliveryramp.StatePassed)
			tc.edit(&manifest)
			if err := manifest.Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestProtocolManifestDoesNotRequireVisualArtifacts(t *testing.T) {
	manifest := validManifest(deliveryramp.ProfileProtocol, deliveryramp.StatePassed)
	manifest.Artifacts = nil
	if err := manifest.Validate(); err != nil {
		t.Fatalf("protocol manifest should validate without visual artifacts: %v", err)
	}
}

func TestVisualManifestRejectsMissingRequiredWorkflowReference(t *testing.T) {
	manifest := validManifest(deliveryramp.ProfileVisual, deliveryramp.StatePassed)
	manifest.Timeline.WorkflowRequired = true
	if err := manifest.Validate(); err == nil || !strings.Contains(err.Error(), "required workflow reference") {
		t.Fatalf("expected missing workflow reference to fail, got %v", err)
	}
}

func TestVisualManifestAcceptsWorkflowReferenceBoundToCell(t *testing.T) {
	manifest := validManifest(deliveryramp.ProfileVisual, deliveryramp.StatePassed)
	manifest.Target.ID, manifest.CellID, manifest.Timeline.WorkflowRequired = "target-linux-xvfb", "cell-1", true
	manifest.Timeline.Workflow = &deliveryramp.WorkflowManifestReference{Provider: "workflow-provider", AssetID: "asset-1", ExecutionID: "execution-1", RunID: "run-1", ArtifactDigest: "sha256:artifact", TargetID: "target-linux-xvfb", CellID: "cell-1", Disposition: "pass", Artifacts: []deliveryramp.WorkflowManifestArtifact{{ID: "workflow-video", Kind: "video", URI: "evidence/workflow-video.mp4", Checksum: "sha256:workflow", Redacted: true}}}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("linked workflow reference should validate: %v", err)
	}
}

func TestVisualManifestRejectsRequiredFailedWorkflow(t *testing.T) {
	manifest := validManifest(deliveryramp.ProfileVisual, deliveryramp.StatePassed)
	manifest.Target.ID, manifest.CellID, manifest.Timeline.WorkflowRequired = "target-linux-xvfb", "cell-1", true
	manifest.Timeline.Workflow = &deliveryramp.WorkflowManifestReference{Provider: "workflow-provider", AssetID: "asset-1", ExecutionID: "execution-1", RunID: "run-1", ArtifactDigest: "sha256:artifact", TargetID: "target-linux-xvfb", CellID: "cell-1", Disposition: "failed"}
	if err := manifest.Validate(); err == nil || !strings.Contains(err.Error(), "disposition") {
		t.Fatalf("expected required failed workflow to fail, got %v", err)
	}
}
