package control

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	authdomain "device-control/internal/auth"
	"device-control/strategy"

	"github.com/google/uuid"
)

// DirectActuation is the small, lease-owned command surface used by a remote
// control. It intentionally has no flow/evidence fields.
type DirectActuation struct {
	Key       string `json:"key,omitempty"`
	Text      string `json:"text,omitempty"`
	Media     string `json:"media,omitempty"`
	Action    string `json:"action,omitempty"`
	Property  string `json:"property,omitempty"`
	Value     any    `json:"value,omitempty"`
	Transport string `json:"transport,omitempty"`
	Repeat    int    `json:"repeat,omitempty"`
}

const maxDirectActuationRepeat = 10

const (
	screenSettlePollInterval      = 500 * time.Millisecond
	screenSettlePollAttempts      = 5
	screenSleepSettlePollAttempts = 11
)

// ScreenRequest describes explicit outcomes, never a power toggle. Ready means
// awake and unlocked; it does not navigate away from the current application.
type ScreenRequest struct {
	Device        string `json:"device"`
	Actor         string `json:"actor"`
	Operation     string `json:"operation"`
	AuthProfileID string `json:"auth_profile_id,omitempty"`
	Home          bool   `json:"home,omitempty"`
}

type ScreenResult struct {
	DeviceID  string               `json:"device_id"`
	Transport string               `json:"transport,omitempty"`
	Operation string               `json:"operation"`
	Status    string               `json:"status"`
	Before    strategy.DeviceState `json:"before"`
	After     strategy.DeviceState `json:"after"`
	Detail    string               `json:"detail,omitempty"`
}

func (s *Service) resolveExactOrUniqueDevice(ctx context.Context, selector string) (Device, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return Device{}, fmt.Errorf("device is required")
	}
	var matches []Device
	if record, found := s.devices.Get(selector); found {
		// Exact durable identity is already unambiguous. The selected adapter is
		// probed below, so a stale record cannot establish availability.
		matches = append(matches, deviceFromRecord(record))
	} else {
		for _, device := range s.Devices(ctx) {
			if device.ID == selector {
				matches = []Device{device}
				break
			}
			if strings.EqualFold(device.Name, selector) {
				matches = append(matches, device)
			}
		}
	}
	if len(matches) != 1 {
		return Device{}, fmt.Errorf("device selector must identify exactly one available device")
	}
	return matches[0], nil
}

