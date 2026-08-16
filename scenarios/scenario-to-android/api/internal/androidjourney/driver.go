// Package androidjourney adapts scenario-to-android's device-control and BAS
// clients to the delivery-ramp Driver seam. Device verbs are deliberately
// interfaces: the production implementation is the device-control client,
// while tests can prove lease and evidence invariants without adb.
package androidjourney

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

type Lease struct {
	ID        string
	DeviceID  string
	Token     string
	ExpiresAt time.Time
}

type ActionResult struct {
	Observed string
	Evidence []deliveryramp.EvidenceReference
}

type RecordingArtifact struct {
	Reference  deliveryramp.EvidenceReference
	Path       string
	StartMs    int64
	EndMs      int64
	HasOffsets bool
}

// ChapterRecorder is optional so older/native-only test doubles can retain
// the run-level recording contract while production device-control clients
// provide one bounded recording for each conformance chapter.
type ChapterRecorder interface {
	StartChapterRecording(context.Context, Lease, string) error
	StopChapterRecording(context.Context, Lease, string) (RecordingArtifact, error)
}

// ReviewRecorder finalizes the producer-owned chapter recordings into one
// review artifact. It is optional so the journey remains testable with small
// device fakes, while the production device-control client owns the bytes.
type ReviewRecorder interface {
	FinalizeReviewRecording(context.Context, Lease, []deliveryramp.EvidenceReference) (ReviewRecording, error)
}

type ReviewRecording struct {
	Reference deliveryramp.EvidenceReference
	Path      string
}

type LogCapture interface {
	StartLogCapture(context.Context, Lease) error
	StopLogCapture(context.Context, Lease) (LogCaptureArtifact, error)
}

// ClockSampler is optional for compatibility with older test doubles. The
// production HTTP client implements it so journey evidence contains explicit
// start/end host-to-device clock calibration samples.
type ClockSampler interface {
	SampleClock(context.Context, Lease) (ClockSample, error)
}

type DeviceClient interface {
	Acquire(context.Context, string, string, time.Duration) (Lease, error)
	ValidateLease(context.Context, Lease) error
	Execute(context.Context, Lease, string, map[string]string) (ActionResult, error)
	StartRecording(context.Context, Lease) error
	StopRecording(context.Context, Lease) (RecordingArtifact, error)
	Release(context.Context, Lease) error
}

// Authenticator lets a journey use a device-control-owned credential profile
// for its unlock step. The credential itself never crosses this boundary.
type Authenticator interface {
	Unlock(context.Context, Lease, string) error
}

// WebViewAttacher is optional on DeviceClient so existing native-only test
// doubles remain small. The production device-control client implements it;
// the driver then supplies BAS with a lease-scoped forwarded CDP endpoint.
type WebViewAttacher interface {
	AttachWebView(context.Context, Lease, string) (WebViewAttachment, error)
}

type WebViewAttachment struct {
	CDPEndpoint string
	RendererID  string
	RendererURL string
}

type BASRequest struct {
	TargetID         string
	Scenario         string
	Artifact         deliveryramp.Artifact
	StepID           string
	RunID            string
	Arguments        map[string]string
	CDPEndpoint      string
	RendererID       string
	RendererURL      string
	FlowPath         string
	IsolationLeaseID string
}

type BASResult struct {
	Completed bool
	Evidence  []deliveryramp.EvidenceReference
}

type BASClient interface {
	Execute(context.Context, BASRequest) (BASResult, error)
}

type Driver struct {
	Devices  DeviceClient
	BAS      BASClient
	WebView  WebViewAttacher
	Actor    string
	LeaseTTL time.Duration
}

var _ deliveryramp.Driver = Driver{}

