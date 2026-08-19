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
	Provenance   map[string]string    `json:"provenance,omitempty"`
}

type SnapshotBundle struct {
	SchemaVersion string                      `json:"schema_version"`
	Generation    uint64                      `json:"generation"`
	PublishedAt   time.Time                   `json:"published_at"`
	Snapshots     map[string]ProviderSnapshot `json:"snapshots"`
	Integrity     string                      `json:"integrity"`
}

type Decision struct {
	ContractVersion string         `json:"contract_version"`
	EventID         string         `json:"event_id"`
	Action          DecisionAction `json:"action"`
	Risk            RiskClass      `json:"risk"`
	ProviderID      string         `json:"provider_id,omitempty"`
	ProviderVersion string         `json:"provider_version,omitempty"`
	Maturity        Maturity       `json:"declared_maturity,omitempty"`
	Health          Health         `json:"health,omitempty"`
	UsedSnapshot    bool           `json:"used_snapshot"`
	Degraded        bool           `json:"degraded"`
	Reason          string         `json:"reason"`
	Replacement     []string       `json:"replacement,omitempty"`
	Repair          *RepairPlan    `json:"repair,omitempty"`
	Evidence        []Evidence     `json:"evidence"`
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
	for _, rule := range snapshot.Rules {
		if rule.Risk == "" || rule.Action == "" {
			return errors.New("policy rules require risk and action")
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
