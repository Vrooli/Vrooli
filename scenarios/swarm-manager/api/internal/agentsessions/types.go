// Package agentsessions defines the Swarm Manager contract for durable,
// typed Agent Manager conversations.
package agentsessions

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Kind string

const (
	KindMetaOrchestration      Kind = "meta_orchestration"
	KindOperatingModeAuthoring Kind = "operating_mode_authoring"
	KindSwarmOperations        Kind = "swarm_operations"
)

type Status string

const (
	StatusDraft          Status = "draft"
	StatusStarting       Status = "starting"
	StatusRunning        Status = "running"
	StatusWaitingForUser Status = "waiting_for_user"
	StatusProposalReady  Status = "proposal_ready"
	StatusApplying       Status = "applying"
	StatusComplete       Status = "complete"
	StatusFailed         Status = "failed"
	StatusCanceled       Status = "canceled"
)

type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
)

type ProposalKind string

const (
	ProposalBacklogBatchImport              ProposalKind = "backlog_batch_import"
	ProposalOperatingModeDraft              ProposalKind = "operating_mode_draft"
	ProposalOperatingModeImplementationPlan ProposalKind = "operating_mode_implementation_plan"
)

type ProposalStatus string

const (
	ProposalStatusDraft      ProposalStatus = "draft"
	ProposalStatusReady      ProposalStatus = "ready"
	ProposalStatusApplied    ProposalStatus = "applied"
	ProposalStatusRejected   ProposalStatus = "rejected"
	ProposalStatusSuperseded ProposalStatus = "superseded"
	ProposalStatusFailed     ProposalStatus = "failed"
)

type ArtifactType string

const (
	ArtifactBacklogItem             ArtifactType = "backlog_item"
	ArtifactInitiative              ArtifactType = "initiative"
	ArtifactOperatingModeProposal   ArtifactType = "operating_mode_proposal"
	ArtifactOperatingModeDefinition ArtifactType = "operating_mode_definition"
	ArtifactCapture                 ArtifactType = "capture"
	ArtifactFile                    ArtifactType = "file"
	ArtifactAgentActivity           ArtifactType = "agent_activity"
)

type ArtifactAction string

const (
	ArtifactActionProposed ArtifactAction = "proposed"
	ArtifactActionCreated  ArtifactAction = "created"
	ArtifactActionUpdated  ArtifactAction = "updated"
	ArtifactActionDeleted  ArtifactAction = "deleted"
	ArtifactActionLinked   ArtifactAction = "linked"
)

type ContextType string

const (
	ContextBacklogItem        ContextType = "backlog_item"
	ContextInitiative         ContextType = "initiative"
	ContextCapture            ContextType = "capture"
	ContextExecution          ContextType = "execution"
	ContextAgentActivity      ContextType = "agent_activity"
	ContextScenario           ContextType = "scenario"
	ContextOperatingMode      ContextType = "operating_mode"
	ContextSession            ContextType = "session"
	ContextOperationsBriefing ContextType = "operations_briefing"
	ContextStartupBrief       ContextType = "startup_brief"
)

const OperationsBriefingLatestRef = "operations_briefing/latest"

const (
	StartupBriefMetaOrchestrationRef      = "startup_brief/meta_orchestration"
	StartupBriefOperatingModeAuthoringRef = "startup_brief/operating_mode_authoring"
	StartupBriefSwarmOperationsRef        = "startup_brief/swarm_operations"
)

type AutoContextPolicy string

const (
	AutoContextDefault AutoContextPolicy = "default"
	AutoContextNone    AutoContextPolicy = "none"
)

type AttributionType string

const (
	AttributionOperator AttributionType = "operator"
	AttributionAgent    AttributionType = "agent"
)

var ErrValidation = errors.New("agent session validation")

type Attribution struct {
	Type        AttributionType `json:"type"`
	RunID       string          `json:"run_id,omitempty"`
	TaskID      string          `json:"task_id,omitempty"`
	ProfileKey  string          `json:"profile_key,omitempty"`
	SessionID   string          `json:"session_id,omitempty"`
	SessionKind Kind            `json:"session_kind,omitempty"`
	Source      string          `json:"source,omitempty"`
}

type Message struct {
	ID            string        `json:"id"`
	Role          MessageRole   `json:"role"`
	Content       string        `json:"content"`
	CreatedAt     string        `json:"created_at"`
	AttachmentIDs []string      `json:"attachment_ids,omitempty"`
	Context       []ContextItem `json:"context,omitempty"`
}

type Attachment struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type,omitempty"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type ContextRef struct {
	Type ContextType `json:"type"`
	Ref  string      `json:"ref"`
}

type ContextItem struct {
	Type         ContextType `json:"type"`
	Ref          string      `json:"ref"`
	Title        string      `json:"title"`
	Summary      string      `json:"summary"`
	NodeID       string      `json:"node_id,omitempty"`
	MetadataJSON string      `json:"metadata_json,omitempty"`
	SelectedAt   string      `json:"selected_at"`
}

