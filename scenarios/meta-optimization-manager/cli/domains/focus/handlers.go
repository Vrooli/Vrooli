package focus

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	focusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/focus"
	focusconnect "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/focus/focus_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/shared"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client focusconnect.FocusServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: focusconnect.NewFocusServiceClient(httpClient, baseURL)}
}

func (h *handlers) next(ctx cliapp.RunContext) error {
	proj, err := projectionFlag(ctx)
	if err != nil {
		return err
	}
	limit, err := limitFlag(ctx)
	if err != nil {
		return err
	}
	resp, err := h.client.GetFocus(context.Background(), connect.NewRequest(&focusv1.GetFocusRequest{
		Limit:      limit,
		Projection: proj,
	}))
	if err != nil {
		return cliapp.WrapAPIError("focus next", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no focus response")
	}
	results := make([]string, 0, len(resp.Msg.Items))
	for _, it := range resp.Msg.Items {
		results = append(results, formatFocusItem(it))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d ranked next-best gap(s).", len(resp.Msg.Items))},
		ResultsHeading: "Focus",
		Results:        results,
		RetrievalHints: []string{
			"`gaps show <id>` — full qualitative context for a gap",
			"`gaps note <id> --add \"<approach>\"` — store an explored approach",
		},
	})
}

func (h *handlers) gapsList(ctx cliapp.RunContext) error {
	proj, err := projectionFlag(ctx)
	if err != nil {
		return err
	}
	status, err := statusFlag(ctx)
	if err != nil {
		return err
	}
	resp, err := h.client.ListGaps(context.Background(), connect.NewRequest(&focusv1.ListGapsRequest{
		Projection: proj,
		CellId:     strings.TrimSpace(ctx.Flag("cell")),
		Status:     status,
	}))
	if err != nil {
		return cliapp.WrapAPIError("gaps list", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no gaps response")
	}
	results := make([]string, 0, len(resp.Msg.Gaps))
	for _, g := range resp.Msg.Gaps {
		results = append(results, formatGap(g))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d gap(s).", len(resp.Msg.Gaps))},
		ResultsHeading: "Gaps",
		Results:        results,
		RetrievalHints: []string{"`gaps show <id>` — provenance + approaches for one gap"},
	})
}

func (h *handlers) gapsShow(ctx cliapp.RunContext) error {
	id := strings.TrimSpace(ctx.Positional("id"))
	resp, err := h.client.GetGap(context.Background(), connect.NewRequest(&focusv1.GetGapRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("gaps show %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Gap == nil {
		return fmt.Errorf("server returned no gap")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Gap %s.", resp.Msg.Gap.GetId())},
		ResultsHeading: "Gap",
		Results:        gapDetail(resp.Msg.Gap),
	})
}

func (h *handlers) gapsNote(ctx cliapp.RunContext) error {
	id := strings.TrimSpace(ctx.Positional("id"))
	approach := strings.TrimSpace(ctx.Flag("add"))
	if approach == "" {
		return fmt.Errorf("provide the approach to store via --add \"<approach>\"")
	}
	resp, err := h.client.AddGapNote(context.Background(), connect.NewRequest(&focusv1.AddGapNoteRequest{Id: id, Approach: approach}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("gaps note %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Gap == nil {
		return fmt.Errorf("server returned no gap")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Stored approach on gap %s.", resp.Msg.Gap.GetId())},
		ResultsHeading: "Gap",
		Results:        gapDetail(resp.Msg.Gap),
	})
}

func formatFocusItem(it *focusv1.FocusItem) string {
	g := it.GetGap()
	return fmt.Sprintf("[%.2f] %s [%s] %s — %s", it.GetPriorityScore(), g.GetId(), statusLabel(g.GetStatus()), g.GetTitle(), it.GetRationale())
}

func formatGap(g *focusv1.Gap) string {
	scope := projectionLabel(g.GetProjection())
	if g.GetGlobal() {
		scope = "global"
	}
	return fmt.Sprintf("%s [%s/%s] %s", g.GetId(), scope, statusLabel(g.GetStatus()), g.GetTitle())
}

func gapDetail(g *focusv1.Gap) []string {
	out := []string{formatGap(g)}
	for _, n := range g.GetNotes() {
		out = append(out, "  · note: "+n)
	}
	for _, a := range g.GetApproaches() {
		out = append(out, "  ↳ approach: "+a)
	}
	for _, f := range g.GetFollowUps() {
		out = append(out, "  → follow-up: "+f)
	}
	return out
}

// limitFlag parses the optional --limit flag (0 => server default). A negative
// or non-numeric value is a usage error.
func limitFlag(ctx cliapp.RunContext) (int32, error) {
	raw := strings.TrimSpace(ctx.Flag("limit"))
	if raw == "" {
		return 0, nil
	}
	// ParseInt with bitSize 32 bounds the result to int32 range, so the
	// conversion below cannot overflow (avoids gosec G109).
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid --limit %q (use a non-negative integer)", raw)
	}
	return int32(n), nil
}

// projectionFlag maps --projection to the shared proto enum. Empty => UNSPECIFIED.
func projectionFlag(ctx cliapp.RunContext) (sharedv1.Projection, error) {
	switch strings.ToLower(strings.TrimSpace(ctx.Flag("projection"))) {
	case "":
		return sharedv1.Projection_PROJECTION_UNSPECIFIED, nil
	case "answer":
		return sharedv1.Projection_PROJECTION_ANSWER, nil
	case "validate":
		return sharedv1.Projection_PROJECTION_VALIDATE, nil
	case "guide":
		return sharedv1.Projection_PROJECTION_GUIDE, nil
	default:
		return 0, fmt.Errorf("unknown projection %q (use answer|validate|guide)", ctx.Flag("projection"))
	}
}

func statusFlag(ctx cliapp.RunContext) (sharedv1.CellStatus, error) {
	switch strings.ToLower(strings.TrimSpace(ctx.Flag("status"))) {
	case "":
		return sharedv1.CellStatus_CELL_STATUS_UNSPECIFIED, nil
	case "now":
		return sharedv1.CellStatus_CELL_STATUS_NOW, nil
	case "in_reach", "in-reach":
		return sharedv1.CellStatus_CELL_STATUS_IN_REACH, nil
	case "missing":
		return sharedv1.CellStatus_CELL_STATUS_MISSING, nil
	default:
		return 0, fmt.Errorf("unknown status %q (use now|in_reach|missing)", ctx.Flag("status"))
	}
}

func projectionLabel(p sharedv1.Projection) string {
	switch p {
	case sharedv1.Projection_PROJECTION_ANSWER:
		return "answer"
	case sharedv1.Projection_PROJECTION_VALIDATE:
		return "validate"
	case sharedv1.Projection_PROJECTION_GUIDE:
		return "guide"
	default:
		return "cross-cutting"
	}
}

func statusLabel(s sharedv1.CellStatus) string {
	switch s {
	case sharedv1.CellStatus_CELL_STATUS_NOW:
		return "now"
	case sharedv1.CellStatus_CELL_STATUS_IN_REACH:
		return "in_reach"
	case sharedv1.CellStatus_CELL_STATUS_MISSING:
		return "missing"
	default:
		return "?"
	}
}
