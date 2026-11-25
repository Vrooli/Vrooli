package hub

import (
	"context"
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

// embeddingRole is the gateway-owned local role. The gateway chooses the model;
// this client accepts the response's model and dimension rather than assuming
// one, so a model change is visible in the stored rows instead of corrupting
// the comparison.
const embeddingRole = "embedding.default"

// embeddingProvenance names this feature in the gateway's privacy audit.
const embeddingProvenance = "notification-hub.decision-relations:decision-text-local-embedding"

// EmbeddingClient is the narrow slice of the generated gateway client the
// embedder uses; tests can supply a fake.
type EmbeddingClient interface {
	Embed(context.Context, *connect.Request[inferencev1.EmbedRequest]) (*connect.Response[inferencev1.EmbedResponse], error)
}

// GatewayEmbedder asks ai-gateway for local-only embeddings. It is optional:
// the Hub works, and relatedness stays lexical, when the gateway is absent.
type GatewayEmbedder struct {
	Client     EmbeddingClient
	ResolveURL func(context.Context) (string, error)
	HTTPClient connect.HTTPClient
	Role       string
}

// NewGatewayEmbedder resolves ai-gateway through lifecycle discovery and uses a
// bounded client: a detail read must never wait a minute on an embedding.
func NewGatewayEmbedder() *GatewayEmbedder {
	return &GatewayEmbedder{
		ResolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		},
		HTTPClient: &http.Client{Timeout: 8 * time.Second},
		Role:       embeddingRole,
	}
}

func (e *GatewayEmbedder) Embed(ctx context.Context, texts []string) (Embedding, error) {
	if e == nil || len(texts) == 0 {
		return Embedding{}, errors.New("embedding requires a configured gateway and non-empty texts")
	}
	client, err := e.client(ctx)
	if err != nil {
		return Embedding{}, err
	}
	inputs := make([]string, len(texts))
	for i, text := range texts {
		if strings.TrimSpace(text) == "" {
			return Embedding{}, errors.New("embedding text is empty")
		}
		inputs[i] = text
	}
	role := e.Role
	if role == "" {
		role = embeddingRole
	}
	request := connect.NewRequest(&inferencev1.EmbedRequest{
		Texts: inputs, Role: role,
		Profile:           sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass:      sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacyProvenance: embeddingProvenance,
	})
	response, err := client.Embed(ctx, request)
	if err != nil {
		return Embedding{}, fmt.Errorf("ai-gateway embedding: %w", err)
	}
	if response == nil || response.Msg == nil {
		return Embedding{}, errors.New("ai-gateway returned no embedding response")
	}
	msg := response.Msg
	if gatewayErr := msg.GetError(); gatewayErr != nil {
		return Embedding{}, fmt.Errorf("ai-gateway refused embedding (code %s)", gatewayErr.GetCode().String())
	}
	if len(msg.GetVectors()) != len(inputs) {
		return Embedding{}, fmt.Errorf("ai-gateway returned %d vectors for %d texts", len(msg.GetVectors()), len(inputs))
	}
	dimensions := int(msg.GetDimension())
	if dimensions <= 0 {
		return Embedding{}, errors.New("ai-gateway returned no embedding dimension")
	}
	vectors := make([][]float32, 0, len(inputs))
	for _, item := range msg.GetVectors() {
		values := item.GetValues()
		if len(values) != dimensions {
			return Embedding{}, fmt.Errorf("ai-gateway returned %d values, expected %d", len(values), dimensions)
		}
		vector := make([]float32, len(values))
		for i, value := range values {
			vector[i] = float32(value)
		}
		vectors = append(vectors, vector)
	}
	return Embedding{Model: msg.GetModel(), Dimensions: dimensions, Vectors: vectors}, nil
}

func (e *GatewayEmbedder) client(ctx context.Context) (EmbeddingClient, error) {
	if e.Client != nil {
		return e.Client, nil
	}
	if e.ResolveURL == nil {
		return nil, errors.New("ai-gateway URL resolver is unavailable")
	}
	baseURL, err := e.ResolveURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve ai-gateway: %w", err)
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, errors.New("ai-gateway URL is empty")
	}
	httpClient := e.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 8 * time.Second}
	}
	return inferenceconnect.NewInferenceServiceClient(httpClient, baseURL), nil
}
