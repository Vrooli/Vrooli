package main

import "strings"

// Constants
const (
	metadataFilename = "metadata.yaml"
	artifactsDirName = "artifacts"
)

var validIssueStatuses = map[string]struct{}{
	"open":      {},
	"active":    {},
	"waiting":   {},
	"completed": {},
	"failed":    {},
	"archived":  {},
}

// Config holds application configuration
type Config struct {
	Port         string
	QdrantURL    string
	IssuesDir    string
	ScenarioRoot string
}

// ProcessorState tracks the state of the background issue processor
type ProcessorState struct {
	Active           bool `json:"active"`
	ConcurrentSlots  int  `json:"concurrent_slots"`
	RefreshInterval  int  `json:"refresh_interval"`
	CurrentlyRunning int  `json:"currently_running"`
}

// Issue represents a tracked issue with all its metadata
type Issue struct {
	ID          string `yaml:"id" json:"id"`
	Title       string `yaml:"title" json:"title"`
	Description string `yaml:"description" json:"description"`
	Type        string `yaml:"type" json:"type"`
	Priority    string `yaml:"priority" json:"priority"`
	AppID       string `yaml:"app_id" json:"app_id"`
	Status      string `yaml:"status" json:"status"`

	Reporter struct {
		Name      string `yaml:"name" json:"name"`
		Email     string `yaml:"email" json:"email"`
		UserID    string `yaml:"user_id,omitempty" json:"user_id,omitempty"`
		Timestamp string `yaml:"timestamp" json:"timestamp"`
	} `yaml:"reporter" json:"reporter"`

	ErrorContext struct {
		ErrorMessage       string            `yaml:"error_message,omitempty" json:"error_message,omitempty"`
		ErrorLogs          string            `yaml:"error_logs,omitempty" json:"error_logs,omitempty"`
		StackTrace         string            `yaml:"stack_trace,omitempty" json:"stack_trace,omitempty"`
		AffectedFiles      []string          `yaml:"affected_files,omitempty" json:"affected_files,omitempty"`
		AffectedComponents []string          `yaml:"affected_components,omitempty" json:"affected_components,omitempty"`
		EnvironmentInfo    map[string]string `yaml:"environment_info,omitempty" json:"environment_info,omitempty"`
	} `yaml:"error_context,omitempty" json:"error_context,omitempty"`

	Investigation struct {
		AgentID                      string   `yaml:"agent_id,omitempty" json:"agent_id,omitempty"`
		StartedAt                    string   `yaml:"started_at,omitempty" json:"started_at,omitempty"`
		CompletedAt                  string   `yaml:"completed_at,omitempty" json:"completed_at,omitempty"`
		Report                       string   `yaml:"report,omitempty" json:"report,omitempty"`
		RootCause                    string   `yaml:"root_cause,omitempty" json:"root_cause,omitempty"`
		SuggestedFix                 string   `yaml:"suggested_fix,omitempty" json:"suggested_fix,omitempty"`
		ConfidenceScore              *int     `yaml:"confidence_score,omitempty" json:"confidence_score,omitempty"`
		InvestigationDurationMinutes *int     `yaml:"investigation_duration_minutes,omitempty" json:"investigation_duration_minutes,omitempty"`
		TokensUsed                   *int     `yaml:"tokens_used,omitempty" json:"tokens_used,omitempty"`
		CostEstimate                 *float64 `yaml:"cost_estimate,omitempty" json:"cost_estimate,omitempty"`
	} `yaml:"investigation,omitempty" json:"investigation,omitempty"`

	Fix struct {
		SuggestedFix       string `yaml:"suggested_fix,omitempty" json:"suggested_fix,omitempty"`
		ImplementationPlan string `yaml:"implementation_plan,omitempty" json:"implementation_plan,omitempty"`
		Applied            bool   `yaml:"applied" json:"applied"`
		AppliedAt          string `yaml:"applied_at,omitempty" json:"applied_at,omitempty"`
		CommitHash         string `yaml:"commit_hash,omitempty" json:"commit_hash,omitempty"`
		PrURL              string `yaml:"pr_url,omitempty" json:"pr_url,omitempty"`
		VerificationStatus string `yaml:"verification_status,omitempty" json:"verification_status,omitempty"`
		RollbackPlan       string `yaml:"rollback_plan,omitempty" json:"rollback_plan,omitempty"`
		FixDurationMinutes *int   `yaml:"fix_duration_minutes,omitempty" json:"fix_duration_minutes,omitempty"`
	} `yaml:"fix,omitempty" json:"fix,omitempty"`

	Attachments []Attachment `yaml:"attachments,omitempty" json:"attachments,omitempty"`

	Metadata struct {
		CreatedAt  string            `yaml:"created_at" json:"created_at"`
		UpdatedAt  string            `yaml:"updated_at" json:"updated_at"`
		ResolvedAt string            `yaml:"resolved_at,omitempty" json:"resolved_at,omitempty"`
		Tags       []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
		Labels     map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
		Watchers   []string          `yaml:"watchers,omitempty" json:"watchers,omitempty"`
		Extra      map[string]string `yaml:"extra,omitempty" json:"extra,omitempty"`
	} `yaml:"metadata" json:"metadata"`

	Notes string `yaml:"notes,omitempty" json:"notes,omitempty"`
}

