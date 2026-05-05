package teamcontract

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	SchemaVersion = 1

	BaseRepoRoot   = "repo-root"
	BaseTeamRoot   = "team-root"
	BaseTeamShared = "team-shared"
	BaseTeamMember = "team-member"
	BaseAgentRoot  = "agent-root"

	DecisionModeYolo     = "yolo"
	DecisionModeApproval = "approval"

	TeamWorkingStateKindCharter            = "charter"
	TeamWorkingStateKindTaskBoard          = "task-board"
	TeamWorkingStateKindDecisionLog        = "decision-log"
	TeamWorkingStateKindKnowledgeLog       = "knowledge-log"
	TeamWorkingStateKindHandoffLog         = "handoff-log"
	TeamWorkingStateKindWorkingRegister    = "working-register"
	TeamWorkingStateKindRollingSnapshot    = "rolling-snapshot"
	TeamWorkingStateKindAppendOnlyEventLog = "append-only-event-log"
	TeamWorkingStateKindOperatorInput      = "operator-input"
)

type TeamWorkingStateKind struct {
	ID         string
	Label      string
	UseText    string
	UpdateMode string
}

var teamWorkingStateKinds = map[string]TeamWorkingStateKind{
	TeamWorkingStateKindCharter: {
		ID:         TeamWorkingStateKindCharter,
		Label:      "Charter",
		UseText:    "team charter and durable team-specific principles",
		UpdateMode: "operator/team curated",
	},
	TeamWorkingStateKindTaskBoard: {
		ID:         TeamWorkingStateKindTaskBoard,
		Label:      "Task board",
		UseText:    "live team tasks and coordination state",
		UpdateMode: "mutable",
	},
	TeamWorkingStateKindDecisionLog: {
		ID:         TeamWorkingStateKindDecisionLog,
		Label:      "Decision log",
		UseText:    "reviewable proposed changes",
		UpdateMode: "append/update via decision commands",
	},
	TeamWorkingStateKindKnowledgeLog: {
		ID:         TeamWorkingStateKindKnowledgeLog,
		Label:      "Knowledge log",
		UseText:    "structured observations, snapshots, and friction signals",
		UpdateMode: "append/supersede by topic",
	},
	TeamWorkingStateKindHandoffLog: {
		ID:         TeamWorkingStateKindHandoffLog,
		Label:      "Handoff log",
		UseText:    "historical handoff archive",
		UpdateMode: "automatic append",
	},
	TeamWorkingStateKindWorkingRegister: {
		ID:         TeamWorkingStateKindWorkingRegister,
		Label:      "Working register",
		UseText:    "current operational list or register",
		UpdateMode: "append or update rows",
	},
	TeamWorkingStateKindRollingSnapshot: {
		ID:         TeamWorkingStateKindRollingSnapshot,
		Label:      "Rolling snapshot",
		UseText:    "current summarized view of recent evidence",
		UpdateMode: "replace/update section or row",
	},
	TeamWorkingStateKindAppendOnlyEventLog: {
		ID:         TeamWorkingStateKindAppendOnlyEventLog,
		Label:      "Append-only event log",
		UseText:    "structured historical events or observations owned by the team",
		UpdateMode: "append-only",
	},
	TeamWorkingStateKindOperatorInput: {
		ID:         TeamWorkingStateKindOperatorInput,
		Label:      "Operator input",
		UseText:    "operator-maintained inputs or state that agents may read and only assigned owners may maintain",
		UpdateMode: "operator-maintained or assigned owner",
	},
}

type OperatingContract struct {
	SchemaVersion   int                        `json:"schemaVersion"`
	Governance      Governance                 `json:"governance"`
	Documents       Documents                  `json:"documents"`
	DecisionContext map[string]DecisionContext `json:"decisionContexts"`
	KnowledgeTopics map[string]KnowledgeTopic  `json:"knowledgeTopics"`
	Members         map[string]MemberContract  `json:"members"`
}

func TeamWorkingStateKindMetadata(kind string) (TeamWorkingStateKind, bool) {
	meta, ok := teamWorkingStateKinds[strings.TrimSpace(kind)]
	return meta, ok
}

