package research

import (
	"context"
	"fmt"
	"strconv"
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

func parseFloatOrZero(v string) float64 {
	if v == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(v, 64)
	return f
}

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
		Query:       query,
		TopN:        cliutil.ParseInt32(ctx.Flag("top-n")),
		Capture:     ctx.BoolFlag("capture"),
		ParentRunId: ctx.Flag("parent-run-id"),
		Policy:      &researchv1.EvidencePolicy{MaxEvidenceBytes: cliutil.ParseInt32(ctx.Flag("max-evidence-bytes"))},
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
	resp, err := h.client.RunL3(context.Background(), connect.NewRequest(&researchv1.RunL3Request{
		Query: query, IdempotencyKey: ctx.Flag("idempotency-key"),
		Policy: &researchv1.EvidencePolicy{MaxEvidenceBytes: cliutil.ParseInt32(ctx.Flag("max-evidence-bytes"))},
	}))
	if err != nil {
		return cliapp.WrapAPIError("start L3 research", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no L3 response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Started L3 run %s (status: %s).", resp.Msg.RunId, resp.Msg.Status)},
		NextCommand: []string{fmt.Sprintf("`research wait %s` — attach once to this execution", resp.Msg.RunId)},
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

func (h *handlers) answer(ctx cliapp.RunContext) error {
	age := int64(0)
	if raw := ctx.Flag("max-age-seconds"); raw != "" {
		v, e := strconv.ParseInt(raw, 10, 64)
		if e != nil {
			return fmt.Errorf("invalid max-age-seconds: %w", e)
		}
		age = v
	}
	var domains []string
	if raw := ctx.Flag("source-domains"); raw != "" {
		domains = strings.Split(raw, ",")
	}
	resp, err := h.client.Answer(context.Background(), connect.NewRequest(&researchv1.AnswerRequest{Query: ctx.Positional("query"), Effort: ctx.Flag("effort"), MaxAgeSeconds: age, SourceDomains: domains, MinimumSources: cliutil.ParseInt32(ctx.Flag("minimum-sources")), TopN: cliutil.ParseInt32(ctx.Flag("top-n")), Capture: ctx.BoolFlag("capture"), FindingId: ctx.Flag("finding-id"), ParentRunId: ctx.Flag("parent-run-id"), Policy: &researchv1.EvidencePolicy{MaxEvidenceBytes: cliutil.ParseInt32(ctx.Flag("max-evidence-bytes"))}}))
	if err != nil {
		return cliapp.WrapAPIError("answer research", err, nil)
	}
	m := resp.Msg
	results := []string{}
	if m.Brief != nil {
		for _, c := range m.Brief.Citations {
			results = append(results, c.Title+" — "+c.Url)
		}
	}
	for _, r := range m.Results {
		results = append(results, r.Title+" — "+r.Url)
	}
	return cliapp.RenderProtoList(ctx, m, cliapp.ListReport{Summary: []string{m.Status + ": " + m.AnswerKind, m.GetBrief().GetSummary(), m.Reason}, ResultsHeading: "Sources", Results: results})
}

func (h *handlers) wait(ctx cliapp.RunContext) error {
	n := int32(90)
	if ctx.Flag("timeout-seconds") != "" {
		n = cliutil.ParseInt32(ctx.Flag("timeout-seconds"))
	}
	resp, err := h.client.WaitResearch(context.Background(), connect.NewRequest(&researchv1.WaitResearchRequest{RunId: ctx.Positional("id"), TimeoutSeconds: n}))
	if err != nil {
		return cliapp.WrapAPIError("wait for research", err, nil)
	}
	m := resp.Msg
	err = cliapp.RenderProtoList(ctx, m, cliapp.ListReport{Summary: []string{"Research " + m.RunId + ": " + m.Status, m.Summary, m.ErrorMsg}})
	if err != nil {
		return err
	}
	if m.TimedOut {
		return fmt.Errorf("wait timed out; execution continues; reattach to research wait %s", m.RunId)
	}
	if m.Status != "complete" {
		return fmt.Errorf("research execution ended as %s", m.Status)
	}
	return nil
}

func (h *handlers) cancel(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.CancelResearch(context.Background(), connect.NewRequest(&researchv1.CancelResearchRequest{
		RunId: id, Reason: ctx.Flag("reason"), IdempotencyKey: ctx.Flag("idempotency-key"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("cancel research", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no cancellation response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Cancellation requested for research run %s (status: %s).", resp.Msg.RunId, resp.Msg.Status)},
		NextCommand: []string{fmt.Sprintf("`research status %s` — inspect the owner terminal state", resp.Msg.RunId)},
	})
}

func (h *handlers) captureStatus(ctx cliapp.RunContext) error {
	resp, err := h.client.GetCaptureStatus(context.Background(), connect.NewRequest(&researchv1.GetCaptureStatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get research capture status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no capture status")
	}
	m := resp.Msg
	summary := []string{fmt.Sprintf("Research capture status: %s.", m.Status), fmt.Sprintf("Pending: %d; delivered: %d; failed: %d.", m.Pending, m.Delivered, m.Failed)}
	if m.OldestPendingAt != "" {
		summary = append(summary, "Oldest pending: "+m.OldestPendingAt)
	}
	return cliapp.RenderProtoList(ctx, m, cliapp.ListReport{Summary: summary, ResultsHeading: "Capture delivery", Results: []string{m.Status}})
}

func (h *handlers) evidenceReceipt(ctx cliapp.RunContext) error {
	resp, err := h.client.GetEvidenceReceipt(context.Background(), connect.NewRequest(&researchv1.GetEvidenceReceiptRequest{ReceiptId: ctx.Positional("receipt-id")}))
	if err != nil {
		return cliapp.WrapAPIError("get evidence receipt", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no evidence receipt")
	}
	m := resp.Msg
	return cliapp.RenderProtoList(ctx, m, cliapp.ListReport{Summary: []string{"Evidence receipt " + m.ReceiptId + ".", "Retrieved: " + m.RetrievedAt, "URL: " + m.OriginalUrl}, ResultsHeading: "Artifact", Results: []string{m.ArtifactId + " (" + m.ContentSha256 + ")"}})
}

func (h *handlers) evidencePassage(ctx cliapp.RunContext) error {
	resp, err := h.client.GetEvidencePassage(context.Background(), connect.NewRequest(&researchv1.GetEvidencePassageRequest{PassageId: ctx.Positional("passage-id")}))
	if err != nil {
		return cliapp.WrapAPIError("get evidence passage", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no evidence passage")
	}
	m := resp.Msg
	return cliapp.RenderProtoList(ctx, m, cliapp.ListReport{Summary: []string{fmt.Sprintf("Evidence passage %s (%d..%d).", m.PassageId, m.StartByte, m.EndByte), "Receipt: " + m.ReceiptId}, ResultsHeading: "Passage", Results: []string{m.Content}})
}

func (h *handlers) evidenceAssessment(ctx cliapp.RunContext) error {
	resp, err := h.client.GetEvidenceAssessment(context.Background(), connect.NewRequest(&researchv1.GetEvidenceAssessmentRequest{AssessmentId: ctx.Positional("assessment-id")}))
	if err != nil {
		return cliapp.WrapAPIError("get evidence assessment", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no evidence assessment")
	}
	m := resp.Msg
	return cliapp.RenderProtoList(ctx, m, cliapp.ListReport{Summary: []string{"Evidence assessment " + m.AssessmentId + ".", "Claim: " + m.ClaimId, "Disposition: " + m.Disposition}, ResultsHeading: "Assessment", Results: []string{m.Reason, m.EvidenceJson}})
}

func (h *handlers) methodCurrent(ctx cliapp.RunContext) error {
	resp, err := h.client.GetMethodRelease(context.Background(), connect.NewRequest(&researchv1.GetMethodReleaseRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get current method release", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no method release")
	}
	summary := []string{"No current method release."}
	if resp.Msg.Found && resp.Msg.Release != nil {
		summary = []string{"Current method release: " + resp.Msg.Release.RevisionHash + "."}
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: summary, ResultsHeading: "Release", Results: nil})
}

func (h *handlers) methodPromote(ctx cliapp.RunContext) error {
	resp, err := h.client.PromoteMethod(context.Background(), connect.NewRequest(&researchv1.PromoteMethodRequest{
		ExpectedCurrentHash: ctx.Flag("expected-current-hash"), Grant: ctx.Flag("grant"),
		Revision: &researchv1.MethodRevision{Id: ctx.Flag("revision-id"), ProgramHash: ctx.Flag("program-hash"), ConfigHash: ctx.Flag("config-hash")},
		Receipt:  &researchv1.EvaluationReceipt{Id: ctx.Flag("receipt-id"), CandidateHash: ctx.Flag("candidate-hash"), ReportHash: ctx.Flag("report-hash"), Accepted: ctx.BoolFlag("receipt-accepted")},
		Report:   &researchv1.EvaluationReport{Accepted: ctx.BoolFlag("report-accepted"), Reason: ctx.Flag("report-reason"), BaselineMethod: ctx.Flag("baseline-method"), CandidateMethod: ctx.Flag("candidate-method"), Population: cliutil.ParseInt32(ctx.Flag("population")), MeanBaselineEffort: parseFloatOrZero(ctx.Flag("baseline-effort")), MeanCandidateEffort: parseFloatOrZero(ctx.Flag("candidate-effort"))},
	}))
	if err != nil {
		return cliapp.WrapAPIError("promote research method", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Release == nil {
		return fmt.Errorf("server returned no promoted release")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{"Promoted method " + resp.Msg.Release.RevisionHash + "."}})
}

func (h *handlers) methodRollback(ctx cliapp.RunContext) error {
	resp, err := h.client.RollbackMethod(context.Background(), connect.NewRequest(&researchv1.RollbackMethodRequest{ExpectedCurrentHash: ctx.Flag("expected-current-hash"), TargetHash: ctx.Flag("target-hash")}))
	if err != nil {
		return cliapp.WrapAPIError("rollback research method", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Release == nil {
		return fmt.Errorf("server returned no rolled-back release")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{"Rolled back method to " + resp.Msg.Release.RevisionHash + "."}})
}

func (h *handlers) methodSuspend(ctx cliapp.RunContext) error {
	resp, err := h.client.SuspendMethod(context.Background(), connect.NewRequest(&researchv1.SuspendMethodRequest{RevisionHash: ctx.Flag("revision-hash"), Reason: ctx.Flag("reason")}))
	if err != nil {
		return cliapp.WrapAPIError("suspend research method", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no suspension")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{"Suspended method " + resp.Msg.RevisionHash + "."}})
}
