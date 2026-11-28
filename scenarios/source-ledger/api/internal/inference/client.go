// Package inference is Source Ledger's only boundary for model inference.
package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"source-ledger/internal/policy"

	aisearch "github.com/vrooli/ai-go/search"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const (
	EmbeddingRole      = "embedding.default"
	ClassificationRole = "classify.routing"
	SummaryRole        = "summarize.default"
	// privacyProvenance identifies the internal journal source and its
	// classification basis without including corpus content.
	privacyProvenance = "source_ledger.internal_journal"
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
	clusteringPrefix              = "clustering: "
	documentPrefix                = "search_document: "
	queryPrefix                   = "search_query: "
	classificationPromptPrefix    = "Classify the memory into exactly one allowed facet ID. Return only the ID, no punctuation or explanation."
	// Facet assignment is a six-way coarse classification, so a compact
	// head-and-tail excerpt is sufficient and keeps corpus replay within the
	// gateway's bounded request window.
	classificationInputRunes = 4000
	// Few-shot examples are policy evidence, not a second corpus payload. Keep
	// each facet's examples short and bounded so the six-way vocabulary cannot
	// crowd the memory being classified out of a local model's context window.
	classificationExamplesPerFacet = 1
	classificationExampleRunes     = 300
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

// ContextualClassifier is an optional extension for callers that have
// validated entry metadata. Keeping it separate preserves the generic Client
// seam for embeddings, summaries, and test fakes while allowing journal
// classification to carry policy-relevant kind information.
type ContextualClassifier interface {
	ClassifyEntry(context.Context, string, string) (string, error)
}

type (
	VocabularyEntry struct {
		ID, Label, Guidance string
		Examples            []string
	}
	VocabularyProvider func(context.Context) ([]VocabularyEntry, error)
)

type GatewayClient struct {
	routing    routingconnect.RoutingServiceClient
	vocabulary VocabularyProvider
	// inflight and lastDone describe foreground calls only, so background
	// indexing can step aside for recall, journal and rule work.
	inflight atomic.Int32
	lastDone atomic.Int64
	// backgroundGate admits one background embedding request at a time. The
	// quiet-gap check alone is not a reservation: parallel index workers can
	// all observe the same idle window and flood the gateway together.
	backgroundGateOnce sync.Once
	backgroundGate     chan struct{}
	// batch, when set, embeds many texts per call through ai-gateway's
	// local-only InferenceService.Embed (the file corpus's bulk path).
	batch inferenceconnect.InferenceServiceClient
	// summaryMaxOutputTokens bounds one compaction summary's generation.
	summaryMaxOutputTokens int
}

// WithSummaryMaxOutputTokens sets the summary generation ceiling. A
// non-positive value keeps policy.DefaultSummaryMaxOutputTokens.
func (c *GatewayClient) WithSummaryMaxOutputTokens(tokens int) *GatewayClient {
	if tokens > 0 {
		c.summaryMaxOutputTokens = tokens
	}
	return c
}

// WithBatchEmbedding enables EmbedBatch through ai-gateway's typed,
// local-only embedding RPC.
func (c *GatewayClient) WithBatchEmbedding(client inferenceconnect.InferenceServiceClient) *GatewayClient {
	c.batch = client
	return c
}

// idleGap is how long foreground inference must have been quiet before a
// background embedding starts. Searches arrive in bursts (Search Hub fans
// out), so one quiet gap covers the burst.
const idleGap = 2 * time.Second

func NewGatewayClient(routing routingconnect.RoutingServiceClient, vocabulary ...VocabularyProvider) *GatewayClient {
	c := &GatewayClient{routing: routing, summaryMaxOutputTokens: policy.DefaultSummaryMaxOutputTokens}
	if len(vocabulary) > 0 {
		c.vocabulary = vocabulary[0]
	}
	return c
}

func (c *GatewayClient) Embed(ctx context.Context, text string, task EmbeddingTask) ([]float64, error) {
	output, err := c.execute(ctx, sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING, EmbeddingRole, embeddingInput(task, text), EmbeddingTimeout, 0)
	if err != nil {
		return nil, err
	}
	return decodeEmbedding(output)
}

// EmbedWhenIdle embeds a document for background indexing. It waits until no
// foreground call is running and none finished within idleGap, and it is not
// itself counted as foreground, so interactive work never queues behind it.
func (c *GatewayClient) EmbedWhenIdle(ctx context.Context, text string) ([]float64, error) {
	release, err := c.acquireBackground(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	output, err := c.route(ctx, sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING, EmbeddingRole, embeddingInput(EmbeddingDocument, text), EmbeddingTimeout, 0)
	if err != nil {
		return nil, err
	}
	return decodeEmbedding(output)
}

// embedEach embeds texts one call at a time, for clients without the batch
// service.
func (c *GatewayClient) embedEach(ctx context.Context, task EmbeddingTask, texts []string, background bool) ([][]float64, error) {
	out := make([][]float64, len(texts))
	for i, text := range texts {
		var err error
		if background {
			out[i], err = c.EmbedWhenIdle(ctx, text)
		} else {
			out[i], err = c.Embed(ctx, text, task)
		}
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// EmbedBatch embeds texts for one task in a single ai-gateway call. A
// background batch first waits for foreground quiet (see EmbedWhenIdle) and
// is not counted as foreground; a foreground batch (a query) is.
func (c *GatewayClient) EmbedBatch(ctx context.Context, task EmbeddingTask, texts []string, background bool) ([][]float64, error) {
	if c == nil || c.batch == nil {
		return c.embedEach(ctx, task, texts, background)
	}
	if background {
		release, err := c.acquireBackground(ctx)
		if err != nil {
			return nil, err
		}
		defer release()
	} else {
		c.inflight.Add(1)
		defer func() {
			c.lastDone.Store(time.Now().UnixNano())
			c.inflight.Add(-1)
		}()
	}
	inputs := make([]string, len(texts))
	for i, text := range texts {
		inputs[i] = embeddingInput(task, text)
	}
	ctx, cancel := context.WithTimeout(ctx, EmbeddingTimeout+time.Duration(len(texts))*time.Second)
	defer cancel()
	resp, err := c.batch.Embed(ctx, connect.NewRequest(&inferencev1.EmbedRequest{
		Role:  EmbeddingRole,
		Texts: inputs,
		// The typed embedding RPC refuses a request without an explicit
		// local-only profile and privacy attribution, so omitting them failed
		// every file-corpus batch and fast query embedding.
		Profile:           sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass:      sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		PrivacyProvenance: privacyProvenance,
	}))
	if err != nil {
		return nil, fmt.Errorf("ai-gateway embed batch: %w", err)
	}
	if e := resp.Msg.GetError(); e != nil {
		return nil, fmt.Errorf("ai-gateway embed batch: %s", e.GetMessage())
	}
	if len(resp.Msg.GetVectors()) != len(texts) {
		return nil, fmt.Errorf("ai-gateway embed batch returned %d vectors for %d texts", len(resp.Msg.GetVectors()), len(texts))
	}
	out := make([][]float64, len(texts))
	for i, v := range resp.Msg.GetVectors() {
		out[i] = v.GetValues()
	}
	return out, nil
}

// acquireBackground reserves the shared low-priority slot before checking
// foreground quiet. Waiting workers recheck after they acquire it, so a
// foreground request arriving during one batch prevents the queued batches
// from starting until the next quiet window.
func (c *GatewayClient) acquireBackground(ctx context.Context) (func(), error) {
	if c == nil {
		return nil, errors.New("inference client is nil")
	}
	c.backgroundGateOnce.Do(func() {
		c.backgroundGate = make(chan struct{}, 1)
	})
	select {
	case c.backgroundGate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	release := func() { <-c.backgroundGate }
	if err := c.waitIdle(ctx); err != nil {
		release()
		return nil, err
	}
	return release, nil
}

func (c *GatewayClient) waitIdle(ctx context.Context) error {
	for c.inflight.Load() > 0 || time.Since(time.Unix(0, c.lastDone.Load())) < idleGap {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return nil
}

func decodeEmbedding(output string) ([]float64, error) {
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
	return c.classify(ctx, prompt, "")
}

func (c *GatewayClient) ClassifyEntry(ctx context.Context, memory, kind string) (string, error) {
	return c.classify(ctx, memory, kind)
}

func (c *GatewayClient) classify(ctx context.Context, memory, kind string) (string, error) {
	allowed := []VocabularyEntry(nil)
	if c.vocabulary != nil {
		var err error
		allowed, err = c.vocabulary(ctx)
		if err != nil {
			return "", fmt.Errorf("resolve classification vocabulary: %w", err)
		}
	}
	output, err := c.execute(ctx, sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION, ClassificationRole, classificationPrompt(classificationMemory(memory, kind), allowed...), GenerationTimeout, ClassificationMaxOutputTokens)
	if err != nil {
		return "", err
	}
	return generatedText(output)
}

func classificationMemory(memory, kind string) string {
	if strings.TrimSpace(kind) == "" {
		return memory
	}
	return "Entry kind: " + strings.TrimSpace(kind) + "\nMemory:\n" + memory
}

func (c *GatewayClient) Summarize(ctx context.Context, prompt string) (string, error) {
	output, err := c.execute(ctx, sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION, SummaryRole, prompt, SummaryTimeout, c.summaryMaxOutputTokens)
	if err != nil {
		return "", err
	}
	return generatedText(output)
}

// execute runs one foreground inference call.
func (c *GatewayClient) execute(ctx context.Context, kind sharedv1.RequestKind, role, input string, timeout time.Duration, maxOutputTokens int) (string, error) {
	if c != nil {
		c.inflight.Add(1)
		defer func() {
			c.lastDone.Store(time.Now().UnixNano())
			c.inflight.Add(-1)
		}()
	}
	return c.route(ctx, kind, role, input, timeout, maxOutputTokens)
}

func (c *GatewayClient) route(ctx context.Context, kind sharedv1.RequestKind, role, input string, timeout time.Duration, maxOutputTokens int) (string, error) {
	if c == nil || c.routing == nil {
		return "", errors.New("ai-gateway routing client is not configured")
	}
	resp, err := c.routing.ExecuteRoute(ctx, connect.NewRequest(&routingv1.ExecuteRouteRequest{Request: &sharedv1.GatewayRequest{Kind: kind, Role: role, Profile: sharedv1.Profile_PROFILE_LOCAL_FIRST, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL, PrivacyProvenance: privacyProvenance, Scenario: "source-ledger", TimeoutMs: int32(timeout / time.Millisecond), MaxOutputTokens: int32(maxOutputTokens)}, InputText: input}))
	if err != nil {
		return "", fmt.Errorf("execute ai-gateway route %s: %w", role, err)
	}
	if resp == nil || resp.Msg == nil {
		return "", errors.New("ai-gateway returned no route response")
	}
	if !resp.Msg.GetValid() {
		return "", fmt.Errorf("ai-gateway rejected %s route: %s", role, routeIssues(resp.Msg.GetIssues()))
	}
	if ev := resp.Msg.GetEvidence(); ev != nil && ev.GetStatus() != "" && ev.GetStatus() != routeStatusSucceeded {
		return "", &RouteNotSucceededError{Role: role, Status: ev.GetStatus(), RejectionReason: ev.GetRejectionReason(), FailureReasons: ev.GetFailureReasons()}
	}
	return strings.TrimSpace(resp.Msg.GetOutputText()), nil
}

// routeStatusSucceeded is the only route evidence status that carries output.
const routeStatusSucceeded = "succeeded"

// RouteNotSucceededError reports a valid route request that ai-gateway could
// not run (failed, or blocked by an open breaker or no eligible provider). The
// gateway's own reason is kept so callers can log and retry on it instead of
// decoding empty output.
type RouteNotSucceededError struct {
	Role            string
	Status          string
	RejectionReason string
	FailureReasons  []string
}

func (e *RouteNotSucceededError) Error() string {
	msg := fmt.Sprintf("ai-gateway %s route %s", e.Role, e.Status)
	if e.RejectionReason != "" {
		msg += ": " + e.RejectionReason
	}
	if len(e.FailureReasons) > 0 {
		msg += " (" + strings.Join(e.FailureReasons, "; ") + ")"
	}
	return msg
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

func classificationPrompt(memory string, allowed ...VocabularyEntry) string {
	prefix := classificationPromptPrefix
	if len(allowed) > 0 {
		prefix += " Allowed facet IDs and policy guidance (return only the exact ID):"
		for _, facet := range allowed {
			prefix += "\n- " + facet.ID + ": " + facet.Label
			if strings.TrimSpace(facet.Guidance) != "" {
				prefix += " — " + facet.Guidance
			}
			for i, example := range facet.Examples {
				if i >= classificationExamplesPerFacet {
					break
				}
				if strings.TrimSpace(example) != "" {
					prefix += "\n  example: " + classificationExcerptLimit(example, classificationExampleRunes)
				}
			}
		}
	}
	return prefix + "\nMemory: " + classificationExcerpt(memory)
}

func classificationExcerpt(memory string) string {
	return classificationExcerptLimit(memory, classificationInputRunes)
}

func classificationExcerptLimit(memory string, limit int) string {
	trimmed := strings.TrimSpace(memory)
	runes := []rune(trimmed)
	if len(runes) <= limit {
		return trimmed
	}
	half := limit / 2
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

// batchEmbedder is GatewayClient's bulk path.
type batchEmbedder interface {
	EmbedBatch(context.Context, EmbeddingTask, []string, bool) ([][]float64, error)
}

// EmbedQueryFast embeds one query through the bulk path when the client has
// one (a typed local call, about half the latency of a routed one).
func (e Embedder) EmbedQueryFast(ctx context.Context, text string) ([]float64, error) {
	if b, ok := e.Client.(batchEmbedder); ok {
		out, err := b.EmbedBatch(ctx, EmbeddingQuery, []string{text}, false)
		if err != nil {
			return nil, err
		}
		return out[0], nil
	}
	return e.EmbedQuery(ctx, text)
}

// EmbedDocumentsWhenIdle embeds a background batch, stepping aside for
// foreground inference first. Clients without a bulk path embed one by one.
func (e Embedder) EmbedDocumentsWhenIdle(ctx context.Context, texts []string) ([][]float64, error) {
	if b, ok := e.Client.(batchEmbedder); ok {
		return b.EmbedBatch(ctx, EmbeddingDocument, texts, true)
	}
	out := make([][]float64, len(texts))
	for i, text := range texts {
		v, err := e.EmbedDocument(ctx, text)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

func (e Embedder) Available(ctx context.Context) bool {
	_, err := e.Embed(ctx, "ping")
	return err == nil
}

var (
	_ aisearch.Embedder     = Embedder{}
	_ aisearch.TaskEmbedder = Embedder{}
)
