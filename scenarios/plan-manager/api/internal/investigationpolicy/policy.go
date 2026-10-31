// Package investigationpolicy owns Plan Manager's trigger eligibility
// decision. It produces a delegation intent; Agent Manager remains the owner
// of evidence collection, diagnosis, and investigation execution.
package investigationpolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const SchemaVersion = "plan-investigation-policy/v1"

type Mode string

const (
	ModeShadow    Mode = "shadow"
	ModeAutomatic Mode = "automatic"
)

type Rule struct {
	Kind                  string  `json:"kind"`
	AfterSeconds          int     `json:"afterSeconds,omitempty"`
	IgnoreKnownOwnerWaits bool    `json:"ignoreKnownOwnerWaits,omitempty"`
	MinDistinctAttempts   int     `json:"minDistinctAttempts,omitempty"`
	WindowSeconds         int     `json:"windowSeconds,omitempty"`
	MinimumNewEvidence    int     `json:"minimumNewEvidence,omitempty"`
	Fraction              float64 `json:"fraction,omitempty"`
	RequireObservedBudget bool    `json:"requireObservedBudget,omitempty"`
	// Enabled is retained for compatibility with early drafts. Rules are
	// enabled by default so an omitted field cannot silently disable a policy;
	// use Disabled for an explicit opt-out.
	Enabled  bool `json:"enabled,omitempty"`
	Disabled bool `json:"disabled,omitempty"`
}

type Limits struct {
	CooldownSeconds               int `json:"cooldownSeconds"`
	MaxInvestigationsPerExecution int `json:"maxInvestigationsPerExecution"`
	MaxConcurrentPerExecution     int `json:"maxConcurrentPerExecution"`
	MaxConcurrentGlobal           int `json:"maxConcurrentGlobal"`
	MaxInvestigatorDepth          int `json:"maxInvestigatorDepth"`
}

type Deduplication struct {
	Basis    []string `json:"basis"`
	ReopenOn string   `json:"reopenOn"`
}

type RecommendationPolicy struct {
	AutoApply    bool     `json:"autoApply"`
	AllowedKinds []string `json:"allowedKinds"`
}

type PolicyScope struct {
	FamilyID    string `json:"familyId,omitempty"`
	ExecutionID string `json:"executionId,omitempty"`
	PhaseID     string `json:"phaseId,omitempty"`
}

func (s PolicyScope) Key() string {
	if strings.TrimSpace(s.PhaseID) != "" {
		return "phase:" + strings.TrimSpace(s.PhaseID)
	}
	if strings.TrimSpace(s.ExecutionID) != "" {
		return "execution:" + strings.TrimSpace(s.ExecutionID)
	}
	if strings.TrimSpace(s.FamilyID) != "" {
		return "family:" + strings.TrimSpace(s.FamilyID)
	}
	return "global"
}

func (s PolicyScope) Validate() error {
	if len(strings.TrimSpace(s.FamilyID)) > 128 || len(strings.TrimSpace(s.ExecutionID)) > 128 || len(strings.TrimSpace(s.PhaseID)) > 128 {
		return fmt.Errorf("%w: policy scope identifiers are bounded", ErrInvalidPolicy)
	}
	return nil
}

type Exclusions struct {
	InvestigationWorkloads bool `json:"investigationWorkloads"`
	DisabledExecutions     bool `json:"disabledExecutions"`
	TerminalExecutions     bool `json:"terminalExecutions"`
}

type Policy struct {
	SchemaVersion   string               `json:"schemaVersion"`
	Version         string               `json:"version"`
	Scope           PolicyScope          `json:"scope,omitempty"`
	Mode            Mode                 `json:"mode"`
	Rules           []Rule               `json:"rules"`
	Limits          Limits               `json:"limits"`
	Deduplication   Deduplication        `json:"deduplication"`
	Recommendations RecommendationPolicy `json:"recommendations"`
	Exclusions      Exclusions           `json:"exclusions"`
}

