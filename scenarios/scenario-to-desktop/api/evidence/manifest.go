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

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
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
	profile, startedAt, completedAt, artifactDigest, items, err := prepareManifestInput(w, input)
	if err != nil {
		return err
	}
	manifest := newManifest(input, profile, artifactDigest, startedAt, completedAt)
	if err := populateTimeline(&manifest, input); err != nil {
		return err
	}
	recordingOK, err := w.appendCaptureArtifacts(ctx, &manifest, input, items)
	if err != nil {
		return err
	}
	if len(manifest.Artifacts) < 2 {
		return fmt.Errorf("journey and recording captures are both required")
	}
	appendPerformanceArtifacts(&manifest, input)
	setPerformanceSummary(&manifest, input)
	manifest.Gates = manifestGates(input, profile, recordingOK, startedAt, completedAt)
	if allRequiredGatesPassed(manifest) {
		manifest.State = deliveryramp.StatePassed
	}
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("validate evidence manifest: %w", err)
	}
	return persistManifest(manifest)
}

func prepareManifestInput(w *ManifestWriter, input smoketest.EvidenceManifestInput) (deliveryramp.Profile, time.Time, time.Time, string, []captures.Capture, error) {
	if w == nil || w.captures == nil {
		return "", time.Time{}, time.Time{}, "", nil, fmt.Errorf("capture service is unavailable")
	}
	if input.Journey == nil {
		return "", time.Time{}, time.Time{}, "", nil, fmt.Errorf("journey is required")
	}
	if strings.TrimSpace(input.RunID) == "" || strings.TrimSpace(input.ScenarioName) == "" {
		return "", time.Time{}, time.Time{}, "", nil, fmt.Errorf("run ID and scenario name are required")
	}
	profile := deliveryramp.Profile(strings.TrimSpace(input.Profile))
	if profile == "" {
		profile = deliveryramp.ProfileVisual
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
		return "", time.Time{}, time.Time{}, "", nil, fmt.Errorf("hash built artifact: %w", err)
	}
	items := capturesForRun(input.Captures, input.RunID)
	if len(items) == 0 {
		return "", time.Time{}, time.Time{}, "", nil, fmt.Errorf("no persisted captures found for run %q", input.RunID)
	}
	return profile, startedAt, completedAt, artifactDigest, items, nil
}

func newManifest(input smoketest.EvidenceManifestInput, profile deliveryramp.Profile, artifactDigest string, startedAt, completedAt time.Time) deliveryramp.Manifest {
	return deliveryramp.Manifest{
		SchemaVersion: deliveryramp.ManifestSchemaVersion, RunID: input.RunID, Profile: profile, State: deliveryramp.StateFailed,
		Target:     deliveryramp.EvidenceTarget{ID: input.Journey.TargetID, Ramp: "scenario-to-desktop", Platform: strings.ToLower(strings.TrimSpace(input.Platform)), OS: "linux", DeviceKind: "host"},
		CellID:     input.Journey.CellID,
		Runner:     deliveryramp.Runner{ID: "linux-xvfb-openbox", Kind: "native", HostOS: "linux", TargetOS: "linux", Isolation: "xvfb", Capabilities: []string{"xvfb", "openbox", "xdotool", "ffmpeg", "electron"}},
		Provenance: deliveryramp.Provenance{ArtifactDigest: artifactDigest, GitCommit: gitCommit(), StartedAt: startedAt, CompletedAt: completedAt},
	}
}

func populateTimeline(manifest *deliveryramp.Manifest, input smoketest.EvidenceManifestInput) error {
	if err := smoketest.ValidateJourneyTimeline(*input.Journey); err != nil {
		return fmt.Errorf("validate journey timeline: %w", err)
	}
	journey := input.Journey
	manifest.Timeline = deliveryramp.TimelineSummary{Version: journey.EvidenceVersion, Capability: journey.Capability, EventCount: len(journey.Events), Ordered: true, RedactionStatus: "verified", WorkflowRequired: journey.WorkflowRequired}
	workflowReference := input.WorkflowReference
	if workflowReference == nil {
		workflowReference = journey.WorkflowReference
	}
	if workflowReference != nil {
		manifest.Timeline.Workflow = &deliveryramp.WorkflowManifestReference{Provider: workflowReference.Provider, AssetID: workflowReference.AssetID, ExecutionID: workflowReference.ExecutionID, RunID: workflowReference.RunID, ArtifactDigest: workflowReference.ArtifactDigest, TargetID: workflowReference.TargetID, CellID: workflowReference.CellID, Disposition: workflowReference.Disposition}
		for _, artifact := range workflowReference.Artifacts {
			manifest.Timeline.Workflow.Artifacts = append(manifest.Timeline.Workflow.Artifacts, deliveryramp.WorkflowManifestArtifact(artifact))
		}
	}
	if journey.ProviderObservation != nil {
		manifest.Timeline.ProviderTier = journey.ProviderObservation.ProviderTier
		manifest.Timeline.SafeRouteClass = journey.ProviderObservation.SafeRouteClass
		manifest.Timeline.FallbackDecision = journey.ProviderObservation.FallbackDecision
	}
	if manifest.Timeline.Version == "" {
		manifest.Timeline.Version = "journey-evidence.v1"
	}
	for _, step := range journey.Steps {
		chapterID := step.ID
		if chapterID == "" {
			chapterID = step.Name
		}
		manifest.Timeline.ChapterIDs = append(manifest.Timeline.ChapterIDs, chapterID)
	}
	return nil
}

