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

	sessionID := strings.TrimSpace(msg.GetSessionId())
	if sessionID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id is required"))
	}
	prompt := strings.TrimSpace(msg.GetPrompt())
	if prompt == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("prompt is required"))
	}
	model := strings.TrimSpace(msg.GetModel())
	if model == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("model is required"))
	}

	maxSteps := int(msg.GetMaxSteps())
	if maxSteps <= 0 {
		maxSteps = 20
	}
	if maxSteps > 100 {
		maxSteps = 100
	}

	source := vision.ClientSourceFromHeader(s.resolveClientSource(req, msg.GetClientSource()))

	preferredType := vision.NavigatorType(strings.TrimSpace(msg.GetNavigatorType()))
	navigator, err := s.deps.Registry.SelectNavigator(ctx, source, preferredType)
	if err != nil {
		return nil, s.mapSelectError(err, preferredType, source)
	}

	provenance := vision.CredentialProvenanceNone
	if s.deps.CredentialAuthority != nil {
		identity, parseErr := credentialauthority.ParseIdentity("vrooli/openrouter")
		if parseErr == nil && s.deps.CredentialAuthority.Status(identity, "api-key").Configured {
			provenance = vision.CredentialProvenanceAuthority
		}
	}
	policy := navigator.CreditPolicy()
	if s.deps.Credits != nil && policy.ShouldChargeCredits(provenance, false, false) {
		userID := entitlement.UserIdentityFromContext(ctx)
		if userID == "" {
			userID = "anonymous"
		}
		canProceed, errCode, errMsg, remaining, cerr := s.deps.Credits.CanPerformAIOperation(ctx, userID, policy.OperationType, provenance == vision.CredentialProvenanceAuthority)
		if cerr != nil {
			s.logger().WithError(cerr).Warn("vision_navigation: credit check failed; continuing")
		} else if !canProceed {
			code := connect.CodeFailedPrecondition
			if errCode == "INSUFFICIENT_CREDITS" {
				code = connect.CodeResourceExhausted
			}
			cErr := connect.NewError(code, fmt.Errorf("%s: %s (remaining=%d)", errCode, errMsg, remaining))
			return nil, cErr
		}
	}

	userID := entitlement.UserIdentityFromContext(ctx)
	if userID == "" {
		userID = "anonymous"
	}

	if req.Msg.EffectPolicy != "" && req.Msg.EffectPolicy != "explicit" && req.Msg.EffectPolicy != "read_only" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unsupported effect_policy"))
	}
	if len(req.Msg.Postconditions) > 16 || len(req.Msg.Extraction) > 16 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("at most 16 postconditions and extractions"))
	}
	if navigator.Type() != vision.NavigatorPlaywright && (req.Msg.EffectPolicy == "read_only" || len(req.Msg.Postconditions) > 0 || len(req.Msg.Extraction) > 0) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("selected navigator does not support enforced task contracts"))
	}
	conditions := make([]vision.NavigationPostcondition, 0, len(req.Msg.Postconditions))
	for _, c := range req.Msg.Postconditions {
		if c.Selector == "" || len(c.Selector) > 1024 || len(c.Expected) > 4096 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid postcondition"))
		}
		switch c.Mode {
		case "exists", "text_equals", "text_contains", "count_equals", "ASSERTION_MODE_EXISTS":
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unsupported postcondition mode"))
		}
		conditions = append(conditions, vision.NavigationPostcondition{Selector: c.Selector, Mode: c.Mode, Expected: c.Expected})
	}
	extraction := make([]vision.NavigationExtraction, 0, len(req.Msg.Extraction))
	names := map[string]bool{}
	for _, e := range req.Msg.Extraction {
		if e.Name == "" || names[e.Name] || len(e.Name) > 128 || e.Selector == "" || len(e.Selector) > 1024 || len(e.Attribute) > 128 || e.Limit < 0 || e.Limit > 100 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid extraction"))
		}
		names[e.Name] = true
		extraction = append(extraction, vision.NavigationExtraction{Name: e.Name, Selector: e.Selector, Attribute: e.Attribute, Limit: int(e.Limit)})
	}
	navReq := vision.NavigationRequest{
		EffectPolicy: req.Msg.EffectPolicy, Postconditions: conditions, Extraction: extraction,
		SessionID:     sessionID,
		Prompt:        prompt,
		Model:         model,
		MaxSteps:      maxSteps,
		NavigatorType: navigator.Type(),
		UserID:        userID,
		CallbackURL:   s.resolveCallbackURL(req),
	}

	handle, err := navigator.Navigate(ctx, navReq)
	if err != nil {
		s.logger().WithError(err).Error("vision_navigation: failed to start navigation")
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("failed to start navigation: %w", err))
	}

	s.logger().WithFields(logrus.Fields{
		"navigation_id": handle.ID(),
		"session_id":    sessionID,
		"model":         model,
		"max_steps":     maxSteps,
		"navigator":     string(navigator.Type()),
	}).Info("vision_navigation: started")

	return connect.NewResponse(&aiv1.StartNavigationResponse{
		NavigationId:  handle.ID(),
		Status:        "started",
		Model:         model,
		MaxSteps:      int32(maxSteps),
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

	if wait := clampStatusWait(req.Msg.GetWaitMillis()); wait > 0 && !session.Status.Terminal() {
		deadline := time.NewTimer(wait)
		defer deadline.Stop()
		// Each GetSession snapshot carries the Changed() channel that was
		// current when it was taken, so a transition between the snapshot
		// and the select still wakes us: the channel is already closed.
		for !session.Status.Terminal() {
			select {
			case <-ctx.Done():
				return nil, connect.NewError(connect.CodeCanceled, ctx.Err())
			case <-deadline.C:
				if fresh, still := s.deps.Tracker.GetSession(navigationID); still {
					session = fresh
				}
				return connect.NewResponse(navigationStatusToProto(session)), nil
			case <-session.Changed():
			}
			session, ok = s.deps.Tracker.GetSession(navigationID)
			if !ok {
				// Session was reaped while we waited; it can only be reaped
				// after completion, so report it as gone rather than hang.
				return nil, connect.NewError(connect.CodeNotFound, errors.New("navigation session not found"))
			}
		}
	}
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
		NavigationId:  session.NavigationID,
		SessionId:     session.SessionID,
		Status:        string(session.Status),
		StepCount:     int32(session.StepCount),
		TotalTokens:   int32(session.TotalTokens),
		StartedAt:     timestamppb.New(session.StartedAt),
		NavigatorType: string(session.NavigatorType),
		Terminal:      session.Status.Terminal(),
		Steps:         steps,
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
