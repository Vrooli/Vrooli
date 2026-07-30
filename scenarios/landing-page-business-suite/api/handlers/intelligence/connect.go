package intelligence

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	intelligencecore "landing-page-business-suite-api/internal/intelligence"
)

// ConnectHandler exposes the unary intelligence contract through generated
// Connect procedures. It shares the same explicit dependencies as the HTTP
// edge, so authorization, rate limits, and usage accounting stay consistent.
type ConnectHandler struct{ deps Dependencies }

func NewConnectHandler(deps Dependencies) *ConnectHandler { return &ConnectHandler{deps: deps} }

func (h *ConnectHandler) Chat(ctx context.Context, request *connect.Request[lpbsv1.ChatRequest]) (*connect.Response[lpbsv1.ChatResponse], error) {
	identity, err := h.authenticatedIdentity(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.allow(identity, clientIPFromHeaders(request.Header())); err != nil {
		return nil, err
	}

	input := request.Msg
	chat := intelligencecore.AIRequest{
		Model:     input.GetModel(),
		MaxTokens: int(input.GetMaxTokens()),
		Metadata: intelligencecore.AIMetadata{
			AppBundleKey: input.GetMetadata().GetAppBundleKey(),
			Operation:    input.GetMetadata().GetOperation(),
		},
	}
	for _, message := range input.GetMessages() {
		chat.Messages = append(chat.Messages, intelligencecore.AIMessage{Role: message.GetRole(), Content: message.GetContent()})
	}
	if err := ValidateRequest(&chat); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	response, err := h.deps.Service.ExecuteChat(ctx, identity, chat)
	if err != nil {
		return nil, connectAIError(err)
	}
	return connect.NewResponse(chatResponse(response)), nil
}

func (h *ConnectHandler) ListModels(_ context.Context, _ *connect.Request[lpbsv1.ListModelsRequest]) (*connect.Response[lpbsv1.ListModelsResponse], error) {
	return connect.NewResponse(&lpbsv1.ListModelsResponse{Models: h.deps.Service.GetAvailableModels()}), nil
}

func (h *ConnectHandler) GetUsage(ctx context.Context, _ *connect.Request[lpbsv1.GetUsageRequest]) (*connect.Response[lpbsv1.GetUsageResponse], error) {
	identity, err := h.authenticatedIdentity(ctx)
	if err != nil {
		return nil, err
	}
	if h.deps.Usage == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("usage service is not configured"))
	}
	tier := resolveTier(ctx, identity, h.deps.SubscriptionTier)
	summary, err := h.deps.Usage(ctx, identity, tier)
	if err != nil {
		h.deps.LogError("ai_usage_fetch_failed", map[string]interface{}{"error": err.Error(), "user_identity": identity})
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get AI usage: %w", err))
	}
	used := summary.Usage["ai_credits"]
	limit := summary.Limits["ai_credits"]
	remaining := summary.Remaining["ai_credits"]
	displayLimit := float64(limit) / 100000.0
	displayRemaining := float64(remaining) / 100000.0
	if limit < 0 {
		displayLimit = -1
		displayRemaining = -1
	}
	return connect.NewResponse(&lpbsv1.GetUsageResponse{
		UserIdentity:       identity,
		Tier:               tier,
		BillingPeriod:      summary.BillingPeriod,
		ResetDate:          summary.ResetDate.Format(time.RFC3339),
		AiCreditsUsed:      used,
		AiCreditsLimit:     limit,
		AiCreditsRemaining: remaining,
		DisplayUsed:        float64(used) / 100000.0,
		DisplayLimit:       displayLimit,
		DisplayRemaining:   displayRemaining,
	}), nil
}

func (h *ConnectHandler) Health(ctx context.Context, _ *connect.Request[lpbsv1.HealthRequest]) (*connect.Response[lpbsv1.HealthResponse], error) {
	if err := h.deps.Service.HealthCheck(ctx); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI gateway health: %w", err))
	}
	return connect.NewResponse(&lpbsv1.HealthResponse{Status: "healthy"}), nil
}

func (h *ConnectHandler) authenticatedIdentity(ctx context.Context) (string, error) {
	identity := strings.TrimSpace(h.deps.UserIdentity(ctx))
	if identity == "" {
		return "", connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	return identity, nil
}

func (h *ConnectHandler) allow(identity, clientIP string) error {
	if !h.deps.UserRateLimiter.Allow(identity) {
		return connect.NewError(connect.CodeResourceExhausted, errors.New("rate limit exceeded; please try again later"))
	}
	if !h.deps.IPRateLimiter.Allow(clientIP) {
		h.deps.Log("ai_rate_limit_ip_exceeded", map[string]interface{}{"level": "warn", "client_ip": clientIP, "user_identity": identity, "security": true})
		return connect.NewError(connect.CodeResourceExhausted, errors.New("rate limit exceeded"))
	}
	return nil
}

func clientIPFromHeaders(headers http.Header) string {
	if forwarded := strings.TrimSpace(headers.Get("X-Forwarded-For")); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	if clientIP := strings.TrimSpace(headers.Get("X-Real-IP")); clientIP != "" {
		return clientIP
	}
	return "unknown"
}

func chatResponse(response *intelligencecore.AIResponse) *lpbsv1.ChatResponse {
	if response == nil {
		return &lpbsv1.ChatResponse{}
	}
	return &lpbsv1.ChatResponse{Id: response.ID, Model: response.Model, Content: response.Content, PromptTokens: protobufTokenCount(response.PromptTokens), CompletionTokens: protobufTokenCount(response.CompletionTokens), TotalTokens: protobufTokenCount(response.TotalTokens), CreditsCharged: response.CreditsCharged, FinishReason: response.FinishReason}
}

func protobufTokenCount(value int) int32 {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	if value < math.MinInt32 {
		return math.MinInt32
	}
	return int32(value)
}

func connectAIError(err error) error {
	switch {
	case errors.Is(err, intelligencecore.ErrInsufficientCredits):
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("insufficient credits for this operation"))
	case errors.Is(err, intelligencecore.ErrNoAPIKeyConfigured):
		return connect.NewError(connect.CodeUnavailable, errors.New("AI service not configured"))
	case errors.Is(err, intelligencecore.ErrModelNotAllowed):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, intelligencecore.ErrProvider):
		return connect.NewError(connect.CodeUnavailable, errors.New("AI provider error"))
	default:
		return connect.NewError(connect.CodeInternal, fmt.Errorf("AI request failed: %w", err))
	}
}

// RegisterConnectRoutes mounts each procedure independently so public model
// discovery and health checks remain public while chat and usage retain the
// user-auth policy of their former REST endpoints.
func RegisterConnectRoutes(router *mux.Router, deps Dependencies, requireUser func(http.HandlerFunc) http.HandlerFunc) {
	_, generated := lpbsconnect.NewIntelligenceServiceHandler(NewConnectHandler(deps))
	router.Handle(lpbsconnect.IntelligenceServiceListModelsProcedure, generated).Methods(http.MethodPost)
	router.Handle(lpbsconnect.IntelligenceServiceHealthProcedure, generated).Methods(http.MethodPost)
	router.Handle(lpbsconnect.IntelligenceServiceChatProcedure, requireUser(generated.ServeHTTP)).Methods(http.MethodPost)
	router.Handle(lpbsconnect.IntelligenceServiceGetUsageProcedure, requireUser(generated.ServeHTTP)).Methods(http.MethodPost)
}

var _ lpbsconnect.IntelligenceServiceHandler = (*ConnectHandler)(nil)
