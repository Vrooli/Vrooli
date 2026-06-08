package evals

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	evalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval/eval_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// runSuiteTimeout bounds a single `evals run` invocation. It is generous because
// the LLM reranker leg scores each case's shortlist with a local model; the
// cross-encoder and rerank-off legs finish in well under a second.
const runSuiteTimeout = 10 * time.Minute

// sweepTimeout bounds a full `evals sweep`: it runs the suite once per arm and,
// on the index-time tier, reindexes the provider per arm — so it can run for many
// minutes. Far longer than runSuiteTimeout.
const sweepTimeout = 45 * time.Minute

// generateTimeout bounds `evals generate`: it samples the provider's index and
// invokes the local LLM once per sampled item (query inversion + negatives), so
// it can run for several minutes on a large --count.
const generateTimeout = 20 * time.Minute

// handlers bundles the closure over *cliapp.ScenarioApp so each RunCtx-func has
// typed access to the EvalService client without re-resolving it.
type handlers struct {
	core   *cliapp.ScenarioApp
	client evalconnect.EvalServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: evalconnect.NewEvalServiceClient(httpClient, baseURL),
	}
}

// register reads an EvalSuite from --suite (raw JSON, or @path) and upserts it.
func (h *handlers) register(ctx cliapp.RunContext) error {
	blob, err := readSuiteArg(ctx.Flag("suite"))
	if err != nil {
		return err
	}
	suite := &evalv1.EvalSuite{}
	if err := protojson.Unmarshal(blob, suite); err != nil {
		return fmt.Errorf("parse --suite JSON: %w", err)
	}

	resp, err := h.client.RegisterSuite(context.Background(), connect.NewRequest(&evalv1.RegisterSuiteRequest{Suite: suite}))
	if err != nil {
		return cliapp.WrapAPIError("register suite", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetSuite() == nil {
		return fmt.Errorf("server returned no suite")
	}
	verb := "Updated"
	if resp.Msg.GetCreated() {
		verb = "Registered"
	}
	s := resp.Msg.GetSuite()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("%s eval suite %s (%d case(s)).", verb, s.GetSuiteId(), len(s.GetCases()))},
		Changes: []string{formatSuite(s)},
		NextCommand: []string{
			fmt.Sprintf("`evals run %s --tag baseline` — run it and store a tagged baseline", s.GetSuiteId()),
			fmt.Sprintf("`evals runs %s` — show this suite's run history", s.GetSuiteId()),
		},
	})
}

// list returns registered suites, optionally filtered by --provider.
func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListSuites(context.Background(), connect.NewRequest(&evalv1.ListSuitesRequest{
		ProviderId: strings.TrimSpace(ctx.Flag("provider")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list suites", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no suites response")
	}
	results := make([]string, 0, len(resp.Msg.GetSuites()))
	for _, s := range resp.Msg.GetSuites() {
		results = append(results, formatSuite(s))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d eval suite(s).", len(resp.Msg.GetSuites()))},
		ResultsHeading: "Suites",
		Results:        results,
		RetrievalHints: []string{
			"`evals list --provider <provider_id>` — filter to one provider's suites",
			"`evals show <suite_id>` — show a suite's cases + expectations",
			"`evals run <suite_id> --tag <tag>` — run it and store a tagged run",
		},
	})
}

