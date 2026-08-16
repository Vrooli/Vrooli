package control

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"device-control/internal/evidence"
	"device-control/strategy"
	"github.com/google/uuid"
)

// ExternalRecordingResult is the reference-only result returned to a ramp
// that interleaves native device verbs with a BAS WebView flow. The capture
// bytes remain in device-control's producer-owned evidence store.
type ExternalRecordingResult struct {
	Reference   evidence.Reference `json:"reference"`
	Path        string             `json:"path"`
	StartOffset int64              `json:"start_offset_ms"`
	EndOffset   int64              `json:"end_offset_ms"`
}

// ReviewRecordingResult is the durable, producer-owned review copy built
// from the chapter recordings of one leased journey.
type ReviewRecordingResult struct {
	Reference evidence.Reference `json:"reference"`
	Path      string             `json:"path"`
}

func (s *Service) StartExternalRecording(ctx context.Context, deviceID, actor, leaseToken string) (strategy.RecordingHandle, error) {
	if _, err := s.sessionForLease(ctx, deviceID, leaseToken); err != nil {
		return strategy.RecordingHandle{}, err
	}
	transport := "usb"
	if record, found := s.devices.Get(deviceID); found && record.Transport != "" {
		transport = record.Transport
	}
	strat, ok := s.strategyForFlow(deviceID, transport)
	if !ok {
		return strategy.RecordingHandle{}, fmt.Errorf("unknown or unavailable device %q", deviceID)
	}
	recorder, ok := strat.(strategy.SessionRecorder)
	if !ok {
		return strategy.RecordingHandle{}, &strategy.AvailabilityError{Reason: "device strategy does not expose screen recording", NextAction: "Use a device strategy that declares screen recording."}
	}
	handle, err := recorder.StartRecording(ctx, strategy.ClaimTransition)
	if err != nil {
		return strategy.RecordingHandle{}, err
	}
	s.mu.Lock()
	s.externalRecordings[handle.ID] = externalRecording{DeviceID: deviceID, Actor: actor, Recorder: recorder, Handle: handle}
	s.mu.Unlock()
	return handle, nil
}

func (s *Service) StopExternalRecording(ctx context.Context, deviceID, actor, leaseToken, handleID string) (ExternalRecordingResult, error) {
	if _, err := s.sessionForLease(ctx, deviceID, leaseToken); err != nil {
		return ExternalRecordingResult{}, err
	}
	s.mu.Lock()
	recording, ok := s.externalRecordings[handleID]
	if ok {
		delete(s.externalRecordings, handleID)
	}
	s.mu.Unlock()
	if !ok || recording.DeviceID != deviceID {
		return ExternalRecordingResult{}, fmt.Errorf("recording %q is not active for device %q", handleID, deviceID)
	}
	if strings.TrimSpace(actor) == "" {
		actor = recording.Actor
	}
	artifact, err := recording.Recorder.StopRecording(ctx, recording.Handle)
	if err != nil {
		return ExternalRecordingResult{}, err
	}
	redacted, err := evidence.RedactCapture(artifact.Bytes, "video/mp4", evidence.DefaultPolicy, false, actor)
	if err != nil {
		return ExternalRecordingResult{}, fmt.Errorf("redact external recording: %w", err)
	}
	_, duration, fps, err := evidence.MeasureVideo(redacted.Bytes)
	if err != nil {
		return ExternalRecordingResult{}, fmt.Errorf("measure external recording: %w", err)
	}
	class := evidence.ClaimClass(artifact.ClaimClass)
	if class == "" {
		class = evidence.ClaimTransition
	}
	id := uuid.NewString()
	ref, err := evidence.NewClaimedVideoReference(id, redacted.Bytes, redacted, artifact.Method, fps, class)
	if err != nil {
		return ExternalRecordingResult{}, fmt.Errorf("validate external recording: %w", err)
	}
	if err := s.persistArtifact(ctx, id, redacted.Bytes, "video"); err != nil {
		return ExternalRecordingResult{}, err
	}
	end := duration.Milliseconds()
	if end <= 0 {
		end = time.Since(recording.Handle.StartedAt).Milliseconds()
	}
	if end < 0 {
		end = 0
	}
	path, err := s.artifactPath(ctx, id)
	if err != nil {
		return ExternalRecordingResult{}, fmt.Errorf("resolve external recording path: %w", err)
	}
	if !filepath.IsAbs(path) {
		return ExternalRecordingResult{}, fmt.Errorf("external recording path is not absolute")
	}
	return ExternalRecordingResult{Reference: ref, Path: path, StartOffset: 0, EndOffset: end}, nil
}

