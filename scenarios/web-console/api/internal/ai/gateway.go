package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/lpbsidentity"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const (
	webConsolePrivacyClass  = sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL
	webConsolePrivacySource = "web_console.user_command_and_session_context"
)

// GatewayProvider keeps source consent in Web Console while Gateway owns the
// provider chain. Call-time discovery selects a live full server before the
// embedded gateway through the common peer/runtime resolution path.
type GatewayProvider struct {
	Source       string
	ProviderName string
	// PrivacyClass must be supplied from a caller or policy source. The normal
	// Web Console task contract classifies user command input plus the session
	// system/workspace context as confidential. Provider config supplies the
	// user's source consent; PrivacySource names the data owner without content.
	PrivacyClass                     sharedv1.PrivacyClass
	PrivacySource                    string
	ResolveCreditCallerAuthorization func(context.Context) (string, error)
	ResolveURL                       func(context.Context) (string, error)
	Client                           *http.Client
}

func (g *GatewayProvider) Name() string { return g.ProviderName }
func (g *GatewayProvider) Generate(ctx context.Context, system, user string) (string, error) {
	profile, err := gatewaySourceProfile(g.Source)
	if err != nil {
		return "", err
	}
	if !recognizedPrivacyClass(g.PrivacyClass) || strings.TrimSpace(g.PrivacySource) == "" {
		return "", errors.New("Web Console generation requires sourced privacy classification")
	}
	if profile == sharedv1.Profile_PROFILE_REMOTE_ONLY && !remotePrivacyClass(g.PrivacyClass) {
		return "", errors.New("remote Web Console generation requires a sourced non-secret privacy classification")
	}
	creditCallerToken := ""
	if strings.EqualFold(strings.TrimSpace(g.Source), "vrooli-credits") {
		if consumerToken(ctx) == "" {
			return "", fmt.Errorf("%w: Vrooli credits require a signed-in subscription session", ErrCreditsRequired)
		}
		if g.ResolveCreditCallerAuthorization == nil {
			return "", fmt.Errorf("%w: Vrooli credits require a configured Web Console service identity", ErrCreditsRequired)
		}
		creditCallerToken, err = g.ResolveCreditCallerAuthorization(ctx)
		if err != nil || strings.TrimSpace(creditCallerToken) == "" {
			return "", fmt.Errorf("%w: Web Console service identity is unavailable", ErrCreditsRequired)
		}
	}
	resolve := g.ResolveURL
	if resolve == nil {
		resolve = func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		}
	}
	base, err := resolve(ctx)
	if err != nil {
		return "", fmt.Errorf("AI gateway unavailable: %w", err)
	}
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Minute}
	}
	request := connect.NewRequest(&inferencev1.RunRequest{
		Source: system + "\n\nUser request:\n" + user, Instruction: "Return only the shell command as a JSON string.",
		SchemaJson: `{"type":"string"}`, Role: "author.generator", Profile: profile, PrivacyClass: g.PrivacyClass,
		PrivacyProvenance: g.PrivacySource, RootOperationId: rootOperationID(ctx),
	})
	request.Header().Set("X-Vrooli-AI-Source", g.Source)
	if receipt := privacyConsentGrant(ctx, g.Source); receipt != "" {
		request.Header().Set("X-Vrooli-AI-Consent-Grant", receipt)
	}
	if token := consumerToken(ctx); token != "" {
		request.Header().Set("Authorization", "Bearer "+token)
	}
	if creditCallerToken != "" {
		request.Header().Set(lpbsidentity.CreditCallerAuthorizationHeader, "Bearer "+strings.TrimSpace(creditCallerToken))
	}
	response, err := inferenceconnect.NewInferenceServiceClient(client, base).Run(ctx, request)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeResourceExhausted {
			return "", fmt.Errorf("%w: %v", ErrCreditsRequired, err)
		}
		return "", err
	}
	if response.Msg.GetError() != nil {
		return "", errors.New(response.Msg.GetError().GetMessage())
	}
	var result string
	if err = json.Unmarshal([]byte(response.Msg.GetValueJson()), &result); err != nil {
		return "", errors.New("AI gateway returned invalid command data")
	}
	if strings.TrimSpace(result) == "" {
		return "", errors.New("AI gateway returned no command")
	}
	return result, nil
}

func recognizedPrivacyClass(class sharedv1.PrivacyClass) bool {
	switch class {
	case sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC,
		sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET:
		return true
	default:
		return false
	}
}

func remotePrivacyClass(class sharedv1.PrivacyClass) bool {
	switch class {
	case sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC,
		sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL:
		return true
	default:
		return false
	}
}

func gatewaySourceProfile(source string) (sharedv1.Profile, error) {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "local":
		return sharedv1.Profile_PROFILE_LOCAL_ONLY, nil
	case "own-key", "vrooli-credits":
		return sharedv1.Profile_PROFILE_REMOTE_ONLY, nil
	default:
		return sharedv1.Profile_PROFILE_UNSPECIFIED, fmt.Errorf("unsupported AI source %q", source)
	}
}

func NewGatewayChain(resolveCreditCallerAuthorization ...func(context.Context) (string, error)) *Chain {
	var callerAuthorization func(context.Context) (string, error)
	if len(resolveCreditCallerAuthorization) > 0 {
		callerAuthorization = resolveCreditCallerAuthorization[0]
	}
	// Provider config owns the source consent and fallback order. Every attempt
	// carries the same confidential classification and non-sensitive source.
	return NewChain(
		&GatewayProvider{Source: "local", ProviderName: "ollama", PrivacyClass: webConsolePrivacyClass, PrivacySource: webConsolePrivacySource},
		&GatewayProvider{Source: "own-key", ProviderName: "openrouter", PrivacyClass: webConsolePrivacyClass, PrivacySource: webConsolePrivacySource},
		&GatewayProvider{Source: "vrooli-credits", ProviderName: "vrooli", PrivacyClass: webConsolePrivacyClass, PrivacySource: webConsolePrivacySource, ResolveCreditCallerAuthorization: callerAuthorization},
	)
}
