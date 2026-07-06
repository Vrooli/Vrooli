package search

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	internalchat "portal/internal/chat"
	"portal/internal/clock"
	"portal/internal/integrations/registry"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"
)

const (
	defaultSuggestBudget = 1500 * time.Millisecond
	defaultAttachBudget  = 5 * time.Second
	defaultSuggestLimit  = 5
	defaultAttachLimit   = 6
	maxContextHits       = 5
)

type HubClient interface {
	Query(ctx context.Context, input QueryInput) (QueryResult, error)
}

type QueryInput struct {
	Query  string
	Types  []string
	Limit  int32
	Group  string
	Budget time.Duration
}

type QueryResult struct {
	Hits      []internalchat.SearchHit
	Degraded  bool
	Reason    string
	LatencyMS int64
}

type AttachmentResult struct {
	Attachment internalchat.SearchAttachment
	Err        error
}

type Service struct {
	chat     *internalchat.Service
	hub      HubClient
	registry *registry.Service
	clock    clock.Clock
}

type Config struct {
	Chat     *internalchat.Service
	Hub      HubClient
	Registry *registry.Service
	Clock    clock.Clock
}

func NewService(cfg Config) *Service {
	clk := cfg.Clock
	if clk == nil {
		clk = clock.System{}
	}
	hub := cfg.Hub
	if hub == nil {
		hub = NewSearchHubClient(nil)
	}
	return &Service{chat: cfg.Chat, hub: hub, registry: cfg.Registry, clock: clk}
}

func (s *Service) Suggest(ctx context.Context, input QueryInput) (QueryResult, error) {
	input.Query = strings.TrimSpace(input.Query)
	if input.Query == "" {
		return QueryResult{}, nil
	}
	if input.Limit <= 0 {
		input.Limit = defaultSuggestLimit
	}
	if input.Budget <= 0 {
		input.Budget = defaultSuggestBudget
	}
	return s.query(ctx, input)
}

func (s *Service) StartAttachment(ctx context.Context, chatID, messageID string) <-chan AttachmentResult {
	ch := make(chan AttachmentResult, 1)
	if s == nil || s.chat == nil {
		ch <- AttachmentResult{Err: errors.New("search service is not configured")}
		close(ch)
		return ch
	}
	go func() {
		defer close(ch)
		bg := context.WithoutCancel(ctx)
		attachCtx, cancel := context.WithTimeout(bg, defaultAttachBudget)
		defer cancel()
		attachment, err := s.AttachForMessage(attachCtx, chatID, messageID)
		ch <- AttachmentResult{Attachment: attachment, Err: err}
	}()
	return ch
}

func (s *Service) AttachForMessage(ctx context.Context, chatID, messageID string) (internalchat.SearchAttachment, error) {
	messages, _, err := s.chat.GetTree(ctx, chatID)
	if err != nil {
		return internalchat.SearchAttachment{}, err
	}
	msg, ok := findMessage(messages, messageID)
	if !ok {
		return internalchat.SearchAttachment{}, internalchat.ErrNotFound{Resource: "message", ID: messageID}
	}
	query := strings.TrimSpace(msg.Content)
	if query == "" {
		return internalchat.SearchAttachment{}, internalchat.ErrInvalidInput
	}
	result, err := s.query(ctx, QueryInput{Query: query, Limit: defaultAttachLimit, Budget: defaultAttachBudget})
	if err != nil {
		return internalchat.SearchAttachment{}, err
	}
	if len(result.Hits) == 0 && !result.Degraded {
		return internalchat.SearchAttachment{}, nil
	}
	return s.chat.CreateSearchAttachment(ctx, internalchat.CreateSearchAttachmentInput{
		ChatID:    chatID,
		MessageID: messageID,
		Query:     query,
		Hits:      result.Hits,
		Degraded:  result.Degraded,
		Reason:    result.Reason,
		LatencyMS: result.LatencyMS,
	})
}

func (s *Service) RecentContextBlock(ctx context.Context, chatID string) string {
	if s == nil || s.chat == nil {
		return ""
	}
	attachments, err := s.chat.ListSearchAttachments(ctx, chatID, 3)
	if err != nil || len(attachments) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Recent Vrooli ecosystem search context. Treat as supplemental context with provenance; do not claim it is exhaustive.\n")
	count := 0
	for _, attachment := range attachments {
		for _, hit := range attachment.Hits {
			if count >= maxContextHits {
				return b.String()
			}
			count++
			b.WriteString(fmt.Sprintf("- [%s/%s] %s", hit.ProviderID, hit.Type, oneLine(hit.Title)))
			if hit.Path != "" {
				b.WriteString(" (")
				b.WriteString(oneLine(hit.Path))
				b.WriteString(")")
			}
			if hit.Snippet != "" {
				b.WriteString(": ")
				b.WriteString(oneLine(hit.Snippet))
			}
			b.WriteString("\n")
		}
	}
	if count == 0 {
		return ""
	}
	return b.String()
}