type Attempt struct {
	ID         string    `json:"id"`
	Failed     bool      `json:"failed"`
	OccurredAt time.Time `json:"occurredAt"`
}

type Observation struct {
	ExecutionID     string `json:"executionId"`
	PhaseID         string `json:"phaseId"`
	PhaseGeneration string `json:"phaseGeneration"`
	FamilyID        string `json:"familyId,omitempty"`
	// SharedFailureRef is an owner-authored, typed reference to the common
	// producer failure that permits family incident coalescing. FamilyID alone
	// is never sufficient: unrelated child failures must remain independent.
	SharedFailureRef      string     `json:"sharedFailureRef,omitempty"`
	RuleVersion           string     `json:"ruleVersion"`
	Now                   time.Time  `json:"now"`
	LastMaterialProgress  time.Time  `json:"lastMaterialProgress"`
	KnownOwnerWait        bool       `json:"knownOwnerWait"`
	Attempts              []Attempt  `json:"attempts"`
	PhaseStartedAt        time.Time  `json:"phaseStartedAt"`
	NewEvidence           int        `json:"newEvidence"`
	BudgetUsed            int64      `json:"budgetUsed"`
	BudgetTotal           int64      `json:"budgetTotal"`
	Terminal              bool       `json:"terminal"`
	TerminalMismatch      bool       `json:"terminalMismatch"`
	Disabled              bool       `json:"disabled"`
	InvestigationWorkload bool       `json:"investigationWorkload"`
	InvestigatorDepth     int        `json:"investigatorDepth"`
	ExistingCount         int        `json:"existingCount"`
	ConcurrentExecution   int        `json:"concurrentExecution"`
	ConcurrentGlobal      int        `json:"concurrentGlobal"`
	LastIncidentAt        *time.Time `json:"lastIncidentAt,omitempty"`
	// DispatchContext is caller-supplied, owner-authored context for an
	// automatic investigation. It is deliberately not interpreted by the
	// eligibility evaluator; the dispatcher validates it before admission.
	CallerKey   string   `json:"callerKey,omitempty"`
	Question    string   `json:"question,omitempty"`
	BriefRef    string   `json:"briefRef,omitempty"`
	RunIDs      []string `json:"runIds,omitempty"`
	WaitSeconds int      `json:"waitSeconds,omitempty"`
}

type Match struct {
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}

type Decision struct {
	Eligible            bool     `json:"eligible"`
	Queued              bool     `json:"queued,omitempty"`
	Mode                Mode     `json:"mode"`
	Matches             []Match  `json:"matches"`
	Reasons             []string `json:"reasons"`
	IncidentFingerprint string   `json:"incidentFingerprint,omitempty"`
}

// Occurrence is one observed trigger evaluation. Incidents coalesce repeated
// observations by fingerprint, while occurrences preserve the operator-facing
// history needed to explain why the incident was retained or suppressed.
type Occurrence struct {
	ID                  string    `json:"occurrenceId"`
	IncidentFingerprint string    `json:"incidentFingerprint"`
	ExecutionID         string    `json:"executionId"`
	FamilyID            string    `json:"familyId,omitempty"`
	SharedFailureRef    string    `json:"sharedFailureRef,omitempty"`
	PhaseID             string    `json:"phaseId"`
	PhaseGeneration     string    `json:"phaseGeneration"`
	PolicyVersion       string    `json:"policyVersion"`
	Eligible            bool      `json:"eligible"`
	Decision            Decision  `json:"decision"`
	ObservedAt          time.Time `json:"observedAt"`
	CreatedAt           time.Time `json:"createdAt"`
}

var (
	ErrInvalidPolicy      = errors.New("invalid plan investigation policy")
	ErrInvalidObservation = errors.New("invalid plan investigation observation")
)

