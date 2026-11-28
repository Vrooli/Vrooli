package validation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ContractVersion is the version of the provider/policy envelope. It is
// deliberately independent from the scenario-validation transport version so
// provider internals can evolve without changing the Test Genie RPC.
const ContractVersion = "security-policy/v1"

type EvidenceState string

const (
	EvidenceClean       EvidenceState = "clean"
	EvidenceFinding     EvidenceState = "finding"
	EvidenceUnsupported EvidenceState = "unsupported"
	EvidenceUnavailable EvidenceState = "unavailable"
	EvidenceStale       EvidenceState = "stale"
	EvidenceFailed      EvidenceState = "failed"
	EvidenceUncheckable EvidenceState = "uncheckable"
)

type ScannerHealth string

const (
	ScannerHealthy  ScannerHealth = "healthy"
	ScannerMissing  ScannerHealth = "unavailable"
	ScannerStale    ScannerHealth = "stale"
	ScannerFailed   ScannerHealth = "failed"
	ScannerTimedOut ScannerHealth = "timed_out"
	ScannerPartial  ScannerHealth = "partial"
)

type ProviderMaturity string

const (
	MaturityExperimental ProviderMaturity = "experimental"
	MaturityAdvisory     ProviderMaturity = "advisory"
	MaturityGuided       ProviderMaturity = "guided"
	MaturityGuarded      ProviderMaturity = "guarded"
	MaturityEnforcing    ProviderMaturity = "enforcing"
)

type RuntimeHealth string

const (
	RuntimeHealthy     RuntimeHealth = "healthy"
	RuntimeDegraded    RuntimeHealth = "degraded"
	RuntimeUnavailable RuntimeHealth = "unavailable"
	RuntimeStale       RuntimeHealth = "stale"
	RuntimeFailed      RuntimeHealth = "failed"
)

type RolloutProfile string

const (
	RolloutAdvisory  RolloutProfile = "advisory"
	RolloutGuided    RolloutProfile = "guided"
	RolloutGuarded   RolloutProfile = "guarded"
	RolloutEnforcing RolloutProfile = "enforcing"
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

type FixClass string

const (
	FixManual        FixClass = "manual"
	FixAssisted      FixClass = "assisted"
	FixPreviewable   FixClass = "previewable"
	FixDeterministic FixClass = "deterministic"
	FixProhibited    FixClass = "prohibited"
)

// ProviderCapability is the auditable declaration for one provider action.
// Declared maturity is never inferred from health or silently promoted.
type ProviderCapability struct {
	ID                string           `json:"id"`
	Owner             string           `json:"owner"`
	IdealPosture      string           `json:"ideal_posture"`
	Maturity          ProviderMaturity `json:"maturity"`
	RuntimeHealth     RuntimeHealth    `json:"runtime_health"`
	EvidenceState     EvidenceState    `json:"evidence_state"`
	EvidenceExpiresAt time.Time        `json:"evidence_expires_at,omitempty"`
	SupportsRepair    bool             `json:"supports_repair"`
}

type ProviderHealth struct {
	Provider          string        `json:"provider"`
	Version           string        `json:"version"`
	RuntimeHealth     RuntimeHealth `json:"runtime_health"`
	CheckedAt         time.Time     `json:"checked_at"`
	EvidenceExpiresAt time.Time     `json:"evidence_expires_at,omitempty"`
	Message           string        `json:"message,omitempty"`
}

type ProviderReadiness struct {
	Provider          string               `json:"provider"`
	Capabilities      []ProviderCapability `json:"capabilities"`
	Health            ProviderHealth       `json:"health"`
	PromotionEvidence []string             `json:"promotion_evidence,omitempty"`
	RollbackPlan      string               `json:"rollback_plan"`
}

type ToolEvent struct {
	ContractVersion  string            `json:"contract_version"`
	EventID          string            `json:"event_id"`
	Runner           string            `json:"runner"`
	Tool             string            `json:"tool"`
	Arguments        []string          `json:"arguments,omitempty"`
	Shell            string            `json:"shell,omitempty"`
	WorkingDirectory string            `json:"working_directory,omitempty"`
	Target           string            `json:"target,omitempty"`
	OccurredAt       time.Time         `json:"occurred_at"`
	Context          map[string]string `json:"context,omitempty"`
}

type DecisionAction string

const (
	DecisionAllow       DecisionAction = "allow"
	DecisionDeny        DecisionAction = "deny"
	DecisionAsk         DecisionAction = "ask"
	DecisionRoute       DecisionAction = "route"
	DecisionRepair      DecisionAction = "repair"
	DecisionRewrite     DecisionAction = "rewrite"
	DecisionUnavailable DecisionAction = "unavailable"
)

type RepairPlan struct {
	ID            string    `json:"id"`
	Owner         string    `json:"owner"`
	Class         FixClass  `json:"class"`
	Scope         []string  `json:"scope"`
	PreviewDigest string    `json:"preview_digest"`
	Idempotent    bool      `json:"idempotent"`
	Rollback      string    `json:"rollback"`
	Validation    string    `json:"validation"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
}

type ProviderDecision struct {
	ContractVersion string           `json:"contract_version"`
	EventID         string           `json:"event_id"`
	Action          DecisionAction   `json:"action"`
	Risk            RiskClass        `json:"risk"`
	Reason          string           `json:"reason"`
	Provider        string           `json:"provider"`
	Maturity        ProviderMaturity `json:"maturity"`
	RuntimeHealth   RuntimeHealth    `json:"runtime_health"`
	EvidenceState   EvidenceState    `json:"evidence_state"`
	Repair          *RepairPlan      `json:"repair,omitempty"`
}

// ClassifyToolEvent intentionally uses conservative lexical signals. The
// runner may add Code Facts evidence, but missing facts must not make an
// opaque command appear safe.
func ClassifyToolEvent(event ToolEvent) RiskClass {
	joined := strings.ToLower(strings.Join(append([]string{event.Tool, event.Shell}, event.Arguments...), " "))
	if strings.TrimSpace(joined) == "" {
		return RiskUnknown
	}
	if strings.ContainsAny(joined, ";|&\n\r") || strings.Contains(joined, "$(") || strings.Contains(joined, "`") {
		return RiskOpaque
	}
	if containsAny(joined, "publish", "npm publish", "pnpm publish", "cargo publish") {
		return RiskPublish
	}
	if containsAny(joined, "--frozen-lockfile", "--frozen", "npm ci", "cargo fetch", "go mod download") {
		return RiskFrozenReproduce
	}
	if containsAny(joined, "postinstall", "preinstall", " install", " run prepare", " lifecycle") {
		return RiskLifecycle
	}
	if containsAny(joined, "add ", " add", " install ", " upgrade", " update", " remove ", " uninstall") {
		if containsAny(joined, "remove", "uninstall") {
			return RiskDependencyRemove
		}
		if containsAny(joined, "upgrade", "update") {
			return RiskDependencyUpgrade
		}
		return RiskDependencyAdd
	}
	if containsAny(joined, "list", "show", "status", "audit", "check", "cat ", "--help", "--version") {
		return RiskInspection
	}
	if event.Shell != "" || len(event.Arguments) > 1 {
		return RiskOpaque
	}
	return RiskUnknown
}

