package control

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	devicedomain "device-control/internal/devices"
	"device-control/strategy"

	"github.com/google/uuid"
)

var ErrVolumeSessionLost = errors.New("volume operation session was lost before actuation")

type VolumeRequest struct {
	Device             string   `json:"device"`
	Actor              string   `json:"actor"`
	Goal               string   `json:"goal,omitempty"`
	Operation          string   `json:"operation,omitempty"`
	Value              *float64 `json:"value,omitempty"`
	Direction          string   `json:"direction,omitempty"`
	VerificationPolicy string   `json:"verification_policy,omitempty"`
	OperationID        string   `json:"operation_id,omitempty"`
}

type VolumeIntent struct {
	Kind       string  `json:"kind"`
	ChangeKind string  `json:"change_kind"`
	Value      float64 `json:"value,omitempty"`
	Direction  string  `json:"direction,omitempty"`
	Repeat     int     `json:"repeat,omitempty"`
	Target     string  `json:"target_domain"`
}

type VolumePlan struct {
	Intent           VolumeIntent `json:"intent"`
	StateTransport   string       `json:"state_transport,omitempty"`
	ActionTransport  string       `json:"action_transport"`
	Action           string       `json:"action"`
	AbsoluteSetpoint *float64     `json:"absolute_setpoint,omitempty"`
	Repeat           int          `json:"repeat,omitempty"`
	MathBasis        string       `json:"math_basis,omitempty"`
	Approximation    string       `json:"approximation,omitempty"`
	StateDomain      string       `json:"state_domain,omitempty"`
	StateAvailable   bool         `json:"state_available"`
	PhysicalMapping  string       `json:"physical_mapping,omitempty"`
}

type VolumeResult struct {
	Status            string               `json:"status"`
	OperationID       string               `json:"operation_id"`
	DeviceID          string               `json:"device_id,omitempty"`
	DeviceName        string               `json:"device_name,omitempty"`
	Plan              VolumePlan           `json:"plan"`
	Before            strategy.DeviceState `json:"before"`
	After             strategy.DeviceState `json:"after"`
	VerificationClass string               `json:"verification_class"`
	Evidence          []string             `json:"evidence"`
	RecoveryAttempts  int                  `json:"recovery_attempts"`
	NextAction        string               `json:"next_action,omitempty"`
	Audit             Audit                `json:"audit,omitempty"`
}

type VolumeError struct {
	Class      string `json:"class"`
	Message    string `json:"message"`
	NextAction string `json:"next_action,omitempty"`
}

func (e *VolumeError) Error() string { return e.Class + ": " + e.Message }

var percentPattern = regexp.MustCompile(`(?i)([-+]?\d+(?:\.\d+)?)\s*(?:%|percent(?:age)?)`)
var numberPattern = regexp.MustCompile(`(?i)(?:^|\s)([-+]?\d+(?:\.\d+)?)(?:\s|$)`)

