package vision_navigation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/sirupsen/logrus"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"

	"github.com/vrooli/browser-automation-studio/services/entitlement"
	"github.com/vrooli/browser-automation-studio/services/vision"
)

// service implements aiconnect.VisionNavigationServiceHandler.
type service struct {
	deps Deps
}

type normalizedStartNavigationRequest struct {
	sessionID string
	prompt    string
	model     string
	maxSteps  int
}

func normalizeStartNavigationRequest(msg *aiv1.StartNavigationRequest) (normalizedStartNavigationRequest, error) {
	sessionID := strings.TrimSpace(msg.GetSessionId())
	if sessionID == "" {
		return normalizedStartNavigationRequest{}, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id is required"))
	}
	prompt := strings.TrimSpace(msg.GetPrompt())
	if prompt == "" {
		return normalizedStartNavigationRequest{}, connect.NewError(connect.CodeInvalidArgument, errors.New("prompt is required"))
	}
	model := strings.TrimSpace(msg.GetModel())
	if model == "" {
		return normalizedStartNavigationRequest{}, connect.NewError(connect.CodeInvalidArgument, errors.New("model is required"))
	}

	maxSteps := int(msg.GetMaxSteps())
	if maxSteps <= 0 {
		maxSteps = 20
	}
	if maxSteps > 100 {
		maxSteps = 100
	}
	return normalizedStartNavigationRequest{sessionID: sessionID, prompt: prompt, model: model, maxSteps: maxSteps}, nil
}

func validateNavigationTaskContract(msg *aiv1.StartNavigationRequest, navigatorType vision.NavigatorType) error {
	if msg.EffectPolicy != "" && msg.EffectPolicy != "explicit" && msg.EffectPolicy != "read_only" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported effect_policy"))
	}
	if len(msg.Postconditions) > 16 || len(msg.Extraction) > 16 {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("at most 16 postconditions and extractions"))
	}
	if navigatorType != vision.NavigatorPlaywright && (msg.EffectPolicy == "read_only" || len(msg.Postconditions) > 0 || len(msg.Extraction) > 0) {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("selected navigator does not support enforced task contracts"))
	}
	return nil
}

func buildNavigationPostconditions(items []*aiv1.NavigationPostcondition) ([]vision.NavigationPostcondition, error) {
	conditions := make([]vision.NavigationPostcondition, 0, len(items))
	for _, condition := range items {
		if condition.Selector == "" || len(condition.Selector) > 1024 || len(condition.Expected) > 4096 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid postcondition"))
		}
		switch condition.Mode {
		case "exists", "text_equals", "text_contains", "count_equals", "ASSERTION_MODE_EXISTS":
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported postcondition mode"))
		}
		conditions = append(conditions, vision.NavigationPostcondition{Selector: condition.Selector, Mode: condition.Mode, Expected: condition.Expected})
	}
	return conditions, nil
}

func buildNavigationExtractions(items []*aiv1.NavigationExtraction) ([]vision.NavigationExtraction, error) {
	extraction := make([]vision.NavigationExtraction, 0, len(items))
	names := map[string]bool{}
	for _, item := range items {
		if item.Name == "" || names[item.Name] || len(item.Name) > 128 || item.Selector == "" || len(item.Selector) > 1024 || len(item.Attribute) > 128 || item.Limit < 0 || item.Limit > 100 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid extraction"))
		}
		names[item.Name] = true
		extraction = append(extraction, vision.NavigationExtraction{Name: item.Name, Selector: item.Selector, Attribute: item.Attribute, Limit: int(item.Limit)})
	}
	return extraction, nil
}

func buildNavigationTaskContract(
	msg *aiv1.StartNavigationRequest,
	navigatorType vision.NavigatorType,
) (string, []vision.NavigationPostcondition, []vision.NavigationExtraction, error) {
	if err := validateNavigationTaskContract(msg, navigatorType); err != nil {
		return "", nil, nil, err
	}
	conditions, err := buildNavigationPostconditions(msg.Postconditions)
	if err != nil {
		return "", nil, nil, err
	}
	extraction, err := buildNavigationExtractions(msg.Extraction)
	if err != nil {
		return "", nil, nil, err
	}
	return msg.EffectPolicy, conditions, extraction, nil
}

func (s *service) navigationCredentialProvenance() vision.CredentialProvenance {
	if s.deps.CredentialAuthority == nil {
		return vision.CredentialProvenanceNone
	}
	identity, err := credentialauthority.ParseIdentity("vrooli/openrouter")
	if err == nil && s.deps.CredentialAuthority.Status(identity, "api-key").Configured {
		return vision.CredentialProvenanceAuthority
	}
	return vision.CredentialProvenanceNone
}

