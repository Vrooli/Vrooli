// Package services provides business logic orchestration.
// This file defines types used by the prompt sync service.
package services

// Skill represents a knowledge module that provides methodology and expertise.
// Skills are injected into the agent's context to enhance specific tasks.
type Skill struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Icon         string   `json:"icon,omitempty"`
	Modes        []string `json:"modes,omitempty"` // Hierarchical path like ["Architecture", "Audits"]
	Content      string   `json:"content"`
	Tags         []string `json:"tags,omitempty"`
	TargetToolID string   `json:"targetToolId,omitempty"` // Optional tool this skill targets
	Draft        bool     `json:"draft,omitempty"`        // Indicates skill may not be fully working
}

// SkillResponse is a skill with additional metadata for API responses.
type SkillResponse struct {
	Skill
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// SkillListResponse is the response for listing skills.
type SkillListResponse struct {
	Skills []SkillResponse `json:"skills"`
	Count  int             `json:"count"`
}

// PromptResponse is the response from prompt-manager for a single prompt.
type PromptResponse struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	Content             string   `json:"content"`
	Modes               []string `json:"modes"`
	Tags                []string `json:"tags"`
	Icon                string   `json:"icon,omitempty"`
	TargetToolID        *string  `json:"targetToolId,omitempty"`
	Draft               bool     `json:"draft"`
	Folder              string   `json:"folder"`
	CreatedAt           string   `json:"createdAt"`
	UpdatedAt           string   `json:"updatedAt"`
	UsageCount          int      `json:"usageCount"`
	LastUsed            *string  `json:"lastUsed,omitempty"`
	EffectivenessRating *int     `json:"effectivenessRating,omitempty"`
}

// SyncResponse is the response from prompt-manager's sync endpoint.
type SyncResponse struct {
	Skills      []PromptResponse `json:"skills"`
	LastUpdated string           `json:"lastUpdated"`
	Hash        string           `json:"hash"`
}

// SkillOverride represents a local override for a prompt's skill properties.
type SkillOverride struct {
	PromptID     string  `json:"promptId"`
	Icon         string  `json:"icon,omitempty"`
	TargetToolID *string `json:"targetToolId,omitempty"`
}

// SkillsConfigFile represents the config/skills.json file structure.
type SkillsConfigFile struct {
	PromptManagerURL    string          `json:"promptManagerUrl"`
	SyncIntervalSeconds int             `json:"syncIntervalSeconds"`
	SkillOverrides      []SkillOverride `json:"skillOverrides"`
}

// SyncStatus contains the result of a sync operation.
type SyncStatus struct {
	Success    bool   `json:"success"`
	SkillCount int    `json:"skillCount"`
	LocalCount int    `json:"localCount"`
	Hash       string `json:"hash"`
	Error      string `json:"error,omitempty"`
}

// CreateSkillRequest is the request to create a skill in prompt-manager.
type CreateSkillRequest struct {
	ID           string   `json:"id,omitempty"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Content      string   `json:"content"`
	Modes        []string `json:"modes,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Icon         string   `json:"icon,omitempty"`
	Draft        bool     `json:"draft,omitempty"`
	Folder       string   `json:"folder"`
	TargetToolID string   `json:"-"` // Not sent to prompt-manager, stored locally
}
