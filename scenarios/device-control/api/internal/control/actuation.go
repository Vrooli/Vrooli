package control

import (
	"context"
	"fmt"
	"strings"
	"time"

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
		_, _ = s.db.ExecContext(ctx, `INSERT INTO device_control_audits (id, actor, device_id, transport, causation_id, lease_id, verb, outcome, created_at, redaction_verified, redaction_opted_out, interactive, evidence_backed) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, record.ID, record.Actor, record.DeviceID, record.Transport, record.CausationID, record.LeaseID, record.Verb, record.Outcome, record.CreatedAt.Format(time.RFC3339Nano), 1, 0, 1, 0)
	}
	s.audits = append(s.audits, record)
}
