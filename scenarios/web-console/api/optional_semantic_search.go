package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
	conversationH "web-console/handlers/conversation"
	"web-console/internal/continuity"
)

type semanticArchiveSearchRequest struct {
	Query     string
	Harnesses []string
	Limit     int32
}

type semanticArchiveSearchResult struct {
	Hits         []*semanticArchiveSearchHit    `json:"hits"`
	Coverage     *semanticArchiveSearchCoverage `json:"coverage"`
	TookMs       uint64String                   `json:"tookMs"`
	Degradations []*semanticArchiveDegradation  `json:"degradations"`
}

type semanticArchiveSearchHit struct {
	StableHitID   string                         `json:"stableHitId"`
	RunID         string                         `json:"runId"`
	EventID       string                         `json:"eventId"`
	Role          string                         `json:"role"`
	OccurredAt    string                         `json:"occurredAt"`
	Snippet       string                         `json:"snippet"`
	Harness       string                         `json:"harness"`
	SourceSession string                         `json:"sourceSessionId"`
	Weak          bool                           `json:"weak"`
	Provenance    *semanticArchiveSourceIdentity `json:"provenance"`
}

type semanticArchiveSourceIdentity struct {
	Harness         string `json:"harness"`
	SourceSessionID string `json:"sourceSessionId"`
}

type semanticArchiveSearchCoverage struct {
	CanonicalVisibleMessages uint64String `json:"canonicalVisibleMessages"`
	CatalogDocuments         uint64String `json:"catalogDocuments"`
	LexicalDocuments         uint64String `json:"lexicalDocuments"`
	SemanticDocuments        uint64String `json:"semanticDocuments"`
	PendingDocuments         uint64String `json:"pendingDocuments"`
	DeletedDocuments         uint64String `json:"deletedDocuments"`
	LexicalRatio             float64      `json:"lexicalRatio"`
	SemanticRatio            float64      `json:"semanticRatio"`
	FreshnessLagMs           uint64String `json:"freshnessLagMs"`
}

type semanticArchiveDegradation struct {
	Reason    string `json:"reason"`
	Leg       string `json:"leg"`
	Detail    string `json:"detail"`
	Retryable bool   `json:"retryable"`
}

type uint64String uint64

func (value *uint64String) UnmarshalJSON(raw []byte) error {
	text := string(raw)
	if len(raw) > 0 && raw[0] == '"' {
		if err := json.Unmarshal(raw, &text); err != nil {
			return err
		}
	}
	parsed, err := strconv.ParseUint(text, 10, 64)
	if err == nil {
		*value = uint64String(parsed)
	}
	return err
}

const optionalSemanticSearchTimeout = 1200 * time.Millisecond

var defaultSemanticArchiveHarnesses = []string{"claude", "codex", "grok", "opencode", "gemini-cli", "qwen-code", "antigravity", "cursor"}

type semanticSearchClient interface {
	SearchConversations(context.Context, semanticArchiveSearchRequest) (*semanticArchiveSearchResult, error)
}

