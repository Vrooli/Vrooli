package evidence

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func validManifest(profile Profile, state RunState) Manifest {
	now := time.Date(2026, 8, 5, 18, 0, 0, 0, time.UTC)
	gates := make([]GateResult, 0, len(RequiredGates(profile)))
	for _, name := range RequiredGates(profile) {
		gates = append(gates, GateResult{Name: name, Disposition: GatePassed, Required: true, StartedAt: now, CompletedAt: now.Add(time.Second)})
	}
	return Manifest{
		SchemaVersion: ManifestSchemaVersion,
		RunID:         "run-1",
		Profile:       profile,
		State:         state,
		Target:        Target{Ramp: "scenario-to-desktop", Platform: "linux", OS: "linux", DeviceKind: "host"},
		Runner:        Runner{ID: "runner-1", Kind: "local", HostOS: "linux", TargetOS: "linux", Isolation: "xvfb"},
		Provenance:    Provenance{ArtifactDigest: "sha256:artifact", GitCommit: "commit", StartedAt: now, CompletedAt: now.Add(time.Minute)},
		Timeline:      TimelineSummary{Version: "journey-evidence.v2", JourneyRef: "capture:journey-1", ChapterIDs: []string{"launch"}, EventCount: 1, Ordered: true, RedactionStatus: "verified"},
		Gates:         gates,
		Artifacts:     []Artifact{{ImmutableRef: "capture:recording-1", LocalPath: "/tmp/recording.mp4", Kind: "recording", Checksum: "sha256:capture", SizeBytes: 42, Width: 1280, Height: 720, DurationMs: 3000, Container: "mp4", Codec: "h264", UsefulFrames: true, CreatedAt: now}},
	}
}

func TestEvidenceStateTransitionsRejectSkippedGates(t *testing.T) {
	if !StateCreated.CanTransitionTo(StateProtocolReady) || !StateArtifactsPersisted.CanTransitionTo(StateGovernanceReported) {
		t.Fatal("expected normal evidence transitions to be allowed")
	}
	for _, invalid := range [][2]RunState{
		{StateCreated, StateJourneyPassed},
		{StateProtocolReady, StatePassed},
		{StatePassed, StateVisualLaunched},
		{StateFailed, StatePassed},
	} {
		if invalid[0].CanTransitionTo(invalid[1]) {
			t.Errorf("unexpected transition %q -> %q", invalid[0], invalid[1])
		}
	}
}

func TestReleaseVisualManifestRoundTripsAndValidates(t *testing.T) {
	original := validManifest(ProfileReleaseVisual, StatePassed)
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Manifest
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
	manifest := validManifest(ProfileVisual, StatePassed)
	manifest.Gates = manifest.Gates[:1]
	if err := manifest.Validate(); err == nil || !strings.Contains(err.Error(), "required gate") {
		t.Fatalf("expected missing visual gates to fail, got %v", err)
	}
}

func TestManifestRejectsMalformedAndUnverifiedArtifacts(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Manifest)
		want string
	}{
		{"relative path", func(m *Manifest) { m.Artifacts[0].LocalPath = "recording.mp4" }, "absolute"},
		{"empty media", func(m *Manifest) { m.Artifacts[0].SizeBytes = 0 }, "positive size"},
		{"blank recording", func(m *Manifest) { m.Artifacts[0].UsefulFrames = false }, "useful frames"},
		{"duplicate gate", func(m *Manifest) { m.Gates = append(m.Gates, m.Gates[0]) }, "duplicate gate"},
		{"non-terminal", func(m *Manifest) { m.State = StateVisualLaunched }, "not terminal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manifest := validManifest(ProfileVisual, StatePassed)
			tc.edit(&manifest)
			if err := manifest.Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestProtocolManifestDoesNotRequireVisualArtifacts(t *testing.T) {
	manifest := validManifest(ProfileProtocol, StatePassed)
	manifest.Artifacts = nil
	if err := manifest.Validate(); err != nil {
		t.Fatalf("protocol manifest should validate without visual artifacts: %v", err)
	}
}
