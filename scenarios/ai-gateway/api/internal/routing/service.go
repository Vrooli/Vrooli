package routing

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/gateway"
	"ai-gateway/internal/providers"
)

type Service struct {
	validator *gateway.Service
	adapters  map[string]providers.Adapter
	order     []string
	repo      Repository
	health    HealthRepository
	breaker   Breaker
	capacity  CapacityAdapter
	clock     func() time.Time
}

// Option configures optional Service collaborators without breaking callers
// that only need routing/evidence.
type Option func(*Service)

// WithHealth enables provider-health/circuit-breaker tracking. A nil repository
// leaves breaker tracking disabled.
func WithHealth(health HealthRepository) Option {
	return func(s *Service) { s.health = health }
}

// WithBreakerPolicy overrides the deterministic breaker thresholds.
func WithBreakerPolicy(policy BreakerPolicy) Option {
	return func(s *Service) { s.breaker = NewBreaker(policy) }
}

// WithCapacity enables capacity-aware local route eligibility. A nil adapter
// leaves capacity gating disabled (local routes proceed unconditionally).
func WithCapacity(capacity CapacityAdapter) Option {
	return func(s *Service) { s.capacity = capacity }
}

// WithClock injects the time source used for breaker transitions and evidence
// timestamps so tests can drive deterministic cooldown behavior.
func WithClock(clock func() time.Time) Option {
	return func(s *Service) {
		if clock != nil {
			s.clock = clock
		}
	}
}

