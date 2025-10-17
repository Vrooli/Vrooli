package server

import issuespkg "app-issue-tracker-api/internal/issues"

const (
	metadataFilename = issuespkg.MetadataFilename
	artifactsDirName = issuespkg.ArtifactsDirName
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
	Port         string
	QdrantURL    string
	IssuesDir    string
	ScenarioRoot string
}

type ProcessorState struct {
	Active            bool `json:"active"`
	ConcurrentSlots   int  `json:"concurrent_slots"`
	RefreshInterval   int  `json:"refresh_interval"`
	CurrentlyRunning  int  `json:"currently_running"`
	MaxIssues         int  `json:"max_issues"`
	MaxIssuesDisabled bool `json:"max_issues_disabled"`
}

type RunningProcess struct {
	IssueID   string `json:"issue_id"`
	AgentID   string `json:"agent_id"`
	StartTime string `json:"start_time"`
}

type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