func (d Driver) Execute(ctx context.Context, request deliveryramp.DriverRequest) (result deliveryramp.JourneyResult, err error) {
	runStarted := time.Now()
	result = deliveryramp.JourneyResult{
		SchemaVersion: deliveryramp.JourneySchemaVersion, EvidenceVersion: deliveryramp.JourneyEvidenceVersion,
		SmokeTestID: request.RunID, PlanID: request.Plan.ID, Profile: request.Plan.Profile,
		Capability: request.Plan.Capability, TargetID: request.Cell.Target.ID, CellID: request.Cell.ID,
		ScenarioName: artifactScenario(request.Artifact, request.Cell.Target.Label), Platform: request.Cell.Target.Platform,
		Disposition: deliveryramp.DispositionNotRun, Steps: make([]deliveryramp.JourneyStep, 0, len(request.Plan.Steps)),
		CreatedAt: runStarted.UTC(),
	}
	if d.Devices == nil {
		return result, fmt.Errorf("device-control client is unavailable")
	}
	if !request.Cell.Target.Available {
		return result, fmt.Errorf("Android target %q is unavailable: %s", request.Cell.Target.ID, request.Cell.Target.Reason)
	}
	if strings.TrimSpace(request.Artifact.ImmutableRef) == "" {
		return result, fmt.Errorf("Android artifact immutable identity is required")
	}
	ttl := d.LeaseTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	lease, err := d.Devices.Acquire(ctx, request.Cell.Target.ID, d.Actor, ttl)
	if err != nil {
		return result, fmt.Errorf("acquire device-control lease: %w", err)
	}
	defer func() { _ = d.Devices.Release(context.Background(), lease) }()
	clockSampler, hasClockSampler := d.Devices.(ClockSampler)
	if hasClockSampler {
		sample, sampleErr := clockSampler.SampleClock(ctx, lease)
		if sampleErr != nil {
			return result, fmt.Errorf("sample device clock at journey start: %w", sampleErr)
		}
		result.ClockOffsetStart = clockOffsetSample(sample)
		// Register this defer before log capture teardown so the logcat stop
		// runs first and the end calibration is the final device observation.
		defer func() {
			end, endErr := clockSampler.SampleClock(context.Background(), lease)
			if endErr != nil {
				if err == nil {
					err = fmt.Errorf("sample device clock at journey end: %w", endErr)
				} else {
					err = errors.Join(err, fmt.Errorf("sample device clock at journey end: %w", endErr))
				}
				result.Disposition = deliveryramp.DispositionFailed
				return
			}
			result.ClockOffsetEnd = clockOffsetSample(end)
		}()
	}
	logCapture, hasLogCapture := d.Devices.(LogCapture)
	if hasLogCapture {
		if err := logCapture.StartLogCapture(ctx, lease); err != nil {
			return result, fmt.Errorf("start device logcat capture: %w", err)
		}
		defer func() {
			capture, stopErr := logCapture.StopLogCapture(context.Background(), lease)
			if stopErr == nil {
				if capture.Reference.ID != "" && len(result.Steps) > 0 {
					// Logcat is run-scoped evidence. Attach the producer-owned
					// artifact to the first step so the durable matrix retains a
					// verifiable reference without inventing a separate timeline
					// file.
					result.Steps[0].Evidence = append(result.Steps[0].Evidence, capture.Reference)
				}
				result.Events = append(result.Events, parseLogcatEvents(capture.Lines)...)
				result.Steps = correlateLogcatEvents(result.Steps, result.Events, result.ClockOffsetStart)
				if assertionFailures := assertionFailureCount(result.Steps); assertionFailures > 0 {
					result.Disposition = deliveryramp.DispositionFailed
					assertionReason := fmt.Sprintf("%d journey assertions were not observed in device evidence: %s", assertionFailures, failedAssertionSummary(result.Steps))
					if strings.TrimSpace(result.DegradedReason) == "" {
						result.DegradedReason = assertionReason
					} else {
						result.DegradedReason += "; " + assertionReason
					}
				}
			}
		}()
	}
	chapterRecorder, perChapter := d.Devices.(ChapterRecorder)
	reviewRecorder, hasReviewRecorder := d.Devices.(ReviewRecorder)
	chapterReferences := make([]deliveryramp.EvidenceReference, 0)
	videoUnavailable := make(map[string]string)
	activeChapter := ""
	chapterStartStep := -1
	chapterActive := false
	stopChapter := func(stopCtx context.Context) error {
		if !chapterActive {
			return nil
		}
		recording, stopErr := chapterRecorder.StopChapterRecording(stopCtx, lease, activeChapter)
		chapterActive = false
		if stopErr != nil {
			videoUnavailable[activeChapter] = fmt.Sprintf("stop recording for chapter %q: %s", activeChapter, stopErr)
			for index := chapterStartStep; index < len(result.Steps); index++ {
				if result.Steps[index].ChapterID == activeChapter || index >= chapterStartStep {
					result.Steps[index].VideoDisposition = deliveryramp.StepUnavailable
					result.Steps[index].VideoError = videoUnavailable[activeChapter]
				}
			}
			return nil
		}
		if !recording.HasOffsets || recording.StartMs < 0 || recording.EndMs < recording.StartMs {
			videoUnavailable[activeChapter] = fmt.Sprintf("chapter %q recording is missing bounded start/end offsets", activeChapter)
			for index := chapterStartStep; index < len(result.Steps); index++ {
				result.Steps[index].VideoDisposition = deliveryramp.StepUnavailable
				result.Steps[index].VideoError = videoUnavailable[activeChapter]
			}
			return nil
		}
		if recording.Reference.ID == "" || recording.Reference.Checksum == "" || !recording.Reference.Redacted {
			videoUnavailable[activeChapter] = fmt.Sprintf("chapter %q recording reference is incomplete or not redacted", activeChapter)
			for index := chapterStartStep; index < len(result.Steps); index++ {
				result.Steps[index].VideoDisposition = deliveryramp.StepUnavailable
				result.Steps[index].VideoError = videoUnavailable[activeChapter]
			}
			return nil
		}
		chapterReferences = append(chapterReferences, recording.Reference)
		if chapterStartStep >= 0 && chapterStartStep < len(result.Steps) {
			result.Steps[chapterStartStep].ChapterID = activeChapter
			result.Steps[chapterStartStep].Evidence = append(result.Steps[chapterStartStep].Evidence, recording.Reference)
			start, end := recording.StartMs, recording.EndMs
			result.Steps[chapterStartStep].VideoStartOffsetMs = &start
			result.Steps[len(result.Steps)-1].VideoEndOffsetMs = &end
			for index := chapterStartStep; index < len(result.Steps); index++ {
				result.Steps[index].ChapterID = activeChapter
			}
		}
		return nil
	}
	if !perChapter {
		if err := d.Devices.StartRecording(ctx, lease); err != nil {
			return result, fmt.Errorf("start device recording before launch: %w", err)
		}
	}
	defer func() {
		if chapterActive {
			_ = stopChapter(context.Background())
		}
	}()
	unsupportedChapters := make(map[string]string)
	for index, spec := range request.Plan.Steps {
		if err := d.Devices.ValidateLease(ctx, lease); err != nil {
			return result, fmt.Errorf("device-control lease lost before step %q: %w", spec.ID, err)
		}
		step := deliveryramp.JourneyStep{ID: spec.ID, Name: spec.ID, Purpose: spec.Purpose, Action: spec.Action, Disposition: deliveryramp.StepPassed, Readiness: spec.Readiness, Settle: spec.Settle, StartedAt: time.Now().UTC(), MonotonicStartMs: time.Since(runStarted).Milliseconds()}
		if spec.Assertion != nil {
			step.AssertionID = spec.Assertion.ID
			step.ExpectedState = spec.Assertion.Expected
			step.AssertionStatus = "pending"
		}
		chapterID := spec.Arguments["chapter_id"]
		if perChapter && chapterActive && chapterID != activeChapter {
			if stopErr := stopChapter(ctx); stopErr != nil {
				return result, stopErr
			}
		}
		if reason, unsupported := unsupportedChapters[chapterID]; unsupported {
			step.Disposition = deliveryramp.StepUnavailable
			step.Error = reason
			step.DegradedReason = reason
			step.CompletedAt = time.Now().UTC()
			result.Steps = append(result.Steps, step)
			if result.Disposition == deliveryramp.DispositionNotRun {
				result.Disposition = deliveryramp.DispositionUnavailable
			}
			continue
		}
		if missing := missingCapability(spec.Arguments["required_capabilities"], request.Cell.Target.Capabilities); missing != "" {
			reason := fmt.Sprintf("chapter %q unavailable: target lacks required capability %q", chapterID, missing)
			unsupportedChapters[chapterID] = reason
			step.Disposition = deliveryramp.StepUnavailable
			step.Error = reason
			step.DegradedReason = reason
			step.CompletedAt = time.Now().UTC()
			result.Steps = append(result.Steps, step)
			if result.Disposition == deliveryramp.DispositionNotRun {
				result.Disposition = deliveryramp.DispositionUnavailable
			}
			continue
		}
		if perChapter && !chapterActive && videoUnavailable[chapterID] == "" {
			activeChapter = chapterID
			chapterStartStep = len(result.Steps)
			if err := chapterRecorder.StartChapterRecording(ctx, lease, activeChapter); err != nil {
				videoUnavailable[activeChapter] = fmt.Sprintf("start recording for chapter %q: %s", activeChapter, err)
				activeChapter = ""
				chapterStartStep = -1
			} else {
				chapterActive = true
			}
		}
		if strings.EqualFold(spec.Action, "bas-flow") {
			if d.BAS == nil {
				return result, fmt.Errorf("BAS client is unavailable for step %q", spec.ID)
			}
			basRequest := BASRequest{TargetID: request.Cell.Target.ID, Scenario: artifactScenario(request.Artifact, request.Cell.Target.Label), Artifact: request.Artifact, StepID: spec.ID, RunID: request.RunID, Arguments: spec.Arguments, FlowPath: strings.TrimSpace(spec.Arguments["flow_path"]), IsolationLeaseID: lease.ID}
			attacher := d.WebView
			if attacher == nil {
				attacher, _ = d.Devices.(WebViewAttacher)
			}
			packageName := ""
			if attacher != nil {
				if request.Artifact.Metadata != nil {
					packageName = request.Artifact.Metadata["package_name"]
				}
				if strings.TrimSpace(packageName) == "" {
					return result, fmt.Errorf("Android BAS flow %q requires an artifact package identity", spec.ID)
				}
			}
			var basResult BASResult
			var basErr error
			for attempt := 0; attempt < 2; attempt++ {
				if attacher != nil {
					endpoint, attachErr := attacher.AttachWebView(ctx, lease, packageName)
					if attachErr != nil {
						return result, fmt.Errorf("attach Android WebView for flow %q: %w", spec.ID, attachErr)
					}
					basRequest.CDPEndpoint = endpoint.CDPEndpoint
					basRequest.RendererID = endpoint.RendererID
					basRequest.RendererURL = endpoint.RendererURL
				}
				basResult, basErr = d.BAS.Execute(ctx, basRequest)
				if basErr == nil || !isTransientWebViewClosure(basErr) || attempt == 1 {
					break
				}
				if err := waitForWebViewRetry(ctx); err != nil {
					return result, fmt.Errorf("BAS flow %q retry wait: %w", spec.ID, err)
				}
			}
			if basErr != nil {
				return result, fmt.Errorf("BAS flow %q failed: %w", spec.ID, basErr)
			}
			if !basResult.Completed {
				return result, fmt.Errorf("BAS flow %q did not complete", spec.ID)
			}
			step.Evidence = append(step.Evidence, basResult.Evidence...)
			// BAS completion is a producer-owned observation of the WebView
			// state, backed by the flow's evidence. Preserve it as an event so
			// the chapter assertion can be correlated even when Android does
			// not emit a second ActivityTaskManager/Capacitor line after an
			// already-running WebView is attached.
			result.Events = append(result.Events, deliveryramp.JourneyEvent{
				Type:             "bas-observation",
				StepID:           step.ID,
				Observed:         "BAS flow completed",
				StartedAt:        step.StartedAt,
				CompletedAt:      time.Now().UTC(),
				MonotonicStartMs: step.MonotonicStartMs,
				MonotonicEndMs:   time.Since(runStarted).Milliseconds(),
				Source:           "browser-automation-studio",
			})
		} else {
			arguments := map[string]string{"step_id": spec.ID, "target": spec.Arguments["target"], "reference": spec.Arguments["reference"]}
			if timeoutMS := strings.TrimSpace(spec.Arguments["timeout_ms"]); timeoutMS != "" {
				arguments["timeout_ms"] = timeoutMS
			}
			if target := strings.TrimSpace(spec.Arguments["target"]); target != "" {
				arguments["value"] = target
			}
			if strings.EqualFold(spec.Action, "package-state") {
				// device-control distinguishes the package identity from the
				// expected installation state. The conformance step encodes the
				// latter in target, so preserve it under the protocol's expected
				// argument instead of silently sending an empty value.
				arguments["expected"] = spec.Arguments["target"]
			}
			if strings.EqualFold(spec.Action, "grant-permission") || strings.EqualFold(spec.Action, "revoke-permission") {
				arguments["permission"] = spec.Arguments["target"]
			}
			if request.Artifact.Metadata != nil {
				arguments["package"] = request.Artifact.Metadata["package_name"]
				if strings.EqualFold(spec.Action, "install") {
					arguments["value"] = request.Artifact.Metadata["apk_path"]
				}
			}
			var actionResult ActionResult
			var actionErr error
			if strings.EqualFold(spec.Action, "screen") && strings.EqualFold(spec.Arguments["target"], "unlock") {
				profileID := ""
				if request.Artifact.Metadata != nil {
					profileID = strings.TrimSpace(request.Artifact.Metadata["auth_profile_id"])
				}
				if profileID != "" {
					authenticator, ok := d.Devices.(Authenticator)
					if !ok {
						actionErr = fmt.Errorf("device-control client cannot use Android auth profile %q", profileID)
					} else {
						actionErr = authenticator.Unlock(ctx, lease, profileID)
						actionResult = ActionResult{Observed: "unlocked"}
					}
				}
			}
			if actionErr == nil && actionResult.Observed == "" {
				actionResult, actionErr = d.Devices.Execute(ctx, lease, spec.Action, arguments)
			}
			if actionErr != nil {
				step.Disposition = deliveryramp.StepFailed
				step.Error = fmt.Sprintf("device-control step %q failed: %s", spec.ID, actionErr)
				result.Disposition = deliveryramp.DispositionFailed
				result.DegradedReason = step.Error
			}
			if actionErr == nil {
				step.Evidence = append(step.Evidence, actionResult.Evidence...)
				result.Events = append(result.Events, deliveryramp.JourneyEvent{Type: "device-action", StepID: step.ID, Observed: spec.Action, StartedAt: step.StartedAt, CompletedAt: time.Now().UTC(), MonotonicStartMs: step.MonotonicStartMs, MonotonicEndMs: time.Since(runStarted).Milliseconds(), Source: "device-control"})
			}
		}
		step.CompletedAt = time.Now().UTC()
		step.MonotonicEndMs = time.Since(runStarted).Milliseconds()
		if reason := videoUnavailable[chapterID]; reason != "" {
			step.VideoDisposition = deliveryramp.StepUnavailable
			step.VideoError = reason
		}
		result.Steps = append(result.Steps, step)
		if perChapter && (index == len(request.Plan.Steps)-1 || request.Plan.Steps[index+1].Arguments["chapter_id"] != activeChapter) {
			if stopErr := stopChapter(ctx); stopErr != nil {
				return result, stopErr
			}
		}
		if step.Disposition == deliveryramp.StepFailed {
			break
		}
	}
	if perChapter {
		if hasReviewRecorder && len(chapterReferences) > 0 {
			review, reviewErr := reviewRecorder.FinalizeReviewRecording(ctx, lease, chapterReferences)
			if reviewErr != nil {
				result.DegradedReason = fmt.Sprintf("review recording unavailable: %s", reviewErr)
				result.Disposition = deliveryramp.DispositionDegraded
			} else if review.Reference.ID == "" || review.Reference.Checksum == "" || !review.Reference.Redacted {
				result.DegradedReason = "review recording reference is incomplete or not redacted"
				result.Disposition = deliveryramp.DispositionDegraded
			} else if !filepath.IsAbs(review.Path) {
				result.DegradedReason = "review recording path is missing or not absolute"
				result.Disposition = deliveryramp.DispositionDegraded
			} else {
				result.ReviewRecording = &review.Reference
				result.ReviewRecordingPath = review.Path
			}
		}
		if result.Disposition == deliveryramp.DispositionNotRun {
			result.Disposition = deliveryramp.DispositionPass
		}
		result.CompletedAt = time.Now().UTC()
		return result, nil
	}
	recording, err := d.Devices.StopRecording(ctx, lease)
	if err != nil {
		return result, fmt.Errorf("stop device recording: %w", err)
	}
	if !recording.HasOffsets || recording.StartMs < 0 || recording.EndMs < recording.StartMs {
		return result, fmt.Errorf("recording is missing bounded start/end offsets")
	}
	if recording.Reference.ID == "" || recording.Reference.Checksum == "" || !recording.Reference.Redacted {
		return result, fmt.Errorf("recording reference is incomplete or not redacted")
	}
	if len(result.Steps) > 0 {
		result.Steps[len(result.Steps)-1].Evidence = append(result.Steps[len(result.Steps)-1].Evidence, recording.Reference)
	}
	if len(result.Steps) > 0 {
		start, end := recording.StartMs, recording.EndMs
		result.Steps[0].VideoStartOffsetMs = &start
		result.Steps[len(result.Steps)-1].VideoEndOffsetMs = &end
	}
	if result.Disposition == deliveryramp.DispositionNotRun {
		result.Disposition = deliveryramp.DispositionPass
	}
	result.CompletedAt = time.Now().UTC()
	return result, nil
}

