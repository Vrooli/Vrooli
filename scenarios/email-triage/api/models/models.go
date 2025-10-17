package models

import (
	"time"
	"encoding/json"
)

// User represents a user in the multi-tenant system
type User struct {
	ID            string          `json:"id" db:"id"`
	EmailProfiles json.RawMessage `json:"email_profiles" db:"email_profiles"`
	Preferences   json.RawMessage `json:"preferences" db:"preferences"`
	PlanType      string          `json:"plan_type" db:"plan_type"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

// EmailAccount represents an email account connected to the system
type EmailAccount struct {
	ID           string          `json:"id" db:"id"`
	UserID       string          `json:"user_id" db:"user_id"`
	EmailAddress string          `json:"email_address" db:"email_address"`
	IMAPSettings json.RawMessage `json:"imap_settings" db:"imap_settings"`
	SMTPSettings json.RawMessage `json:"smtp_settings" db:"smtp_settings"`
	LastSync     *time.Time      `json:"last_sync" db:"last_sync"`
	SyncEnabled  bool            `json:"sync_enabled" db:"sync_enabled"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
}

// IMAPConfig represents IMAP connection settings
type IMAPConfig struct {
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	UseTLS   bool   `json:"use_tls"`
}

// SMTPConfig represents SMTP connection settings  
type SMTPConfig struct {
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	UseTLS   bool   `json:"use_tls"`
}

// TriageRule represents an AI-generated or manual email processing rule
type TriageRule struct {
	ID            string          `json:"id" db:"id"`
	UserID        string          `json:"user_id" db:"user_id"`
	Name          string          `json:"name" db:"name"`
	Description   string          `json:"description" db:"description"`
	Conditions    json.RawMessage `json:"conditions" db:"conditions"`
	Actions       json.RawMessage `json:"actions" db:"actions"`
	Priority      int             `json:"priority" db:"priority"`
	Enabled       bool            `json:"enabled" db:"enabled"`
	CreatedByAI   bool            `json:"created_by_ai" db:"created_by_ai"`
	AIConfidence  float64         `json:"ai_confidence" db:"ai_confidence"`
	MatchCount    int             `json:"match_count" db:"match_count"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

// RuleCondition represents a condition in a triage rule
type RuleCondition struct {
	Field    string      `json:"field"`     // sender, subject, body, etc.
	Operator string      `json:"operator"`  // contains, equals, matches, etc.
	Value    interface{} `json:"value"`     // The value to match against
	Weight   float64     `json:"weight"`    // Importance of this condition (0-1)
}

// RuleAction represents an action to take when a rule matches
type RuleAction struct {
	Type       string                 `json:"type"`       // forward, archive, label, etc.
	Parameters map[string]interface{} `json:"parameters"` // Action-specific parameters
}

// ProcessedEmail represents an email that has been processed by the system
type ProcessedEmail struct {
	ID               string          `json:"id" db:"id"`
	AccountID        string          `json:"account_id" db:"account_id"`
	MessageID        string          `json:"message_id" db:"message_id"`
	Subject          string          `json:"subject" db:"subject"`
	SenderEmail      string          `json:"sender_email" db:"sender_email"`
	RecipientEmails  []string        `json:"recipient_emails" db:"recipient_emails"`
	BodyPreview      string          `json:"body_preview" db:"body_preview"`
	FullBody         string          `json:"full_body" db:"full_body"`
	PriorityScore    float64         `json:"priority_score" db:"priority_score"`
	VectorID         *string         `json:"vector_id" db:"vector_id"`
	ProcessedAt      time.Time       `json:"processed_at" db:"processed_at"`
	ActionsTaken     json.RawMessage `json:"actions_taken" db:"actions_taken"`
	Metadata         json.RawMessage `json:"metadata" db:"metadata"`
}

// EmailVector represents vector embedding for semantic search
type EmailVector struct {
	ID      string                 `json:"id"`
	Vector  []float32             `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

// SearchResult represents a search result with similarity score
type SearchResult struct {
	EmailID         string    `json:"email_id"`
	Subject         string    `json:"subject"`
	Sender          string    `json:"sender"`
	Preview         string    `json:"preview"`
	SimilarityScore float64   `json:"similarity_score"`
	ProcessedAt     time.Time `json:"processed_at"`
	PriorityScore   float64   `json:"priority_score"`
}

// HealthStatus represents system health information
type HealthStatus struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Timestamp time.Time         `json:"timestamp"`
	Readiness bool              `json:"readiness"`
	Services  map[string]string `json:"services,omitempty"`
}

