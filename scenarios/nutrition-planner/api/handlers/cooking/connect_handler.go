package cooking

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/identity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/cooking"

	internal "nutrition-planner/internal/cooking"
	"nutrition-planner/internal/decimalx"
	"nutrition-planner/internal/recipe"
	"nutrition-planner/internal/workspace"
)

type connectHandler struct {
	repo       internal.Repository
	recipes    recipe.Service
	workspaces workspace.Service
	logger     *log.Logger
}

func NewConnectHandler(repo internal.Repository, recipes recipe.Service, workspaces workspace.Service, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{repo: repo, recipes: recipes, workspaces: workspaces, logger: logger}
}

func (h *connectHandler) scope(ctx context.Context, w string) error {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if w == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	if _, err := h.workspaces.Get(ctx, w, p.Subject); err != nil {
		var nf workspace.ErrNotFound
		var forbidden workspace.ErrForbidden
		if errors.As(err, &nf) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		if errors.As(err, &forbidden) {
			return connect.NewError(connect.CodePermissionDenied, err)
		}
		return connect.NewError(connect.CodeInternal, err)
	}
	return nil
}

func (h *connectHandler) StartSession(ctx context.Context, req *connect.Request[v1.StartSessionRequest]) (*connect.Response[v1.SessionResponse], error) {
	m := req.Msg
	if err := h.scope(ctx, m.WorkspaceId); err != nil {
		return nil, err
	}
	scale, err := decimalx.Parse(m.Scale)
	if err != nil || scale.IsUnknown() || scale.IsZero() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scale must be a known positive amount"))
	}
	zero, _ := decimalx.Parse("0")
	if cmp, _ := decimalx.Compare(scale, zero); cmp <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scale must be positive"))
	}
	r, err := h.recipes.GetRevision(ctx, m.RecipeId, m.WorkspaceId, m.RecipeRevision)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	found := false
	for _, method := range r.Methods {
		if method.ID == m.MethodId {
			found = true
			break
		}
	}
	if !found {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("method_id is not present in the pinned recipe revision"))
	}
	s, err := h.repo.Create(ctx, internal.Session{ID: m.SessionId, WorkspaceID: m.WorkspaceId, RecipeID: m.RecipeId, RecipeRevision: m.RecipeRevision, MethodID: m.MethodId, Scale: m.Scale, StartedAt: time.Now().UTC()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.SessionResponse{Session: toProto(s)}), nil
}