// show prints one suite's cases + expectations.
func (h *handlers) show(ctx cliapp.RunContext) error {
	id := ctx.Positional("suite_id")
	resp, err := h.client.GetSuite(context.Background(), connect.NewRequest(&evalv1.GetSuiteRequest{SuiteId: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get suite %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetSuite() == nil {
		return fmt.Errorf("server returned no suite")
	}
	s := resp.Msg.GetSuite()
	results := make([]string, 0, len(s.GetCases()))
	for _, c := range s.GetCases() {
		results = append(results, formatCase(c))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{formatSuite(s)},
		ResultsHeading: "Cases",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("`evals run %s --tag <tag>` — run this suite", s.GetSuiteId())},
	})
}

// run executes a suite and stores a tagged run.
func (h *handlers) run(ctx cliapp.RunContext) error {
	id := ctx.Positional("suite_id")
	limit, err := parseLimit(ctx.Flag("limit"))
	if err != nil {
		return err
	}
	// RunSuite synchronously executes every case against the live provider; with
	// an LLM reranker leg active this can take minutes, far past the scenario's
	// default client timeout. Use a dedicated long-timeout client so the server
	// is never cancelled mid-run (which would discard the immutable run record).
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(h.core, runSuiteTimeout)
	runClient := evalconnect.NewEvalServiceClient(httpClient, baseURL)
	resp, err := runClient.RunSuite(context.Background(), connect.NewRequest(&evalv1.RunSuiteRequest{
		SuiteId: id,
		Tag:     strings.TrimSpace(ctx.Flag("tag")),
		Limit:   limit,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("run suite %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetRun() == nil {
		return fmt.Errorf("server returned no run")
	}
	r := resp.Msg.GetRun()
	if err := cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Ran %s → run %s (tag %q).", r.GetSuiteId(), r.GetRunId(), r.GetTag())},
		Changes: append([]string{formatAggregate(r), formatConfig(r.GetConfig())}, formatCaseResults(r)...),
		NextCommand: []string{
			fmt.Sprintf("`evals runs %s` — see this suite's run history", r.GetSuiteId()),
			fmt.Sprintf("`evals compare <other_run> %s` — diff against another tagged run", r.GetRunId()),
		},
	}); err != nil {
		return err
	}

	// Opt-in CI gate (the run is always stored first; --assert only affects the
	// exit code). The hub never imposes this — a consumer adds it to its own
	// `make test` when it wants enforcement.
	if ctx.BoolFlag("assert") {
		if failing := failingCases(r); len(failing) > 0 {
			return fmt.Errorf("--assert: %d case(s) did not meet expectations: %s",
				len(failing), strings.Join(failing, ", "))
		}
	}
	return nil
}

// failingCases returns the case_ids whose outcome a --assert gate treats as a
// regression (below_expectation or unexpected_hit). met / above_expectation /
// n/a never fail the gate.
func failingCases(r *evalv1.EvalRun) []string {
	var out []string
	for _, cr := range r.GetResults() {
		switch cr.GetOutcome() {
		case "below_expectation", "unexpected_hit":
			out = append(out, cr.GetCaseId())
		}
	}
	return out
}

// sweep runs the two-tier overfit-safe tuning sweep and renders the ranked arms
// + the promotion verdict. --apply gates the write-back; default is preview.
func (h *handlers) sweep(ctx cliapp.RunContext) error {
	id := ctx.Positional("suite_id")
	limit, err := parseLimit(ctx.Flag("limit"))
	if err != nil {
		return err
	}
	// A sweep runs the suite once per arm and (index-time tier) reindexes per arm,
	// so it needs a much longer client timeout than a single run.
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(h.core, sweepTimeout)
	client := evalconnect.NewEvalServiceClient(httpClient, baseURL)
	resp, err := client.Sweep(context.Background(), connect.NewRequest(&evalv1.SweepRequest{
		SuiteId:       id,
		QueryTimeOnly: ctx.BoolFlag("query-time-only"),
		Apply:         ctx.BoolFlag("apply"),
		Limit:         limit,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("sweep suite %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetResult() == nil {
		return fmt.Errorf("server returned no sweep result")
	}
	res := resp.Msg.GetResult()
	results := make([]string, 0, len(res.GetArms()))
	for _, a := range res.GetArms() {
		results = append(results, formatSweepArm(a, res.GetWinnerTag()))
	}
	next := []string{fmt.Sprintf("`evals runs %s` — every arm is stored as a tagged run", id)}
	if !ctx.BoolFlag("apply") && res.GetWinnerTag() != "" {
		next = append([]string{fmt.Sprintf("`evals sweep %s --apply` — write back the recommended winner", id)}, next...)
	}
	return cliapp.RenderProtoList(ctx, res, cliapp.ListReport{
		Summary:        formatSweepSummary(res),
		ResultsHeading: "Arms (best-first)",
		Results:        results,
		RetrievalHints: next,
	})
}

// generate proposes machine-generated cases for a suite by sampling + inverting
// the provider's index. --apply appends them (each marked generated); default is
// a preview of the proposals + the resulting corpus's adequacy.
func (h *handlers) generate(ctx cliapp.RunContext) error {
	id := ctx.Positional("suite_id")
	count, err := parseLimit(ctx.Flag("count"))
	if err != nil {
		return fmt.Errorf("invalid --count: %w", err)
	}
	// Generation calls the local LLM once per sampled item, so it needs a long
	// client timeout (like run/sweep) — far past the scenario's default.
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(h.core, generateTimeout)
	client := evalconnect.NewEvalServiceClient(httpClient, baseURL)
	resp, err := client.Generate(context.Background(), connect.NewRequest(&evalv1.GenerateRequest{
		SuiteId:   id,
		Count:     count,
		Negatives: ctx.BoolFlag("negatives"),
		Apply:     ctx.BoolFlag("apply"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("generate cases for %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no generate response")
	}
	msg := resp.Msg
	results := make([]string, 0, len(msg.GetProposed()))
	for _, gc := range msg.GetProposed() {
		results = append(results, formatGeneratedCase(gc))
	}

	summary := []string{fmt.Sprintf("Generate %s [provider=%s]", msg.GetSuiteId(), msg.GetProviderId()), msg.GetSummary()}
	if msg.GetApplied() {
		summary = append(summary, fmt.Sprintf("APPLIED — suite now has %d case(s).", len(msg.GetSuite().GetCases())))
	} else if len(msg.GetProposed()) > 0 {
		summary = append(summary, "preview only — re-run with --apply to append these cases.")
	}
	summary = append(summary, formatAdequacy(msg.GetAdequacy())...)

	next := []string{}
	if !ctx.BoolFlag("apply") && len(msg.GetProposed()) > 0 {
		next = append(next, fmt.Sprintf("`evals generate %s --apply` — append the proposed cases", id))
	}
	next = append(next, fmt.Sprintf("`evals show %s` — review the suite's cases", id))
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Proposed cases",
		Results:        results,
		RetrievalHints: next,
	})
}

// runs lists a suite's run history.
func (h *handlers) runs(ctx cliapp.RunContext) error {
	id := ctx.Positional("suite_id")
	limit, err := parseLimit(ctx.Flag("limit"))
	if err != nil {
		return err
	}
	resp, err := h.client.ListRuns(context.Background(), connect.NewRequest(&evalv1.ListRunsRequest{
		SuiteId: id,
		Tag:     strings.TrimSpace(ctx.Flag("tag")),
		Limit:   limit,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("list runs for %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no runs response")
	}
	results := make([]string, 0, len(resp.Msg.GetRuns()))
	for _, r := range resp.Msg.GetRuns() {
		results = append(results, formatRunLine(r))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d run(s) for %s.", len(resp.Msg.GetRuns()), id)},
		ResultsHeading: "Runs (newest first)",
		Results:        results,
		RetrievalHints: []string{
			"`evals runs <suite_id> --tag <tag>` — filter to one experiment tag",
			"`evals compare <run_a> <run_b>` — per-case A/B delta",
		},
	})
}

// showRun prints one immutable run.
func (h *handlers) showRun(ctx cliapp.RunContext) error {
	id := ctx.Positional("run_id")
	resp, err := h.client.GetRun(context.Background(), connect.NewRequest(&evalv1.GetRunRequest{RunId: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get run %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetRun() == nil {
		return fmt.Errorf("server returned no run")
	}
	r := resp.Msg.GetRun()
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{formatRunLine(r), formatAggregate(r), formatConfig(r.GetConfig())},
		ResultsHeading: "Cases",
		Results:        formatCaseResults(r),
	})
}

// compare diffs two runs per case.
func (h *handlers) compare(ctx cliapp.RunContext) error {
	a := ctx.Positional("run_a")
	b := ctx.Positional("run_b")
	resp, err := h.client.CompareRuns(context.Background(), connect.NewRequest(&evalv1.CompareRunsRequest{RunA: a, RunB: b}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("compare runs %q vs %q", a, b), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no compare response")
	}
	results := make([]string, 0, len(resp.Msg.GetDeltas()))
	for _, d := range resp.Msg.GetDeltas() {
		results = append(results, formatDelta(d))
	}
	tagA := resp.Msg.GetRunA().GetTag()
	tagB := resp.Msg.GetRunB().GetTag()
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("A=%s (%s)  →  B=%s (%s)", a, tagA, b, tagB)},
		ResultsHeading: "Per-case delta (A → B)",
		Results:        results,
	})
}

// readSuiteArg returns suite JSON bytes from the flag value: a leading '@' means
// "read from this file path"; otherwise the value is the JSON itself.
func readSuiteArg(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("--suite is required (JSON, or @path to a file)")
	}
	if strings.HasPrefix(raw, "@") {
		path := strings.TrimPrefix(raw, "@")
		blob, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read --suite file %q: %w", path, err)
		}
		return blob, nil
	}
	return []byte(raw), nil
}

func parseLimit(raw string) (int32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid --limit %q (want a non-negative integer)", raw)
	}
	return int32(n), nil
}

func formatSuite(s *evalv1.EvalSuite) string {
	if s == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s [provider=%s, %d case(s), %s]", s.GetSuiteId(), s.GetProviderId(), len(s.GetCases()), s.GetState())
}

func formatCase(c *evalv1.EvalCase) string {
	tags := ""
	if len(c.GetTags()) > 0 {
		tags = " {" + strings.Join(c.GetTags(), ",") + "}"
	}
	exp := []string{}
	if len(c.GetExpectIds()) > 0 {
		exp = append(exp, "ids="+strings.Join(c.GetExpectIds(), "|"))
	}
	if c.GetExpectWithinTopK() > 0 {
		exp = append(exp, fmt.Sprintf("topK=%d", c.GetExpectWithinTopK()))
	}
	if c.GetExpectMinScore() > 0 {
		exp = append(exp, fmt.Sprintf("min=%.2f", c.GetExpectMinScore()))
	}
	if c.GetExpectMaxScore() > 0 {
		exp = append(exp, fmt.Sprintf("max=%.2f", c.GetExpectMaxScore()))
	}
	if c.GetExpectNoStrongHit() {
		exp = append(exp, "no-strong-hit")
	}
	expectation := ""
	if len(exp) > 0 {
		expectation = " expect[" + strings.Join(exp, " ") + "]"
	}
	return fmt.Sprintf("%s%s: %q%s", c.GetCaseId(), tags, c.GetQuery(), expectation)
}

func formatAggregate(r *evalv1.EvalRun) string {
	a := r.GetAggregate()
	return fmt.Sprintf("aggregate: cases=%d met=%d below=%d mean_strong_top1=%.3f max_gibberish=%.3f p95=%dms",
		a.GetCases(), a.GetMet(), a.GetBelow(), a.GetMeanStrongTop1(), a.GetMaxGibberishScore(), a.GetLatencyP95Ms())
}

func formatConfig(c *evalv1.ConfigSnapshot) string {
	if c == nil {
		return "config: (none captured)"
	}
	return fmt.Sprintf("config: reranker=%s rerank_enabled=%t embed_model=%s indexed=%d",
		emptyDash(c.GetRerankerLeg()), c.GetRerankEnabled(), emptyDash(c.GetEmbedModel()), c.GetIndexedCount())
}

func formatCaseResults(r *evalv1.EvalRun) []string {
	out := make([]string, 0, len(r.GetResults()))
	for _, cr := range r.GetResults() {
		rank := "-"
		if cr.GetExpectedRank() > 0 {
			rank = strconv.Itoa(int(cr.GetExpectedRank()))
		}
		out = append(out, fmt.Sprintf("%-16s %-18s top=%.3f rank=%s", cr.GetCaseId(), cr.GetOutcome(), cr.GetObservedTopScore(), rank))
	}
	return out
}

func formatRunLine(r *evalv1.EvalRun) string {
	a := r.GetAggregate()
	return fmt.Sprintf("%s  tag=%-14s %s  met=%d/%d strong_top1=%.3f gibberish=%.3f",
		r.GetCreatedAt(), r.GetTag(), r.GetRunId(), a.GetMet(), a.GetCases(), a.GetMeanStrongTop1(), a.GetMaxGibberishScore())
}

func formatDelta(d *evalv1.CaseDelta) string {
	return fmt.Sprintf("%-16s %-18s → %-18s  top %.3f → %.3f  rank %d → %d",
		d.GetCaseId(), emptyDash(d.GetOutcomeA()), emptyDash(d.GetOutcomeB()),
		d.GetTopScoreA(), d.GetTopScoreB(), d.GetExpectedRankA(), d.GetExpectedRankB())
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

// formatSweepSummary renders the verdict header: the provider, the incumbent vs
// winner, whether a winner was promoted, the decision stats, and the
// recommendation lines.
func formatSweepSummary(res *evalv1.SweepResult) []string {
	st := res.GetStats()
	verdict := "no significant improvement — incumbent retained"
	if res.GetWinnerTag() != "" {
		verdict = "winner: " + res.GetWinnerTag()
		if res.GetPromoted() {
			verdict += " (PROMOTED — written back)"
		} else {
			verdict += " (preview only — re-run with --apply to write back)"
		}
	}
	out := []string{
		fmt.Sprintf("Sweep %s [provider=%s]", res.GetSuiteId(), res.GetProviderId()),
		verdict,
		fmt.Sprintf("incumbent recall=%.3f  winner recall=%.3f  margin=%+.3f  95%% CI=[%+.3f,%+.3f]",
			st.GetIncumbentScore(), st.GetWinnerScore(), st.GetMargin(), st.GetCiLow(), st.GetCiHigh()),
		fmt.Sprintf("held-out: winner=%.3f incumbent=%.3f  |  arms: query-time=%d index-time=%d  dropped-interactions=%d",
			st.GetHeldoutWinnerScore(), st.GetHeldoutIncumbentScore(), st.GetQueryTimeArms(), st.GetIndexTimeArms(), st.GetDroppedIndexInteractions()),
	}
	for _, line := range strings.Split(strings.TrimSpace(res.GetRecommendation()), "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, "→ "+line)
		}
	}
	return out
}

// formatGeneratedCase renders one proposed case: its provenance (source item +
// stratum) followed by the case itself.
func formatGeneratedCase(gc *evalv1.GeneratedCase) string {
	c := gc.GetCase()
	prov := gc.GetStratum()
	if gc.GetSourceId() != "" {
		prov = fmt.Sprintf("%s ← %s", emptyDash(gc.GetStratum()), gc.GetSourceId())
	}
	return fmt.Sprintf("[%s] %s", prov, formatCase(c))
}

// formatAdequacy renders the warn-level adequacy findings (each prefixed "⚠"),
// or a single "corpus adequate" line when there are none.
func formatAdequacy(ws []*evalv1.AdequacyWarning) []string {
	if len(ws) == 0 {
		return []string{"adequacy: corpus adequate (no warnings)"}
	}
	out := make([]string, 0, len(ws)+1)
	out = append(out, fmt.Sprintf("adequacy: %d warning(s)", len(ws)))
	for _, w := range ws {
		out = append(out, fmt.Sprintf("  ⚠ %s: %s", w.GetCode(), w.GetMessage()))
	}
	return out
}

// formatSweepArm renders one ranked arm line: a marker for the winner, the tier,
// recall, feasibility, the floor regime, and any note.
func formatSweepArm(a *evalv1.SweepArm, winnerTag string) string {
	marker := " "
	if a.GetTag() == winnerTag {
		marker = "★"
	}
	feasible := "ok"
	if !a.GetFeasible() {
		feasible = "INFEASIBLE"
	}
	cfg := a.GetConfig()
	note := ""
	if strings.TrimSpace(a.GetNote()) != "" {
		note = "  (" + a.GetNote() + ")"
	}
	return fmt.Sprintf("%s [%-11s] recall=%.3f %-10s engine=%s rerank=%t/blend=%t prefix=%t floor=%s%s",
		marker, a.GetTier(), a.GetScore(), feasible,
		emptyDash(cfg.GetEngine()), cfg.GetRerankEnabled(), cfg.GetRerankBlend(), cfg.GetEmbedTaskPrefix(), emptyDash(cfg.GetFloorRegime()), note)
}
