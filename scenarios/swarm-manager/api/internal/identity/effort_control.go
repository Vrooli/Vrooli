package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Delegated effort actions. An effort revision authorizes only these named
// owner actions. Every child narrows the grant and an unknown action is refused
// at admission, never inferred from a global default.
const (
	DelegatedActionAuthorChildPlans = "author-child-plans"
	DelegatedActionReviewGraph      = "review-graph"
	DelegatedActionDispatch         = "dispatch"
	DelegatedActionRepair           = "repair"
	DelegatedActionReviewEvidence   = "review-evidence"
	DelegatedActionPlanRound        = "plan-round"
)

var delegatedActionVocabulary = map[string]struct{}{
	DelegatedActionAuthorChildPlans: {},
	DelegatedActionReviewGraph:      {},
	DelegatedActionDispatch:         {},
	DelegatedActionRepair:           {},
	DelegatedActionReviewEvidence:   {},
	DelegatedActionPlanRound:        {},
}

// Sentinel errors for effort-control admission. Callers map these to their own
// transport error shapes; identity stays free of HTTP concerns.
var (
	ErrEffortIdentityMismatch   = errors.New("effort identity mismatch")
	ErrRevisionConflict         = errors.New("effort revision conflict")
	ErrUnauthorizedAction       = errors.New("delegated action is not authorized")
	ErrCandidatePolicyImmutable = errors.New("candidate policy binding is immutable")
	ErrAdditionalCapacityDenied = errors.New("effort-wide descendant allowance is exhausted")
	ErrHumanAcceptanceRequired  = errors.New("product acceptance requires an authenticated human actor")
)

// EffortScope binds the path scope of one effort revision. Deny always wins,
// and an empty allow list means "the effort workspace only", not "everything".
type EffortScope struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

func (scope EffortScope) Validate() error {
	for _, pattern := range append(append([]string{}, scope.Allow...), scope.Deny...) {
		if strings.TrimSpace(pattern) == "" {
			return fmt.Errorf("effort scope patterns must not be blank")
		}
	}
	return nil
}

// Allows reports whether a repo-relative path is inside the bound scope. A path
// matched by any deny pattern is refused before allow is consulted.
func (scope EffortScope) Allows(path string) bool {
	path = trimScopePath(path)
	if path == "" {
		return false
	}
	for _, deny := range scope.Deny {
		if scopeGlobMatches(deny, path) {
			return false
		}
	}
	if len(scope.Allow) == 0 {
		return true
	}
	for _, allow := range scope.Allow {
		if scopeGlobMatches(allow, path) {
			return true
		}
	}
	return false
}

// AggregateLimits is the reviewed, effort-wide resource and supervision bound.
// Descendant depth and active-descendant caps are shared across every branch so
// siblings and nested planners cannot mint additional capacity.
type AggregateLimits struct {
	MaxWorkers            int    `json:"max_workers"`
	MaxConcurrency        int    `json:"max_concurrency"`
	MaxDepth              int    `json:"max_depth"`
	MaxActiveDescendants  int    `json:"max_active_descendants"`
	MaxPremiumDescendants int    `json:"max_premium_descendants"`
	MaxTokens             int64  `json:"max_tokens,omitempty"`
	MaxChargeMicroUSD     int64  `json:"max_charge_micro_usd,omitempty"`
	MaxWallSeconds        int64  `json:"max_wall_seconds,omitempty"`
	Deadline              string `json:"deadline,omitempty"`
}

