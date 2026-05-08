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
	ID               string         `json:"id"`
	Fingerprint      string         `json:"fingerprint"`
	Type             Type           `json:"type"`
	Severity         Severity       `json:"severity"`
	Status           Status         `json:"status"`
	Title            string         `json:"title"`
	Summary          string         `json:"summary"`
	DetectedAt       time.Time      `json:"detectedAt"`
	LastSeenAt       time.Time      `json:"lastSeenAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	ResolvedAt       *time.Time     `json:"resolvedAt,omitempty"`
	AcknowledgedAt   *time.Time     `json:"acknowledgedAt,omitempty"`
	IgnoredAt        *time.Time     `json:"ignoredAt,omitempty"`
	BootID           string         `json:"bootId,omitempty"`
	PreviousBootID   string         `json:"previousBootId,omitempty"`
	SourceCheckIDs   []string       `json:"sourceCheckIds,omitempty"`
	SourceResultIDs  []string       `json:"sourceResultIds,omitempty"`
	Evidence         map[string]any `json:"evidence,omitempty"`
	Recommendations  []string       `json:"recommendations,omitempty"`
	EventCount       int            `json:"eventCount"`
	ObservationCount int            `json:"observationCount"`
	OperatorNotes    string         `json:"operatorNotes,omitempty"`
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
	Fingerprint     string
	Type            Type
	Severity        Severity
	Title           string
	Summary         string
	ObservedAt      time.Time
	BootID          string
	PreviousBootID  string
	SourceCheckID   string
	Evidence        map[string]any
	Recommendations []string
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