// ParseVolumeIntent turns natural language into a typed operation. Percent
// signs are never silently treated as normalized absolute values: "by 50%"
// is a fraction of current, while "set to 50%" is an absolute setpoint.
func ParseVolumeIntent(request VolumeRequest) (VolumeIntent, error) {
	target := strings.TrimSpace(request.VerificationPolicy)
	if target == "" {
		target = "physical_output"
	}
	if target != "physical_output" && target != "receiver_volume" {
		return VolumeIntent{}, &VolumeError{Class: "invalid_input", Message: "verification_policy must be physical_output or receiver_volume"}
	}
	if strings.TrimSpace(request.Operation) != "" {
		return parseTypedVolumeIntent(request, target)
	}
	goal := strings.ToLower(strings.TrimSpace(request.Goal))
	if goal == "" {
		return VolumeIntent{}, &VolumeError{Class: "invalid_input", Message: "goal or operation is required"}
	}
	if strings.Contains(goal, "mute") {
		return VolumeIntent{Kind: "mute", ChangeKind: "mute", Direction: "mute", Target: target}, nil
	}
	if value, found := percentage(goal); found {
		if value < 0 || value > 100 {
			return VolumeIntent{}, &VolumeError{Class: "invalid_input", Message: "percentage must be between 0 and 100"}
		}
		fraction := value / 100
		if strings.Contains(goal, "set") || strings.Contains(goal, "to ") {
			return VolumeIntent{Kind: "absolute", ChangeKind: "absolute_normalized", Value: fraction, Target: target}, nil
		}
		if strings.Contains(goal, "by") || strings.Contains(goal, "lower") || strings.Contains(goal, "reduce") || strings.Contains(goal, "down") || strings.Contains(goal, "decrease") {
			return VolumeIntent{Kind: "relative", ChangeKind: "fraction_of_current", Value: -fraction, Direction: "down", Repeat: boundedRelativeRepeat(fraction), Target: target}, nil
		}
		return VolumeIntent{}, &VolumeError{Class: "invalid_input", Message: "percentage is ambiguous; use 'by 50%' or 'set to 50%'"}
	}
	if strings.Contains(goal, "up") || strings.Contains(goal, "down") || strings.Contains(goal, "louder") || strings.Contains(goal, "quieter") {
		direction := "down"
		if strings.Contains(goal, "up") || strings.Contains(goal, "louder") {
			direction = "up"
		}
		repeat := 1
		if value, found := volumeNumber(goal); found && value >= 1 && value <= 10 {
			repeat = int(value)
		}
		return VolumeIntent{Kind: "directional", ChangeKind: "direction", Direction: direction, Repeat: repeat, Target: target}, nil
	}
	if value, found := volumeNumber(goal); found {
		if value < 0 || value > 1 {
			return VolumeIntent{}, &VolumeError{Class: "invalid_input", Message: "absolute normalized volume must be between 0 and 1"}
		}
		return VolumeIntent{Kind: "absolute", ChangeKind: "absolute_normalized", Value: value, Target: target}, nil
	}
	return VolumeIntent{}, &VolumeError{Class: "invalid_input", Message: "volume goal is not understood"}
}

func parseTypedVolumeIntent(request VolumeRequest, target string) (VolumeIntent, error) {
	kind := strings.ToLower(strings.TrimSpace(request.Operation))
	switch kind {
	case "mute":
		return VolumeIntent{Kind: "mute", ChangeKind: "mute", Direction: "mute", Target: target}, nil
	case "relative", "relative_reduction", "relative_increase":
		if request.Value == nil || *request.Value < 0 || *request.Value > 1 {
			return VolumeIntent{}, &VolumeError{Class: "invalid_input", Message: "relative value must be between 0 and 1"}
		}
		direction := strings.ToLower(strings.TrimSpace(request.Direction))
		if direction == "" {
			direction = "down"
		}
		if direction != "up" && direction != "down" {
			return VolumeIntent{}, &VolumeError{Class: "invalid_input", Message: "relative direction must be up or down"}
		}
		value := *request.Value
		if direction == "down" {
			value = -value
		}
		return VolumeIntent{Kind: "relative", ChangeKind: "fraction_of_current", Value: value, Direction: direction, Repeat: boundedRelativeRepeat(math.Abs(value)), Target: target}, nil
	case "absolute", "set":
		if request.Value == nil || *request.Value < 0 || *request.Value > 1 {
			return VolumeIntent{}, &VolumeError{Class: "invalid_input", Message: "absolute value must be between 0 and 1"}
		}
		return VolumeIntent{Kind: "absolute", ChangeKind: "absolute_normalized", Value: *request.Value, Target: target}, nil
	case "directional", "direction":
		direction := strings.ToLower(strings.TrimSpace(request.Direction))
		if direction != "up" && direction != "down" {
			return VolumeIntent{}, &VolumeError{Class: "invalid_input", Message: "direction must be up or down"}
		}
		repeat := 1
		if request.Value != nil {
			repeat = int(*request.Value)
		}
		if repeat < 1 || repeat > 10 {
			return VolumeIntent{}, &VolumeError{Class: "invalid_input", Message: "direction repeat must be between 1 and 10"}
		}
		return VolumeIntent{Kind: "directional", ChangeKind: "direction", Direction: direction, Repeat: repeat, Target: target}, nil
	default:
		return VolumeIntent{}, &VolumeError{Class: "invalid_input", Message: "unsupported volume operation " + kind}
	}
}

