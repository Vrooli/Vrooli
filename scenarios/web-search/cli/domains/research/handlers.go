package research

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	researchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/research"
	researchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/research/research_v1connect"

	"github.com/vrooli/cli-core/cliapp"

	"web-search/cli/internal/cliutil"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client researchconnect.ResearchServiceClient
}

// researchClientTimeout bounds research RPCs. RunL2 is synchronous and
// legitimately slow — sequential page fetches plus an LLM synthesis whose
// model may cold-load on a CPU-resident ollama — so the scenario default 30s
// client would abort the call while the server is still working (and cancel
// the server's in-flight synthesis with it). Matches the server's
// WriteTimeout budget.
const researchClientTimeout = 5 * time.Minute

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, researchClientTimeout)
	return &handlers{
		core:   core,
		client: researchconnect.NewResearchServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) l2(ctx cliapp.RunContext) error {
	query := ctx.Positional("query")
	resp, err := h.client.RunL2(context.Background(), connect.NewRequest(&researchv1.RunL2Request{
		Query:   query,
		TopN:    cliutil.ParseInt32(ctx.Flag("top-n")),
		Capture: ctx.BoolFlag("capture"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("run L2 research", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no L2 response")
	}
	msg := resp.Msg
	summary := []string{fmt.Sprintf("L2 research for %q.", query)}
	if msg.Abstained {
		reason := strings.TrimSpace(msg.AbstainReason)
		if reason == "" {
			reason = "sources insufficient or disagree"
		}
		summary = append(summary, "Synthesis abstained: "+reason+".")
	} else {
		summary = append(summary, "Synthesis: "+strings.TrimSpace(msg.Synthesis))
	}
	if len(msg.CapturedFindingIds) > 0 {
		summary = append(summary, fmt.Sprintf("Captured %d finding(s): %s", len(msg.CapturedFindingIds), strings.Join(msg.CapturedFindingIds, ", ")))
	}
	if len(msg.DegradedEngines) > 0 {
		parts := make([]string, 0, len(msg.DegradedEngines))
		for _, issue := range msg.DegradedEngines {
			if issue == nil {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s: %s", issue.Engine, issue.Reason))
		}
		if len(parts) > 0 {
			summary = append(summary, fmt.Sprintf("⚠ %d engine(s) unavailable during candidate search (%s) — inputs may be partial.", len(parts), strings.Join(parts, "; ")))
		}
	}
	results := make([]string, 0)
	if msg.Brief != nil {
		for _, c := range msg.Brief.Citations {
			results = append(results, fmt.Sprintf("[%d] %s — %s", c.ResultIndex, c.Title, c.Url))
		}
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Citations",
		Results:        results,
		RetrievalHints: []string{
			"`research l2 <query> --capture` — persist the synthesis as a finding",
			"`research l3 <query>` — start an iterative research-and-reconcile run",
		},
	})
}

func (h *handlers) l3(ctx cliapp.RunContext) error {
	query := ctx.Positional("query")
	resp, err := h.client.RunL3(context.Background(), connect.NewRequest(&researchv1.RunL3Request{Query: query}))
	if err != nil {
		return cliapp.WrapAPIError("start L3 research", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no L3 response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Started L3 run %s (status: %s).", resp.Msg.RunId, resp.Msg.Status)},
		NextCommand: []string{fmt.Sprintf("`research status %s` — poll this run", resp.Msg.RunId)},
	})
}

func (h *handlers) status(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetResearchStatus(context.Background(), connect.NewRequest(&researchv1.GetResearchStatusRequest{RunId: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get research status %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no status response")
	}
	msg := resp.Msg
	summary := []string{fmt.Sprintf("Run %s status: %s.", msg.RunId, msg.Status)}
	if strings.TrimSpace(msg.Summary) != "" {
		summary = append(summary, "Summary: "+strings.TrimSpace(msg.Summary))
	}
	if strings.TrimSpace(msg.ErrorMsg) != "" {
		summary = append(summary, "Error: "+strings.TrimSpace(msg.ErrorMsg))
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Run",
		Results:        []string{fmt.Sprintf("%s [%s]", msg.RunId, msg.Status)},
	})
}

func (h *handlers) gather(ctx cliapp.RunContext) error {
	query := ctx.Positional("query")
	resp, err := h.client.GatherRelatedFindings(context.Background(), connect.NewRequest(&researchv1.GatherRelatedFindingsRequest{
		Query: query,
		Max:   cliutil.ParseInt32(ctx.Flag("max")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("gather related findings", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no gather response")
	}
	msg := resp.Msg
	summary := []string{fmt.Sprintf("Gathered %d finding(s) near %q (bounded sweep, cap %d).", len(msg.Findings), query, msg.CapApplied)}
	results := make([]string, 0, len(msg.Findings))
	for _, g := range msg.Findings {
		results = append(results, fmt.Sprintf("%s — %s [status=%s confidence=%.2f score=%.3f]", g.FindingId, g.Claim, g.Status, g.Confidence, g.Score))
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Nearby Findings",
		Results:        results,
		RetrievalHints: []string{
			"`research l2 <focused sub-query>` — research a gap the nearby findings don't cover",
			"`findings supersede <old-id> --replacement <new-id>` — reconcile after answering",
		},
	})
}