// Screen performs one semantic operation on an exact identity or unique name.
// It keeps the lease through verification, and does not interpret delivery as
// proof of the requested display or authentication state.
func (s *Service) Screen(ctx context.Context, req ScreenRequest) (result ScreenResult, err error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	if op != "wake" && op != "sleep" && op != "ready" {
		return result, fmt.Errorf("operation must be wake, sleep, or ready")
	}
	if req.Home && op != "ready" {
		return result, fmt.Errorf("home is available only with ready")
	}
	device, err := s.resolveExactOrUniqueDevice(ctx, req.Device)
	if err != nil {
		return result, err
	}
	result = ScreenResult{DeviceID: device.ID, Operation: op, Status: "unavailable"}
	actor := firstNonEmpty(strings.TrimSpace(req.Actor), "semantic-screen")
	lease, err := s.AcquireContext(ctx, device.ID, actor, 2*time.Minute)
	if err != nil {
		return result, err
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if _, releaseErr := s.ReleaseContext(cleanupCtx, lease.ID); releaseErr != nil {
			result.Status = "unavailable"
			if err == nil {
				err = fmt.Errorf("screen operation completed but lease release failed: %w", releaseErr)
			} else {
				err = fmt.Errorf("%v; lease release failed: %w", err, releaseErr)
			}
		}
	}()
	var controller strategy.ScreenController
	var reader strategy.StateReader
	var selected strategy.Strategy
	profiles := device.Transports
	if len(profiles) == 0 {
		profiles = []strategy.DeviceTransport{{Name: device.Transport}}
	}
	for _, profile := range profiles {
		candidate, ok := s.strategyForFlow(device.ID, profile.Name)
		if !ok {
			continue
		}
		c, canControl := candidate.(strategy.ScreenController)
		r, canRead := candidate.(strategy.StateReader)
		if !canControl || !canRead {
			continue
		}
		decl, probeErr := candidate.Describe(ctx)
		if probeErr != nil || decl.Capabilities[strategy.CapScreenControl].Status != strategy.StatusAvailable {
			continue
		}
		controller, reader, selected, result.Transport = c, r, candidate, profile.Name
		break
	}
	if controller == nil {
		return result, fmt.Errorf("device has no available, state-readable screen-control transport")
	}
	result.Before, err = reader.ReadState(ctx)
	if err != nil {
		return result, fmt.Errorf("read screen state before operation: %w", err)
	}
	want := "on"
	if op == "sleep" {
		want = "off"
	}
	if result.Before.ScreenState != want {
		if err = s.ValidateLease(ctx, device.ID, lease.LeaseToken); err != nil {
			return result, err
		}
		err = controller.SetScreenState(ctx, want)
		audit := Audit{ID: uuid.NewString(), Actor: actor, DeviceID: device.ID, Transport: result.Transport, LeaseID: lease.ID, Verb: "screen-" + op, Outcome: "success", CreatedAt: time.Now().UTC(), RedactionVerified: true, Interactive: true}
		if err != nil {
			audit.Outcome = "failed"
		}
		s.persistDirectAudit(ctx, audit)
		if err != nil {
			return result, fmt.Errorf("screen command failed: %w", err)
		}
	}
	// Sleep may take longer than wake to appear in state while Android enters
	// Doze. Allow eleven state-read attempts, without repeating the effect.
	settleAttempts := screenSettlePollAttempts
	if op == "sleep" {
		settleAttempts = screenSleepSettlePollAttempts
	}
	for attempt := 0; attempt < settleAttempts; attempt++ {
		result.After, err = reader.ReadState(ctx)
		if err == nil && result.After.ScreenState == want {
			break
		}
		if attempt == settleAttempts-1 {
			return result, fmt.Errorf("screen %s was not verified", want)
		}
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(screenSettlePollInterval):
		}
	}
	if op == "ready" {
		if result.After.LockState == "locked" && req.AuthProfileID == "" {
			dismisser, supported := selected.(strategy.KeyguardDismisser)
			if supported {
				if err = s.ValidateLease(ctx, device.ID, lease.LeaseToken); err != nil {
					return result, err
				}
				err = dismisser.DismissKeyguard(ctx)
				audit := Audit{ID: uuid.NewString(), Actor: actor, DeviceID: device.ID, Transport: result.Transport, LeaseID: lease.ID, Verb: "dismiss-keyguard", Outcome: "success", CreatedAt: time.Now().UTC(), RedactionVerified: true, Interactive: true}
				if err != nil {
					audit.Outcome = "failed"
				}
				s.persistDirectAudit(ctx, audit)
				if err != nil {
					return result, fmt.Errorf("keyguard dismissal failed: %w", err)
				}
				for attempt := 0; attempt < 5; attempt++ {
					result.After, err = reader.ReadState(ctx)
					if err != nil {
						return result, err
					}
					if result.After.LockState == "unlocked" {
						break
					}
					if attempt == 4 {
						break
					}
					select {
					case <-ctx.Done():
						return result, ctx.Err()
					case <-time.After(500 * time.Millisecond):
					}
				}
			}
		}
		if result.After.LockState == "locked" && req.AuthProfileID != "" {
			unlock, unlockErr := s.UnlockDevice(ctx, req.AuthProfileID, device.ID, actor, lease.LeaseToken)
			if unlockErr != nil {
				return result, unlockErr
			}
			if unlock.Outcome != authdomain.OutcomeUnlocked && unlock.Outcome != authdomain.OutcomeAlreadyUnlocked {
				return result, fmt.Errorf("unlock was not verified: %s", unlock.Outcome)
			}
			result.After, err = reader.ReadState(ctx)
			if err != nil {
				return result, err
			}
		}
		if result.After.ScreenState != "on" || result.After.LockState != "unlocked" {
			return result, fmt.Errorf("ready requires a verified awake and unlocked device; screen=%q lock=%q", result.After.ScreenState, result.After.LockState)
		}
		if req.Home {
			navigator, supported := selected.(strategy.HomeNavigator)
			if !supported {
				return result, fmt.Errorf("selected transport does not support the Home action")
			}
			expected := ""
			if inspector, ok := selected.(strategy.Inspector); ok {
				if info, inspectErr := inspector.Inspect(ctx); inspectErr == nil {
					expected, _, _ = strings.Cut(info.HomeComponent, "/")
				}
			}
			if err = s.ValidateLease(ctx, device.ID, lease.LeaseToken); err != nil {
				return result, err
			}
			err = navigator.GoHome(ctx)
			audit := Audit{ID: uuid.NewString(), Actor: actor, DeviceID: device.ID, Transport: result.Transport, LeaseID: lease.ID, Verb: "navigate-home", Outcome: "success", CreatedAt: time.Now().UTC(), RedactionVerified: true, Interactive: true}
			if err != nil {
				audit.Outcome = "failed"
			}
			s.persistDirectAudit(ctx, audit)
			if err != nil {
				return result, fmt.Errorf("home action failed: %w", err)
			}
			for attempt := 0; attempt < 5; attempt++ {
				result.After, err = reader.ReadState(ctx)
				if err != nil {
					return result, err
				}
				if expected != "" && result.After.ForegroundPackage == expected && result.After.ScreenState == "on" && result.After.LockState == "unlocked" {
					break
				}
				if attempt == 4 {
					result.Status = "unverified"
					result.Detail = "Home was requested, but the default Home foreground was not verified; an override may be active."
					return result, nil
				}
				select {
				case <-ctx.Done():
					return result, ctx.Err()
				case <-time.After(500 * time.Millisecond):
				}
			}
		}
	}
	result.Status = "verified"
	result.Detail = "Device-reported screen state verified; physical panel output may require independent confirmation."
	return result, nil
}

