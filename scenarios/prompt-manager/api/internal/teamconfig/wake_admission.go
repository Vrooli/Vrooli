package teamconfig

import (
	"fmt"
	"strings"
)

// WakeAdmission controls whether an ordinary heartbeat buys an agent run.
// Legacy configurations omit this block and retain the historical "always"
// behavior. The on-change mode is explicit because proactive members (for
// example contrarian scans) must not silently lose their sampling cadence.
type WakeAdmission struct {
	Mode          string   `json:"mode"`
	ChangeSources []string `json:"changeSources,omitempty"`
}

const (
	WakeAdmissionAlways   = "always"
	WakeAdmissionOnChange = "on-change"
)

func (a *WakeAdmission) Validate() error {
	if a == nil {
		return nil
	}
	mode := strings.TrimSpace(a.Mode)
	if mode == "" {
		mode = WakeAdmissionAlways
	}
	if mode != WakeAdmissionAlways && mode != WakeAdmissionOnChange {
		return fmt.Errorf("wakeAdmission.mode must be %q or %q", WakeAdmissionAlways, WakeAdmissionOnChange)
	}
	allowed := map[string]bool{"team": true, "member": true, "inbox": true, "corpus": true}
	seen := map[string]bool{}
	for _, source := range a.ChangeSources {
		source = strings.TrimSpace(source)
		if !allowed[source] || seen[source] {
			return fmt.Errorf("wakeAdmission.changeSources contains unsupported or duplicate source %q", source)
		}
		seen[source] = true
	}
	if mode == WakeAdmissionOnChange && len(seen) == 0 {
		return fmt.Errorf("wakeAdmission.changeSources is required for on-change mode")
	}
	return nil
}
