package teamcontract

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"prompt-manager/finding"
)

const (
	SchemaVersion = 1

	BaseRepoRoot   = "repo-root"
	BaseTeamRoot   = "team-root"
	BaseTeamShared = "team-shared"
	BaseTeamMember = "team-member"
	BaseAgentRoot  = "agent-root"

	TeamWorkingStateKindCharter            = "charter"
	TeamWorkingStateKindTaskBoard          = "task-board"
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
	SchemaVersion   int                       `json:"schemaVersion"`
	Governance      Governance                `json:"governance"`
	Documents       Documents                 `json:"documents"`
	KnowledgeTopics map[string]KnowledgeTopic `json:"knowledgeTopics"`
	Members         map[string]MemberContract `json:"members"`
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

type Governance struct{}

type Documents struct {
	PlanOfRecord []PlanOfRecordDocument `json:"planOfRecord"`
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

type SharedStateDocument struct {
	ID             string  `json:"id"`
	Path           PathRef `json:"path"`
	OwnerMemberID  string  `json:"ownerMemberId,omitempty"`
	Kind           string  `json:"kind"`
	Required       bool    `json:"required"`
	OptionalReason string  `json:"optionalReason,omitempty"`
}

type KnowledgeTopic struct {
	OwnerMemberID      string `json:"ownerMemberId"`
	SupersedesPrevious bool   `json:"supersedesPrevious"`
	Retention          string `json:"retention,omitempty"`
}

// MemberContract describes the team-level half of a member's runtime
// contract: work-graph position, write surfaces, safety rules, and
// read-only-mode behavior. Per-member message-flow declarations
// (intake/required_read/evidence_consumed/output) live in topics.json,
// not here — see api/memberflow/schema.go for the canonical declaration
// and api/memberflow/validation.go for the rules that enforce it.
type MemberContract struct {
	Lane                 string           `json:"lane"`
	AllowedWrites        []WriteRef       `json:"allowedWrites,omitempty"`
	ForbiddenWrites      []WriteRef       `json:"forbiddenWrites,omitempty"`
	SafetyCriticalRules  []string         `json:"safetyCriticalRules,omitempty"`
	ReadOnlyModeBehavior ReadOnlyBehavior `json:"readOnlyModeBehavior"`
	TaskParameters       map[string]any   `json:"taskParameters,omitempty"`
}

type ReadOnlyBehavior struct {
	StillWriteKnowledge bool `json:"stillWriteKnowledge"`
	StillWriteHandoff   bool `json:"stillWriteHandoff"`
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
	TeamID    string
	MemberIDs []string
	StoreDir  string
	RepoRoot  string
}

type RenderInput struct {
	TeamID         string
	TeamName       string
	MemberID       string
	StoreDir       string
	RepoRoot       string
	RequireHandoff bool
}

func Minimal(_ string, memberIDs ...string) *OperatingContract {
	topicID := "heartbeat-note"
	members := make(map[string]MemberContract, len(memberIDs))
	for _, memberID := range memberIDs {
		if strings.TrimSpace(memberID) == "" {
			continue
		}
		members[memberID] = MemberContract{
			Lane:          "Apply the team mission within this member's assigned scope.",
			AllowedWrites: []WriteRef{{Kind: "knowledge"}, {Kind: "handoff"}},
			ReadOnlyModeBehavior: ReadOnlyBehavior{
				StillWriteKnowledge: true,
				StillWriteHandoff:   true,
			},
		}
	}
	return &OperatingContract{
		SchemaVersion: SchemaVersion,
		Governance:    Governance{},
		Documents:     Documents{},
		KnowledgeTopics: map[string]KnowledgeTopic{
			topicID: {OwnerMemberID: firstMemberID(memberIDs), SupersedesPrevious: false, Retention: "append-only"},
		},
		Members: members,
	}
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
	findings := ValidateFindings(contract, input)
	if len(findings) == 0 {
		return nil
	}
	return fmt.Errorf("%s", findings[0].Detail)
}

// ValidateFindings reports all independent contract defects that can be
// evaluated from the supplied document. It intentionally avoids fail-fast
// control flow: the read path must leave a malformed team visible and give
// its operator the whole repair list in one pass.
func ValidateFindings(contract *OperatingContract, input ValidationInput) []finding.Finding {
	var findings []finding.Finding
	add := func(rule, field, format string, args ...any) {
		entry, ok := contractRuleCatalog[rule]
		severity := finding.SeverityError
		if ok {
			severity = entry.Severity
		}
		findings = append(findings, finding.Finding{
			Rule:     rule,
			Severity: severity,
			Kind:     finding.KindDeclaration,
			Team:     input.TeamID,
			Path:     field,
			Detail:   fmt.Sprintf(format, args...),
		})
	}
	if contract == nil {
		add("contract_missing", "operatingContract", "operatingContract is required")
		return findings
	}
	if contract.SchemaVersion != SchemaVersion {
		add("contract_schema_version_invalid", "operatingContract.schemaVersion", "operatingContract.schemaVersion must equal %d", SchemaVersion)
	}
	if contract.KnowledgeTopics == nil {
		add("contract_knowledge_topics_missing", "operatingContract.knowledgeTopics", "operatingContract.knowledgeTopics is required")
	}
	if contract.Members == nil {
		add("contract_members_missing", "operatingContract.members", "operatingContract.members is required")
	}
	for _, memberID := range input.MemberIDs {
		memberID = strings.TrimSpace(memberID)
		if memberID == "" {
			continue
		}
		if _, ok := contract.Members[memberID]; !ok {
			add("contract_member_absent", "operatingContract.members", "operatingContract.members missing active member %q", memberID)
		}
	}

	for id, member := range contract.Members {
		if strings.TrimSpace(id) == "" {
			add("contract_member_id_empty", "operatingContract.members", "operatingContract.members contains an empty member id")
		}
		if err := validateWriteRefs(member.AllowedWrites, input, id, "allowedWrites"); err != nil {
			add("contract_member_allowed_writes_invalid", "operatingContract.members."+id+".allowedWrites", "%s", err)
		}
		if err := validateWriteRefs(member.ForbiddenWrites, input, id, "forbiddenWrites"); err != nil {
			add("contract_member_forbidden_writes_invalid", "operatingContract.members."+id+".forbiddenWrites", "%s", err)
		}
	}
	if err := validateDocuments(contract, input); err != nil {
		add("contract_documents_invalid", "operatingContract.documents", "%s", err)
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Detail < findings[j].Detail
	})
	return findings
}

func RenderMemberPolicy(contract *OperatingContract, input RenderInput) (string, error) {
	if err := Validate(contract, ValidationInput{
		TeamID: input.TeamID, MemberIDs: []string{input.MemberID}, StoreDir: input.StoreDir, RepoRoot: input.RepoRoot,
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
	b.WriteString("\n## Your Member Contract\n\n")
	b.WriteString(fmt.Sprintf("Agent ID: %s\n", input.MemberID))
	if member.Lane != "" {
		b.WriteString(fmt.Sprintf("Lane: %s\n", member.Lane))
	}
	// Document authority and the allowed/forbidden write paths are deliberately
	// absent here. Both were rendered a second time in this section while
	// `<storage-map>` (RenderTeamStorage) and the task's declared surfaces
	// (Write Surface) already carry them with more context — Storage Map adds
	// each surface's kind, owner, and purpose; the brief names what each write
	// surface is for. A contract restated in two vocabularies is a contract an
	// agent has to reconcile at read time.
	b.WriteString("\n## Operating Constraints\n\n")
	b.WriteString("Your write surfaces are declared in `<storage-map>` and cannot be widened by the task.\n")
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
		// SharedState files (tasks.json and authored team documents).
		// are runtime data: created on first write under RuntimeData class, not
		// pre-populated in the repo. Validate structure only; doc.Required
		// stays meaningful for prompt rendering ("kind is in scope for this
		// team"), but does not imply on-disk existence at contract-load time.
		if err := validatePathRefStructure([]PathRef{doc.Path}, input, "", "sharedState."+doc.ID); err != nil {
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
		if err != nil {
			continue
		}
		if path == hubPath {
			return true
		}
		// A declared path ending in "/" is a canon root covering everything
		// beneath it, so a hub inside a root is listed by containment. Without
		// this, declaring a root instead of a file list makes the hub look
		// unlisted and fails the whole team contract. NormalizePath runs
		// filepath.Clean, which strips the trailing slash, so root-ness is read
		// from the raw ref rather than the normalized path.
		if strings.HasSuffix(strings.TrimSpace(ref.Path), "/") && strings.HasPrefix(hubPath, path+"/") {
			return true
		}
	}
	return false
}

// validatePathRefStructure validates that each PathRef is well-formed
// (resolvable by NormalizePath) without requiring the file to exist on disk.
// Used for RuntimeData-class documents (sharedState, member runtime files),
// which are created on first write rather than pre-populated in the repo.
func validatePathRefStructure(paths []PathRef, input ValidationInput, activeMemberID, field string) error {
	if len(paths) == 0 {
		return fmt.Errorf("operatingContract.documents.%s requires at least one path", field)
	}
	for _, ref := range paths {
		if _, err := NormalizePath(ref, input, activeMemberID); err != nil {
			return fmt.Errorf("operatingContract.documents.%s: %w", field, err)
		}
	}
	return nil
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
			case "handoff", "knowledge", "task", "inbox-message", "backlog":
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

func deriveRepoRoot(configDir string) string {
	if configDir == "" {
		return ""
	}
	abs, err := filepath.Abs(configDir)
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

func RenderTeamStorage(contract *OperatingContract, input RenderInput) (string, error) {
	if err := Validate(contract, ValidationInput{
		TeamID: input.TeamID, MemberIDs: []string{input.MemberID}, StoreDir: input.StoreDir, RepoRoot: input.RepoRoot,
	}); err != nil {
		return "", err
	}

	validationInput := ValidationInput{TeamID: input.TeamID, StoreDir: input.StoreDir, RepoRoot: input.RepoRoot}
	var b strings.Builder
	b.WriteString("## Your Team Storage\n\n")

	if len(contract.Documents.PlanOfRecord) > 0 {
		b.WriteString("Plan of record, read/propose only:\n")
		renderStoragePlanOfRecordHubs(&b, contract.Documents.PlanOfRecord, validationInput, input.MemberID)
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
	b.WriteString("- unified work filing: file findings and requests once into swarm-manager\n")
	b.WriteString("- knowledge: " + primitiveKnowledgeAvailability(member) + "\n")
	b.WriteString("- task board: " + primitiveTaskAvailability(member) + "\n")
	return strings.TrimRight(b.String(), "\n") + "\n", nil
}

func primitiveKnowledgeAvailability(member MemberContract) string {
	if writeRefsContainKind(member.ForbiddenWrites, "knowledge") {
		return "`unavailable` - do not write team knowledge from this heartbeat"
	}
	if writeRefsContainKind(member.AllowedWrites, "knowledge") {
		return "`write-allowed` - record structured observations and friction signals using required topics"
	}
	return "`unavailable` - no knowledge surface is declared for this member"
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
			FileCount: len(doc.Paths),
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
		// The file count used to live in a second rendering of the same
		// documents under "## Document Authority". That section printed a
		// strict subset of this one, so it was removed and its only unique
		// value folded in here.
		if row.FileCount > 1 {
			b.WriteString(fmt.Sprintf("  Navigation: %d declared files; start at the hub and follow its file map to the relevant spoke.\n", row.FileCount))
		} else {
			b.WriteString("  Navigation: start at the hub and follow its file map to the relevant spoke.\n")
		}
	}
}

type storagePlanOfRecordRow struct {
	Hub       string
	Policy    string
	Consumers string
	UseFor    string
	FileCount int
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
