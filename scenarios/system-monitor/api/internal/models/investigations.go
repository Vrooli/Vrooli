package models

import (
	"time"
)

// Investigation represents an anomaly investigation
type Investigation struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"` // queued, in_progress, completed, failed
	AnomalyID string                 `json:"anomaly_id"`
	StartTime time.Time              `json:"start_time"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Findings  string                 `json:"findings,omitempty"`
	Progress  int                    `json:"progress"`          // 0-100
	Details   map[string]interface{} `json:"details,omitempty"`
	Steps     []InvestigationStep    `json:"steps,omitempty"`
}

// InvestigationStep represents a step in an investigation
type InvestigationStep struct {
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Findings  string     `json:"findings,omitempty"`
}

// InvestigationRequest represents a request to trigger an investigation
type InvestigationRequest struct {
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Context     map[string]interface{} `json:"context"`
	TimeRange   string                 `json:"time_range"`
	AutoResolve bool                   `json:"auto_resolve"`
}

// InvestigationResponse represents the response from an investigation
type InvestigationResponse struct {
	InvestigationID string                 `json:"investigation_id"`
	Status          string                 `json:"status"`
	Findings        []InvestigationFinding `json:"findings"`
	RootCause       *RootCause             `json:"root_cause,omitempty"`
	Recommendations []string               `json:"recommendations"`
	AffectedSystems []string               `json:"affected_systems"`
	Impact          ImpactAssessment       `json:"impact"`
	Resolution      *ResolutionPlan        `json:"resolution,omitempty"`
	StartedAt       time.Time              `json:"started_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	Duration        string                 `json:"duration"`
}

// InvestigationFinding represents a finding during investigation
type InvestigationFinding struct {
	Type        string    `json:"type"` // metric, log, process, network
	Description string    `json:"description"`
	Confidence  float64   `json:"confidence"` // 0.0 to 1.0
	Evidence    []string  `json:"evidence"`
	Timestamp   time.Time `json:"timestamp"`
	Impact      string    `json:"impact"` // low, medium, high
}

// RootCause represents the identified root cause of an issue
type RootCause struct {
	Category    string          `json:"category"`
	Description string          `json:"description"`
	Confidence  float64         `json:"confidence"`
	Evidence    []string        `json:"evidence"`
	Timeline    []TimelineEvent `json:"timeline"`
}

// TimelineEvent represents an event in the investigation timeline
type TimelineEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Event     string    `json:"event"`
	Source    string    `json:"source"`
	Relevance string    `json:"relevance"` // high, medium, low
}

// ImpactAssessment assesses the impact of an anomaly
type ImpactAssessment struct {
	Severity        string   `json:"severity"` // low, medium, high, critical
	AffectedUsers   int      `json:"affected_users"`
	PerformanceLoss float64  `json:"performance_loss"` // percentage
	EstimatedCost   float64  `json:"estimated_cost"`   // in dollars
	BusinessImpact  []string `json:"business_impact"`
}

// ResolutionPlan provides steps to resolve an issue
type ResolutionPlan struct {
	Steps          []ResolutionStep `json:"steps"`
	EstimatedTime  string           `json:"estimated_time"`
	RequiredSkills []string         `json:"required_skills"`
	RiskLevel      string           `json:"risk_level"`
	AutoExecutable bool             `json:"auto_executable"`
}

// ResolutionStep represents a single step in resolution
type ResolutionStep struct {
	Order       int    `json:"order"`
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
	Expected    string `json:"expected"`
	Risk        string `json:"risk"`
}

// Anomaly represents a detected system anomaly
type Anomaly struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	MetricData  map[string]interface{} `json:"metric_data"`
	DetectedAt  time.Time              `json:"detected_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Status      string                 `json:"status"`
}