// APIError represents a standardized API error response
type APIError struct {
	Error     string      `json:"error"`
	Message   string      `json:"message,omitempty"`
	Details   interface{} `json:"details,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// CreateAccountRequest represents the request to create a new email account
type CreateAccountRequest struct {
	EmailAddress string `json:"email_address" validate:"required,email"`
	IMAPServer   string `json:"imap_server" validate:"required"`
	IMAPPort     int    `json:"imap_port" validate:"required,min=1,max=65535"`
	SMTPServer   string `json:"smtp_server" validate:"required"`
	SMTPPort     int    `json:"smtp_port" validate:"required,min=1,max=65535"`
	Password     string `json:"password" validate:"required"`
	UseTLS       bool   `json:"use_tls"`
}

// CreateRuleRequest represents the request to create a new triage rule
type CreateRuleRequest struct {
	Description       string                   `json:"description" validate:"required"`
	UseAIGeneration   bool                     `json:"use_ai_generation"`
	ManualConditions  []RuleCondition         `json:"manual_conditions,omitempty"`
	Actions           []RuleAction            `json:"actions" validate:"required"`
	Priority          int                     `json:"priority"`
}

// CreateRuleResponse represents the response after creating a rule
type CreateRuleResponse struct {
	Success             bool            `json:"success"`
	RuleID              string          `json:"rule_id"`
	GeneratedConditions []RuleCondition `json:"generated_conditions,omitempty"`
	AIConfidence        float64         `json:"ai_confidence,omitempty"`
	PreviewMatches      int             `json:"preview_matches"`
}

// SearchEmailsResponse represents the response for email search
type SearchEmailsResponse struct {
	Results     []SearchResult `json:"results"`
	Total       int            `json:"total"`
	QueryTimeMs int            `json:"query_time_ms"`
	Page        int            `json:"page"`
	Limit       int            `json:"limit"`
}

// DashboardStats represents dashboard analytics data
type DashboardStats struct {
	EmailsProcessed   int                    `json:"emails_processed"`
	RulesActive       int                    `json:"rules_active"`
	ActionsAutomated  int                    `json:"actions_automated"`
	AverageProcessTime float64              `json:"average_process_time_ms"`
	TopRules          []RulePerformance      `json:"top_rules"`
	RecentActivity    []ActivityLog          `json:"recent_activity"`
	UsageStats        UsageStatistics        `json:"usage_stats"`
}

// RulePerformance represents performance metrics for a rule
type RulePerformance struct {
	RuleID      string  `json:"rule_id"`
	RuleName    string  `json:"rule_name"`
	MatchCount  int     `json:"match_count"`
	SuccessRate float64 `json:"success_rate"`
	AverageTime float64 `json:"average_time_ms"`
}

// ActivityLog represents a recent activity entry
type ActivityLog struct {
	Timestamp   time.Time   `json:"timestamp"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Details     interface{} `json:"details,omitempty"`
}

// UsageStatistics represents usage limits and current consumption
type UsageStatistics struct {
	PlanType              string `json:"plan_type"`
	EmailAccountsUsed     int    `json:"email_accounts_used"`
	EmailAccountsLimit    int    `json:"email_accounts_limit"`
	MonthlyEmailsProcessed int   `json:"monthly_emails_processed"`
	MonthlyEmailsLimit     int   `json:"monthly_emails_limit"`
	RulesUsed             int    `json:"rules_used"`
	RulesLimit            int    `json:"rules_limit"`
}