func NewService(adapters []providers.Adapter, repo Repository, opts ...Option) *Service {
	s := &Service{
		validator: gateway.New(),
		adapters:  map[string]providers.Adapter{},
		repo:      repo,
		breaker:   NewBreaker(DefaultBreakerPolicy()),
		clock:     time.Now,
	}
	for _, adapter := range adapters {
		name := strings.TrimSpace(strings.ToLower(adapter.Provider))
		if name == "" {
			continue
		}
		s.adapters[name] = adapter
		s.order = append(s.order, name)
	}
	sort.Strings(s.order)
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func NewSQLService(db *sql.DB, adapters []providers.Adapter) *Service {
	return NewService(adapters, NewSQLRepository(db),
		WithHealth(NewSQLHealthRepository(db)),
		WithCapacity(&CLICapacityAdapter{}))
}

func (s *Service) now() time.Time {
	if s.clock == nil {
		return time.Now()
	}
	return s.clock()
}

// effectiveBreaker returns the breaker state routing should act on for a
// candidate. It reports an empty state when health tracking is disabled so
// callers can distinguish "no tracking" from a healthy closed breaker.
func (s *Service) effectiveBreaker(ctx context.Context, provider, role string, kind sharedv1.RequestKind) BreakerState {
	if s.health == nil {
		return BreakerState("")
	}
	h, found, err := s.health.Get(ctx, HealthKey{Provider: provider, Role: role, Kind: kind})
	if err != nil || !found {
		return BreakerClosed
	}
	return s.breaker.Effective(h, s.now())
}

// recordOutcome updates persisted breaker state after an execution attempt.
// It is best-effort: a health-store write failure must not fail the caller's
// request, since routing already succeeded or failed on its own merits.
func (s *Service) recordOutcome(ctx context.Context, provider, role string, kind sharedv1.RequestKind, class FailureClass, success bool) {
	if s.health == nil {
		return
	}
	key := HealthKey{Provider: provider, Role: role, Kind: kind}
	h, _, err := s.health.Get(ctx, key)
	if err != nil {
		return
	}
	h.Provider, h.Role, h.Kind = provider, role, kind
	now := s.now()
	if success {
		h = s.breaker.OnSuccess(h, now)
	} else {
		h = s.breaker.OnFailure(h, class, now)
	}
	_ = s.health.Upsert(ctx, h)
}

// capacityOwner builds an op-scoped owner id for a capacity claim so the broker
// ledger attributes the reservation to this gateway request.
func capacityOwner(req *sharedv1.GatewayRequest) string {
	return "ai-gateway:" + firstNonEmpty(req.GetScenario(), "unknown") + ":" + firstNonEmpty(req.GetOperation(), "route")
}

func allowReclaim(req *sharedv1.GatewayRequest) bool {
	return strings.EqualFold(strings.TrimSpace(req.GetMetadata()["capacity_allow_reclaim"]), "true")
}

// probeCapacity returns a capacity verdict for a local candidate without holding
// a reservation (claim then immediate release). Used by planning/preview. A nil
// adapter or an absent footprint yields a non-blocking verdict.
func (s *Service) probeCapacity(ctx context.Context, req *sharedv1.GatewayRequest) CapacityEvaluation {
	if s.capacity == nil {
		return CapacityEvaluation{Verdict: CapacityNotEvaluated}
	}
	bytes := capacityRequirementBytes(req.GetMetadata())
	if bytes <= 0 {
		return CapacityEvaluation{Verdict: CapacityUnknown}
	}
	eval, err := s.capacity.Claim(ctx, CapacityRequest{OwnerID: capacityOwner(req), RequiredBytes: bytes, AllowReclaim: allowReclaim(req)})
	if eval.ClaimID != "" {
		s.capacity.Release(ctx, eval.ClaimID)
		eval.ClaimID = ""
	}
	if err != nil {
		return CapacityEvaluation{Verdict: CapacityUnknown, RequiredBytes: bytes}
	}
	return eval
}

// acquireCapacity holds an op-scoped claim around a local execution attempt. The
// returned evaluation carries the live ClaimID the caller must release. Non-local
// candidates, a nil adapter, or an absent footprint return a zero evaluation.
func (s *Service) acquireCapacity(ctx context.Context, req *sharedv1.GatewayRequest, candidate *routingv1.RouteCandidate) CapacityEvaluation {
	if s.capacity == nil || !strings.EqualFold(candidate.GetLocality(), "local") {
		return CapacityEvaluation{}
	}
	bytes := capacityRequirementBytes(req.GetMetadata())
	if bytes <= 0 {
		return CapacityEvaluation{Verdict: CapacityUnknown}
	}
	eval, err := s.capacity.Claim(ctx, CapacityRequest{OwnerID: capacityOwner(req), RequiredBytes: bytes, AllowReclaim: allowReclaim(req)})
	if err != nil {
		return CapacityEvaluation{Verdict: CapacityUnknown, RequiredBytes: bytes}
	}
	return eval
}

func (s *Service) releaseCapacity(ctx context.Context, eval CapacityEvaluation) {
	if s.capacity != nil && eval.ClaimID != "" {
		s.capacity.Release(ctx, eval.ClaimID)
	}
}

func capacityReason(eval CapacityEvaluation) string {
	switch eval.Verdict {
	case CapacityInsufficient:
		return "local capacity broker reports insufficient capacity for this route"
	case CapacityAdvisoryReclaimUnavailable:
		return "local route would need capacity reclaim but broker enforcement is advisory; treating local as unavailable"
	default:
		return "local capacity broker verdict: " + string(eval.Verdict)
	}
}

func (s *Service) Preview(ctx context.Context, req *sharedv1.GatewayRequest) (*routingv1.PreviewRouteResponse, error) {
	issues := s.validator.Validate(req)
	if len(issues) > 0 {
		return &routingv1.PreviewRouteResponse{Valid: false, Issues: issues}, nil
	}
	plan := s.plan(ctx, req)
	return &routingv1.PreviewRouteResponse{
		Valid:            true,
		Candidates:       plan.candidates,
		SelectedProvider: plan.selectedProvider,
		PolicyReasons:    plan.policyReasons,
		FallbackAllowed:  plan.fallbackAllowed,
		RoutePlanId:      plan.id,
	}, nil
}

func (s *Service) Execute(ctx context.Context, req *sharedv1.GatewayRequest, input string) (*routingv1.ExecuteRouteResponse, error) {
	started := time.Now()
	issues := s.validator.Validate(req)
	if strings.TrimSpace(input) == "" {
		issues = append(issues, &sharedv1.ValidationIssue{Field: "input_text", Code: "required", Message: "input_text is required for execution"})
	}
	if len(issues) > 0 {
		return &routingv1.ExecuteRouteResponse{Valid: false, Issues: issues}, nil
	}

	plan := s.plan(ctx, req)
	if plan.selectedProvider == "" {
		ev := s.evidence(req, plan, "blocked", time.Since(started), false, []string{"no eligible provider route"})
		ev.RejectionReason = blockedReason(plan)
		ev.CapacityVerdict = blockedCapacityVerdict(plan)
		if err := s.repo.Create(ctx, ev); err != nil {
			return nil, fmt.Errorf("persist blocked route evidence: %w", err)
		}
		return &routingv1.ExecuteRouteResponse{Valid: true, Evidence: ev, PolicyReasons: plan.policyReasons}, nil
	}

	attempts := s.executionAttempts(plan)
	var failures []string
	var lastFailureClass FailureClass
	for i, candidate := range attempts {
		adapter, ok := s.adapters[candidate.GetProvider()]
		if !ok {
			lastFailureClass = FailurePolicyError
			failures = append(failures, fmt.Sprintf("%s: adapter not configured", candidate.GetProvider()))
			continue
		}
		ctxExec := ctx
		cancel := func() {}
		if req.GetTimeoutMs() > 0 {
			ctxExec, cancel = context.WithTimeout(ctx, time.Duration(req.GetTimeoutMs())*time.Millisecond)
		}
		// Hold an op-scoped capacity claim around a local execution attempt so the
		// broker ledger shows the reservation as CLAIMED. Release is best-effort in
		// all paths; a crash falls back to the claim's bounded TTL.
		capEval := s.acquireCapacity(ctxExec, req, candidate)
		result, err := adapter.Execute(ctxExec, providers.ExecutionRequest{
			Kind:            req.GetKind(),
			Role:            req.GetRole(),
			InputText:       input,
			MaxOutputTokens: req.GetMaxOutputTokens(),
			Timeout:         timeoutDuration(req.GetTimeoutMs()),
		})
		s.releaseCapacity(ctxExec, capEval)
		cancel()
		if err != nil {
			class := ClassifyProviderError(err)
			lastFailureClass = class
			s.recordOutcome(ctx, candidate.GetProvider(), candidate.GetRole(), req.GetKind(), class, false)
			failures = append(failures, fmt.Sprintf("%s: %v", candidate.GetProvider(), err))
			if i == 0 && !plan.fallbackAllowed {
				break
			}
			continue
		}
		s.recordOutcome(ctx, candidate.GetProvider(), candidate.GetRole(), req.GetKind(), FailureNone, true)
		fallbackUsed := i > 0
		ev := s.evidence(req, plan, "succeeded", time.Since(started), fallbackUsed, failures)
		ev.SelectedProvider = candidate.GetProvider()
		ev.SelectedLocality = candidate.GetLocality()
		ev.BreakerState = candidate.GetBreakerState()
		applyCapacityEvidence(ev, candidate, capEval)
		if err := s.repo.Create(ctx, ev); err != nil {
			return nil, fmt.Errorf("persist successful route evidence: %w", err)
		}
		return &routingv1.ExecuteRouteResponse{
			Valid:         true,
			Evidence:      ev,
			OutputText:    result.OutputText,
			PolicyReasons: plan.policyReasons,
		}, nil
	}

	ev := s.evidence(req, plan, "failed", time.Since(started), len(failures) > 1, failures)
	ev.FailureClass = string(lastFailureClass)
	if err := s.repo.Create(ctx, ev); err != nil {
		return nil, fmt.Errorf("persist failed route evidence: %w", err)
	}
	return &routingv1.ExecuteRouteResponse{Valid: true, Evidence: ev, PolicyReasons: plan.policyReasons}, nil
}

func (s *Service) ListEvidence(ctx context.Context, filter EvidenceFilter) ([]*routingv1.RouteEvidence, error) {
	return s.repo.List(ctx, filter)
}

func (s *Service) GetEvidence(ctx context.Context, eventID string) (*routingv1.RouteEvidence, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return nil, fmt.Errorf("event_id is required")
	}
	ev, err := s.repo.Get(ctx, eventID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("route evidence %q was not found", eventID)
	}
	return ev, err
}