func (p Policy) Validate() error {
	if p.SchemaVersion != SchemaVersion || strings.TrimSpace(p.Version) == "" || (p.Mode != ModeShadow && p.Mode != ModeAutomatic) {
		return fmt.Errorf("%w: schemaVersion, version, and mode are required", ErrInvalidPolicy)
	}
	if err := p.Scope.Validate(); err != nil {
		return err
	}
	scopeParts := 0
	for _, value := range []string{p.Scope.FamilyID, p.Scope.ExecutionID, p.Scope.PhaseID} {
		if strings.TrimSpace(value) != "" {
			scopeParts++
		}
	}
	if scopeParts > 1 {
		return fmt.Errorf("%w: a stored policy scope must target one family, execution, or phase", ErrInvalidPolicy)
	}
	if p.Limits.CooldownSeconds < 0 || p.Limits.MaxInvestigationsPerExecution < 1 || p.Limits.MaxConcurrentPerExecution < 1 || p.Limits.MaxConcurrentGlobal < 1 || p.Limits.MaxInvestigatorDepth < 0 {
		return fmt.Errorf("%w: limits must be positive and finite", ErrInvalidPolicy)
	}
	if len(p.Rules) == 0 {
		return fmt.Errorf("%w: at least one trigger rule is required", ErrInvalidPolicy)
	}
	for i, rule := range p.Rules {
		if !knownRule(rule.Kind) {
			return fmt.Errorf("%w: rules[%d] has unknown kind %q", ErrInvalidPolicy, i, rule.Kind)
		}
		if rule.Kind == "no_material_progress" && rule.AfterSeconds <= 0 || rule.Kind == "repeated_failure" && (rule.MinDistinctAttempts <= 0 || rule.WindowSeconds <= 0) || rule.Kind == "phase_elapsed" && (rule.AfterSeconds <= 0 || rule.MinimumNewEvidence < 0) || rule.Kind == "budget_fraction" && (rule.Fraction <= 0 || rule.Fraction > 1) {
			return fmt.Errorf("%w: rules[%d] has invalid thresholds", ErrInvalidPolicy, i)
		}
	}
	return nil
}