func isTransientWebViewClosure(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "target page, context or browser has been closed") || strings.Contains(message, "target page or context has been closed")
}

func waitForWebViewRetry(ctx context.Context) error {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func missingCapability(declared string, available []string) string {
	for _, required := range strings.Split(declared, ",") {
		required = strings.ToLower(strings.TrimSpace(required))
		if required == "" {
			continue
		}
		if capabilityPresent(required, available) {
			continue
		}
		return required
	}
	return ""
}

func capabilityPresent(required string, available []string) bool {
	for _, raw := range available {
		value := strings.ToLower(strings.TrimSpace(raw))
		if value == required || strings.ReplaceAll(value, "_", "-") == required {
			return true
		}
		if required == "network-control" && strings.Contains(value, "network_control") {
			return true
		}
		if required != "network-control" && required != "webview-attach" && (value == "device-control" || strings.Contains(value, "native_window")) {
			return true
		}
		if required == "webview-attach" && (value == "android-webview" || strings.Contains(value, "electron_cdp")) {
			return true
		}
	}
	return false
}

func artifactScenario(artifact deliveryramp.Artifact, fallback string) string {
	if artifact.Metadata != nil && strings.TrimSpace(artifact.Metadata["scenario_name"]) != "" {
		return artifact.Metadata["scenario_name"]
	}
	return fallback
}

func parseLogcatEvents(lines []string) []deliveryramp.JourneyEvent {
	events := make([]deliveryramp.JourneyEvent, 0)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		fields := strings.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}
		timestamp, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			continue
		}
		lower := strings.ToLower(trimmed)
		typeName := ""
		switch {
		case strings.Contains(lower, "activitytaskmanager:") && strings.Contains(lower, " start "):
			typeName = "activity-start"
		case strings.Contains(lower, "activitytaskmanager:") && strings.Contains(lower, " displayed "):
			typeName = "first-frame"
		case strings.Contains(lower, "capacitor") && (strings.Contains(lower, "load") || strings.Contains(lower, "ready")):
			typeName = "webview-ready"
		}
		if typeName == "" {
			continue
		}
		seconds := int64(timestamp)
		nanos := int64((timestamp - float64(seconds)) * 1e9)
		observedAt := time.Unix(seconds, nanos).UTC()
		events = append(events, deliveryramp.JourneyEvent{Type: typeName, Observed: trimmed, StartedAt: observedAt, CompletedAt: observedAt, DeviceTimestamp: timestamp, Source: "logcat", Raw: trimmed})
	}
	return events
}