// EffectiveAction computes the rollout decision. Guarded and enforcing modes
// fail closed only for high-risk mutations; advisory and guided remain usable
// when providers are immature or unavailable.
func EffectiveAction(profile RolloutProfile, capability ProviderCapability, risk RiskClass) DecisionAction {
	highRisk := risk == RiskDependencyAdd || risk == RiskDependencyUpgrade || risk == RiskDependencyRemove || risk == RiskLifecycle || risk == RiskPublish || risk == RiskOpaque || risk == RiskUnknown
	healthy := capability.RuntimeHealth == RuntimeHealthy && capability.EvidenceState == EvidenceClean
	if !healthy {
		if profile == RolloutGuarded || profile == RolloutEnforcing {
			if highRisk {
				return DecisionDeny
			}
		}
		return DecisionAsk
	}
	if profile == RolloutAdvisory || capability.Maturity == MaturityExperimental || capability.Maturity == MaturityAdvisory {
		return DecisionAllow
	}
	if profile == RolloutGuided || capability.Maturity == MaturityGuided {
		if highRisk {
			return DecisionAsk
		}
		return DecisionAllow
	}
	return DecisionAllow
}

func ValidateProviderDecision(decision ProviderDecision) error {
	if decision.ContractVersion != ContractVersion {
		return fmt.Errorf("contract_version must be %q", ContractVersion)
	}
	if strings.TrimSpace(decision.EventID) == "" || strings.TrimSpace(decision.Provider) == "" {
		return errors.New("event_id and provider are required")
	}
	if decision.Action == "" || decision.Risk == "" {
		return errors.New("action and risk are required")
	}
	if decision.Action == DecisionRepair && decision.Repair == nil {
		return errors.New("repair action requires a repair plan")
	}
	return nil
}

func RepairDigest(plan RepairPlan) string {
	data, _ := json.Marshal(struct {
		ID         string   `json:"id"`
		Owner      string   `json:"owner"`
		Class      FixClass `json:"class"`
		Scope      []string `json:"scope"`
		Validation string   `json:"validation"`
	}{plan.ID, plan.Owner, plan.Class, plan.Scope, plan.Validation})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func containsAny(value string, parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(value, part) {
			return true
		}
	}
	return false
}
