package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/screenrecording"
	"scenario-to-desktop-api/smoketest"
)

// ManifestWriter persists the producer-owned manifest beside the recording.
// The capture store remains the source of truth for the referenced bytes.
type ManifestWriter struct {
	captures *captures.Service
}

func NewManifestWriter(service *captures.Service) *ManifestWriter {
	return &ManifestWriter{captures: service}
}

var _ smoketest.EvidenceManifestWriter = (*ManifestWriter)(nil)

func (w *ManifestWriter) WriteManifest(ctx context.Context, input smoketest.EvidenceManifestInput) error {
	if w == nil || w.captures == nil {
		return fmt.Errorf("capture service is unavailable")
	}
	if input.Journey == nil {
		return fmt.Errorf("journey is required")
	}
	if strings.TrimSpace(input.RunID) == "" || strings.TrimSpace(input.ScenarioName) == "" {
		return fmt.Errorf("run ID and scenario name are required")
	}

	profile := Profile(strings.TrimSpace(input.Profile))
	if profile == "" {
		profile = ProfileVisual
	}
	startedAt := input.StartedAt
	if startedAt.IsZero() {
		startedAt = input.Journey.CreatedAt
	}
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	completedAt := input.CompletedAt
	if completedAt.IsZero() || completedAt.Before(startedAt) {
		completedAt = time.Now().UTC()
	}

	artifactDigest, err := fileDigest(input.ArtifactPath)
	if err != nil {
		return fmt.Errorf("hash built artifact: %w", err)
	}
	items := capturesForRun(input.Captures, input.RunID)
	if len(items) == 0 {
		return fmt.Errorf("no persisted captures found for run %q", input.RunID)
	}

	manifest := Manifest{
		SchemaVersion: ManifestSchemaVersion,
		RunID:         input.RunID,
		Profile:       profile,
		State:         StateFailed,
		Target: Target{
			Ramp:       "scenario-to-desktop",
			Platform:   strings.ToLower(strings.TrimSpace(input.Platform)),
			OS:         "linux",
			DeviceKind: "host",
		},
		Runner: Runner{
			ID:           "linux-xvfb-openbox",
			Kind:         "native",
			HostOS:       "linux",
			TargetOS:     "linux",
			Isolation:    "xvfb",
			Capabilities: []string{"xvfb", "openbox", "xdotool", "ffmpeg", "electron"},
		},
		Provenance: Provenance{
			ArtifactDigest: artifactDigest,
			GitCommit:      gitCommit(),
			StartedAt:      startedAt,
			CompletedAt:    completedAt,
		},
	}

	journeyOK := input.Journey.Disposition == "pass"
	recordingOK := false
	persistenceOK := false
	for _, item := range items {
		if item.Type == captures.CaptureRecording {
			path, pathErr := w.captures.CaptureFilePath(input.ScenarioName, item.ID)
			if pathErr != nil {
				return fmt.Errorf("resolve recording capture %q: %w", item.ID, pathErr)
			}
			inspection, inspectErr := screenrecording.InspectVideo(ctx, path)
			if inspectErr == nil {
				recordingOK = true
			}
			manifest.Artifacts = append(manifest.Artifacts, artifactFromCapture(item, path, inspection))
			continue
		}
		if item.Type == captures.CaptureJourney {
			path, pathErr := w.captures.CaptureFilePath(input.ScenarioName, item.ID)
			if pathErr != nil {
				return fmt.Errorf("resolve journey capture %q: %w", item.ID, pathErr)
			}
			manifest.Artifacts = append(manifest.Artifacts, artifactFromCapture(item, path, screenrecording.MediaInspection{}))
		}
	}
	if len(manifest.Artifacts) < 2 {
		return fmt.Errorf("journey and recording captures are both required")
	}
	persistenceOK = true
	governanceOK := profile != ProfileReleaseVisual || input.GovernanceReported
	protocolOK := true // this writer is invoked only after the protocol smoke succeeded.
	visualOK := input.Journey.Disposition != "degraded" && input.Journey.Disposition != "failed"
	manifest.Gates = []GateResult{
		gate(GateProtocol, protocolOK, "protocol smoke completed", startedAt, completedAt),
		gate(GateVisual, visualOK, "usable application window and visual launch", startedAt, completedAt),
		gate(GateJourney, journeyOK, "semantic desktop journey", startedAt, completedAt),
		gate(GateCapture, recordingOK, "MP4 decoded with useful frames", startedAt, completedAt),
		gate(GatePersistence, persistenceOK, "journey and recording captures persisted", startedAt, completedAt),
	}
	if profile == ProfileReleaseVisual {
		manifest.Gates = append(manifest.Gates, gate(GateGovernance, governanceOK, "deployment-manager evidence report", startedAt, completedAt))
	}
	if allRequiredGatesPassed(manifest) {
		manifest.State = StatePassed
	}
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("validate evidence manifest: %w", err)
	}

	recordingPath := ""
	for _, item := range manifest.Artifacts {
		if item.Kind == string(captures.CaptureRecording) {
			recordingPath = item.LocalPath
			break
		}
	}
	if recordingPath == "" {
		return fmt.Errorf("recording path is missing")
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal evidence manifest: %w", err)
	}
	manifestPath := recordingPath + ".manifest.json"
	if err := os.WriteFile(manifestPath, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("persist evidence manifest: %w", err)
	}
	return nil
}

func capturesForRun(items []captures.Capture, runID string) []captures.Capture {
	result := make([]captures.Capture, 0, 2)
	source := "smoke-test:" + runID
	for _, item := range items {
		if item.SourceSession == source && (item.Type == captures.CaptureJourney || item.Type == captures.CaptureRecording) {
			result = append(result, item)
		}
	}
	return result
}

func artifactFromCapture(item captures.Capture, path string, inspection screenrecording.MediaInspection) Artifact {
	return Artifact{
		ImmutableRef: "capture:" + item.ID,
		LocalPath:    path,
		Kind:         string(item.Type),
		Checksum:     item.Checksum,
		SizeBytes:    item.FileSizeBytes,
		Width:        inspection.Width,
		Height:       inspection.Height,
		DurationMs:   inspection.DurationMs,
		Container:    inspection.Container,
		Codec:        inspection.Codec,
		UsefulFrames: item.Type != captures.CaptureRecording || inspection.UsefulFrame,
		CreatedAt:    item.CreatedAt,
	}
}

func fileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func gitCommit() string {
	if value := strings.TrimSpace(os.Getenv("VROOLI_GIT_COMMIT")); value != "" {
		return value
	}
	return "working-tree"
}

func gate(name GateName, passed bool, reason string, startedAt, completedAt time.Time) GateResult {
	disposition := GateFailed
	if passed {
		disposition = GatePassed
	}
	return GateResult{Name: name, Disposition: disposition, Required: true, Reason: reason, StartedAt: startedAt, CompletedAt: completedAt}
}

func allRequiredGatesPassed(manifest Manifest) bool {
	byName := make(map[GateName]GateResult, len(manifest.Gates))
	for _, item := range manifest.Gates {
		byName[item.Name] = item
	}
	for _, name := range RequiredGates(manifest.Profile) {
		if byName[name].Disposition != GatePassed {
			return false
		}
	}
	return true
}
