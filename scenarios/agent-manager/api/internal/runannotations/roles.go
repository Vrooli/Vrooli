package runannotations

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

// This file owns which model a label is asked of: the per-label decision role,
// the grouping that batches labels by that role, the learned fact that an
// escalation role answers with the same model as the first pass (so it is not
// asked twice), and the sizing of digests and batches from the declared limits
// of the model that will actually read them.

// labelRole is the decision role a label asks: its own, else the pack default.
func (s *Service) labelRole(label Label) string {
	if role := strings.TrimSpace(label.Role); role != "" {
		return role
	}
	if role := strings.TrimSpace(s.pack.Pack.Runtime.DefaultRole); role != "" {
		return role
	}
	return DefaultDecisionRole
}

// labelEscalateRole is the stronger role a label may be re-asked with: its own,
// else the pack's. Empty means the label does not escalate.
func (s *Service) labelEscalateRole(label Label) string {
	if role := strings.TrimSpace(label.EscalateRole); role != "" {
		return role
	}
	return strings.TrimSpace(s.pack.Pack.Runtime.EscalateRole)
}

// Affinity is the scheduler's model-affinity key: the distinct decision roles
// the pack's active decision labels resolve to, sorted and joined. It follows
// the roles actually in use, so re-routing a label changes the affinity and a
// separate pack-wide setting can never drift from them.
func (s *Service) Affinity() string {
	if s == nil || s.pack == nil {
		return ""
	}
	seen := map[string]bool{}
	var roles []string
	for _, label := range s.pack.Pack.Labels {
		if label.Optional || (label.Method != MethodDecision && label.Fallback != MethodDecision) {
			continue
		}
		if role := s.labelRole(label); !seen[role] {
			seen[role] = true
			roles = append(roles, role)
		}
	}
	sort.Strings(roles)
	return strings.Join(roles, "+")
}

// decisionGroup is the labels answered by one (role, escalation role): one
// model's batch, in pack order.
type decisionGroup struct {
	role     string
	escalate string
	labels   []string
}

// groupDecisionLabels groups pending labels by the role they ask. A pinned pass
// asks every label of the one walk role and never escalates.
func (s *Service) groupDecisionLabels(pass *passState, labelIDs []string) []decisionGroup {
	byKey := map[string]*decisionGroup{}
	var order []string
	for _, labelID := range labelIDs {
		compiled, ok := s.pack.Label(labelID)
		if !ok {
			continue
		}
		role, escalate := s.labelRole(compiled.label), s.labelEscalateRole(compiled.label)
		if pass.pinned() {
			role, escalate = pass.walkRole(role), ""
		}
		key := role + "\x00" + escalate
		group := byKey[key]
		if group == nil {
			group = &decisionGroup{role: role, escalate: escalate}
			byKey[key] = group
			order = append(order, key)
		}
		group.labels = append(group.labels, labelID)
	}
	groups := make([]decisionGroup, 0, len(order))
	for _, key := range order {
		groups = append(groups, *byKey[key])
	}
	return groups
}

// roleResolution is the model a role last answered with.
type roleResolution struct {
	provider     string
	model        string
	resourceRole string
	at           time.Time
}

// resolver remembers what the gateway resolved each role to. It is learned from
// answers, not configured: the gateway owns the candidate walk, so the engine
// only trusts what it observed, and only for a bounded time.
type resolver struct {
	mu       sync.Mutex
	resolved map[string]roleResolution
	futile   map[string]time.Time
	// bootstrapped marks roles whose stored answers were already consulted, so a
	// role that has never answered is not looked up on every call.
	bootstrapped map[string]bool
}

func newResolver() *resolver {
	return &resolver{resolved: map[string]roleResolution{}, futile: map[string]time.Time{}, bootstrapped: map[string]bool{}}
}