func (limits AggregateLimits) Validate() error {
	for _, field := range []struct {
		name           string
		value, minimum int64
	}{
		{"max_workers", int64(limits.MaxWorkers), 1},
		{"max_concurrency", int64(limits.MaxConcurrency), 1},
		{"max_active_descendants", int64(limits.MaxActiveDescendants), 1},
	} {
		if field.value < field.minimum {
			return fmt.Errorf("aggregate_limits.%s must be at least %d", field.name, field.minimum)
		}
	}
	if limits.MaxConcurrency > limits.MaxWorkers {
		return fmt.Errorf("aggregate_limits.max_concurrency cannot exceed max_workers")
	}
	if limits.MaxActiveDescendants > limits.MaxConcurrency {
		return fmt.Errorf("aggregate_limits.max_active_descendants cannot exceed max_concurrency")
	}
	if limits.MaxDepth < 0 || limits.MaxDepth > 2 {
		return fmt.Errorf("aggregate_limits.max_depth must be between 0 and 2")
	}
	if limits.MaxPremiumDescendants < 0 || limits.MaxPremiumDescendants > limits.MaxActiveDescendants {
		return fmt.Errorf("aggregate_limits.max_premium_descendants must be between 0 and max_active_descendants")
	}
	for _, field := range []struct {
		name           string
		value, maximum int64
	}{
		{"max_tokens", limits.MaxTokens, 2147483647},
		{"max_charge_micro_usd", limits.MaxChargeMicroUSD, 1000000000000},
		{"max_wall_seconds", limits.MaxWallSeconds, 604800},
	} {
		if field.value < 0 || field.value > field.maximum {
			return fmt.Errorf("aggregate_limits.%s must be between 0 and %d", field.name, field.maximum)
		}
	}
	return nil
}

// CanActivate decides whether one more descendant may start against the single
// effort-wide allowance. It is a pure reservation check; the caller applies it
// under its own owner lock.
func (limits AggregateLimits) CanActivate(activeDescendants, activePremium int, premium bool) error {
	if activeDescendants < 0 || activePremium < 0 {
		return fmt.Errorf("active descendant counts must not be negative")
	}
	if activeDescendants >= limits.MaxActiveDescendants {
		return fmt.Errorf("%w: %d of %d active descendants", ErrAdditionalCapacityDenied, activeDescendants, limits.MaxActiveDescendants)
	}
	if premium {
		if limits.MaxPremiumDescendants <= 0 {
			return fmt.Errorf("%w: no premium descendant is authorized", ErrAdditionalCapacityDenied)
		}
		if activePremium >= limits.MaxPremiumDescendants {
			return fmt.Errorf("%w: %d of %d premium descendants", ErrAdditionalCapacityDenied, activePremium, limits.MaxPremiumDescendants)
		}
	}
	return nil
}

// RepairLimits is the reviewed cumulative waste bound. It is copied from the
// approved effort policy at admission and never reset by a new plan or worker.
type RepairLimits struct {
	PerFingerprint         int `json:"per_fingerprint"`
	PerComponent           int `json:"per_component"`
	PerEffort              int `json:"per_effort"`
	ComponentActiveMinutes int `json:"component_active_minutes"`
	ProbeAttempts          int `json:"probe_attempts,omitempty"`
	TransportAttempts      int `json:"transport_attempts,omitempty"`
	StatusReads            int `json:"status_reads,omitempty"`
}

func (limits RepairLimits) Validate() error {
	for _, field := range []struct {
		name           string
		value, minimum int
	}{
		{"per_fingerprint", limits.PerFingerprint, 1},
		{"per_component", limits.PerComponent, 1},
		{"per_effort", limits.PerEffort, 1},
		{"component_active_minutes", limits.ComponentActiveMinutes, 1},
	} {
		if field.value < field.minimum {
			return fmt.Errorf("repair_limits.%s must be at least %d", field.name, field.minimum)
		}
	}
	if limits.PerComponent < limits.PerFingerprint {
		return fmt.Errorf("repair_limits.per_component cannot be less than per_fingerprint")
	}
	if limits.PerEffort < limits.PerComponent {
		return fmt.Errorf("repair_limits.per_effort cannot be less than per_component")
	}
	for _, field := range []struct {
		name  string
		value int
	}{
		{"probe_attempts", limits.ProbeAttempts},
		{"transport_attempts", limits.TransportAttempts},
		{"status_reads", limits.StatusReads},
	} {
		if field.value < 0 {
			return fmt.Errorf("repair_limits.%s must not be negative", field.name)
		}
	}
	return nil
}