func TeamWorkingStateKindIDs() []string {
	ids := make([]string, 0, len(teamWorkingStateKinds))
	for id := range teamWorkingStateKinds {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

type Governance struct {
	DecisionMode        string               `json:"decisionMode"`
	TeamPendingCeiling  TeamPendingCeiling   `json:"teamPendingCeiling"`
	Supersession        SupersessionPolicy   `json:"supersession"`
	StaleDecisionPolicy *StaleDecisionPolicy `json:"staleDecisionPolicy,omitempty"`
}

type TeamPendingCeiling struct {
	Value                 int    `json:"value"`
	ReadOnlyWhenAtOrAbove bool   `json:"readOnlyWhenAtOrAbove"`
	Rationale             string `json:"rationale,omitempty"`
}

type SupersessionPolicy struct {
	RequiredBeforeNewDecision    bool `json:"requiredBeforeNewDecision"`
	AllowedInReadOnlyMode        bool `json:"allowedInReadOnlyMode"`
	ReplacementMustSetSupersedes bool `json:"replacementMustSetSupersedes"`
}

type StaleDecisionPolicy struct {
	AfterHeartbeats  int      `json:"afterHeartbeats"`
	OwnerMemberID    string   `json:"ownerMemberId"`
	RequiredOutcomes []string `json:"requiredOutcomes"`
}

type Documents struct {
	PlanOfRecord []PlanOfRecordDocument `json:"planOfRecord"`
	Notebooks    []NotebookDocument     `json:"notebooks"`
	SharedState  []SharedStateDocument  `json:"sharedState"`
}

type PlanOfRecordDocument struct {
	ID             string    `json:"id"`
	Hub            *PathRef  `json:"hub,omitempty"`
	Paths          []PathRef `json:"paths"`
	WritePolicy    string    `json:"writePolicy"`
	Consumers      []string  `json:"consumers,omitempty"`
	Rationale      string    `json:"rationale,omitempty"`
	UseFor         string    `json:"useFor,omitempty"`
	Required       *bool     `json:"required,omitempty"`
	OptionalReason string    `json:"optionalReason,omitempty"`
}

type NotebookDocument struct {
	ID               string    `json:"id"`
	Paths            []PathRef `json:"paths"`
	Posture          string    `json:"posture,omitempty"`
	WritePolicy      string    `json:"writePolicy"`
	CuratorMemberID  string    `json:"curatorMemberId"`
	PromotionContext string    `json:"promotionContext"`
	Required         *bool     `json:"required,omitempty"`
	OptionalReason   string    `json:"optionalReason,omitempty"`
}

type SharedStateDocument struct {
	ID             string  `json:"id"`
	Path           PathRef `json:"path"`
	OwnerMemberID  string  `json:"ownerMemberId,omitempty"`
	Kind           string  `json:"kind"`
	Required       bool    `json:"required"`
	OptionalReason string  `json:"optionalReason,omitempty"`
}

type DecisionContext struct {
	OwnerMemberIDs            []string `json:"ownerMemberIds,omitempty"`
	Description               string   `json:"description,omitempty"`
	ExternalAuthorizedRaisers []string `json:"externalAuthorizedRaisers,omitempty"`
}

type KnowledgeTopic struct {
	OwnerMemberID      string `json:"ownerMemberId"`
	SupersedesPrevious bool   `json:"supersedesPrevious"`
	Retention          string `json:"retention,omitempty"`
}

// MemberContract describes the team-level half of a member's runtime
// contract: decision-graph position, write surfaces, safety rules, and
// read-only-mode behavior. Per-member message-flow declarations
// (intake/required_read/evidence_consumed/output) live in topics.json,
// not here — see api/memberflow/schema.go for the canonical declaration
// and api/memberflow/validation.go for the rules that enforce it.
type MemberContract struct {
	Lane                       string           `json:"lane"`
	OwnedDecisionContexts      []string         `json:"ownedDecisionContexts"`
	NewDecisionCapPerHeartbeat *int             `json:"newDecisionCapPerHeartbeat,omitempty"`
	NewDecisionCapsByContext   map[string]int   `json:"newDecisionCapsByContext,omitempty"`
	PendingOwnedDecisionCap    *int             `json:"pendingOwnedDecisionCap,omitempty"`
	AllowedWrites              []WriteRef       `json:"allowedWrites,omitempty"`
	ForbiddenWrites            []WriteRef       `json:"forbiddenWrites,omitempty"`
	SafetyCriticalRules        []string         `json:"safetyCriticalRules,omitempty"`
	ReadOnlyModeBehavior       ReadOnlyBehavior `json:"readOnlyModeBehavior"`
	TaskParameters             map[string]any   `json:"taskParameters,omitempty"`
}

type ReadOnlyBehavior struct {
	SkipNewDecisions     bool `json:"skipNewDecisions"`
	StillWriteKnowledge  bool `json:"stillWriteKnowledge"`
	StillRunSupersession bool `json:"stillRunSupersession"`
	StillWriteHandoff    bool `json:"stillWriteHandoff"`
}

type PathRef struct {
	Base           string `json:"base,omitempty"`
	Path           string `json:"path,omitempty"`
	MemberID       string `json:"memberId,omitempty"`
	AgentID        string `json:"agentId,omitempty"`
	Required       *bool  `json:"required,omitempty"`
	OptionalReason string `json:"optionalReason,omitempty"`
}

type WriteRef struct {
	Base           string `json:"base,omitempty"`
	Path           string `json:"path,omitempty"`
	Kind           string `json:"kind,omitempty"`
	MemberID       string `json:"memberId,omitempty"`
	AgentID        string `json:"agentId,omitempty"`
	Required       *bool  `json:"required,omitempty"`
	OptionalReason string `json:"optionalReason,omitempty"`
}

type ValidationInput struct {
	TeamID       string
	DecisionMode string
	MemberIDs    []string
	StoreDir     string
	RepoRoot     string
}

type RenderInput struct {
	TeamID         string
	TeamName       string
	DecisionMode   string
	MemberID       string
	StoreDir       string
	RepoRoot       string
	RequireHandoff bool
}

func Minimal(decisionMode string, memberIDs ...string) *OperatingContract {
	if decisionMode == "" {
		decisionMode = DecisionModeYolo
	}
	contextID := "general"
	topicID := "heartbeat-note"
	members := make(map[string]MemberContract, len(memberIDs))
	for _, memberID := range memberIDs {
		if strings.TrimSpace(memberID) == "" {
			continue
		}
		cap := 1
		pendingCap := 3
		members[memberID] = MemberContract{
			Lane:                       "Apply the team mission within this member's assigned scope.",
			OwnedDecisionContexts:      []string{contextID},
			NewDecisionCapPerHeartbeat: &cap,
			PendingOwnedDecisionCap:    &pendingCap,
			AllowedWrites:              []WriteRef{{Kind: "knowledge"}, {Kind: "decision"}, {Kind: "handoff"}},
			ReadOnlyModeBehavior: ReadOnlyBehavior{
				SkipNewDecisions:     true,
				StillWriteKnowledge:  true,
				StillRunSupersession: true,
				StillWriteHandoff:    true,
			},
		}
	}
	return &OperatingContract{
		SchemaVersion: SchemaVersion,
		Governance: Governance{
			DecisionMode:       decisionMode,
			TeamPendingCeiling: TeamPendingCeiling{Value: 12, ReadOnlyWhenAtOrAbove: true},
			Supersession: SupersessionPolicy{
				RequiredBeforeNewDecision:    true,
				AllowedInReadOnlyMode:        true,
				ReplacementMustSetSupersedes: true,
			},
		},
		Documents: Documents{},
		DecisionContext: map[string]DecisionContext{
			contextID: {OwnerMemberIDs: compactMemberIDs(memberIDs), Description: "General team decision proposals."},
		},
		KnowledgeTopics: map[string]KnowledgeTopic{
			topicID: {OwnerMemberID: firstMemberID(memberIDs), SupersedesPrevious: false, Retention: "append-only"},
		},
		Members: members,
	}
}

func compactMemberIDs(memberIDs []string) []string {
	out := make([]string, 0, len(memberIDs))
	seen := map[string]struct{}{}
	for _, id := range memberIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return []string{"unassigned"}
	}
	return out
}

func firstMemberID(memberIDs []string) string {
	for _, id := range memberIDs {
		if strings.TrimSpace(id) != "" {
			return strings.TrimSpace(id)
		}
	}
	return "unassigned"
}

func Validate(contract *OperatingContract, input ValidationInput) error {
	if contract == nil {
		return fmt.Errorf("operatingContract is required")
	}
	if contract.SchemaVersion != SchemaVersion {
		return fmt.Errorf("operatingContract.schemaVersion must equal %d", SchemaVersion)
	}
	if strings.TrimSpace(contract.Governance.DecisionMode) == "" {
		return fmt.Errorf("operatingContract.governance.decisionMode is required")
	}
	if input.DecisionMode != "" && contract.Governance.DecisionMode != input.DecisionMode {
		return fmt.Errorf("operatingContract.governance.decisionMode %q must match team decisionMode %q", contract.Governance.DecisionMode, input.DecisionMode)
	}
	if contract.Governance.DecisionMode != DecisionModeYolo && contract.Governance.DecisionMode != DecisionModeApproval {
		return fmt.Errorf("operatingContract.governance.decisionMode must be 'yolo' or 'approval'")
	}
	if contract.Governance.TeamPendingCeiling.Value < 0 {
		return fmt.Errorf("operatingContract.governance.teamPendingCeiling.value must be non-negative")
	}
	if contract.DecisionContext == nil {
		return fmt.Errorf("operatingContract.decisionContexts is required")
	}
	if contract.KnowledgeTopics == nil {
		return fmt.Errorf("operatingContract.knowledgeTopics is required")
	}
	if contract.Members == nil {
		return fmt.Errorf("operatingContract.members is required")
	}
	for _, memberID := range input.MemberIDs {
		memberID = strings.TrimSpace(memberID)
		if memberID == "" {
			continue
		}
		if _, ok := contract.Members[memberID]; !ok {
			return fmt.Errorf("operatingContract.members missing active member %q", memberID)
		}
	}

	for id, member := range contract.Members {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("operatingContract.members contains an empty member id")
		}
		if member.NewDecisionCapPerHeartbeat != nil && *member.NewDecisionCapPerHeartbeat < 0 {
			return fmt.Errorf("operatingContract.members.%s.newDecisionCapPerHeartbeat must be non-negative", id)
		}
		if member.PendingOwnedDecisionCap != nil && *member.PendingOwnedDecisionCap < 0 {
			return fmt.Errorf("operatingContract.members.%s.pendingOwnedDecisionCap must be non-negative", id)
		}
		for contextID, cap := range member.NewDecisionCapsByContext {
			if cap < 0 {
				return fmt.Errorf("operatingContract.members.%s.newDecisionCapsByContext.%s must be non-negative", id, contextID)
			}
			if _, ok := contract.DecisionContext[contextID]; !ok {
				return fmt.Errorf("operatingContract.members.%s.newDecisionCapsByContext.%s is not declared in decisionContexts", id, contextID)
			}
		}
		for _, contextID := range member.OwnedDecisionContexts {
			if _, ok := contract.DecisionContext[contextID]; !ok {
				return fmt.Errorf("operatingContract.members.%s.ownedDecisionContexts contains undeclared context %q", id, contextID)
			}
		}
		if err := validateWriteRefs(member.AllowedWrites, input, id, "allowedWrites"); err != nil {
			return err
		}
		if err := validateWriteRefs(member.ForbiddenWrites, input, id, "forbiddenWrites"); err != nil {
			return err
		}
	}
	for contextID, dc := range contract.DecisionContext {
		if len(dc.OwnerMemberIDs) == 0 && len(dc.ExternalAuthorizedRaisers) == 0 {
			return fmt.Errorf("operatingContract.decisionContexts.%s requires ownerMemberIds or externalAuthorizedRaisers", contextID)
		}
		for _, ownerID := range dc.OwnerMemberIDs {
			if _, ok := contract.Members[ownerID]; !ok {
				return fmt.Errorf("operatingContract.decisionContexts.%s owner %q is not a contract member", contextID, ownerID)
			}
		}
	}
	if policy := contract.Governance.StaleDecisionPolicy; policy != nil {
		if policy.AfterHeartbeats < 1 {
			return fmt.Errorf("operatingContract.governance.staleDecisionPolicy.afterHeartbeats must be at least 1")
		}
		if _, ok := contract.Members[policy.OwnerMemberID]; !ok {
			return fmt.Errorf("operatingContract.governance.staleDecisionPolicy.ownerMemberId %q is not a contract member", policy.OwnerMemberID)
		}
		if len(policy.RequiredOutcomes) == 0 {
			return fmt.Errorf("operatingContract.governance.staleDecisionPolicy.requiredOutcomes is required")
		}
	}
	return validateDocuments(contract, input)
}

