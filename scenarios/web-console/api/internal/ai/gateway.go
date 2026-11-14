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
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// GatewayProvider keeps source consent in Web Console while Gateway owns the
// provider chain. Call-time discovery selects a live full server before the
// embedded gateway through the common peer/runtime resolution path.
type GatewayProvider struct {
	Source       string
	ProviderName string
	// PrivacyClass must be supplied from a caller or policy source. The normal
	// Web Console chain has no classification input, so remote sources remain
	// ineligible until that owner adds one.
	PrivacyClass sharedv1.PrivacyClass
	ResolveURL   func(context.Context) (string, error)
	Client       *http.Client
}

func (g *GatewayProvider) Name() string { return g.ProviderName }
func (g *GatewayProvider) Generate(ctx context.Context, system, user string) (string, error) {
	profile, err := gatewaySourceProfile(g.Source)
	if err != nil {
		return "", err
	}
	if profile == sharedv1.Profile_PROFILE_REMOTE_ONLY && !remotePrivacyClass(g.PrivacyClass) {
		return "", errors.New("remote Web Console generation requires a sourced non-secret privacy classification")
	}
	if strings.EqualFold(strings.TrimSpace(g.Source), "vrooli-credits") && consumerToken(ctx) == "" {
		return "", fmt.Errorf("%w: Vrooli credits require a signed-in subscription session", ErrCreditsRequired)
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
	})
	request.Header().Set("X-Vrooli-AI-Source", g.Source)
	if strings.EqualFold(strings.TrimSpace(g.Source), "vrooli-credits") {
		request.Header().Set("Authorization", "Bearer "+consumerToken(ctx))
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

func NewGatewayChain() *Chain {
	// The caller currently has no privacy-classification source. Keep remote
	// options visible for source-consent compatibility, while Generate refuses
	// them until a class is explicitly provided.
	return NewChain(&GatewayProvider{Source: "local", ProviderName: "ollama"}, &GatewayProvider{Source: "own-key", ProviderName: "openrouter"}, &GatewayProvider{Source: "vrooli-credits", ProviderName: "vrooli"})
}