// ListProviderHealth returns persisted breaker records with the effective state
// computed at read time. Returns an empty slice when health tracking is disabled.
func (s *Service) ListProviderHealth(ctx context.Context) ([]*routingv1.ProviderHealth, error) {
	if s.health == nil {
		return nil, nil
	}
	records, err := s.health.List(ctx)
	if err != nil {
		return nil, err
	}
	now := s.now()
	out := make([]*routingv1.ProviderHealth, 0, len(records))
	for _, h := range records {
		out = append(out, &routingv1.ProviderHealth{
			Provider:            h.Provider,
			Role:                h.Role,
			Kind:                h.Kind,
			State:               string(stateOrClosed(h.State)),
			EffectiveState:      string(s.breaker.Effective(h, now)),
			ConsecutiveFailures: int64(h.ConsecutiveFailures),
			LastFailureClass:    string(h.LastFailureClass),
			LastSuccessAt:       formatTime(h.LastSuccessAt),
			LastFailureAt:       formatTime(h.LastFailureAt),
			CooldownUntil:       formatTime(h.CooldownUntil),
			OpenedAt:            formatTime(h.OpenedAt),
			Generation:          h.Generation,
			UpdatedAt:           formatTime(h.UpdatedAt),
		})
	}
	return out, nil
}