// note records the model a successful decision resolved its role to.
func (r *resolver) note(role string, result *inferencev1.DecideResponse, now time.Time) {
	if result == nil || result.GetModel() == "" || role == "" {
		return
	}
	r.mu.Lock()
	r.resolved[role] = roleResolution{provider: result.GetProvider(), model: result.GetModel(), resourceRole: result.GetResourceRole(), at: now}
	r.mu.Unlock()
}

func (r *resolver) resolution(role string) (resolved roleResolution, known, tried bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	resolved, known = r.resolved[role]
	return resolved, known, r.bootstrapped[role]
}

// seed records a resolution read from stored answers. It is stamped with the
// time of the answer it came from, so it never counts as a fresh resolution for
// the same-model check, only as the best available size for the first call.
func (r *resolver) seed(role string, resolved roleResolution) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bootstrapped[role] = true
	if _, known := r.resolved[role]; !known && resolved.model != "" {
		r.resolved[role] = resolved
	}
}

// markFutile remembers that escalating from role to escalate answered with the
// same model, until now+ttl.
func (r *resolver) markFutile(role, escalate string, now time.Time, ttl time.Duration) {
	r.mu.Lock()
	r.futile[role+">"+escalate] = now.Add(ttl)
	r.mu.Unlock()
}

// sameModel reports whether asking escalate after role would ask the same model
// again: the same role, a remembered futile escalation, or two fresh
// resolutions that agree.
func (r *resolver) sameModel(role, escalate string, now time.Time, ttl time.Duration) bool {
	if role == escalate {
		return true
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if expires, ok := r.futile[role+">"+escalate]; ok {
		if now.Before(expires) {
			return true
		}
		delete(r.futile, role+">"+escalate)
	}
	first, okFirst := r.resolved[role]
	second, okSecond := r.resolved[escalate]
	if !okFirst || !okSecond || now.Sub(first.at) > ttl || now.Sub(second.at) > ttl {
		return false
	}
	return first.model == second.model && first.resourceRole == second.resourceRole
}

// EscalationStats counts what escalation did in one pass, so the saving from
// never re-asking the same model is measured rather than assumed.
type EscalationStats struct {
	// EligibleStates are states with an answer below the confidence floor.
	EligibleStates int `json:"eligible_states"`
	// Calls are escalation requests actually sent, and StatesAsked the states in them.
	Calls       int `json:"calls"`
	StatesAsked int `json:"states_asked"`
	// SkippedStates and SkippedCalls are the inference not spent because the
	// escalation role resolves to the model that already answered.
	SkippedStates int `json:"skipped_states"`
	SkippedCalls  int `json:"skipped_calls"`
	// SameModelAnswers are states an escalation call answered with the model
	// that had already answered: the one wasted call that teaches the engine.
	SameModelAnswers int `json:"same_model_answers"`
	Adopted          int `json:"adopted"`
	// CallSeconds is the wall time of the escalation calls that were made.
	CallSeconds float64 `json:"call_seconds"`
}

func (e *EscalationStats) add(other EscalationStats) {
	e.EligibleStates += other.EligibleStates
	e.Calls += other.Calls
	e.StatesAsked += other.StatesAsked
	e.SkippedStates += other.SkippedStates
	e.SkippedCalls += other.SkippedCalls
	e.SameModelAnswers += other.SameModelAnswers
	e.Adopted += other.Adopted
	e.CallSeconds += other.CallSeconds
}

// passState is what every unit of one evaluation pass shares: where it writes,
// whether it is a pinned bake-off, and what escalation did.
type passState struct {
	store *Store
	// pin is the resource role a bake-off pass is restricted to; empty is a
	// production pass. role is the decision role walked for a pinned pass.
	pin  string
	role string

	mu         sync.Mutex
	escalation EscalationStats
}

func (p *passState) pinned() bool { return p != nil && p.pin != "" }

// walkRole is the decision role a request names: the pinned pass's role when
// one was given, else the label's.
func (p *passState) walkRole(labelRole string) string {
	if p.pinned() && strings.TrimSpace(p.role) != "" {
		return strings.TrimSpace(p.role)
	}
	return labelRole
}

func (p *passState) addEscalation(delta EscalationStats) {
	p.mu.Lock()
	p.escalation.add(delta)
	p.mu.Unlock()
}

func (p *passState) escalationStats() EscalationStats {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.escalation
}

const (
	// charsPerToken is a deliberately conservative estimate of text density for
	// sizing a state under a token window.
	charsPerToken = 3
	// promptOverheadChars covers the fixed wrapper a decision model adds around
	// the state and the questions.
	promptOverheadChars = 600
	// minDigestChars is the smallest state worth sending.
	minDigestChars = 400
)

// resolutionOf is what a decision role last resolved to: learned from an answer
// in this process, else from the stored answers a previous process left.
func (s *Service) resolutionOf(ctx context.Context, role string) (roleResolution, bool) {
	resolved, known, tried := s.roles.resolution(role)
	if known || tried {
		return resolved, known
	}
	provider, model, resourceRole, found, err := s.store.LastResolution(ctx, role)
	if err != nil || !found {
		s.roles.seed(role, roleResolution{})
		return roleResolution{}, false
	}
	seeded := roleResolution{provider: provider, model: model, resourceRole: resourceRole}
	s.roles.seed(role, seeded)
	return seeded, true
}

// roleLimits returns the declared limits of the model a decision role resolves
// to, or zero values when none are declared or the role has not answered yet.
// The gateway owns the candidate walk, so the resource role is learned from the
// last answer (or, for a pinned pass, is the pinned candidate); the first call
// to a role is sized by the pack and every later one by the model.
func (s *Service) roleLimits(ctx context.Context, pass *passState, role string) ModelLimits {
	if s.limits == nil {
		return ModelLimits{}
	}
	provider, resourceRole := "", pass.pin
	if !pass.pinned() {
		resolved, ok := s.resolutionOf(ctx, role)
		if !ok {
			return ModelLimits{}
		}
		provider, resourceRole = resolved.provider, resolved.resourceRole
	}
	limits, _ := s.limits.ResourceLimits(ctx, provider, resourceRole)
	return limits
}

// questionCap is how many questions one call may carry: the pack's cap, lowered
// to the model's own when it declares one.
func (s *Service) questionCap(limits ModelLimits) int {
	capacity := s.pack.Pack.Runtime.Batch.MaxQuestions
	if limits.MaxQuestions > 0 && limits.MaxQuestions < capacity {
		capacity = limits.MaxQuestions
	}
	return max(capacity, 1)
}

// digestBudget is the largest state, in characters, one call to a role may
// carry alongside the questions it asks. The pack's digest size is the cap; a
// model that declares a smaller window lowers it. Overflow handling remains the
// safety net for a window that was declared too generously.
func (s *Service) digestBudget(limits ModelLimits, questions []*inferencev1.DecisionQuestion) int {
	budget := s.digestLimit
	if limits.ContextTokens <= 0 {
		return budget
	}
	used := promptOverheadChars
	for _, question := range questions {
		used += len(question.GetId()) + len(question.GetInstructions()) + len(question.GetTrueDescription()) + len(question.GetFalseDescription())
		for _, option := range question.GetOptions() {
			used += len(option.GetLabel()) + len(option.GetDescription())
		}
		for _, level := range question.GetLevels() {
			used += len(level)
		}
	}
	fits := limits.ContextTokens*charsPerToken - used
	if fits < budget {
		budget = max(fits, minDigestChars)
	}
	return budget
}

// exceedsLimits reports a question the role's model cannot represent: more
// options or score levels than it declares. Asking anyway would overflow or
// truncate the answer.
func exceedsLimits(label Label, limits ModelLimits) bool {
	question := label.Question
	if question == nil {
		return false
	}
	switch question.Type {
	case QuestionChoice:
		return limits.MaxOptions > 0 && len(question.Options) > limits.MaxOptions
	case QuestionScore:
		cap := limits.ScoreLevels
		if cap == 0 {
			cap = limits.MaxOptions
		}
		return cap > 0 && len(question.Levels) > cap
	}
	return false
}
