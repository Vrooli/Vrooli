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

func (s *Service) Validate(ctx context.Context, flow Flow, strategyID string) GapReport {
	d, ok := s.strategyForDevice(strategyID)
	if !ok {
		return GapReport{Gaps: []string{"unknown strategy " + strategyID}}
	}
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
	sess, err := s.sessionForLease(deviceID, leaseToken)
	if err != nil {
		return RunResult{}, err
	}
	return s.execute(ctx, flow, deviceID, actor, sess, false)
}

func (s *Service) execute(ctx context.Context, flow Flow, deviceID, actor string, sess Session, releaseLease bool) (RunResult, error) {
	strat, ok := s.strategyForDevice(deviceID)
	if !ok {
		return RunResult{}, fmt.Errorf("unknown device %q", deviceID)
	}
	g := s.Validate(ctx, flow, deviceID)
	if !g.Runnable {
		return RunResult{RunID: uuid.NewString(), Disposition: "capability_gap", Chapters: []Chapter{{ID: "preflight", Title: "Capability preflight", Disposition: "failed", Message: strings.Join(g.Gaps, "; ")}}}, nil
	}
	if flow.AllowUnredactedCapture {
		if err := evidence.ValidateOptOut(actor); err != nil {
			return RunResult{}, err
		}
	}
	if sess.ID == "" {
		acquired, acquireErr := s.Acquire(deviceID, actor, 10*time.Minute)
		if acquireErr != nil {
			return RunResult{}, acquireErr
		}
		sess = acquired
		releaseLease = true
	}
	if releaseLease {
		defer func() { _, _ = s.Release(sess.ID) }()
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
				stepErr = strat.Actuate(stepctx, strategy.Actuation{Pointer: &strategy.PointerEvent{Kind: "tap", X: x, Y: y}})
			}
		case "key":
			stepErr = strat.Actuate(stepctx, strategy.Actuation{Key: &strategy.KeyEvent{Kind: "press", Key: step.Target}})
		case "swipe":
			stepErr = strat.Actuate(stepctx, strategy.Actuation{Action: "swipe", Value: step.Target})
		case "text":
			stepErr = strat.Actuate(stepctx, strategy.Actuation{Text: step.Target})
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
					} else if re = s.persistArtifact(id, redacted.Bytes, "image"); re != nil {
						stepErr = re
						redactionVerified = false
					} else {
						result.Evidence = append(result.Evidence, ref)
						redactionVerified = ref.RedactionVerified
					}
				}
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
			stepErr = strat.Actuate(stepctx, strategy.Actuation{Action: step.Kind, Value: step.Target, Expected: expected})
		case "rotate", "network":
			stepErr = strat.Actuate(stepctx, strategy.Actuation{Action: step.Kind, Value: step.Target})
		case "deep-link":
			packageName, _ := step.Arguments["package"].(string)
			stepErr = strat.Actuate(stepctx, strategy.Actuation{Action: step.Kind, Value: step.Target, Package: packageName})
		case "device-logs", "screenrecord":
			var output []byte
			packageName, _ := step.Arguments["package"].(string)
			stepErr = strat.Actuate(stepctx, strategy.Actuation{Action: step.Kind, Value: step.Target, Package: packageName, Output: &output})
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
					} else if refErr = s.persistArtifact(id, redacted.Bytes, "log"); refErr != nil {
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
					} else if refErr = s.persistArtifact(id, redacted.Bytes, "video"); refErr != nil {
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
			stepErr = strat.Actuate(stepctx, strategy.Actuation{Action: step.Kind, Package: packageName, Permission: permission, Value: value})
		}
		cancel()
		outcome := "success"
		if stepErr != nil {
			outcome = "failure"
			chapter.Disposition = "failed"
			chapter.Message = stepErr.Error()
			result.Disposition = "failed"
			if runctx.Err() != nil {
				chapter.Disposition = "cancelled"
				chapter.Message = "flow stopped: " + runctx.Err().Error()
				result.Disposition = "cancelled"
			}
		}
		s.mu.Lock()
		audit := Audit{ID: uuid.NewString(), Actor: actor, DeviceID: deviceID, LeaseID: sess.ID, Verb: step.Kind, Outcome: outcome, CreatedAt: time.Now().UTC(), RedactionVerified: redactionVerified, RedactionOptedOut: flow.AllowUnredactedCapture}
		if s.db != nil {
			_, _ = s.db.Exec(`INSERT INTO device_control_audits (id, actor, device_id, lease_id, verb, outcome, created_at, redaction_verified, redaction_opted_out) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, audit.ID, audit.Actor, audit.DeviceID, audit.LeaseID, audit.Verb, audit.Outcome, audit.CreatedAt.Format(time.RFC3339Nano), boolInt(redactionVerified), boolInt(flow.AllowUnredactedCapture))
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