type routePlan struct {
	id               string
	candidates       []*routingv1.RouteCandidate
	selectedProvider string
	selectedLocality string
	policyReasons    []string
	fallbackAllowed  bool
}

// classifyCandidate resolves one provider into a route candidate and reports
// whether it is eligible. Rejected candidates carry a stable rejection_reason.
// The evaluation order is hard policy first (role/capability/locality), then
// provider breaker state, then local capacity — never opaque scoring.
func (s *Service) classifyCandidate(ctx context.Context, req *sharedv1.GatewayRequest, providerName string) (*routingv1.RouteCandidate, bool) {
	adapter := s.adapters[providerName]
	role, rejected := policyCandidate(ctx, req, providerName, adapter)
	if rejected != nil {
		return rejected, false
	}
	candidate := &routingv1.RouteCandidate{
		Provider: providerName,
		Role:     role.Role,
		Locality: role.Locality,
		Reasons:  []string{"eligible by role capability and profile policy"},
	}
	switch state := s.effectiveBreaker(ctx, providerName, role.Role, req.GetKind()); state {
	case BreakerOpen:
		candidate.BreakerState = string(state)
		candidate.RejectionReason = "provider_breaker_open"
		candidate.Reasons = []string{"provider circuit breaker is open; skipping until cooldown elapses"}
		return candidate, false
	case BreakerHalfOpen:
		candidate.BreakerState = string(state)
		candidate.HalfOpenProbe = true
		candidate.Reasons = append(candidate.Reasons, "provider breaker is half-open; eligible as a bounded recovery probe")
	default:
		candidate.BreakerState = string(state)
	}
	if !s.capacityAdmits(ctx, req, candidate) {
		return candidate, false
	}
	return candidate, true
}

