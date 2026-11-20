package search

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search/search_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp + the generated
// Connect-Go client. Mirrors cli/domains/adoptions/.
type handlers struct {
	core   *cliapp.ScenarioApp
	client searchconnect.SearchServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: searchconnect.NewSearchServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) queryCall(ctx cliapp.OperationContext) (*searchv1.SearchResponse, error) {
	req := &searchv1.SearchRequest{
		Query: ctx.Positional("query"),
		Ask:   ctx.BoolFlag("ask"),
	}
	for _, raw := range ctx.FlagValues("kind") {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		kind, err := parseSearchKind(raw)
		if err != nil {
			return nil, err
		}
		req.Kinds = append(req.Kinds, kind)
	}
	limit, err := parseLimit(ctx.Flag("limit"))
	if err != nil {
		return nil, err
	}
	req.Limit = limit

	resp, err := h.client.Search(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("search", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no search response")
	}
	return resp.Msg, nil
}

func (h *handlers) queryReport(_ cliapp.OperationContext, msg *searchv1.SearchResponse) cliapp.ListReport {
	results := msg.GetResults()
	rows := make([]string, 0, len(results)+1)
	for i, result := range results {
		if result == nil {
			continue
		}
		line := fmt.Sprintf("%2d. [%s] %s", i+1, searchKindLabel(result.GetKind()), nonEmpty(result.GetTitle(), result.GetId()))
		if subtitle := strings.TrimSpace(result.GetSubtitle()); subtitle != "" {
			line += " — " + subtitle
		}
		if result.GetScore() != 0 {
			line += fmt.Sprintf(" (score %.3f)", result.GetScore())
		}
		if availability := availabilityLabel(result.GetAvailability()); availability != "" {
			line += " [" + availability + "]"
		}
		if href := strings.TrimSpace(result.GetHref()); href != "" {
			line += " → " + href
		}
		rows = append(rows, line)
	}
	rows = append(rows, semanticLegDetail(msg.GetStatus()))

	summary := fmt.Sprintf("Found %d result(s) for %q in %d ms (total matches: %d).",
		len(results), msg.GetQuery(), msg.GetTookMs(), msg.GetTotal())
	return cliapp.ListReport{
		Summary: []string{
			summary,
			describeSearchStatus(msg.GetStatus()),
		},
		ResultsHeading: "Ranked results",
		Results:        rows,
		RetrievalHints: []string{
			"`search suggest <prefix>` — complete a query from indexed titles and tags",
			"`search status` — confirm index freshness and the semantic-leg state",
		},
	}
}

func (h *handlers) suggestCall(ctx cliapp.OperationContext) (*searchv1.SuggestResponse, error) {
	req := &searchv1.SuggestRequest{Query: ctx.Positional("query")}
	limit, err := parseLimit(ctx.Flag("limit"))
	if err != nil {
		return nil, err
	}
	req.Limit = limit
	resp, err := h.client.Suggest(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("suggest search terms", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no suggestions")
	}
	return resp.Msg, nil
}

func (h *handlers) suggestReport(ctx cliapp.OperationContext, msg *searchv1.SuggestResponse) cliapp.ListReport {
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d suggestion(s) for %q.", len(msg.GetSuggestions()), ctx.Positional("query"))},
		ResultsHeading: "Suggestions",
		Results:        msg.GetSuggestions(),
		RetrievalHints: []string{"`search query <term>` — run the suggested query"},
	}
}

func (h *handlers) statusCall(_ cliapp.OperationContext) (*searchv1.SearchStatus, error) {
	resp, err := h.client.GetSearchStatus(context.Background(), connect.NewRequest(&searchv1.GetSearchStatusRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get search status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no search status")
	}
	return resp.Msg, nil
}

func (h *handlers) statusReport(_ cliapp.OperationContext, status *searchv1.SearchStatus) cliapp.ListReport {
	rows := []string{
		fmt.Sprintf("indexed: %t", status.GetIndexed()),
		fmt.Sprintf("documents: %d", status.GetDocumentCount()),
		fmt.Sprintf("indexed_at: %s", nonEmpty(status.GetIndexedAt(), "(never)")),
		fmt.Sprintf("stale: %t", status.GetStale()),
		fmt.Sprintf("semantic_leg: %s", semanticLegLabel(status.GetSemanticLeg())),
		semanticLegDetail(status),
	}
	if lastError := strings.TrimSpace(status.GetLastError()); lastError != "" {
		rows = append(rows, "last_error: "+lastError)
	}
	return cliapp.ListReport{
		Summary:        []string{"Search index status."},
		ResultsHeading: "Index",
		Results:        rows,
		RetrievalHints: []string{"`search reindex` — rebuild the lexical index from owned sources"},
	}
}

func (h *handlers) reindexCall(ctx cliapp.OperationContext) (*searchv1.ReindexResponse, error) {
	resp, err := h.client.Reindex(context.Background(), connect.NewRequest(&searchv1.ReindexRequest{Force: ctx.BoolFlag("force")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("reindex search", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no reindex response")
	}
	return resp.Msg, nil
}

func (h *handlers) reindexReport(ctx cliapp.OperationContext, msg *searchv1.ReindexResponse) cliapp.MutationReport {
	force := "false"
	if ctx.BoolFlag("force") {
		force = "true"
	}
	return cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Reindexed %d document(s) at %s.", msg.GetDocumentCount(), nonEmpty(msg.GetIndexedAt(), "(time unavailable)"))},
		Changes: []string{
			fmt.Sprintf("force=%s", force),
		},
		NextCommand: []string{"`search status` — confirm index freshness"},
	}
}

func describeSearchStatus(status *searchv1.SearchStatus) string {
	if status == nil {
		return "Index: status unavailable; semantic search is not active (results are lexical only)."
	}
	line := fmt.Sprintf("Index: indexed=%t documents=%d semantic-leg=%s",
		status.GetIndexed(), status.GetDocumentCount(), semanticLegLabel(status.GetSemanticLeg()))
	if status.GetStale() {
		line += " stale=true"
	}
	if lastError := strings.TrimSpace(status.GetLastError()); lastError != "" {
		line += " last-error=" + lastError
	}
	return line
}

func semanticLegDetail(status *searchv1.SearchStatus) string {
	if status == nil {
		return "Semantic leg: unknown — results are lexical only."
	}
	switch status.GetSemanticLeg() {
	case searchv1.SemanticLegState_SEMANTIC_LEG_STATE_READY:
		return "Semantic leg: ready."
	case searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DEGRADED:
		return "Semantic leg: degraded — lexical results only; the semantic leg did not answer."
	case searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DISABLED:
		return "Semantic leg: disabled — results are lexical only."
	default:
		return "Semantic leg: unspecified — results are lexical only."
	}
}

func semanticLegLabel(value searchv1.SemanticLegState) string {
	switch value {
	case searchv1.SemanticLegState_SEMANTIC_LEG_STATE_READY:
		return "ready"
	case searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DEGRADED:
		return "degraded"
	case searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DISABLED:
		return "disabled"
	default:
		return "unspecified"
	}
}

func searchKindLabel(value searchv1.SearchKind) string {
	switch value {
	case searchv1.SearchKind_SEARCH_KIND_WORKSPACE:
		return "workspace"
	case searchv1.SearchKind_SEARCH_KIND_PAGE:
		return "page"
	case searchv1.SearchKind_SEARCH_KIND_DISPLAY:
		return "display"
	case searchv1.SearchKind_SEARCH_KIND_STATE:
		return "state"
	case searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE:
		return "page-type"
	case searchv1.SearchKind_SEARCH_KIND_ASSET:
		return "asset"
	case searchv1.SearchKind_SEARCH_KIND_COMMAND:
		return "command"
	case searchv1.SearchKind_SEARCH_KIND_EVIDENCE:
		return "evidence"
	default:
		return "unspecified"
	}
}

func parseSearchKind(raw string) (searchv1.SearchKind, error) {
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(raw)), "_", "-")
	normalized = strings.TrimPrefix(normalized, "search-kind-")
	switch normalized {
	case "workspace":
		return searchv1.SearchKind_SEARCH_KIND_WORKSPACE, nil
	case "page":
		return searchv1.SearchKind_SEARCH_KIND_PAGE, nil
	case "display":
		return searchv1.SearchKind_SEARCH_KIND_DISPLAY, nil
	case "state":
		return searchv1.SearchKind_SEARCH_KIND_STATE, nil
	case "page-type":
		return searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE, nil
	case "asset":
		return searchv1.SearchKind_SEARCH_KIND_ASSET, nil
	case "command":
		return searchv1.SearchKind_SEARCH_KIND_COMMAND, nil
	case "evidence":
		return searchv1.SearchKind_SEARCH_KIND_EVIDENCE, nil
	case "", "unspecified":
		return searchv1.SearchKind_SEARCH_KIND_UNSPECIFIED, nil
	default:
		return 0, fmt.Errorf("--kind must be one of workspace, page, display, state, page-type, asset, command, evidence (got %q)", raw)
	}
}

func availabilityLabel(value searchv1.SearchAvailability) string {
	switch value {
	case searchv1.SearchAvailability_SEARCH_AVAILABILITY_AVAILABLE:
		return "available"
	case searchv1.SearchAvailability_SEARCH_AVAILABILITY_DRAFT:
		return "draft"
	case searchv1.SearchAvailability_SEARCH_AVAILABILITY_PLANNED:
		return "planned"
	case searchv1.SearchAvailability_SEARCH_AVAILABILITY_UNAVAILABLE:
		return "unavailable"
	default:
		return ""
	}
}

func parseLimit(raw string) (int32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("--limit must be a non-negative integer (got %q)", raw)
	}
	return int32(value), nil
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
