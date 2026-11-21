package coverage

import "sort"

// distinctDestinationIDs flattens the target→destinations map once so the
// device-identity reader is called with a small, deduplicated set.
func distinctDestinationIDs(byTarget map[string][]string) []string {
	seen := map[string]struct{}{}
	for _, ids := range byTarget {
		for _, id := range ids {
			if id != "" {
				seen[id] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// classifyProtection turns the successful copies of one target into a
// graduated protection verdict.
//
// Two or more distinct storage devices is protected. One device is a warning,
// not an issue: the copy protects against deletion and corruption, but not
// against losing that device. No successful copy is the only real failure.
func classifyProtection(destinations []DestinationStatus, deviceKeys map[string]string, deviceWarnings map[string][]string) (ProtectionLevel, int, []string) {
	successful := 0
	devices := map[string]struct{}{}
	var warnings []string
	for _, status := range destinations {
		warnings = append(warnings, deviceWarnings[status.DestinationID]...)
		if status.LastSuccessAt.IsZero() {
			continue
		}
		successful++
		if key := deviceKeys[status.DestinationID]; key != "" {
			devices[key] = struct{}{}
		}
	}
	switch {
	case successful == 0:
		return ProtectionUnprotected, 0, append([]string{"no successful backup copy exists"}, warnings...)
	case len(devices) >= 2:
		return ProtectionMultiDevice, len(devices), warnings
	case len(devices) == 1:
		if successful > 1 {
			warnings = append(warnings, "every successful copy is on the same storage device; losing one device loses them all")
		} else {
			warnings = append(warnings, "single copy: losing this storage device loses the only backup")
		}
		return ProtectionSingleCopy, 1, warnings
	default:
		warnings = append(warnings, "a copy exists but its storage device identity is not proven")
		return ProtectionIdentityUnknown, 0, warnings
	}
}
