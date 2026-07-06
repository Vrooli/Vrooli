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
}

func NewService(adapters []providers.Adapter, repo Repository) *Service {
	s := &Service{
		validator: gateway.New(),
		adapters:  map[string]providers.Adapter{},
		repo:      repo,
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
	return s
}

func NewSQLService(db *sql.DB, adapters []providers.Adapter) *Service {
	return NewService(adapters, NewSQLRepository(db))
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
		if err := s.repo.Create(ctx, ev); err != nil {
			return nil, fmt.Errorf("persist blocked route evidence: %w", err)
		}
		return &routingv1.ExecuteRouteResponse{Valid: true, Evidence: ev, PolicyReasons: plan.policyReasons}, nil
	}

	attempts := s.executionAttempts(plan)
	var failures []string
	for i, candidate := range attempts {
		adapter, ok := s.adapters[candidate.GetProvider()]
		if !ok {
			failures = append(failures, fmt.Sprintf("%s: adapter not configured", candidate.GetProvider()))
			continue
		}
		ctxExec := ctx
		cancel := func() {}
		if req.GetTimeoutMs() > 0 {
			ctxExec, cancel = context.WithTimeout(ctx, time.Duration(req.GetTimeoutMs())*time.Millisecond)
		}
		result, err := adapter.Execute(ctxExec, providers.ExecutionRequest{
			Kind:            req.GetKind(),
			Role:            req.GetRole(),
			InputText:       input,
			MaxOutputTokens: req.GetMaxOutputTokens(),
			Timeout:         timeoutDuration(req.GetTimeoutMs()),
		})
		cancel()
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", candidate.GetProvider(), err))
			if i == 0 && !candidate.GetFallbackEligible() {
				break
			}
			continue
		}
		fallbackUsed := i > 0
		ev := s.evidence(req, plan, "succeeded", time.Since(started), fallbackUsed, failures)
		ev.SelectedProvider = candidate.GetProvider()
		ev.SelectedLocality = candidate.GetLocality()
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

type routePlan struct {
	id               string
	candidates       []*routingv1.RouteCandidate
	selectedProvider string
	selectedLocality string
	policyReasons    []string
	fallbackAllowed  bool
}

func (s *Service) plan(ctx context.Context, req *sharedv1.GatewayRequest) routePlan {
	plan := routePlan{id: newID("plan"), policyReasons: []string{profileReason(req.GetProfile())}}
	var eligible []*routingv1.RouteCandidate
	for _, providerName := range s.order {
		adapter := s.adapters[providerName]
		inventory, err := adapter.ListRoles(ctx)
		if err != nil {
			plan.candidates = append(plan.candidates, &routingv1.RouteCandidate{
				Provider: providerName,
				Role:     req.GetRole(),
				Locality: adapter.Locality,
				Reasons:  []string{fmt.Sprintf("inventory unavailable: %v", err)},
			})
			continue
		}
		role, ok := findRole(inventory.Roles, req.GetRole())
		if !ok {
			plan.candidates = append(plan.candidates, &routingv1.RouteCandidate{
				Provider: providerName,
				Role:     req.GetRole(),
				Locality: adapter.Locality,
				Reasons:  []string{"role is not exposed by provider policy"},
			})
			continue
		}
		if !roleSupportsKind(role, req.GetKind()) {
			plan.candidates = append(plan.candidates, &routingv1.RouteCandidate{
				Provider: providerName,
				Role:     role.Role,
				Locality: role.Locality,
				Reasons:  []string{"role capabilities do not satisfy request kind"},
			})
			continue
		}
		if reason := localityRejection(req, role.Locality); reason != "" {
			plan.candidates = append(plan.candidates, &routingv1.RouteCandidate{
				Provider: providerName,
				Role:     role.Role,
				Locality: role.Locality,
				Reasons:  []string{reason},
			})
			continue
		}
		candidate := &routingv1.RouteCandidate{
			Provider: providerName,
			Role:     role.Role,
			Locality: role.Locality,
			Reasons:  []string{"eligible by role capability and profile policy"},
		}
		eligible = append(eligible, candidate)
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
