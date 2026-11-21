package control

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"device-control/strategy"
)

// ObserveSemantics returns a bounded, ephemeral accessibility snapshot under a
// device lease. The snapshot is not retained as evidence or an action authority.
func (s *Service) ObserveSemantics(ctx context.Context, deviceID, actor, leaseToken, transport string) (strategy.SemanticSnapshot, error) {
	if strings.TrimSpace(leaseToken) == "" {
		if strings.TrimSpace(actor) == "" {
			actor = "cli"
		}
		session, err := s.AcquireContext(ctx, deviceID, actor, time.Minute)
		if err != nil {
			return strategy.SemanticSnapshot{}, err
		}
		leaseToken = session.LeaseToken
		defer func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, _ = s.ReleaseContext(cleanup, session.ID)
		}()
	}
	session, err := s.sessionForAccess(ctx, deviceID, leaseToken, true)
	if err != nil {
		return strategy.SemanticSnapshot{}, err
	}
	probeCtx, cleanup, err := s.registerSessionOperation(ctx, session, deviceID, 15*time.Second)
	if err != nil {
		return strategy.SemanticSnapshot{}, err
	}
	defer cleanup()
	adapter, ok := s.strategyForFlow(deviceID, transport)
	if !ok {
		return strategy.SemanticSnapshot{}, fmt.Errorf("device transport unavailable")
	}
	observer, ok := adapter.(strategy.SemanticObserver)
	if !ok {
		return strategy.SemanticSnapshot{}, &strategy.UnsupportedCapabilityError{Capability: strategy.CapSemanticTree, Operation: "observe"}
	}
	return observer.ObserveSemantics(probeCtx)
}

func (s *Service) ReadDeviceState(ctx context.Context, deviceID string) (strategy.DeviceState, error) {
	return s.readDeviceState(ctx, deviceID, true)
}