func navigationUserID(ctx context.Context) string {
	userID := entitlement.UserIdentityFromContext(ctx)
	if userID == "" {
		return "anonymous"
	}
	return userID
}

func (s *service) authorizeNavigationCredits(
	ctx context.Context,
	navigator vision.VisionNavigator,
	provenance vision.CredentialProvenance,
) error {
	policy := navigator.CreditPolicy()
	if s.deps.Credits == nil || !policy.ShouldChargeCredits(provenance, false, false) {
		return nil
	}
	userID := navigationUserID(ctx)
	canProceed, errCode, errMsg, remaining, err := s.deps.Credits.CanPerformAIOperation(ctx, userID, policy.OperationType, provenance == vision.CredentialProvenanceAuthority)
	if err != nil {
		s.logger().WithError(err).Warn("vision_navigation: credit check failed; continuing")
		return nil
	}
	if canProceed {
		return nil
	}
	code := connect.CodeFailedPrecondition
	if errCode == "INSUFFICIENT_CREDITS" {
		code = connect.CodeResourceExhausted
	}
	return connect.NewError(code, fmt.Errorf("%s: %s (remaining=%d)", errCode, errMsg, remaining))
}

// =============================================================================
// ListNavigators
// =============================================================================

func (s *service) ListNavigators(
	ctx context.Context,
	req *connect.Request[aiv1.ListNavigatorsRequest],
) (*connect.Response[aiv1.ListNavigatorsResponse], error) {
	source := vision.ClientSourceFromHeader(s.resolveClientSource(req, req.Msg.GetClientSource()))

	navigators := s.deps.Registry.ListNavigators(ctx, source)
	out := make([]*aiv1.NavigatorInfo, 0, len(navigators))
	for _, n := range navigators {
		out = append(out, navigatorInfoToProto(n))
	}
	return connect.NewResponse(&aiv1.ListNavigatorsResponse{
		Navigators: out,
		Default:    string(s.deps.Registry.GetDefault()),
	}), nil
}

// =============================================================================
// StartNavigation
// =============================================================================

func (s *service) StartNavigation(
	ctx context.Context,
	req *connect.Request[aiv1.StartNavigationRequest],
) (*connect.Response[aiv1.StartNavigationResponse], error) {
	msg := req.Msg
	input, err := normalizeStartNavigationRequest(msg)
	if err != nil {
		return nil, err
	}

	source := vision.ClientSourceFromHeader(s.resolveClientSource(req, msg.GetClientSource()))

	preferredType := vision.NavigatorType(strings.TrimSpace(msg.GetNavigatorType()))
	navigator, err := s.deps.Registry.SelectNavigator(ctx, source, preferredType)
	if err != nil {
		return nil, s.mapSelectError(err, preferredType, source)
	}

	if err := s.authorizeNavigationCredits(ctx, navigator, s.navigationCredentialProvenance()); err != nil {
		return nil, err
	}

	effectPolicy, conditions, extraction, err := buildNavigationTaskContract(msg, navigator.Type())
	if err != nil {
		return nil, err
	}
	navReq := vision.NavigationRequest{
		EffectPolicy: effectPolicy, Postconditions: conditions, Extraction: extraction,
		SessionID:     input.sessionID,
		Prompt:        input.prompt,
		Model:         input.model,
		MaxSteps:      input.maxSteps,
		NavigatorType: navigator.Type(),
		UserID:        navigationUserID(ctx),
		CallbackURL:   s.resolveCallbackURL(req),
	}

	handle, err := navigator.Navigate(ctx, navReq)
	if err != nil {
		s.logger().WithError(err).Error("vision_navigation: failed to start navigation")
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("failed to start navigation: %w", err))
	}

	s.logger().WithFields(logrus.Fields{
		"navigation_id": handle.ID(),
		"session_id":    input.sessionID,
		"model":         input.model,
		"max_steps":     input.maxSteps,
		"navigator":     string(navigator.Type()),
	}).Info("vision_navigation: started")

	return connect.NewResponse(&aiv1.StartNavigationResponse{
		NavigationId:  handle.ID(),
		Status:        "started",
		Model:         input.model,
		MaxSteps:      int32(input.maxSteps),
		NavigatorType: string(navigator.Type()),
	}), nil
}

