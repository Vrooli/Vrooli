package retrieval

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/api-core/discovery"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const (
	EmbeddingRole       = "embedding.default"
	codeFactsProvenance = "code-facts.repository-index:catalog-filtered-private-source-text-local-embedding"
)

type EmbeddingClient interface {
	Embed(context.Context, *connect.Request[inferencev1.EmbedRequest]) (*connect.Response[inferencev1.EmbedResponse], error)
}

type RouteResolver func(context.Context) (string, error)

// GatewayEmbedder uses the typed embedding contract. Role selection and
// LOCAL_ONLY eligibility remain gateway-owned; the resolved policy model is
// checked against the response before vectors are accepted.
type GatewayEmbedder struct {
	Client        EmbeddingClient
	ResolveURL    RouteResolver
	HTTPClient    connect.HTTPClient
	Dimensions    int
	ExpectedModel string
	TaskPrefix    string
}

func NewGatewayEmbedder(model string, dimensions int, taskPrefix string) *GatewayEmbedder {
	return &GatewayEmbedder{ExpectedModel: model, Dimensions: dimensions, TaskPrefix: taskPrefix,
		ResolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		},
		HTTPClient: &http.Client{Timeout: 45 * time.Second}}
}

func (e *GatewayEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if e == nil || len(texts) == 0 {
		return nil, errors.New("embedding requires a configured gateway and non-empty texts")
	}
	if strings.TrimSpace(e.ExpectedModel) == "" {
		return nil, errors.New("embedding requires a model resolved from Code Facts policy")
	}
	client, err := e.client(ctx)
	if err != nil {
		return nil, err
	}
	inputs := make([]string, len(texts))
	for index, text := range texts {
		if strings.TrimSpace(text) == "" {
			return nil, errors.New("embedding text is empty")
		}
		inputs[index] = e.TaskPrefix + text
	}
	request := connect.NewRequest(&inferencev1.EmbedRequest{
		Texts: inputs, Role: EmbeddingRole,
		Profile:           sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass:      sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacyProvenance: codeFactsProvenance,
	})
	response, callErr := client.Embed(ctx, request)
	if callErr != nil {
		return nil, fmt.Errorf("ai-gateway embedding request: %w", callErr)
	}
	if response == nil || response.Msg == nil {
		return nil, errors.New("ai-gateway returned no embedding response")
	}
	msg := response.Msg
	if gatewayErr := msg.GetError(); gatewayErr != nil {
		return nil, fmt.Errorf("ai-gateway refused embedding request (code %s)", gatewayErr.GetCode().String())
	}
	if msg.GetModel() != e.ExpectedModel {
		return nil, errors.New("ai-gateway embedding model differs from the resolved Code Facts policy")
	}
	if len(msg.GetVectors()) != len(inputs) {
		return nil, fmt.Errorf("ai-gateway returned %d embedding vectors, expected %d", len(msg.GetVectors()), len(inputs))
	}
	result := make([][]float32, 0, len(inputs))
	for _, item := range msg.GetVectors() {
		values := item.GetValues()
		if len(values) == 0 || len(values) != e.Dimensions || int(msg.GetDimension()) != e.Dimensions {
			return nil, fmt.Errorf("ai-gateway embedding dimensions %d, expected %d", len(values), e.Dimensions)
		}
		vector := make([]float32, len(values))
		for i, value := range values {
			vector[i] = float32(value)
		}
		result = append(result, vector)
	}
	return result, nil
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
		httpClient = http.DefaultClient
	}
	return inferenceconnect.NewInferenceServiceClient(httpClient, baseURL), nil
}

type GatewaySemanticSearcher struct {
	Store         VectorStore
	ResolvePolicy func(context.Context) (aisearch.EmbeddingPolicy, error)
}

func (s GatewaySemanticSearcher) SearchSemantic(ctx context.Context, query Query) ([]Candidate, error) {
	if s.Store == nil {
		return nil, errors.New("semantic search is not configured")
	}
	resolve := s.ResolvePolicy
	if resolve == nil {
		resolve = func(ctx context.Context) (aisearch.EmbeddingPolicy, error) {
			return aisearch.ResolveEmbeddingPolicy(ctx, EmbeddingRole)
		}
	}
	policy, err := resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve local embedding role %q: %w", EmbeddingRole, err)
	}
	if query.EmbeddingModel != "" && query.EmbeddingModel != policy.Model {
		return nil, fmt.Errorf("query embedding model %q differs from policy model", query.EmbeddingModel)
	}
	if query.EmbeddingDimensions > 0 && query.EmbeddingDimensions != policy.Dimensions {
		return nil, fmt.Errorf("query embedding dimensions %d differ from policy dimensions", query.EmbeddingDimensions)
	}
	model, dimensions := policy.Model, policy.Dimensions
	embedder := NewGatewayEmbedder(model, dimensions, TaskPrefixForEmbeddingModel(model, true))
	vectors, err := embedder.Embed(ctx, []string{query.Text})
	if err != nil {
		return nil, err
	}
	if len(vectors) != 1 {
		return nil, fmt.Errorf("query embedder returned %d vectors", len(vectors))
	}
	query.EmbeddingModel, query.EmbeddingDimensions = model, dimensions
	return s.Store.Query(ctx, vectors[0], query)
}

var _ Embedder = (*GatewayEmbedder)(nil)

func TaskPrefixForEmbeddingModel(model string, query bool) string {
	queryPrefix, documentPrefix := aisearch.TaskPrefixesFor(model)
	if query {
		return queryPrefix
	}
	return documentPrefix
}