func (a *conversationAdapter) SearchSemantic(ctx context.Context, request *conversationv1.SearchSemanticRequest) (*conversationv1.SearchSemanticResponse, error) {
	query := strings.TrimSpace(request.GetQuery())
	if query == "" {
		return nil, conversationH.ErrInvalidArgument
	}
	limit := request.GetLimit()
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	harnesses, err := normalizeSemanticHarnesses(request.GetHarnesses())
	if err != nil {
		return nil, err
	}
	local, err := a.SearchArchived(ctx, conversationH.ArchivedSearchFilter{
		SearchQuery: conversationH.SearchQuery{Query: query, Limit: int(limit)},
	})
	if err != nil {
		return nil, err
	}
	result := &conversationv1.SearchSemanticResponse{
		Local:         archivedSearchResultToProto(local),
		SemanticError: "Agent Manager semantic search is unavailable; local archive results are intact",
	}
	client, err := a.semanticSearchClient(ctx)
	if err != nil {
		return result, nil
	}
	semanticCtx, cancel := context.WithTimeout(ctx, optionalSemanticSearchTimeout)
	defer cancel()
	semantic, err := runOptionalSemanticSearch(semanticCtx, client, semanticArchiveSearchRequest{Query: query, Harnesses: harnesses, Limit: limit})
	if err != nil {
		result.SemanticError = "Agent Manager semantic search failed; local archive results are intact"
		return result, nil
	}
	if semantic.Coverage == nil {
		result.SemanticError = "Agent Manager semantic search returned no coverage; local archive results are intact"
		return result, nil
	}
	result.SemanticAvailable = true
	result.SemanticError = ""
	result.SemanticTookMs = uint64(semantic.TookMs)
	if coverage := semantic.Coverage; coverage != nil {
		result.Coverage = &conversationv1.SemanticSearchCoverage{
			CanonicalVisibleMessages: uint64(coverage.CanonicalVisibleMessages), CatalogDocuments: uint64(coverage.CatalogDocuments),
			LexicalDocuments: uint64(coverage.LexicalDocuments), SemanticDocuments: uint64(coverage.SemanticDocuments),
			PendingDocuments: uint64(coverage.PendingDocuments), DeletedDocuments: uint64(coverage.DeletedDocuments),
			LexicalRatio: coverage.LexicalRatio, SemanticRatio: coverage.SemanticRatio, FreshnessLagMs: uint64(coverage.FreshnessLagMs),
		}
	}
	requestedHarnesses := make(map[string]struct{}, len(harnesses))
	for _, harness := range harnesses {
		requestedHarnesses[harness] = struct{}{}
	}
	seenStableIDs := make(map[string]struct{}, len(semantic.Hits))
	for _, hit := range semantic.Hits {
		if hit == nil {
			continue
		}
		harness, sessionID := hit.Harness, hit.SourceSession
		if hit.Provenance != nil {
			if hit.Harness != "" && hit.Provenance.Harness != "" && !strings.EqualFold(strings.TrimSpace(hit.Harness), strings.TrimSpace(hit.Provenance.Harness)) {
				continue
			}
			if hit.SourceSession != "" && hit.Provenance.SourceSessionID != "" && hit.SourceSession != hit.Provenance.SourceSessionID {
				continue
			}
			if hit.Provenance.Harness != "" {
				harness = hit.Provenance.Harness
			}
			if hit.Provenance.SourceSessionID != "" {
				sessionID = hit.Provenance.SourceSessionID
			}
		}
		// Treat the returned source identity as data that still needs to satisfy
		// the caller's filter. Agent Manager applies the same filter during
		// retrieval, but a stale or malformed response must not leak another
		// harness into this result set.
		if _, allowed := requestedHarnesses[strings.ToLower(strings.TrimSpace(harness))]; !allowed {
			continue
		}
		stableID := strings.TrimSpace(hit.StableHitID)
		if stableID == "" {
			continue
		}
		if _, duplicate := seenStableIDs[stableID]; duplicate {
			continue
		}
		if len(result.SemanticHits) >= int(limit) {
			break
		}
		seenStableIDs[stableID] = struct{}{}
		result.SemanticHits = append(result.SemanticHits, &conversationv1.SemanticSearchHit{
			StableHitId: stableID, RunId: hit.RunID, EventId: hit.EventID, Role: hit.Role,
			OccurredAt: hit.OccurredAt, Snippet: hit.Snippet, Harness: harness,
			SourceSessionId: sessionID, Weak: hit.Weak,
		})
	}
	for _, degradation := range semantic.Degradations {
		if degradation != nil {
			result.Degradations = append(result.Degradations, &conversationv1.SemanticSearchDegradation{
				Reason: degradation.Reason, Leg: degradation.Leg, Detail: degradation.Detail, Retryable: degradation.Retryable,
			})
		}
	}
	return result, nil
}

