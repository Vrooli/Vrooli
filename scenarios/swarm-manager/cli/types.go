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
	Creates         []string `json:"creates,omitempty"`
	SpawnedFrom     string   `json:"spawned_from,omitempty"`
}

type BacklogItemResponse struct {
	Item BacklogItem `json:"item"`
}

type ListBacklogResponse struct {
	Items []BacklogItem `json:"items"`
}

type PendingQuestion struct {
	ID            string `json:"id"`
	Source        string `json:"source"`
	ItemKind      string `json:"item_kind"`
	ItemName      string `json:"item_name"`
	Title         string `json:"title,omitempty"`
	Description   string `json:"description,omitempty"`
	Criticality   string `json:"criticality,omitempty"`
	ReviewStatus  string `json:"review_status,omitempty"`
	ReviewComment string `json:"review_comment,omitempty"`
	ReviewType    string `json:"review_type,omitempty"`
	ModuleID      string `json:"module_id,omitempty"`
}

type PendingQuestionsItem struct {
	Kind      string            `json:"kind"`
	Name      string            `json:"name"`
	Questions []PendingQuestion `json:"questions"`
}

type PendingQuestionsResponse struct {
	Items []PendingQuestionsItem `json:"items"`
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
	Creates         []string `json:"creates,omitempty"`
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
	Creates         *[]string `json:"creates,omitempty"`
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
		r.AcceptanceDeny == nil &&
		r.Creates == nil
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
	ForceableBlockingReasons []string                  `json:"forceable_blocking_reasons,omitempty"`
	Advisories               []string                  `json:"advisories,omitempty"`
}

type ProcessBlockingQuestion struct {
	ID         string `json:"id,omitempty"`
	Importance string `json:"importance,omitempty"`
	Question   string `json:"question,omitempty"`
}

// BlockingReason is a structured blocking reason with forceability metadata.
type BlockingReason struct {
	Message   string `json:"message"`
	Forceable bool   `json:"forceable"`
}

type QueueBacklogResponse struct {
	Item                BacklogItem      `json:"item"`
	TaskID              string           `json:"task_id"`
	RunID               string           `json:"run_id"`
	BaseURL             string           `json:"base_url"`
	Created             string           `json:"created"`
	DryRun              bool             `json:"dry_run,omitempty"`
	Queued              bool             `json:"queued,omitempty"`
	Message             string           `json:"message,omitempty"`
	BlockingReasons     []BlockingReason `json:"blocking_reasons,omitempty"`
	UnansweredQuestions int              `json:"unanswered_questions,omitempty"`
	PendingSuggestions  int              `json:"pending_suggestions,omitempty"`
	Advisories          []string         `json:"advisories,omitempty"`
}

type ResearchResponse struct {
	TaskID          string           `json:"task_id"`
	RunID           string           `json:"run_id"`
	BaseURL         string           `json:"base_url"`
	Created         string           `json:"created"`
	DryRun          bool             `json:"dry_run,omitempty"`
	Started         bool             `json:"started,omitempty"`
	Message         string           `json:"message,omitempty"`
	BlockingReasons []BlockingReason `json:"blocking_reasons,omitempty"`
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

// ScenarioFix mirrors the API ScenarioFix shape returned by
// GET /scenarios/{name}/context.
type ScenarioFix struct {
	Name       string  `json:"name"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	Priority   int     `json:"priority"`
	Initiative string  `json:"initiative,omitempty"`
	Updated    string  `json:"updated,omitempty"`
	ArchivedAt *string `json:"archived_at,omitempty"`
	Path       string  `json:"path"`
}

// ScenarioFixHistory mirrors the API ScenarioFixHistory shape.
type ScenarioFixHistory struct {
	Active   []ScenarioFix `json:"active"`
	Archived []ScenarioFix `json:"archived"`
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

// ScenarioReviewQueueItem represents a single scenario in the review queue.
type ScenarioReviewQueueItem struct {
	ScenarioName             string  `json:"scenario_name"`
	PendingBacklogCount      int     `json:"pending_backlog_count"`
	LastReviewClassification string  `json:"last_review_classification,omitempty"`
	LastReviewAt             string  `json:"last_review_at,omitempty"`
	RecentExecutionCount     int     `json:"recent_execution_count"`
	CompositeScore           float64 `json:"composite_score"`
	PrimarySignal            string  `json:"primary_signal"`
	CooldownUntil            string  `json:"cooldown_until,omitempty"`
}

// ScenarioReviewQueueResponse is the response from the review-queue endpoint.
type ScenarioReviewQueueResponse struct {
	Items          []ScenarioReviewQueueItem `json:"items"`
	TotalScenarios int                       `json:"total_scenarios"`
	ExcludedCount  int                       `json:"excluded_count"`
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
	Name               string   `json:"name"`
	Title              string   `json:"title"`
	Description        string   `json:"description,omitempty"`
	Status             string   `json:"status"`
	Mode               string   `json:"mode,omitempty"`
	Priority           int      `json:"priority,omitempty"`
	DependsOn          []string `json:"depends_on,omitempty"`
	Items              []string `json:"items"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	Created            string   `json:"created"`
	Updated            string   `json:"updated"`
	SpawnedFrom        string   `json:"spawned_from,omitempty"`
}

// InitiativeRollup provides aggregated status counts for initiative items.
type InitiativeRollup struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	InProgress int `json:"in_progress"`
	Failed     int `json:"failed"`
	Pending    int `json:"pending"`
}

