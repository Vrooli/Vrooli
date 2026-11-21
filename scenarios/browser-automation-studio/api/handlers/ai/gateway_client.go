package ai

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/browser-automation-studio/middleware"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const (
	basAIPrivacySource = "browser_automation_studio.workflow_prompt_and_owner_context"
	basAIStringSchema  = `{"type":"string"}`
)

// gatewayRolePromptClient routes BAS AI requests through AI Gateway. The
// request-scoped source preference selects the route; signed session,
// entitlement and Gateway consent checks independently authorize remote work.
type gatewayRolePromptClient struct {
	resolveURL func(context.Context) (string, error)
	httpClient *http.Client
}

type privacyConsentGrantContextKey struct{}
type privacyConsentSourceContextKey struct{}

// PrivacyConsentContextMiddleware carries the opaque receipt and selected
// source preference to Gateway-backed AI work. The source is routing input,
// never authorization; Gateway remains authoritative for consent verification.
func PrivacyConsentContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receipt := strings.TrimSpace(r.Header.Get("X-Vrooli-AI-Consent-Grant"))
		if len(receipt) > 16384 || strings.ContainsAny(receipt, "\r\n") {
			http.Error(w, "AI consent grant header is invalid", http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), privacyConsentGrantContextKey{}, receipt)
		source := strings.TrimSpace(r.Header.Get("X-Vrooli-AI-Consent-Source"))
		if source != "" && source != "own-key" && source != "vrooli-credits" && source != "local" {
			http.Error(w, "AI consent source is invalid", http.StatusBadRequest)
			return
		}
		ctx = context.WithValue(ctx, privacyConsentSourceContextKey{}, source)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func privacyConsentGrant(ctx context.Context) string {
	receipt, _ := ctx.Value(privacyConsentGrantContextKey{}).(string)
	return strings.TrimSpace(receipt)
}

func privacyConsentSource(ctx context.Context) string {
	source, _ := ctx.Value(privacyConsentSourceContextKey{}).(string)
	return strings.TrimSpace(source)
}

// PrivacyConsentSourceFromContext returns the allowlisted request-scoped AI
// source preference installed by PrivacyConsentContextMiddleware. Callers may
// use it for route selection only; it never grants remote consent.
func PrivacyConsentSourceFromContext(ctx context.Context) string {
	return privacyConsentSource(ctx)
}

func newGatewayRolePromptClient() *gatewayRolePromptClient {
	return &gatewayRolePromptClient{
		resolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		},
		httpClient: &http.Client{Timeout: 90 * time.Second},
	}
}

// NewGatewayRolePromptClient returns the shared Gateway client for BAS workflow
// generation and other role-based text operations.
func NewGatewayRolePromptClient() *gatewayRolePromptClient {
	return newGatewayRolePromptClient()
}

// NewGatewayRolePromptClientWithResolver constructs the production Gateway
// client with an explicit endpoint resolver. The default constructor remains
// discovery-backed; this seam also supports controlled local deployments and
// end-to-end tests without changing process-wide discovery state.
func NewGatewayRolePromptClientWithResolver(resolveURL func(context.Context) (string, error), client *http.Client) *gatewayRolePromptClient {
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	return &gatewayRolePromptClient{resolveURL: resolveURL, httpClient: client}
}

// ExecutePrompt implements the workflow AIClient contract with the standard
// prose generation role. Structured output requirements are included in the
// caller's prompt and validated by the workflow service.
func (c *gatewayRolePromptClient) ExecutePrompt(ctx context.Context, prompt string) (string, error) {
	return c.ExecutePromptWithRole(ctx, "write.default", prompt)
}

// Model identifies this role-routed client without claiming a concrete model;
// AI Gateway resolves the model per request.
func (c *gatewayRolePromptClient) Model() string {
	return "ai-gateway/write.default"
}

func (c *gatewayRolePromptClient) ExecutePromptWithRole(ctx context.Context, role, prompt string) (string, error) {
	if c == nil || c.resolveURL == nil {
		return "", errors.New("AI Gateway is not configured")
	}
	if strings.TrimSpace(prompt) == "" {
		return "", errors.New("prompt is required")
	}
	baseURL, err := c.resolveURL(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve AI Gateway: %w", err)
	}
	source, profile, token, err := basGatewayRoute(ctx)
	if err != nil {
		return "", err
	}
	rootOperationID := gatewayRootOperationIDFromContext(ctx)
	if rootOperationID == "" {
		rootOperationID, err = newGatewayRootOperationID()
		if err != nil {
			return "", fmt.Errorf("create AI Gateway root operation ID: %w", err)
		}
	}
	request := connect.NewRequest(&inferencev1.RunRequest{
		Source:            prompt,
		SchemaJson:        basAIStringSchema,
		Instruction:       "Return the requested result as a JSON string. Preserve valid JSON text when the task requests JSON.",
		Role:              strings.TrimSpace(role),
		Profile:           profile,
		PrivacyClass:      sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacyProvenance: basAIPrivacySource,
		MaxOutputTokens:   4096,
		RootOperationId:   rootOperationID,
	})
	request.Header().Set("X-Vrooli-AI-Source", source)
	if source != "local" {
		if receipt := privacyConsentGrant(ctx); receipt != "" {
			request.Header().Set("X-Vrooli-AI-Consent-Grant", receipt)
		}
	}
	if token != "" {
		request.Header().Set("Authorization", "Bearer "+token)
	}
	client := c.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := inferenceconnect.NewInferenceServiceClient(client, strings.TrimRight(baseURL, "/")).Run(ctx, request)
	if err != nil {
		code := connect.CodeOf(err)
		if code == connect.CodeUnknown {
			code = connect.CodeUnavailable
		}
		gatewayConnectErr := connect.NewError(code, fmt.Errorf("AI Gateway transport failed; if the request may have reached a hosted provider, reuse root_operation_id=%s before retrying: %w", rootOperationID, err))
		gatewayConnectErr.Meta().Set("X-Vrooli-Root-Operation-ID", rootOperationID)
		return "", gatewayConnectErr
	}
	if gatewayErr := response.Msg.GetError(); gatewayErr != nil {
		message := gatewayErr.GetMessage()
		if gatewayErr.GetCode() == inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_OUTCOME_UNCERTAIN {
			message = fmt.Sprintf("%s; root_operation_id=%s", message, rootOperationID)
		}
		gatewayConnectErr := connect.NewError(gatewayConnectCode(gatewayErr.GetCode()), errors.New(message))
		gatewayConnectErr.Meta().Set("X-Vrooli-Root-Operation-ID", rootOperationID)
		return "", gatewayConnectErr
	}
	var result string
	if err := json.Unmarshal([]byte(response.Msg.GetValueJson()), &result); err != nil {
		return "", fmt.Errorf("decode AI Gateway string response: %w", err)
	}
	if strings.TrimSpace(result) == "" {
		return "", errors.New("AI Gateway returned an empty response")
	}
	return result, nil
}

