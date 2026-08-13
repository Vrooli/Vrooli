package control

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	authdomain "device-control/internal/auth"
	"device-control/internal/evidence"
	internalflows "device-control/internal/flows"
	sessionsdomain "device-control/internal/sessions"
	"device-control/strategy"
	"github.com/google/uuid"
)

func (s *Service) Validate(ctx context.Context, flow Flow, strategyID string) GapReport {
	d, ok := s.strategyForDevice(strategyID)
	if !ok {
		return GapReport{Gaps: []string{"unknown strategy " + strategyID}}
	}
	return validateAgainstDeclaration(ctx, flow, d)
}

func validateAgainstDeclaration(ctx context.Context, flow Flow, d strategy.Strategy) GapReport {
	decl, _ := d.Describe(ctx)
	g := GapReport{Runnable: true, Gaps: []string{}, Warnings: []string{}}
	for _, step := range flow.Steps {
		if !knownStepKind(step.Kind) {
			g.Runnable = false
			g.Gaps = append(g.Gaps, fmt.Sprintf("step %s uses unsupported kind %q", step.ID, step.Kind))
		}
		if step.Kind == "semantic-target" && decl.Capabilities[strategy.CapSemanticTree].Status != strategy.StatusAvailable {
			g.Runnable = false
			g.Gaps = append(g.Gaps, fmt.Sprintf("step %s requires %s", step.ID, strategy.CapSemanticTree))
		}
		if (step.Kind == "recording-start" || step.Kind == "recording-stop") && decl.Capabilities[strategy.CapScreenRecording].Status != strategy.StatusAvailable && decl.Capabilities[strategy.CapNativeRecording].Status != strategy.StatusAvailable {
			g.Runnable = false
			g.Gaps = append(g.Gaps, fmt.Sprintf("step %s requires %s", step.ID, strategy.CapScreenRecording))
		}
		if step.Kind == "pinch" && decl.Capabilities[strategy.CapMultiTouch].Status != strategy.StatusAvailable {
			g.Runnable = false
			g.Gaps = append(g.Gaps, fmt.Sprintf("step %s requires %s", step.ID, strategy.CapMultiTouch))
		}
		for _, cap := range step.RequiredCapabilities {
			if decl.Capabilities[cap].Status != strategy.StatusAvailable {
				g.Runnable = false
				g.Gaps = append(g.Gaps, fmt.Sprintf("step %s requires %s (%s)", step.ID, cap, decl.Capabilities[cap].NextAction))
			}
		}
	}
	return g
}

func claimClassForStep(step Step) string {
	if value, ok := step.Arguments["claim_class"].(string); ok {
		switch value {
		case string(strategy.ClaimStatic), string(strategy.ClaimTransition), string(strategy.ClaimAnimation):
			return value
		}
	}
	return string(strategy.ClaimTransition)
}

func (s *Service) Run(ctx context.Context, flow Flow, deviceID, actor string) (RunResult, error) {
	return s.execute(ctx, flow, deviceID, actor, Session{}, true)
}

func (s *Service) RunWithLease(ctx context.Context, flow Flow, deviceID, actor, leaseToken string) (RunResult, error) {
	if leaseToken == "" {
		return s.Run(ctx, flow, deviceID, actor)
	}
	sess, err := s.sessionForLease(ctx, deviceID, leaseToken)
	if err != nil {
		return RunResult{}, err
	}
	return s.execute(ctx, flow, deviceID, actor, sess, false)
}

