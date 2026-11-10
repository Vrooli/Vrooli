package control

import (
	"context"
	"errors"
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

// ExternalRecordingInspection is a read-only owner snapshot. A process or
// artifact observation never clears an uncertain stop outcome.
type ExternalRecordingInspection struct {
	HandleID            string    `json:"handle_id"`
	OwnerState          string    `json:"owner_state"`
	ProcessState        string    `json:"process_state"`
	ArtifactAvailable   bool      `json:"artifact_available"`
	RecoveredEvidenceID string    `json:"recovered_evidence_id,omitempty"`
	ObservedAt          time.Time `json:"observed_at"`
	Reason              string    `json:"reason,omitempty"`
}

// ReviewRecordingResult is the durable, producer-owned review copy built
// from the chapter recordings of one leased journey.
type ReviewRecordingResult struct {
	Reference evidence.Reference `json:"reference"`
	Path      string             `json:"path"`
}

// CaptureRequest uses the same evidence and lease owner as ordinary flows.
// Recording is bounded so a single request can own its complete lifecycle.
type CaptureRequest struct {
	Device       string `json:"device"`
	Actor        string `json:"actor"`
	Kind         string `json:"kind"`
	Seconds      int    `json:"seconds,omitempty"`
	Package      string `json:"package,omitempty"`
	RequireAudio bool   `json:"require_audio,omitempty"`
	// Wake brings a sleeping or locked screen up before a screenshot or
	// recording. Home also returns to the launcher; it may change the foreground
	// application, so it is opt-in.
	Wake bool `json:"wake,omitempty"`
	Home bool `json:"home,omitempty"`
}

func (s *Service) Capture(ctx context.Context, request CaptureRequest) (RunResult, error) {
	kind := strings.ToLower(strings.TrimSpace(request.Kind))
	if kind != "screenshot" && kind != "recording" && kind != "logs" && kind != "console" && kind != "network" {
		return RunResult{}, fmt.Errorf("capture kind must be screenshot, recording, logs, console, or network")
	}
	if kind == "recording" && (request.Seconds < 1 || request.Seconds > 30) {
		return RunResult{}, fmt.Errorf("recording duration must be between 1 and 30 seconds")
	}
	if request.RequireAudio && kind != "recording" {
		return RunResult{}, fmt.Errorf("audio is available only for recording")
	}
	if kind == "logs" && strings.TrimSpace(request.Package) == "" {
		return RunResult{}, fmt.Errorf("application logs require an exact package")
	}
	device, err := s.resolveExactOrUniqueDevice(ctx, request.Device)
	if err != nil {
		return RunResult{}, err
	}
	if kind == "console" || kind == "network" {
		return RunResult{Disposition: "no_evidence", Binding: RunBinding{DeviceID: device.ID}, Chapters: []Chapter{{ID: "capture", Title: kind, Disposition: "unavailable", Message: "Direct device capture is unavailable; a debuggable WebView may be inspected through the WebView attachment and browser workflow. No conclusion about events is possible."}}}, nil
	}
	var prepare []Chapter
	if (request.Wake || request.Home) && (kind == "screenshot" || kind == "recording") {
		prepare = s.prepareScreen(ctx, device, request)
	}
	profiles := device.Transports
	if len(profiles) == 0 {
		profiles = []strategy.DeviceTransport{{Name: device.Transport}}
	}
	transport := ""
	found := false
	for _, profile := range profiles {
		currentTransport, routeErr := s.requireCurrentEffectRoute(device.ID, profile.Name)
		if routeErr != nil {
			continue
		}
		candidate, ok := s.strategyForFlow(device.ID, currentTransport)
		if !ok {
			continue
		}
		var declaration strategy.Declaration
		var probeErr error
		if prober, ok := candidate.(strategy.CaptureDescriber); ok {
			declaration, probeErr = prober.DescribeCapture(ctx, kind)
		} else {
			declaration, probeErr = candidate.Describe(ctx)
		}
		if probeErr != nil || declaration.Status != strategy.StatusAvailable {
			continue
		}
		if kind == "screenshot" {
			_, supported := candidate.(strategy.Observer)
			if !supported || declaration.Capabilities[strategy.CapScreenshot].Status != strategy.StatusAvailable {
				continue
			}
		} else if kind == "recording" {
			_, supported := candidate.(strategy.SessionRecorder)
			if !supported || (declaration.Capabilities[strategy.CapNativeRecording].Status != strategy.StatusAvailable && declaration.Capabilities[strategy.CapScreenRecording].Status != strategy.StatusAvailable) {
				continue
			}
			if request.RequireAudio {
				if _, ok := candidate.(strategy.AudioSessionRecorder); !ok || declaration.Capabilities[strategy.CapRecordingAudio].Status != strategy.StatusAvailable {
					continue
				}
			}
		} else {
			_, supported := candidate.(strategy.ApplicationLogReader)
			if !supported || declaration.Capabilities[strategy.CapDeviceLogs].Status != strategy.StatusAvailable {
				continue
			}
		}
		transport, found = profile.Name, true
		break
	}
	if !found {
		message := "no available transport provides this capture"
		if request.RequireAudio {
			message = "no available transport provides recording with audio"
		}
		return RunResult{Disposition: "capability_gap", Binding: RunBinding{DeviceID: device.ID}, Chapters: append(prepare, Chapter{ID: "capture", Title: kind, Disposition: "unavailable", Message: message}), NextActions: s.captureGapNextActions(ctx, device, kind)}, nil
	}
	flow := Flow{ID: "capture-" + uuid.NewString(), Name: kind + " capture", Transport: transport, RedactSensitiveContent: kind == "logs", MaxDurationMS: 65000}
	if kind == "screenshot" {
		flow.Steps = []Step{{ID: "capture", Kind: "observe", RequiredCapabilities: []string{strategy.CapScreenshot}, TimeoutMS: 20000}}
	} else if kind == "recording" {
		startTimeoutMS, flowTimeoutMS := int64(15000), int64(65000)
		if request.RequireAudio {
			// Android's per-session projection and RECORD_AUDIO prompts are
			// operator-owned consent boundaries and may remain visible while the
			// service waits. Keep that bounded wait inside the capture request.
			startTimeoutMS, flowTimeoutMS = 65000, 100000
		}
		flow.MaxDurationMS = flowTimeoutMS
		flow.Steps = []Step{{ID: "record", Kind: "recording-start", TimeoutMS: startTimeoutMS, Arguments: map[string]any{"require_audio": request.RequireAudio}}, {ID: "duration", Kind: "wait", TimeoutMS: int64(request.Seconds+3) * 1000, Arguments: map[string]any{"settle_ms": float64(request.Seconds * 1000)}}, {ID: "stop", Kind: "recording-stop", Target: "record", TimeoutMS: 30000, Arguments: map[string]any{"minimum_duration_ms": float64(request.Seconds * 500)}}}
	} else {
		flow.Steps = []Step{{ID: "logs", Kind: "device-logs", RequiredCapabilities: []string{strategy.CapDeviceLogs}, TimeoutMS: 10000, Arguments: map[string]any{"package": strings.TrimSpace(request.Package)}}}
	}
	result, err := s.Run(ctx, flow, device.ID, firstNonEmpty(strings.TrimSpace(request.Actor), "device-capture"))
	result.Chapters = append(prepare, result.Chapters...)
	if result.Disposition != "passed" {
		result.NextActions = captureFailureNextActions(device, result.Chapters, s.unlockSetupAction(ctx, device))
	}
	return result, err
}

// prepareScreen wakes (and optionally homes) the screen through the
// same verified Screen owner a caller would use by hand. A device without a
// screen-control transport is reported as skipped rather than failing a capture
// that may still succeed.
func (s *Service) prepareScreen(ctx context.Context, device Device, request CaptureRequest) []Chapter {
	result, err := s.Screen(ctx, ScreenRequest{Device: device.ID, Actor: request.Actor, Operation: "ready", Home: request.Home})
	if err != nil {
		return []Chapter{{ID: "prepare", Title: "wake screen", Disposition: "skipped", Message: err.Error()}}
	}
	return []Chapter{{ID: "prepare", Title: "wake screen", Disposition: "passed", Message: fmt.Sprintf("screen %s via %s", result.After.ScreenState, result.Transport)}}
}

func (s *Service) StartExternalRecording(ctx context.Context, deviceID, actor, leaseToken string, requestedTransport ...string) (strategy.RecordingHandle, error) {
	session, err := s.sessionForLease(ctx, deviceID, leaseToken)
	if err != nil {
		return strategy.RecordingHandle{}, err
	}
	transport := ""
	if len(requestedTransport) > 0 {
		transport = strings.TrimSpace(requestedTransport[0])
	}
	if transport == "" {
		if record, found := s.devices.Get(deviceID); found {
			transport = record.Transport
		}
		if transport == "" {
			transport = "usb"
		}
	}
	var routeErr error
	transport, routeErr = s.requireCurrentEffectRoute(deviceID, transport)
	if routeErr != nil {
		return strategy.RecordingHandle{}, routeErr
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
	s.externalRecordings[handle.ID] = externalRecording{DeviceID: deviceID, LeaseID: session.ID, Actor: actor, Recorder: recorder, Handle: handle}
	s.mu.Unlock()
	return handle, nil
}

func (s *Service) StopExternalRecording(ctx context.Context, deviceID, actor, leaseToken, handleID string) (result ExternalRecordingResult, retErr error) {
	if _, err := s.sessionForLease(ctx, deviceID, leaseToken); err != nil {
		return ExternalRecordingResult{}, err
	}
	s.mu.Lock()
	recording, ok := s.externalRecordings[handleID]
	if !ok || recording.DeviceID != deviceID {
		s.mu.Unlock()
		return ExternalRecordingResult{}, fmt.Errorf("recording %q is not active for device %q", handleID, deviceID)
	}
	closing, admitted := recording.State.beginClose()
	if !admitted {
		s.mu.Unlock()
		if recording.Recovered != nil {
			return *recording.Recovered, fmt.Errorf("recording %q stop outcome uncertain; automatic replay is unavailable", handleID)
		}
		return ExternalRecordingResult{}, fmt.Errorf("recording %q stop is in progress or its outcome uncertain; automatic replay is unavailable", handleID)
	}
	recording.State = closing
	s.externalRecordings[handleID] = recording
	s.mu.Unlock()
	// The recorder can consume its native handle before returning an error.
	// Keep that unresolved owner state and never replay an ambiguous stop.
	defer func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		next, removed := recording.State.closeResult(retErr)
		if removed {
			delete(s.externalRecordings, handleID)
		} else {
			recording.State = next
			if result.Reference.ID != "" {
				copy := result
				recording.Recovered = &copy
			}
			s.externalRecordings[handleID] = recording
		}
	}()
	if strings.TrimSpace(actor) == "" {
		actor = recording.Actor
	}
	artifact, stopErr := recording.Recorder.StopRecording(ctx, recording.Handle)
	if stopErr != nil {
		defer func() {
			if !errors.Is(retErr, stopErr) {
				retErr = errors.Join(retErr, fmt.Errorf("recording stop outcome uncertain: %w", stopErr))
			}
		}()
	}
	if stopErr != nil && len(artifact.Bytes) == 0 {
		return ExternalRecordingResult{}, fmt.Errorf("recording stop outcome uncertain: %w", stopErr)
	}
	// A recorder may produce an artifact and separately fail restoration. Use
	// the normal evidence storage path before reporting both.
	redacted, err := evidence.CaptureWithPolicy(artifact.Bytes, "video/mp4", evidence.DefaultPolicy, false, false, actor, nil)
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
	// A successor lease may stop an abandoned handle for cleanup. Its output
	// remains attributed to the lease that started the recording.
	if err := s.persistRecording(ctx, id, redacted.Bytes, deviceID, recording.LeaseID); err != nil {
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
	result = ExternalRecordingResult{Reference: ref, Path: path, StartOffset: 0, EndOffset: end}
	if stopErr != nil {
		return result, fmt.Errorf("recording stop outcome uncertain: %w", stopErr)
	}
	return result, nil
}

// InspectExternalRecording reads one retained recording owner through its
// strategy and never replays StopRecording or changes lifecycle inventory.
func (s *Service) InspectExternalRecording(ctx context.Context, deviceID, leaseToken, handleID string) (ExternalRecordingInspection, error) {
	if _, err := s.sessionForLease(ctx, deviceID, leaseToken); err != nil {
		return ExternalRecordingInspection{}, err
	}
	s.mu.Lock()
	recording, ok := s.externalRecordings[handleID]
	if !ok || recording.DeviceID != deviceID {
		s.mu.Unlock()
		return ExternalRecordingInspection{}, fmt.Errorf("recording %q is not retained for device %q", handleID, deviceID)
	}
	inspection := ExternalRecordingInspection{HandleID: handleID, OwnerState: "active", ProcessState: "unknown", ObservedAt: time.Now().UTC()}
	switch recording.State {
	case stateUncertain:
		inspection.OwnerState = "uncertain"
	case stateClosing:
		inspection.OwnerState = "stopping"
	}
	if recording.State == stateClosing {
		inspection.Reason = "the owner stop request has not returned"
		if recording.Recovered != nil {
			inspection.RecoveredEvidenceID = recording.Recovered.Reference.ID
		}
		s.mu.Unlock()
		return inspection, nil
	}
	if recording.Recovered != nil {
		inspection.RecoveredEvidenceID = recording.Recovered.Reference.ID
	}
	inspector, supported := recording.Recorder.(strategy.RecordingInspector)
	s.mu.Unlock()
	if !supported {
		inspection.Reason = "recording strategy does not expose read-only native status"
		return inspection, nil
	}
	native, err := inspector.InspectRecording(ctx, recording.Handle)
	if err != nil {
		inspection.Reason = "recording strategy could not verify native process or artifact state"
		return inspection, nil
	}
	inspection.ProcessState = native.State
	inspection.ArtifactAvailable = native.ArtifactAvailable
	inspection.ObservedAt = native.ObservedAt
	inspection.Reason = native.Reason
	return inspection, nil
}

// FinalizeReviewRecording concatenates already-retained chapter recordings.
// It is deliberately a continuation of the same device-control recording
// mechanism: callers cannot provide arbitrary paths or raw bytes, only
// producer-issued evidence ids whose content and provenance match the lease.
func (s *Service) FinalizeReviewRecording(ctx context.Context, deviceID, actor, leaseToken string, referenceIDs []string) (ReviewRecordingResult, error) {
	session, err := s.sessionForLease(ctx, deviceID, leaseToken)
	if err != nil {
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
	audioTracks := make([]bool, 0, len(referenceIDs))
	audioCount := 0
	for index, id := range referenceIDs {
		data, kind, readErr := s.ArtifactContext(ctx, id)
		if readErr != nil {
			return ReviewRecordingResult{}, fmt.Errorf("read chapter recording %q: %w", id, readErr)
		}
		if err := s.verifyRecordingProvenance(ctx, id, data, deviceID, session.ID); err != nil {
			return ReviewRecordingResult{}, err
		}
		// Artifact kinds are currently kept in memory, so retained recordings
		// lose their kind when the service restarts. Validate those bytes as a
		// video rather than rejecting a recoverable producer-owned artifact.
		if kind == "" {
			if _, _, _, probeErr := evidence.MeasureVideo(data); probeErr != nil {
				return ReviewRecordingResult{}, fmt.Errorf("chapter evidence %q has no retained kind and is not a readable video: %w", id, probeErr)
			}
		} else if kind != "video" {
			return ReviewRecordingResult{}, fmt.Errorf("chapter evidence %q is %q, not video", id, kind)
		}
		hasAudio, audioErr := evidence.InspectAudioTrack(data)
		if audioErr != nil {
			return ReviewRecordingResult{}, fmt.Errorf("inspect audio in chapter %q: %w", id, audioErr)
		}
		audioTracks = append(audioTracks, hasAudio)
		if hasAudio {
			audioCount++
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
		if audioCount > 0 {
			if audioTracks[index] {
				filters = append(filters, fmt.Sprintf("[%d:a:0]aresample=48000:async=1:first_pts=0,aformat=sample_fmts=fltp:channel_layouts=stereo[a%d]", index, index))
			} else {
				data, readErr := os.ReadFile(input)
				if readErr != nil {
					return ReviewRecordingResult{}, fmt.Errorf("read silent chapter duration: %w", readErr)
				}
				_, duration, _, measureErr := evidence.MeasureVideo(data)
				if measureErr != nil {
					return ReviewRecordingResult{}, fmt.Errorf("measure silent chapter duration: %w", measureErr)
				}
				filters = append(filters, fmt.Sprintf("anullsrc=r=48000:cl=stereo,atrim=duration=%.6f,asetpts=PTS-STARTPTS[a%d]", duration.Seconds(), index))
			}
		}
	}
	labels := make([]string, 0, len(inputs))
	for index := range inputs {
		labels = append(labels, fmt.Sprintf("[v%d]", index))
		if audioCount > 0 {
			labels = append(labels, fmt.Sprintf("[a%d]", index))
		}
	}
	audioOutputs := 0
	if audioCount > 0 {
		audioOutputs = 1
	}
	filter := strings.Join(filters, ";") + ";" + strings.Join(labels, "") + fmt.Sprintf("concat=n=%d:v=1:a=%d[outv]", len(inputs), audioOutputs)
	if audioCount > 0 {
		filter += "[outa]"
	}
	args = append(args, "-filter_complex", filter, "-map", "[outv]")
	if audioCount > 0 {
		args = append(args, "-map", "[outa]", "-c:a", "aac")
	}
	args = append(args, "-c:v", "libx264", "-pix_fmt", "yuv420p", "-movflags", "+faststart", outputPath)
	if combined, runErr := exec.CommandContext(ctx, "ffmpeg", args...).CombinedOutput(); runErr != nil {
		return ReviewRecordingResult{}, fmt.Errorf("concatenate chapter recordings: %w (%s)", runErr, strings.TrimSpace(string(combined)))
	}
	combined, err := os.ReadFile(outputPath)
	if err != nil {
		return ReviewRecordingResult{}, fmt.Errorf("read review recording: %w", err)
	}
	redacted, err := evidence.CaptureWithPolicy(combined, "video/mp4", evidence.DefaultPolicy, false, false, actor, nil)
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
	if audioCount > 0 && (ref.AudioTrackPresent == nil || !*ref.AudioTrackPresent) {
		return ReviewRecordingResult{}, fmt.Errorf("review recording audio track is absent or could not be verified")
	}
	if audioCount == 0 {
		ref.AudioTrackCoverage = "no_chapters"
	} else if audioCount == len(inputs) {
		ref.AudioTrackCoverage = "all_chapters"
	} else {
		ref.AudioTrackCoverage = "some_chapters"
	}
	if err := s.persistRecording(ctx, id, redacted.Bytes, deviceID, session.ID); err != nil {
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