func RenderMemberPolicy(contract *OperatingContract, input RenderInput) (string, error) {
	if err := Validate(contract, ValidationInput{
		TeamID: input.TeamID, DecisionMode: input.DecisionMode, MemberIDs: []string{input.MemberID}, StoreDir: input.StoreDir, RepoRoot: input.RepoRoot,
	}); err != nil {
		return "", err
	}
	member, ok := contract.Members[input.MemberID]
	if !ok {
		return "", fmt.Errorf("operatingContract.members missing active member %q", input.MemberID)
	}

	var b strings.Builder
	renderMemberPolicyBody(&b, contract, member, input)
	return strings.TrimRight(b.String(), "\n") + "\n", nil
}

func renderMemberPolicyBody(b *strings.Builder, contract *OperatingContract, member MemberContract, input RenderInput) {
	b.WriteString("\n## Governance\n\n")
	b.WriteString(fmt.Sprintf("Decision mode: %s\n", contract.Governance.DecisionMode))
	b.WriteString(fmt.Sprintf("Pending decision ceiling: %d\n", contract.Governance.TeamPendingCeiling.Value))
	if contract.Governance.TeamPendingCeiling.ReadOnlyWhenAtOrAbove {
		b.WriteString(fmt.Sprintf("When pending decisions are >= %d:\n", contract.Governance.TeamPendingCeiling.Value))
		if member.ReadOnlyModeBehavior.SkipNewDecisions {
			b.WriteString("- skip new decision creation\n")
		}
		if member.ReadOnlyModeBehavior.StillWriteKnowledge {
			b.WriteString("- still write required knowledge snapshots\n")
		}
		if member.ReadOnlyModeBehavior.StillRunSupersession {
			b.WriteString("- still perform supersession when it shrinks the queue\n")
		}
		if member.ReadOnlyModeBehavior.StillWriteHandoff {
			b.WriteString("- still write HANDOFF\n")
		}
	}
	if policy := contract.Governance.StaleDecisionPolicy; policy != nil && policy.OwnerMemberID == input.MemberID {
		b.WriteString(fmt.Sprintf("Stale decision scan: review pending decisions older than %d heartbeats; outcomes: %s.\n", policy.AfterHeartbeats, strings.Join(policy.RequiredOutcomes, ", ")))
	}
	b.WriteString("\n## Your Member Contract\n\n")
	b.WriteString(fmt.Sprintf("Agent ID: %s\n", input.MemberID))
	if member.Lane != "" {
		b.WriteString(fmt.Sprintf("Lane: %s\n", member.Lane))
	}
	writeStringList(b, "Owned decision contexts", member.OwnedDecisionContexts)
	b.WriteString("\nDecision caps:\n")
	if member.NewDecisionCapPerHeartbeat != nil {
		b.WriteString(fmt.Sprintf("- max new decisions this heartbeat: %d\n", *member.NewDecisionCapPerHeartbeat))
	}
	if len(member.NewDecisionCapsByContext) > 0 {
		keys := sortedKeys(member.NewDecisionCapsByContext)
		for _, contextID := range keys {
			b.WriteString(fmt.Sprintf("- %s: max %d new decisions this heartbeat\n", contextID, member.NewDecisionCapsByContext[contextID]))
		}
	}
	if member.PendingOwnedDecisionCap != nil {
		b.WriteString(fmt.Sprintf("- skip new decisions when %d+ owned-context decisions are already pending\n", *member.PendingOwnedDecisionCap))
	}
	b.WriteString("\n## Document Authority\n\n")
	renderDocuments(b, contract, input)
	b.WriteString("\n## Write Rules\n\n")
	renderWriteRefs(b, "Allowed writes", member.AllowedWrites, input)
	renderWriteRefs(b, "Forbidden writes", member.ForbiddenWrites, input)
	writeStringList(b, "Safety-critical rules", member.SafetyCriticalRules)
	if len(member.TaskParameters) > 0 {
		b.WriteString("\nTask parameters:\n")
		for _, key := range sortedAnyKeys(member.TaskParameters) {
			b.WriteString(fmt.Sprintf("- %s: %v\n", key, member.TaskParameters[key]))
		}
	}
}

