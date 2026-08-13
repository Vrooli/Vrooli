package control

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"device-control/internal/evidence"
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
		for _, cap := range step.RequiredCapabilities {
			if decl.Capabilities[cap].Status != strategy.StatusAvailable {
				g.Runnable = false
				g.Gaps = append(g.Gaps, fmt.Sprintf("step %s requires %s (%s)", step.ID, cap, decl.Capabilities[cap].NextAction))
			}
		}
	}
	return g
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
		switch step.Kind {
		case "tap":
			x, y, e := coordinates(step)
			if e != nil {
				stepErr = e
			} else {
				stepErr = dispatch(stepctx, strategy.Actuation{Pointer: &strategy.PointerEvent{Kind: "tap", X: x, Y: y}})
			}
		case "key":
			stepErr = dispatch(stepctx, strategy.Actuation{Key: &strategy.KeyEvent{Kind: "press", Key: step.Target}})
		case "swipe":
			stepErr = dispatch(stepctx, strategy.Actuation{Action: "swipe", Value: step.Target})
		case "text":
			stepErr = dispatch(stepctx, strategy.Actuation{Text: step.Target})
		case "observe":
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
		case "semantic-target", "semantic-assert":
			expected, _ := step.Arguments["expected"].(string)
			stepErr = dispatch(stepctx, strategy.Actuation{Action: step.Kind, Value: step.Target, Expected: expected})
		case "rotate", "network":
			stepErr = dispatch(stepctx, strategy.Actuation{Action: step.Kind, Value: step.Target})
		case "deep-link":
			packageName, _ := step.Arguments["package"].(string)
			stepErr = dispatch(stepctx, strategy.Actuation{Action: step.Kind, Value: step.Target, Package: packageName})
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
					ref, refErr := evidence.NewVideoReference(id, redacted.Bytes, redacted, "native", 30)
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
	return result, nil
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
