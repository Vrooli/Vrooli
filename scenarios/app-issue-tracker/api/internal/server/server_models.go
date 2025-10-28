package server

import (
	automationpkg "app-issue-tracker-api/internal/automation"
	issuespkg "app-issue-tracker-api/internal/issues"
)

const (
	metadataFilename = issuespkg.MetadataFilename
	artifactsDirName = issuespkg.ArtifactsDirName

	// Issue status constants
	StatusOpen      = issuespkg.StatusOpen
	StatusActive    = issuespkg.StatusActive
	StatusCompleted = issuespkg.StatusCompleted
	StatusFailed    = issuespkg.StatusFailed
	StatusArchived  = issuespkg.StatusArchived
)

func ValidIssueStatuses() []string {
	return issuespkg.ValidStatuses()
}

func NormalizeIssueStatus(status string) (string, bool) {
	return issuespkg.NormalizeStatus(status)
}

func IsValidIssueStatus(status string) bool {
	return issuespkg.IsValidStatus(status)
}

type Issue = issuespkg.Issue
type Attachment = issuespkg.Attachment
type PromptPreviewRequest = issuespkg.PromptPreviewRequest
type PromptPreviewResponse = issuespkg.PromptPreviewResponse
type UpdateIssueRequest = issuespkg.UpdateIssueRequest
type CreateIssueRequest = issuespkg.CreateIssueRequest
type ArtifactPayload = issuespkg.ArtifactPayload
type LegacyAttachmentPayload = issuespkg.LegacyAttachmentPayload
type AgentConversationEntry = issuespkg.AgentConversationEntry
type AgentConversationPayload = issuespkg.AgentConversationPayload
type Agent = issuespkg.Agent
type App = issuespkg.App

func cloneStringSlice(values []string) []string {
	return issuespkg.CloneStringSlice(values)
}

func normalizeStringSlice(values []string) []string {
	return issuespkg.NormalizeStringSlice(values)
}

func cloneStringMap(values map[string]string) map[string]string {
	return issuespkg.CloneStringMap(values)
}

type Config struct {
	Port                    string
	QdrantURL               string
	IssuesDir               string
	ScenarioRoot            string
	WebsocketAllowedOrigins []string
}

const (
	AgentStatusRunning    = automationpkg.AgentStatusRunning
	AgentStatusCancelling = automationpkg.AgentStatusCancelling
	AgentStatusCompleted  = automationpkg.AgentStatusCompleted
	AgentStatusFailed     = automationpkg.AgentStatusFailed
	AgentStatusCancelled  = automationpkg.AgentStatusCancelled

	AgentStatusExtraKey          = automationpkg.AgentStatusExtraKey
	AgentStatusTimestampExtraKey = automationpkg.AgentStatusTimestampExtraKey
)

type ProcessorState = automationpkg.ProcessorState

type RunningProcess = automationpkg.RunningProcess

type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