func (a *conversationAdapter) semanticSearchClient(ctx context.Context) (semanticSearchClient, error) {
	if a != nil && a.semanticClientFactory != nil {
		return a.semanticClientFactory(ctx)
	}
	base, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return nil, err
	}
	return &connectJSONSemanticSearchClient{http: http.DefaultClient, baseURL: base}, nil
}

func archivedSearchResultToProto(result conversationH.ArchivedSearchResult) *conversationv1.SearchArchivedResponse {
	out := &conversationv1.SearchArchivedResponse{Truncated: result.Truncated, TotalMatches: result.TotalMatches, DistinctSessions: result.DistinctSessions, Error: result.Error}
	for _, match := range result.Matches {
		ranges := make([]*conversationv1.TextRange, 0, len(match.Ranges))
		for _, textRange := range match.Ranges {
			ranges = append(ranges, &conversationv1.TextRange{Start: int32(textRange.Start), End: int32(textRange.End)})
		}
		out.Matches = append(out.Matches, &conversationv1.ArchivedSearchMatch{
			EventId: match.EventID, SessionId: match.SessionID, Sequence: match.Sequence, Role: match.Role,
			CreatedAt: match.CreatedAt, Excerpt: match.Excerpt, Ranges: ranges,
		})
	}
	return out
}

func normalizeSemanticHarnesses(harnesses []string) ([]string, error) {
	if len(harnesses) == 0 {
		return append([]string(nil), defaultSemanticArchiveHarnesses...), nil
	}
	out := make([]string, 0, len(harnesses))
	seen := make(map[string]bool, len(harnesses))
	for _, harness := range harnesses {
		harness = strings.ToLower(strings.TrimSpace(harness))
		if continuity.NativeArchiveSourceForAgentType(harness) == "" {
			return nil, conversationH.ErrInvalidArgument
		}
		if !seen[harness] {
			seen[harness] = true
			out = append(out, harness)
		}
	}
	return out, nil
}

func runOptionalSemanticSearch(ctx context.Context, client semanticSearchClient, request semanticArchiveSearchRequest) (*semanticArchiveSearchResult, error) {
	if client == nil {
		return nil, errors.New("semantic search client unavailable")
	}
	harnesses, err := normalizeSemanticHarnesses(request.Harnesses)
	if err != nil {
		return nil, err
	}
	request.Query = strings.TrimSpace(request.Query)
	request.Harnesses = harnesses
	response, err := client.SearchConversations(ctx, request)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, errors.New("semantic search returned an empty response")
	}
	return response, nil
}

const agentManagerSearchConversationsProcedure = "/agent_manager.v1.ConversationSearchService/SearchConversations"

type connectJSONSemanticSearchClient struct {
	http    *http.Client
	baseURL string
}

func (client *connectJSONSemanticSearchClient) SearchConversations(ctx context.Context, request semanticArchiveSearchRequest) (*semanticArchiveSearchResult, error) {
	body, err := json.Marshal(struct {
		Query   string `json:"query"`
		Mode    string `json:"mode"`
		Filters struct {
			Harnesses []string `json:"harnesses"`
		} `json:"filters"`
		PageSize int32 `json:"pageSize"`
	}{Query: request.Query, Mode: "CONVERSATION_SEARCH_MODE_SEMANTIC", Filters: struct {
		Harnesses []string `json:"harnesses"`
	}{Harnesses: request.Harnesses}, PageSize: request.Limit})
	if err != nil {
		return nil, fmt.Errorf("encode semantic request: %w", err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(client.baseURL, "/")+agentManagerSearchConversationsProcedure, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build semantic request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Connect-Protocol-Version", "1")
	httpClient := client.http
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	response, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		limited, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("semantic Connect returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(limited)))
	}
	var decoded semanticArchiveSearchResult
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode semantic Connect response: %w", err)
	}
	return &decoded, nil
}