// policyCandidate applies the hard policy filters (inventory/role/capability/
// locality) and returns either the resolved role (ok) or a rejected candidate.
func policyCandidate(ctx context.Context, req *sharedv1.GatewayRequest, providerName string, adapter providers.Adapter) (providers.Role, *routingv1.RouteCandidate) {
	inventory, err := adapter.ListRoles(ctx)
	if err != nil {
		return providers.Role{}, rejectedCandidate(providerName, req.GetRole(), adapter.Locality,
			fmt.Sprintf("inventory unavailable: %v", err), "inventory_unavailable")
	}
	role, ok := findRole(inventory.Roles, req.GetRole())
	if !ok {
		return providers.Role{}, rejectedCandidate(providerName, req.GetRole(), adapter.Locality,
			"role is not exposed by provider policy", "role_not_exposed")
	}
	if !roleSupportsKind(role, req.GetKind()) {
		return providers.Role{}, rejectedCandidate(providerName, role.Role, role.Locality,
			"role capabilities do not satisfy request kind", "capability_mismatch")
	}
	if reason := localityRejection(req, role.Locality); reason != "" {
		return providers.Role{}, rejectedCandidate(providerName, role.Role, role.Locality, reason, "locality_forbidden")
	}
	return role, nil
}

// capacityAdmits evaluates local capacity for a local candidate, annotates its
// verdict, and reports whether it stays eligible. Remote candidates, a nil
// adapter, or an absent footprint always admit.
func (s *Service) capacityAdmits(ctx context.Context, req *sharedv1.GatewayRequest, candidate *routingv1.RouteCandidate) bool {
	if !strings.EqualFold(candidate.GetLocality(), "local") || s.capacity == nil {
		return true
	}
	eval := s.probeCapacity(ctx, req)
	candidate.CapacityVerdict = string(eval.Verdict)
	if eval.Verdict.blocksLocal() {
		candidate.RejectionReason = "insufficient_capacity"
		candidate.Reasons = append(candidate.Reasons, capacityReason(eval))
		return false
	}
	return true
}

func rejectedCandidate(provider, role, locality, reason, code string) *routingv1.RouteCandidate {
	return &routingv1.RouteCandidate{
		Provider:        provider,
		Role:            role,
		Locality:        locality,
		Reasons:         []string{reason},
		RejectionReason: code,
	}
}

func (s *Service) plan(ctx context.Context, req *sharedv1.GatewayRequest) routePlan {
	plan := routePlan{id: newID("plan"), policyReasons: []string{profileReason(req.GetProfile())}}
	var eligible []*routingv1.RouteCandidate
	for _, providerName := range s.order {
		candidate, ok := s.classifyCandidate(ctx, req, providerName)
		if ok {
			eligible = append(eligible, candidate)
		} else {
			plan.candidates = append(plan.candidates, candidate)
		}
	}

	sort.SliceStable(eligible, func(i, j int) bool {
		return rank(req.GetProfile(), eligible[i].GetLocality(), eligible[i].GetProvider()) <
			rank(req.GetProfile(), eligible[j].GetLocality(), eligible[j].GetProvider())
	})
	if len(eligible) > 0 {
		eligible[0].Selected = true
		plan.selectedProvider = eligible[0].GetProvider()
		plan.selectedLocality = eligible[0].GetLocality()
		plan.policyReasons = append(plan.policyReasons, fmt.Sprintf("selected %s because it is the highest-ranked eligible %s provider", plan.selectedProvider, plan.selectedLocality))
	}
	for i, candidate := range eligible {
		if i > 0 {
			candidate.FallbackEligible = true
			plan.fallbackAllowed = true
		}
		plan.candidates = append(plan.candidates, candidate)
	}
	if len(eligible) == 0 {
		plan.policyReasons = append(plan.policyReasons, "no provider satisfied role, capability, locality, and privacy constraints")
	}
	sort.SliceStable(plan.candidates, func(i, j int) bool {
		if plan.candidates[i].GetSelected() != plan.candidates[j].GetSelected() {
			return plan.candidates[i].GetSelected()
		}
		if plan.candidates[i].GetFallbackEligible() != plan.candidates[j].GetFallbackEligible() {
			return plan.candidates[i].GetFallbackEligible()
		}
		return plan.candidates[i].GetProvider() < plan.candidates[j].GetProvider()
	})
	return plan
}