// InitiativeResponse wraps a single initiative with rollup status and the
// deduped scenarios its member items target.
type InitiativeResponse struct {
	Initiative      Initiative       `json:"initiative"`
	Rollup          InitiativeRollup `json:"rollup"`
	TargetScenarios []string         `json:"target_scenarios,omitempty"`
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
	Priority    int      `json:"priority,omitempty"`
	DependsOn   []string `json:"depends_on,omitempty"`
	Items       []string `json:"items,omitempty"`
}

type InitiativeUpdateRequest struct {
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	Status      *string   `json:"status,omitempty"`
	Priority    *int      `json:"priority,omitempty"`
	DependsOn   *[]string `json:"depends_on,omitempty"`
	Items       *[]string `json:"items,omitempty"`
}

func (r InitiativeUpdateRequest) HasChanges() bool {
	return r.Title != nil || r.Description != nil || r.Status != nil ||
		r.Priority != nil || r.DependsOn != nil || r.Items != nil
}

// InitiativeContextItem is the compact member-item view returned inside the
// initiative context payload.
type InitiativeContextItem struct {
	Kind       string   `json:"kind"`
	Name       string   `json:"name"`
	Title      string   `json:"title"`
	Status     string   `json:"status"`
	Priority   int      `json:"priority"`
	DependsOn  []string `json:"depends_on,omitempty"`
	Initiative string   `json:"initiative,omitempty"`
	ArchivedAt *string  `json:"archived_at,omitempty"`
}

// ScenarioContextRollup aggregates completion stats across every item
// (initiative-assigned or orphan) targeting a scenario.
type ScenarioContextRollup struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	InProgress int `json:"in_progress"`
	Failed     int `json:"failed"`
	Pending    int `json:"pending"`
	Archived   int `json:"archived"`
}

// ScenarioContextOrphanItem is a backlog item targeting a scenario but not
// assigned to any initiative. Orphans signal that a readiness-style umbrella
// initiative may be warranted.
type ScenarioContextOrphanItem struct {
	Kind       string  `json:"kind"`
	Name       string  `json:"name"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	Priority   int     `json:"priority"`
	ArchivedAt *string `json:"archived_at,omitempty"`
}

// ScenarioContextResponse is the full coverage view for a scenario: every
// initiative whose items target the scenario, every orphan item targeting
// the scenario, and a combined completion rollup.
type ScenarioContextResponse struct {
	ScenarioName string                      `json:"scenario_name"`
	Initiatives  []InitiativeResponse        `json:"initiatives"`
	OrphanItems  []ScenarioContextOrphanItem `json:"orphan_items"`
	Rollup       ScenarioContextRollup       `json:"rollup"`
	Fixes        ScenarioFixHistory          `json:"fixes"`
}

// InitiativeContextResponse pairs an initiative with its immediate
// neighborhood for single-call loading by agents and the CLI.
type InitiativeContextResponse struct {
	Initiative            Initiative              `json:"initiative"`
	Rollup                InitiativeRollup        `json:"rollup"`
	Items                 []InitiativeContextItem `json:"items"`
	UpstreamInitiatives   []Initiative            `json:"upstream_initiatives"`
	DownstreamInitiatives []Initiative            `json:"downstream_initiatives"`
	TargetScenarios       []string                `json:"target_scenarios,omitempty"`
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
