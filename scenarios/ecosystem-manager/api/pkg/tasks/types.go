package tasks

import "time"

// TaskItem represents a unified task in the ecosystem
type TaskItem struct {
	ID                 string                 `json:"id" yaml:"id"`
	Title              string                 `json:"title" yaml:"title"`
	Type               string                 `json:"type" yaml:"type"`           // resource | scenario
	Operation          string                 `json:"operation" yaml:"operation"` // generator | improver
	Category           string                 `json:"category" yaml:"category"`
	Priority           string                 `json:"priority" yaml:"priority"`
	EffortEstimate     string                 `json:"effort_estimate" yaml:"effort_estimate"`
	Urgency            string                 `json:"urgency" yaml:"urgency"`
	ImpactScore        int                    `json:"impact_score" yaml:"impact_score"`
	Requirements       map[string]interface{} `json:"requirements" yaml:"requirements"`
	Dependencies       []string               `json:"dependencies" yaml:"dependencies"`
	Blocks             []string               `json:"blocks" yaml:"blocks"`
	RelatedScenarios   []string               `json:"related_scenarios" yaml:"related_scenarios"`
	RelatedResources   []string               `json:"related_resources" yaml:"related_resources"`
	AssignedResources  map[string]bool        `json:"assigned_resources" yaml:"assigned_resources"`
	Status             string                 `json:"status" yaml:"status"`
	ProgressPercent    int                    `json:"progress_percentage" yaml:"progress_percentage"`
	CurrentPhase       string                 `json:"current_phase" yaml:"current_phase"`
	StartedAt          string                 `json:"started_at" yaml:"started_at"`
	CompletedAt        string                 `json:"completed_at" yaml:"completed_at"`
	EstimatedComplete  string                 `json:"estimated_completion" yaml:"estimated_completion"`
	ValidationCriteria []string               `json:"validation_criteria" yaml:"validation_criteria"`
	CreatedBy          string                 `json:"created_by" yaml:"created_by"`
	CreatedAt          string                 `json:"created_at" yaml:"created_at"`
	UpdatedAt          string                 `json:"updated_at" yaml:"updated_at"`
	Tags               []string               `json:"tags" yaml:"tags"`
	Notes              string                 `json:"notes" yaml:"notes"`
	Results            map[string]interface{} `json:"results" yaml:"results"`
}

// OperationConfig represents configuration for each operation type
type OperationConfig struct {
	Name               string                 `json:"name" yaml:"name"`
	Type               string                 `json:"type" yaml:"type"`
	Target             string                 `json:"target" yaml:"target"`
	Description        string                 `json:"description" yaml:"description"`
	AdditionalSections []string               `json:"additional_sections" yaml:"additional_sections"`
	Variables          map[string]interface{} `json:"variables" yaml:"variables"`
	EffortAllocation   map[string]string      `json:"effort_allocation" yaml:"effort_allocation"`
	SuccessCriteria    []string               `json:"success_criteria" yaml:"success_criteria"`
	Principles         []string               `json:"principles" yaml:"principles"`
}

// PromptsConfig represents the unified prompts configuration
type PromptsConfig struct {
	Name         string                     `json:"name" yaml:"name"`
	Type         string                     `json:"type" yaml:"type"`
	Target       string                     `json:"target" yaml:"target"`
	Description  string                     `json:"description" yaml:"description"`
	BaseSections []string                   `json:"base_sections" yaml:"base_sections"`
	Operations   map[string]OperationConfig `json:"operations" yaml:"operations"`
	GlobalConfig map[string]interface{}     `json:"global_config" yaml:"global_config"`
}

// ClaudeCodeRequest represents a request to the Claude Code resource
type ClaudeCodeRequest struct {
	Prompt  string                 `json:"prompt"`
	Context map[string]interface{} `json:"context"`
}

// ClaudeCodeResponse represents a response from Claude Code
type ClaudeCodeResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	Output      string `json:"output"`
	Error       string `json:"error,omitempty"`
	RateLimited bool   `json:"rate_limited,omitempty"`
	RetryAfter  int    `json:"retry_after,omitempty"` // Seconds to wait before retry
	MaxTurnsExceeded bool `json:"max_turns_exceeded,omitempty"`
}

// RunningProcess tracks an executing claude-code process
type RunningProcess struct {
	TaskID    string
	Cmd       interface{} // Will hold *exec.Cmd but avoid import here
	Context   interface{} // Will hold context.Context
	Cancel    interface{} // Will hold context.CancelFunc
	StartTime time.Time
	ProcessID int
	AgentID   string
}

// ResourceInfo represents information about a discovered resource
type ResourceInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Port        int    `json:"port,omitempty"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Healthy     bool   `json:"healthy"`
	Version     string `json:"version,omitempty"`
	Status      string `json:"status,omitempty"` // e.g., "[UNREGISTERED]", "[MISSING]"
}

// ScenarioInfo represents information about a discovered scenario
type ScenarioInfo struct {
	Name           string `json:"name"`
	Path           string `json:"path"`
	Category       string `json:"category"`
	Description    string `json:"description"`
	Version        string `json:"version,omitempty"`
	Status         string `json:"status,omitempty"` // e.g., "available", "running"
	// Legacy fields - kept for compatibility but not populated
	PRDComplete    int    `json:"prd_completion_percentage,omitempty"`
	Healthy        bool   `json:"healthy,omitempty"`
	P0Requirements int    `json:"p0_requirements,omitempty"`
	P0Completed    int    `json:"p0_completed,omitempty"`
}

// PRDStatus represents the status of a scenario's PRD
type PRDStatus struct {
	CompletionPercentage int      `json:"completion_percentage"`
	P0Requirements       int      `json:"p0_requirements"`
	P0Completed          int      `json:"p0_completed"`
	P1Requirements       int      `json:"p1_requirements"`
	P1Completed          int      `json:"p1_completed"`
	MissingRequirements  []string `json:"missing_requirements"`
}

// ServiceConfig represents a resource's service.json configuration
type ServiceConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Port        int    `json:"port,omitempty"`
	Category    string `json:"category,omitempty"`
	Version     string `json:"version,omitempty"`
}