// applyCapacityEvidence records the capacity verdict and any acquired claim on
// route evidence. The execution claim (capEval) is preferred; when no claim was
// acquired (remote route or absent footprint) the planning verdict on the
// selected candidate is used.
func applyCapacityEvidence(ev *routingv1.RouteEvidence, candidate *routingv1.RouteCandidate, capEval CapacityEvaluation) {
	verdict := string(capEval.Verdict)
	if verdict == "" {
		verdict = candidate.GetCapacityVerdict()
	}
	ev.CapacityVerdict = verdict
	ev.CapacityClaimId = capEval.ClaimID
	ev.CapacityRequiredBytes = capEval.RequiredBytes
	ev.CapacityGrantedBytes = capEval.GrantedBytes
	ev.CapacityReclaimRequired = capEval.ReclaimRequired
}

// blockedReason derives a stable rejection code for a route that selected no
// provider. If every candidate that reached health evaluation was suppressed by
// an open breaker, it reports that specifically; otherwise the route simply had
// no eligible provider.
func blockedReason(plan routePlan) string {
	reasons := map[string]bool{}
	for _, c := range plan.candidates {
		if r := c.GetRejectionReason(); r != "" {
			reasons[r] = true
		}
	}
	// Report the most actionable cause first. A local route that failed capacity
	// or an open breaker is more informative than the expected locality rejection
	// of the remote peer that the profile forbids.
	for _, r := range []string{"insufficient_capacity", "provider_breaker_open"} {
		if reasons[r] {
			return r
		}
	}
	return "no_eligible_route"
}

// blockedCapacityVerdict returns the capacity verdict of a capacity-rejected
// candidate so blocked evidence can carry it. Empty when capacity was not the
// cause.
func blockedCapacityVerdict(plan routePlan) string {
	for _, c := range plan.candidates {
		if c.GetRejectionReason() == "insufficient_capacity" {
			return c.GetCapacityVerdict()
		}
	}
	return ""
}

func (s *Service) executionAttempts(plan routePlan) []*routingv1.RouteCandidate {
	var out []*routingv1.RouteCandidate
	for _, candidate := range plan.candidates {
		if candidate.GetSelected() {
			out = append([]*routingv1.RouteCandidate{candidate}, out...)
			continue
		}
		if candidate.GetFallbackEligible() {
			out = append(out, candidate)
		}
	}
	return out
}

func (s *Service) evidence(req *sharedv1.GatewayRequest, plan routePlan, status string, latency time.Duration, fallback bool, failures []string) *routingv1.RouteEvidence {
	return &routingv1.RouteEvidence{
		EventId:          newID("rt"),
		RequestId:        firstNonEmpty(req.GetRequestId(), newID("req")),
		Scenario:         strings.TrimSpace(req.GetScenario()),
		Operation:        strings.TrimSpace(req.GetOperation()),
		Role:             strings.TrimSpace(req.GetRole()),
		Profile:          req.GetProfile(),
		PrivacyClass:     req.GetPrivacyClass(),
		SelectedProvider: plan.selectedProvider,
		SelectedLocality: plan.selectedLocality,
		Status:           status,
		PolicyReasons:    append([]string{}, plan.policyReasons...),
		FailureReasons:   append([]string{}, failures...),
		FallbackUsed:     fallback,
		PromptRedacted:   true,
		ResponseRedacted: true,
		LatencyMs:        latency.Milliseconds(),
		CreatedAt:        nowUTC(),
	}
}

func findRole(roles []providers.Role, role string) (providers.Role, bool) {
	for _, candidate := range roles {
		if candidate.Role == role {
			return candidate, true
		}
	}
	return providers.Role{}, false
}