// FinalizeReviewRecording concatenates already-retained chapter recordings.
// It is deliberately a continuation of the same device-control recording
// mechanism: callers cannot provide arbitrary paths or raw bytes, only
// producer-issued evidence ids bound to their active device lease.
func (s *Service) FinalizeReviewRecording(ctx context.Context, deviceID, actor, leaseToken string, referenceIDs []string) (ReviewRecordingResult, error) {
	if _, err := s.sessionForLease(ctx, deviceID, leaseToken); err != nil {
		return ReviewRecordingResult{}, err
	}
	if len(referenceIDs) == 0 {
		return ReviewRecordingResult{}, fmt.Errorf("review recording requires at least one chapter reference")
	}
	workDir, err := os.MkdirTemp("", "device-control-review-")
	if err != nil {
		return ReviewRecordingResult{}, fmt.Errorf("create review recording workspace: %w", err)
	}
	defer os.RemoveAll(workDir)
	inputs := make([]string, 0, len(referenceIDs))
	for index, id := range referenceIDs {
		data, kind, readErr := s.ArtifactContext(ctx, id)
		if readErr != nil {
			return ReviewRecordingResult{}, fmt.Errorf("read chapter recording %q: %w", id, readErr)
		}
		if kind != "video" {
			return ReviewRecordingResult{}, fmt.Errorf("chapter evidence %q is %q, not video", id, kind)
		}
		path := filepath.Join(workDir, fmt.Sprintf("chapter-%04d.mp4", index))
		if writeErr := os.WriteFile(path, data, 0o600); writeErr != nil {
			return ReviewRecordingResult{}, fmt.Errorf("write chapter recording %q: %w", id, writeErr)
		}
		inputs = append(inputs, path)
	}
	outputPath := filepath.Join(workDir, "review.mp4")
	args := []string{"-y", "-loglevel", "error"}
	filters := make([]string, 0, len(inputs))
	for index, input := range inputs {
		args = append(args, "-i", input)
		filters = append(filters, fmt.Sprintf("[%d:v]scale=720:1280:force_original_aspect_ratio=decrease,pad=720:1280:(ow-iw)/2:(oh-ih)/2,setsar=1[v%d]", index, index))
	}
	labels := make([]string, 0, len(inputs))
	for index := range inputs {
		labels = append(labels, fmt.Sprintf("[v%d]", index))
	}
	filter := strings.Join(filters, ";") + ";" + strings.Join(labels, "") + fmt.Sprintf("concat=n=%d:v=1:a=0[outv]", len(inputs))
	args = append(args, "-filter_complex", filter, "-map", "[outv]", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-movflags", "+faststart", outputPath)
	if combined, runErr := exec.CommandContext(ctx, "ffmpeg", args...).CombinedOutput(); runErr != nil {
		return ReviewRecordingResult{}, fmt.Errorf("concatenate chapter recordings: %w (%s)", runErr, strings.TrimSpace(string(combined)))
	}
	combined, err := os.ReadFile(outputPath)
	if err != nil {
		return ReviewRecordingResult{}, fmt.Errorf("read review recording: %w", err)
	}
	redacted, err := evidence.RedactCapture(combined, "video/mp4", evidence.DefaultPolicy, false, actor)
	if err != nil {
		return ReviewRecordingResult{}, fmt.Errorf("redact review recording: %w", err)
	}
	_, _, fps, err := evidence.MeasureVideo(redacted.Bytes)
	if err != nil {
		return ReviewRecordingResult{}, fmt.Errorf("measure review recording: %w", err)
	}
	id := uuid.NewString()
	ref, err := evidence.NewClaimedVideoReference(id, redacted.Bytes, redacted, "chapter-concat", fps, evidence.ClaimTransition)
	if err != nil {
		return ReviewRecordingResult{}, fmt.Errorf("validate review recording: %w", err)
	}
	if err := s.persistArtifact(ctx, id, redacted.Bytes, "video"); err != nil {
		return ReviewRecordingResult{}, err
	}
	path, err := s.artifactPath(ctx, id)
	if err != nil {
		return ReviewRecordingResult{}, fmt.Errorf("resolve review recording path: %w", err)
	}
	if !filepath.IsAbs(path) {
		return ReviewRecordingResult{}, fmt.Errorf("review recording path is not absolute")
	}
	return ReviewRecordingResult{Reference: ref, Path: path}, nil
}
