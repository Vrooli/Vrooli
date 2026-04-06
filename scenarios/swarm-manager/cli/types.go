package main

import "encoding/json"

type healthResponse struct {
	Status     string         `json:"status"`
	Service    string         `json:"service"`
	Version    string         `json:"version"`
	Readiness  bool           `json:"readiness"`
	Timestamp  string         `json:"timestamp"`
	Deps       map[string]any `json:"dependencies"`
	Error      string         `json:"error,omitempty"`
	Message    string         `json:"message,omitempty"`
	Operations map[string]any `json:"operations,omitempty"`
}

// BacklogItem represents a tracked unit of work for the swarm.
type BacklogItem struct {
	Name            string   `json:"name"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Status          string   `json:"status"`
	Priority        int      `json:"priority"`
	Tags            []string `json:"tags"`
	Created         string   `json:"created"`
	Updated         string   `json:"updated"`
	Kind            string   `json:"kind"`
	DependsOn       []string `json:"depends_on,omitempty"`
	Initiative      string   `json:"initiative,omitempty"`
	Effort          string   `json:"effort,omitempty"`
	AcceptanceAllow []string `json:"acceptance_allow,omitempty"`
	AcceptanceDeny  []string `json:"acceptance_deny,omitempty"`
}

type BacklogItemResponse struct {
	Item BacklogItem `json:"item"`
}

type ListBacklogResponse struct {
	Items []BacklogItem `json:"items"`
}

type CreateBacklogRequest struct {
	Name            string   `json:"name"`
	Title           string   `json:"title"`
	Description     string   `json:"description,omitempty"`
	Priority        int      `json:"priority,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	Kind            string   `json:"kind"`
	DependsOn       []string `json:"depends_on,omitempty"`
	Initiative      string   `json:"initiative,omitempty"`
	Effort          string   `json:"effort,omitempty"`
	AcceptanceAllow []string `json:"acceptance_allow,omitempty"`
	AcceptanceDeny  []string `json:"acceptance_deny,omitempty"`
	SpawnedFrom     string   `json:"spawned_from,omitempty"`
}

type UpdateBacklogRequest struct {
	Title           *string   `json:"title,omitempty"`
	Description     *string   `json:"description,omitempty"`
	Status          *string   `json:"status,omitempty"`
	Priority        *int      `json:"priority,omitempty"`
	Tags            *[]string `json:"tags,omitempty"`
	DependsOn       *[]string `json:"depends_on,omitempty"`
	Initiative      *string   `json:"initiative,omitempty"`
	Effort          *string   `json:"effort,omitempty"`
	AcceptanceAllow *[]string `json:"acceptance_allow,omitempty"`
	AcceptanceDeny  *[]string `json:"acceptance_deny,omitempty"`
}

func (r UpdateBacklogRequest) Empty() bool {
	return r.Title == nil &&
		r.Description == nil &&
		r.Status == nil &&
		r.Priority == nil &&
		r.Tags == nil &&
		r.DependsOn == nil &&
		r.Initiative == nil &&
		r.Effort == nil &&
		r.AcceptanceAllow == nil &&
		r.AcceptanceDeny == nil
}

type BacklogFile struct {
	Name     string        `json:"name"`
	Path     string        `json:"path"`
	Type     string        `json:"type"`
	Size     int64         `json:"size,omitempty,string"`
	Children []BacklogFile `json:"children,omitempty"`
}

type BacklogFilesResponse struct {
	Files []BacklogFile `json:"files"`
}

type BacklogFileResponse struct {
	File BacklogFile `json:"file"`
}

type ProcessPreflightResponse struct {
	Item      BacklogItem      `json:"item"`
	Preflight ProcessPreflight `json:"preflight"`
}