func (s *Service) execute(ctx context.Context, flow Flow, deviceID, actor string, sess Session, releaseLease bool) (RunResult, error) {
	strat, ok := s.strategyForFlow(deviceID, flow.Transport)
	if !ok {
		if flow.Transport == "" {
			return RunResult{}, fmt.Errorf("unknown device %q", deviceID)
		}
		return RunResult{}, fmt.Errorf("transport %q is unavailable for device %q; request usb or promote the device before requesting wireless", flow.Transport, deviceID)
	}
	g := validateAgainstDeclaration(ctx, flow, strat)
	if !g.Runnable {
		return RunResult{RunID: uuid.NewString(), Disposition: "capability_gap", Chapters: []Chapter{{ID: "preflight", Title: "Capability preflight", Disposition: "failed", Message: strings.Join(g.Gaps, "; ")}}}, nil
	}
	if flow.AllowUnredactedCapture {
		if err := evidence.ValidateOptOut(actor); err != nil {
			return RunResult{}, err
		}
	}
	if sess.ID == "" {
		acquired, acquireErr := s.AcquireContext(ctx, deviceID, actor, 10*time.Minute)
		if acquireErr != nil {
			return RunResult{}, acquireErr
		}
		sess = acquired
		releaseLease = true
	}
	if releaseLease {
		defer func() { _, _ = s.ReleaseContext(ctx, sess.ID) }()
	}
	runctx, cancelRun := context.WithCancel(ctx)
	s.mu.Lock()
	s.activeCancels[sess.ID] = cancelRun
	s.mu.Unlock()
	defer func() {
		cancelRun()
		s.mu.Lock()
		delete(s.activeCancels, sess.ID)
		s.mu.Unlock()
	}()
	result := RunResult{RunID: uuid.NewString(), Disposition: "passed", Chapters: []Chapter{}, Resolutions: []Resolution{}, Evidence: []evidence.Reference{}}
	stateManager := &sessionsdomain.StateManager{}
	stateRestorer, hasStateRestorer := strat.(strategy.StateRestorer)
	restorationPrepared := false
	prepareRestoration := func(snapshot strategy.DeviceState, snapshotErr error) {
		if restorationPrepared || snapshotErr != nil || !hasStateRestorer || snapshot.Orientation == "" {
			return
		}
		state := snapshot
		stateManager.Push("device-state", func(restoreCtx context.Context) error {
			return stateRestorer.RestoreState(restoreCtx, state)
		})
		restorationPrepared = true
	}
	restoreBeforeReturn := func() {
		// A cancelled flow still owns a device state that must be restored. Detach
		// only cancellation, retain request values, and keep the cleanup bounded.
		restoreCtx, cancelRestore := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancelRestore()
		result.Restoration = stateManager.Restore(restoreCtx)
		for _, event := range result.Restoration {
			if event.Status == "failed" {
				result.Disposition = "failed"
				result.Chapters = append(result.Chapters, Chapter{ID: "restoration-" + event.Name, Title: "state restoration", Disposition: "failed", Message: event.Reason})
			}
		}
	}
	authFailure := func(message string) (RunResult, error) {
		result.Disposition = "auth_failed"
		result.Chapters = append(result.Chapters, Chapter{ID: "authentication", Title: "Require unlocked device", Disposition: "failed", Message: message})
		restoreBeforeReturn()
		return result, nil
	}
	var preAuthState *strategy.DeviceState
	if flow.RequireUnlocked || flow.AuthProfileID != "" {
		reader, readerOK := strat.(strategy.StateReader)
		if !readerOK {
			return authFailure("strategy does not expose live lock-state verification")
		}
		state, stateErr := reader.ReadState(runctx)
		if stateErr != nil || (state.LockState != "locked" && state.LockState != "unlocked") {
			return authFailure("device lock state is unknown; flow stopped")
		}
		stateSnapshot := state
		preAuthState = &stateSnapshot
		prepareRestoration(state, nil)
		if state.LockState == "locked" {
			if flow.AuthProfileID == "" {
				return authFailure("device is locked and no authentication profile was supplied")
			}
			unlock, unlockErr := s.UnlockDevice(runctx, flow.AuthProfileID, deviceID, actor, sess.LeaseToken)
			if unlockErr != nil || (unlock.Outcome != authdomain.OutcomeUnlocked && unlock.Outcome != authdomain.OutcomeAlreadyUnlocked) {
				message := unlock.Outcome
				if unlockErr != nil {
					message = unlockErr.Error()
				}
				return authFailure("unlock precondition failed: " + message)
			}
		}
	}
	if !restorationPrepared {
		if reader, readerOK := strat.(strategy.StateReader); readerOK {
			snapshot, snapshotErr := reader.ReadState(ctx)
			if preAuthState != nil {
				snapshot = *preAuthState
				snapshotErr = nil
			}
			prepareRestoration(snapshot, snapshotErr)
		}
	}
	recordingHandles := map[string]strategy.RecordingHandle{}
	recorder, hasRecorder := strat.(strategy.SessionRecorder)
	storeRecording := func(step Step, artifact strategy.RecordingArtifact) error {
		redacted, redactErr := evidence.RedactCapture(artifact.Bytes, "video/mp4", evidence.DefaultPolicy, flow.AllowUnredactedCapture, actor)
		if redactErr != nil {
			return redactErr
		}
		frameCount, duration, effectiveFPS, measureErr := evidence.MeasureVideo(redacted.Bytes)
		if measureErr != nil {
			return measureErr
		}
		artifact.FrameCount, artifact.Duration, artifact.EffectiveFPS = frameCount, duration, effectiveFPS
		id := uuid.NewString()
		class := evidence.ClaimClass(artifact.ClaimClass)
		if class == "" {
			class = evidence.ClaimClass(claimClassForStep(step))
		}
		ref, refErr := evidence.NewClaimedVideoReference(id, redacted.Bytes, redacted, artifact.Method, artifact.EffectiveFPS, class)
		if refErr != nil {
			return refErr
		}
		if refErr = s.persistArtifact(ctx, id, redacted.Bytes, "video"); refErr != nil {
			return refErr
		}
		result.Evidence = append(result.Evidence, ref)
		return nil
	}
	var previousFrame, lastFrame []byte
	dispatch := func(stepctx context.Context, event strategy.Actuation) error {
		if flow.SuppressActuation {
			return nil
		}
		return strat.Actuate(stepctx, event)
	}
	for _, step := range flow.Steps {
		chapter := Chapter{ID: step.ID, Title: step.Kind, Disposition: "passed", Message: "completed"}
		if err := runctx.Err(); err != nil {
			chapter.Disposition = "cancelled"
			chapter.Message = "flow stopped: " + err.Error()
			result.Disposition = "cancelled"
			result.Chapters = append(result.Chapters, chapter)
			break
		}
		if step.TimeoutMS <= 0 {
			step.TimeoutMS = 30000
		}
		stepctx, cancel := context.WithTimeout(runctx, time.Duration(step.TimeoutMS)*time.Millisecond)
		var stepErr error
		redactionVerified := true
		if requiresVisibleSurface(step.Kind) {
			if stateReader, stateOK := strat.(strategy.StateReader); stateOK {
				state, stateErr := stateReader.ReadState(stepctx)
				if stateErr != nil || state.LockState != "unlocked" || state.ScreenState == "off" {
					stepErr = fmt.Errorf("visible surface unavailable: lock state is %q and screen state is %q", state.LockState, state.ScreenState)
				}
			}
		}
		// Do not dispatch screenshot-dependent work until a fresh state probe
		// proves the device is awake and unlocked.
		if stepErr == nil {
			switch step.Kind {
			case "tap":
				x, y, e := coordinates(step)
				if e != nil {
					stepErr = e
				} else {
					stepErr = dispatch(stepctx, strategy.Actuation{Pointer: &strategy.PointerEvent{Kind: "tap", X: x, Y: y, Normalized: !pixelStep(step)}})
				}
			case "key":
				stepErr = dispatch(stepctx, strategy.Actuation{Key: &strategy.KeyEvent{Kind: "press", Key: step.Target}})
			case "swipe", "long-press", "double-tap", "drag", "fling":
				stepErr = executeGesture(stepctx, dispatch, step)
			case "pinch":
				stepErr = executePinch(stepctx, dispatch, step)
			case "scroll-to":
				stepErr = executeScrollTo(stepctx, dispatch, step)
			case "text":
				stepErr = dispatch(stepctx, strategy.Actuation{Text: step.Target})
			case "observe":
				if stateReader, ok := strat.(strategy.StateReader); ok {
					state, stateErr := stateReader.ReadState(stepctx)
					if stateErr != nil {
						stepErr = stateErr
						break
					}
					if state.LockState == "locked" || state.ScreenState == "off" {
						stepErr = fmt.Errorf("visible surface unavailable: screen is %s%s", state.LockState, screenStateSuffix(state.ScreenState))
						break
					}
				}
				frame, e := strat.Observe(stepctx)
				stepErr = e
				if stepErr == nil {
					redacted, re := evidence.RedactCaptureWithRegions(frame.Bytes, frame.MediaType, evidence.DefaultPolicy, flow.AllowUnredactedCapture, actor, sensitiveRegions(step))
					if re != nil {
						stepErr = re
						redactionVerified = false
					} else {
						id := uuid.NewString()
						ref, re := evidence.NewReference(id, redacted.Bytes, redacted)
						if re != nil {
							stepErr = re
							redactionVerified = false
						} else if re = s.persistArtifact(ctx, id, redacted.Bytes, "image"); re != nil {
							stepErr = re
							redactionVerified = false
						} else {
							previousFrame = lastFrame
							lastFrame = append([]byte(nil), redacted.Bytes...)
							result.Evidence = append(result.Evidence, ref)
							redactionVerified = ref.RedactionVerified
						}
					}
				}
			case "assert-frame-different":
				if len(previousFrame) == 0 || len(lastFrame) == 0 {
					stepErr = fmt.Errorf("frame-difference assertion requires two observe captures")
				} else if bytes.Equal(previousFrame, lastFrame) {
					stepErr = fmt.Errorf("frame-difference assertion failed: before and after captures are identical")
				}
			case "wait":
				settle := 25 * time.Millisecond
				if raw, ok := step.Arguments["settle_ms"].(float64); ok && raw > 0 {
					settle = time.Duration(raw) * time.Millisecond
				}
				timer := time.NewTimer(settle)
				select {
				case <-timer.C:
				case <-stepctx.Done():
					stepErr = fmt.Errorf("bounded wait exceeded %dms", step.TimeoutMS)
				}
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
			case "semantic-target":
				semantic, ok := strat.(strategy.SemanticResolver)
				if !ok {
					stepErr = fmt.Errorf("strategy does not implement deterministic semantic resolution")
					break
				}
				frame, observeErr := strat.Observe(stepctx)
				if observeErr != nil {
					stepErr = observeErr
					break
				}
				resolver := internalflows.NewResolver(nil)
				resolved, resolveErr := resolver.Resolve(stepctx, internalflows.Request{
					Target: step.Target,
					Frame:  internalflows.Frame{Bytes: frame.Bytes, MediaType: frame.MediaType, Width: frame.Width, Height: frame.Height, OriginalWidth: frame.Width, OriginalHeight: frame.Height},
					Semantic: func(ctx context.Context, target string) (internalflows.SemanticMatch, error) {
						match, err := semantic.ResolveSemantic(ctx, target)
						return internalflows.SemanticMatch{Bounds: match.Bounds, Confidence: match.Confidence}, err
					},
				})
				if resolveErr != nil {
					stepErr = resolveErr
					break
				}
				if len(resolved.DeviceBounds) != 4 {
					stepErr = fmt.Errorf("semantic resolver returned invalid device bounds")
					break
				}
				result.Resolutions = append(result.Resolutions, Resolution{Target: step.Target, Rung: resolved.Rung, Confidence: resolved.Confidence})
				stepErr = dispatch(stepctx, strategy.Actuation{Pointer: &strategy.PointerEvent{Kind: "semantic-target", X: float64((resolved.DeviceBounds[0] + resolved.DeviceBounds[2]) / 2), Y: float64((resolved.DeviceBounds[1] + resolved.DeviceBounds[3]) / 2)}})
			case "semantic-assert":
				expected, _ := step.Arguments["expected"].(string)
				stepErr = dispatch(stepctx, strategy.Actuation{Action: step.Kind, Value: step.Target, Expected: expected})
			case "rotate", "network":
				stepErr = dispatch(stepctx, strategy.Actuation{Action: step.Kind, Value: step.Target})
			case "deep-link":
				packageName, _ := step.Arguments["package"].(string)
				stepErr = dispatch(stepctx, strategy.Actuation{Action: step.Kind, Value: step.Target, Package: packageName})
			case "recording-start":
				if !hasRecorder {
					stepErr = fmt.Errorf("strategy does not support session-scoped recording")
					break
				}
				class := strategy.ClaimClass(claimClassForStep(step))
				var handle strategy.RecordingHandle
				handle, stepErr = recorder.StartRecording(stepctx, class)
				if stepErr == nil {
					recordingHandles[step.ID] = handle
				}
			case "recording-stop":
				if !hasRecorder {
					stepErr = fmt.Errorf("strategy does not support session-scoped recording")
					break
				}
				key := step.Target
				if key == "" {
					key = step.ID
				}
				handle, ok := recordingHandles[key]
				if !ok {
					stepErr = fmt.Errorf("recording %q is not active", key)
					break
				}
				var artifact strategy.RecordingArtifact
				artifact, stepErr = recorder.StopRecording(stepctx, handle)
				if stepErr == nil {
					stepErr = storeRecording(step, artifact)
				}
			case "device-logs", "screenrecord":
				var output []byte
				packageName, _ := step.Arguments["package"].(string)
				stepErr = dispatch(stepctx, strategy.Actuation{Action: step.Kind, Value: step.Target, Package: packageName, Output: &output})
				if stepErr == nil && len(output) == 0 {
					stepErr = fmt.Errorf("%s produced no output", step.Kind)
				}
				if stepErr == nil && step.Kind == "device-logs" {
					redacted, redactErr := evidence.RedactCapture(output, "text/plain", evidence.DefaultPolicy, flow.AllowUnredactedCapture, actor)
					if redactErr != nil {
						stepErr = redactErr
					} else {
						id := uuid.NewString()
						ref, refErr := evidence.NewLogReference(id, redacted.Bytes, redacted)
						if refErr != nil {
							stepErr = refErr
						} else if refErr = s.persistArtifact(ctx, id, redacted.Bytes, "log"); refErr != nil {
							stepErr = refErr
						} else {
							result.Evidence = append(result.Evidence, ref)
							redactionVerified = ref.RedactionVerified
						}
					}
				}
				if stepErr == nil && step.Kind == "screenrecord" {
					redacted, redactErr := evidence.RedactCapture(output, "video/mp4", evidence.DefaultPolicy, flow.AllowUnredactedCapture, actor)
					if redactErr != nil {
						stepErr = redactErr
					} else {
						id := uuid.NewString()
						_, _, effectiveFPS, measureErr := evidence.MeasureVideo(redacted.Bytes)
						if measureErr != nil {
							stepErr = measureErr
							break
						}
						ref, refErr := evidence.NewClaimedVideoReference(id, redacted.Bytes, redacted, "native", effectiveFPS, evidence.ClaimClass(claimClassForStep(step)))
						if refErr != nil {
							stepErr = refErr
						} else if refErr = s.persistArtifact(ctx, id, redacted.Bytes, "video"); refErr != nil {
							stepErr = refErr
						} else {
							result.Evidence = append(result.Evidence, ref)
							redactionVerified = ref.RedactionVerified
						}
					}
				}
			case "install", "launch", "stop", "uninstall", "clear-data", "grant-permission", "revoke-permission", "package-state":
				packageName := step.Target
				if value, ok := step.Arguments["package"].(string); ok && value != "" {
					packageName = value
				}
				permission := step.Target
				if value, ok := step.Arguments["permission"].(string); ok && value != "" {
					permission = value
				}
				value := step.Target
				if step.Kind == "package-state" {
					value, _ = step.Arguments["expected"].(string)
				}
				stepErr = dispatch(stepctx, strategy.Actuation{Action: step.Kind, Package: packageName, Permission: permission, Value: value})
			}
		}
		cancel()
		disconnected := false
		var availability *strategy.AvailabilityError
		if stepErr != nil && errors.As(stepErr, &availability) {
			disconnected = true
			result.Disposition = "device_disconnected"
			result.Incomplete = true
			result.DisconnectReason = availability.Reason
			result.DisconnectStep = step.ID
			chapter.Disposition = "device_disconnected"
			chapter.Message = "device transport stopped answering: " + availability.Reason
			_, _ = s.ReleaseContext(ctx, sess.ID)
		}
		outcome := "success"
		if stepErr != nil {
			outcome = "failure"
			if !disconnected {
				chapter.Disposition = "failed"
				chapter.Message = stepErr.Error()
				result.Disposition = "failed"
			}
			if runctx.Err() != nil && !disconnected {
				chapter.Disposition = "cancelled"
				chapter.Message = "flow stopped: " + runctx.Err().Error()
				result.Disposition = "cancelled"
			}
		}
		s.mu.Lock()
		audit := Audit{ID: uuid.NewString(), Actor: actor, DeviceID: deviceID, LeaseID: sess.ID, Verb: step.Kind, Outcome: outcome, CreatedAt: time.Now().UTC(), RedactionVerified: redactionVerified, RedactionOptedOut: flow.AllowUnredactedCapture}
		if s.db != nil {
			_, _ = s.db.ExecContext(ctx, `INSERT INTO device_control_audits (id, actor, device_id, lease_id, verb, outcome, created_at, redaction_verified, redaction_opted_out) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, audit.ID, audit.Actor, audit.DeviceID, audit.LeaseID, audit.Verb, audit.Outcome, audit.CreatedAt.Format(time.RFC3339Nano), boolInt(redactionVerified), boolInt(flow.AllowUnredactedCapture))
		}
		s.audits = append(s.audits, audit)
		s.mu.Unlock()
		result.Chapters = append(result.Chapters, chapter)
		if stepErr != nil {
			break
		}
	}
	restoreBeforeReturn()
	s.mu.Lock()
	s.runs[result.RunID] = result
	s.flowRuns[result.RunID] = flow
	s.mu.Unlock()
	return result, nil
}

type FlowExport struct {
	RunID    string            `json:"run_id"`
	Flow     Flow              `json:"flow"`
	Rungs    map[string]string `json:"rungs"`
	Excluded []string          `json:"excluded_steps,omitempty"`
}

func (s *Service) ExportFlow(id string) (FlowExport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result, ok := s.runs[id]
	flow, flowOK := s.flowRuns[id]
	if !ok || !flowOK {
		return FlowExport{}, fmt.Errorf("run %q not found", id)
	}
	export := FlowExport{RunID: id, Flow: flow, Rungs: map[string]string{}, Excluded: []string{}}
	failed := map[string]bool{}
	for _, chapter := range result.Chapters {
		if chapter.Disposition == "failed" || chapter.Disposition == "cancelled" || chapter.Disposition == "device_disconnected" {
			failed[chapter.ID] = true
			export.Excluded = append(export.Excluded, chapter.ID+": "+chapter.Message)
		}
	}
	steps := make([]Step, 0, len(flow.Steps))
	for _, step := range flow.Steps {
		if !failed[step.ID] {
			steps = append(steps, step)
		}
	}
	export.Flow.Steps = steps
	for _, resolution := range result.Resolutions {
		export.Rungs[resolution.Target] = resolution.Rung
	}
	return export, nil
}

func screenStateSuffix(screen string) string {
	if screen == "" {
		return ""
	}
	return " and screen is " + screen
}

func requiresVisibleSurface(kind string) bool {
	switch kind {
	case "observe", "semantic-target", "recording-start", "screenrecord":
		return true
	default:
		return false
	}
}

func (s *Service) strategyForDevice(deviceID string) (strategy.Strategy, bool) {
	if strat, ok := s.registry.Get(deviceID); ok {
		return strat, true
	}
	record, ok := s.devices.Get(deviceID)
	if !ok {
		return nil, false
	}
	strat, ok := s.registry.Get(record.StrategyID)
	if !ok {
		return nil, false
	}
	if scoped, ok := strat.(strategy.DeviceScoped); ok && record.Serial != "" {
		return scoped.ForDevice(record.Serial), true
	}
	return strat, true
}

func pixelStep(step Step) bool {
	space, _ := step.Arguments["coordinate_space"].(string)
	return strings.EqualFold(space, "pixel")
}

func gesturePoint(step Step, prefix string) (float64, float64, error) {
	x, xok := number(step.Arguments[prefix+"x"])
	y, yok := number(step.Arguments[prefix+"y"])
	if prefix == "" {
		x, xok = number(step.Arguments["x"])
		y, yok = number(step.Arguments["y"])
	}
	if !xok || !yok || x < 0 || x > 1 || y < 0 || y > 1 {
		return 0, 0, fmt.Errorf("gesture %s requires normalized %s coordinates", step.ID, prefix)
	}
	return x, y, nil
}

func gestureDuration(step Step, fallback int) int {
	if value, ok := number(step.Arguments["duration_ms"]); ok && value > 0 {
		return int(value)
	}
	return fallback
}

func executeGesture(ctx context.Context, dispatch func(context.Context, strategy.Actuation) error, step Step) error {
	x0, y0, err := gesturePoint(step, "start_")
	if step.Kind == "long-press" || step.Kind == "double-tap" {
		x0, y0, err = gesturePoint(step, "")
	}
	if err != nil {
		return err
	}
	x1, y1 := x0, y0
	if step.Kind == "drag" || step.Kind == "fling" || step.Kind == "swipe" {
		x1, y1, err = gesturePoint(step, "end_")
		if err != nil {
			return err
		}
	}
	duration := gestureDuration(step, 300)
	if step.Kind == "long-press" {
		duration = gestureDuration(step, 800)
	}
	if step.Kind == "double-tap" {
		for i := 0; i < 2; i++ {
			if err := dispatch(ctx, strategy.Actuation{Pointer: &strategy.PointerEvent{Kind: "tap", X: x0, Y: y0, Normalized: true}}); err != nil {
				return err
			}
		}
		return nil
	}
	return dispatch(ctx, strategy.Actuation{Pointer: &strategy.PointerEvent{Kind: step.Kind, X: x0, Y: y0, Button: fmt.Sprintf("%f,%f", x1, y1), Normalized: true, DurationMS: duration}})
}

func executePinch(ctx context.Context, dispatch func(context.Context, strategy.Actuation) error, step Step) error {
	x0, y0, err := gesturePoint(step, "start_")
	if err != nil {
		return err
	}
	x1, y1, err := gesturePoint(step, "end_")
	if err != nil {
		return err
	}
	return dispatch(ctx, strategy.Actuation{Action: "pinch", Value: fmt.Sprintf("%f,%f,%f,%f", x0, y0, x1, y1)})
}

func executeScrollTo(ctx context.Context, dispatch func(context.Context, strategy.Actuation) error, step Step) error {
	// Target re-resolution is owned by the flow layer; this bounded adapter
	// action only performs one scroll attempt.
	return dispatch(ctx, strategy.Actuation{Action: "scroll-to", Value: step.Target})
}

// strategyForFlow keeps release-grade flows on USB unless they explicitly
// request wireless. Promotion stores the endpoint-bound strategy separately so
// an explicit wireless flow can use it without changing the stable identity.
func (s *Service) strategyForFlow(deviceID, requestedTransport string) (strategy.Strategy, bool) {
	requestedTransport = strings.ToLower(strings.TrimSpace(requestedTransport))
	if requestedTransport == "" {
		requestedTransport = "usb"
	}
	if requestedTransport != "usb" && requestedTransport != "wireless" {
		return nil, false
	}
	record, ok := s.devices.Get(deviceID)
	if !ok {
		if requestedTransport == "usb" {
			return s.strategyForDevice(deviceID)
		}
		return nil, false
	}
	if requestedTransport == "wireless" {
		if record.Transport != "wireless" {
			return nil, false
		}
		s.mu.Lock()
		deferred, promoted := s.transportStrategies[deviceID]
		s.mu.Unlock()
		return deferred, promoted
	}
	base, ok := s.registry.Get(record.StrategyID)
	if !ok {
		return nil, false
	}
	if scoped, ok := base.(strategy.DeviceScoped); ok && record.Serial != "" {
		return scoped.ForDevice(record.Serial), true
	}
	return base, true
}