func NormalizePath(ref PathRef, input ValidationInput, activeMemberID string) (string, error) {
	base := strings.TrimSpace(ref.Base)
	path := strings.TrimSpace(ref.Path)
	if base == "" {
		return "", fmt.Errorf("path base is required")
	}
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("path %q must be relative", path)
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || clean == "" || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", fmt.Errorf("path %q must not escape its base", path)
	}
	memberID := ref.MemberID
	if memberID == "" {
		memberID = activeMemberID
	}
	agentID := ref.AgentID
	if agentID == "" {
		agentID = activeMemberID
	}
	switch base {
	case BaseRepoRoot:
		return clean, nil
	case BaseTeamRoot:
		return filepath.ToSlash(filepath.Join("scenarios/prompt-manager/store/teams", input.TeamID, clean)), nil
	case BaseTeamShared:
		return filepath.ToSlash(filepath.Join("scenarios/prompt-manager/store/teams", input.TeamID, "shared", clean)), nil
	case BaseTeamMember:
		if memberID == "" {
			return "", fmt.Errorf("team-member path %q requires memberId or active member context", path)
		}
		return filepath.ToSlash(filepath.Join("scenarios/prompt-manager/store/teams", input.TeamID, "members", memberID, clean)), nil
	case BaseAgentRoot:
		if agentID == "" {
			return "", fmt.Errorf("agent-root path %q requires agentId or active member context", path)
		}
		return filepath.ToSlash(filepath.Join("scenarios/prompt-manager/store/agents", agentID, clean)), nil
	default:
		return "", fmt.Errorf("unsupported path base %q", base)
	}
}