// =============================================================================
// GetNavigationStatus
// =============================================================================

// maxStatusWait caps the server-side wait a single GetNavigationStatus call
// may block for. Callers needing longer simply call again; the wait is a
// server primitive so no client ever has to poll.
const maxStatusWait = 300 * time.Second

func waitForNavigationStatus(
	ctx context.Context,
	tracker vision.SessionTracker,
	navigationID string,
	session *vision.NavigationSession,
	wait time.Duration,
) (*vision.NavigationSession, error) {
	if wait <= 0 || session.Status.Terminal() {
		return session, nil
	}

	deadline := time.NewTimer(wait)
	defer deadline.Stop()
	for !session.Status.Terminal() {
		select {
		case <-ctx.Done():
			return nil, connect.NewError(connect.CodeCanceled, ctx.Err())
		case <-deadline.C:
			if fresh, still := tracker.GetSession(navigationID); still {
				session = fresh
			}
			return session, nil
		case <-session.Changed():
		}
		next, ok := tracker.GetSession(navigationID)
		if !ok {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("navigation session not found"))
		}
		session = next
	}
	return session, nil
}

func (s *service) GetNavigationStatus(
	ctx context.Context,
	req *connect.Request[aiv1.GetNavigationStatusRequest],
) (*connect.Response[aiv1.GetNavigationStatusResponse], error) {
	navigationID := strings.TrimSpace(req.Msg.GetNavigationId())
	if navigationID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("navigation_id is required"))
	}
	if s.deps.Tracker == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("navigation session not found"))
	}
	session, ok := s.deps.Tracker.GetSession(navigationID)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("navigation session not found"))
	}

	// Each GetSession snapshot carries the Changed() channel that was current
	// when it was taken, so a transition between the snapshot and the select
	// still wakes the shared wait helper: the channel is already closed.
	waitedSession, waitErr := waitForNavigationStatus(ctx, s.deps.Tracker, navigationID, session, clampStatusWait(req.Msg.GetWaitMillis()))
	if waitErr != nil {
		return nil, waitErr
	}
	session = waitedSession
	return connect.NewResponse(navigationStatusToProto(session)), nil
}

// clampStatusWait bounds a requested wait to [0, maxStatusWait].
func clampStatusWait(millis int64) time.Duration {
	if millis <= 0 {
		return 0
	}
	wait := time.Duration(millis) * time.Millisecond
	if wait > maxStatusWait {
		return maxStatusWait
	}
	return wait
}

func navigationStatusToProto(session *vision.NavigationSession) *aiv1.GetNavigationStatusResponse {
	steps := make([]*aiv1.NavigationStep, 0, len(session.Steps))
	for _, st := range session.Steps {
		steps = append(steps, &aiv1.NavigationStep{
			Index:       int32(st.Index),
			ActionType:  st.ActionType,
			Selector:    st.Selector,
			Value:       st.Value,
			Url:         st.URL,
			Description: st.Description,
			Success:     st.Success,
			Error:       st.Error,
			At:          timestamppb.New(st.At),
		})
	}
	data, _ := structpb.NewStruct(session.ExtractedData)
	return &aiv1.GetNavigationStatusResponse{
		VerifiedSuccess: session.VerifiedSuccess, ExtractedData: data, VerificationError: session.VerificationError,
		FinalUrl: session.FinalURL, Error: session.Error, Summary: session.Summary,
		TotalDurationMs: session.TotalDurationMs,
		NavigationId:    session.NavigationID,
		SessionId:       session.SessionID,
		Status:          string(session.Status),
		StepCount:       int32(session.StepCount),
		TotalTokens:     int32(session.TotalTokens),
		StartedAt:       timestamppb.New(session.StartedAt),
		NavigatorType:   string(session.NavigatorType),
		Terminal:        session.Status.Terminal(),
		Steps:           steps,
	}
}

// =============================================================================
// AbortNavigation
// =============================================================================

func (s *service) AbortNavigation(
	ctx context.Context,
	req *connect.Request[aiv1.AbortNavigationRequest],
) (*connect.Response[aiv1.AbortNavigationResponse], error) {
	navigationID := strings.TrimSpace(req.Msg.GetNavigationId())
	if navigationID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("navigation_id is required"))
	}
	if s.deps.Tracker == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("navigation session not found"))
	}
	if err := s.deps.Tracker.AbortNavigation(ctx, navigationID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("navigation session not found"))
		}
		s.logger().WithError(err).Error("vision_navigation: abort failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&aiv1.AbortNavigationResponse{
		NavigationId: navigationID,
		Status:       "aborting",
		Message:      "Abort signal sent. Navigation will stop after current step.",
	}), nil
}

