package retrieval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/api-core/discovery"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const (
	EmbeddingRole       = "embedding.default"
	codeFactsProvenance = "code-facts.repository-index:catalog-filtered-private-source-text-local-embedding"
)

type RouteClient interface {
	ExecuteRoute(context.Context, *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error)
}

type RouteResolver func(context.Context) (string, error)

// GatewayEmbedder uses the generic text-embedding route. It has no provider or
// model selector; role policy and LOCAL_ONLY eligibility remain gateway-owned.
type GatewayEmbedder struct {
	Client     RouteClient
	ResolveURL RouteResolver
	HTTPClient connect.HTTPClient
	Dimensions int
	TaskPrefix string
}

func NewGatewayEmbedder(dimensions int, taskPrefix string) *GatewayEmbedder {
	return &GatewayEmbedder{Dimensions: dimensions, TaskPrefix: taskPrefix,
		ResolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		},
		HTTPClient: &http.Client{Timeout: 45 * time.Second}}
}

func (e *GatewayEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if e == nil || len(texts) == 0 {
		return nil, errors.New("embedding requires a configured gateway and non-empty texts")
	}
	client, err := e.client(ctx)
	if err != nil {
		return nil, err
	}
	result := make([][]float32, 0, len(texts))
	for _, text := range texts {
		if strings.TrimSpace(text) == "" {
			return nil, errors.New("embedding text is empty")
		}
		request := connect.NewRequest(&routingv1.ExecuteRouteRequest{
			Request: &sharedv1.GatewayRequest{Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING,
				Role: EmbeddingRole, Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY,
				PrivacyClass:      sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
				PrivacyProvenance: codeFactsProvenance, Operation: "repository_index", Scenario: "code-facts",
				TimeoutMs: 45000, RequestId: uuid.NewString()},
			InputText: e.TaskPrefix + text,
		})
		response, callErr := client.ExecuteRoute(ctx, request)
		if callErr != nil {
			return nil, fmt.Errorf("ai-gateway embedding route: %w", callErr)
		}
		if response == nil || response.Msg == nil {
			return nil, errors.New("ai-gateway returned no embedding response")
		}
		msg := response.Msg
		if !msg.GetValid() {
			return nil, fmt.Errorf("ai-gateway refused local embedding: %s", strings.Join(msg.GetPolicyReasons(), "; "))
		}
		evidence := msg.GetEvidence()
		if evidence == nil || evidence.GetSelectedLocality() != "local" {
			locality := "unknown"
			if evidence != nil {
				locality = evidence.GetSelectedLocality()
			}
			return nil, fmt.Errorf("ai-gateway selected non-local embedding locality %q", locality)
		}
		var output struct {
			Embedding []float32 `json:"embedding"`
		}
		if err := json.Unmarshal([]byte(msg.GetOutputText()), &output); err != nil {
			return nil, fmt.Errorf("decode ai-gateway embedding: %w", err)
		}
		vector := output.Embedding
		if len(vector) == 0 || (e.Dimensions > 0 && len(vector) != e.Dimensions) {
			return nil, fmt.Errorf("ai-gateway embedding dimensions %d, expected %d", len(vector), e.Dimensions)
		}
		result = append(result, vector)
	}
	return result, nil
}

func (e *GatewayEmbedder) client(ctx context.Context) (RouteClient, error) {
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
	return routingconnect.NewRoutingServiceClient(httpClient, baseURL), nil
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
	model, dimensions := query.EmbeddingModel, query.EmbeddingDimensions
	if model == "" {
		policy, err := resolve(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve local embedding role %q: %w", EmbeddingRole, err)
		}
		model, dimensions = policy.Model, policy.Dimensions
	}
	embedder := NewGatewayEmbedder(dimensions, TaskPrefixForEmbeddingModel(model, true))
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
