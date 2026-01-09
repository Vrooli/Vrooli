package tasks

import (
	"fmt"
	"regexp"
	"strings"
)

var ValidTaskTypes = []string{"resource", "scenario"}

var ValidTaskOperations = []string{"generator", "improver"}

const (
	StatusPending            = "pending"
	StatusInProgress         = "in-progress"
	StatusCompleted          = "completed"
	StatusFailed             = "failed"
	StatusCompletedFinalized = "completed-finalized"
	StatusFailedBlocked      = "failed-blocked"
	StatusArchived           = "archived"
)

var QueueStatuses = []string{
	StatusPending,
	StatusInProgress,
	StatusCompleted,
	StatusFailed,
	StatusCompletedFinalized,
	StatusFailedBlocked,
	StatusArchived,
}

// IsValidTaskType reports whether the provided type is supported.
func IsValidTaskType(taskType string) bool {
	taskType = strings.TrimSpace(strings.ToLower(taskType))
	for _, candidate := range ValidTaskTypes {
		if candidate == taskType {
			return true
		}
	}
	return false
}

// IsValidTaskOperation reports whether the provided operation is supported.
func IsValidTaskOperation(operation string) bool {
	operation = strings.TrimSpace(strings.ToLower(operation))
	for _, candidate := range ValidTaskOperations {
		if candidate == operation {
			return true
		}
	}
	return false
}

// TaskItem represents a unified task in the ecosystem
type TaskItem struct {
	ID                          string         `json:"id" yaml:"id"`
	Title                       string         `json:"title" yaml:"title"`
	Type                        string         `json:"type" yaml:"type"`           // resource | scenario
	Operation                   string         `json:"operation" yaml:"operation"` // generator | improver
	Target                      string         `json:"target,omitempty" yaml:"target,omitempty"`
	Targets                     []string       `json:"targets,omitempty" yaml:"targets,omitempty"`
	Category                    string         `json:"category" yaml:"category"`
	Priority                    string         `json:"priority" yaml:"priority"`
	EffortEstimate              string         `json:"effort_estimate" yaml:"effort_estimate"`
	Urgency                     string         `json:"urgency" yaml:"urgency"`
	Dependencies                []string       `json:"dependencies" yaml:"dependencies"`
	Blocks                      []string       `json:"blocks" yaml:"blocks"`
	RelatedScenarios            []string       `json:"related_scenarios" yaml:"related_scenarios"`
	RelatedResources            []string       `json:"related_resources" yaml:"related_resources"`
	Status                      string         `json:"status" yaml:"status"`
	CurrentPhase                string         `json:"current_phase" yaml:"current_phase"`
	StartedAt                   string         `json:"started_at" yaml:"started_at"`
	CompletedAt                 string         `json:"completed_at" yaml:"completed_at"`
	CooldownUntil               string         `json:"cooldown_until,omitempty" yaml:"cooldown_until,omitempty"`
	CompletionCount             int            `json:"completion_count" yaml:"completion_count"`
	LastCompletedAt             string         `json:"last_completed_at" yaml:"last_completed_at"`
	ValidationCriteria          []string       `json:"validation_criteria" yaml:"validation_criteria"`
	CreatedBy                   string         `json:"created_by" yaml:"created_by"`
	CreatedAt                   string         `json:"created_at" yaml:"created_at"`
	UpdatedAt                   string         `json:"updated_at" yaml:"updated_at"`
	Tags                        []string       `json:"tags" yaml:"tags"`
	Notes                       string         `json:"notes" yaml:"notes"`
	Results                     map[string]any `json:"results" yaml:"results"`
	ConsecutiveCompletionClaims float64        `json:"consecutive_completion_claims" yaml:"consecutive_completion_claims"`
	ConsecutiveFailures         int            `json:"consecutive_failures" yaml:"consecutive_failures"`
	ProcessorAutoRequeue        bool           `json:"processor_auto_requeue" yaml:"processor_auto_requeue"`
	SteerMode                   string         `json:"steer_mode,omitempty" yaml:"steer_mode,omitempty"`                       // Optional manual steering mode when Auto Steer profile is not set
	AutoSteerProfileID          string         `json:"auto_steer_profile_id,omitempty" yaml:"auto_steer_profile_id,omitempty"` // Auto Steer profile to use
	// Ephemeral prompt context (not persisted)
	LatestOutputPath string `json:"-" yaml:"-"`
}

// OperationConfig represents configuration for each operation type
type OperationConfig struct {
	Name               string            `json:"name" yaml:"name"`
	Type               string            `json:"type" yaml:"type"`
	Target             string            `json:"target" yaml:"target"`
	Description        string            `json:"description" yaml:"description"`
	Template           string            `json:"template,omitempty" yaml:"template,omitempty"`
	AdditionalSections []string          `json:"additional_sections" yaml:"additional_sections"`
	Variables          map[string]any    `json:"variables" yaml:"variables"`
	EffortAllocation   map[string]string `json:"effort_allocation" yaml:"effort_allocation"`
	SuccessCriteria    []string          `json:"success_criteria" yaml:"success_criteria"`
	Principles         []string          `json:"principles" yaml:"principles"`
}

// PromptsConfig represents the unified prompts configuration
type PromptsConfig struct {
	Name         string                     `json:"name" yaml:"name"`
	Type         string                     `json:"type" yaml:"type"`
	Target       string                     `json:"target" yaml:"target"`
	Description  string                     `json:"description" yaml:"description"`
	BaseSections []string                   `json:"base_sections" yaml:"base_sections"`
	Operations   map[string]OperationConfig `json:"operations" yaml:"operations"`
	GlobalConfig map[string]any             `json:"global_config" yaml:"global_config"`
}