func validateDocuments(contract *OperatingContract, input ValidationInput) error {
	for _, doc := range contract.Documents.PlanOfRecord {
		if strings.TrimSpace(doc.ID) == "" {
			return fmt.Errorf("operatingContract.documents.planOfRecord contains a document without id")
		}
		if strings.TrimSpace(doc.WritePolicy) == "" {
			return fmt.Errorf("operatingContract.documents.planOfRecord.%s.writePolicy is required", doc.ID)
		}
		if len(doc.Consumers) == 0 && strings.TrimSpace(doc.Rationale) == "" {
			return fmt.Errorf("operatingContract.documents.planOfRecord.%s requires consumers or rationale", doc.ID)
		}
		if len(doc.Paths) > 1 && doc.Hub == nil {
			return fmt.Errorf("operatingContract.documents.planOfRecord.%s.hub is required when multiple paths are declared", doc.ID)
		}
		if err := validatePathRefs(doc.Paths, docRequired(doc.Required), doc.OptionalReason, input, "", "planOfRecord."+doc.ID); err != nil {
			return err
		}
		if doc.Hub != nil {
			if err := validatePathRefs([]PathRef{*doc.Hub}, docRequired(doc.Required), doc.OptionalReason, input, "", "planOfRecord."+doc.ID+".hub"); err != nil {
				return err
			}
			if !planOfRecordHubInPaths(doc, input) {
				return fmt.Errorf("operatingContract.documents.planOfRecord.%s.hub must also be listed in paths", doc.ID)
			}
		}
	}
	for _, doc := range contract.Documents.Notebooks {
		if strings.TrimSpace(doc.ID) == "" {
			return fmt.Errorf("operatingContract.documents.notebooks contains a document without id")
		}
		if strings.TrimSpace(doc.CuratorMemberID) == "" || strings.TrimSpace(doc.PromotionContext) == "" {
			return fmt.Errorf("operatingContract.documents.notebooks.%s requires curatorMemberId and promotionContext", doc.ID)
		}
		if _, ok := contract.Members[doc.CuratorMemberID]; !ok {
			return fmt.Errorf("operatingContract.documents.notebooks.%s curatorMemberId %q is not a contract member", doc.ID, doc.CuratorMemberID)
		}
		if _, ok := contract.DecisionContext[doc.PromotionContext]; !ok {
			return fmt.Errorf("operatingContract.documents.notebooks.%s promotionContext %q is not declared", doc.ID, doc.PromotionContext)
		}
		if err := validatePathRefs(doc.Paths, docRequired(doc.Required), doc.OptionalReason, input, "", "notebooks."+doc.ID); err != nil {
			return err
		}
	}
	for _, doc := range contract.Documents.SharedState {
		if strings.TrimSpace(doc.ID) == "" {
			return fmt.Errorf("operatingContract.documents.sharedState contains a document without id")
		}
		if _, ok := TeamWorkingStateKindMetadata(doc.Kind); !ok {
			return fmt.Errorf("operatingContract.documents.sharedState.%s kind %q is not a supported team working state kind", doc.ID, doc.Kind)
		}
		if doc.OwnerMemberID != "" {
			if _, ok := contract.Members[doc.OwnerMemberID]; !ok {
				return fmt.Errorf("operatingContract.documents.sharedState.%s ownerMemberId %q is not a contract member", doc.ID, doc.OwnerMemberID)
			}
		}
		if err := validatePathRefs([]PathRef{doc.Path}, doc.Required, doc.OptionalReason, input, "", "sharedState."+doc.ID); err != nil {
			return err
		}
	}
	return nil
}