type Proposal struct {
	ID          string         `json:"id"`
	Kind        ProposalKind   `json:"kind"`
	Status      ProposalStatus `json:"status"`
	Summary     string         `json:"summary"`
	PayloadJSON string         `json:"payload_json"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
	Attribution *Attribution   `json:"attribution,omitempty"`
}

type Artifact struct {
	ID             string         `json:"id"`
	SessionID      string         `json:"session_id"`
	ArtifactType   ArtifactType   `json:"artifact_type"`
	Action         ArtifactAction `json:"action"`
	EntityRef      string         `json:"entity_ref"`
	Title          string         `json:"title,omitempty"`
	ProposalID     string         `json:"proposal_id,omitempty"`
	ActivityID     string         `json:"activity_id,omitempty"`
	RunID          string         `json:"run_id,omitempty"`
	MutationSource string         `json:"mutation_source,omitempty"`
	Attribution    *Attribution   `json:"attribution,omitempty"`
	CreatedAt      string         `json:"created_at"`
}

type Session struct {
	ID            string       `json:"id"`
	Title         string       `json:"title"`
	Kind          Kind         `json:"kind"`
	Status        Status       `json:"status"`
	SkillID       string       `json:"skill_id"`
	TaskID        string       `json:"task_id,omitempty"`
	RunID         string       `json:"run_id,omitempty"`
	ProfileKey    string       `json:"profile_key,omitempty"`
	FailureReason string       `json:"failure_reason,omitempty"`
	CreatedAt     string       `json:"created_at"`
	UpdatedAt     string       `json:"updated_at"`
	Messages      []Message    `json:"messages,omitempty"`
	Proposals     []Proposal   `json:"proposals,omitempty"`
	Artifacts     []Artifact   `json:"artifacts,omitempty"`
	CreatedBy     *Attribution `json:"created_by,omitempty"`
	Attachments   []Attachment `json:"attachments,omitempty"`
}

func (s Session) Validate() error {
	if strings.TrimSpace(s.ID) == "" {
		return validationError("id is required")
	}
	if strings.TrimSpace(s.Title) == "" {
		return validationError("title is required")
	}
	if !IsKnownKind(s.Kind) {
		return validationError("kind must be meta_orchestration, operating_mode_authoring, or swarm_operations")
	}
	if !IsKnownStatus(s.Status) {
		return validationError("status is invalid")
	}
	if strings.TrimSpace(s.SkillID) == "" {
		return validationError("skill_id is required")
	}
	if err := validateRFC3339("created_at", s.CreatedAt); err != nil {
		return err
	}
	if err := validateRFC3339("updated_at", s.UpdatedAt); err != nil {
		return err
	}
	if s.CreatedBy != nil {
		if err := s.CreatedBy.Validate(); err != nil {
			return fmt.Errorf("%w: created_by: %v", ErrValidation, err)
		}
	}
	for i := range s.Messages {
		if err := s.Messages[i].Validate(); err != nil {
			return fmt.Errorf("%w: messages[%d]: %v", ErrValidation, i, err)
		}
	}
	for i := range s.Proposals {
		if err := s.Proposals[i].Validate(); err != nil {
			return fmt.Errorf("%w: proposals[%d]: %v", ErrValidation, i, err)
		}
	}
	for i := range s.Artifacts {
		if err := s.Artifacts[i].Validate(); err != nil {
			return fmt.Errorf("%w: artifacts[%d]: %v", ErrValidation, i, err)
		}
	}
	for i := range s.Attachments {
		if err := s.Attachments[i].Validate(); err != nil {
			return fmt.Errorf("%w: attachments[%d]: %v", ErrValidation, i, err)
		}
	}
	return nil
}

func (m Message) Validate() error {
	if strings.TrimSpace(m.ID) == "" {
		return validationError("id is required")
	}
	if !IsKnownMessageRole(m.Role) {
		return validationError("role is invalid")
	}
	for i := range m.Context {
		if err := m.Context[i].Validate(); err != nil {
			return fmt.Errorf("%w: context[%d]: %v", ErrValidation, i, err)
		}
	}
	return validateRFC3339("created_at", m.CreatedAt)
}

func (a Attachment) Validate() error {
	if strings.TrimSpace(a.ID) == "" {
		return validationError("id is required")
	}
	if strings.TrimSpace(a.Filename) == "" {
		return validationError("filename is required")
	}
	if a.SizeBytes < 0 {
		return validationError("size_bytes must be zero or greater")
	}
	return validateRFC3339("created_at", a.CreatedAt)
}

func (c ContextItem) Validate() error {
	if !IsKnownContextType(c.Type) {
		return validationError("context type is invalid")
	}
	if strings.TrimSpace(c.Ref) == "" {
		return validationError("context ref is required")
	}
	if strings.TrimSpace(c.Title) == "" {
		return validationError("context title is required")
	}
	if strings.TrimSpace(c.MetadataJSON) != "" && !json.Valid([]byte(c.MetadataJSON)) {
		return validationError("context metadata_json must be valid JSON")
	}
	return validateRFC3339("selected_at", c.SelectedAt)
}

func (p Proposal) Validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return validationError("id is required")
	}
	if !IsKnownProposalKind(p.Kind) {
		return validationError("kind is invalid")
	}
	if !IsKnownProposalStatus(p.Status) {
		return validationError("status is invalid")
	}
	if strings.TrimSpace(p.Summary) == "" {
		return validationError("summary is required")
	}
	if !json.Valid([]byte(p.PayloadJSON)) {
		return validationError("payload_json must be valid JSON")
	}
	if err := validateRFC3339("created_at", p.CreatedAt); err != nil {
		return err
	}
	if err := validateRFC3339("updated_at", p.UpdatedAt); err != nil {
		return err
	}
	if p.Attribution != nil {
		if err := p.Attribution.Validate(); err != nil {
			return fmt.Errorf("%w: attribution: %v", ErrValidation, err)
		}
	}
	return nil
}

func (a Artifact) Validate() error {
	if strings.TrimSpace(a.ID) == "" {
		return validationError("id is required")
	}
	if strings.TrimSpace(a.SessionID) == "" {
		return validationError("session_id is required")
	}
	if !IsKnownArtifactType(a.ArtifactType) {
		return validationError("artifact_type is invalid")
	}
	if !IsKnownArtifactAction(a.Action) {
		return validationError("action is invalid")
	}
	if strings.TrimSpace(a.EntityRef) == "" {
		return validationError("entity_ref is required")
	}
	if err := validateRFC3339("created_at", a.CreatedAt); err != nil {
		return err
	}
	if a.Attribution != nil {
		if err := a.Attribution.Validate(); err != nil {
			return fmt.Errorf("%w: attribution: %v", ErrValidation, err)
		}
	}
	return nil
}

func (a Attribution) Validate() error {
	switch a.Type {
	case AttributionOperator:
	case AttributionAgent:
		if strings.TrimSpace(a.RunID) == "" {
			return validationError("run_id is required for agent attribution")
		}
	default:
		return validationError("type is invalid")
	}
	if a.SessionKind != "" && !IsKnownKind(a.SessionKind) {
		return validationError("session_kind is invalid")
	}
	return nil
}

func IsKnownKind(kind Kind) bool {
	switch kind {
	case KindMetaOrchestration, KindOperatingModeAuthoring, KindSwarmOperations:
		return true
	default:
		return false
	}
}

func IsKnownStatus(status Status) bool {
	switch status {
	case StatusDraft, StatusStarting, StatusRunning, StatusWaitingForUser, StatusProposalReady,
		StatusApplying, StatusComplete, StatusFailed, StatusCanceled:
		return true
	default:
		return false
	}
}

func IsKnownMessageRole(role MessageRole) bool {
	switch role {
	case MessageRoleUser, MessageRoleAssistant, MessageRoleSystem:
		return true
	default:
		return false
	}
}

func IsKnownProposalKind(kind ProposalKind) bool {
	switch kind {
	case ProposalBacklogBatchImport, ProposalOperatingModeDraft, ProposalOperatingModeImplementationPlan:
		return true
	default:
		return false
	}
}

func IsKnownProposalStatus(status ProposalStatus) bool {
	switch status {
	case ProposalStatusDraft, ProposalStatusReady, ProposalStatusApplied, ProposalStatusRejected,
		ProposalStatusSuperseded, ProposalStatusFailed:
		return true
	default:
		return false
	}
}

func IsKnownArtifactType(artifactType ArtifactType) bool {
	switch artifactType {
	case ArtifactBacklogItem, ArtifactInitiative, ArtifactOperatingModeProposal,
		ArtifactOperatingModeDefinition, ArtifactCapture, ArtifactFile, ArtifactAgentActivity:
		return true
	default:
		return false
	}
}

func IsKnownArtifactAction(action ArtifactAction) bool {
	switch action {
	case ArtifactActionProposed, ArtifactActionCreated, ArtifactActionUpdated,
		ArtifactActionDeleted, ArtifactActionLinked:
		return true
	default:
		return false
	}
}

func IsKnownContextType(contextType ContextType) bool {
	switch contextType {
	case ContextBacklogItem, ContextInitiative, ContextCapture, ContextExecution,
		ContextAgentActivity, ContextScenario, ContextOperatingMode, ContextSession, ContextOperationsBriefing, ContextStartupBrief:
		return true
	default:
		return false
	}
}

func StartupBriefRefForKind(kind Kind) string {
	switch kind {
	case KindMetaOrchestration:
		return StartupBriefMetaOrchestrationRef
	case KindOperatingModeAuthoring:
		return StartupBriefOperatingModeAuthoringRef
	case KindSwarmOperations:
		return StartupBriefSwarmOperationsRef
	default:
		return ""
	}
}

func validationError(message string) error {
	return fmt.Errorf("%w: %s", ErrValidation, message)
}

func validateRFC3339(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return validationError(field + " is required")
	}
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		return fmt.Errorf("%w: %s must be RFC3339", ErrValidation, field)
	}
	return nil
}
