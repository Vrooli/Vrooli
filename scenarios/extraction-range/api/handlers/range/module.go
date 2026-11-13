package rangehandler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"connectrpc.com/connect"
	"extraction-range/internal/module"
	rangescore "extraction-range/internal/range"
	"extraction-range/internal/rangeengine"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	rangev1 "github.com/vrooli/vrooli/packages/proto/gen/go/extraction-range/v1/range"
	rangeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/extraction-range/v1/range/range_v1connect"
)

type Handler struct {
	engine             *rangeengine.Engine
	logger             *log.Logger
	mu                 sync.Mutex
	resilienceRequests map[string]int
	sleep              func(time.Duration)
	scores             rangescore.Repository
}

func New(db ...*database.RoutedDB) (*Handler, error) {
	engine, err := rangeengine.New()
	if err != nil {
		return nil, err
	}
	h := &Handler{engine: engine, logger: log.Default(), resilienceRequests: map[string]int{}, sleep: time.Sleep}
	if len(db) > 0 {
		h.scores = rangescore.NewRepository(db[0])
	}
	return h, nil
}

func (h *Handler) Module() module.Module {
	path, connectHandler := rangeconnect.NewRangeServiceHandler(h)
	return module.Module{Name: "range", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: connectHandler})
	}, Endpoints: Endpoints}
}

func (h *Handler) ListLevels(_ context.Context, req *connect.Request[rangev1.ListLevelsRequest]) (*connect.Response[rangev1.ListLevelsResponse], error) {
	response := &rangev1.ListLevelsResponse{Levels: []*rangev1.LevelSummary{}}
	for _, level := range h.engine.List() {
		if req.Msg.GetFamily() != "" && req.Msg.GetFamily() != level.Family {
			continue
		}
		response.Levels = append(response.Levels, summary(level))
	}
	return connect.NewResponse(response), nil
}

func (h *Handler) GetLevel(_ context.Context, req *connect.Request[rangev1.GetLevelRequest]) (*connect.Response[rangev1.GetLevelResponse], error) {
	level, ok := h.engine.Get(req.Msg.GetId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("unknown level %q", req.Msg.GetId()))
	}
	return connect.NewResponse(&rangev1.GetLevelResponse{Level: summary(level), TargetUrl: "/practice/" + level.ID}), nil
}

func (h *Handler) Submit(ctx context.Context, req *connect.Request[rangev1.SubmitRequest]) (*connect.Response[rangev1.SubmitResponse], error) {
	items := make([]rangeengine.Record, 0, len(req.Msg.GetRecords()))
	for _, record := range req.Msg.GetRecords() {
		if record == nil {
			continue
		}
		items = append(items, rangeengine.Record{"id": record.GetId(), "name": record.GetName(), "category": record.GetCategory(), "value": record.GetValue(), "active": record.GetActive()})
	}
	result, err := h.engine.Score(req.Msg.GetTool(), req.Msg.GetLevelId(), items)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if h.scores != nil {
		if err := h.scores.SaveBest(ctx, result); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("save score: %w", err))
		}
	}
	return connect.NewResponse(&rangev1.SubmitResponse{Tool: result.Tool, LevelId: result.LevelID, Family: result.Family, Correct: int32(result.Correct), Submitted: int32(result.Submitted), Expected: int32(result.Expected), Precision: result.Precision, Recall: result.Recall, F1: result.F1, Passed: result.Passed}), nil
}

func (h *Handler) Scoreboard(ctx context.Context, req *connect.Request[rangev1.ScoreboardRequest]) (*connect.Response[rangev1.ScoreboardResponse], error) {
	response := &rangev1.ScoreboardResponse{Scores: []*rangev1.SubmitResponse{}}
	results := h.engine.Scoreboard()
	if h.scores != nil {
		var err error
		results, err = h.scores.List(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load scoreboard: %w", err))
		}
	}
	for _, result := range results {
		if req.Msg.GetTool() != "" && req.Msg.GetTool() != result.Tool {
			continue
		}
		if req.Msg.GetFamily() != "" && req.Msg.GetFamily() != result.Family {
			continue
		}
		response.Scores = append(response.Scores, &rangev1.SubmitResponse{Tool: result.Tool, LevelId: result.LevelID, Family: result.Family, Correct: int32(result.Correct), Submitted: int32(result.Submitted), Expected: int32(result.Expected), Precision: result.Precision, Recall: result.Recall, F1: result.F1, Passed: result.Passed})
	}
	return connect.NewResponse(response), nil
}

func summary(level rangeengine.Level) *rangev1.LevelSummary {
	return &rangev1.LevelSummary{Id: level.ID, Family: level.Family, FamilyName: level.FamilyName, Difficulty: int32(level.Difficulty), Title: level.Title, Task: level.Task, Tags: append([]string(nil), level.Tags...)}
}