func planOfRecordHubInPaths(doc PlanOfRecordDocument, input ValidationInput) bool {
	if doc.Hub == nil {
		return true
	}
	hubPath, err := NormalizePath(*doc.Hub, input, "")
	if err != nil {
		return false
	}
	for _, ref := range doc.Paths {
		path, err := NormalizePath(ref, input, "")
		if err == nil && path == hubPath {
			return true
		}
	}
	return false
}

func validatePathRefs(paths []PathRef, required bool, optionalReason string, input ValidationInput, activeMemberID, field string) error {
	if len(paths) == 0 {
		return fmt.Errorf("operatingContract.documents.%s requires at least one path", field)
	}
	if !required && strings.TrimSpace(optionalReason) == "" {
		return fmt.Errorf("operatingContract.documents.%s optional paths require optionalReason", field)
	}
	for _, ref := range paths {
		normalized, err := NormalizePath(ref, input, activeMemberID)
		if err != nil {
			return fmt.Errorf("operatingContract.documents.%s: %w", field, err)
		}
		if required {
			if err := validateExists(normalized, input); err != nil {
				return fmt.Errorf("operatingContract.documents.%s: %w", field, err)
			}
		}
	}
	return nil
}

func validateWriteRefs(refs []WriteRef, input ValidationInput, activeMemberID, field string) error {
	for _, ref := range refs {
		if ref.Kind != "" {
			switch ref.Kind {
			case "handoff", "decision", "knowledge", "task", "inbox-message":
				continue
			default:
				return fmt.Errorf("operatingContract.members.%s contains unsupported write kind %q", field, ref.Kind)
			}
		}
		_, err := NormalizePath(PathRef{Base: ref.Base, Path: ref.Path, MemberID: ref.MemberID, AgentID: ref.AgentID}, input, activeMemberID)
		if err != nil {
			return fmt.Errorf("operatingContract.members.%s: %w", field, err)
		}
	}
	return nil
}

func validateExists(repoRelative string, input ValidationInput) error {
	root := strings.TrimSpace(input.RepoRoot)
	if root == "" {
		root = deriveRepoRoot(input.StoreDir)
	}
	if root == "" {
		return nil
	}
	full := filepath.Join(root, filepath.FromSlash(repoRelative))
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	cleanFull, err := filepath.Abs(full)
	if err != nil {
		return err
	}
	if cleanFull != cleanRoot && !strings.HasPrefix(cleanFull, cleanRoot+string(os.PathSeparator)) {
		return fmt.Errorf("normalized path %q escapes repo root", repoRelative)
	}
	if _, err := os.Stat(cleanFull); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("required path %q does not exist", repoRelative)
		}
		return fmt.Errorf("stat required path %q: %w", repoRelative, err)
	}
	return nil
}

func deriveRepoRoot(storeDir string) string {
	if storeDir == "" {
		return ""
	}
	abs, err := filepath.Abs(storeDir)
	if err != nil {
		return ""
	}
	if filepath.Base(abs) == "store" && filepath.Base(filepath.Dir(abs)) == "prompt-manager" {
		return filepath.Dir(filepath.Dir(filepath.Dir(abs)))
	}
	return ""
}

func docRequired(v *bool) bool {
	return v == nil || *v
}

func writeStringList(b *strings.Builder, title string, values []string) {
	if len(values) == 0 {
		return
	}
	b.WriteString("\n" + title + ":\n")
	for _, value := range values {
		b.WriteString("- " + value + "\n")
	}
}

