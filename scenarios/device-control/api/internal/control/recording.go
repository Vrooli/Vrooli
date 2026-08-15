package control

import (
	"context"
	"fmt"
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
	StartOffset int64              `json:"start_offset_ms"`
	EndOffset   int64              `json:"end_offset_ms"`
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
	return ExternalRecordingResult{Reference: ref, StartOffset: 0, EndOffset: end}, nil
}
