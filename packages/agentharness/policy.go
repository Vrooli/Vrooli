// Package agentpolicy is the provider-neutral, offline policy runtime used by
// coding-agent resource adapters. Provider processes publish snapshots; this
// package owns normalization, scope, maturity, health, rollout, and evidence
// semantics.
package agentharness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	ContractVersion = "agent-policy/v1"
	MaxClockSkew    = 5 * time.Minute
)

type RiskClass string

const (
	RiskInspection        RiskClass = "inspection"
	RiskFrozenReproduce   RiskClass = "frozen_lockfile_reproduction"
	RiskDependencyAdd     RiskClass = "dependency_addition"
	RiskDependencyUpgrade RiskClass = "dependency_upgrade"
	RiskDependencyRemove  RiskClass = "dependency_removal"
	RiskLifecycle         RiskClass = "lifecycle_execution"
	RiskPublish           RiskClass = "publish"
	RiskFilesystemRemoval RiskClass = "filesystem_removal"
	RiskOpaque            RiskClass = "opaque"
	RiskUnknown           RiskClass = "unknown"
)

type DecisionAction string

const (
	ActionAllow       DecisionAction = "allow"
	ActionAsk         DecisionAction = "ask"
	ActionRewrite     DecisionAction = "rewrite"
	ActionRoute       DecisionAction = "route"
	ActionRepair      DecisionAction = "repair"
	ActionDeny        DecisionAction = "deny"
	ActionUnavailable DecisionAction = "unavailable"
)

type Maturity string

const (
	MaturityExperimental Maturity = "experimental"
	MaturityAdvisory     Maturity = "advisory"
	MaturityGuided       Maturity = "guided"
	MaturityGuarded      Maturity = "guarded"
	MaturityEnforcing    Maturity = "enforcing"
)

type Health string

const (
	HealthHealthy     Health = "healthy"
	HealthDegraded    Health = "degraded"
	HealthUnavailable Health = "unavailable"
	HealthStale       Health = "stale"
	HealthFailed      Health = "failed"
)

type RolloutProfile string

const (
	ProfileAdvisory  RolloutProfile = "advisory"
	ProfileGuided    RolloutProfile = "guided"
	ProfileGuarded   RolloutProfile = "guarded"
	ProfileEnforcing RolloutProfile = "enforcing"
)

type EvidenceState string

const (
	EvidenceClean       EvidenceState = "clean"
	EvidenceFinding     EvidenceState = "finding"
	EvidenceUnknown     EvidenceState = "unknown"
	EvidenceUnsupported EvidenceState = "unsupported"
	EvidenceUnavailable EvidenceState = "unavailable"
	EvidenceStale       EvidenceState = "stale"
)

type ToolEvent struct {
	ContractVersion  string            `json:"contract_version"`
	EventID          string            `json:"event_id,omitempty"`
	Runner           string            `json:"runner"`
	Tool             string            `json:"tool"`
	Arguments        []string          `json:"arguments,omitempty"`
	Shell            string            `json:"shell,omitempty"`
	WorkingDirectory string            `json:"working_directory,omitempty"`
	Target           string            `json:"target,omitempty"`
	OccurredAt       time.Time         `json:"occurred_at"`
	Context          map[string]string `json:"context,omitempty"`
}

type ProviderScope struct {
	Runners    []string `json:"runners,omitempty"`
	Roots      []string `json:"roots,omitempty"`
	Ecosystems []string `json:"ecosystems,omitempty"`
	// Risks limits the snapshot to these risk classes, so a provider that
	// vouches for one kind of action is not read as vouching for another.
	Risks []RiskClass `json:"risks,omitempty"`
}

// PathRule is a provider's disposition for one filesystem location: whether
// an agent may delete inside Root. The most specific containing rule wins; a
// stricter rule nested inside a recursive target wins over it.
type PathRule struct {
	Root   string         `json:"root"`
	Action DecisionAction `json:"action"`
	Reason string         `json:"reason"`
	// Source names the declaration, such as a repo-contract storage root.
	Source string `json:"source,omitempty"`
	Owner  string `json:"owner,omitempty"`
	// Match turns the rule into a declaration of generated names: it applies
	// to each existing child of Root whose name matches the glob (a name, never
	// a path), found when the deletion is decided so a leftover made a moment
	// ago is covered. Root itself is then not covered.
	Match string `json:"match,omitempty"`
	// Only limits an allow rule: a folder is allowed only while every file below
	// it matches one of these name globs, so a folder holding anything else asks.
	Only []string `json:"only,omitempty"`
	// Provider is the snapshot that published the rule. The runtime sets it
	// on load; a provider cannot claim another's name.
	Provider string `json:"provider,omitempty"`
}