func renderDocuments(b *strings.Builder, contract *OperatingContract, input RenderInput) {
	validationInput := ValidationInput{TeamID: input.TeamID, DecisionMode: input.DecisionMode, StoreDir: input.StoreDir, RepoRoot: input.RepoRoot}
	if len(contract.Documents.PlanOfRecord) > 0 {
		b.WriteString("Plan of record authorities:\n")
		for _, doc := range contract.Documents.PlanOfRecord {
			hub, err := planOfRecordHubPath(doc, validationInput, input.MemberID)
			if err != nil {
				continue
			}
			b.WriteString("- " + hub + "\n")
			b.WriteString(fmt.Sprintf("  Policy: %s\n", doc.WritePolicy))
			if consumers := storageConsumerLine(doc.Consumers, doc.Rationale); consumers != "" {
				b.WriteString(fmt.Sprintf("  Consumers: %s\n", consumers))
			}
			if useFor := storageUseForLine(doc); useFor != "" {
				b.WriteString(fmt.Sprintf("  Use for: %s\n", useFor))
			}
			if len(doc.Paths) > 1 {
				b.WriteString(fmt.Sprintf("  Coverage: %d declared files; start at the hub and follow its file map to the relevant spoke.\n", len(doc.Paths)))
			}
		}
		b.WriteString("\n")
	}
	if len(contract.Documents.Notebooks) > 0 {
		b.WriteString("Notebook docs:\n")
		for _, doc := range contract.Documents.Notebooks {
			for _, ref := range doc.Paths {
				if p, err := NormalizePath(ref, validationInput, input.MemberID); err == nil {
					b.WriteString("- " + p + "\n")
				}
			}
		}
		b.WriteString("\nNotebook rules:\n")
		for _, doc := range contract.Documents.Notebooks {
			b.WriteString(fmt.Sprintf("- %s: writePolicy=%s, curator=%s, promotionContext=%s\n", doc.ID, doc.WritePolicy, doc.CuratorMemberID, doc.PromotionContext))
		}
		b.WriteString("\n")
	}
	if len(contract.Documents.SharedState) > 0 {
		b.WriteString("Team working state:\n")
		for _, doc := range contract.Documents.SharedState {
			if p, err := NormalizePath(doc.Path, validationInput, input.MemberID); err == nil {
				b.WriteString("- " + p + "\n")
			}
		}
	}
}

func RenderTeamStorage(contract *OperatingContract, input RenderInput) (string, error) {
	if err := Validate(contract, ValidationInput{
		TeamID: input.TeamID, DecisionMode: input.DecisionMode, MemberIDs: []string{input.MemberID}, StoreDir: input.StoreDir, RepoRoot: input.RepoRoot,
	}); err != nil {
		return "", err
	}

	validationInput := ValidationInput{TeamID: input.TeamID, DecisionMode: input.DecisionMode, StoreDir: input.StoreDir, RepoRoot: input.RepoRoot}
	var b strings.Builder
	b.WriteString("## Your Team Storage\n\n")

	if len(contract.Documents.PlanOfRecord) > 0 {
		b.WriteString("Plan of record, read/propose only:\n")
		renderStoragePlanOfRecordHubs(&b, contract.Documents.PlanOfRecord, validationInput, input.MemberID)
		b.WriteString("\n")
	}

	if len(contract.Documents.Notebooks) > 0 {
		b.WriteString("Notebook, append unresolved learning:\n")
		for _, doc := range contract.Documents.Notebooks {
			posture := strings.TrimSpace(doc.Posture)
			if posture == "" {
				posture = "debt"
			}
			for _, ref := range doc.Paths {
				p, err := NormalizePath(ref, validationInput, input.MemberID)
				if err != nil {
					return "", err
				}
				b.WriteString(fmt.Sprintf("- `%s`\n", p))
				b.WriteString(fmt.Sprintf("  Curator: `%s`\n", doc.CuratorMemberID))
				b.WriteString(fmt.Sprintf("  Promotion context: `%s`\n", doc.PromotionContext))
				b.WriteString(fmt.Sprintf("  Posture: `%s`\n", posture))
			}
		}
		b.WriteString("\n")
	}

	if len(contract.Documents.SharedState) > 0 {
		b.WriteString("Team working state:\n")
		for _, doc := range contract.Documents.SharedState {
			p, err := NormalizePath(doc.Path, validationInput, input.MemberID)
			if err != nil {
				return "", err
			}
			meta, ok := TeamWorkingStateKindMetadata(doc.Kind)
			if !ok {
				return "", fmt.Errorf("operatingContract.documents.sharedState.%s kind %q is not a supported team working state kind", doc.ID, doc.Kind)
			}
			owner := strings.TrimSpace(doc.OwnerMemberID)
			if owner == "" {
				owner = "team"
			}
			b.WriteString(fmt.Sprintf("- `%s`\n", p))
			b.WriteString(fmt.Sprintf("  Kind: `%s`\n", doc.Kind))
			b.WriteString(fmt.Sprintf("  Owner: `%s`\n", owner))
			b.WriteString(fmt.Sprintf("  Use for: %s\n", meta.UseText))
		}
		b.WriteString("\n")
	}

	member := contract.Members[input.MemberID]
	b.WriteString("Primitive availability for this member:\n")
	b.WriteString("- decisions: " + primitiveDecisionAvailability(member) + "\n")
	b.WriteString("- knowledge: " + primitiveKnowledgeAvailability(member) + "\n")
	b.WriteString("- handoff: " + primitiveHandoffAvailability(member, input.RequireHandoff) + "\n")
	b.WriteString("- task board: " + primitiveTaskAvailability(member) + "\n")
	return strings.TrimRight(b.String(), "\n") + "\n", nil
}

func primitiveDecisionAvailability(member MemberContract) string {
	if writeRefsContainKind(member.ForbiddenWrites, "decision") || (member.NewDecisionCapPerHeartbeat != nil && *member.NewDecisionCapPerHeartbeat == 0) {
		return "`review-only` - review pending decisions when useful; do not create decisions from this heartbeat"
	}
	if writeRefsContainKind(member.AllowedWrites, "decision") || writeRefsContainPathSuffix(member.AllowedWrites, "decisions.jsonl") {
		return "`write-allowed` - propose reviewable changes within your owned contexts and caps"
	}
	return "`unavailable` - no decision surface is declared for this member"
}

