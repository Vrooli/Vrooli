// Package incidents owns durable operator-facing incident lifecycle records.
package incidents

import "time"

type Type string

const (
	TypeHostIntegrity   Type = "host_integrity"
	TypeUncleanBoot     Type = "unclean_boot"
	TypeResourceFailure Type = "resource_failure"
	TypeScenarioFailure Type = "scenario_failure"
	TypeAutohealFailure Type = "autoheal_failure"
	TypeManual          Type = "manual"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

type Status string

const (
	StatusOpen         Status = "open"
	StatusAcknowledged Status = "acknowledged"
	StatusResolved     Status = "resolved"
	StatusIgnored      Status = "ignored"
)

type Incident struct {
	ID                    string                 `json:"id"`
	Fingerprint           string                 `json:"fingerprint"`
	Type                  Type                   `json:"type"`
	Severity              Severity               `json:"severity"`
	Status                Status                 `json:"status"`
	Title                 string                 `json:"title"`
	Summary               string                 `json:"summary"`
	DetectedAt            time.Time              `json:"detectedAt"`
	LastSeenAt            time.Time              `json:"lastSeenAt"`
	UpdatedAt             time.Time              `json:"updatedAt"`
	ResolvedAt            *time.Time             `json:"resolvedAt,omitempty"`
	AcknowledgedAt        *time.Time             `json:"acknowledgedAt,omitempty"`
	IgnoredAt             *time.Time             `json:"ignoredAt,omitempty"`
	BootID                string                 `json:"bootId,omitempty"`
	PreviousBootID        string                 `json:"previousBootId,omitempty"`
	SourceCheckIDs        []string               `json:"sourceCheckIds,omitempty"`
	SourceResultIDs       []string               `json:"sourceResultIds,omitempty"`
	Evidence              map[string]any         `json:"evidence,omitempty"`
	Recommendations       []string               `json:"recommendations,omitempty"`
	Diagnosis             string                 `json:"diagnosis,omitempty"`
	Confidence            string                 `json:"confidence,omitempty"`
	EvidenceItems         []EvidenceItem         `json:"evidenceItems,omitempty"`
	CorroborationNeeded   []string               `json:"corroborationNeeded,omitempty"`
	SafeActions           []string               `json:"safeActions,omitempty"`
	OperatorActions       []string               `json:"operatorActions,omitempty"`
	RollbackOrFallback    []string               `json:"rollbackOrFallback,omitempty"`
	PostChecks            []string               `json:"postChecks,omitempty"`
	RemediationCandidates []RemediationCandidate `json:"remediationCandidates,omitempty"`
	RemediationArtifacts  []RemediationArtifact  `json:"remediationArtifacts,omitempty"`
	Outcome               *Outcome               `json:"outcome,omitempty"`
	EventCount            int                    `json:"eventCount"`
	ObservationCount      int                    `json:"observationCount"`
	OperatorNotes         string                 `json:"operatorNotes,omitempty"`
}

type EvidenceItem struct {
	ID                    string         `json:"id"`
	Kind                  string         `json:"kind"`
	Severity              Severity       `json:"severity"`
	Summary               string         `json:"summary"`
	Source                string         `json:"source"`
	BootID                string         `json:"bootId,omitempty"`
	Timestamp             *time.Time     `json:"timestamp,omitempty"`
	Data                  map[string]any `json:"data,omitempty"`
	PlatformApplicability string         `json:"platformApplicability,omitempty"`
}

type RemediationCandidate struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Applicability      string   `json:"applicability"`
	Platforms          []string `json:"platforms,omitempty"`
	RequiresOperator   bool     `json:"requiresOperator"`
	RequiresPrivilege  bool     `json:"requiresPrivilege"`
	RiskLevel          string   `json:"riskLevel,omitempty"`
	TemplateID         string   `json:"templateId,omitempty"`
	PreflightChecks    []string `json:"preflightChecks,omitempty"`
	Simulation         string   `json:"simulation,omitempty"`
	ArtifactPolicy     string   `json:"artifactPolicy,omitempty"`
	RollbackOrFallback []string `json:"rollbackOrFallback,omitempty"`
	PostChecks         []string `json:"postChecks,omitempty"`
	DecisionPrompt     string   `json:"decisionPrompt,omitempty"`
}

type RemediationArtifact struct {
	ID            string         `json:"id"`
	RemediationID string         `json:"remediationId"`
	Path          string         `json:"path"`
	GeneratedAt   time.Time      `json:"generatedAt"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type Outcome struct {
	RemediationID       string    `json:"remediationId,omitempty"`
	Status              string    `json:"status"`
	Note                string    `json:"note,omitempty"`
	ReportedAt          time.Time `json:"reportedAt"`
	AskID               string    `json:"askId,omitempty"`
	IncidentFingerprint string    `json:"incidentFingerprint,omitempty"`
	ScriptPath          string    `json:"scriptPath,omitempty"`
	ExitStatus          int       `json:"exitStatus,omitempty"`
	Output              string    `json:"output,omitempty"`
}

type Observation struct {
	ID            int64          `json:"id"`
	IncidentID    string         `json:"incidentId"`
	ObservedAt    time.Time      `json:"observedAt"`
	SourceCheckID string         `json:"sourceCheckId,omitempty"`
	Severity      Severity       `json:"severity"`
	Status        string         `json:"status,omitempty"`
	Message       string         `json:"message"`
	Evidence      map[string]any `json:"evidence,omitempty"`
}

type StatusHistory struct {
	ID         int64     `json:"id"`
	IncidentID string    `json:"incidentId"`
	FromStatus Status    `json:"fromStatus,omitempty"`
	ToStatus   Status    `json:"toStatus"`
	Note       string    `json:"note,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

type ListFilters struct {
	Status   Status     `json:"status,omitempty"`
	Severity Severity   `json:"severity,omitempty"`
	Type     Type       `json:"type,omitempty"`
	Since    *time.Time `json:"since,omitempty"`
	Until    *time.Time `json:"until,omitempty"`
	Limit    int        `json:"limit"`
}

type ListResponse struct {
	Incidents []Incident  `json:"incidents"`
	Total     int         `json:"total"`
	Filters   ListFilters `json:"filters"`
}

type UpsertInput struct {
	Fingerprint           string
	Type                  Type
	Severity              Severity
	Title                 string
	Summary               string
	ObservedAt            time.Time
	BootID                string
	PreviousBootID        string
	SourceCheckID         string
	Evidence              map[string]any
	Recommendations       []string
	Diagnosis             string
	Confidence            string
	EvidenceItems         []EvidenceItem
	CorroborationNeeded   []string
	SafeActions           []string
	OperatorActions       []string
	RollbackOrFallback    []string
	PostChecks            []string
	RemediationCandidates []RemediationCandidate
	RemediationArtifacts  []RemediationArtifact
	Outcome               *Outcome
}

func ValidStatus(value string) bool {
	switch Status(value) {
	case "", StatusOpen, StatusAcknowledged, StatusResolved, StatusIgnored:
		return true
	default:
		return false
	}
}

func ValidSeverity(value string) bool {
	switch Severity(value) {
	case "", SeverityInfo, SeverityWarning, SeverityCritical:
		return true
	default:
		return false
	}
}

func ValidType(value string) bool {
	switch Type(value) {
	case "", TypeHostIntegrity, TypeUncleanBoot, TypeResourceFailure, TypeScenarioFailure, TypeAutohealFailure, TypeManual:
		return true
	default:
		return false
	}
}