func percentage(text string) (float64, bool) {
	match := percentPattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return 0, false
	}
	value, err := strconv.ParseFloat(match[1], 64)
	return value, err == nil
}

func volumeNumber(text string) (float64, bool) {
	match := numberPattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return 0, false
	}
	value, err := strconv.ParseFloat(match[1], 64)
	return value, err == nil
}

func boundedRelativeRepeat(fraction float64) int {
	repeat := int(math.Ceil(fraction * 10))
	if repeat < 1 {
		return 1
	}
	if repeat > 10 {
		return 10
	}
	return repeat
}

func PlanVolume(record devicedomain.Record, state strategy.DeviceState, intent VolumeIntent) (VolumePlan, error) {
	plan := VolumePlan{Intent: intent, StateDomain: "unknown"}
	if value, ok := state.Properties["volume"]; ok && value.Status == strategy.StatusAvailable {
		plan.StateTransport = firstNonEmpty(value.SourceTransport, value.Transport)
		plan.StateDomain = firstNonEmpty(value.StateDomain, "unknown")
		plan.StateAvailable = true
	}
	if plan.StateDomain == "unknown" && plan.StateTransport != "" {
		plan.StateDomain = stateDomainFor(plan.StateTransport, "volume")
	}
	propertyTransport := findVolumePropertyTransport(record.Transports, "volume")
	mutePropertyTransport := findVolumePropertyTransport(record.Transports, "muted")
	relativeTransport := findRelativeVolumeTransport(record.Transports, intent.Direction)
	muteOperationTransport := findOperationTransport(record.Transports, "mute")
	switch intent.Kind {
	case "absolute":
		if propertyTransport == "" {
			return VolumePlan{}, &VolumeError{Class: "unsupported_operation", Message: "no available transport declares writable volume", NextAction: "Use a transport that declares the volume-set operation or request a relative change"}
		}
		if intent.Target == "physical_output" && !physicalOutputProperty(record.Transports, propertyTransport, "volume", plan.StateDomain) {
			return VolumePlan{}, &VolumeError{Class: "unsupported_operation", Message: "available volume is not declared as physical output", NextAction: "Use a relative physical-output operation through a transport that declares volume-up/volume-down, or request receiver_volume explicitly"}
		}
		plan.ActionTransport, plan.Action, plan.AbsoluteSetpoint = propertyTransport, "property:volume", floatPtr(intent.Value)
		if plan.StateTransport == "" {
			plan.StateTransport = propertyTransport
		}
	case "mute":
		if mutePropertyTransport != "" {
			if intent.Target == "physical_output" && !physicalOutputProperty(record.Transports, mutePropertyTransport, "muted", plan.StateDomain) {
				return VolumePlan{}, &VolumeError{Class: "unsupported_operation", Message: "available mute state is not declared as physical output", NextAction: "Use a transport that declares mute actuation for physical output, or request receiver_volume explicitly"}
			}
			plan.ActionTransport = mutePropertyTransport
			plan.Action = "property:muted"
		} else if muteOperationTransport != "" {
			plan.ActionTransport = muteOperationTransport
			plan.Action = "mute"
		}
		if plan.ActionTransport == "" {
			return VolumePlan{}, &VolumeError{Class: "unsupported_operation", Message: "no available transport declares mute support"}
		}
	case "relative", "directional":
		// Android TV Remote's directional keys are the right fallback for a
		// plain TV-speaker route, but they are not a reliable actuator when the
		// live RemoteSetVolumeLevel witness identifies an ARC/eARC amplifier.
		// Prefer a writable physical-output setter in that case. This keeps the
		// semantic operation stable while allowing the transport-specific route
		// to change with the observed audio topology.
		if externalOutputVolume(state) && propertyTransport != "" && physicalOutputProperty(record.Transports, propertyTransport, "volume", plan.StateDomain) {
			current, ok := asNumber(state.Properties["volume"].Value)
			if !ok {
				return VolumePlan{}, &VolumeError{Class: "unavailable_state", Message: "state-bearing transport returned a non-numeric volume"}
			}
			desired := clamp01(current * (1 + intent.Value))
			plan.ActionTransport, plan.Action, plan.AbsoluteSetpoint = propertyTransport, "property:volume", floatPtr(desired)
			plan.MathBasis = fmt.Sprintf("%0.3f * (1 + %0.3f) = %0.3f", current, intent.Value, desired)
			plan.Approximation = "ARC/eARC output route selected a writable master-volume setter instead of directional remote keys"
		} else if relativeTransport != "" {
			plan.ActionTransport = relativeTransport
			plan.Action = "volume-" + intent.Direction
			plan.Repeat = intent.Repeat
			if intent.Kind == "relative" {
				plan.MathBasis = fmt.Sprintf("ceil(abs(%0.3f) * 10) bounded to 10 discrete remote keys", intent.Value)
				plan.Approximation = "relative percentage is approximated by bounded Android TV Remote key presses"
			}
		} else if propertyTransport != "" && plan.StateAvailable && (intent.Target != "physical_output" || physicalOutputProperty(record.Transports, propertyTransport, "volume", plan.StateDomain)) {
			current, ok := asNumber(state.Properties["volume"].Value)
			if !ok {
				return VolumePlan{}, &VolumeError{Class: "unavailable_state", Message: "state-bearing transport returned a non-numeric volume"}
			}
			desired := clamp01(current * (1 + intent.Value))
			plan.ActionTransport, plan.Action, plan.AbsoluteSetpoint = propertyTransport, "property:volume", floatPtr(desired)
			plan.MathBasis = fmt.Sprintf("%0.3f * (1 + %0.3f) = %0.3f", current, intent.Value, desired)
		} else {
			return VolumePlan{}, &VolumeError{Class: "unsupported_operation", Message: "no volume action transport is available"}
		}
	}
	if plan.ActionTransport == "" {
		return VolumePlan{}, &VolumeError{Class: "unsupported_operation", Message: "no compatible volume transport is available"}
	}
	if plan.StateDomain == "receiver_volume" {
		plan.PhysicalMapping = "The selected state transport reports receiver volume; physical output mapping is not established"
	}
	return plan, nil
}

