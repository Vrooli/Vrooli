package testquality

import "sort"

// Requirement declarations and execution observations have separate owners.
// This model reconciles them without deriving requirement completion statuses.
type RequirementDeclaration struct {
	ID          string                      `json:"id"`
	Validations []RequirementResponsibility `json:"validation"`
}
type RequirementResponsibility struct {
	Type  string `json:"type"`
	Ref   string `json:"ref"`
	Phase string `json:"phase"`
}
type RequirementRegistry struct {
	SchemaVersion string                   `json:"schemaVersion"`
	Requirements  []RequirementDeclaration `json:"requirements"`
}

type ExecutionState string

const (
	ExecutionPassed  ExecutionState = "passed"
	ExecutionFailed  ExecutionState = "failed"
	ExecutionSkipped ExecutionState = "skipped"
	ExecutionNotRun  ExecutionState = "not_run"
	ExecutionUnknown ExecutionState = "unknown"
)

// TestLinks is supplied by a language/runner owner. IDs are already extracted
// using its established grammar, never by substring matching in this layer.
type TestLinks struct {
	Target        Target         `json:"target"`
	IDs           []string       `json:"requirementIds"`
	EvidenceKind  EvidenceKind   `json:"evidenceKind"`
	RunID         string         `json:"runId,omitempty"`
	ExpectedRunID string         `json:"-"`
	Execution     ExecutionState `json:"execution"`
}
type RequirementLink struct {
	ID           string         `json:"requirementId"`
	Target       Target         `json:"target"`
	Registration string         `json:"registration"` // registered, stale, unknown
	Execution    ExecutionState `json:"execution"`
	Reason       Reason         `json:"reason"`
	RunID        string         `json:"runId,omitempty"`
}
type RequirementScope struct {
	ID            string `json:"requirementId"`
	Applicability string `json:"applicability"` // applicable, not_applicable, unknown
}
type TraceabilityReport struct {
	EvidenceUnavailableReason Reason             `json:"evidenceUnavailableReason,omitempty"`
	SchemaVersion             string             `json:"schemaVersion"`
	UnavailableReason         Reason             `json:"unavailableReason,omitempty"`
	Requirements              []RequirementScope `json:"requirements"`
	Links                     []RequirementLink  `json:"links"`
	Limitations               []string           `json:"limitations"`
}

func (r TraceabilityReport) Normalized() TraceabilityReport {
	r.Links = append([]RequirementLink(nil), r.Links...)
	r.Requirements = append([]RequirementScope(nil), r.Requirements...)
	r.Limitations = append([]string(nil), r.Limitations...)
	unsupported := r.SchemaVersion != "requirement-traceability/v1"
	if unsupported {
		r.UnavailableReason = UnsupportedVersion
	}
	if r.UnavailableReason != "" {
		r.UnavailableReason = NormalizeReason(r.UnavailableReason)
	}
	if r.EvidenceUnavailableReason != "" {
		r.EvidenceUnavailableReason = NormalizeReason(r.EvidenceUnavailableReason)
	}
	if r.UnavailableReason != "" && r.UnavailableReason != ReasonNone {
		r.Requirements = nil
	}
	for i := range r.Requirements {
		switch r.Requirements[i].Applicability {
		case "applicable", "not_applicable":
		default:
			r.Requirements[i].Applicability = "unknown"
		}
	}
	for i := range r.Links {
		link := &r.Links[i]
		if r.UnavailableReason != "" && r.UnavailableReason != ReasonNone {
			link.Registration = "unknown"
		}
		switch link.Registration {
		case "registered", "stale":
		default:
			link.Registration = "unknown"
		}
		if unsupported {
			link.Execution = ExecutionUnknown
			link.Reason = UnsupportedVersion
		}
		switch link.Execution {
		case ExecutionPassed, ExecutionFailed, ExecutionSkipped, ExecutionNotRun:
		default:
			link.Execution = ExecutionUnknown
		}
		link.Reason = NormalizeReason(link.Reason)
	}
	return r
}

func ReconcileRequirements(registry RequirementRegistry, unavailable Reason, phase string, tests []TestLinks) TraceabilityReport {
	report := TraceabilityReport{SchemaVersion: "requirement-traceability/v1", Requirements: []RequirementScope{}, Links: []RequirementLink{},
		Limitations: []string{"Declared links and passing test observations do not establish behavioral adequacy or requirement completion."}}
	if unavailable == "" {
		unavailable = ReasonNone
	}
	if unavailable == ReasonNone && (registry.SchemaVersion != "requirement-registry/v1" || registry.Requirements == nil) {
		unavailable = MissingInput
	}
	if len(registry.Requirements) > 10000 || len(tests) > 10000 {
		report.UnavailableReason = ResolutionLimit
		return report
	}
	known := map[string]bool{}
	for _, declaration := range registry.Requirements {
		if declaration.ID == "" || known[declaration.ID] {
			unavailable = ParseFailure
		}
		known[declaration.ID] = true
	}
	if unavailable == ReasonNone {
		for _, declaration := range registry.Requirements {
			applicability := "not_applicable"
			if len(declaration.Validations) == 0 || phase == "" {
				applicability = "unknown"
			}
			for _, validation := range declaration.Validations {
				if validation.Phase == phase && phase != "" {
					applicability = "applicable"
					break
				}
				if validation.Phase == "" {
					applicability = "unknown"
				}
			}
			report.Requirements = append(report.Requirements, RequirementScope{ID: declaration.ID, Applicability: applicability})
		}
	}
	for _, test := range tests {
		seen := map[string]bool{}
		for _, id := range test.IDs {
			if seen[id] {
				continue
			}
			seen[id] = true
			if len(report.Links) >= 10000 {
				report.Links = nil
				report.Requirements = nil
				report.UnavailableReason = ResolutionLimit
				return report
			}
			link := RequirementLink{ID: id, Target: test.Target, Registration: "unknown", Execution: ExecutionUnknown, Reason: unavailable, RunID: test.RunID}
			if unavailable == ReasonNone {
				link.Registration = "stale"
				if known[id] {
					link.Registration = "registered"
				}
			}
			switch {
			case test.EvidenceKind == Static:
				link.Execution = ExecutionNotRun
				if link.Reason == ReasonNone {
					link.Reason = NotExecuted
				}
			case test.EvidenceKind != Runtime:
				link.Reason = MissingInput
			case test.RunID == "" || test.ExpectedRunID == "" || test.RunID != test.ExpectedRunID:
				link.Reason = StaleEvidence
			default:
				switch test.Execution {
				case ExecutionPassed, ExecutionFailed, ExecutionSkipped, ExecutionNotRun:
					link.Execution = test.Execution
				default:
					link.Reason = MissingInput
				}
			}
			report.Links = append(report.Links, link)
		}
	}
	sort.Slice(report.Requirements, func(i, j int) bool { return report.Requirements[i].ID < report.Requirements[j].ID })
	report.UnavailableReason = unavailable
	return report
}
