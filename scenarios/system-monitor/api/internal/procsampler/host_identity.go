package procsampler

import (
	"path"
	"strconv"
	"strings"
)

// unifiedCgroupPath returns the cgroup-v2 path from /proc/<pid>/cgroup. On
// hybrid/v1 systems it falls back to the first non-empty controller path.
// Empty input stays empty so missing permissions never become a guessed owner.
func unifiedCgroupPath(contents string) string {
	var first string
	for _, line := range strings.Split(contents, "\n") {
		fields := strings.SplitN(strings.TrimSpace(line), ":", 3)
		if len(fields) != 3 || fields[2] == "" {
			continue
		}
		if fields[1] == "" {
			return fields[2]
		}
		if first == "" {
			first = fields[2]
		}
	}
	return first
}

// hostIdentityOwner returns a stable local host identity only when the sample
// has an explicit systemd identity. These labels intentionally remain distinct
// from scenario/resource owners. Agent scopes are exact cgroup names so their
// sessions can be reconciled with `vrooli agent list`; app scopes and systemd
// units are named for owner reports. A generic user slice is not specific enough
// and remains unknown.
func hostIdentityOwner(sample ProcessSample) string {
	cgroup := strings.Trim(strings.TrimSpace(sample.CgroupPath), "/")
	if cgroup == "" {
		return ""
	}
	parts := strings.Split(cgroup, "/")
	last := parts[len(parts)-1]

	if strings.HasPrefix(last, "vrooli-agent-") && strings.HasSuffix(last, ".scope") {
		return "agent-scope:" + strings.TrimSuffix(last, ".scope")
	}
	if strings.HasPrefix(last, "vrooli-service-") && strings.HasSuffix(last, ".scope") {
		service := strings.TrimSuffix(strings.TrimPrefix(last, "vrooli-service-"), ".scope")
		service = stripVrooliScopeNonce(service)
		if service != "" {
			return "vrooli-service:" + service
		}
	}
	if strings.HasPrefix(last, "app-") && strings.HasSuffix(last, ".scope") {
		appID := strings.TrimSuffix(strings.TrimPrefix(last, "app-"), ".scope")
		if suffix := appScopePID(appID); suffix != "" {
			appID = strings.TrimSuffix(appID, "-"+suffix)
		}
		appName := executableBase(sample)
		if appName == "" || appName == "node" || appName == "electron" {
			appName = strings.ToLower(appID)
			appName = strings.TrimPrefix(appName, "gnome-org.gnome.")
			appName = strings.TrimPrefix(appName, "gnome-")
			appName = strings.TrimPrefix(appName, "com.google.")
		}
		if appName != "" {
			return "desktop-app:" + appName
		}
	}
	if strings.HasSuffix(last, ".service") && !strings.HasPrefix(last, "user@") {
		slice := ""
		for i := len(parts) - 2; i >= 0; i-- {
			if strings.HasSuffix(parts[i], ".slice") {
				slice = parts[i]
				break
			}
		}
		if slice == "" {
			slice = "systemd"
		}
		return "systemd-unit:" + slice + "/" + last
	}
	if strings.HasPrefix(last, "session-") && strings.HasSuffix(last, ".scope") {
		return "systemd-session:" + strings.TrimSuffix(last, ".scope")
	}
	return ""
}

func stripVrooliScopeNonce(name string) string {
	index := strings.LastIndexByte(name, '-')
	if index < 0 || len(name)-index-1 != 16 {
		return name
	}
	for _, r := range name[index+1:] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return name
		}
	}
	return name[:index]
}

func appScopePID(appID string) string {
	// The final component is the PID systemd app scopes append after the app ID.
	// Work from the last hyphen because desktop IDs themselves may contain them.
	index := strings.LastIndexByte(appID, '-')
	if index < 0 {
		return ""
	}
	last := appID[index+1:]
	if _, err := strconv.Atoi(last); err != nil {
		return ""
	}
	return last
}

// executableBase supplies a readable app label only after the cgroup identifies
// that the process belongs to a systemd app scope.
func executableBase(sample ProcessSample) string {
	return strings.ToLower(path.Base(sample.ExecutablePath))
}