type ProcessPreflight struct {
	BacklogKind              string                    `json:"backlog_kind"`
	BacklogName              string                    `json:"backlog_name"`
	Ready                    bool                      `json:"ready"`
	ArchivedRevival          bool                      `json:"archived_revival"`
	ResolvedTargetScenarioID string                    `json:"resolved_target_scenario_id,omitempty"`
	TargetScenarioExists     bool                      `json:"target_scenario_exists"`
	SuggestedOperation       string                    `json:"suggested_operation,omitempty"`
	SuggestedSteerProfileID  string                    `json:"suggested_steer_profile_id,omitempty"`
	BlockingReasons          []string                  `json:"blocking_reasons,omitempty"`
	BlockingQuestions        []ProcessBlockingQuestion `json:"blocking_questions,omitempty"`
}

type ProcessBlockingQuestion struct {
	ID         string `json:"id,omitempty"`
	Importance string `json:"importance,omitempty"`
	Question   string `json:"question,omitempty"`
}

type QueueBacklogResponse struct {
	Item                BacklogItem `json:"item"`
	TaskID              string      `json:"task_id"`
	RunID               string      `json:"run_id"`
	BaseURL             string      `json:"base_url"`
	Created             string      `json:"created"`
	DryRun              bool        `json:"dry_run,omitempty"`
	Queued              bool        `json:"queued,omitempty"`
	Message             string      `json:"message,omitempty"`
	BlockingReasons     []string    `json:"blocking_reasons,omitempty"`
	UnansweredQuestions int         `json:"unanswered_questions,omitempty"`
	PendingSuggestions  int         `json:"pending_suggestions,omitempty"`
}

type ResearchResponse struct {
	TaskID  string `json:"task_id"`
	RunID   string `json:"run_id"`
	BaseURL string `json:"base_url"`
	Created string `json:"created"`
	DryRun  bool   `json:"dry_run,omitempty"`
}

// ImportBacklogResponse reports the results of importing an edited markdown export.
type ImportBacklogResponse struct {
	DryRun  bool           `json:"dry_run"`
	Changes []ImportChange `json:"changes"`
	Errors  []string       `json:"errors"`
	Summary string         `json:"summary"`
}

// ImportChange describes a single change from an import operation.
type ImportChange struct {
	Item    string   `json:"item"`
	Action  string   `json:"action"`
	Details []string `json:"details"`
}

// Scenario represents a scenario entry in the catalog.
type Scenario struct {
	Name              string   `json:"name"`
	DisplayName       string   `json:"display_name"`
	Description       string   `json:"description"`
	Status            string   `json:"status"`
	Priority          int      `json:"priority"`
	CompletenessScore *int     `json:"completeness_score,omitempty"`
	IsGreenfield      bool     `json:"is_greenfield"`
	Tags              []string `json:"tags"`
}

type ScenarioResponse struct {
	Scenario Scenario `json:"scenario"`
}

type ListScenariosResponse struct {
	Scenarios []Scenario `json:"scenarios"`
}

type ScenarioFile struct {
	Name     string         `json:"name"`
	Path     string         `json:"path"`
	Type     string         `json:"type"`
	Size     int64          `json:"size,omitempty,string"`
	Children []ScenarioFile `json:"children,omitempty"`
}

type ScenarioFilesResponse struct {
	Files []ScenarioFile `json:"files"`
}

type DeleteScenarioResponse struct {
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
	Message  string `json:"message"`
}

// QueueItem represents a local queue entry.
type QueueItem struct {
	ID      string          `json:"id"`
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Created string          `json:"created"`
}

type QueueListResponse struct {
	Items []QueueItem `json:"items"`
}

