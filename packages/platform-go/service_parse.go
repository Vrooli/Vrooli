package platform

import (
	"encoding/csv"
	"io"
	"strconv"
	"strings"
)

// ParseNativeServiceState interprets captured native service-manager output.
// It is shared by injected tests and platform backends so callers do not
// reproduce localized or manager-specific parsing rules.
func ParseNativeServiceState(target, raw string, commandErr bool) ServiceState {
	switch target {
	case "linux":
		lower := strings.ToLower(strings.TrimSpace(raw))
		if lower == "active" || strings.Contains(lower, "active (running)") {
			return ServiceStateRunning
		}
		if lower == "inactive" || lower == "failed" || strings.Contains(lower, "could not be found") {
			if lower == "failed" {
				return ServiceStateFailed
			}
			return ServiceStateStopped
		}
	case "macos":
		lower := strings.ToLower(raw)
		if strings.Contains(lower, "state = running") || strings.Contains(lower, "state = active") {
			return ServiceStateRunning
		}
		if strings.Contains(lower, "state = exited") || strings.Contains(lower, "state = stopped") || strings.Contains(lower, "could not find service") {
			return ServiceStateStopped
		}
	case "windows":
		for _, line := range strings.Split(raw, "\n") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			fields := strings.Fields(parts[1])
			if len(fields) == 0 {
				continue
			}
			switch fields[0] {
			case "4":
				return ServiceStateRunning
			case "1", "2", "3", "5":
				return ServiceStateStopped
			}
		}
		lower := strings.ToLower(raw)
		if strings.Contains(lower, "running") {
			return ServiceStateRunning
		}
		if strings.Contains(lower, "ready") || strings.Contains(lower, "disabled") {
			return ServiceStateStopped
		}
		reader := csv.NewReader(strings.NewReader(raw))
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil || len(record) < 3 {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(record[2])) {
			case "running":
				return ServiceStateRunning
			case "ready", "disabled":
				return ServiceStateStopped
			}
		}
	}
	if commandErr && strings.Contains(strings.ToLower(raw), "does not exist") {
		return ServiceStateStopped
	}
	return ServiceStateUnknown
}

// ServiceStateExitCode converts the common native exit code representation to
// typed service state when a backend reports one separately from its output.
func ServiceStateExitCode(code int) ServiceState {
	switch strconv.Itoa(code) {
	case "0":
		return ServiceStateRunning
	default:
		return ServiceStateUnknown
	}
}
