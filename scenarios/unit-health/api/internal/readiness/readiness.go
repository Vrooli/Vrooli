package readiness

import (
	"fmt"
	"strings"
)

type Status string

const (
	Ready       Status = "ready"
	Missing     Status = "missing"
	Stale       Status = "stale"
	Unavailable Status = "unavailable"
	Unsupported Status = "unsupported"
)

type Requirement struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Version     string `json:"version,omitempty"`
	Status      Status `json:"status"`
	Source      string `json:"source"`
	Remediation string `json:"remediation"`
}

type Report struct {
	Status       Status        `json:"status"`
	Requirements []Requirement `json:"requirements"`
	Source       string        `json:"source"`
}

func (r Report) Validate() error {
	if r.Status == "" {
		return fmt.Errorf("readiness: status is required")
	}
	switch r.Status {
	case Ready, Missing, Stale, Unavailable, Unsupported:
	default:
		return fmt.Errorf("readiness: unsupported status %q", r.Status)
	}
	if strings.TrimSpace(r.Source) == "" {
		return fmt.Errorf("readiness: source is required")
	}
	for _, requirement := range r.Requirements {
		if strings.TrimSpace(requirement.ID) == "" || strings.TrimSpace(requirement.Kind) == "" {
			return fmt.Errorf("readiness: requirement id and kind are required")
		}
		if strings.TrimSpace(requirement.Remediation) == "" {
			return fmt.Errorf("readiness: requirement %q has no governed remediation", requirement.ID)
		}
	}
	return nil
}

func (r Report) BlocksExecution() bool { return r.Status != Ready }