// PromptPreviewRequest is used to preview investigation prompts
type PromptPreviewRequest struct {
	IssueID string `json:"issue_id,omitempty"`
	AgentID string `json:"agent_id,omitempty"`
	Issue   *Issue `json:"issue,omitempty"`
}

// PromptPreviewResponse contains the generated prompt preview
type PromptPreviewResponse struct {
	IssueID        string `json:"issue_id"`
	AgentID        string `json:"agent_id"`
	IssueTitle     string `json:"issue_title,omitempty"`
	IssueStatus    string `json:"issue_status,omitempty"`
	PromptTemplate string `json:"prompt_template"`
	PromptMarkdown string `json:"prompt_markdown"`
	GeneratedAt    string `json:"generated_at"`
	Source         string `json:"source"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

// Attachment represents a file attachment to an issue
type Attachment struct {
	Name     string `yaml:"name" json:"name"`
	Type     string `yaml:"type,omitempty" json:"type,omitempty"`
	Path     string `yaml:"path" json:"path"`
	Size     int64  `yaml:"size,omitempty" json:"size,omitempty"`
	Category string `yaml:"category,omitempty" json:"category,omitempty"`
}

// UpdateIssueRequest contains fields that can be updated for an issue
type UpdateIssueRequest struct {
	Title         *string            `json:"title"`
	Description   *string            `json:"description"`
	Type          *string            `json:"type"`
	Priority      *string            `json:"priority"`
	AppID         *string            `json:"app_id"`
	Status        *string            `json:"status"`
	Tags          *[]string          `json:"tags"`
	Labels        *map[string]string `json:"labels"`
	Watchers      *[]string          `json:"watchers"`
	Notes         *string            `json:"notes"`
	ResolvedAt    *string            `json:"resolved_at"`
	MetadataExtra *map[string]string `json:"metadata_extra"`

	Reporter *struct {
		Name      *string `json:"name"`
		Email     *string `json:"email"`
		UserID    *string `json:"user_id"`
		Timestamp *string `json:"timestamp"`
	} `json:"reporter"`

	ErrorContext *struct {
		ErrorMessage       *string            `json:"error_message"`
		ErrorLogs          *string            `json:"error_logs"`
		StackTrace         *string            `json:"stack_trace"`
		AffectedFiles      *[]string          `json:"affected_files"`
		AffectedComponents *[]string          `json:"affected_components"`
		EnvironmentInfo    *map[string]string `json:"environment_info"`
	} `json:"error_context"`

	Investigation *struct {
		AgentID                      *string  `json:"agent_id"`
		StartedAt                    *string  `json:"started_at"`
		CompletedAt                  *string  `json:"completed_at"`
		Report                       *string  `json:"report"`
		RootCause                    *string  `json:"root_cause"`
		SuggestedFix                 *string  `json:"suggested_fix"`
		ConfidenceScore              *int     `json:"confidence_score"`
		InvestigationDurationMinutes *int     `json:"investigation_duration_minutes"`
		TokensUsed                   *int     `json:"tokens_used"`
		CostEstimate                 *float64 `json:"cost_estimate"`
	} `json:"investigation"`

	Fix *struct {
		SuggestedFix       *string `json:"suggested_fix"`
		ImplementationPlan *string `json:"implementation_plan"`
		Applied            *bool   `json:"applied"`
		AppliedAt          *string `json:"applied_at"`
		CommitHash         *string `json:"commit_hash"`
		PrURL              *string `json:"pr_url"`
		VerificationStatus *string `json:"verification_status"`
		RollbackPlan       *string `json:"rollback_plan"`
		FixDurationMinutes *int    `json:"fix_duration_minutes"`
	} `json:"fix"`

	Artifacts []ArtifactPayload `json:"artifacts"`
}

// ArtifactPayload represents a new artifact being uploaded
type ArtifactPayload struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
	ContentType string `json:"content_type"`
}

// LegacyAttachmentPayload is the legacy format for attachments (deprecated)
type LegacyAttachmentPayload struct {
	Name        string `json:"name"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
	ContentType string `json:"content_type"`
}

