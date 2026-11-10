package control

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	authdomain "device-control/internal/auth"
	"device-control/internal/evidence"
	internalflows "device-control/internal/flows"
	sessionsdomain "device-control/internal/sessions"
	"device-control/strategy"
	"github.com/google/uuid"
)

func (s *Service) Run(ctx context.Context, flow Flow, deviceID, actor string) (RunResult, error) {
	return s.execute(ctx, flow, deviceID, actor, Session{}, true, false)
}

func (s *Service) RunWithLease(ctx context.Context, flow Flow, deviceID, actor, leaseToken string) (RunResult, error) {
	return s.RunWithLeaseConfirmed(ctx, flow, deviceID, actor, leaseToken, false)
}

// RunWithLeaseConfirmed keeps one-shot operator confirmation outside the
// replayable flow definition. It is required for artifact installation.
func (s *Service) RunWithLeaseConfirmed(ctx context.Context, flow Flow, deviceID, actor, leaseToken string, confirmed bool) (RunResult, error) {
	if flowRequiresConfirmation(flow) && !confirmed {
		return RunResult{}, fmt.Errorf("flow contains install-artifact and requires explicit confirmation")
	}
	if leaseToken == "" {
		return s.execute(ctx, flow, deviceID, actor, Session{}, true, confirmed)
	}
	sess, err := s.sessionForLease(ctx, deviceID, leaseToken)
	if err != nil {
		return RunResult{}, err
	}
	return s.execute(ctx, flow, deviceID, actor, sess, false, confirmed)
}

func flowRequiresConfirmation(flow Flow) bool {
	for _, step := range flow.Steps {
		if step.Kind == "install-artifact" {
			return true
		}
	}
	return false
}