func externalOutputVolume(state strategy.DeviceState) bool {
	property, ok := state.Properties["volume"]
	if !ok || property.Status != strategy.StatusAvailable {
		return false
	}
	return strings.Contains(strings.ToLower(property.Reason), "arc-earc-amplifier")
}

func physicalOutputProperty(profiles []strategy.DeviceTransport, transport, property, observedDomain string) bool {
	for _, profile := range profiles {
		if firstNonEmpty(profile.Name, profile.StrategyID) != transport {
			continue
		}
		for _, descriptor := range profile.Properties {
			if strings.EqualFold(strings.TrimSpace(descriptor.Name), property) {
				domain := firstNonEmpty(descriptor.StateDomain, observedDomain)
				return domain == "physical_output"
			}
		}
	}
	return observedDomain == "physical_output"
}

func findVolumePropertyTransport(profiles []strategy.DeviceTransport, property string) string {
	for _, profile := range profiles {
		if !profileAvailable(profile) || !profileHasWritableProperty(profile, property) {
			continue
		}
		return firstNonEmpty(profile.Name, profile.StrategyID)
	}
	return ""
}

func findRelativeVolumeTransport(profiles []strategy.DeviceTransport, direction string) string {
	operation := "volume-" + strings.ToLower(strings.TrimSpace(direction))
	return findOperationTransport(profiles, operation)
}

