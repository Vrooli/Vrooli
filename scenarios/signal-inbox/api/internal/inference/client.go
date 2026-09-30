// Package inference is Signal Inbox's sole boundary for model inference.
package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	aisearch "github.com/vrooli/ai-go/search"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const (
	EmbeddingRole      = "embedding.default"
	ClassificationRole = "classify.routing"
	queryPrefix        = "search_query: "
	documentPrefix     = "search_document: "
)

// EmbeddingTask preserves the asymmetric query/document instruction required
// by the gateway's embedding role.
type EmbeddingTask string

const (
	EmbeddingDocument EmbeddingTask = "document"
	EmbeddingQuery    EmbeddingTask = "query"
)

// Client is the scenario-owned inference seam. Callers never import provider
// clients or invoke resource commands directly.
type Client interface {
	Embed(context.Context, string, EmbeddingTask) ([]float64, error)
	Classify(context.Context, string) (string, error)
}

// GatewayClient routes every inference request through ai-gateway.
type GatewayClient struct {
	routing routingconnect.RoutingServiceClient
}

func NewGatewayClient(routing routingconnect.RoutingServiceClient) *GatewayClient {
	return &GatewayClient{routing: routing}
}

func (c *GatewayClient) Embed(ctx context.Context, text string, task EmbeddingTask) ([]float64, error) {
	output, err := c.execute(ctx, sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING, EmbeddingRole, embeddingInput(task, text))
	if err != nil {
		return nil, err
	}
	var response struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		return nil, fmt.Errorf("decode ai-gateway embedding response: %w", err)
	}
	if len(response.Embedding) == 0 {
		return nil, errors.New("ai-gateway embedding response contained no vector")
	}
	return response.Embedding, nil
}

func (c *GatewayClient) Classify(ctx context.Context, prompt string) (string, error) {
	return c.execute(ctx, sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION, ClassificationRole, prompt)
}

func (c *GatewayClient) execute(ctx context.Context, kind sharedv1.RequestKind, role, input string) (string, error) {
	if c == nil || c.routing == nil {
		return "", errors.New("ai-gateway routing client is not configured")
	}
	response, err := c.routing.ExecuteRoute(ctx, connect.NewRequest(&routingv1.ExecuteRouteRequest{
		Request: &sharedv1.GatewayRequest{
			Kind:         kind,
			Role:         role,
			Profile:      sharedv1.Profile_PROFILE_LOCAL_FIRST,
			PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
			Scenario:     "signal-inbox",
		},
		InputText: input,
	}))
	if err != nil {
		return "", fmt.Errorf("execute ai-gateway route %s: %w", role, err)
	}
	if response == nil || response.Msg == nil {
		return "", errors.New("ai-gateway returned no route response")
	}
	if !response.Msg.GetValid() {
		return "", fmt.Errorf("ai-gateway rejected %s route: %s", role, routeIssues(response.Msg.GetIssues()))
	}
	return strings.TrimSpace(response.Msg.GetOutputText()), nil
}

func embeddingInput(task EmbeddingTask, text string) string {
	if task == EmbeddingQuery {
		return queryPrefix + text
	}
	return documentPrefix + text
}

func routeIssues(issues []*sharedv1.ValidationIssue) string {
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		parts = append(parts, issue.GetMessage())
	}
	return strings.Join(parts, "; ")
}

// Embedder adapts Client to ai-go's task-aware embedding seam.
type Embedder struct{ Client Client }

func (e Embedder) Embed(ctx context.Context, text string) ([]float64, error) {
	return e.EmbedDocument(ctx, text)
}

func (e Embedder) EmbedQuery(ctx context.Context, text string) ([]float64, error) {
	return e.Client.Embed(ctx, text, EmbeddingQuery)
}

func (e Embedder) EmbedDocument(ctx context.Context, text string) ([]float64, error) {
	return e.Client.Embed(ctx, text, EmbeddingDocument)
}

func (e Embedder) Available(ctx context.Context) bool {
	_, err := e.Embed(ctx, "ping")
	return err == nil
}

var (
	_ aisearch.Embedder     = Embedder{}
	_ aisearch.TaskEmbedder = Embedder{}
)