func correlateLogcatEvents(steps []deliveryramp.JourneyStep, events []deliveryramp.JourneyEvent, samples ...*deliveryramp.ClockOffsetSample) []deliveryramp.JourneyStep {
	var offsetMs int64
	if len(samples) > 0 && samples[0] != nil {
		offsetMs = samples[0].OffsetMs
	}
	for eventIndex := range events {
		for stepIndex := range steps {
			step := &steps[stepIndex]
			event := events[eventIndex]
			hostEventTime := event.StartedAt.Add(time.Duration(offsetMs) * time.Millisecond)
			if event.Type == "device-action" || event.Type == "bas-observation" {
				hostEventTime = event.StartedAt
			}
			if hostEventTime.Before(step.StartedAt.Add(-2*time.Second)) || hostEventTime.After(step.CompletedAt.Add(2*time.Second)) {
				continue
			}
			if !eventMatchesStep(step, event) {
				continue
			}
			events[eventIndex].StepID = step.ID
			events[eventIndex].MonotonicStartMs = step.MonotonicStartMs
			events[eventIndex].MonotonicEndMs = step.MonotonicEndMs
			if step.AssertionStatus == "" || step.AssertionStatus == "pending" {
				step.AssertionStatus = "observed"
			}
			break
		}
	}
	for index := range steps {
		step := &steps[index]
		if step.AssertionID == "" || step.Disposition != deliveryramp.StepPassed || step.AssertionStatus == "observed" {
			continue
		}
		step.AssertionStatus = "failed"
		step.Error = fmt.Sprintf("expected device evidence %q was not observed", step.ExpectedState)
		step.Disposition = deliveryramp.StepFailed
	}
	return steps
}