func findOperationTransport(profiles []strategy.DeviceTransport, operation string) string {
	for _, profile := range profiles {
		if !profileAvailable(profile) || !profileSupportsOperation(profile, operation) {
			continue
		}
		return firstNonEmpty(profile.Name, profile.StrategyID)
	}
	return ""
}

func profileAvailable(profile strategy.DeviceTransport) bool {
	health := strings.ToLower(strings.TrimSpace(profile.Health))
	return health != strategy.HealthUnreachable && health != "stale" && health != strategy.StatusUnavailable
}

func profileHasWritableProperty(profile strategy.DeviceTransport, name string) bool {
	for _, descriptor := range profile.Properties {
		if strings.EqualFold(strings.TrimSpace(descriptor.Name), name) && descriptor.Writable {
			return true
		}
	}
	return false
}

func profileSupportsOperation(profile strategy.DeviceTransport, operation string) bool {
	for _, declared := range profile.Operations {
		if strings.EqualFold(strings.TrimSpace(declared), operation) {
			return true
		}
	}
	// Compatibility for profiles persisted before semantic operations were
	// added: the role is already a transport-level declaration, not a device
	// name or model match.
	return (operation == "mute" || operation == "volume-up" || operation == "volume-down") && strings.EqualFold(profile.Role, "relative-key-actuation")
}

func (s *Service) ExecuteVolume(ctx context.Context, request VolumeRequest) (VolumeResult, error) {
	intent, err := ParseVolumeIntent(request)
	if err != nil {
		return VolumeResult{Status: "failed", VerificationClass: "unavailable"}, err
	}
	device, err := s.resolveVolumeDevice(ctx, request.Device)
	if err != nil {
		return VolumeResult{Status: "failed", VerificationClass: "unavailable"}, err
	}
	before, beforeErr := s.ReadDeviceState(ctx, device.ID)
	plan, err := PlanVolume(deviceRecord(device), before, intent)
	if err != nil {
		result := VolumeResult{Status: "failed", OperationID: strings.TrimSpace(request.OperationID), DeviceID: device.ID, DeviceName: device.Name, Before: before, VerificationClass: "unavailable"}
		if typed, ok := err.(*VolumeError); ok {
			result.NextAction = typed.NextAction
		}
		if result.OperationID == "" {
			result.OperationID = uuid.NewString()
		}
		return result, err
	}
	opID := strings.TrimSpace(request.OperationID)
	if opID == "" {
		opID = uuid.NewString()
	}
	lease, err := s.AcquireContext(ctx, device.ID, firstNonEmpty(request.Actor, "semantic-volume"), 2*time.Minute)
	if err != nil {
		return VolumeResult{Status: "failed", OperationID: opID, DeviceID: device.ID, DeviceName: device.Name, Plan: plan, Before: before, VerificationClass: "unavailable", NextAction: "Retry after the device lease becomes available"}, err
	}
	recoveryAttempts := 0
	causationID := opID
	var actuationErr error
	for attempt := 0; attempt < 2; attempt++ {
		actuationErr = s.executeVolumeAction(ctx, device.ID, lease.LeaseToken, plan, causationID)
		if actuationErr == nil {
			break
		}
		if attempt == 0 && recoverableVolumeSessionError(actuationErr) {
			recoveryAttempts = 1
			_, _ = s.ReleaseContext(ctx, lease.ID)
			lease, actuationErr = s.AcquireContext(ctx, device.ID, firstNonEmpty(request.Actor, "semantic-volume"), 2*time.Minute)
			if actuationErr != nil {
				break
			}
			continue
		}
		break
	}
	_, _ = s.ReleaseContext(ctx, lease.ID)
	result := VolumeResult{Status: "failed", OperationID: opID, DeviceID: device.ID, DeviceName: device.Name, Plan: plan, Before: before, RecoveryAttempts: recoveryAttempts, Evidence: []string{"operation:" + opID}}
	if beforeErr == nil {
		result.Evidence = append(result.Evidence, "state-before:"+opID)
	}
	if actuationErr != nil {
		result.VerificationClass = "unavailable"
		result.NextAction = "Reconnect or pair the selected volume transport, then retry the same operation"
		audit := Audit{ID: uuid.NewString(), Actor: firstNonEmpty(request.Actor, "semantic-volume"), DeviceID: device.ID, Transport: plan.ActionTransport, CausationID: causationID, OperationID: opID, LeaseID: lease.ID, Verb: "semantic-volume", Outcome: "unavailable", CreatedAt: time.Now().UTC(), RedactionVerified: true, Interactive: true, EvidenceBacked: false}
		s.persistDirectAudit(ctx, audit)
		result.Audit = audit
		return result, actuationErr
	}
	after, afterErr := s.ReadDeviceState(ctx, device.ID)
	result.After = after
	if afterErr == nil {
		result.Evidence = append(result.Evidence, "state-after:"+opID)
	}
	result.VerificationClass = volumeVerification(intent, plan, before, after, afterErr)
	result.Status = "ok"
	if result.VerificationClass == "unavailable" {
		result.NextAction = "Use a state-bearing transport or inspect the device directly; command delivery alone is not verification"
	}
	audit := Audit{ID: uuid.NewString(), Actor: firstNonEmpty(request.Actor, "semantic-volume"), DeviceID: device.ID, Transport: plan.ActionTransport, CausationID: causationID, OperationID: opID, LeaseID: lease.ID, Verb: "semantic-volume", Outcome: result.VerificationClass, CreatedAt: time.Now().UTC(), RedactionVerified: true, Interactive: true, EvidenceBacked: result.VerificationClass == "verified"}
	s.persistDirectAudit(ctx, audit)
	result.Audit = audit
	return result, nil
}

