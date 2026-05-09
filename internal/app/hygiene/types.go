package hygiene

import (
	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	planapp "github.com/vrooli/vrooli/internal/app/plans"
)

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type Check struct {
	Name     string   `json:"name"`
	Passed   bool     `json:"passed"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
}

type Finding struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Path     string   `json:"path,omitempty"`
	Message  string   `json:"message"`
}

type Action struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Command string `json:"command,omitempty"`
}

type PlanCandidate struct {
	Path   string `json:"path"`
	Status string `json:"status,omitempty"`
	Reason string `json:"reason"`
}

type PlanFix struct {
	Source string             `json:"source"`
	Plan   planapp.PlanRecord `json:"plan"`
}

type Report struct {
	Success          bool                         `json:"success"`
	Root             string                       `json:"root"`
	Checks           []Check                      `json:"checks"`
	Findings         []Finding                    `json:"findings,omitempty"`
	Actions          []Action                     `json:"actions,omitempty"`
	PlanCandidates   []PlanCandidate              `json:"plan_candidates,omitempty"`
	FixesApplied     []PlanFix                    `json:"fixes_applied,omitempty"`
	Contract         contractapp.ValidationOutput `json:"contract,omitempty"`
	BlockingFailures int                          `json:"blocking_failures"`
	Warnings         int                          `json:"warnings"`
}

type Request struct {
	FixSafe bool
	Plans   bool
	FailOn  Severity
}