func (s *Service) execute(ctx context.Context, flow Flow, deviceID, actor string, sess Session, releaseLease, confirmed bool) (RunResult, error) {
	requestedFlow := flow
	transport, routeErr := s.requireCurrentEffectRoute(deviceID, flow.Transport)
	if routeErr != nil {
		return RunResult{}, routeErr
	}
	flow.Transport = transport
	strat, ok := s.strategyForFlow(deviceID, flow.Transport)
	if !ok {
		if flow.Transport == "" {
			return RunResult{}, fmt.Errorf("unknown device %q", deviceID)
		}
		return RunResult{}, fmt.Errorf("transport %q is unavailable for device %q; request usb or promote the device before requesting wireless", flow.Transport, deviceID)
	}
	if strings.TrimSpace(flow.Transport) == "" {
		if record, found := s.devices.Get(deviceID); found {
			flow.Transport = record.Transport
			if strings.EqualFold(flow.Transport, "wireless") && !s.hasTransportStrategy(deviceID) {
				flow.Transport = "usb"
			}
		}
	}
	g, transportUnavailable := validateAgainstDeclarationDetailed(ctx, flow, strat)
	if candidate, usbReport, fallback := s.preflightUSBForImplicitWireless(ctx, deviceID, requestedFlow, g); fallback {
		strat, g = candidate, usbReport
		flow.Transport = "usb"
	}
	if transportUnavailable && !g.Runnable && strings.TrimSpace(requestedFlow.Transport) == "" && s.recoverWirelessBeforeUse(ctx, deviceID) {
		if candidate, candidateOK := s.strategyForFlow(deviceID, "wireless"); candidateOK {
			wirelessFlow := requestedFlow
			wirelessFlow.Transport = "wireless"
			if wirelessReport, _ := validateAgainstDeclarationDetailed(ctx, wirelessFlow, candidate); wirelessReport.Runnable {
				strat, flow, g = candidate, wirelessFlow, wirelessReport
			}
		}
	}
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
	runDuration := time.Duration(maxFlowDurationMS) * time.Millisecond
	if flow.MaxDurationMS > 0 && time.Duration(flow.MaxDurationMS)*time.Millisecond < runDuration {
		runDuration = time.Duration(flow.MaxDurationMS) * time.Millisecond
	}
	runctx, finishRun, err := s.registerSessionOperation(ctx, sess, deviceID, runDuration)
	if err != nil {
		return RunResult{}, err
	}
	defer finishRun()
	result := RunResult{RunID: uuid.NewString(), Disposition: "passed", Chapters: []Chapter{}, Resolutions: []Resolution{}, Evidence: []evidence.Reference{}, Binding: RunBinding{DeviceID: deviceID, Transport: flow.Transport, LeaseID: sess.ID, ApplicationID: flow.ApplicationID, ApplicationRevision: flow.ApplicationRevision}}
	stateManager := &sessionsdomain.StateManager{}
	stateRestorer, hasStateRestorer := strat.(strategy.StateRestorer)
	restorationPrepared := false
	var restorationState strategy.DeviceState
	prepareRestoration := func(snapshot strategy.DeviceState, snapshotErr error) {
		if snapshotErr != nil || !hasStateRestorer {
			return
		}
		if snapshot.LockState == "locked" {
			restorationState.LockState = snapshot.LockState
		}
		if snapshot.Orientation != "" && restorationState.Orientation == "" {
			restorationState.Orientation = snapshot.Orientation
			restorationState.AutoRotate = snapshot.AutoRotate
			restorationState.AutoRotateKnown = snapshot.AutoRotateKnown
			restorationState.RotationSettings = snapshot.RotationSettings
		}
		if restorationPrepared {
			return
		}
		stateManager.Push("device-state", func(restoreCtx context.Context) error {
			return stateRestorer.RestoreState(restoreCtx, restorationState)
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
	if flow.RequireUnlocked || flow.AuthProfileID != "" {
		reader, readerOK := strat.(strategy.StateReader)
		if !readerOK {
			return authFailure("strategy does not expose live lock-state verification")
		}
		state, stateErr := reader.ReadState(runctx)
		if stateErr != nil || (state.LockState != "locked" && state.LockState != "unlocked") {
			return authFailure("device lock state is unknown; flow stopped")
		}
		if state.LockState == "locked" {
			if flow.AuthProfileID == "" {
				return authFailure("device is locked and no authentication profile was supplied")
			}
			// Register cleanup before unlocking, including partial failures. An
			// authentication flow owns keyguard cleanup, not orientation settings.
			prepareRestoration(strategy.DeviceState{LockState: "locked"}, nil)
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
	recordingHandles := map[string]strategy.RecordingHandle{}
	recordingAudioRequired := map[string]bool{}
	recorder, hasRecorder := strat.(strategy.SessionRecorder)
	markRecordingUncertain := func(handle strategy.RecordingHandle) {
		if handle.ID == "" {
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if _, exists := s.externalRecordings[handle.ID]; !exists {
			s.externalRecordings[handle.ID] = externalRecording{DeviceID: deviceID, LeaseID: sess.ID, Actor: actor, Recorder: recorder, Handle: handle, State: stateUncertain}
		}
	}
	storeRecording := func(step Step, artifact strategy.RecordingArtifact, requireAudio bool) error {
		redacted, redactErr := evidence.CaptureWithPolicy(artifact.Bytes, "video/mp4", evidence.DefaultPolicy, flow.RedactSensitiveContent, flow.AllowUnredactedCapture, actor, nil)
		if redactErr != nil {
			return redactErr
		}
		frameCount, duration, effectiveFPS, measureErr := evidence.MeasureVideo(redacted.Bytes)
		if measureErr != nil {
			return measureErr
		}
		if artifact.Method == "screenshot-fallback" && artifact.EffectiveFPS > 0 && artifact.EffectiveFPS < effectiveFPS {
			effectiveFPS = artifact.EffectiveFPS
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
		if refErr = s.persistRecording(ctx, id, redacted.Bytes, deviceID, sess.ID); refErr != nil {
			return refErr
		}
		if requireAudio && (ref.AudioTrackPresent == nil || !*ref.AudioTrackPresent) {
			ref.Disposition = evidence.DispositionDegraded
			ref.DispositionReason = "requested audio track was absent or could not be verified"
			result.Evidence = append(result.Evidence, ref)
			return &strategy.EvidenceUnavailableError{Reason: ref.DispositionReason}
		}
		minimumMS, _ := number(step.Arguments["minimum_duration_ms"])
		if class != evidence.ClaimStatic && (frameCount < 2 || minimumMS > 0 && duration < time.Duration(minimumMS)*time.Millisecond) {
			ref.Disposition = evidence.DispositionDegraded
			ref.DispositionReason = "recording is too short to establish the requested interval"
			result.Evidence = append(result.Evidence, ref)
			return &strategy.EvidenceUnavailableError{Reason: fmt.Sprintf("recording contains %d frame(s) over %s; the requested interval was not observed", frameCount, duration)}
		}
		if ref.Disposition == evidence.DispositionDegraded {
			result.Evidence = append(result.Evidence, ref)
			return &strategy.EvidenceUnavailableError{Reason: ref.DispositionReason}
		}
		result.Evidence = append(result.Evidence, ref)
		return nil
	}
	var previousFrame, lastFrame []byte
	stepCausationID := ""
	seenIdempotencyKeys := map[string]string{}
	dispatch := func(stepctx context.Context, event strategy.Actuation) error {
		if flow.SuppressActuation {
			return nil
		}
		if event.Action == "device-logs" && event.Package != "" && event.Value == "" {
			if logReader, ok := strat.(strategy.ApplicationLogReader); ok {
				output, err := logReader.ReadApplicationLogs(stepctx, event.Package)
				if err == nil && event.Output != nil {
					*event.Output = output
				}
				return err
			}
		}
		if event.Action == "rotate" && restorationState.Orientation == "" {
			reader, readable := strat.(strategy.StateReader)
			if !readable || !hasStateRestorer {
				return fmt.Errorf("rotation requires a restorable state snapshot")
			}
			snapshot, err := reader.ReadState(stepctx)
			if err != nil || snapshot.Orientation == "" || (snapshot.RotationSettings != nil && len(snapshot.RotationSettings) != 2) {
				return fmt.Errorf("rotation baseline is unavailable")
			}
			snapshot.LockState = ""
			prepareRestoration(snapshot, nil)
		}
		if lifecycleFlowAction(event.Action) {
			lifecycle, lifecycleOK := strat.(strategy.AppLifecycle)
			if !lifecycleOK {
				return &strategy.UnsupportedCapabilityError{Capability: strategy.CapAppLifecycle, Operation: event.Action}
			}
			lifecycleResult, lifecycleErr := lifecycle.AppLifecycle(stepctx, strategy.AppLifecycleRequest{Operation: event.Action, Package: event.Package, Permission: event.Permission, Value: event.Value})
			if lifecycleErr != nil {
				return lifecycleErr
			}
			if lifecycleResult.Status != strategy.StatusAvailable {
				return &strategy.AvailabilityError{Reason: lifecycleResult.Reason, NextAction: "re-probe app lifecycle capability before retrying"}
			}
			return nil
		}
		actuator, ok := strat.(strategy.InputActuator)
		if !ok {
			return &strategy.UnsupportedCapabilityError{Capability: strategy.CapInput, Operation: "actuate"}
		}
		if strings.TrimSpace(event.CausationID) == "" {
			event.CausationID = stepCausationID
			if event.CausationID == "" {
				event.CausationID = uuid.NewString()
			}
		}
		return actuator.Actuate(stepctx, event)
	}
	for _, step := range flow.Steps {
		chapter := Chapter{ID: step.ID, Title: step.Kind, Disposition: "passed", Message: "completed"}
		stepEvidenceStart := len(result.Evidence)
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
		if key := strings.TrimSpace(step.IdempotencyKey); key != "" {
			if prior, seen := seenIdempotencyKeys[key]; seen {
				chapter.Message = "deduplicated from step " + prior
				result.Chapters = append(result.Chapters, chapter)
				continue
			}
			seenIdempotencyKeys[key] = step.ID
			if s.db != nil {
				claimed, claimErr := s.claimCommandReceipt(ctx, deviceID, flow.Transport, key, step.Kind, result.RunID)
				if claimErr != nil {
					chapter.Disposition = "failed"
					chapter.Message = claimErr.Error()
					chapter.FailureClass = classifyFlowFailure(claimErr)
					result.Disposition = "failed"
					result.Chapters = append(result.Chapters, chapter)
					break
				}
				if !claimed {
					chapter.Message = "deduplicated from a prior command receipt"
					result.Chapters = append(result.Chapters, chapter)
					continue
				}
			}
		}
		stepctx, cancel := context.WithTimeout(runctx, time.Duration(step.TimeoutMS)*time.Millisecond)
		stepCausationID = uuid.NewString()
		var stepErr error
		redactionVerified := false
		noEvidence := false
		executeAttempt := func() error {
			stepErr = nil
			redactionVerified = false
			if requiresVisibleSurface(step.Kind) {
				if stateReader, stateOK := strat.(strategy.StateReader); stateOK {
					state, stateErr := stateReader.ReadState(stepctx)
					if stateErr != nil || state.LockState != "unlocked" || state.ScreenState != "on" {
						stepErr = fmt.Errorf("visible surface unavailable: lock state is %q and screen state is %q", state.LockState, state.ScreenState)
					}
				}
			}
			if stepErr == nil && step.ObservationRequired && len(lastFrame) == 0 {
				stepErr = fmt.Errorf("step %q requires a prior observation", step.ID)
			}
			if stepErr == nil {
				for _, condition := range step.Preconditions {
					if conditionErr := s.evaluateFlowCondition(stepctx, strat, condition, previousFrame, lastFrame); conditionErr != nil {
						stepErr = fmt.Errorf("precondition %q failed: %w", condition.Kind, conditionErr)
						break
					}
				}
			}
			// Do not dispatch screenshot-dependent work until a fresh state probe
			// proves the device is awake and unlocked.
			if stepErr == nil {
				switch step.Kind {
				case "wireless-debugging-assert":
					reader, supported := strat.(strategy.StateReader)
					if !supported {
						stepErr = fmt.Errorf("strategy cannot observe wireless debugging state")
						break
					}
					observed, err := reader.ReadState(stepctx)
					if err != nil || observed.WirelessDebugging == nil || observed.WirelessDebugging.NativeEnabled == nil || *observed.WirelessDebugging.NativeEnabled != (step.Target == "enable") {
						stepErr = fmt.Errorf("native wireless debugging expected state was not observed")
					}
				case "wireless-debugging":
					if flow.SuppressActuation {
						break
					}
					controller, supported := strat.(strategy.WirelessDebuggingController)
					if !supported {
						stepErr = fmt.Errorf("strategy does not implement native wireless debugging control")
						break
					}
					stepErr = controller.SetWirelessDebugging(stepctx, step.Target == "enable")
				case "reverse-tcp-remove":
					remover, ok := strat.(strategy.ReverseTCPRemover)
					if !ok {
						stepErr = fmt.Errorf("selected strategy does not support reverse TCP recovery")
						break
					}
					devicePort, hostPort, parseErr := reverseTCPPorts(step)
					if parseErr != nil {
						stepErr = parseErr
						break
					}
					stepErr = remover.RemoveReverseTCP(stepctx, devicePort, hostPort)

				case "reverse-tcp":
					forwarder, supported := strat.(strategy.ReverseTCPForwarder)
					if !supported {
						stepErr = fmt.Errorf("strategy does not implement reverse TCP forwarding")
						break
					}
					devicePort, hostPort, parseErr := reverseTCPPorts(step)
					if parseErr != nil {
						stepErr = parseErr
						break
					}
					cleanup, openErr := forwarder.OpenReverseTCP(stepctx, devicePort, hostPort)
					if cleanup != nil {
						stateManager.Push(fmt.Sprintf("reverse-tcp-%d", devicePort), cleanup)
					}
					stepErr = openErr
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
					observer, observerOK := strat.(strategy.Observer)
					if !observerOK {
						stepErr = &strategy.UnsupportedCapabilityError{Capability: strategy.CapScreenshot, Operation: "observe"}
						break
					}
					frame, e := observer.Observe(stepctx)
					stepErr = e
					if stepErr == nil {
						redacted, re := evidence.CaptureWithPolicy(frame.Bytes, frame.MediaType, evidence.DefaultPolicy, flow.RedactSensitiveContent, flow.AllowUnredactedCapture, actor, sensitiveRegions(step))
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
								if ref.Disposition == evidence.DispositionDegraded {
									stepErr = &strategy.EvidenceUnavailableError{Reason: ref.DispositionReason}
								}
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
				case "property-assert":
					property, ok := strat.(strategy.PropertyActuator)
					if !ok {
						stepErr = &strategy.UnsupportedCapabilityError{Capability: strategy.CapProperty, Operation: "assert-property"}
						break
					}
					name := step.Target
					if v, ok := step.Arguments["name"].(string); ok && v != "" {
						name = v
					}
					expected, supplied := step.Arguments["equals"]
					if name == "" || !supplied {
						stepErr = fmt.Errorf("property assertion requires name and equals")
						break
					}
					actual, err := property.GetProperty(stepctx, name)
					if err != nil {
						stepErr = err
					} else if !reflect.DeepEqual(actual, expected) {
						stepErr = fmt.Errorf("property assertion failed for %s", name)
					}
				case "property-get":
					property, ok := strat.(strategy.PropertyActuator)
					if !ok {
						stepErr = &strategy.UnsupportedCapabilityError{Capability: strategy.CapProperty, Operation: "get-property"}
						break
					}
					name := step.Target
					if value, ok := step.Arguments["name"].(string); ok && value != "" {
						name = value
					}
					_, stepErr = property.GetProperty(stepctx, name)
				case "property-set":
					property, ok := strat.(strategy.PropertyActuator)
					if !ok {
						stepErr = &strategy.UnsupportedCapabilityError{Capability: strategy.CapProperty, Operation: "set-property"}
						break
					}
					name := step.Target
					if value, ok := step.Arguments["name"].(string); ok && value != "" {
						name = value
					}
					value, valueOK := step.Arguments["value"]
					if !valueOK {
						value = step.Target
					}
					oldValue, oldErr := property.GetProperty(stepctx, name)
					if oldErr != nil {
						stepErr = oldErr
						break
					}
					stepErr = property.SetProperty(stepctx, strategy.PropertySet{Name: name, Value: value, CausationID: stepCausationID})
					if stepErr == nil {
						s.rememberObservedState(deviceID, flow.Transport, name, value)
						stateClass := strategy.StateBearing
						if declaration, describeErr := strat.Describe(stepctx); describeErr == nil {
							for _, descriptor := range declaration.Properties {
								if descriptor.Name == name && descriptor.StateClass != "" {
									stateClass = descriptor.StateClass
								}
							}
						}
						s.EmitStateChange(strategy.StateChangeEvent{DeviceID: deviceID, Transport: flow.Transport, Attribute: name, OldValue: oldValue, NewValue: value, CausationID: stepCausationID, StateClass: stateClass})
					}
				case "sensor-read":
					reader, ok := strat.(strategy.SensorReader)
					if !ok {
						stepErr = &strategy.UnsupportedCapabilityError{Capability: strategy.CapSensor, Operation: "read-sensors"}
						break
					}
					var readings []strategy.SensorReading
					readings, stepErr = reader.ReadSensors(stepctx)
					if stepErr == nil {
						for _, reading := range readings {
							oldValue, changed, hadPrevious := s.observeState(deviceID, flow.Transport, reading.Name, reading.Value)
							if !hadPrevious || !changed {
								continue
							}
							observedAt := reading.ObservedAt
							if observedAt.IsZero() {
								observedAt = time.Now().UTC()
							}
							stateClass := reading.StateClass
							if stateClass == "" {
								stateClass = strategy.StateBearing
							}
							s.EmitStateChange(strategy.StateChangeEvent{DeviceID: deviceID, Transport: flow.Transport, Attribute: reading.Name, OldValue: oldValue, NewValue: reading.Value, ObservedAt: observedAt, CausationID: stepCausationID, StateClass: stateClass})
						}
					}
				case "media-play", "media-pause", "media-stop", "media-next", "media-previous", "media-volume":
					media, ok := strat.(strategy.MediaController)
					if !ok {
						stepErr = &strategy.UnsupportedCapabilityError{Capability: strategy.CapMedia, Operation: "control-media"}
						break
					}
					action := strings.TrimPrefix(step.Kind, "media-")
					var value any
					if raw, present := step.Arguments["value"]; present {
						value = raw
					} else if step.Target != "" {
						value = step.Target
					}
					stepErr = media.ControlMedia(stepctx, strategy.MediaCommand{Action: action, Value: value, CausationID: stepCausationID})
					if stepErr == nil {
						s.EmitStateChange(strategy.StateChangeEvent{DeviceID: deviceID, Transport: flow.Transport, Attribute: "media." + action, NewValue: value, CausationID: stepCausationID, StateClass: strategy.EventBearing})
					}
				case "semantic-target":
					semantic, ok := strat.(strategy.SemanticResolver)
					if !ok {
						stepErr = fmt.Errorf("strategy does not implement deterministic semantic resolution")
						break
					}
					observer, observerOK := strat.(strategy.Observer)
					if !observerOK {
						stepErr = &strategy.UnsupportedCapabilityError{Capability: strategy.CapScreenshot, Operation: "semantic-target"}
						break
					}
					frame, observeErr := observer.Observe(stepctx)
					if observeErr != nil {
						stepErr = observeErr
						break
					}
					resolver := internalflows.NewResolverWithAnchors(nil, s.anchors)
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
				case "restart-device":
					restarter, ok := strat.(strategy.DeviceRestarter)
					if !ok {
						stepErr = &strategy.UnsupportedCapabilityError{Capability: strategy.CapDeviceRestart, Operation: step.Kind}
						break
					}
					if !flow.SuppressActuation {
						stepErr = restarter.RestartDevice(stepctx)
					}
				case "assert-foreground":
					reader, ok := strat.(strategy.StateReader)
					if !ok {
						stepErr = fmt.Errorf("strategy does not expose live foreground state")
						break
					}
					if strings.TrimSpace(step.Target) == "" {
						stepErr = fmt.Errorf("foreground assertion requires an application identity")
						break
					}
					state, err := reader.ReadState(stepctx)
					if err != nil {
						stepErr = err
						break
					}
					if state.ForegroundPackage == "" || state.Unavailable["foreground_package"] != "" {
						stepErr = fmt.Errorf("foreground application is unknown")
						break
					}
					if state.ForegroundPackage != step.Target {
						stepErr = fmt.Errorf("foreground application %q does not match %q", state.ForegroundPackage, step.Target)
					}
				case "rotate", "network":
					stepErr = dispatch(stepctx, strategy.Actuation{Action: step.Kind, Value: step.Target})
				case "deep-link":
					packageName, _ := step.Arguments["package"].(string)
					stepErr = dispatch(stepctx, strategy.Actuation{Action: step.Kind, Value: step.Target, Package: packageName})
				case "share":
					packageName, _ := step.Arguments["package"].(string)
					stepErr = dispatch(stepctx, strategy.Actuation{Action: step.Kind, Value: step.Target, Package: packageName})
				case "screen":
					if flow.SuppressActuation {
						break
					}
					if screen, ok := strat.(strategy.ScreenController); ok && (step.Target == "wake" || step.Target == "sleep") {
						wanted := "on"
						if step.Target == "sleep" {
							wanted = "off"
						}
						stepErr = screen.SetScreenState(stepctx, wanted)
					} else {
						stepErr = dispatch(stepctx, strategy.Actuation{Action: step.Kind, Value: step.Target})
					}
				case "logcat-start", "clipboard-write":
					stepErr = dispatch(stepctx, strategy.Actuation{Action: step.Kind, Value: step.Target})
				case "recording-start":
					if !hasRecorder {
						stepErr = fmt.Errorf("strategy does not support session-scoped recording")
						break
					}
					class := strategy.ClaimClass(claimClassForStep(step))
					var handle strategy.RecordingHandle
					requireAudio, _ := step.Arguments["require_audio"].(bool)
					if requireAudio {
						if audioRecorder, ok := strat.(strategy.AudioSessionRecorder); ok {
							handle, stepErr = audioRecorder.StartRecordingWithAudio(stepctx, class)
						} else {
							stepErr = fmt.Errorf("strategy does not support recording with audio")
						}
					} else {
						handle, stepErr = recorder.StartRecording(stepctx, class)
					}
					if stepErr == nil {
						recordingHandles[step.ID] = handle
						recordingAudioRequired[step.ID] = requireAudio
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
					// Stop may consume a native handle even on error; never replay it.
					delete(recordingHandles, key)
					requireAudio := recordingAudioRequired[key]
					delete(recordingAudioRequired, key)
					var artifact strategy.RecordingArtifact
					artifact, stepErr = recorder.StopRecording(stepctx, handle)
					if stepErr != nil {
						markRecordingUncertain(handle)
						result.Incomplete = true
					}
					if stepErr == nil {
						stepErr = storeRecording(step, artifact, requireAudio)
					}
				case "device-logs", "framework-logs", "logcat-stop", "clock-sample", "screenshot", "clipboard-read", "screenrecord":
					var output []byte
					packageName, _ := step.Arguments["package"].(string)
					component, _ := step.Arguments["component"].(string)
					logTag, _ := step.Arguments["tag"].(string)
					logPriority, _ := step.Arguments["priority"].(string)
					stepErr = dispatch(stepctx, strategy.Actuation{Action: step.Kind, Value: step.Target, Package: packageName, Component: component, LogTag: logTag, LogPriority: logPriority, Output: &output})
					if stepErr == nil && len(output) == 0 {
						if step.Kind == "device-logs" {
							noEvidence = true
							result.Disposition = "no_evidence"
							chapter.Disposition = "unavailable"
							chapter.Message = "No application log entries were available in the bounded snapshot."
							break
						}
						stepErr = fmt.Errorf("%s produced no output", step.Kind)
					}
					if stepErr == nil && (step.Kind == "device-logs" || step.Kind == "framework-logs" || step.Kind == "logcat-stop" || step.Kind == "clock-sample") {
						redacted, redactErr := evidence.CaptureWithPolicy(output, "text/plain", evidence.DefaultPolicy, flow.RedactSensitiveContent, flow.AllowUnredactedCapture, actor, nil)
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
					if stepErr == nil && step.Kind == "screenshot" {
						redacted, redactErr := evidence.CaptureWithPolicy(output, "image/png", evidence.DefaultPolicy, flow.RedactSensitiveContent, flow.AllowUnredactedCapture, actor, nil)
						if redactErr != nil {
							stepErr = redactErr
						} else {
							id := uuid.NewString()
							ref, refErr := evidence.NewReference(id, redacted.Bytes, redacted)
							if refErr != nil {
								stepErr = refErr
							} else if refErr = s.persistArtifact(ctx, id, redacted.Bytes, "screenshot"); refErr != nil {
								stepErr = refErr
							} else {
								result.Evidence = append(result.Evidence, ref)
								redactionVerified = ref.RedactionVerified
							}
						}
					}
					if stepErr == nil && step.Kind == "screenrecord" {
						redacted, redactErr := evidence.CaptureWithPolicy(output, "video/mp4", evidence.DefaultPolicy, flow.RedactSensitiveContent, flow.AllowUnredactedCapture, actor, nil)
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
							} else if refErr = s.persistRecording(ctx, id, redacted.Bytes, deviceID, sess.ID); refErr != nil {
								stepErr = refErr
							} else {
								result.Evidence = append(result.Evidence, ref)
								redactionVerified = ref.RedactionVerified
							}
						}
					}
				case "home-set", "install", "launch", "stop", "uninstall", "clear-data", "grant-permission", "revoke-permission", "package-state":
					packageName := step.Target
					if value, ok := step.Arguments["package"].(string); ok && value != "" {
						packageName = value
					}
					permission := step.Target
					if value, ok := step.Arguments["permission"].(string); ok && value != "" {
						permission = value
					}
					value := step.Target
					if argumentValue, ok := step.Arguments["value"].(string); ok && argumentValue != "" {
						value = argumentValue
					}
					if step.Kind == "package-state" {
						value, _ = step.Arguments["expected"].(string)
					}
					stepErr = dispatch(stepctx, strategy.Actuation{Action: step.Kind, Package: packageName, Permission: permission, Value: value})
				case "install-artifact":
					chapter.ArtifactInstall, stepErr = s.installVerifiedAndroidArtifact(stepctx, deviceID, sess, strat, step)
					if stepErr == nil {
						proof := chapter.ArtifactInstall
						chapter.Message = fmt.Sprintf("installed and verified %s version %s from %s", proof.PackageName, proof.VersionName, proof.ImmutableRef)
					}
				}
			}
			if stepErr == nil {
				for _, condition := range step.Postconditions {
					conditionErr := s.evaluateFlowCondition(stepctx, strat, condition, previousFrame, lastFrame)
					kind := strings.ToLower(strings.TrimSpace(condition.Kind))
					if conditionErr != nil && step.Kind == "screen" && (kind == "screen-on" || kind == "screen-off") {
						conditionErr = s.waitForScreenPostcondition(stepctx, strat, condition, previousFrame, lastFrame, conditionErr)
					}
					if conditionErr != nil {
						stepErr = fmt.Errorf("postcondition %q failed: %w", condition.Kind, conditionErr)
						break
					}
				}
			}
			return stepErr
		}
		retryBudget := step.RetryBudget
		if flow.RetryBudget > 0 && (retryBudget == 0 || flow.RetryBudget < retryBudget) {
			retryBudget = flow.RetryBudget
		}
		attempts := 0
		for {
			stepErr = executeAttempt()
			if stepErr == nil || attempts >= retryBudget || !retryableFlowStep(step.Kind) {
				break
			}
			attempts++
			select {
			case <-time.After(10 * time.Millisecond):
			case <-stepctx.Done():
				stepErr = stepctx.Err()
			}
		}
		if key := strings.TrimSpace(step.IdempotencyKey); key != "" && s.db != nil {
			outcome := "passed"
			if stepErr != nil {
				outcome = "failed"
				if runctx.Err() != nil {
					outcome = "unknown"
				}
			}
			if receiptErr := s.finishCommandReceipt(ctx, deviceID, flow.Transport, key, outcome); receiptErr != nil && stepErr == nil {
				stepErr = receiptErr
			}
		}
		chapter.Attempts = attempts + 1
		if stepErr != nil && runctx.Err() == nil && (step.Kind == "semantic-assert" || step.Kind == "semantic-target" || step.Kind == "assert-foreground") {
			var unavailable *strategy.AvailabilityError
			if !errors.As(stepErr, &unavailable) {
				diagnosticCtx, stopDiagnostic := context.WithTimeout(runctx, 3*time.Second)
				ref, captureErr := s.captureAssertionFailure(diagnosticCtx, strat, flow, step, actor)
				stopDiagnostic()
				if captureErr == nil {
					result.Evidence = append(result.Evidence, *ref)
				} else {
					// Keep the original failure; diagnostic capture never changes it to a pass.
					stepErr = fmt.Errorf("%w; failure frame unavailable", stepErr)
				}
			}
		}
		cancel()
		var evidenceUnavailable *strategy.EvidenceUnavailableError
		if errors.As(stepErr, &evidenceUnavailable) {
			noEvidence = true
			result.Disposition = "no_evidence"
			chapter.Disposition = "unavailable"
			chapter.Message = evidenceUnavailable.Reason
			stepErr = nil
		}
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
		if noEvidence {
			outcome = "no_evidence"
		}
		if stepErr != nil {
			outcome = "failure"
			chapter.FailureClass = classifyFlowFailure(stepErr)
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
		for _, ref := range result.Evidence[stepEvidenceStart:] {
			chapter.EvidenceIDs = append(chapter.EvidenceIDs, ref.ID)
		}
		s.mu.Lock()
		audit := Audit{ID: uuid.NewString(), Actor: actor, DeviceID: deviceID, Transport: flow.Transport, CausationID: stepCausationID, LeaseID: sess.ID, Verb: step.Kind, Outcome: outcome, CreatedAt: time.Now().UTC(), RedactionVerified: redactionVerified, RedactionOptedOut: flow.AllowUnredactedCapture}
		if install := chapter.ArtifactInstall; install != nil {
			audit.Details = map[string]string{
				"immutable_ref":                        install.ImmutableRef,
				"artifact_checksum":                    install.ArtifactChecksum,
				"apk_checksum":                         install.APKChecksum,
				"signing_certificate_sha256":           install.SigningCertificateSHA256,
				"installed_signing_certificate_sha256": install.InstalledSigningCertificateSHA256,
				"package_name":                         install.PackageName,
				"version_name":                         install.VersionName,
				"version_code":                         strconv.FormatInt(install.VersionCode, 10),
				"installed_version_code":               strconv.FormatInt(install.InstalledVersionCode, 10),
				"install_accepted":                     strconv.FormatBool(install.InstallAccepted),
				"installed":                            strconv.FormatBool(install.Installed),
				"installed_version":                    install.InstalledVersion,
				"post_install_verified":                strconv.FormatBool(install.PostInstallVerified),
			}
		}
		if s.db != nil {
			details, _ := json.Marshal(audit.Details)
			_, _ = s.db.ExecContext(ctx, `INSERT INTO device_control_audits (id, actor, device_id, transport, causation_id, lease_id, verb, outcome, created_at, redaction_verified, redaction_opted_out, details_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, audit.ID, audit.Actor, audit.DeviceID, audit.Transport, audit.CausationID, audit.LeaseID, audit.Verb, audit.Outcome, audit.CreatedAt.Format(time.RFC3339Nano), boolInt(redactionVerified), boolInt(flow.AllowUnredactedCapture), string(details))
		}
		s.audits = append(s.audits, audit)
		s.mu.Unlock()
		result.Chapters = append(result.Chapters, chapter)
		if stepErr != nil || noEvidence {
			break
		}
	}
	// A failed, cancelled, or incomplete flow still owns its active recorder.
	// Stop each remaining handle once under a bounded cleanup context. A failed
	// stop remains an uncertain owner marker and is never replayed implicitly.
	for key, handle := range recordingHandles {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		_, stopErr := recorder.StopRecording(cleanupCtx, handle)
		cancelCleanup()
		chapter := Chapter{ID: "cleanup-recording-" + key, Title: "recording cleanup", Disposition: "passed", Message: "recording stopped during cleanup; no recording evidence retained"}
		if stopErr != nil {
			markRecordingUncertain(handle)
			chapter.Disposition, chapter.Message = "failed", "recording stop outcome uncertain: "+stopErr.Error()
		}
		result.Chapters = append(result.Chapters, chapter)
		result.Incomplete = true
		if result.Disposition == "passed" {
			result.Disposition = "failed"
		}
	}
	restoreBeforeReturn()
	s.mu.Lock()
	s.runs[result.RunID] = result
	s.flowRuns[result.RunID] = flow
	s.runDevices[result.RunID] = deviceID
	s.mu.Unlock()
	return result, nil
}
