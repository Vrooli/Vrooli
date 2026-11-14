package intents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing/routing_v1connect"
)

const queryTimeout = 60 * time.Second

type handlers struct {
	core *cliapp.ScenarioApp
}

func newHandlers(core *cliapp.ScenarioApp) *handlers { return &handlers{core: core} }

// queryFunc runs one public query; swapped by tests.
type queryFunc func(ctx context.Context, req *routingv1.QueryRequest) (*routingv1.QueryResponse, error)

func (h *handlers) liveQuery() queryFunc {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(h.core, queryTimeout)
	client := routingconnect.NewRoutingServiceClient(httpClient, baseURL)
	return func(ctx context.Context, req *routingv1.QueryRequest) (*routingv1.QueryResponse, error) {
		resp, err := client.Query(ctx, connect.NewRequest(req))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	}
}

// resolveRoot finds the repository root: --root, then VROOLI_ROOT, then the
// nearest ancestor of the working directory that holds scenarios/.
func resolveRoot(flag string) (string, error) {
	candidates := []string{strings.TrimSpace(flag), strings.TrimSpace(os.Getenv("VROOLI_ROOT"))}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd)
	}
	for _, start := range candidates {
		for dir := start; dir != "" && dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			if info, err := os.Stat(filepath.Join(dir, "scenarios")); err == nil && info.IsDir() {
				return dir, nil
			}
		}
	}
	return "", fmt.Errorf("could not find the repository root; pass --root <path> or set VROOLI_ROOT")
}

// runResult is the complete `intents run` report.
type runResult struct {
	Root        string           `json:"root"`
	Types       string           `json:"types"`
	Limit       int              `json:"limit"`
	Cases       int              `json:"cases"`
	Passed      int              `json:"passed"`
	AnyLeaf     int              `json:"any_leaf"`
	Unavailable int              `json:"unavailable"`
	Recall      float64          `json:"recall"`
	AnyLeafRate float64          `json:"any_leaf_rate"`
	Scenarios   []scenarioRecall `json:"scenarios"`
	Results     []caseResult     `json:"results"`
}

func (h *handlers) runCall(ctx cliapp.OperationContext) (runResult, error) {
	return h.run(ctx, h.liveQuery())
}

func (h *handlers) run(ctx cliapp.OperationContext, query queryFunc) (runResult, error) {
	root, err := resolveRoot(ctx.Flag("root"))
	if err != nil {
		return runResult{}, err
	}
	sources, err := corpusSources(root)
	if err != nil {
		return runResult{}, err
	}
	if corpus := strings.TrimSpace(ctx.Flag("corpus")); corpus != "" {
		sources = []string{corpus}
	}
	cases, err := loadCorpus(sources)
	if err != nil {
		return runResult{}, err
	}
	if only := strings.TrimSpace(ctx.Flag("scenario")); only != "" {
		kept := cases[:0:0]
		for _, c := range cases {
			if c.Scenario == only {
				kept = append(kept, c)
			}
		}
		cases = kept
	}
	if len(cases) == 0 {
		return runResult{}, fmt.Errorf("no intent cases found (looked in %d corpus file(s); --scenario %q)", len(sources), ctx.Flag("scenario"))
	}
	limit := DefaultLimit
	if raw := strings.TrimSpace(ctx.Flag("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return runResult{}, fmt.Errorf("--limit must be a positive integer, got %q", raw)
		}
		limit = n
	}
	types := strings.TrimSpace(ctx.Flag("types"))
	if types == "" {
		types = DefaultTypes
	}
	results := runCases(cases, types, limit, query)
	return summarize(root, types, limit, results), nil
}

// maxConsecutiveDegraded stops a run that is only feeding a failing provider.
// Each query fans out to live providers, and the router opens a provider's
// circuit after a streak of failures; a measurement must never be the load that
// takes a corpus out of service for everyone else.
const maxConsecutiveDegraded = 3

// runCases evaluates every case through the public query path, in corpus order
// and one at a time (see maxConsecutiveDegraded). After a streak of degraded
// answers the remaining cases are reported unavailable rather than attempted.
func runCases(cases []loadedCase, defaultTypes string, limit int, query queryFunc) []caseResult {
	results := make([]caseResult, len(cases))
	streak := 0
	for i, c := range cases {
		if streak >= maxConsecutiveDegraded {
			results[i] = caseResult{ID: c.ID, Scenario: c.Scenario, Query: c.Query, Tags: c.Tags,
				Unavailable: fmt.Sprintf("skipped: %d consecutive degraded answers; retry when the federation recovers (search-hub federation status)", streak)}
			continue
		}
		results[i] = runCase(c, defaultTypes, limit, query)
		if results[i].Unavailable != "" {
			streak++
		} else {
			streak = 0
		}
	}
	return results
}

func runCase(c loadedCase, defaultTypes string, limit int, query queryFunc) caseResult {
	types := defaultTypes
	if c.Types != "" {
		types = c.Types
	}
	resp, err := query(context.Background(), &routingv1.QueryRequest{Query: c.Query, Types: splitList(types), Limit: int32(limit)})
	if err != nil {
		return caseResult{ID: c.ID, Scenario: c.Scenario, Query: c.Query, Tags: c.Tags, Unavailable: err.Error()}
	}
	result := grade(c, orderedHits(resp), limit)
	// A pass stands, but a miss from a degraded answer proves nothing: the
	// provider that holds the leaf may be the one that failed. Report it as
	// unavailable so an outage can never masquerade as a discoverability defect.
	if !result.Pass {
		if reason := degradedReason(resp); reason != "" {
			result.Unavailable = "degraded answer: " + reason
		}
	}
	return result
}

