package control

import (
	"strings"

	devicedomain "device-control/internal/devices"
	identitydomain "device-control/internal/identity"
	"device-control/strategy"
)

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

// hasTransportStrategy reports whether an endpoint-bound transport strategy is
// registered (restoring persisted state first) for the device. It is used to
// decide whether an unselected transport can default to wireless.
func (s *Service) hasTransportStrategy(deviceID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.restoreTransportStrategies()
	_, ok := s.transportStrategies[deviceID]
	return ok
}

// strategyForFlow resolves a requested transport profile against the durable
// identity. Promotion keeps the legacy wireless endpoint-bound strategy
// separate, while ordinary composed identities can select any named profile.
func (s *Service) strategyForFlow(deviceID, requestedTransport string) (strategy.Strategy, bool) {
	requestedTransport = strings.ToLower(strings.TrimSpace(requestedTransport))
	record, ok := s.devices.Get(deviceID)
	if !ok {
		if requestedTransport == "" || requestedTransport == "usb" {
			return s.strategyForDevice(deviceID)
		}
		return nil, false
	}
	if requestedTransport == "" {
		// Default an unselected transport to the device's current transport. A
		// composed identity's selected REST/mDNS transport is the natural default
		// for a modality-specific device. A wireless device resolves to its
		// endpoint-bound transport when one is registered — this covers both a
		// USB-promoted phone and a directly-onboarded network TV/box, the latter
		// of which has no USB base to fall back to. USB remains the default only
		// when no wireless strategy is available.
		requestedTransport = strings.ToLower(strings.TrimSpace(record.Transport))
		if requestedTransport == "" {
			requestedTransport = "usb"
		}
		if requestedTransport == "wireless" && !s.hasTransportStrategy(deviceID) {
			requestedTransport = "usb"
		}
	}
	if requestedTransport == "wireless" && record.Transport == "wireless" {
		s.mu.Lock()
		s.restoreTransportStrategies()
		deferred, promoted := s.transportStrategies[deviceID]
		s.mu.Unlock()
		return deferred, promoted
	}
	for _, profile := range record.Transports {
		name := strings.ToLower(strings.TrimSpace(profile.Name))
		strategyID := strings.ToLower(strings.TrimSpace(profile.StrategyID))
		if requestedTransport != name && requestedTransport != strategyID {
			continue
		}
		base, found := s.registry.Get(profile.StrategyID)
		if !found {
			return nil, false
		}
		// Preserve both pieces of routing context.  A composed identity may
		// have a stale mDNS endpoint after restart, while the serial remains
		// the durable certificate lookup key.  DeviceScoped must be applied
		// before endpoint scoping so ForEndpoint does not erase that serial.
		serial := record.Serial
		if strings.EqualFold(profile.StrategyID, "android-adb") {
			// A composed identity (for example a Cast TV that also exposes ADB over
			// the network) keeps its own serial; the ADB profile is addressed by the
			// hardware serial it was confirmed under and its network endpoint.
			serial = firstNonEmpty(adbSerialClaim(record), record.Serial)
		}
		if scoped, scopedOK := base.(strategy.DeviceScoped); scopedOK && serial != "" {
			base = scoped.ForDevice(serial)
		}
		if restorer, restorable := base.(interface {
			RestoreWireless(string) strategy.Strategy
		}); restorable && profile.Endpoint != "" && strings.EqualFold(profile.Name, "wireless") {
			return restorer.RestoreWireless(profile.Endpoint), true
		}
		if endpointScoped, endpointOK := base.(interface {
			ForEndpoint(string) strategy.Strategy
		}); endpointOK && profile.Endpoint != "" {
			base = endpointScoped.ForEndpoint(profile.Endpoint)
		}
		return base, true
	}
	if requestedTransport != "usb" && requestedTransport != strings.ToLower(strings.TrimSpace(record.Transport)) && requestedTransport != strings.ToLower(strings.TrimSpace(record.StrategyID)) {
		return nil, false
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

// adbSerialClaim returns the hardware serial an identity carries as an
// adb-serial claim, or "" when it has none.
func adbSerialClaim(record devicedomain.Record) string {
	for _, claim := range record.Claims {
		if claim.Kind == identitydomain.ADBSerial {
			return claim.Value
		}
	}
	return ""
}