func roleSupportsKind(role providers.Role, kind sharedv1.RequestKind) bool {
	caps := map[string]struct{}{}
	for _, cap := range role.Capabilities {
		caps[strings.ToLower(strings.TrimSpace(cap))] = struct{}{}
	}
	switch kind {
	case sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION:
		return hasAny(caps, "generate", "chat", "completion")
	case sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING:
		return hasAny(caps, "embedding", "embeddings")
	case sharedv1.RequestKind_REQUEST_KIND_STRUCTURED_EXTRACTION:
		return hasAny(caps, "extract", "structured", "json", "tools", "generate", "chat")
	default:
		return false
	}
}

func hasAny(set map[string]struct{}, values ...string) bool {
	for _, value := range values {
		if _, ok := set[value]; ok {
			return true
		}
	}
	return false
}

func localityRejection(req *sharedv1.GatewayRequest, locality string) string {
	locality = strings.ToLower(strings.TrimSpace(locality))
	if req.GetPrivacyClass() == sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET && locality == "remote" {
		return "secret requests cannot route to remote providers"
	}
	switch req.GetProfile() {
	case sharedv1.Profile_PROFILE_LOCAL_ONLY:
		if locality != "local" {
			return "local-only profile forbids remote providers"
		}
	case sharedv1.Profile_PROFILE_REMOTE_ONLY:
		if locality != "remote" {
			return "remote-only profile forbids local providers"
		}
	case sharedv1.Profile_PROFILE_PRIVACY_SENSITIVE:
		if req.GetPrivacyClass() >= sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL && locality != "local" {
			return "privacy-sensitive confidential requests require local providers"
		}
	}
	return ""
}

func rank(profile sharedv1.Profile, locality, provider string) int {
	locality = strings.ToLower(strings.TrimSpace(locality))
	base := 10
	switch profile {
	case sharedv1.Profile_PROFILE_REMOTE_ONLY:
		if locality == "remote" {
			base = 0
		}
	case sharedv1.Profile_PROFILE_QUALITY_FIRST:
		if locality == "remote" {
			base = 0
		} else {
			base = 5
		}
	case sharedv1.Profile_PROFILE_LOCAL_ONLY,
		sharedv1.Profile_PROFILE_LOCAL_FIRST,
		sharedv1.Profile_PROFILE_CHEAP_FIRST,
		sharedv1.Profile_PROFILE_PRIVACY_SENSITIVE:
		if locality == "local" {
			base = 0
		} else {
			base = 5
		}
	}
	if provider == providers.ProviderOllama {
		return base
	}
	return base + 1
}

func profileReason(profile sharedv1.Profile) string {
	switch profile {
	case sharedv1.Profile_PROFILE_LOCAL_ONLY:
		return "local-only profile admits only local providers"
	case sharedv1.Profile_PROFILE_LOCAL_FIRST:
		return "local-first profile ranks local providers before remote providers"
	case sharedv1.Profile_PROFILE_REMOTE_ONLY:
		return "remote-only profile admits only remote providers"
	case sharedv1.Profile_PROFILE_QUALITY_FIRST:
		return "quality-first profile ranks remote providers before local fallbacks"
	case sharedv1.Profile_PROFILE_CHEAP_FIRST:
		return "cheap-first profile ranks local providers before remote providers"
	case sharedv1.Profile_PROFILE_PRIVACY_SENSITIVE:
		return "privacy-sensitive profile ranks local providers and blocks remote routing for confidential data"
	default:
		return "unspecified profile"
	}
}

func timeoutDuration(timeoutMs int32) time.Duration {
	if timeoutMs <= 0 {
		return 0
	}
	return time.Duration(timeoutMs) * time.Millisecond
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}

func profileEnum(v int32) sharedv1.Profile {
	return sharedv1.Profile(v)
}

func privacyEnum(v int32) sharedv1.PrivacyClass {
	return sharedv1.PrivacyClass(v)
}
