package findings

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	findingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/findings"
	findingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/findings/findings_v1connect"

	"github.com/vrooli/cli-core/cliapp"

	"web-search/cli/internal/cliutil"
)

const defaultTimeWindow = "this_week"

type handlers struct {
	core   *cliapp.ScenarioApp
	client findingsconnect.FindingsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: findingsconnect.NewFindingsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	status, err := statusFromFlag(ctx.Flag("status"))
	if err != nil {
		return err
	}
	resp, err := h.client.ListFindings(context.Background(), connect.NewRequest(&findingsv1.ListFindingsRequest{
		Status:          status,
		IncludeArchived: ctx.BoolFlag("include-archived"),
		Limit:           cliutil.ParseInt32(ctx.Flag("limit")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list findings", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no findings response")
	}
	results := make([]string, 0, len(resp.Msg.Findings))
	for _, f := range resp.Msg.Findings {
		results = append(results, formatFinding(f))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d finding(s).", len(resp.Msg.Findings))},
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: []string{
			"`findings get <id>` — show a single finding",
			"`findings search <query>` — semantic search the corpus",
			"`findings list --include-archived` — include superseded",
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetFinding(context.Background(), connect.NewRequest(&findingsv1.GetFindingRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get finding %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Finding == nil {
		return fmt.Errorf("server returned no finding")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched finding %s.", resp.Msg.Finding.Id)},
		ResultsHeading: "Finding",
		Results:        []string{formatFinding(resp.Msg.Finding)},
	})
}

func (h *handlers) add(ctx cliapp.RunContext) error {
	cites, err := parseCitations(ctx.Flag("citations"))
	if err != nil {
		return err
	}
	resp, err := h.client.AddFinding(context.Background(), connect.NewRequest(&findingsv1.AddFindingRequest{
		Claim:      ctx.Flag("claim"),
		Confidence: parseFloat(ctx.Flag("confidence")),
		Query:      ctx.Flag("query"),
		Source:     sourceFromFlag(ctx.Flag("source")),
		Citations:  cites,
	}))
	if err != nil {
		return cliapp.WrapAPIError("add finding", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Finding == nil {
		return fmt.Errorf("server returned no finding")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Added finding %s.", resp.Msg.Finding.Id)},
		Changes:     []string{formatFinding(resp.Msg.Finding)},
		NextCommand: []string{fmt.Sprintf("`findings get %s` — show this finding", resp.Msg.Finding.Id)},
	})
}

func (h *handlers) edit(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.EditFinding(context.Background(), connect.NewRequest(&findingsv1.EditFindingRequest{
		Id:         id,
		Claim:      ctx.Flag("claim"),
		Confidence: parseFloat(ctx.Flag("confidence")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("edit finding %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Finding == nil {
		return fmt.Errorf("server returned no finding")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Edited finding %s.", resp.Msg.Finding.Id)},
		Changes: []string{formatFinding(resp.Msg.Finding)},
	})
}

func (h *handlers) supersede(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.SupersedeFinding(context.Background(), connect.NewRequest(&findingsv1.SupersedeFindingRequest{
		Id:          id,
		Replacement: ctx.Flag("replacement"),
		Reason:      ctx.Flag("reason"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("supersede finding %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Finding == nil {
		return fmt.Errorf("server returned no finding")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Superseded finding %s.", resp.Msg.Finding.Id)},
		Changes: []string{formatFinding(resp.Msg.Finding)},
	})
}

func (h *handlers) flag(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.FlagFinding(context.Background(), connect.NewRequest(&findingsv1.FlagFindingRequest{
		Id:     id,
		Reason: ctx.Flag("reason"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("flag finding %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Finding == nil {
		return fmt.Errorf("server returned no finding")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Flagged finding %s as disputed.", resp.Msg.Finding.Id)},
		Changes: []string{formatFinding(resp.Msg.Finding)},
	})
}

func (h *handlers) prune(ctx cliapp.RunContext) error {
	dryRun := ctx.BoolFlag("dry-run")
	if !dryRun && !ctx.BoolFlag("force") {
		return fmt.Errorf("refusing to prune without --force: pruning permanently deletes superseded findings; preview with --dry-run, then re-run with --force to execute")
	}
	resp, err := h.client.PruneFindings(context.Background(), connect.NewRequest(&findingsv1.PruneFindingsRequest{DryRun: dryRun}))
	if err != nil {
		return cliapp.WrapAPIError("prune findings", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no prune response")
	}
	verb := "Pruned"
	if dryRun {
		verb = "Would prune"
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("%s %d superseded finding(s).", verb, resp.Msg.Pruned)},
		Changes: resp.Msg.FindingIds,
	})
}

func (h *handlers) search(ctx cliapp.RunContext) error {
	query := ctx.Positional("query")
	resp, err := h.client.SearchFindings(context.Background(), connect.NewRequest(&findingsv1.SearchFindingsRequest{
		Query:           query,
		Limit:           cliutil.ParseInt32(ctx.Flag("limit")),
		IncludeArchived: ctx.BoolFlag("include-archived"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("search findings", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no search response")
	}
	results := make([]string, 0, len(resp.Msg.Hits))
	for _, hit := range resp.Msg.Hits {
		results = append(results, formatHit(hit))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d hit(s) for %q (method: %s).", len(resp.Msg.Hits), query, resp.Msg.Method)},
		ResultsHeading: "Hits",
		Results:        results,
	})
}

func (h *handlers) count(ctx cliapp.RunContext) error {
	window := strings.TrimSpace(ctx.Flag("window"))
	if window == "" {
		window = defaultTimeWindow
	}
	token, err := timeWindowToken(window)
	if err != nil {
		return err
	}
	resp, err := h.client.CountFindings(context.Background(), connect.NewRequest(&findingsv1.CountFindingsRequest{
		Window: &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Token{Token: token}},
	}))
	if err != nil {
		return cliapp.WrapAPIError("count findings", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no count response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d finding(s) captured (%s).", resp.Msg.Count, window)},
		ResultsHeading: "Findings captured",
		Results:        []string{fmt.Sprintf("%d (%s)", resp.Msg.Count, window)},
	})
}

func (h *handlers) effectiveness(ctx cliapp.RunContext) error {
	resp, err := h.client.ListEffectiveness(context.Background(), connect.NewRequest(&findingsv1.ListEffectivenessRequest{
		Limit:           cliutil.ParseInt32(ctx.Flag("limit")),
		IncludeDisputed: ctx.BoolFlag("include-disputed"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list finding effectiveness", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no effectiveness response")
	}
	results := make([]string, 0, len(resp.Msg.Items))
	for _, it := range resp.Msg.Items {
		results = append(results, formatEffectiveness(it))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d finding(s) by usage effectiveness (effective = age-decayed confidence × usage factor).", len(resp.Msg.Items))},
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: []string{
			"`findings use <id>` — record an explicit 'used' signal",
			"never-surfaced + fully-decayed findings are the GC's supersede candidates",
		},
	})
}

func (h *handlers) use(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.RecordUsage(context.Background(), connect.NewRequest(&findingsv1.RecordUsageRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("record usage for finding %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Finding == nil {
		return fmt.Errorf("server returned no finding")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Recorded usage for finding %s.", resp.Msg.Finding.Id)},
		Changes: []string{formatFinding(resp.Msg.Finding)},
	})
}

func (h *handlers) gc(ctx cliapp.RunContext) error {
	dryRun := ctx.BoolFlag("dry-run")
	resp, err := h.client.RunGC(context.Background(), connect.NewRequest(&findingsv1.RunGCRequest{DryRun: dryRun}))
	if err != nil {
		return cliapp.WrapAPIError("run findings GC", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no GC response")
	}
	msg := resp.Msg
	mode := "applied"
	if msg.DryRun {
		mode = "dry-run (nothing mutated)"
	}
	summary := []string{
		fmt.Sprintf("Store-consistency GC %s.", mode),
		fmt.Sprintf("Superseded (never-surfaced + decayed): %d", len(msg.SupersededDecayed)),
		fmt.Sprintf("Cold-archive candidates (superseded past TTL, report only): %d", len(msg.ColdArchiveCandidates)),
		fmt.Sprintf("Stale disputes (past TTL, flagged for human — never auto-resolved): %d", len(msg.StaleDisputes)),
		fmt.Sprintf("Orphans (brief_id with no brief): %d", len(msg.Orphans)),
	}
	results := make([]string, 0)
	for _, id := range msg.SupersededDecayed {
		results = append(results, "superseded: "+id)
	}
	for _, id := range msg.StaleDisputes {
		results = append(results, "stale-dispute: "+id)
	}
	for _, id := range msg.Orphans {
		results = append(results, "orphan: "+id)
	}
	report := cliapp.MutationReport{Result: summary}
	if len(results) > 0 {
		report.Changes = results
	}
	return cliapp.RenderProtoMutation(ctx, msg, report)
}

// --- helpers ---

func formatEffectiveness(it *findingsv1.FindingEffectiveness) string {
	if it == nil || it.Finding == nil {
		return "(nil)"
	}
	last := "never"
	if it.LastSurfacedAt != nil {
		last = it.LastSurfacedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — %s [surfaced=%d used=%d last_surfaced=%s eff_conf=%.2f usage_factor=%.2f score=%.3f]",
		it.Finding.Id, it.Finding.Claim, it.SurfacedCount, it.UsedCount, last,
		it.EffectiveConfidence, it.UsageFactor, it.EffectiveScore)
}

func parseFloat(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return v
}

// parseCitations parses "url|title,url|title" into citation inputs. A bare
// entry with no pipe is treated as a url with an empty title.
func parseCitations(raw string) ([]*findingsv1.CitationInput, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out []*findingsv1.CitationInput
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		url, title, found := strings.Cut(entry, "|")
		c := &findingsv1.CitationInput{Url: strings.TrimSpace(url)}
		if found {
			c.Title = strings.TrimSpace(title)
		}
		out = append(out, c)
	}
	return out, nil
}

func statusFromFlag(s string) (findingsv1.FindingStatus, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return findingsv1.FindingStatus_FINDING_STATUS_UNSPECIFIED, nil
	case "active":
		return findingsv1.FindingStatus_FINDING_STATUS_ACTIVE, nil
	case "disputed":
		return findingsv1.FindingStatus_FINDING_STATUS_DISPUTED, nil
	case "superseded":
		return findingsv1.FindingStatus_FINDING_STATUS_SUPERSEDED, nil
	default:
		return 0, fmt.Errorf("unknown status %q (use active, disputed, or superseded)", s)
	}
}

func sourceFromFlag(s string) findingsv1.FindingSource {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "l2":
		return findingsv1.FindingSource_FINDING_SOURCE_L2
	case "l3":
		return findingsv1.FindingSource_FINDING_SOURCE_L3
	default:
		return findingsv1.FindingSource_FINDING_SOURCE_MANUAL
	}
}

func timeWindowToken(token string) (measuresv1.TimeWindowToken, error) {
	name := "TIME_WINDOW_TOKEN_" + strings.ToUpper(token)
	v, ok := measuresv1.TimeWindowToken_value[name]
	if !ok || measuresv1.TimeWindowToken(v) == measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_UNSPECIFIED {
		return 0, fmt.Errorf("unknown time window %q (use one of: this_week, last_7d, last_30d, this_month, last_month, this_quarter)", token)
	}
	return measuresv1.TimeWindowToken(v), nil
}

func formatFinding(f *findingsv1.Finding) string {
	if f == nil {
		return "(nil)"
	}
	created := ""
	if f.CreatedAt != nil {
		created = f.CreatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — %s [status=%s confidence=%.2f citations=%d created=%s]",
		f.Id, f.Claim, f.Status.String(), f.Confidence, len(f.Citations), created)
}

func formatHit(hit *findingsv1.FindingHit) string {
	if hit == nil || hit.Finding == nil {
		return "(nil)"
	}
	weak := ""
	if hit.Weak {
		weak = " (weak)"
	}
	return fmt.Sprintf("[%.3f%s] %s", hit.Score, weak, formatFinding(hit.Finding))
}
