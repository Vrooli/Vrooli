package main

import "time"

type HealthResponse struct {
	Status     string                 `json:"status"`
	Readiness  bool                   `json:"readiness"`
	Service    string                 `json:"service,omitempty"`
	Operations map[string]interface{} `json:"operations,omitempty"`
	Timestamp  string                 `json:"timestamp,omitempty"`
}

type Draft struct {
	ID         string    `json:"id"`
	EntityType string    `json:"entity_type"`
	EntityName string    `json:"entity_name"`
	Content    string    `json:"content"`
	Owner      string    `json:"owner,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Status     string    `json:"status"`
}

type DraftListResponse struct {
	Drafts []Draft `json:"drafts"`
	Total  int     `json:"total"`
}

type ValidateRequest struct {
	EntityType string `json:"entity_type"`
	EntityName string `json:"entity_name"`
}

type AIGenerateDraftRequest struct {
	EntityType             string         `json:"entity_type"`
	EntityName             string         `json:"entity_name"`
	Content                string         `json:"content,omitempty"`
	Owner                  string         `json:"owner,omitempty"`
	Section                string         `json:"section"`
	Context                string         `json:"context"`
	Action                 string         `json:"action"`
	IncludeExistingContent *bool          `json:"include_existing_content,omitempty"`
	ReferencePRDs          []ReferencePRD `json:"reference_prds,omitempty"`
	Model                  string         `json:"model,omitempty"`
	SaveGeneratedToDraft   *bool          `json:"save_generated_to_draft,omitempty"`
}

type ReferencePRD struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type AIGenerateDraftResponse struct {
	DraftID       string `json:"draft_id"`
	EntityType    string `json:"entity_type"`
	EntityName    string `json:"entity_name"`
	Section       string `json:"section"`
	GeneratedText string `json:"generated_text"`
	Model         string `json:"model"`
	SavedToDraft  bool   `json:"saved_to_draft"`
	Success       bool   `json:"success"`
	Message       string `json:"message,omitempty"`
	DraftFilePath string `json:"draft_file_path,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
}

type PublishTemplateRequest struct {
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables"`
	Force     bool              `json:"force"`
	RunHooks  bool              `json:"run_hooks"`
}

type PublishRequest struct {
	CreateBackup bool                    `json:"create_backup"`
	DeleteDraft  bool                    `json:"delete_draft"`
	Template     *PublishTemplateRequest `json:"template,omitempty"`
}

type PublishResponse struct {
	Success         bool   `json:"success"`
	Message         string `json:"message"`
	PublishedTo     string `json:"published_to"`
	BackupPath      string `json:"backup_path,omitempty"`
	PublishedAt     string `json:"published_at"`
	DraftRemoved    bool   `json:"draft_removed,omitempty"`
	CreatedScenario bool   `json:"created_scenario,omitempty"`
	ScenarioID      string `json:"scenario_id,omitempty"`
	ScenarioType    string `json:"scenario_type,omitempty"`
	ScenarioPath    string `json:"scenario_path,omitempty"`
}