func primitiveKnowledgeAvailability(member MemberContract) string {
	if writeRefsContainKind(member.ForbiddenWrites, "knowledge") {
		return "`unavailable` - do not write team knowledge from this heartbeat"
	}
	if writeRefsContainKind(member.AllowedWrites, "knowledge") || writeRefsContainPathSuffix(member.AllowedWrites, "knowledge.jsonl") {
		return "`write-allowed` - record structured observations and friction signals using required topics"
	}
	return "`unavailable` - no knowledge surface is declared for this member"
}

func primitiveHandoffAvailability(member MemberContract, requireHandoff bool) string {
	if writeRefsContainKind(member.ForbiddenWrites, "handoff") {
		return "`unavailable` - do not write a persistent handoff"
	}
	if writeRefsContainKind(member.AllowedWrites, "handoff") {
		if requireHandoff {
			return "`required` - preserve next-run continuity with final ## HANDOFF"
		}
		return "`allowed` - preserve next-run continuity when useful"
	}
	return "`unavailable` - no handoff surface is declared for this member"
}

func primitiveTaskAvailability(member MemberContract) string {
	if writeRefsContainKind(member.ForbiddenWrites, "task") {
		return "`review-only` - review task board context when useful; do not update tasks"
	}
	if writeRefsContainKind(member.AllowedWrites, "task") {
		return "`write-allowed` - maintain live team work only when your task asks for it"
	}
	return "`review-only` - review task board context when useful; no task write surface is declared"
}

func writeRefsContainKind(refs []WriteRef, kind string) bool {
	for _, ref := range refs {
		if ref.Kind == kind {
			return true
		}
	}
	return false
}

func writeRefsContainPathSuffix(refs []WriteRef, suffix string) bool {
	for _, ref := range refs {
		if ref.Kind == "" && strings.HasSuffix(filepath.ToSlash(strings.TrimSpace(ref.Path)), suffix) {
			return true
		}
	}
	return false
}

func renderStoragePlanOfRecordHubs(b *strings.Builder, docs []PlanOfRecordDocument, input ValidationInput, activeMemberID string) {
	rows := make([]storagePlanOfRecordRow, 0, len(docs))
	for _, doc := range docs {
		hub, err := planOfRecordHubPath(doc, input, activeMemberID)
		if err != nil {
			continue
		}
		rows = append(rows, storagePlanOfRecordRow{
			Hub:       hub,
			Policy:    doc.WritePolicy,
			Consumers: storageConsumerLine(doc.Consumers, doc.Rationale),
			UseFor:    storageUseForLine(doc),
		})
	}
	for _, row := range rows {
		b.WriteString(fmt.Sprintf("- `%s`\n", row.Hub))
		b.WriteString(fmt.Sprintf("  Policy: `%s`\n", row.Policy))
		if row.Consumers != "" {
			b.WriteString(fmt.Sprintf("  Consumers: `%s`\n", row.Consumers))
		}
		if row.UseFor != "" {
			b.WriteString(fmt.Sprintf("  Use for: %s\n", row.UseFor))
		}
		b.WriteString("  Navigation: start at the hub and follow its file map to the relevant spoke.\n")
	}
}

type storagePlanOfRecordRow struct {
	Hub       string
	Policy    string
	Consumers string
	UseFor    string
}

func planOfRecordHubPath(doc PlanOfRecordDocument, input ValidationInput, activeMemberID string) (string, error) {
	if doc.Hub != nil {
		return NormalizePath(*doc.Hub, input, activeMemberID)
	}
	if len(doc.Paths) == 0 {
		return "", fmt.Errorf("plan-of-record document %q has no paths", doc.ID)
	}
	return NormalizePath(doc.Paths[0], input, activeMemberID)
}

func storageUseForLine(doc PlanOfRecordDocument) string {
	if useFor := strings.TrimSpace(doc.UseFor); useFor != "" {
		return useFor
	}
	if rationale := strings.TrimSpace(doc.Rationale); rationale != "" {
		return rationale
	}
	return storageConsumerLine(doc.Consumers, "")
}

func storageConsumerLine(consumers []string, rationale string) string {
	if len(consumers) > 0 {
		return strings.Join(consumers, ", ")
	}
	return strings.TrimSpace(rationale)
}

func renderWriteRefs(b *strings.Builder, title string, refs []WriteRef, input RenderInput) {
	if len(refs) == 0 {
		return
	}
	validationInput := ValidationInput{TeamID: input.TeamID, DecisionMode: input.DecisionMode, StoreDir: input.StoreDir, RepoRoot: input.RepoRoot}
	b.WriteString(title + ":\n")
	for _, ref := range refs {
		if ref.Kind != "" {
			b.WriteString("- " + ref.Kind + "\n")
			continue
		}
		if p, err := NormalizePath(PathRef{Base: ref.Base, Path: ref.Path, MemberID: ref.MemberID, AgentID: ref.AgentID}, validationInput, input.MemberID); err == nil {
			b.WriteString("- " + p + "\n")
		}
	}
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedAnyKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
