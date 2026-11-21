package scores

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	scoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring"
	scoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring/scoring_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers closes over the Connect client so each subcommand has typed API
// access without re-resolving the base URL.
type handlers struct {
	core   *cliapp.ScenarioApp
	client scoringconnect.ScoreServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: scoringconnect.NewScoreServiceClient(httpClient, baseURL),
	}
}

// get calls ScoreService.GetScore. Output routing: --json emits the proto
// wire shape (identical to a direct curl of the RPC); human consumers get
// the formatted status report (format.go).
func (h *handlers) get(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.GetScore(context.Background(), connect.NewRequest(&scoringv1.GetScoreRequest{
		Scenario: scenario,
	}))
	if err != nil {
		return cliapp.WrapAPIError("get score", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no score response")
	}

	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	_, err = fmt.Fprint(ctx.Stdout(), FormatReport(resp.Msg))
	return err
}

func (h *handlers) trend(ctx cliapp.RunContext) error {
	limit, err := parseOptionalInt32Flag(ctx, "limit")
	if err != nil {
		return err
	}
	resp, err := h.client.GetScoreTrend(context.Background(), connect.NewRequest(&scoringv1.GetScoreTrendRequest{
		Scenario: ctx.Positional("scenario"),
		Limit:    limit,
	}))
	if err != nil {
		return cliapp.WrapAPIError("get score trend", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no score trend response")
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	_, err = fmt.Fprint(ctx.Stdout(), FormatTrend(resp.Msg))
	return err
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	limit, err := parseOptionalInt32Flag(ctx, "page_size")
	if err != nil {
		return err
	}
	req := &scoringv1.ListScoresRequest{
		SortBy:    sortByFlag(ctx.Flag("sort")),
		Order:     sortOrderFlag(ctx.Flag("order")),
		PageSize:  limit,
		PageToken: ctx.Flag("page-token"),
		Rung:      ctx.Flag("rung"),
		Category:  ctx.Flag("category"),
		Recompute: ctx.BoolFlag("recompute"),
	}
	if v := strings.TrimSpace(ctx.Flag("min-score")); v != "" {
		parsed, err := parseInt32(v)
		if err != nil {
			return fmt.Errorf("--min-score must be an integer: %w", err)
		}
		req.MinScore = &parsed
	}
	if v := strings.TrimSpace(ctx.Flag("max-score")); v != "" {
		parsed, err := parseInt32(v)
		if err != nil {
			return fmt.Errorf("--max-score must be an integer: %w", err)
		}
		req.MaxScore = &parsed
	}
	resp, err := h.client.ListScores(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("list scores", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no score list response")
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	_, err = fmt.Fprint(ctx.Stdout(), FormatList(resp.Msg))
	return err
}

func parseOptionalInt32Flag(ctx cliapp.RunContext, name string) (int32, error) {
	value := strings.TrimSpace(ctx.Flag(name))
	if value == "" {
		return 0, nil
	}
	parsed, err := parseInt32(value)
	if err != nil {
		return 0, fmt.Errorf("--%s must be an integer: %w", name, err)
	}
	return parsed, nil
}

func parseInt32(value string) (int32, error) {
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(parsed), nil
}

func sortByFlag(value string) scoringv1.ScoreSortBy {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "rung":
		return scoringv1.ScoreSortBy_SCORE_SORT_BY_RUNG
	case "last-scored", "last_scored", "freshness":
		return scoringv1.ScoreSortBy_SCORE_SORT_BY_LAST_SCORED
	case "scenario", "name":
		return scoringv1.ScoreSortBy_SCORE_SORT_BY_SCENARIO
	case "priority":
		return scoringv1.ScoreSortBy_SCORE_SORT_BY_PRIORITY
	default:
		return scoringv1.ScoreSortBy_SCORE_SORT_BY_COMPOSITE
	}
}

func sortOrderFlag(value string) scoringv1.SortOrder {
	if strings.EqualFold(strings.TrimSpace(value), "asc") {
		return scoringv1.SortOrder_SORT_ORDER_ASC
	}
	return scoringv1.SortOrder_SORT_ORDER_DESC
}