func (s *Service) ActuateDevice(ctx context.Context, deviceID, actor, leaseToken string, request DirectActuation) (Audit, error) {
	if strings.TrimSpace(leaseToken) == "" {
		return Audit{}, fmt.Errorf("direct actuation requires a lease")
	}
	session, err := s.sessionForLease(ctx, deviceID, leaseToken)
	if err != nil {
		return Audit{}, err
	}
	if strings.TrimSpace(request.Transport) == "" {
		request.Transport = s.transportForDirectActuation(deviceID, request)
	}
	adapter, ok := s.strategyForFlow(deviceID, request.Transport)
	if !ok {
		return Audit{}, fmt.Errorf("transport %q is unavailable for device %q", request.Transport, deviceID)
	}
	transport := strings.TrimSpace(request.Transport)
	if transport == "" {
		if record, found := s.devices.Get(deviceID); found {
			transport = strings.TrimSpace(record.Transport)
		}
	}
	if transport == "" {
		transport = "usb"
	}
	inputRequested := request.Key != "" || request.Text != ""
	mediaAction := strings.TrimSpace(request.Media)
	if mediaAction == "" {
		mediaAction = strings.TrimSpace(request.Action)
	}
	requested := 0
	if inputRequested {
		requested++
	}
	if request.Property != "" {
		requested++
	}
	if mediaAction != "" {
		requested++
	}
	if requested != 1 {
		return Audit{}, fmt.Errorf("direct actuation requires exactly one of key/text, media, or property")
	}
	if request.Key != "" && request.Text != "" {
		return Audit{}, fmt.Errorf("direct actuation key and text cannot be combined")
	}
	repeat := request.Repeat
	if repeat == 0 {
		repeat = 1
	}
	if repeat < 1 || repeat > maxDirectActuationRepeat {
		return Audit{}, fmt.Errorf("repeat must be between 1 and %d", maxDirectActuationRepeat)
	}
	if request.Key == "" && repeat != 1 {
		return Audit{}, fmt.Errorf("repeat is supported only for key actuation")
	}
	causationID := uuid.NewString()
	s.mu.Lock()
	s.actuationCauses[deviceID] = actuationCause{id: causationID, createdAt: time.Now()}
	s.mu.Unlock()
	if inputRequested {
		input, ok := adapter.(strategy.InputActuator)
		if !ok {
			return Audit{}, &strategy.UnsupportedCapabilityError{Capability: strategy.CapInput, Operation: "direct actuation"}
		}
		for i := 0; i < repeat && err == nil; i++ {
			err = input.Actuate(ctx, strategy.Actuation{Key: keyEvent(request.Key), Text: request.Text, CausationID: causationID})
		}
	} else if request.Property != "" {
		property, ok := adapter.(strategy.PropertyActuator)
		if !ok {
			return Audit{}, &strategy.UnsupportedCapabilityError{Capability: strategy.CapProperty, Operation: "direct property actuation"}
		}
		err = property.SetProperty(ctx, strategy.PropertySet{Name: request.Property, Value: request.Value, CausationID: causationID})
	} else if mediaAction != "" {
		media, ok := adapter.(strategy.MediaController)
		if !ok {
			return Audit{}, &strategy.UnsupportedCapabilityError{Capability: strategy.CapMedia, Operation: "direct media actuation"}
		}
		err = media.ControlMedia(ctx, strategy.MediaCommand{Action: mediaAction, Value: request.Value, CausationID: causationID})
	} else {
		return Audit{}, fmt.Errorf("direct actuation requires key, text, property, or action")
	}
	outcome := "success"
	if err != nil {
		outcome = "failed"
	}
	record := Audit{ID: uuid.NewString(), Actor: actor, DeviceID: deviceID, Transport: transport, CausationID: causationID, LeaseID: session.ID, Verb: "direct-actuation", Outcome: outcome, CreatedAt: time.Now().UTC(), RedactionVerified: true, Interactive: true, EvidenceBacked: false}
	// A command failure is still an interactive actuation attempt and must be
	// auditable. It remains explicitly non-evidence-backed; only flow runs
	// create reproducible evidence.
	s.persistDirectAudit(ctx, record)
	if err != nil {
		return Audit{}, err
	}
	return record, nil
}