func (s *Service) executeVolumeAction(ctx context.Context, deviceID, token string, plan VolumePlan, causationID string) error {
	if err := s.ValidateLease(ctx, deviceID, token); err != nil {
		return fmt.Errorf("%w: %v", ErrVolumeSessionLost, err)
	}
	adapter, ok := s.strategyForFlow(deviceID, plan.ActionTransport)
	if !ok {
		return &VolumeError{Class: "transport_unavailable", Message: "selected volume transport is unavailable"}
	}
	if strings.HasPrefix(plan.Action, "property:") {
		property, ok := adapter.(strategy.PropertyActuator)
		if !ok {
			return &VolumeError{Class: "unsupported_operation", Message: "selected transport does not support property actuation"}
		}
		name := strings.TrimPrefix(plan.Action, "property:")
		value := any(float64(0))
		if plan.AbsoluteSetpoint != nil {
			value = *plan.AbsoluteSetpoint
		}
		if name == "muted" {
			value = true
		}
		return property.SetProperty(ctx, strategy.PropertySet{Name: name, Value: value, CausationID: causationID})
	}
	media, ok := adapter.(strategy.MediaController)
	if !ok {
		return &VolumeError{Class: "unsupported_operation", Message: "selected transport does not support relative volume keys"}
	}
	for i := 0; i < maxInt(plan.Repeat, 1); i++ {
		if err := media.ControlMedia(ctx, strategy.MediaCommand{Action: plan.Action, CausationID: causationID}); err != nil {
			return err
		}
	}
	return nil
}

func recoverableVolumeSessionError(err error) bool {
	if errors.Is(err, ErrVolumeSessionLost) {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "session") || strings.Contains(text, "lease") || strings.Contains(text, "not connected")
}