func (s *Service) query(ctx context.Context, input QueryInput) (QueryResult, error) {
	start := s.clock.Now()
	result, err := s.hub.Query(ctx, input)
	latency := s.clock.Now().Sub(start)
	if latency < 0 {
		latency = 0
	}
	ok := err == nil
	reason := result.Reason
	if err != nil {
		reason = err.Error()
	}
	if s.registry != nil {
		s.registry.Observe(registry.IntegrationSearchHub, latency, ok, result.Degraded, reason)
	}
	if err != nil {
		return QueryResult{
			Degraded:  true,
			Reason:    reason,
			LatencyMS: latency.Milliseconds(),
		}, nil
	}
	if result.LatencyMS == 0 {
		result.LatencyMS = latency.Milliseconds()
	}
	return result, nil
}

type SearchHubClient struct {
	httpClient *http.Client
	resolver   *discovery.Resolver
}

func NewSearchHubClient(client *http.Client) *SearchHubClient {
	if client == nil {
		client = &http.Client{Timeout: defaultAttachBudget}
	}
	return &SearchHubClient{httpClient: client, resolver: discovery.NewResolver(discovery.ResolverConfig{})}
}

func (c *SearchHubClient) Query(ctx context.Context, input QueryInput) (QueryResult, error) {
	baseURL, err := c.baseURL(ctx)
	if err != nil {
		return QueryResult{}, err
	}
	budget := input.Budget
	if budget <= 0 {
		budget = defaultSuggestBudget
	}
	queryCtx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	client := routingconnect.NewRoutingServiceClient(c.httpClient, strings.TrimRight(baseURL, "/"))
	resp, err := client.Query(queryCtx, connect.NewRequest(&routingv1.QueryRequest{
		Query: strings.TrimSpace(input.Query),
		Types: input.Types,
		Limit: input.Limit,
		Group: strings.TrimSpace(input.Group),
	}))
	if err != nil {
		return QueryResult{}, err
	}
	hits := projectHits(resp.Msg)
	return QueryResult{
		Hits:      limitHits(hits, int(input.Limit)),
		Degraded:  resp.Msg.GetDegraded(),
		Reason:    degradedReason(resp.Msg),
		LatencyMS: resp.Msg.GetLatencyMs(),
	}, nil
}

func (c *SearchHubClient) baseURL(ctx context.Context) (string, error) {
	if value := strings.TrimSpace(os.Getenv("SEARCH_HUB_API_URL")); value != "" {
		return strings.TrimRight(value, "/"), nil
	}
	return c.resolver.ResolveScenarioURLDefault(ctx, "search-hub")
}

func projectHits(resp *routingv1.QueryResponse) []internalchat.SearchHit {
	if resp == nil {
		return nil
	}
	source := resp.GetRanked()
	if len(source) == 0 {
		for _, group := range resp.GetGroups() {
			source = append(source, group.GetHits()...)
		}
	}
	out := make([]internalchat.SearchHit, 0, len(source))
	for _, hit := range source {
		out = append(out, internalchat.SearchHit{
			ProviderID:  hit.GetProviderId(),
			Type:        hit.GetType(),
			Title:       hit.GetTitle(),
			Snippet:     hit.GetSnippet(),
			Path:        hit.GetPath(),
			Score:       hit.GetScore(),
			RerankScore: hit.GetRerankScore(),
			Locations:   hit.GetLocations(),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		left := out[i].RerankScore
		right := out[j].RerankScore
		if left == 0 && right == 0 {
			left = out[i].Score
			right = out[j].Score
		}
		return left > right
	})
	return out
}

func limitHits(hits []internalchat.SearchHit, limit int) []internalchat.SearchHit {
	if limit <= 0 || len(hits) <= limit {
		return hits
	}
	return hits[:limit]
}

func degradedReason(resp *routingv1.QueryResponse) string {
	if resp == nil || !resp.GetDegraded() {
		return ""
	}
	for _, group := range resp.GetGroups() {
		if group.GetDegraded() && group.GetNote() != "" {
			return group.GetNote()
		}
	}
	return "search-hub returned degraded results"
}

func findMessage(messages []internalchat.Message, id string) (internalchat.Message, bool) {
	for _, msg := range messages {
		if msg.ID == id {
			return msg, true
		}
	}
	return internalchat.Message{}, false
}

func oneLine(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 220 {
		return value[:220] + "..."
	}
	return value
}