type ExecutionRecord struct {
	ExecutionID   string `json:"execution_id"`
	BacklogKind   string `json:"backlog_kind"`
	BacklogName   string `json:"backlog_name"`
	TaskID        string `json:"task_id"`
	RunID         string `json:"run_id"`
	Status        string `json:"status"`
	Mode          string `json:"mode"`
	ScheduledAt   string `json:"scheduled_at"`
	StartedAt     string `json:"started_at"`
	FinishedAt    string `json:"finished_at"`
	FailureReason string `json:"failure_reason"`
	StartedBy     string `json:"started_by"`
	Operation     string `json:"operation"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type ExecutionListResponse struct {
	Items []ExecutionRecord `json:"items"`
}

type ExecutionItemResponse struct {
	Execution ExecutionRecord `json:"execution"`
}

type SpecSyncArchiveResponse struct {
	ExecutionID string `json:"execution_id"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

type PromptTraceResponse struct {
	Trace PromptTrace `json:"trace"`
}

type PromptTrace struct {
	Purpose        string `json:"purpose"`
	Prompt         string `json:"prompt"`
	PromptRevision string `json:"prompt_revision,omitempty"`
	UsedFallback   bool   `json:"used_fallback"`
	CapturedAt     string `json:"captured_at"`
	ExperimentID   string `json:"experiment_id,omitempty"`
	VariantID      string `json:"variant_id,omitempty"`
}

type PromptCatalogEntry struct {
	ID                string   `json:"id"`
	Title             string   `json:"title"`
	Group             string   `json:"group"`
	UsageType         string   `json:"usage_type"`
	SourceType        string   `json:"source_type"`
	Trigger           string   `json:"trigger"`
	BacklogKinds      []string `json:"backlog_kinds,omitempty"`
	Modes             []string `json:"modes,omitempty"`
	Operations        []string `json:"operations,omitempty"`
	SkillID           string   `json:"skill_id,omitempty"`
	Builder           string   `json:"builder,omitempty"`
	Purpose           string   `json:"purpose"`
	OutputPaths       []string `json:"output_paths,omitempty"`
	VariableKeys      []string `json:"variable_keys,omitempty"`
	ReferenceSkillIDs []string `json:"reference_skill_ids,omitempty"`
	ExperimentID      string   `json:"experiment_id,omitempty"`
}

type ExperimentVariantStats struct {
	VariantID       string  `json:"variantId"`
	TotalRuns       int     `json:"totalRuns"`
	ReadyCount      int     `json:"readyCount"`
	NeedsWorkCount  int     `json:"needsWorkCount"`
	FixupRate       float64 `json:"fixupRate"`
	AvgDurationSecs float64 `json:"avgDurationSecs,omitempty"`
}

type ExperimentResultsResponse struct {
	ExperimentID  string                   `json:"experimentId"`
	SkillID       string                   `json:"skillId,omitempty"`
	Status        string                   `json:"status,omitempty"`
	Variants      []ExperimentVariantStats `json:"variants"`
	TotalOutcomes int                      `json:"totalOutcomes"`
	AnalyzedAt    string                   `json:"analyzedAt"`
}

type PromptCatalogResponse struct {
	Items []PromptCatalogEntry `json:"items"`
}

type PromptSkillSummary struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	DefaultScope    string   `json:"default_scope,omitempty"`
	Draft           bool     `json:"draft"`
	UpdatedAt       string   `json:"updated_at,omitempty"`
	CreatedAt       string   `json:"created_at,omitempty"`
	UsageType       string   `json:"usage_type"`
	Groups          []string `json:"groups,omitempty"`
	TriggerCount    int      `json:"trigger_count"`
	ImpactSummary   string   `json:"impact_summary"`
	CurrentContent  string   `json:"current_content,omitempty"`
	RequiredMissing []string `json:"required_missing,omitempty"`
}

type PromptSkillsResponse struct {
	Items []PromptSkillSummary `json:"items"`
}

type PromptSkillResponse struct {
	Item PromptSkillSummary `json:"item"`
}

type PromptSkillVersion struct {
	Version   int    `json:"version"`
	Content   string `json:"content"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updatedAt"`
	CreatedBy string `json:"createdBy,omitempty"`
}

type PromptSkillVersionsResponse struct {
	SkillID  string               `json:"skillId"`
	Current  int                  `json:"current"`
	Versions []PromptSkillVersion `json:"versions"`
}

type PromptPreviewResponse struct {
	SkillID   string            `json:"skill_id"`
	WithScope bool              `json:"with_scope"`
	Variables map[string]string `json:"variables,omitempty"`
	Prompt    string            `json:"prompt"`
}