// ClaudeCodeRequest represents a request to the Claude Code resource
type ClaudeCodeRequest struct {
	Prompt  string         `json:"prompt"`
	Context map[string]any `json:"context"`
}

// ClaudeCodeResponse represents a response from Claude Code
type ClaudeCodeResponse struct {
	Success          bool   `json:"success"`
	Message          string `json:"message"`
	Output           string `json:"output"`
	Error            string `json:"error,omitempty"`
	RateLimited      bool   `json:"rate_limited,omitempty"`
	RetryAfter       int    `json:"retry_after,omitempty"` // Seconds to wait before retry
	MaxTurnsExceeded bool   `json:"max_turns_exceeded,omitempty"`
	IdleTimeout      bool   `json:"idle_timeout,omitempty"`
	FinalMessage     string `json:"final_message,omitempty"`
	TranscriptPath   string `json:"transcript_path,omitempty"`
	LastMessagePath  string `json:"last_message_path,omitempty"`
	CleanOutputPath  string `json:"clean_output_path,omitempty"`
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
	Name        string `json:"name"`
	Path        string `json:"path"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Version     string `json:"version,omitempty"`
	Status      string `json:"status,omitempty"` // e.g., "available", "running"
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

// NormalizeTargets merges single and multi-select target inputs into a canonical representation.
func NormalizeTargets(primary string, targets []string) ([]string, string) {
	candidate := make([]string, 0, len(targets)+1)

	if strings.TrimSpace(primary) != "" {
		candidate = append(candidate, primary)
	}

	candidate = append(candidate, targets...)

	seen := make(map[string]struct{})
	normalized := make([]string, 0, len(candidate))

	for _, raw := range candidate {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}

		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	var canonical string
	if len(normalized) > 0 {
		canonical = normalized[0]
	}

	return normalized, canonical
}

// InferTargetFromTitle attempts to extract a target name from a human-readable title.
func InferTargetFromTitle(title string, taskType string) string {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return ""
	}

	typeToken := strings.ToLower(strings.TrimSpace(taskType))
	var typePattern string
	if typeToken == "" {
		typePattern = "(resource|scenario)"
	} else {
		typePattern = regexp.QuoteMeta(typeToken)
	}

	targetPattern := `([A-Za-z0-9][A-Za-z0-9\-_.\s/]+?)`
	patterns := []string{
		fmt.Sprintf(`(?i)(enhance|improve|upgrade|fix|polish|validate)\s+%s\s+%s`, typePattern, targetPattern),
		fmt.Sprintf(`(?i)(enhance|improve|upgrade|fix|polish|validate)\s+%s\s+%s`, targetPattern, typePattern),
		fmt.Sprintf(`(?i)(create|generate|build|produce)\s+%s\s+%s`, typePattern, targetPattern),
		fmt.Sprintf(`(?i)(create|generate|build|produce)\s+%s\s+%s`, targetPattern, typePattern),
	}

	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}

		match := re.FindStringSubmatch(trimmed)
		if len(match) >= 3 {
			target := strings.TrimSpace(match[len(match)-1])
			target = strings.Trim(target, "-_")
			target = strings.ReplaceAll(target, "  ", " ")
			if target != "" {
				return target
			}
		}
	}

	return ""
}

// InferTargetFromID attempts to recover a target from a generated task ID.
func InferTargetFromID(id, taskType, operation string) string {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return ""
	}

	prefix := strings.ToLower(strings.TrimSpace(taskType))
	op := strings.ToLower(strings.TrimSpace(operation))
	compoundPrefix := fmt.Sprintf("%s-%s-", prefix, op)

	lowerID := strings.ToLower(trimmed)
	if strings.HasPrefix(lowerID, compoundPrefix) {
		trimmed = trimmed[len(compoundPrefix):]
	}

	reNumeric := regexp.MustCompile(`-[0-9]{4,}$`)
	for {
		if loc := reNumeric.FindStringIndex(trimmed); loc != nil && loc[0] > 0 {
			trimmed = trimmed[:loc[0]]
		} else {
			break
		}
	}

	trimmed = strings.Trim(trimmed, "-_")
	trimmed = strings.TrimSpace(trimmed)
	trimmed = strings.ReplaceAll(trimmed, "-", " ")
	trimmed = strings.Join(strings.Fields(trimmed), " ")

	if trimmed == "" {
		return ""
	}

	return trimmed
}

// CollectTargets returns a canonical list of targets for a task, inferring when necessary.
func CollectTargets(item *TaskItem) []string {
	normalizedTargets, _ := NormalizeTargets(item.Target, item.Targets)
	if len(normalizedTargets) > 0 {
		return normalizedTargets
	}

	var derived []string

	// Use related fields when explicit targets are missing
	if strings.EqualFold(item.Type, "resource") {
		derived = append(derived, item.RelatedResources...)
	} else if strings.EqualFold(item.Type, "scenario") {
		derived = append(derived, item.RelatedScenarios...)
	}

	for _, candidate := range derived {
		trimmed := strings.TrimSpace(candidate)
		if trimmed != "" {
			normalizedTargets = append(normalizedTargets, trimmed)
		}
	}

	if len(normalizedTargets) > 0 {
		return normalizedTargets
	}

	if inferred := InferTargetFromTitle(item.Title, item.Type); inferred != "" {
		return []string{inferred}
	}

	if inferred := InferTargetFromID(item.ID, item.Type, item.Operation); inferred != "" {
		return []string{inferred}
	}

	return normalizedTargets
}