func (w *ManifestWriter) appendCaptureArtifacts(ctx context.Context, manifest *deliveryramp.Manifest, input smoketest.EvidenceManifestInput, items []captures.Capture) (bool, error) {
	recordingOK := false
	for _, item := range items {
		path, err := w.captures.CaptureFilePath(input.ScenarioName, item.ID)
		if err != nil {
			return false, fmt.Errorf("resolve %s capture %q: %w", item.Type, item.ID, err)
		}
		switch item.Type {
		case captures.CaptureRecording:
			inspection, inspectErr := screenrecording.InspectVideo(ctx, path)
			recordingOK = recordingOK || inspectErr == nil
			manifest.Artifacts = append(manifest.Artifacts, artifactFromCapture(item, path, inspection))
		case captures.CaptureJourney:
			manifest.Timeline.JourneyRef = "capture:" + item.ID
			manifest.Artifacts = append(manifest.Artifacts, artifactFromCapture(item, path, screenrecording.MediaInspection{}))
		}
	}
	return recordingOK, nil
}

func appendPerformanceArtifacts(manifest *deliveryramp.Manifest, input smoketest.EvidenceManifestInput) {
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
		trace, err := readValidatedLaunchTrace(tracePath.path, tracePath.runKind)
		if err != nil {
			manifest.Performance.Status, manifest.Performance.Reason = "degraded", err.Error()
			continue
		}
		artifact, err := performanceArtifact(tracePath.path, recordingPathFor(manifest.Artifacts))
		if err != nil {
			manifest.Performance.Status, manifest.Performance.Reason = "degraded", err.Error()
			continue
		}
		artifact.Kind = tracePath.kind
		manifest.Artifacts = append(manifest.Artifacts, artifact)
		manifest.Performance.TraceRefs = append(manifest.Performance.TraceRefs, artifact.ImmutableRef)
		if tracePath.runKind == smoketest.LaunchRunProtocol {
			manifest.Performance.ProtocolPhases = LaunchPhaseDurations(trace)
		} else {
			manifest.Performance.DemoPhases = LaunchPhaseDurations(trace)
		}
	}
	for _, profileDir := range []string{input.ProtocolProfileDir, input.DemoProfileDir} {
		appendProfileArtifacts(manifest, input, profileDir)
	}
}

func appendProfileArtifacts(manifest *deliveryramp.Manifest, input smoketest.EvidenceManifestInput, profileDir string) {
	if strings.TrimSpace(profileDir) == "" {
		return
	}
	entries, err := os.ReadDir(profileDir)
	if err != nil {
		if input.ProfileMode != "" && input.ProfileMode != "disabled" {
			manifest.Performance.Status, manifest.Performance.Reason = "degraded", fmt.Sprintf("profile directory unavailable: %v", err)
		}
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		artifact, err := performanceArtifact(filepath.Join(profileDir, entry.Name()), recordingPathFor(manifest.Artifacts))
		if err != nil {
			manifest.Performance.Status, manifest.Performance.Reason = "degraded", err.Error()
			continue
		}
		artifact.Kind = "profile"
		manifest.Artifacts = append(manifest.Artifacts, artifact)
		manifest.Performance.ProfileRefs = append(manifest.Performance.ProfileRefs, artifact.ImmutableRef)
	}
}