func (h *connectHandler) GetSession(ctx context.Context, req *connect.Request[v1.GetSessionRequest]) (*connect.Response[v1.SessionResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	s, err := h.repo.Get(ctx, req.Msg.WorkspaceId, req.Msg.SessionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&v1.SessionResponse{Session: toProto(s)}), nil
}

func (h *connectHandler) ListSessions(ctx context.Context, req *connect.Request[v1.ListSessionsRequest]) (*connect.Response[v1.ListSessionsResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	ss, err := h.repo.List(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListSessionsResponse{Sessions: make([]*v1.CookingSession, 0, len(ss))}
	for _, s := range ss {
		out.Sessions = append(out.Sessions, toProto(s))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) SaveSession(ctx context.Context, req *connect.Request[v1.SaveSessionRequest]) (*connect.Response[v1.SessionResponse], error) {
	m := req.Msg
	if err := h.scope(ctx, m.WorkspaceId); err != nil {
		return nil, err
	}
	s, err := h.repo.Get(ctx, m.WorkspaceId, m.SessionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	recipeRevision, err := h.recipes.GetRevision(ctx, s.RecipeID, m.WorkspaceId, s.RecipeRevision)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("pinned recipe revision is unavailable"))
	}
	var selected *recipe.Method
	for i := range recipeRevision.Methods {
		if recipeRevision.Methods[i].ID == s.MethodID {
			selected = &recipeRevision.Methods[i]
			break
		}
	}
	if selected == nil || m.CurrentStepIndex < 0 || int(m.CurrentStepIndex) > len(selected.Steps) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("current step is outside the pinned method"))
	}
	validSteps := map[string]bool{}
	for _, step := range selected.Steps {
		validSteps[step.ID] = true
	}
	for _, id := range m.CompletedSteps {
		if !validSteps[id] {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("completed step is outside the pinned method"))
		}
	}
	s.CurrentStepIndex = int(m.CurrentStepIndex)
	s.CompletedSteps = append([]string(nil), m.CompletedSteps...)
	s.Timers = make([]internal.Timer, 0, len(m.Timers))
	for _, t := range m.Timers {
		if t.Id == "" || !validSteps[t.StepId] || t.DurationSeconds <= 0 || t.ElapsedSeconds < 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("timer must reference a pinned step and have a positive duration"))
		}
		started, e := time.Parse(time.RFC3339Nano, t.StartedAt)
		if e != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("timer started_at must be RFC3339"))
		}
		timer := internal.Timer{ID: t.Id, StepID: t.StepId, DurationSeconds: t.DurationSeconds, StartedAt: started, ElapsedSeconds: t.ElapsedSeconds}
		if t.PausedAt != "" {
			timer.PausedAt, e = time.Parse(time.RFC3339Nano, t.PausedAt)
			if e != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("timer paused_at must be RFC3339"))
			}
		}
		s.Timers = append(s.Timers, timer)
	}
	if m.Finish {
		s.Status = "finished"
		s.ActualYield = m.ActualYield
		s.YieldUnit = m.YieldUnit
		if m.ActualYield != "" {
			amount, e := decimalx.Parse(m.ActualYield)
			if e != nil || amount.IsUnknown() || amount.IsZero() || m.YieldUnit == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("actual yield needs a known positive amount and unit"))
			}
			zero, _ := decimalx.Parse("0")
			if cmp, _ := decimalx.Compare(amount, zero); cmp <= 0 {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("actual yield must be positive"))
			}
		}
	} else if m.ActualYield != "" || m.YieldUnit != "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("yield can only be recorded while finishing"))
	}
	canonical, _ := json.Marshal(m)
	sum := sha256.Sum256(canonical)
	saved, err := h.repo.Save(ctx, m.WorkspaceId, m.EventId, hex.EncodeToString(sum[:]), s, m.ExpectedVersion)
	if err != nil {
		return nil, connect.NewError(connect.CodeAborted, err)
	}
	return connect.NewResponse(&v1.SessionResponse{Session: toProto(saved)}), nil
}

func toProto(s internal.Session) *v1.CookingSession {
	out := &v1.CookingSession{Id: s.ID, WorkspaceId: s.WorkspaceID, RecipeId: s.RecipeID, RecipeRevision: s.RecipeRevision, MethodId: s.MethodID, Scale: s.Scale, CurrentStepIndex: int32(s.CurrentStepIndex), CompletedSteps: s.CompletedSteps, Status: s.Status, ActualYield: s.ActualYield, YieldUnit: s.YieldUnit, Version: s.Version, StartedAt: s.StartedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: s.UpdatedAt.UTC().Format(time.RFC3339Nano)}
	if !s.FinishedAt.IsZero() {
		out.FinishedAt = s.FinishedAt.UTC().Format(time.RFC3339Nano)
	}
	for _, t := range s.Timers {
		pt := ""
		if !t.PausedAt.IsZero() {
			pt = t.PausedAt.UTC().Format(time.RFC3339Nano)
		}
		out.Timers = append(out.Timers, &v1.Timer{Id: t.ID, StepId: t.StepID, DurationSeconds: t.DurationSeconds, StartedAt: t.StartedAt.UTC().Format(time.RFC3339Nano), PausedAt: pt, ElapsedSeconds: t.ElapsedSeconds})
	}
	return out
}