type PromptSimulateResponse struct {
	EntryID   string            `json:"entry_id"`
	Group     string            `json:"group"`
	UsageType string            `json:"usage_type"`
	Kind      string            `json:"kind"`
	Mode      string            `json:"mode,omitempty"`
	SkillID   string            `json:"skill_id"`
	Variables map[string]string `json:"variables,omitempty"`
	Prompt    string            `json:"prompt"`
}

type AgentManagerStatusResponse struct {
	Enabled   bool    `json:"enabled"`
	Available bool    `json:"available"`
	URL       *string `json:"url,omitempty"`
	ProfileID *string `json:"profile_id,omitempty"`
}

// AgentManagerRunResponse represents the status of an agent-manager run.
type AgentManagerRunResponse struct {
	RunID           string  `json:"run_id"`
	TaskID          string  `json:"task_id,omitempty"`
	Status          string  `json:"status"`
	StartedAt       string  `json:"started_at,omitempty"`
	FinishedAt      string  `json:"finished_at,omitempty"`
	ErrorMessage    string  `json:"error_message,omitempty"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
	Active          bool    `json:"active"`
}

// AgentManagerStopResponse represents the result of stopping a run.
type AgentManagerStopResponse struct {
	RunID   string `json:"run_id"`
	Stopped bool   `json:"stopped"`
	Status  string `json:"status"`
}

// Initiative represents a named grouping of backlog items.
type Initiative struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status"`
	Items       []string `json:"items"`
	Created     string   `json:"created"`
	Updated     string   `json:"updated"`
}

// InitiativeRollup provides aggregated status counts for initiative items.
type InitiativeRollup struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	InProgress int `json:"in_progress"`
	Failed     int `json:"failed"`
	Pending    int `json:"pending"`
}

// InitiativeResponse wraps a single initiative with rollup status.
type InitiativeResponse struct {
	Initiative Initiative       `json:"initiative"`
	Rollup     InitiativeRollup `json:"rollup"`
}

// ListInitiativesResponse wraps the initiative list endpoint response.
type ListInitiativesResponse struct {
	Items []InitiativeResponse `json:"items"`
}

type InitiativeCreateRequest struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status,omitempty"`
	Items       []string `json:"items,omitempty"`
}

type InitiativeUpdateRequest struct {
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	Status      *string   `json:"status,omitempty"`
	Items       *[]string `json:"items,omitempty"`
}

func (r InitiativeUpdateRequest) HasChanges() bool {
	return r.Title != nil || r.Description != nil || r.Status != nil || r.Items != nil
}

// Capture represents a quick-capture entry.
type Capture struct {
	ID             string          `json:"id"`
	Text           string          `json:"text"`
	Attachments    []string        `json:"attachments,omitempty"`
	Created        string          `json:"created"`
	Status         string          `json:"status"`
	Classification *Classification `json:"classification,omitempty"`
}

// Classification contains AI-generated backlog item suggestions.
type Classification struct {
	Items        []ClassificationItem `json:"items"`
	ClassifiedAt string               `json:"classified_at"`
}

// ClassificationItem is a single suggested backlog item from classification.
type ClassificationItem struct {
	Kind        string   `json:"kind"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    int      `json:"priority"`
	Tags        []string `json:"tags"`
	Confidence  float64  `json:"confidence"`
}

// ListCapturesResponse wraps the captures list endpoint response.
type ListCapturesResponse struct {
	Captures []Capture `json:"captures"`
}

// CaptureResponse wraps a single capture endpoint response.
type CaptureResponse struct {
	Capture Capture `json:"capture"`
}

// CaptureCreateResponse wraps the create capture endpoint response.
type CaptureCreateResponse struct {
	Capture Capture `json:"capture"`
	TaskID  string  `json:"task_id,omitempty"`
	RunID   string  `json:"run_id,omitempty"`
	BaseURL string  `json:"base_url,omitempty"`
}
