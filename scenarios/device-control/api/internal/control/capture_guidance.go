package control

import (
	"context"
	"fmt"
	"strings"
	"time"

	authdomain "device-control/internal/auth"
	"device-control/strategy"
)

// captureGapNextActions names the exact next commands for a capability gap, so
// an agent never ends on "no available transport provides this capture".
// Order: confirm an already-observed ADB relation, onboard an ADB endpoint on
// the device's own host, then fall back to diagnosis.
func (s *Service) captureGapNextActions(ctx context.Context, device Device, kind string) []string {
	if kind != "screenshot" && kind != "recording" && kind != "logs" {
		return nil
	}
	var actions []string
	// Read retained records only: guidance on a gap must not start an inventory.
	for _, candidate := range s.devices.CorrelationCandidates(time.Now().UTC()) {
		other := ""
		switch device.ID {
		case candidate.LeftID:
			other = candidate.RightID
		case candidate.RightID:
			other = candidate.LeftID
		default:
			continue
		}
		if record, ok := s.devices.Get(other); ok && record.StrategyID == "android-adb" && candidate.Disposition == "unconfirmed" {
			actions = append(actions, fmt.Sprintf("device-control device relation-confirm %s  (same physical device as %s over ADB; then retry)", candidate.ID, other))
			actions = appendUnique(actions, adbOwnerAction(record.Transport, record.Health, record.HealthReason, device.Name, other))
		}
	}
	for _, profile := range device.Transports {
		if profile.StrategyID == "android-adb" {
			actions = appendUnique(actions, adbOwnerAction(profile.Name, profile.Health, profile.HealthReason, device.Name, device.ID))
		}
	}
	if len(actions) > 0 {
		return actions
	}
	if host := deviceHost(device); host != "" && device.StrategyID != "android-adb" && !hasADBTransport(device) {
		actions = append(actions, fmt.Sprintf("device-control device onboard-network --endpoint %s:5555 --attach-to %s  (needs Developer options > Network debugging on the device)", host, device.ID))
	}
	return append(actions, fmt.Sprintf("device-control device describe %s  (lists each transport and why it cannot capture)", device.ID))
}

func hasADBTransport(device Device) bool {
	for _, profile := range device.Transports {
		if profile.StrategyID == "android-adb" {
			return true
		}
	}
	return false
}

// deviceHost returns the first network host any transport profile reports.
func deviceHost(device Device) string {
	endpoints := []string{device.Endpoint}
	for _, profile := range device.Transports {
		endpoints = append(endpoints, profile.Endpoint)
		endpoints = append(endpoints, profile.Endpoints...)
	}
	for _, endpoint := range endpoints {
		if host := strategyHost(endpoint); host != "" {
			return host
		}
	}
	return ""
}

func strategyHost(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" || strings.HasPrefix(endpoint, "emulator-") {
		return ""
	}
	if i := strings.LastIndex(endpoint, ":"); i > 0 && !strings.Contains(endpoint[i+1:], ":") {
		endpoint = endpoint[:i]
	}
	if strings.Count(endpoint, ".") != 3 {
		return ""
	}
	return endpoint
}

// adbOwnerAction names the step that restores an ADB transport that exists but
// cannot serve a request: the owner's physical step when only they can take it,
// otherwise the exact reconnect command. It returns "" for a healthy transport.
func adbOwnerAction(transport, health, reason, deviceName, deviceID string) string {
	switch health {
	case "", strategy.StatusAvailable:
		return ""
	case "unauthorized":
		return fmt.Sprintf("On the %s screen, accept the \"Allow USB debugging\" prompt and choose Always allow, then retry", firstNonEmpty(deviceName, "device"))
	}
	reason = firstNonEmpty(reason, "no reason reported")
	if transport == "usb" {
		return fmt.Sprintf("USB ADB is %s (%s): plug %s in with a data cable, unlock it, and retry", health, reason, firstNonEmpty(deviceName, "the device"))
	}
	return fmt.Sprintf("Wireless ADB is %s (%s): run device-control device reconnect %s", health, reason, deviceID)
}

func appendUnique(items []string, item string) []string {
	if item == "" {
		return items
	}
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}

// captureFailureNextActions turns the typed reason of a failed capture into the
// command that fixes it. unlockHint is the owner's setup command for a locked
// screen (see Service.unlockSetupAction). A sleeping screen needs the wake the
// one-call verbs already do.
func captureFailureNextActions(device Device, chapters []Chapter, unlockHint string) []string {
	for _, chapter := range chapters {
		message := strings.ToLower(chapter.Message)
		switch {
		case strings.Contains(message, "lock state is \"locked\""), strings.Contains(message, "lock=\"locked\""):
			return []string{unlockHint}
		case strings.Contains(message, "screen state is \"off\""), chapter.FailureClass == "app-state":
			return []string{fmt.Sprintf("device-control device screenshot %s  (wakes the screen first)", device.ID)}
		}
	}
	return nil
}

// unlockSetupAction names the exact step that makes a locked device usable
// without hand-unlocking it. The PIN is never passed on a command line: the
// owner pipes it to provision, and the credential authority holds it.
func (s *Service) unlockSetupAction(ctx context.Context, device Device) string {
	name := firstNonEmpty(device.Name, "the device")
	if s.auth == nil {
		return fmt.Sprintf("%s is locked: unlock it by hand", name)
	}
	profile, reason := authdomain.UnlockCandidate(s.auth.List(ctx), device.ID)
	switch reason {
	case authdomain.NoProfile:
		return fmt.Sprintf("%s is locked and has no unlock profile. One-time setup: device-control auth create --device %s --method pin --credential-identity device-control/%s/unlock --credential-field pin, then run: read -rs PIN && printf %%s \"$PIN\" | device-control auth provision <profile-id>. After that every verb unlocks it itself", name, device.ID, device.ID)
	case authdomain.NeedsReprovide:
		return fmt.Sprintf("%s is locked and its saved PIN last failed (%s), so it was not retried. Re-provision it: read -rs PIN && printf %%s \"$PIN\" | device-control auth provision %s", name, profile.LastOutcome, profile.ID)
	}
	return fmt.Sprintf("%s is locked and unlock did not complete (profile %s): device-control device screen %s --operation ready --auth-profile %s", name, profile.ID, device.ID, profile.ID)
}
