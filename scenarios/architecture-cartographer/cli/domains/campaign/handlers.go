package campaign

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"connectrpc.com/connect"

	campaignv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/campaign"
	campaignconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/campaign/campaign_v1connect"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client campaignconnect.CampaignServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: campaignconnect.NewCampaignServiceClient(httpClient, baseURL),
	}
}

// loadedAudit is the parsed result of a test-genie findings document: the
// flattened findings, the document's own scenario (when present), and the
// set of finding-source tokens whose phases actually RAN this audit (used to
// scope a reaudit's coverage so non-run sources are not false-validated).
type loadedAudit struct {
	scenario       string
	findings       []*architecturev1.ArchitectureFinding
	coveredSources []string
	// hasSourceTokens is true when at least one phase carried a findingSource
	// token. A foreign/hand-built document without any tokens cannot scope
	// coverage, so the reaudit falls back to all-sources-covered.
	hasSourceTokens bool
}

// loadAuditFindings reads a test-genie findings document (the persisted
// coverage/runs/<runID>/findings.json artifact, or a `test-genie execute
// --json` SuiteExecutionResult) from a path (or stdin when "-") and flattens
// every phase's findings into the shared ArchitectureFinding slice the tracker
// ingests. The findings serialize with proto-int enums; encoding/json
// round-trips them back into the generated type since both sides share this
// contract.
//
// It hard-errors on the two silent-empty footguns the assessment found:
// a document with no `phases` key at all (e.g. a single per-phase file), and a
// well-formed document that yields zero findings. An empty campaign is never
// useful, so there is no override flag.
func loadAuditFindings(path string) (loadedAudit, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return loadedAudit{}, fmt.Errorf("no audit report given (--from-audit is required)")
	}
	var (
		data []byte
		err  error
	)
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return loadedAudit{}, fmt.Errorf("read audit report %q: %w", path, err)
	}
	// Phases is a pointer so a missing key (nil) is distinguishable from a
	// present-but-empty array.
	var report struct {
		Scenario string `json:"scenario"`
		Phases   *[]struct {
			Name          string                                `json:"name"`
			Status        string                                `json:"status"`
			FindingSource string                                `json:"findingSource"`
			Findings      []*architecturev1.ArchitectureFinding `json:"findings"`
		} `json:"phases"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return loadedAudit{}, fmt.Errorf("parse audit report %q (expected a test-genie findings document): %w", path, err)
	}
	if report.Phases == nil {
		return loadedAudit{}, fmt.Errorf(
			"audit report %q is not a test-genie suite findings document (expected `phases[].findings`) — pass the findings.json persisted under coverage/runs/<runID>/ or `test-genie execute <scenario> --json` output",
			path)
	}

	out := loadedAudit{scenario: strings.TrimSpace(report.Scenario)}
	covered := map[string]struct{}{}
	for _, p := range *report.Phases {
		for _, f := range p.Findings {
			if f != nil {
				out.findings = append(out.findings, f)
			}
		}
		src := strings.TrimSpace(p.FindingSource)
		if src == "" {
			continue
		}
		out.hasSourceTokens = true
		// A source is covered only when its phase actually ran (passed or
		// failed); a skipped phase saw nothing and must not validate items.
		switch strings.ToLower(strings.TrimSpace(p.Status)) {
		case "passed", "failed":
			covered[src] = struct{}{}
		}
	}
	for src := range covered {
		out.coveredSources = append(out.coveredSources, src)
	}
	sort.Strings(out.coveredSources)

	if len(out.findings) == 0 {
		return loadedAudit{}, fmt.Errorf(
			"audit report %q contains zero findings — nothing to track (an empty campaign is never useful)",
			path)
	}
	return out, nil
}

// parseProfile maps the --profile flag to the wire enum. Empty/omitted →
// BALANCED. An unknown value is a clear error so a typo never silently
// reorders the worklist.
func parseProfile(raw string) (campaignv1.RankProfile, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "balanced":
		return campaignv1.RankProfile_RANK_PROFILE_BALANCED, nil
	case "fast":
		return campaignv1.RankProfile_RANK_PROFILE_FAST, nil
	case "long-term", "long_term", "longterm":
		return campaignv1.RankProfile_RANK_PROFILE_LONG_TERM, nil
	default:
		return campaignv1.RankProfile_RANK_PROFILE_UNSPECIFIED,
			fmt.Errorf("unknown --profile %q (want one of: fast, balanced, long-term)", raw)
	}
}

func profileName(p campaignv1.RankProfile) string {
	switch p {
	case campaignv1.RankProfile_RANK_PROFILE_FAST:
		return "fast"
	case campaignv1.RankProfile_RANK_PROFILE_LONG_TERM:
		return "long-term"
	default:
		return "balanced"
	}
}

func (h *handlers) create(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	audit, err := loadAuditFindings(ctx.Flag("from-audit"))
	if err != nil {
		return err
	}
	// Guard against ingesting scenario A's findings into a scenario-B campaign:
	// the artifact records the scenario it was produced for.
	if audit.scenario != "" && !strings.EqualFold(audit.scenario, strings.TrimSpace(scenario)) {
		return fmt.Errorf(
			"audit report is for scenario %q but the campaign targets %q — re-run the audit for %q or fix the create argument",
			audit.scenario, scenario, scenario)
	}
	resp, err := h.client.CreateCampaign(context.Background(), connect.NewRequest(&campaignv1.CreateCampaignRequest{
		Scenario: scenario,
		Name:     ctx.Flag("name"),
		Findings: audit.findings,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("create campaign for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetStatus() == nil {
		return fmt.Errorf("server returned no campaign")
	}
	st := resp.Msg.GetStatus()
	hint := fmt.Sprintf("`campaign next %s --profile balanced` to get the ranked worklist.", st.GetCampaign().GetId())
	return h.renderStatus(ctx, st,
		fmt.Sprintf("Campaign %s created for %q — %d finding(s) ingested.", st.GetCampaign().GetId(), scenario, st.GetTotal()),
		hint)
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	scenario := ctx.Flag("scenario")
	resp, err := h.client.ListCampaigns(context.Background(), connect.NewRequest(&campaignv1.ListCampaignsRequest{Scenario: scenario}))
	if err != nil {
		return cliapp.WrapAPIError("list campaigns", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no campaigns response")
	}
	campaigns := resp.Msg.GetCampaigns()
	results := make([]string, 0, len(campaigns))
	for _, c := range campaigns {
		name := c.GetName()
		if name == "" {
			name = "(unnamed)"
		}
		results = append(results, fmt.Sprintf("%s  %s  [%s]  %s", c.GetId(), c.GetScenario(), lifecycleName(c.GetStatus()), name))
	}
	summary := fmt.Sprintf("%d campaign(s).", len(campaigns))
	if len(campaigns) == 0 {
		summary = "No campaigns. `campaign create <scenario> --from-audit <report.json>` to start one."
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Campaigns",
		Results:        results,
	})
}

func (h *handlers) status(ctx cliapp.RunContext) error {
	id := ctx.Positional("campaign-id")
	resp, err := h.client.GetCampaignStatus(context.Background(), connect.NewRequest(&campaignv1.GetCampaignStatusRequest{CampaignId: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get campaign %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetStatus() == nil {
		return fmt.Errorf("server returned no campaign")
	}
	st := resp.Msg.GetStatus()
	return h.renderStatus(ctx, st,
		fmt.Sprintf("Campaign %s (%s): %d/%d resolved, %d validated, %d regression(s).",
			st.GetCampaign().GetId(), lifecycleName(st.GetCampaign().GetStatus()),
			st.GetResolved(), st.GetTotal(), st.GetValidated(), st.GetRegressions()),
		fmt.Sprintf("`campaign next %s` for the next worklist chunk.", id))
}

func (h *handlers) next(ctx cliapp.RunContext) error {
	id := ctx.Positional("campaign-id")
	profile, err := parseProfile(ctx.Flag("profile"))
	if err != nil {
		return err
	}
	resp, err := h.client.NextCampaignStep(context.Background(), connect.NewRequest(&campaignv1.NextCampaignStepRequest{
		CampaignId: id,
		Profile:    profile,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("next step for campaign %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no worklist")
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	items := resp.Msg.GetItems()
	results := make([]string, 0, len(items))
	for _, f := range items {
		results = append(results, findingLine(f))
	}
	summary := fmt.Sprintf("%d open item(s) — worklist ordered for the '%s' profile.", len(items), profileName(profile))
	if len(items) == 0 {
		summary = "No open items. Re-audit to confirm, then close the campaign."
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Worklist",
		Results:        results,
	})
}

func (h *handlers) resolve(ctx cliapp.RunContext) error {
	id := ctx.Positional("campaign-id")
	resp, err := h.client.ResolveItem(context.Background(), connect.NewRequest(&campaignv1.ResolveItemRequest{
		CampaignId: id,
		StableId:   ctx.Flag("stable_id"),
		Note:       ctx.Flag("note"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("resolve item in campaign %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetItem() == nil {
		return fmt.Errorf("server returned no item")
	}
	f := resp.Msg.GetItem()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Resolved %s (%s).", f.GetStableId(), statusName(f.GetStatus()))},
		Changes:     []string{findingLine(f)},
		NextCommand: []string{fmt.Sprintf("`campaign reaudit %s --from-audit <report.json>` to confirm the fix held.", id)},
	})
}

func (h *handlers) apply(ctx cliapp.RunContext) error {
	id := ctx.Positional("campaign-id")
	resp, err := h.client.ApplyItem(context.Background(), connect.NewRequest(&campaignv1.ApplyItemRequest{
		CampaignId: id,
		StableId:   ctx.Flag("stable_id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("apply item in campaign %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetItem() == nil {
		return fmt.Errorf("server returned no item")
	}
	f := resp.Msg.GetItem()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Applied %s (status-only, %s).", f.GetStableId(), statusName(f.GetStatus()))},
		Changes:     []string{findingLine(f)},
		NextCommand: []string{fmt.Sprintf("`campaign reaudit %s --from-audit <report.json>` to confirm.", id)},
	})
}

func (h *handlers) reaudit(ctx cliapp.RunContext) error {
	id := ctx.Positional("campaign-id")
	audit, err := loadAuditFindings(ctx.Flag("from-audit"))
	if err != nil {
		return err
	}
	// Scope the reaudit to the sources whose phases actually ran. A foreign
	// document without findingSource tokens can't scope coverage, so fall back
	// to all-sources-covered and say so (empty covered_sources = full suite).
	coveredSources := audit.coveredSources
	if !audit.hasSourceTokens {
		coveredSources = nil
		fmt.Fprintln(ctx.Stderr(), "note: audit report has no per-phase findingSource tokens — treating every source as covered (full-suite reaudit)")
	}
	resp, err := h.client.ReauditCampaign(context.Background(), connect.NewRequest(&campaignv1.ReauditCampaignRequest{
		CampaignId:     id,
		Findings:       audit.findings,
		CoveredSources: coveredSources,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("reaudit campaign %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no reaudit result")
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	st := resp.Msg.GetStatus()
	status := fmt.Sprintf("Reaudit: %d validated, %d still open, %d regression(s), %d not re-audited. Progress: %d/%d resolved.",
		len(resp.Msg.GetValidated()), len(resp.Msg.GetStillOpen()), len(resp.Msg.GetRegressions()), len(resp.Msg.GetNotReaudited()),
		st.GetResolved()+st.GetValidated(), st.GetTotal())
	var triage []cliapp.TriageGroup
	if regs := resp.Msg.GetRegressions(); len(regs) > 0 {
		items := make([]string, 0, len(regs))
		for _, f := range regs {
			items = append(items, findingLine(f))
		}
		triage = append(triage, cliapp.TriageGroup{Heading: "⚠️  Regressions (introduced or reappeared)", Items: items})
	}
	if nr := resp.Msg.GetNotReaudited(); len(nr) > 0 {
		items := make([]string, 0, len(nr))
		for _, f := range nr {
			items = append(items, findingLine(f))
		}
		triage = append(triage, cliapp.TriageGroup{
			Heading: fmt.Sprintf("%d item(s) not re-audited — their phases did not run", len(nr)),
			Items:   items,
		})
	}
	return ctx.RenderOperational(cliapp.OperationalReport{
		Status: []string{status},
		Triage: triage,
		NextSteps: []string{
			fmt.Sprintf("`campaign next %s` for the remaining worklist.", id),
			fmt.Sprintf("`campaign close %s` once all items are validated.", id),
		},
	})
}

func (h *handlers) close(ctx cliapp.RunContext) error {
	id := ctx.Positional("campaign-id")
	resp, err := h.client.CloseCampaign(context.Background(), connect.NewRequest(&campaignv1.CloseCampaignRequest{CampaignId: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("close campaign %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetStatus() == nil {
		return fmt.Errorf("server returned no campaign")
	}
	st := resp.Msg.GetStatus()
	return h.renderStatus(ctx, st,
		fmt.Sprintf("Campaign %s closed.", st.GetCampaign().GetId()),
		"")
}

// renderStatus renders a CampaignStatus as human output (or JSON when
// --json). Open items are grouped into a triage block.
func (h *handlers) renderStatus(ctx cliapp.RunContext, st *campaignv1.CampaignStatus, summary, hint string) error {
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), st)
	}
	var triage []cliapp.TriageGroup
	var open, regressed []string
	for _, f := range st.GetItems() {
		if f.GetRegressed() {
			regressed = append(regressed, findingLine(f))
		}
		if isOpenStatus(f.GetStatus()) {
			open = append(open, findingLine(f))
		}
	}
	if len(regressed) > 0 {
		triage = append(triage, cliapp.TriageGroup{Heading: "⚠️  Regressions", Items: regressed})
	}
	if len(open) > 0 {
		triage = append(triage, cliapp.TriageGroup{Heading: "Open items", Items: open})
	}
	next := []string{}
	if hint != "" {
		next = append(next, hint)
	}
	return ctx.RenderOperational(cliapp.OperationalReport{
		Status:    []string{summary},
		Triage:    triage,
		NextSteps: next,
	})
}

// findingLine renders one tracked item for human output.
func findingLine(f *campaignv1.CampaignItem) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s  %s/%s  [%s]", f.GetStableId(), f.GetSource(), f.GetCode(), f.GetSeverity())
	if locs := f.GetLocations(); len(locs) > 0 {
		fmt.Fprintf(&b, " %s", strings.Join(locs, ", "))
	}
	if eff := f.GetEffort(); eff != "" && eff != "unspecified" {
		fmt.Fprintf(&b, " {%s}", eff)
	}
	b.WriteString("  → " + statusName(f.GetStatus()))
	if f.GetRegressed() {
		b.WriteString(" (REGRESSED)")
	}
	return b.String()
}

func statusName(s campaignv1.CampaignItemStatus) string {
	return strings.ToLower(strings.TrimPrefix(s.String(), "CAMPAIGN_ITEM_STATUS_"))
}

func lifecycleName(s campaignv1.CampaignLifecycle) string {
	return strings.ToLower(strings.TrimPrefix(s.String(), "CAMPAIGN_LIFECYCLE_"))
}

func isOpenStatus(s campaignv1.CampaignItemStatus) bool {
	switch s {
	case campaignv1.CampaignItemStatus_CAMPAIGN_ITEM_STATUS_RESOLVED,
		campaignv1.CampaignItemStatus_CAMPAIGN_ITEM_STATUS_VALIDATED,
		campaignv1.CampaignItemStatus_CAMPAIGN_ITEM_STATUS_COMMITTED,
		campaignv1.CampaignItemStatus_CAMPAIGN_ITEM_STATUS_FORCE_RESOLVED:
		return false
	default:
		return true
	}
}
