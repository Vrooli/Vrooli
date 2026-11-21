package control

import (
	"fmt"
	"strings"
	"time"

	devicedomain "device-control/internal/devices"
	"device-control/strategy"
)

const (
	currentRouteObservationMaxAge = 15 * time.Minute
	currentRouteObservationGap    = "no current Device Control transport observation"
)

// requireCurrentEffectRoute resolves one transport from Device Control's own
// recent inventory before an operation can reach a strategy. Bridge attachment
// alone is identity evidence, not an actionable transport observation.
func (s *Service) requireCurrentEffectRoute(deviceID, requestedTransport string) (string, error) {
	record, found, err := s.requireCurrentDeviceObservation(deviceID)
	if err != nil {
		return "", err
	}
	if !found {
		if strings.TrimSpace(requestedTransport) != "" && !strings.EqualFold(strings.TrimSpace(requestedTransport), "usb") {
			return "", noCurrentEffectRoute(deviceID)
		}
		return "usb", nil
	}

	transport := strings.TrimSpace(requestedTransport)
	if transport == "" {
		transport = strings.TrimSpace(record.Transport)
		if transport == "" {
			transport = strings.TrimSpace(record.StrategyID)
		}
		if strings.EqualFold(transport, "wireless") && !s.hasTransportStrategy(deviceID) {
			transport = "usb"
		}
	}
	if transport == "" {
		return "", noCurrentEffectRoute(deviceID)
	}

	if len(record.Transports) > 0 {
		profile, ok := findTransportProfile(record.Transports, transport)
		if !ok || !profileAvailable(profile) {
			return "", fmt.Errorf("transport %q has no current Device Control observation for device %q", transport, deviceID)
		}
		observedAt := profile.ObservedAt
		if observedAt.IsZero() {
			observedAt = record.ObservedAt
		}
		if !recentRouteObservation(observedAt, time.Now().UTC()) {
			return "", fmt.Errorf("transport %q observation is stale for device %q", transport, deviceID)
		}
		return firstNonEmpty(profile.Name, profile.StrategyID), nil
	}

	// Older local single-route records have no profile slice. Keep that route
	// usable only when the current Device Control inventory itself observed it
	// recently; Bridge-owned rows must always carry an explicit local profile.
	hostNodeID := strings.TrimSpace(record.HostNodeID)
	if hostNodeID != "" && !(strings.EqualFold(record.Kind, "desktop") && strings.EqualFold(hostNodeID, "local")) {
		return "", noCurrentEffectRoute(deviceID)
	}
	legacyTransport := firstNonEmpty(record.Transport, record.StrategyID)
	if requestedTransport != "" && !strings.EqualFold(requestedTransport, record.Transport) && !strings.EqualFold(requestedTransport, record.StrategyID) {
		return "", fmt.Errorf("transport %q has no current Device Control observation for device %q", requestedTransport, deviceID)
	}
	return legacyTransport, nil
}

func (s *Service) requireCurrentDeviceObservation(deviceID string) (devicedomain.Record, bool, error) {
	record, found := s.devices.Get(deviceID)
	if !found {
		// Non-enumerating singleton strategies (for example, the local desktop
		// adapter in focused tests) are addressed by their own strategy ID. A
		// physical device ID must have a Device Control inventory record.
		if _, registered := s.registry.Get(deviceID); registered {
			return devicedomain.Record{}, false, nil
		}
		return devicedomain.Record{}, false, noCurrentEffectRoute(deviceID)
	}
	if directActuationDeviceUnavailable(record) || strings.Contains(strings.ToLower(record.HealthReason), strings.ToLower(currentRouteObservationGap)) {
		return devicedomain.Record{}, false, noCurrentEffectRoute(deviceID)
	}
	if !recentRouteObservation(record.ObservedAt, time.Now().UTC()) {
		return devicedomain.Record{}, false, fmt.Errorf("device %q Device Control observation is stale", deviceID)
	}
	status := strings.ToLower(strings.TrimSpace(record.Status))
	health := strings.ToLower(strings.TrimSpace(record.Health))
	if status != strategy.StatusAvailable && health != strategy.StatusAvailable {
		return devicedomain.Record{}, false, noCurrentEffectRoute(deviceID)
	}
	return record, true, nil
}

func recentRouteObservation(observedAt, now time.Time) bool {
	if observedAt.IsZero() {
		return false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	age := now.Sub(observedAt)
	return age >= 0 && age <= currentRouteObservationMaxAge
}

func noCurrentEffectRoute(deviceID string) error {
	return fmt.Errorf("device %q has no current actionable transport observation from Device Control", deviceID)
}

func directActuationDeviceUnavailable(record devicedomain.Record) bool {
	for _, status := range []string{record.Status, record.Health} {
		switch strings.ToLower(strings.TrimSpace(status)) {
		case strategy.StatusUnavailable, strategy.HealthUnreachable, "stale":
			return true
		}
	}
	return false
}