type ProviderCapability struct {
	ID                string   `json:"id"`
	IdealPosture      string   `json:"ideal_posture"`
	DeclaredMaturity  Maturity `json:"declared_maturity"`
	SupportsAnalysis  bool     `json:"supports_analysis"`
	SupportsEnforce   bool     `json:"supports_enforcement"`
	SupportsRepair    bool     `json:"supports_repair"`
	SupportsSnapshots bool     `json:"supports_snapshot_publication"`
}

type ProviderHealth struct {
	State     Health    `json:"state"`
	CheckedAt time.Time `json:"checked_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Message   string    `json:"message,omitempty"`
}

type ProviderReadiness struct {
	State             string   `json:"state"`
	PromotionEvidence []string `json:"promotion_evidence,omitempty"`
	RollbackPlan      string   `json:"rollback_plan"`
}

type Evidence struct {
	Code     string            `json:"code"`
	Message  string            `json:"message"`
	Source   string            `json:"source,omitempty"`
	Facts    map[string]string `json:"facts,omitempty"`
	Severity string            `json:"severity,omitempty"`
}

type RepairPlan struct {
	ID            string    `json:"id"`
	Owner         string    `json:"owner"`
	Operation     string    `json:"operation"`
	TargetRoot    string    `json:"target_root"`
	Scope         []string  `json:"scope"`
	PreviewDigest string    `json:"preview_digest"`
	TransactionID string    `json:"transaction_id"`
	ExpiresAt     time.Time `json:"expires_at"`
	Rollback      string    `json:"rollback"`
	Validator     string    `json:"validator"`
	Idempotent    bool      `json:"idempotent"`
}

type PolicyRule struct {
	// EvaluatorID is issued by the owning policy provider and names this exact
	// evaluator. It must be namespaced as <provider-id>:<stable-rule-name>.
	// Empty IDs are legacy and are never eligible for temporary suppression.
	EvaluatorID string         `json:"evaluator_id,omitempty"`
	Risk        RiskClass      `json:"risk"`
	Action      DecisionAction `json:"action"`
	Reason      string         `json:"reason"`
	Evidence    []Evidence     `json:"evidence,omitempty"`
	Replacement []string       `json:"replacement,omitempty"`
	Repair      *RepairPlan    `json:"repair,omitempty"`
}

type ProviderSnapshot struct {
	ProviderID   string               `json:"provider_id"`
	Version      string               `json:"version"`
	Capabilities []ProviderCapability `json:"capabilities"`
	Scope        ProviderScope        `json:"scope"`
	Health       ProviderHealth       `json:"health"`
	Readiness    ProviderReadiness    `json:"readiness"`
	Evidence     EvidenceState        `json:"evidence_state"`
	CapturedAt   time.Time            `json:"captured_at"`
	ExpiresAt    time.Time            `json:"expires_at"`
	Rules        []PolicyRule         `json:"rules,omitempty"`
	PathRules    []PathRule           `json:"path_rules,omitempty"`
	// NameRules are declarations of generated names (PathRule.Match). They are
	// a separate field so a runner older than the name match, which ignores
	// fields it does not know, never reads one as a rule for the whole Root.
	NameRules []PathRule `json:"name_rules,omitempty"`
	// RemovalApproval is the command an agent runs to ask the operator to
	// approve deleting a location these path rules withhold. The runtime
	// appends one --path argument per withheld path to the refusal it shows.
	RemovalApproval string            `json:"removal_approval,omitempty"`
	Provenance      map[string]string `json:"provenance,omitempty"`
}

type SnapshotBundle struct {
	SchemaVersion string                      `json:"schema_version"`
	Generation    uint64                      `json:"generation"`
	PublishedAt   time.Time                   `json:"published_at"`
	Snapshots     map[string]ProviderSnapshot `json:"snapshots"`
	Integrity     string                      `json:"integrity"`
}

type Decision struct {
	ContractVersion      string         `json:"contract_version"`
	EventID              string         `json:"event_id"`
	Action               DecisionAction `json:"action"`
	Risk                 RiskClass      `json:"risk"`
	ProviderID           string         `json:"provider_id,omitempty"`
	ProviderVersion      string         `json:"provider_version,omitempty"`
	EvaluatorID          string         `json:"evaluator_id,omitempty"`
	EvaluationDurationMS int64          `json:"evaluation_duration_ms,omitempty"`
	Maturity             Maturity       `json:"declared_maturity,omitempty"`
	Health               Health         `json:"health,omitempty"`
	UsedSnapshot         bool           `json:"used_snapshot"`
	Degraded             bool           `json:"degraded"`
	Reason               string         `json:"reason"`
	Replacement          []string       `json:"replacement,omitempty"`
	Repair               *RepairPlan    `json:"repair,omitempty"`
	Evidence             []Evidence     `json:"evidence"`
}

// evaluatorIDOwnedBy prevents a provider from assigning another provider's
// namespace. Empty or legacy identifiers remain usable as ordinary policy
// rules, but can never name a selectable temporary check.
func evaluatorIDOwnedBy(id, providerID string) bool {
	id = strings.TrimSpace(id)
	providerID = strings.TrimSpace(providerID)
	if providerID == "" || strings.Count(id, ":") != 1 || !strings.HasPrefix(id, providerID+":") || len(id) <= len(providerID)+1 {
		return false
	}
	for _, r := range id {
		if r == ':' {
			continue
		}
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._/-", r)) {
			return false
		}
	}
	return true
}

func (e *ToolEvent) Normalize(now time.Time) error {
	if e.ContractVersion == "" {
		e.ContractVersion = ContractVersion
	}
	if e.ContractVersion != ContractVersion {
		return fmt.Errorf("unsupported tool event contract %q", e.ContractVersion)
	}
	e.Runner = strings.TrimSpace(e.Runner)
	e.Tool = strings.TrimSpace(e.Tool)
	if e.Runner == "" || (e.Tool == "" && len(e.Arguments) == 0) {
		return errors.New("runner and tool or arguments are required")
	}
	for i, arg := range e.Arguments {
		if strings.IndexByte(arg, 0) >= 0 {
			return fmt.Errorf("argument %d contains NUL", i)
		}
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = now.UTC()
	}
	if e.EventID == "" {
		e.EventID = EventID(*e)
	}
	return nil
}

func ValidateSnapshot(snapshot ProviderSnapshot) error {
	if strings.TrimSpace(snapshot.ProviderID) == "" || strings.TrimSpace(snapshot.Version) == "" {
		return errors.New("provider_id and version are required")
	}
	if snapshot.CapturedAt.IsZero() || snapshot.ExpiresAt.IsZero() || !snapshot.ExpiresAt.After(snapshot.CapturedAt) {
		return errors.New("snapshot capture and expiry window are required")
	}
	if snapshot.Health.State == "" || snapshot.Evidence == "" || snapshot.Readiness.State == "" {
		return errors.New("snapshot health, readiness, and evidence state are required")
	}
	seen := map[string]bool{}
	for _, capability := range snapshot.Capabilities {
		if strings.TrimSpace(capability.ID) == "" || seen[capability.ID] {
			return errors.New("snapshot capabilities must have unique ids")
		}
		seen[capability.ID] = true
	}
	seenEvaluatorIDs := map[string]bool{}
	for _, rule := range snapshot.Rules {
		if rule.Risk == "" || rule.Action == "" {
			return errors.New("policy rules require risk and action")
		}
		if id := strings.TrimSpace(rule.EvaluatorID); id != "" {
			if seenEvaluatorIDs[id] {
				return fmt.Errorf("provider %s declares duplicate evaluator ID %q", snapshot.ProviderID, id)
			}
			seenEvaluatorIDs[id] = true
		}
		if rule.Action != ActionAllow && strings.TrimSpace(rule.Reason) == "" && len(rule.Evidence) == 0 {
			return errors.New("non-allow rules require reason or evidence")
		}
		if rule.Action == ActionRepair {
			if rule.Repair == nil {
				return errors.New("repair rules require a repair plan")
			}
			if err := ValidateRepairPlan(*rule.Repair); err != nil {
				return err
			}
		}
	}
	for _, rule := range snapshot.NameRules {
		if err := validateNameRule(rule); err != nil {
			return err
		}
	}
	for _, rule := range snapshot.PathRules {
		if strings.TrimSpace(rule.Root) == "" || !filepath.IsAbs(rule.Root) {
			return fmt.Errorf("path rule root %q must be absolute", rule.Root)
		}
		if filepath.Dir(filepath.Clean(rule.Root)) == filepath.Clean(rule.Root) {
			return fmt.Errorf("path rule root %q is a filesystem root", rule.Root)
		}
		if rule.Action != ActionAllow && rule.Action != ActionAsk && rule.Action != ActionDeny {
			return fmt.Errorf("path rule %q action must be allow, ask, or deny", rule.Root)
		}
		if strings.TrimSpace(rule.Reason) == "" {
			return fmt.Errorf("path rule %q requires a reason", rule.Root)
		}
		if err := validatePathRulePatterns(rule); err != nil {
			return err
		}
	}
	return nil
}

// validatePathRulePatterns checks a rule's name globs.
func validateNameRule(rule PathRule) error {
	switch {
	case strings.TrimSpace(rule.Match) == "":
		return fmt.Errorf("name rule %q requires a match", rule.Root)
	case rule.Action != ActionAllow:
		return fmt.Errorf("name rule %q: only allow rules may match generated names", rule.Root)
	case strings.TrimSpace(rule.Reason) == "" || !filepath.IsAbs(rule.Root) || filepath.Dir(filepath.Clean(rule.Root)) == filepath.Clean(rule.Root):
		return fmt.Errorf("name rule %q needs an absolute root below the filesystem root and a reason", rule.Root)
	case strings.Trim(rule.Match, "*?[]") == "":
		return fmt.Errorf("name rule %q: match %q names nothing specific", rule.Root, rule.Match)
	}
	return validatePathRulePatterns(rule)
}

func validatePathRulePatterns(rule PathRule) error {
	if strings.TrimSpace(rule.Match) != "" {
		if strings.ContainsAny(rule.Match, `/\`) {
			return fmt.Errorf("path rule %q match %q must be a name, not a path", rule.Root, rule.Match)
		}
		if _, err := filepath.Match(rule.Match, "x"); err != nil {
			return fmt.Errorf("path rule %q match %q: %w", rule.Root, rule.Match, err)
		}
	}
	for _, glob := range rule.Only {
		if _, err := filepath.Match(glob, "x"); err != nil {
			return fmt.Errorf("path rule %q only %q: %w", rule.Root, glob, err)
		}
	}
	if len(rule.Only) > 0 && rule.Action != ActionAllow {
		return fmt.Errorf("path rule %q: only applies to allow rules", rule.Root)
	}
	return nil
}

func ValidateBundle(bundle SnapshotBundle) error {
	if bundle.SchemaVersion != ContractVersion || bundle.Snapshots == nil {
		return fmt.Errorf("bundle schema must be %q", ContractVersion)
	}
	if bundle.PublishedAt.IsZero() {
		return errors.New("bundle published_at is required")
	}
	for id, snapshot := range bundle.Snapshots {
		if id != snapshot.ProviderID {
			return fmt.Errorf("snapshot key %q does not match provider_id", id)
		}
		if err := ValidateSnapshot(snapshot); err != nil {
			return fmt.Errorf("provider %s: %w", id, err)
		}
	}
	return nil
}

func ValidateRepairPlan(plan RepairPlan) error {
	if plan.ID == "" || plan.Owner == "" || plan.Operation == "" || plan.TargetRoot == "" || plan.TransactionID == "" || plan.Rollback == "" || plan.Validator == "" {
		return errors.New("repair plan is missing required safety fields")
	}
	if filepath.IsAbs(plan.TargetRoot) || filepath.Clean(plan.TargetRoot) == "." || strings.Contains(filepath.ToSlash(plan.TargetRoot), "../") || filepath.ToSlash(plan.TargetRoot) == ".." {
		return errors.New("repair target root must be a relative scenario root")
	}
	if len(plan.Scope) == 0 || plan.PreviewDigest == "" || plan.ExpiresAt.IsZero() || !plan.Idempotent {
		return errors.New("repair plan requires scoped digest, expiry, and idempotence")
	}
	for _, path := range plan.Scope {
		if filepath.IsAbs(path) || filepath.Clean(path) == ".." || strings.HasPrefix(filepath.ToSlash(filepath.Clean(path)), "../") {
			return errors.New("repair scope escapes its target root")
		}
	}
	return nil
}

func PreviewDigest(plan RepairPlan) string {
	data, _ := json.Marshal(struct {
		ID         string   `json:"id"`
		Operation  string   `json:"operation"`
		TargetRoot string   `json:"target_root"`
		Scope      []string `json:"scope"`
		Validator  string   `json:"validator"`
	}{plan.ID, plan.Operation, plan.TargetRoot, append([]string(nil), plan.Scope...), plan.Validator})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func EventID(event ToolEvent) string {
	data, _ := json.Marshal(struct {
		Runner string   `json:"runner"`
		Tool   string   `json:"tool"`
		Args   []string `json:"args"`
		Shell  string   `json:"shell"`
		Dir    string   `json:"dir"`
		Target string   `json:"target"`
	}{event.Runner, event.Tool, event.Arguments, event.Shell, event.WorkingDirectory, event.Target})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func highRisk(risk RiskClass) bool {
	return risk != RiskInspection && risk != RiskFrozenReproduce
}

func maturityRank(maturity Maturity) int {
	switch maturity {
	case MaturityEnforcing:
		return 4
	case MaturityGuarded:
		return 3
	case MaturityGuided:
		return 2
	case MaturityAdvisory:
		return 1
	default:
		return 0
	}
}

func snapshotIDs(bundle SnapshotBundle) []string {
	ids := make([]string, 0, len(bundle.Snapshots))
	for id := range bundle.Snapshots {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
