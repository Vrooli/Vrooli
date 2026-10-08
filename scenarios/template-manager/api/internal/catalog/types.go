package catalog

import (
	"fmt"
	"time"
)

type TemplateKind string

const (
	KindScenario TemplateKind = "scenario"
	KindDesign   TemplateKind = "design"
	KindResource TemplateKind = "resource"
)

type TemplateRecord struct {
	ID             string
	Kind           TemplateKind
	DisplayName    string
	Version        string
	ManifestPath   string
	SourcePath     string
	Tags           []string
	Status         string
	CurrentVersion string
	LatestVersion  string
	LagCount       int32
	UpdatedAt      time.Time
}

// ScenarioTemplate is the source-of-truth projection read from a scenario
// template manifest. The registry persists it so validation history and source
// metadata share one template identity.
type ScenarioTemplate struct {
	ID           string
	Version      string
	ManifestPath string
	SourcePath   string
	UpdatedAt    time.Time
}

type ValidationMode string

const (
	ModeShallow ValidationMode = "shallow"
	ModeDeep    ValidationMode = "deep"
	ModeDrift   ValidationMode = "drift"
)

type PhaseResult struct {
	Phase        string `json:"phase"`
	Status       string `json:"status"`
	FindingCount int32  `json:"finding_count"`
}

type ValidationFinding struct {
	Key      string `json:"key"`
	Severity string `json:"severity"`
	Summary  string `json:"summary"`
	Source   string `json:"source"`
}

type ValidationRun struct {
	ID           string
	TemplateID   string
	Mode         ValidationMode
	Target       string
	Status       string
	Trigger      string
	StartedAt    time.Time
	FinishedAt   time.Time
	PhaseResults []PhaseResult
	Findings     []ValidationFinding
}

type DriftSnapshot struct {
	ID         string
	TemplateID string
	Target     string
	Status     string
	DriftCount int32
	CapturedAt time.Time
}

type DebtEntry struct {
	Key         string
	TemplateID  string
	Source      string
	Severity    string
	Status      string
	Title       string
	Detail      string
	FirstSeenAt time.Time
	LastSeenAt  time.Time
}

type MeasureWindow struct {
	From time.Time
	To   time.Time
}

type StandingBucket struct {
	Standing string
	Count    int64
}

type MonitorStatus struct {
	ID              string
	Enabled         bool
	IntervalSeconds int64
	InFlight        bool
	LastRunID       string
	LastStatus      string
	LastStartedAt   time.Time
	LastFinishedAt  time.Time
	NextRunAt       time.Time
	GreenStreak     int64
	UpdatedAt       time.Time
}

type ErrNotFound struct {
	Kind string
	ID   string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("%s %q not found", e.Kind, e.ID)
}