// PolicyBinding retains only the owner reference to the approved policy
// revision. The effort workspace owns the writable policy; the aggregate keeps
// this read-only digest so there is never a second policy authority.
type PolicyBinding struct {
	Source string `json:"source"`
	Digest string `json:"digest"`
}

func (binding PolicyBinding) Validate() error {
	if strings.TrimSpace(binding.Source) == "" {
		return fmt.Errorf("policy_binding.source is required")
	}
	if strings.TrimSpace(binding.Digest) == "" {
		return fmt.Errorf("policy_binding.digest is required")
	}
	return nil
}

// CandidatePolicyBinding is the immutable binding of the installed runner,
// model, capability and credential-pool references resolved at admission. Its
// Digest must match ComputeDigest, so a later edit cannot silently retarget
// dispatch within the same revision.
type CandidatePolicyBinding struct {
	EconomicalRunner string   `json:"economical_runner"`
	EconomicalModel  string   `json:"economical_model"`
	EconomicalEffort string   `json:"economical_effort"`
	FallbackRunner   string   `json:"fallback_runner,omitempty"`
	FallbackModel    string   `json:"fallback_model,omitempty"`
	PremiumRunner    string   `json:"premium_runner,omitempty"`
	PremiumModel     string   `json:"premium_model,omitempty"`
	CredentialPool   string   `json:"credential_pool,omitempty"`
	Capabilities     []string `json:"capabilities,omitempty"`
	Withheld         []string `json:"withheld,omitempty"`
	Digest           string   `json:"digest,omitempty"`
}

// Bind stamps the canonical digest onto the binding.
func (binding CandidatePolicyBinding) Bind() CandidatePolicyBinding {
	binding.Digest = binding.ComputeDigest()
	return binding
}

// ComputeDigest canonicalizes ordering before hashing so equivalent bindings
// carry the same identity regardless of how the slices were assembled.
func (binding CandidatePolicyBinding) ComputeDigest() string {
	projection := binding
	projection.Digest = ""
	projection.Capabilities = sortedUnique(projection.Capabilities)
	projection.Withheld = sortedUnique(projection.Withheld)
	raw, _ := json.Marshal(projection)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func (binding CandidatePolicyBinding) Validate() error {
	if strings.TrimSpace(binding.EconomicalRunner) == "" {
		return fmt.Errorf("candidate_policy.economical_runner is required")
	}
	if strings.TrimSpace(binding.EconomicalModel) == "" {
		return fmt.Errorf("candidate_policy.economical_model is required")
	}
	if strings.TrimSpace(binding.Digest) == "" {
		return fmt.Errorf("candidate_policy.digest is required")
	}
	if binding.Digest != binding.ComputeDigest() {
		return ErrCandidatePolicyImmutable
	}
	if (strings.TrimSpace(binding.PremiumRunner) == "") != (strings.TrimSpace(binding.PremiumModel) == "") {
		return fmt.Errorf("candidate_policy premium runner and model must be set together")
	}
	if (strings.TrimSpace(binding.FallbackRunner) == "") != (strings.TrimSpace(binding.FallbackModel) == "") {
		return fmt.Errorf("candidate_policy fallback runner and model must be set together")
	}
	return nil
}

// WorkReference points at an existing canonical owner record. The aggregate
// retains the reference; it never copies plan phase status, family membership,
// capability inventory, or asset state.
type WorkReference struct {
	Owner string `json:"owner"`
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Role  string `json:"role,omitempty"`
}

func (reference WorkReference) Validate() error {
	for _, field := range []struct {
		name  string
		value string
	}{
		{"owner", reference.Owner},
		{"kind", reference.Kind},
		{"id", reference.ID},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("work_reference.%s is required", field.name)
		}
	}
	return nil
}