// CreateIssueRequest contains all fields needed to create a new issue
type CreateIssueRequest struct {
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Type          string            `json:"type"`
	Priority      string            `json:"priority"`
	AppID         string            `json:"app_id"`
	Status        string            `json:"status"`
	Tags          []string          `json:"tags"`
	Labels        map[string]string `json:"labels"`
	Watchers      []string          `json:"watchers"`
	Notes         string            `json:"notes"`
	Environment   map[string]string `json:"environment"`
	MetadataExtra map[string]string `json:"metadata_extra"`
	Reporter      *struct {
		Name      string `json:"name"`
		Email     string `json:"email"`
		UserID    string `json:"user_id"`
		Timestamp string `json:"timestamp"`
	} `json:"reporter"`
	ReporterName          string                    `json:"reporter_name"`
	ReporterEmail         string                    `json:"reporter_email"`
	ReporterUserID        string                    `json:"reporter_user_id"`
	ErrorMessage          string                    `json:"error_message"`
	ErrorLogs             string                    `json:"error_logs"`
	StackTrace            string                    `json:"stack_trace"`
	AffectedFiles         []string                  `json:"affected_files"`
	AffectedComponents    []string                  `json:"affected_components"`
	AppLogs               string                    `json:"app_logs"`
	ConsoleLogs           string                    `json:"console_logs"`
	NetworkLogs           string                    `json:"network_logs"`
	ScreenshotData        string                    `json:"screenshot_data"`
	ScreenshotContentType string                    `json:"screenshot_content_type"`
	ScreenshotFilename    string                    `json:"screenshot_filename"`
	Attachments           []LegacyAttachmentPayload `json:"attachments"`
	Artifacts             []ArtifactPayload         `json:"artifacts"`
}

// Agent represents an AI agent that can investigate issues
type Agent struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	DisplayName    string   `json:"display_name"`
	Description    string   `json:"description"`
	Capabilities   []string `json:"capabilities"`
	IsActive       bool     `json:"is_active"`
	SuccessRate    float64  `json:"success_rate"`
	TotalRuns      int      `json:"total_runs"`
	SuccessfulRuns int      `json:"successful_runs"`
}

// App represents an application that can have issues
type App struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	TotalIssues int    `json:"total_issues"`
	OpenIssues  int    `json:"open_issues"`
}

// ApiResponse is a standard API response wrapper
type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// Helper functions

// cloneStringSlice creates a deep copy of a string slice
func cloneStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

// normalizeStringSlice removes empty and whitespace-only strings
func normalizeStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	var result []string
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// cloneStringMap creates a deep copy of a string map with trimmed keys/values
func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	clone := make(map[string]string, len(values))
	for k, v := range values {
		clone[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	if len(clone) == 0 {
		return nil
	}
	return clone
}