func (p Policy) Digest() (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	canonical := p
	canonical.Deduplication.Basis = append([]string(nil), p.Deduplication.Basis...)
	sort.Strings(canonical.Deduplication.Basis)
	raw, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func (p Policy) Evaluate(o Observation) (Decision, error) {
	if err := p.Validate(); err != nil {
		return Decision{}, err
	}
	if strings.TrimSpace(o.ExecutionID) == "" || strings.TrimSpace(o.PhaseID) == "" || o.Now.IsZero() {
		return Decision{}, fmt.Errorf("%w: execution, phase, and now are required", ErrInvalidObservation)
	}
	if len(strings.TrimSpace(o.FamilyID)) > 128 || len(strings.TrimSpace(o.SharedFailureRef)) > 512 {
		return Decision{}, fmt.Errorf("%w: family and shared failure references are bounded", ErrInvalidObservation)
	}
	decision := Decision{Eligible: true, Mode: p.Mode}
	addBlock := func(reason string) { decision.Eligible = false; decision.Reasons = append(decision.Reasons, reason) }
	addQueue := func(reason string) {
		decision.Queued = true
		decision.Reasons = append(decision.Reasons, "queued: "+reason)
	}
	if p.Exclusions.DisabledExecutions && o.Disabled {
		addBlock("execution is disabled")
	}
	if p.Exclusions.TerminalExecutions && o.Terminal {
		addBlock("execution is terminal")
	}
	if p.Exclusions.InvestigationWorkloads && o.InvestigationWorkload {
		addBlock("investigation workload is excluded")
	}
	if o.InvestigatorDepth >= p.Limits.MaxInvestigatorDepth {
		addBlock("investigator ancestry depth limit reached")
	}
	if o.ExistingCount >= p.Limits.MaxInvestigationsPerExecution {
		addQueue("per-execution investigation limit reached")
	}
	if o.ConcurrentExecution >= p.Limits.MaxConcurrentPerExecution {
		addQueue("per-execution concurrency limit reached")
	}
	if o.ConcurrentGlobal >= p.Limits.MaxConcurrentGlobal {
		addQueue("global concurrency limit reached")
	}
	if o.LastIncidentAt != nil && o.Now.Sub(*o.LastIncidentAt) < time.Duration(p.Limits.CooldownSeconds)*time.Second {
		addBlock("cooldown is active")
	}
	for _, rule := range p.Rules {
		if rule.Disabled {
			continue
		}
		matched, detail := matchesRule(rule, o)
		if matched {
			decision.Matches = append(decision.Matches, Match{Kind: rule.Kind, Detail: detail})
		} else if detail == "known owner wait" {
			decision.Reasons = append(decision.Reasons, rule.Kind+" suppressed: "+detail)
		}
	}
	if len(decision.Matches) == 0 {
		decision.Eligible = false
		decision.Queued = false
		decision.Reasons = append(decision.Reasons, "no trigger rule matched")
	}
	if !decision.Eligible {
		decision.Queued = false
	}
	if digest, err := p.Digest(); err == nil {
		decision.IncidentFingerprint = incidentFingerprint(o, digest, decision.Matches)
	}
	if p.Mode == ModeShadow {
		decision.Reasons = append(decision.Reasons, "shadow mode records intent without dispatch")
	}
	return decision, nil
}

func matchesRule(rule Rule, o Observation) (bool, string) {
	switch rule.Kind {
	case "no_material_progress":
		if rule.IgnoreKnownOwnerWaits && o.KnownOwnerWait {
			return false, "known owner wait"
		}
		return !o.LastMaterialProgress.IsZero() && o.Now.Sub(o.LastMaterialProgress) >= time.Duration(rule.AfterSeconds)*time.Second, "material progress threshold exceeded"
	case "repeated_failure":
		cutoff := o.Now.Add(-time.Duration(rule.WindowSeconds) * time.Second)
		count := 0
		for _, attempt := range o.Attempts {
			if attempt.Failed && !attempt.OccurredAt.Before(cutoff) {
				count++
			}
		}
		return count >= rule.MinDistinctAttempts, fmt.Sprintf("%d failed attempts in window", count)
	case "phase_elapsed":
		return !o.PhaseStartedAt.IsZero() && o.Now.Sub(o.PhaseStartedAt) >= time.Duration(rule.AfterSeconds)*time.Second && o.NewEvidence >= rule.MinimumNewEvidence, "phase elapsed with required new evidence"
	case "budget_fraction":
		return o.BudgetTotal > 0 && float64(o.BudgetUsed)/float64(o.BudgetTotal) >= rule.Fraction, "observed budget fraction exceeded"
	case "terminal_mismatch":
		return o.TerminalMismatch, "terminal state disagrees with expected outcome"
	default:
		return false, "unknown rule"
	}
}

func incidentFingerprint(o Observation, policyDigest string, matches []Match) string {
	kinds := make([]string, 0, len(matches))
	for _, match := range matches {
		kinds = append(kinds, match.Kind)
	}
	sort.Strings(kinds)
	identity := []string{o.ExecutionID, o.PhaseID, o.PhaseGeneration}
	// A family-wide incident is valid only when the owner supplies both a
	// family identity and a typed common-failure reference. This prevents the
	// dangerous shortcut of merging every symptom in a family.
	if strings.TrimSpace(o.FamilyID) != "" && strings.TrimSpace(o.SharedFailureRef) != "" {
		identity = []string{"family:" + strings.TrimSpace(o.FamilyID), "shared:" + strings.TrimSpace(o.SharedFailureRef)}
	}
	raw := strings.Join(append(identity, policyDigest, strings.Join(kinds, ",")), "\x00")
	digest := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(digest[:])
}

func knownRule(kind string) bool {
	switch kind {
	case "no_material_progress", "repeated_failure", "phase_elapsed", "budget_fraction", "terminal_mismatch":
		return true
	default:
		return false
	}
}