// CompletionStanding keeps automatic owner evidence completion strictly
// separate from authenticated human product acceptance. Marking evidence
// complete can halt the finite leader; it can never set HumanAccepted.
type CompletionStanding struct {
	EvidenceComplete bool     `json:"evidence_complete"`
	EvidenceRefs     []string `json:"evidence_refs,omitempty"`
	HumanAccepted    bool     `json:"human_accepted"`
	HumanActor       string   `json:"human_actor,omitempty"`
	AcceptedAt       string   `json:"accepted_at,omitempty"`
}

// MarkEvidenceComplete records owner evidence completion. It never implies
// human acceptance.
func (standing CompletionStanding) MarkEvidenceComplete(refs ...string) CompletionStanding {
	standing.EvidenceComplete = true
	standing.EvidenceRefs = sortedUnique(append(append([]string{}, standing.EvidenceRefs...), refs...))
	return standing
}

// HumanAccept records an authenticated human product disposition. A blank
// actor is refused: completion evidence is not a substitute for a human.
func (standing CompletionStanding) HumanAccept(actor string) (CompletionStanding, error) {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return standing, ErrHumanAcceptanceRequired
	}
	standing.HumanAccepted = true
	standing.HumanActor = actor
	return standing, nil
}

// EffortControl is the versioned, reviewed authority for one finite effort. It
// binds an exact revision to delegated actions, path scope, aggregate and
// repair limits, supervision allowance, canonical work references, and an
// immutable candidate-policy binding. Completion standing is runtime state kept
// in the same aggregate but excluded from the authority digest.
type EffortControl struct {
	EffortID         string                 `json:"effort_id"`
	Slug             string                 `json:"slug,omitempty"`
	Revision         int64                  `json:"revision"`
	FamilyID         string                 `json:"family_id,omitempty"`
	DelegatedActions []string               `json:"delegated_actions,omitempty"`
	Scope            EffortScope            `json:"scope"`
	AggregateLimits  AggregateLimits        `json:"aggregate_limits"`
	RepairLimits     RepairLimits           `json:"repair_limits"`
	PolicyBinding    PolicyBinding          `json:"policy_binding"`
	CandidatePolicy  CandidatePolicyBinding `json:"candidate_policy"`
	WorkReferences   []WorkReference        `json:"work_references,omitempty"`
	Completion       CompletionStanding     `json:"completion"`
}

