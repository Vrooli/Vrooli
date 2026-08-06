package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	if err := smoketest.ValidateJourneyTimeline(*input.Journey); err != nil {
		return fmt.Errorf("validate journey timeline: %w", err)
	}
	manifest.Timeline = TimelineSummary{
		Version:         input.Journey.EvidenceVersion,
		Capability:      input.Journey.Capability,
		EventCount:      len(input.Journey.Events),
		Ordered:         true,
		RedactionStatus: "verified",
	}
	if input.Journey.ProviderObservation != nil {
		manifest.Timeline.ProviderTier = input.Journey.ProviderObservation.ProviderTier
		manifest.Timeline.SafeRouteClass = input.Journey.ProviderObservation.SafeRouteClass
		manifest.Timeline.FallbackDecision = input.Journey.ProviderObservation.FallbackDecision
	}
	if manifest.Timeline.Version == "" {
		manifest.Timeline.Version = "journey-evidence.v1"
	}
	for _, step := range input.Journey.Steps {
		chapterID := step.ID
		if chapterID == "" {
			chapterID = step.Name
		}
		manifest.Timeline.ChapterIDs = append(manifest.Timeline.ChapterIDs, chapterID)
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
			manifest.Timeline.JourneyRef = "capture:" + item.ID
			manifest.Artifacts = append(manifest.Artifacts, artifactFromCapture(item, path, screenrecording.MediaInspection{}))
		}
	}
	if len(manifest.Artifacts) < 2 {
		return fmt.Errorf("journey and recording captures are both required")
	}
	for _, tracePath := range []struct {
		path    string
		kind    string
		runKind smoketest.LaunchRunKind
	}{
		{path: input.ProtocolTracePath, kind: "protocol_launch_trace", runKind: smoketest.LaunchRunProtocol},
		{path: input.DemoTracePath, kind: "demo_launch_trace", runKind: smoketest.LaunchRunDemo},
	} {
		if strings.TrimSpace(tracePath.path) == "" {
			continue
		}
		trace, traceErr := readValidatedLaunchTrace(tracePath.path, tracePath.runKind)
		if traceErr != nil {
			manifest.Performance.Status = "degraded"
			manifest.Performance.Reason = traceErr.Error()
			continue
		}
		artifact, traceErr := performanceArtifact(tracePath.path, recordingPathFor(manifest.Artifacts))
		if traceErr != nil {
			manifest.Performance.Status = "degraded"
			manifest.Performance.Reason = traceErr.Error()
			continue
		}
		artifact.Kind = tracePath.kind
		manifest.Artifacts = append(manifest.Artifacts, artifact)
		manifest.Performance.TraceRefs = append(manifest.Performance.TraceRefs, artifact.ImmutableRef)
		phases := LaunchPhaseDurations(trace)
		if tracePath.runKind == smoketest.LaunchRunProtocol {
			manifest.Performance.ProtocolPhases = phases
		} else {
			manifest.Performance.DemoPhases = phases
		}
	}
	for _, profileDir := range []string{input.ProtocolProfileDir, input.DemoProfileDir} {
		if strings.TrimSpace(profileDir) == "" {
			continue
		}
		entries, readErr := os.ReadDir(profileDir)
		if readErr != nil {
			if input.ProfileMode != "" && input.ProfileMode != "disabled" {
				manifest.Performance.Status = "degraded"
				manifest.Performance.Reason = fmt.Sprintf("profile directory unavailable: %v", readErr)
			}
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			artifact, profileErr := performanceArtifact(filepath.Join(profileDir, entry.Name()), recordingPathFor(manifest.Artifacts))
			if profileErr != nil {
				manifest.Performance.Status = "degraded"
				manifest.Performance.Reason = profileErr.Error()
				continue
			}
			artifact.Kind = "profile"
			manifest.Artifacts = append(manifest.Artifacts, artifact)
			manifest.Performance.ProfileRefs = append(manifest.Performance.ProfileRefs, artifact.ImmutableRef)
		}
	}
	manifest.Performance.ProtocolSummary = input.ProtocolResourceSummary
	manifest.Performance.DemoSummary = input.DemoResourceSummary
	manifest.Performance.DemoProcessTree = input.DemoProcessTree
	if manifest.Performance.Status == "" {
		switch {
		case input.ProfileMode != "" && input.ProfileMode != "disabled" && len(manifest.Performance.ProfileRefs) == 0:
			manifest.Performance.Status = "degraded"
			manifest.Performance.Reason = "profiling was requested but no profile artifact was available"
		case len(manifest.Performance.TraceRefs) == 2 && input.ProtocolResourceSummary != nil && input.DemoResourceSummary != nil:
			manifest.Performance.Status = "measured"
		case len(manifest.Performance.TraceRefs) > 0:
			manifest.Performance.Status = "degraded"
			manifest.Performance.Reason = "one or more performance phases lack complete metrics"
		default:
			manifest.Performance.Status = "unavailable"
			manifest.Performance.Reason = "launch traces were not persisted"
		}
	}
	persistenceOK = true
	governanceOK := profile != ProfileReleaseVisual || input.GovernanceReported
	protocolOK := true // this writer is invoked only after the protocol smoke succeeded.
	visualOK := input.Journey.Disposition == "pass"
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

func readValidatedLaunchTrace(path string, expectedKind smoketest.LaunchRunKind) (smoketest.LaunchTrace, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return smoketest.LaunchTrace{}, fmt.Errorf("read launch trace %q: %w", path, err)
	}
	var trace smoketest.LaunchTrace
	if err := json.Unmarshal(data, &trace); err != nil {
		return smoketest.LaunchTrace{}, fmt.Errorf("decode launch trace %q: %w", path, err)
	}
	if trace.RunKind != expectedKind {
		return smoketest.LaunchTrace{}, fmt.Errorf("launch trace %q has run kind %q, want %q", path, trace.RunKind, expectedKind)
	}
	if err := trace.Validate(); err != nil {
		return smoketest.LaunchTrace{}, fmt.Errorf("validate launch trace %q: %w", path, err)
	}
	return trace, nil
}

func validateLaunchTraceFile(path string, expectedKind smoketest.LaunchRunKind) error {
	_, err := readValidatedLaunchTrace(path, expectedKind)
	return err
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

func recordingPathFor(artifacts []Artifact) string {
	for _, artifact := range artifacts {
		if artifact.Kind == string(captures.CaptureRecording) {
			return artifact.LocalPath
		}
	}
	return ""
}

func performanceArtifact(sourcePath, recordingPath string) (Artifact, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return Artifact{}, fmt.Errorf("read performance artifact %q: %w", sourcePath, err)
	}
	if len(data) == 0 {
		return Artifact{}, fmt.Errorf("performance artifact %q is empty", sourcePath)
	}
	digest := sha256.Sum256(data)
	checksum := "sha256:" + hex.EncodeToString(digest[:])
	destination := sourcePath
	if recordingPath != "" {
		destination = recordingPath + "." + filepath.Base(filepath.Dir(sourcePath)) + "." + filepath.Base(sourcePath)
		if err := os.WriteFile(destination, data, 0o600); err != nil {
			return Artifact{}, fmt.Errorf("persist performance artifact: %w", err)
		}
	}
	return Artifact{
		ImmutableRef: "performance:" + checksum,
		LocalPath:    destination,
		Kind:         "performance",
		Checksum:     checksum,
		SizeBytes:    int64(len(data)),
		UsefulFrames: true,
		CreatedAt:    time.Now().UTC(),
	}, nil
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