// =============================================================================
// ResumeNavigation
// =============================================================================

func (s *service) ResumeNavigation(
	ctx context.Context,
	req *connect.Request[aiv1.ResumeNavigationRequest],
) (*connect.Response[aiv1.ResumeNavigationResponse], error) {
	navigationID := strings.TrimSpace(req.Msg.GetNavigationId())
	if navigationID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("navigation_id is required"))
	}
	if s.deps.Tracker == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("navigation session not found"))
	}
	if err := s.deps.Tracker.ResumeNavigation(ctx, navigationID); err != nil {
		switch {
		case strings.Contains(err.Error(), "not found"):
			return nil, connect.NewError(connect.CodeNotFound, errors.New("navigation session not found"))
		case strings.Contains(err.Error(), "not awaiting human"):
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("navigation is not awaiting human intervention"))
		}
		s.logger().WithError(err).Error("vision_navigation: resume failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&aiv1.ResumeNavigationResponse{
		NavigationId: navigationID,
		Status:       "resumed",
		Message:      "Navigation resumed. Will continue from where it paused.",
	}), nil
}

// =============================================================================
// helpers
// =============================================================================

func (s *service) logger() *logrus.Logger { return s.deps.Logger }

// resolveClientSource prefers an explicit request field; falls back to the
// X-Client-Source request header to preserve the legacy contract.
func (s *service) resolveClientSource(req connect.AnyRequest, explicit string) string {
	if v := strings.TrimSpace(explicit); v != "" {
		return v
	}
	return strings.TrimSpace(req.Header().Get("X-Client-Source"))
}

// resolveCallbackURL builds the URL playwright-driver POSTs step events to.
// Uses Deps.CallbackBase when set; otherwise derives scheme+host from the
// Connect request headers (Host / X-Forwarded-Host / X-Forwarded-Proto).
func (s *service) resolveCallbackURL(req connect.AnyRequest) string {
	if base := strings.TrimRight(s.deps.CallbackBase, "/"); base != "" {
		return base + "/api/v1/internal/ai-navigate/callback"
	}
	h := req.Header()
	scheme := "http"
	if v := strings.TrimSpace(h.Get("X-Forwarded-Proto")); v != "" {
		scheme = v
	}
	host := strings.TrimSpace(h.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(h.Get("Host"))
	}
	if host == "" {
		host = "127.0.0.1:8110"
	}
	return fmt.Sprintf("%s://%s/api/v1/internal/ai-navigate/callback", scheme, host)
}

// mapSelectError maps registry.SelectNavigator errors onto Connect codes.
func (s *service) mapSelectError(err error, preferred vision.NavigatorType, source vision.ClientSource) error {
	switch {
	case errors.Is(err, vision.ErrNavigatorNotFound):
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("navigator type %q is not registered", preferred))
	case errors.Is(err, vision.ErrNavigatorNotAvailable):
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("navigator %q is not currently available", preferred))
	case errors.Is(err, vision.ErrNavigatorNotAllowed):
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("navigator %q is not allowed for client source %q", preferred, source))
	case errors.Is(err, vision.ErrNoNavigatorsAvailable):
		return connect.NewError(connect.CodeUnavailable, errors.New("no navigators are currently available"))
	default:
		s.logger().WithError(err).Error("vision_navigation: navigator selection failed")
		return connect.NewError(connect.CodeInternal, err)
	}
}

func navigatorInfoToProto(n vision.NavigatorInfo) *aiv1.NavigatorInfo {
	allowed := make([]string, 0, len(n.AllowedSources))
	for _, src := range n.AllowedSources {
		allowed = append(allowed, string(src))
	}
	bypass := make([]string, 0, len(n.CreditPolicy.BypassConditions))
	for _, b := range n.CreditPolicy.BypassConditions {
		bypass = append(bypass, string(b))
	}
	return &aiv1.NavigatorInfo{
		Type:        string(n.Type),
		Available:   n.Available,
		Description: n.Description,
		CreditPolicy: &aiv1.CreditPolicyInfo{
			RequiresCredits:  n.CreditPolicy.RequiresCredits,
			CreditsPerStep:   int32(n.CreditPolicy.CreditsPerStep),
			BypassConditions: bypass,
		},
		AllowedSources:    allowed,
		UnavailableReason: n.UnavailableReason,
	}
}