func keyEvent(key string) *strategy.KeyEvent {
	if strings.TrimSpace(key) == "" {
		return nil
	}
	return &strategy.KeyEvent{Kind: "press", Key: key}
}

func (s *Service) transportForDirectActuation(deviceID string, request DirectActuation) string {
	record, ok := s.devices.Get(deviceID)
	if !ok {
		return ""
	}
	mediaAction := strings.TrimSpace(request.Media)
	if mediaAction == "" {
		mediaAction = strings.TrimSpace(request.Action)
	}
	for _, profile := range record.Transports {
		candidate, candidateOK := s.strategyForFlow(deviceID, profile.Name)
		if !candidateOK {
			continue
		}
		if request.Property != "" {
			if _, ok := candidate.(strategy.PropertyActuator); ok && profile.Capabilities[strategy.CapProperty].Status == strategy.StatusAvailable {
				return profile.Name
			}
		}
		if request.Key != "" || request.Text != "" {
			if _, ok := candidate.(strategy.InputActuator); ok && profile.Capabilities[strategy.CapInput].Status == strategy.StatusAvailable {
				return profile.Name
			}
		}
		if mediaAction != "" {
			if _, ok := candidate.(strategy.MediaController); ok && profile.Capabilities[strategy.CapMedia].Status == strategy.StatusAvailable {
				if profile.Capabilities[strategy.CapMedia].StateClass == strategy.StateBearing {
					return profile.Name
				}
			}
		}
	}
	return ""
}

func (s *Service) persistDirectAudit(ctx context.Context, record Audit) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		details, _ := json.Marshal(record.Details)
		_, _ = s.db.ExecContext(ctx, `INSERT INTO device_control_audits (id, actor, device_id, transport, causation_id, operation_id, lease_id, verb, outcome, created_at, redaction_verified, redaction_opted_out, interactive, evidence_backed, details_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, record.ID, record.Actor, record.DeviceID, record.Transport, record.CausationID, record.OperationID, record.LeaseID, record.Verb, record.Outcome, record.CreatedAt.Format(time.RFC3339Nano), 1, 0, 1, 0, string(details))
	}
	s.audits = append(s.audits, record)
}