func (s *Service) readDeviceState(ctx context.Context, deviceID string, allowReconnect bool) (strategy.DeviceState, error) {
	record, found := s.devices.Get(deviceID)
	if !found {
		return strategy.DeviceState{}, fmt.Errorf("unknown or unavailable device %q", deviceID)
	}
	profiles := append([]strategy.DeviceTransport(nil), record.Transports...)
	if len(profiles) == 0 {
		name := record.Transport
		if name == "" {
			name = "usb"
		}
		profiles = append(profiles, strategy.DeviceTransport{StrategyID: record.StrategyID, Name: name, Endpoint: record.Endpoint, Health: record.Health, HealthReason: record.HealthReason, Properties: append([]strategy.PropertyDescriptor(nil), record.Properties...)})
	}
	combined := strategy.DeviceState{Properties: map[string]strategy.PropertyValue{}, Unavailable: map[string]string{}, Conflicts: map[string]string{}}
	var readErrors []string
	readCount := 0
	propertySources := map[string]strategy.PropertyDescriptor{}
	for _, profile := range profiles {
		transport := profile.Name
		if transport == "" {
			transport = profile.StrategyID
		}
		candidate, ok := s.strategyForFlow(deviceID, transport)
		if !ok {
			readErrors = append(readErrors, fmt.Sprintf("%s: strategy unavailable", transport))
			continue
		}
		reader, ok := candidate.(strategy.StateReader)
		if !ok {
			readErrors = append(readErrors, fmt.Sprintf("%s: state reader unavailable", transport))
			for _, descriptor := range profile.Properties {
				combined.Unavailable[descriptor.Name] = fmt.Sprintf("transport %s does not expose a state reader", transport)
			}
			continue
		}
		state, err := reader.ReadState(ctx)
		if err != nil {
			readErrors = append(readErrors, fmt.Sprintf("%s: %v", transport, err))
			for _, descriptor := range profile.Properties {
				combined.Unavailable[descriptor.Name] = fmt.Sprintf("transport %s failed: %v", transport, err)
			}
			continue
		}
		if len(state.Unavailable) > 0 && !hasObservedDeviceState(state) {
			readErrors = append(readErrors, fmt.Sprintf("%s: all state probes unavailable", transport))
			continue
		}
		readCount++
		if readCount == 1 {
			combined.AutoRotate = state.AutoRotate
			combined.AutoRotateKnown = state.AutoRotateKnown
			combined.BatteryLevel = state.BatteryLevel
			combined.Charging = state.Charging
			combined.ChargingKnown = state.ChargingKnown
			combined.ThermalStatus = state.ThermalStatus
			combined.DisplayWidth = state.DisplayWidth
			combined.DisplayHeight = state.DisplayHeight
			combined.DisplayDensity = state.DisplayDensity
		}
		// A reachable secondary transport may be the only one able to read a
		// field, for example after a promoted phone switches back to USB. Do
		// not let an empty first observation mask that state.
		mergeKnownString := func(name string, destination *string, value string) {
			if value == "" || value == "unknown" || state.Unavailable[name] != "" {
				return
			}
			if *destination == "" || *destination == "unknown" {
				*destination = value
			}
			delete(combined.Unavailable, name)
		}
		mergeKnownString("foreground_package", &combined.ForegroundPackage, state.ForegroundPackage)
		mergeKnownString("screen_state", &combined.ScreenState, state.ScreenState)
		mergeKnownString("lock_state", &combined.LockState, state.LockState)
		mergeKnownString("orientation", &combined.Orientation, state.Orientation)
		if observed := state.WirelessDebugging; observed != nil {
			delete(combined.Unavailable, "wireless_debugging")
			if combined.WirelessDebugging == nil {
				copy := *observed
				copy.Transport = transport
				combined.WirelessDebugging = &copy
			} else {
				current := combined.WirelessDebugging
				mixed := false
				if current.NativeSupported == nil && observed.NativeSupported != nil {
					current.NativeSupported = observed.NativeSupported
					mixed = true
				}
				if current.NativeEnabled == nil && observed.NativeEnabled != nil {
					current.NativeEnabled = observed.NativeEnabled
					mixed = true
				}
				if current.LegacyTCPEnabled == nil && observed.LegacyTCPEnabled != nil {
					current.LegacyTCPEnabled = observed.LegacyTCPEnabled
					current.LegacyTCPPort = observed.LegacyTCPPort
					mixed = true
				}
				if mixed && current.Transport != transport {
					current.Transport = ""
				}
			}
		}
		if state.DreamState != "" && (combined.DreamState == "" || combined.DreamState == "unknown" && state.DreamState != "unknown") {
			combined.DreamState = state.DreamState
			combined.DreamComponent = state.DreamComponent
			if state.DreamState != "unknown" {
				delete(combined.Unavailable, "dream_state")
			}
		}
		for name, value := range state.Properties {
			candidateDescriptor := descriptorFor(profile.Properties, name)
			currentDescriptor, already := propertySources[name]
			if already {
				if reason, conflict := statePropertyConflict(name, combined.Properties[name], value); conflict {
					combined.Conflicts[name] = reason
				}
			}
			if already && currentDescriptor.StateClass == strategy.StateBearing && candidateDescriptor.StateClass != strategy.StateBearing {
				continue
			}
			current := combined.Properties[name]
			replace := !already || current.Status != strategy.StatusAvailable || (value.Status == strategy.StatusAvailable && value.Confidence > current.Confidence)
			if replace {
				value.Transport = transport
				if value.SourceTransport == "" {
					value.SourceTransport = transport
				}
				if value.StateDomain == "" {
					value.StateDomain = stateDomainFor(transport, name)
				}
				if value.ObservedAt.IsZero() {
					value.ObservedAt = time.Now().UTC()
				}
				combined.Properties[name] = value
				propertySources[name] = candidateDescriptor
			}
		}
		for name, reason := range state.Unavailable {
			if name == "wireless_debugging" && combined.WirelessDebugging != nil {
				continue
			}
			if (name == "foreground_package" && combined.ForegroundPackage != "") ||
				(name == "screen_state" && combined.ScreenState != "" && combined.ScreenState != "unknown") ||
				(name == "lock_state" && combined.LockState != "" && combined.LockState != "unknown") ||
				(name == "orientation" && combined.Orientation != "" && combined.Orientation != "unknown") {
				continue
			}
			if name == "dream_state" && combined.DreamState != "" && combined.DreamState != "unknown" {
				continue
			}
			if _, available := combined.Properties[name]; !available {
				combined.Unavailable[name] = reason
			}
		}
	}
	if readCount == 0 {
		// A previously verified wireless endpoint may have disconnected while
		// the network was temporarily unreachable. Re-establish identity once
		// before giving up; this retries observation only, never an action.
		if allowReconnect && s.recoverWirelessBeforeUse(ctx, deviceID) {
			return s.readDeviceState(ctx, deviceID, false)
		}
		return combined, &strategy.AvailabilityError{Reason: fmt.Sprintf("no transport could read state for %q: %s", deviceID, strings.Join(readErrors, "; ")), NextAction: "Use a reachable transport that declares state-bearing properties."}
	}
	if len(combined.Unavailable) == 0 {
		combined.Unavailable = nil
	}
	if len(combined.Conflicts) == 0 {
		combined.Conflicts = nil
	}
	return combined, nil
}

