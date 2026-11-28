package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/search"
	searchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/search/search_v1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

type handlers struct {
	core            *cliapp.ScenarioApp
	client          searchconnect.SearchServiceClient
	compositionHTTP connect.HTTPClient
	compositionURL  string
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core: core, client: searchconnect.NewSearchServiceClient(httpClient, baseURL),
		compositionHTTP: httpClient, compositionURL: strings.TrimRight(baseURL, "/"),
	}
}

// compositionSearchCall queries the bounded CLI source-composition endpoint.
func (h *handlers) compositionSearchCall(ctx cliapp.OperationContext) (*structpb.Struct, error) {
	query := strings.TrimSpace(ctx.Positional("query"))
	scenario := strings.TrimSpace(ctx.Positional("scenario"))
	if query == "" || scenario == "" {
		return nil, fmt.Errorf("query and scenario are required")
	}
	body, err := json.Marshal(map[string]string{"query": query, "scenario": scenario})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, h.compositionURL+"/api/v1/cli-composition/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.compositionHTTP.Do(req)
	if err != nil {
		return nil, cliapp.WrapAPIError("search CLI composition", err, nil)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("search CLI composition returned HTTP %d", resp.StatusCode)
	}
	var value map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&value); err != nil {
		return nil, fmt.Errorf("decode CLI composition response: %w", err)
	}
	return structpb.NewStruct(value)
}

func (h *handlers) compositionSearchReport(ctx cliapp.OperationContext, response *structpb.Struct) cliapp.ListReport {
	body := response.AsMap()
	raw, _ := body["results"].([]any)
	results := make([]string, 0, len(raw))
	for _, item := range raw {
		hit, _ := item.(map[string]any)
		results = append(results, fmt.Sprintf("%v — %v (%v; %v)", hit["title"], hit["text"], hit["attestation"], hit["evidence"]))
	}
	degradations, _ := body["degradations"].([]any)
	for _, item := range degradations {
		d, _ := item.(map[string]any)
		results = append(results, fmt.Sprintf("degraded %v: %v", d["code"], d["message"]))
	}
	return cliapp.ListReport{Summary: []string{compositionSearchSummary(ctx.Positional("query"), ctx.Positional("scenario"), body, len(raw))}, ResultsHeading: "Composition evidence", Results: results}
}

func compositionSearchSummary(query, requestedScenario string, response map[string]any, resultCount int) string {
	query = strings.TrimSpace(query)
	requestedScenario = strings.TrimSpace(requestedScenario)
	selectedScenario, _ := response["scenario"].(string)
	scope := "requested scenario " + strconv.Quote(requestedScenario) + "; selected source scope unavailable"
	if selectedScenario != "" {
		scope = "scenario:" + selectedScenario
	}
	return fmt.Sprintf("Static CLI composition for query %s (%s): found %d result(s); runtime dispatch remains unverified.", strconv.Quote(query), scope, resultCount)
}

// searchCall runs the SearchService.Search RPC. It is the operation half of the
// proto_list primitive: it never sees the output format, so it cannot branch on
// --json. searchReport supplies the human rendering.
func (h *handlers) searchCall(ctx cliapp.OperationContext) (*searchv1.SearchResponse, error) {
	text := ctx.Positional("text")
	limit := int32(10)
	if v := strings.TrimSpace(ctx.Flag("limit")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}
	mode := parseMode(ctx.Flag("mode"))

	resp, err := h.client.Search(context.Background(), connect.NewRequest(&searchv1.SearchRequest{
		Query: text,
		Limit: limit,
		Mode:  mode,
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("search %q", text), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no search response")
	}
	return resp.Msg, nil
}

// searchReport maps the search response to the human ListReport (the render half
// of the proto_list primitive). --json consumers get the proto wire shape.
func (h *handlers) searchReport(_ cliapp.OperationContext, msg *searchv1.SearchResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Results))
	for i, r := range msg.Results {
		// Prefer the server's canonical full_path; fall back to assembling the
		// pieces for any result that predates the field.
		full := strings.TrimSpace(r.FullPath)
		if full == "" {
			full = strings.TrimSpace(strings.Join([]string{r.Origin, r.Group, r.Name}, " "))
		}
		desc := truncate(r.Description, 80)
		// WS2: the server computes the regime-aware weak flag once; the CLI just
		// renders it (no client-side threshold).
		weak := ""
		if r.Weak {
			weak = " (weak)"
		}
		results = append(results, fmt.Sprintf("%d. %s — %s [score=%.3f source=%s]%s",
			i+1, full, desc, r.Score, r.Source, weak))
	}
	if len(results) == 0 {
		results = append(results, "(no matches)")
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d result(s) (mode=%s).", len(msg.Results), msg.ModeUsed.String())},
		ResultsHeading: "Results",
		Results:        results,
	}
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// statusCall runs the SearchService.Status RPC (operation half of the
// operational primitive).
func (h *handlers) statusCall(ctx cliapp.OperationContext) (*searchv1.StatusResponse, error) {
	resp, err := h.client.Status(context.Background(), connect.NewRequest(&searchv1.StatusRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("search status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no status response")
	}
	return resp.Msg, nil
}

// statusReport renders backend availability through the operational contract
// (Status -> Triage -> Next Steps); --json consumers get the proto wire shape.
func (h *handlers) statusReport(_ cliapp.OperationContext, msg *searchv1.StatusResponse) cliapp.OperationalReport {
	reranker := msg.Reranker
	if reranker == "" {
		reranker = "none"
	}
	return cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Available: %v", msg.Available),
			fmt.Sprintf("Ollama: %v  Qdrant: %v  Indexed: %d", msg.Ollama, msg.Qdrant, msg.IndexedCount),
			fmt.Sprintf("Reranker: %s", reranker),
		},
	}
}

// parseMode maps the CLI --mode flag to the proto enum. Empty / unknown
// values fall back to UNSPECIFIED (let the server choose).
func parseMode(s string) searchv1.Mode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "ai":
		return searchv1.Mode_MODE_AI
	case "text":
		return searchv1.Mode_MODE_TEXT
	default:
		return searchv1.Mode_MODE_UNSPECIFIED
	}
}
