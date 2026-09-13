package teamconfig

import (
	"fmt"
	"strings"
)

// Supervision selects owner-backed standing admission for one heartbeat member.
// Effort identities and discovery roots belong to Agent Manager.
type Supervision struct {
	DispatchAuthorization        *SupervisorDispatchBinding `json:"dispatchAuthorization,omitempty"`
	DiagnosticAllowance          DiagnosticAllowance        `json:"diagnosticAllowance"`
	DiscoveryLimit               int                        `json:"discoveryLimit"`
	MaxEffortsPerWake            int                        `json:"maxEffortsPerWake"`
	MinWakeIntervalSeconds       int                        `json:"minWakeIntervalSeconds"`
	HealthySampleIntervalSeconds int                        `json:"healthySampleIntervalSeconds,omitempty"`
	MaxHealthySamplesPerWake     int                        `json:"maxHealthySamplesPerWake,omitempty"`
}

// Selection metadata only; AM checks the grant and the canonical credential.
type SupervisorDispatchBinding struct {
	EffortRef       string `json:"effortRef"`
	AuthorizationID string `json:"authorizationId"`
}

// DiagnosticAllowance measures inference attempts, not tokens or dollars.
// AM's qualified execution profile continues to own each run's resource limits.
type DiagnosticAllowance struct {
	MaxWakesPerWindow int    `json:"maxWakesPerWindow"`
	WindowSeconds     int    `json:"windowSeconds"`
	AccountingRef     string `json:"accountingRef"`
}

func (s *Supervision) Validate() error {
	if s == nil {
		return nil
	}
	if d := s.DispatchAuthorization; d != nil && (strings.TrimSpace(d.EffortRef) == "" || len(d.EffortRef) > 512 || strings.TrimSpace(d.AuthorizationID) == "" || len(d.AuthorizationID) > 128) {
		return fmt.Errorf("supervision.dispatchAuthorization requires exact effortRef and authorizationId")
	}
	a := s.DiagnosticAllowance
	if a.MaxWakesPerWindow < 1 || a.MaxWakesPerWindow > 100 || a.WindowSeconds < 60 || a.WindowSeconds > 2592000 || strings.TrimSpace(a.AccountingRef) == "" || len(a.AccountingRef) > 512 {
		return fmt.Errorf("supervision.diagnosticAllowance requires 1..100 wakes, a 60..2592000 second window and a bounded accountingRef")
	}
	if s.DiscoveryLimit < 1 || s.DiscoveryLimit > 100 {
		return fmt.Errorf("supervision.discoveryLimit must be between 1 and 100")
	}
	if s.MaxEffortsPerWake < 1 || s.MaxEffortsPerWake > 20 || s.MaxEffortsPerWake > s.DiscoveryLimit {
		return fmt.Errorf("supervision.maxEffortsPerWake must be between 1 and min(20, discoveryLimit)")
	}
	if s.MinWakeIntervalSeconds < 60 || s.MinWakeIntervalSeconds > 86400 {
		return fmt.Errorf("supervision.minWakeIntervalSeconds must be between 60 and 86400")
	}
	if s.HealthySampleIntervalSeconds != 0 && (s.HealthySampleIntervalSeconds < s.MinWakeIntervalSeconds || s.HealthySampleIntervalSeconds > 2592000) {
		return fmt.Errorf("supervision.healthySampleIntervalSeconds must be zero or between the wake interval and 2592000")
	}
	if (s.HealthySampleIntervalSeconds == 0 && s.MaxHealthySamplesPerWake != 0) || (s.HealthySampleIntervalSeconds != 0 && (s.MaxHealthySamplesPerWake < 1 || s.MaxHealthySamplesPerWake > s.MaxEffortsPerWake)) {
		return fmt.Errorf("supervision.maxHealthySamplesPerWake must bound enabled sampling within maxEffortsPerWake")
	}
	return nil
}