func hasObservedDeviceState(state strategy.DeviceState) bool {
	if state.ForegroundPackage != "" || (state.ScreenState != "" && state.ScreenState != "unknown") ||
		(state.LockState != "" && state.LockState != "unknown") || (state.DreamState != "" && state.DreamState != "unknown") ||
		state.Orientation != "" || state.AutoRotateKnown || state.ChargingKnown || state.BatteryLevel > 0 ||
		state.ThermalStatus != "" || state.DisplayWidth > 0 || state.DisplayHeight > 0 || state.DisplayDensity > 0 ||
		state.WirelessDebugging != nil {
		return true
	}
	for _, property := range state.Properties {
		if property.Status == strategy.StatusAvailable {
			return true
		}
	}
	return false
}

func statePropertyConflict(name string, current, candidate strategy.PropertyValue) (string, bool) {
	if current.Status != strategy.StatusAvailable || candidate.Status != strategy.StatusAvailable {
		return "", false
	}
	if name == "volume" {
		if current.StateDomain != "" && candidate.StateDomain != "" && current.StateDomain != candidate.StateDomain {
			return fmt.Sprintf("volume domains disagree: %s=%s, %s=%s", current.SourceTransport, current.StateDomain, candidate.SourceTransport, candidate.StateDomain), true
		}
		left, leftOK := stateNumber(current.Value)
		right, rightOK := stateNumber(candidate.Value)
		if leftOK && rightOK && math.Abs(left-right) > 0.05 {
			return fmt.Sprintf("volume observations disagree: %s=%.3f, %s=%.3f", current.SourceTransport, left, candidate.SourceTransport, right), true
		}
	}
	if name == "muted" {
		left, leftOK := current.Value.(bool)
		right, rightOK := candidate.Value.(bool)
		if leftOK && rightOK && left != right {
			return fmt.Sprintf("mute observations disagree: %s=%t, %s=%t", current.SourceTransport, left, candidate.SourceTransport, right), true
		}
	}
	return "", false
}

func stateNumber(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case uint64:
		return float64(number), true
	default:
		return 0, false
	}
}

func stateDomainFor(transport, property string) string {
	transport = strings.ToLower(strings.TrimSpace(transport))
	property = strings.ToLower(strings.TrimSpace(property))
	if transport == "google-cast" || strings.Contains(transport, "cast") {
		switch property {
		case "volume":
			return "receiver_volume"
		case "muted":
			return "receiver_mute"
		default:
			return "receiver_media"
		}
	}
	return "unknown"
}

func descriptorFor(descriptors []strategy.PropertyDescriptor, name string) strategy.PropertyDescriptor {
	for _, descriptor := range descriptors {
		if descriptor.Name == name {
			return descriptor
		}
	}
	return strategy.PropertyDescriptor{Name: name, StateClass: strategy.StateBearing}
}