func eventMatchesStep(step *deliveryramp.JourneyStep, event deliveryramp.JourneyEvent) bool {
	if event.Type == "device-action" {
		switch strings.ToLower(step.Action) {
		case "launch", "observe", "bas-flow":
			return false
		}
		return event.StepID == step.ID
	}
	switch strings.ToLower(step.Action) {
	case "launch":
		return event.Type == "activity-start" || event.Type == "first-frame"
	case "observe", "bas-flow":
		return event.Type == "first-frame" || event.Type == "webview-ready" || (strings.EqualFold(step.Action, "bas-flow") && event.Type == "bas-observation")
	default:
		return event.Type == "device-action"
	}
}

func assertionFailureCount(steps []deliveryramp.JourneyStep) int {
	count := 0
	for _, step := range steps {
		if step.AssertionStatus == "failed" {
			count++
		}
	}
	return count
}

func failedAssertionSummary(steps []deliveryramp.JourneyStep) string {
	failed := make([]string, 0)
	for _, step := range steps {
		if step.AssertionStatus == "failed" {
			failed = append(failed, fmt.Sprintf("%s (%s)", step.ID, step.ExpectedState))
		}
	}
	return strings.Join(failed, ", ")
}

func clockOffsetSample(sample ClockSample) *deliveryramp.ClockOffsetSample {
	return &deliveryramp.ClockOffsetSample{CapturedAt: sample.HostTime, HostTime: sample.HostTime, DeviceTime: sample.DeviceTime, OffsetMs: sample.OffsetMs, UncertaintyMs: sample.UncertaintyMs, Evidence: sample.Evidence}
}
