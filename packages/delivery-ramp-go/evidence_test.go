package deliveryramp

import (
	"testing"
	"time"
)

func TestVisualManifestRequiresOrderedRedactedTimelineAndUsefulRecording(t *testing.T) {
	now := time.Date(2026, 8, 10, 13, 0, 0, 0, time.UTC)
	manifest := Manifest{
		SchemaVersion: ManifestSchemaVersion,
		RunID:         "run-1", Profile: ProfileVisual, State: StatePassed,
		Target:     EvidenceTarget{ID: "local", Ramp: "reference", Platform: "linux-amd64", OS: "linux"},
		Runner:     Runner{ID: "runner", TargetOS: "linux"},
		Provenance: Provenance{ArtifactDigest: "sha256:artifact", StartedAt: now, CompletedAt: now.Add(time.Second)},
		Timeline:   TimelineSummary{Version: JourneyEvidenceVersion, JourneyRef: "capture:journey", ChapterIDs: []string{"launch"}, Ordered: true, RedactionStatus: "verified"},
		Gates:      []GateResult{{Name: GateProtocol, Disposition: GatePassed, Required: true}, {Name: GateVisual, Disposition: GatePassed, Required: true}, {Name: GateJourney, Disposition: GatePassed, Required: true}, {Name: GateCapture, Disposition: GatePassed, Required: true}, {Name: GatePersistence, Disposition: GatePassed, Required: true}},
		Artifacts:  []Artifact{{ImmutableRef: "capture:recording", Kind: "recording", Checksum: "sha256:recording", SizeBytes: 42, Width: 1280, Height: 720, DurationMs: 1000, Container: "mp4", Codec: "h264", UsefulFrames: true, CreatedAt: now}},
	}
	if err := manifest.Validate(); err != nil {
		t.Fatal(err)
	}
	manifest.Timeline.RedactionStatus = "unverified"
	if err := manifest.Validate(); err == nil {
		t.Fatal("unverified timeline was accepted")
	}
}

func TestProtocolManifestDoesNotRequireVisualArtifacts(t *testing.T) {
	now := time.Date(2026, 8, 10, 13, 0, 0, 0, time.UTC)
	manifest := Manifest{
		SchemaVersion: ManifestSchemaVersion, RunID: "run-1", Profile: ProfileProtocol, State: StatePassed,
		Target: EvidenceTarget{ID: "local", Ramp: "reference", Platform: "linux-amd64", OS: "linux"}, Runner: Runner{ID: "runner", TargetOS: "linux"},
		Provenance: Provenance{ArtifactDigest: "sha256:artifact", StartedAt: now, CompletedAt: now.Add(time.Second)},
		Gates:      []GateResult{{Name: GateProtocol, Disposition: GatePassed, Required: true}},
	}
	if err := manifest.Validate(); err != nil {
		t.Fatal(err)
	}
}