type gatewayRootOperationIDKey struct{}

// WithGatewayRootOperationID attaches a caller-generated idempotency ID for a
// hosted Gateway operation. It is a recovery key, never an authorization token.
func WithGatewayRootOperationID(ctx context.Context, operationID string) context.Context {
	return context.WithValue(ctx, gatewayRootOperationIDKey{}, strings.TrimSpace(operationID))
}

func gatewayRootOperationIDFromContext(ctx context.Context) string {
	operationID, _ := ctx.Value(gatewayRootOperationIDKey{}).(string)
	return strings.TrimSpace(operationID)
}

// GatewayRootOperationIDFromContext returns the caller's optional recovery
// key. Empty means the Gateway client should create one for this attempt.
func GatewayRootOperationIDFromContext(ctx context.Context) string {
	return gatewayRootOperationIDFromContext(ctx)
}

func newGatewayRootOperationID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func gatewayConnectCode(code inferencev1.InferenceErrorCode) connect.Code {
	switch code {
	case inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST,
		inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNSUPPORTED_SCHEMA,
		inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_VALIDATION_FAILED:
		return connect.CodeInvalidArgument
	case inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNSUPPORTED_SAMPLING:
		return connect.CodeFailedPrecondition
	case inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_CONTEXT_OVERFLOW:
		return connect.CodeResourceExhausted
	case inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE,
		inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_PROVIDER_FAILED,
		inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_OUTCOME_UNCERTAIN:
		return connect.CodeUnavailable
	default:
		return connect.CodeInternal
	}
}

func basGatewayRoute(ctx context.Context) (source string, profile sharedv1.Profile, token string, err error) {
	token = monetization.AccessTokenFromContext(ctx)
	// A consent-source header is a request-scoped source preference set by the
	// UI. It does not authorize remote use: Gateway still requires the matching
	// signed principal and opaque grant for every remote dispatch.
	switch privacyConsentSource(ctx) {
	case "own-key":
		if token == "" {
			return "", sharedv1.Profile_PROFILE_UNSPECIFIED, "", errors.New("AI Gateway own-key inference requires a shared signed-in session with a saved credential")
		}
		return "own-key", sharedv1.Profile_PROFILE_REMOTE_ONLY, token, nil
	case "vrooli-credits":
		if token == "" {
			return "", sharedv1.Profile_PROFILE_UNSPECIFIED, "", errors.New("BAS AI credits inference requires a signed shared session")
		}
		ent := monetization.SnapshotFromContext(ctx)
		if ent == nil || !ent.HasFeatureForApp("browser-automation-studio", entitlement.FeatureAI) {
			return "", sharedv1.Profile_PROFILE_UNSPECIFIED, "", errors.New("BAS AI requires the signed session's ai feature entitlement when using Vrooli credits")
		}
		return "vrooli-credits", sharedv1.Profile_PROFILE_REMOTE_ONLY, token, nil
	case "local":
		return "local", sharedv1.Profile_PROFILE_LOCAL_ONLY, "", nil
	}
	if middleware.HasOwnKeySourceHint(ctx) {
		if token == "" {
			return "", sharedv1.Profile_PROFILE_UNSPECIFIED, "", errors.New("AI Gateway own-key inference requires a shared signed-in session with a saved credential; raw BAS BYOK keys are not forwarded")
		}
		return "own-key", sharedv1.Profile_PROFILE_REMOTE_ONLY, token, nil
	}
	if token != "" {
		// A signed-in request using Vrooli's shared balance is a paid BAS
		// feature. The entitlement was fetched and verified by the server-side
		// shared-session middleware; a caller supplied header or UI flag cannot
		// create this context value. Gateway remains authoritative for balance
		// reservation and charging.
		ent := monetization.SnapshotFromContext(ctx)
		if ent == nil || !ent.HasFeatureForApp("browser-automation-studio", entitlement.FeatureAI) {
			return "", sharedv1.Profile_PROFILE_UNSPECIFIED, "", errors.New("BAS AI requires the signed session's ai feature entitlement when using Vrooli credits")
		}
		return "vrooli-credits", sharedv1.Profile_PROFILE_REMOTE_ONLY, token, nil
	}
	return "local", sharedv1.Profile_PROFILE_LOCAL_ONLY, "", nil
}

var _ interface {
	ExecutePromptWithRole(context.Context, string, string) (string, error)
} = (*gatewayRolePromptClient)(nil)

var _ interface {
	ExecutePrompt(context.Context, string) (string, error)
	Model() string
} = (*gatewayRolePromptClient)(nil)