func volumeVerification(intent VolumeIntent, plan VolumePlan, before, after strategy.DeviceState, afterErr error) string {
	if afterErr != nil {
		return "unverified"
	}
	value, present := after.Properties["volume"]
	if intent.Kind == "mute" {
		if muted, ok := after.Properties["muted"]; ok && muted.Status == strategy.StatusAvailable {
			if got, ok := muted.Value.(bool); ok && got {
				if intent.Target == "receiver_volume" {
					return "verified"
				}
				return "conflicted"
			}
		}
		return "unverified"
	}
	if intent.Kind == "relative" || intent.Kind == "directional" {
		beforeValue, beforePresent := before.Properties["volume"]
		afterValue, afterPresent := after.Properties["volume"]
		if !beforePresent || beforeValue.Status != strategy.StatusAvailable || !afterPresent || afterValue.Status != strategy.StatusAvailable {
			return "unverified"
		}
		beforeNumber, beforeOK := asNumber(beforeValue.Value)
		afterNumber, afterOK := asNumber(afterValue.Value)
		if !beforeOK || !afterOK || beforeValue.StateDomain == "" || afterValue.StateDomain == "" {
			return "unverified"
		}
		changedInRequestedDirection := afterNumber < beforeNumber-0.005
		if intent.Direction == "up" {
			changedInRequestedDirection = afterNumber > beforeNumber+0.005
		}
		if !changedInRequestedDirection {
			return "conflicted"
		}
		if intent.Target == "receiver_volume" || afterValue.StateDomain == "physical_output" {
			return "verified"
		}
		return "conflicted"
	}
	if !present || value.Status != strategy.StatusAvailable {
		return "unverified"
	}
	got, ok := asNumber(value.Value)
	if !ok || plan.AbsoluteSetpoint == nil {
		return "unverified"
	}
	if math.Abs(got-*plan.AbsoluteSetpoint) > 0.03 {
		return "conflicted"
	}
	if intent.Target == "receiver_volume" || value.StateDomain == "physical_output" {
		return "verified"
	}
	return "conflicted"
}

func (s *Service) resolveVolumeDevice(ctx context.Context, selector string) (Device, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return Device{}, &VolumeError{Class: "invalid_input", Message: "device is required"}
	}
	if record, found := s.devices.Get(selector); found {
		// Transport profiles restored from durable state are intentionally
		// marked unreachable until inventory probes them. Refresh only when the
		// cached record cannot yet route a volume operation; ordinary requests
		// stay on the fast cached path.
		if !recordHasAvailableVolumeRoute(record) {
			_ = s.Devices(ctx)
			if refreshed, refreshedOK := s.devices.Get(record.ID); refreshedOK {
				record = refreshed
			}
		}
		return deviceFromRecord(record), nil
	}
	for _, record := range s.devices.ListActionable(time.Now().UTC(), 15*time.Minute) {
		if strings.EqualFold(strings.TrimSpace(record.Name), selector) {
			return deviceFromRecord(record), nil
		}
	}
	for _, device := range s.Devices(ctx) {
		if device.ID == selector {
			return device, nil
		}
	}
	var matches []Device
	for _, device := range s.Devices(ctx) {
		if strings.EqualFold(strings.TrimSpace(device.Name), selector) {
			matches = append(matches, device)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return Device{}, &VolumeError{Class: "device_selection_required", Message: "device name matches multiple logical devices"}
	}
	return Device{}, &VolumeError{Class: "device_unavailable", Message: "device was not found in actionable inventory", NextAction: "Run device discovery and inspect diagnostics"}
}

func recordHasAvailableVolumeRoute(record devicedomain.Record) bool {
	for _, profile := range record.Transports {
		if !profileAvailable(profile) {
			continue
		}
		if profileHasWritableProperty(profile, "volume") || profileSupportsOperation(profile, "volume-up") || profileSupportsOperation(profile, "volume-down") {
			return true
		}
	}
	return false
}

func deviceRecord(device Device) devicedomain.Record {
	return devicedomain.Record{ID: device.ID, Name: device.Name, Kind: device.Kind, StrategyID: device.StrategyID, Endpoint: device.Endpoint, Status: device.Status, Health: device.Health, Transports: device.Transports, Properties: device.Properties}
}

func asNumber(value any) (float64, bool) {
	switch value := value.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	default:
		return 0, false
	}
}

func clamp01(value float64) float64 {
	return math.Max(0, math.Min(1, value))
}

func floatPtr(value float64) *float64 { return &value }
func maxInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