// degradedReason summarizes the degraded providers of a response, or "" when
// the answer was complete.
func degradedReason(resp *routingv1.QueryResponse) string {
	var notes []string
	for _, g := range resp.GetGroups() {
		if g.GetDegraded() {
			notes = append(notes, g.GetProviderId())
		}
	}
	if len(notes) == 0 && resp.GetPartial() {
		return "partial results (query deadline)"
	}
	return strings.Join(notes, ", ")
}

func summarize(root, types string, limit int, results []caseResult) runResult {
	out := runResult{Root: root, Types: types, Limit: limit, Results: results, Scenarios: rollUp(results)}
	for _, r := range results {
		if r.Unavailable != "" {
			out.Unavailable++
			continue
		}
		out.Cases++
		if r.Pass {
			out.Passed++
		}
		if r.AnyLeafRank > 0 {
			out.AnyLeaf++
		}
	}
	if out.Cases > 0 {
		out.Recall = float64(out.Passed) / float64(out.Cases)
		out.AnyLeafRate = float64(out.AnyLeaf) / float64(out.Cases)
	}
	return out
}

func (h *handlers) runReport(_ cliapp.OperationContext, r runResult) cliapp.MutationReport {
	result := []string{fmt.Sprintf("Intent recall@%d over --type %s: %d/%d cases surfaced a qualifying leaf of the owning scenario (%.0f%%); %d/%d surfaced any leaf of it; %d unavailable.",
		r.Limit, r.Types, r.Passed, r.Cases, r.Recall*100, r.AnyLeaf, r.Cases, r.Unavailable)}
	for _, s := range r.Scenarios {
		result = append(result, fmt.Sprintf("  %s: %d/%d qualifying, %d/%d any leaf", s.Scenario, s.Passed, s.Cases, s.AnyLeaf, s.Cases))
	}
	changes := make([]string, 0, len(r.Results))
	for _, c := range r.Results {
		switch {
		case c.Unavailable != "":
			changes = append(changes, fmt.Sprintf("UNAVAILABLE %s/%s: %s", c.Scenario, c.ID, c.Unavailable))
		case c.Pass:
			changes = append(changes, fmt.Sprintf("pass  %s/%s rank=%d  %q", c.Scenario, c.ID, c.Rank, c.Query))
		default:
			changes = append(changes, fmt.Sprintf("MISS  %s/%s any-leaf-rank=%d  %q  top=%s", c.Scenario, c.ID, c.AnyLeafRank, c.Query, strings.Join(c.Top, " ")))
		}
	}
	return cliapp.MutationReport{
		Result:      result,
		Changes:     changes,
		NextCommand: []string{"`search-hub intents coverage` — scenarios that expose leaves but declare no intent", "fix a MISS in the owning scenario's skill/program/command descriptions, then re-run"},
	}
}

func (h *handlers) runExit(ctx cliapp.OperationContext, r runResult) error {
	if ctx.BoolFlag("assert") && r.Passed < r.Cases {
		return fmt.Errorf("intent recall %d/%d below 100%% (--assert)", r.Passed, r.Cases)
	}
	return nil
}

// coverageResult is the `intents coverage` report.
type coverageResult struct {
	Root      string           `json:"root"`
	Scenarios []scenarioLeaves `json:"scenarios"`
	Uncovered []scenarioLeaves `json:"uncovered"`
	Corpora   []string         `json:"corpora"`
}

func (h *handlers) coverageCall(ctx cliapp.OperationContext) (coverageResult, error) {
	root, err := resolveRoot(ctx.Flag("root"))
	if err != nil {
		return coverageResult{}, err
	}
	sources, err := corpusSources(root)
	if err != nil {
		return coverageResult{}, err
	}
	cases, err := loadCorpus(sources)
	if err != nil {
		return coverageResult{}, err
	}
	leaves, err := inventoryLeaves(root)
	if err != nil {
		return coverageResult{}, err
	}
	attachIntents(leaves, cases)
	return coverageResult{Root: root, Scenarios: leaves, Uncovered: uncovered(leaves), Corpora: sources}, nil
}

func (h *handlers) coverageReport(_ cliapp.OperationContext, r coverageResult) cliapp.MutationReport {
	covered := len(r.Scenarios) - len(r.Uncovered)
	result := []string{fmt.Sprintf("%d of %d scenarios with discoverable leaves declare at least one intent; %d do not.", covered, len(r.Scenarios), len(r.Uncovered))}
	changes := make([]string, 0, len(r.Uncovered))
	for _, s := range r.Uncovered {
		changes = append(changes, fmt.Sprintf("no intent: %s (skills=%d programs=%d commands=%d)", s.Scenario, s.Skills, s.Programs, s.Commands))
	}
	sort.Strings(r.Corpora)
	return cliapp.MutationReport{
		Result:      result,
		Changes:     changes,
		NextCommand: []string{"add cases to scenarios/search-hub/intents/<scenario>.json or scenarios/<scenario>/.vrooli/intents.json, then `search-hub intents run --scenario <name>`"},
	}
}

func splitList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
