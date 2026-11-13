package rangecli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	rangev1 "github.com/vrooli/vrooli/packages/proto/gen/go/extraction-range/v1/range"
	rangeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/extraction-range/v1/range/range_v1connect"
)

type handlers struct {
	client rangeconnect.RangeServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	client, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: rangeconnect.NewRangeServiceClient(client, base)}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*rangev1.ListLevelsResponse, error) {
	r, err := h.client.ListLevels(context.Background(), connect.NewRequest(&rangev1.ListLevelsRequest{Family: ctx.Flag("family")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list range levels", err, nil)
	}
	if r == nil || r.Msg == nil {
		return nil, fmt.Errorf("server returned no levels")
	}
	return r.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, m *rangev1.ListLevelsResponse) cliapp.ListReport {
	out := make([]string, 0, len(m.Levels))
	for _, l := range m.Levels {
		out = append(out, fmt.Sprintf("%s · %s · difficulty %d — %s", l.Id, l.Family, l.Difficulty, l.Title))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d practice level(s).", len(out))}, ResultsHeading: "Levels", Results: out, RetrievalHints: []string{"`range get <id>` — view a task brief and target URL", "`range scoreboard` — see best tool scores"}}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*rangev1.GetLevelResponse, error) {
	id := ctx.Positional("id")
	r, err := h.client.GetLevel(context.Background(), connect.NewRequest(&rangev1.GetLevelRequest{Id: id}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get range level", err, nil)
	}
	if r == nil || r.Msg == nil || r.Msg.Level == nil {
		return nil, fmt.Errorf("server returned no level")
	}
	return r.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, m *rangev1.GetLevelResponse) cliapp.ListReport {
	l := m.Level
	return cliapp.ListReport{Summary: []string{l.Title, "Family: " + l.Family, fmt.Sprintf("Difficulty: %d", l.Difficulty)}, ResultsHeading: "Task", Results: []string{l.Task, "Target: " + m.TargetUrl}}
}

func (h *handlers) submitCall(ctx cliapp.OperationContext) (*rangev1.SubmitResponse, error) {
	b, err := os.ReadFile(ctx.Flag("file"))
	if err != nil {
		return nil, fmt.Errorf("read records file: %w", err)
	}
	var records []*rangev1.ExtractedRecord
	if err := json.Unmarshal(b, &records); err != nil {
		return nil, fmt.Errorf("decode records JSON array: %w", err)
	}
	r, err := h.client.Submit(context.Background(), connect.NewRequest(&rangev1.SubmitRequest{Tool: strings.TrimSpace(ctx.Flag("tool")), LevelId: ctx.Positional("level_id"), Records: records}))
	if err != nil {
		return nil, cliapp.WrapAPIError("submit range results", err, nil)
	}
	if r == nil || r.Msg == nil {
		return nil, fmt.Errorf("server returned no score")
	}
	return r.Msg, nil
}

func (h *handlers) submitReport(_ cliapp.OperationContext, m *rangev1.SubmitResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("%s on %s: F1 %.3f (%d/%d records); passed=%t", m.Tool, m.LevelId, m.F1, m.Correct, m.Expected, m.Passed)}, Changes: []string{fmt.Sprintf("precision %.3f · recall %.3f", m.Precision, m.Recall)}, NextCommand: []string{"`range scoreboard` — review your tool's best scores"}}
}

func (h *handlers) scoreboardCall(ctx cliapp.OperationContext) (*rangev1.ScoreboardResponse, error) {
	r, err := h.client.Scoreboard(context.Background(), connect.NewRequest(&rangev1.ScoreboardRequest{Tool: ctx.Flag("tool"), Family: ctx.Flag("family")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("read range scoreboard", err, nil)
	}
	if r == nil || r.Msg == nil {
		return nil, fmt.Errorf("server returned no scoreboard")
	}
	return r.Msg, nil
}

func (h *handlers) scoreboardReport(_ cliapp.OperationContext, m *rangev1.ScoreboardResponse) cliapp.ListReport {
	rows := make([]string, 0, len(m.Scores))
	for _, s := range m.Scores {
		rows = append(rows, fmt.Sprintf("%s · %s · F1 %.3f (%d/%d) · passed=%t", s.Tool, s.LevelId, s.F1, s.Correct, s.Expected, s.Passed))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d tool/level score(s).", len(rows))}, ResultsHeading: "Scoreboard", Results: rows}
}