// Validate checks the authored authority of one revision. Completion standing
// is intentionally not validated here because it is runtime state.
func (control EffortControl) Validate() error {
	if strings.TrimSpace(control.EffortID) == "" {
		return fmt.Errorf("effort_id is required")
	}
	if control.Revision < 1 {
		return fmt.Errorf("%w: revision must be at least 1", ErrRevisionConflict)
	}
	if len(control.DelegatedActions) == 0 {
		return fmt.Errorf("%w: at least one delegated action is required", ErrUnauthorizedAction)
	}
	for _, action := range control.DelegatedActions {
		if _, ok := delegatedActionVocabulary[action]; !ok {
			return fmt.Errorf("%w: %q", ErrUnauthorizedAction, action)
		}
	}
	if err := control.Scope.Validate(); err != nil {
		return err
	}
	if err := control.AggregateLimits.Validate(); err != nil {
		return err
	}
	if err := control.RepairLimits.Validate(); err != nil {
		return err
	}
	if err := control.PolicyBinding.Validate(); err != nil {
		return err
	}
	if err := control.CandidatePolicy.Validate(); err != nil {
		return err
	}
	for _, reference := range control.WorkReferences {
		if err := reference.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Authorizes reports whether the revision delegates the named action.
func (control EffortControl) Authorizes(action string) bool {
	for _, delegated := range control.DelegatedActions {
		if delegated == action {
			return true
		}
	}
	return false
}

// AuthorityDigest is the revision identity of the authored authority. It
// excludes completion standing and normalizes slice ordering, so runtime
// progress cannot masquerade as an authority change.
func (control EffortControl) AuthorityDigest() string {
	projection := control
	projection.Completion = CompletionStanding{}
	projection.DelegatedActions = sortedUnique(projection.DelegatedActions)
	projection.Scope.Allow = sortedUnique(projection.Scope.Allow)
	projection.Scope.Deny = sortedUnique(projection.Scope.Deny)
	projection.WorkReferences = sortedWorkReferences(projection.WorkReferences)
	raw, _ := json.Marshal(projection)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// VerifyCandidatePolicy refuses a binding that does not match the digest
// captured at admission.
func (control EffortControl) VerifyCandidatePolicy(binding CandidatePolicyBinding) error {
	if binding.ComputeDigest() != control.CandidatePolicy.Digest {
		return ErrCandidatePolicyImmutable
	}
	return nil
}

// Admit performs a version-checked initial admission. Admitting over an
// existing effort is a revision conflict; existing authority changes only
// through Amend.
func (control EffortControl) Admit(current *EffortControl) (EffortControl, error) {
	if err := control.Validate(); err != nil {
		return control, err
	}
	if current == nil {
		return control, nil
	}
	if control.EffortID != current.EffortID {
		return control, ErrEffortIdentityMismatch
	}
	return control, fmt.Errorf("%w: an existing effort revision must be amended", ErrRevisionConflict)
}

// Amend is the supported change path for a target or grant. It requires the
// exact next revision and preserves effort identity, so a changed target or
// grant cannot ride an unchanged revision.
func (control EffortControl) Amend(current EffortControl) (EffortControl, error) {
	if err := control.Validate(); err != nil {
		return control, err
	}
	if control.EffortID != current.EffortID {
		return control, ErrEffortIdentityMismatch
	}
	if control.Slug != "" && current.Slug != "" && control.Slug != current.Slug {
		return control, fmt.Errorf("effort slug is part of the identity and cannot be amended")
	}
	if control.Revision != current.Revision+1 {
		return control, fmt.Errorf("%w: expected revision %d, got %d", ErrRevisionConflict, current.Revision+1, control.Revision)
	}
	if current.FamilyID != "" && control.FamilyID != current.FamilyID {
		return control, fmt.Errorf("changing the owning family requires a new effort identity")
	}
	if current.CandidatePolicy.Digest != "" && control.CandidatePolicy.Digest != current.CandidatePolicy.Digest && control.PolicyBinding.Digest == current.PolicyBinding.Digest {
		return control, fmt.Errorf("a changed candidate policy binding requires a changed policy revision")
	}
	return control, nil
}

func sortedUnique(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			set[trimmed] = struct{}{}
		}
	}
	sorted := make([]string, 0, len(set))
	for value := range set {
		sorted = append(sorted, value)
	}
	sort.Strings(sorted)
	return sorted
}

func sortedWorkReferences(references []WorkReference) []WorkReference {
	if len(references) == 0 {
		return nil
	}
	sorted := append([]WorkReference(nil), references...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Owner != sorted[j].Owner {
			return sorted[i].Owner < sorted[j].Owner
		}
		if sorted[i].Kind != sorted[j].Kind {
			return sorted[i].Kind < sorted[j].Kind
		}
		return sorted[i].ID < sorted[j].ID
	})
	return sorted
}

func trimScopePath(path string) string {
	return strings.Trim(strings.ReplaceAll(path, "\\", "/"), "/")
}

// scopeGlobMatches is a conservative deny-wins matcher: a pattern matches a
// path when it equals the path or the pattern's literal root prefixes it.
func scopeGlobMatches(pattern, path string) bool {
	pattern = trimScopePath(pattern)
	if pattern == "" || pattern == "**" {
		return true
	}
	if pattern == path {
		return true
	}
	root := pattern
	if index := strings.IndexAny(pattern, "*?[{"); index >= 0 {
		root = strings.Trim(pattern[:index], "/")
	}
	if root == "" {
		return true
	}
	return path == root || strings.HasPrefix(path, root+"/")
}
