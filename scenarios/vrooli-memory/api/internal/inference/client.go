// Package inference is Vrooli Memory's only boundary for model inference.
package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	aisearch "github.com/vrooli/ai-go/search"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const (
	EmbeddingRole      = "embedding.default"
	ClassificationRole = "classify.routing"
	SummaryRole        = "summarize.default"
	// Text generation routinely takes longer than a small embedding request on
	// local models.  Leaving this at the gateway's short provider default turns
	// normal cold-start latency into breaker failures.
	GenerationTimeout = 60 * time.Second
	// Compaction summarizes real clusters and can legitimately require a cold
	// local model load plus generation. It is a background, cancellable pass;
	// applying the short classification deadline turns normal work into a
	// misleading infrastructure failure and leaves the frontier unchanged.
	SummaryTimeout   = 5 * time.Minute
	EmbeddingTimeout = 15 * time.Second
	// Classification is a label-selection operation, not free-form generation.
	// Bounding its output keeps the queue moving and prevents a concise request
	// from monopolizing a local model context window.
	ClassificationMaxOutputTokens = 32
	SummaryMaxOutputTokens        = 256
	clusteringPrefix              = "clustering: "
	documentPrefix                = "search_document: "
	queryPrefix                   = "search_query: "
	classificationPromptPrefix    = "Classify the memory into exactly one allowed facet ID. Return only the ID, no punctuation or explanation. Allowed facet IDs: standing-rule, environment-fact, gotcha, episode, thread, entity-record.\nMemory: "
	// Facet assignment is a six-way coarse classification, so a compact
	// head-and-tail excerpt is sufficient and keeps corpus replay within the
	// gateway's bounded request window.
	classificationInputRunes = 4000
	// The local embedding provider rejects 6k-character payloads once its own
	// prompt overhead is included. Keep this below the measured 5k ceiling with
	// a deliberate safety margin so a queued import cannot repeatedly trip the
	// provider breaker.
	embeddingInputRunes = 4000
)

type EmbeddingTask string

const (
	EmbeddingDocument   EmbeddingTask = "document"
	EmbeddingQuery      EmbeddingTask = "query"
	EmbeddingClustering EmbeddingTask = "clustering"
)

// Client is the scenario-owned inference seam. Implementations route all
// requests through ai-gateway; callers must not import provider clients.
type Client interface {
	Embed(context.Context, string, EmbeddingTask) ([]float64, error)
	Classify(context.Context, string) (string, error)
	Summarize(context.Context, string) (string, error)
}

type GatewayClient struct {
	routing routingconnect.RoutingServiceClient
}

func NewGatewayClient(routing routingconnect.RoutingServiceClient) *GatewayClient {
	return &GatewayClient{routing: routing}
}

func (c *GatewayClient) Embed(ctx context.Context, text string, task EmbeddingTask) ([]float64, error) {
	output, err := c.execute(ctx, sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING, EmbeddingRole, embeddingInput(task, text), EmbeddingTimeout, 0)
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
	output, err := c.execute(ctx, sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION, ClassificationRole, classificationPrompt(prompt), GenerationTimeout, ClassificationMaxOutputTokens)
	if err != nil {
		return "", err
	}
	return generatedText(output)
}

func (c *GatewayClient) Summarize(ctx context.Context, prompt string) (string, error) {
	output, err := c.execute(ctx, sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION, SummaryRole, prompt, SummaryTimeout, SummaryMaxOutputTokens)
	if err != nil {
		return "", err
	}
	return generatedText(output)
}

func (c *GatewayClient) execute(ctx context.Context, kind sharedv1.RequestKind, role, input string, timeout time.Duration, maxOutputTokens int) (string, error) {
	if c == nil || c.routing == nil {
		return "", errors.New("ai-gateway routing client is not configured")
	}
	resp, err := c.routing.ExecuteRoute(ctx, connect.NewRequest(&routingv1.ExecuteRouteRequest{Request: &sharedv1.GatewayRequest{Kind: kind, Role: role, Profile: sharedv1.Profile_PROFILE_LOCAL_FIRST, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL, Scenario: "vrooli-memory", TimeoutMs: int32(timeout / time.Millisecond), MaxOutputTokens: int32(maxOutputTokens)}, InputText: input}))
	if err != nil {
		return "", fmt.Errorf("execute ai-gateway route %s: %w", role, err)
	}
	if resp == nil || resp.Msg == nil {
		return "", errors.New("ai-gateway returned no route response")
	}
	if !resp.Msg.GetValid() {
		return "", fmt.Errorf("ai-gateway rejected %s route: %s", role, routeIssues(resp.Msg.GetIssues()))
	}
	return strings.TrimSpace(resp.Msg.GetOutputText()), nil
}

func embeddingInput(task EmbeddingTask, text string) string {
	text = embeddingExcerpt(text)
	switch task {
	case EmbeddingClustering:
		return clusteringPrefix + text
	case EmbeddingQuery:
		return queryPrefix + text
	default:
		return documentPrefix + text
	}
}

func embeddingExcerpt(text string) string {
	trimmed := strings.TrimSpace(text)
	runes := []rune(trimmed)
	if len(runes) <= embeddingInputRunes {
		return trimmed
	}
	half := embeddingInputRunes / 2
	return string(runes[:half]) + "\n[... embedding excerpt omitted ...]\n" + string(runes[len(runes)-half:])
}

func classificationPrompt(memory string) string {
	return classificationPromptPrefix + classificationExcerpt(memory)
}

func classificationExcerpt(memory string) string {
	trimmed := strings.TrimSpace(memory)
	runes := []rune(trimmed)
	if len(runes) <= classificationInputRunes {
		return trimmed
	}
	half := classificationInputRunes / 2
	return string(runes[:half]) + "\n[... classification excerpt omitted ...]\n" + string(runes[len(runes)-half:])
}

// generatedText decodes the resource gateway's stable JSON envelope.  Keeping
// this at the scenario inference seam prevents provider-shaped JSON from
// leaking into persisted facet IDs or summary nodes.
func generatedText(output string) (string, error) {
	var response struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal([]byte(output), &response); err == nil {
		return strings.TrimSpace(response.Response), nil
	}
	return strings.TrimSpace(output), nil
}

func routeIssues(issues []*sharedv1.ValidationIssue) string {
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		parts = append(parts, issue.GetMessage())
	}
	return strings.Join(parts, "; ")
}

// Embedder adapts Client to ai-go's task-aware embedder seam.
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
