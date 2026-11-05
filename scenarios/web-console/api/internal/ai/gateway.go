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
)

// GatewayProvider keeps source consent in Web Console while Gateway owns the
// provider chain. Call-time discovery selects a live full server before the
// embedded gateway through the common peer/runtime resolution path.
type GatewayProvider struct {
	Source       string
	ProviderName string
	ResolveURL   func(context.Context) (string, error)
	Client       *http.Client
}

func (g *GatewayProvider) Name() string { return g.ProviderName }
func (g *GatewayProvider) Generate(ctx context.Context, system, user string) (string, error) {
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
	request := connect.NewRequest(&inferencev1.RunRequest{Source: system + "\n\nUser request:\n" + user, Instruction: "Return only the shell command as a JSON string.", SchemaJson: `{"type":"string"}`, Role: "extract.structured"})
	request.Header().Set("X-Vrooli-AI-Source", g.Source)
	response, err := inferenceconnect.NewInferenceServiceClient(client, base).Run(ctx, request)
	if err != nil {
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

func NewGatewayChain() *Chain {
	return NewChain(&GatewayProvider{Source: "local", ProviderName: "ollama"}, &GatewayProvider{Source: "own-key", ProviderName: "openrouter"}, &GatewayProvider{Source: "vrooli-credits", ProviderName: "vrooli"})
}