func setPerformanceSummary(manifest *deliveryramp.Manifest, input smoketest.EvidenceManifestInput) {
	manifest.Performance.ProtocolSummary, manifest.Performance.DemoSummary, manifest.Performance.DemoProcessTree = input.ProtocolResourceSummary, input.DemoResourceSummary, input.DemoProcessTree
	if manifest.Performance.Status != "" {
		return
	}
	switch {
	case input.ProfileMode != "" && input.ProfileMode != "disabled" && len(manifest.Performance.ProfileRefs) == 0:
		manifest.Performance.Status, manifest.Performance.Reason = "degraded", "profiling was requested but no profile artifact was available"
	case len(manifest.Performance.TraceRefs) == 2 && input.ProtocolResourceSummary != nil && input.DemoResourceSummary != nil:
		manifest.Performance.Status = "measured"
	case len(manifest.Performance.TraceRefs) > 0:
		manifest.Performance.Status, manifest.Performance.Reason = "degraded", "one or more performance phases lack complete metrics"
	default:
		manifest.Performance.Status, manifest.Performance.Reason = "unavailable", "launch traces were not persisted"
	}
}

func manifestGates(input smoketest.EvidenceManifestInput, profile deliveryramp.Profile, recordingOK bool, startedAt, completedAt time.Time) []deliveryramp.GateResult {
	gates := []deliveryramp.GateResult{
		gate(deliveryramp.GateProtocol, true, "protocol smoke completed", startedAt, completedAt),
		gate(deliveryramp.GateVisual, input.Journey.Disposition == "pass", "usable application window and visual launch", startedAt, completedAt),
		gate(deliveryramp.GateJourney, input.Journey.Disposition == "pass", "semantic desktop journey", startedAt, completedAt),
		gate(deliveryramp.GateCapture, recordingOK, "MP4 decoded with useful frames", startedAt, completedAt),
		gate(deliveryramp.GatePersistence, true, "journey and recording captures persisted", startedAt, completedAt),
	}
	if profile == deliveryramp.ProfileReleaseVisual {
		gates = append(gates, gate(deliveryramp.GateGovernance, input.GovernanceReported, "deployment-manager evidence report", startedAt, completedAt))
	}
	return gates
}

func persistManifest(manifest deliveryramp.Manifest) error {
	recordingPath := recordingPathFor(manifest.Artifacts)
	if recordingPath == "" {
		return fmt.Errorf("recording path is missing")
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal evidence manifest: %w", err)
	}
	if err := os.WriteFile(recordingPath+".manifest.json", append(data, '\n'), 0o600); err != nil {
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

func artifactFromCapture(item captures.Capture, path string, inspection screenrecording.MediaInspection) deliveryramp.Artifact {
	return deliveryramp.Artifact{
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

func recordingPathFor(artifacts []deliveryramp.Artifact) string {
	for _, artifact := range artifacts {
		if artifact.Kind == string(captures.CaptureRecording) {
			return artifact.LocalPath
		}
	}
	return ""
}

func performanceArtifact(sourcePath, recordingPath string) (deliveryramp.Artifact, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return deliveryramp.Artifact{}, fmt.Errorf("read performance artifact %q: %w", sourcePath, err)
	}
	if len(data) == 0 {
		return deliveryramp.Artifact{}, fmt.Errorf("performance artifact %q is empty", sourcePath)
	}
	digest := sha256.Sum256(data)
	checksum := "sha256:" + hex.EncodeToString(digest[:])
	destination := sourcePath
	if recordingPath != "" {
		destination = recordingPath + "." + filepath.Base(filepath.Dir(sourcePath)) + "." + filepath.Base(sourcePath)
		if err := os.WriteFile(destination, data, 0o600); err != nil {
			return deliveryramp.Artifact{}, fmt.Errorf("persist performance artifact: %w", err)
		}
	}
	return deliveryramp.Artifact{
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

func gate(name deliveryramp.GateName, passed bool, reason string, startedAt, completedAt time.Time) deliveryramp.GateResult {
	disposition := deliveryramp.GateFailed
	if passed {
		disposition = deliveryramp.GatePassed
	}
	return deliveryramp.GateResult{Name: name, Disposition: disposition, Required: true, Reason: reason, StartedAt: startedAt, CompletedAt: completedAt}
}

func allRequiredGatesPassed(manifest deliveryramp.Manifest) bool {
	byName := make(map[deliveryramp.GateName]deliveryramp.GateResult, len(manifest.Gates))
	for _, item := range manifest.Gates {
		byName[item.Name] = item
	}
	for _, name := range deliveryramp.RequiredGates(manifest.Profile) {
		if byName[name].Disposition != deliveryramp.GatePassed {
			return false
		}
	}
	return true
}
