package autofiler

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/settings"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

type itemLoader interface {
	LoadItem(kind backlog.BacklogKind, name string) (backlog.BacklogItem, error)
}

type ConnectService struct {
	settings   SettingsProvider
	sweeper    *Sweeper
	items      itemLoader
	reconciler BacklogReconciler
	dismissals *DismissalStore
}

const defaultRunNowTimeout = 45 * time.Second

func NewConnectService(settings SettingsProvider, sweeper *Sweeper, items itemLoader, reconciler BacklogReconciler, dismissals *DismissalStore) *ConnectService {
	return &ConnectService{
		settings:   settings,
		sweeper:    sweeper,
		items:      items,
		reconciler: reconciler,
		dismissals: dismissals,
	}
}

func RegisterConnectService(router *mux.Router, svc *ConnectService) {
	path, handler := apiconnect.NewAutoFilerServiceHandler(svc)
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}

var _ apiconnect.AutoFilerServiceHandler = (*ConnectService)(nil)

func (s *ConnectService) GetStatus(_ context.Context, _ *connect.Request[apipb.AutoFilerStatusRequest]) (*connect.Response[apipb.AutoFilerStatusResponse], error) {
	resp, err := s.statusResponse()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectService) RunNow(ctx context.Context, _ *connect.Request[apipb.AutoFilerRunNowRequest]) (*connect.Response[apipb.AutoFilerStatusResponse], error) {
	if s.sweeper == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("auto-filer sweeper is not configured"))
	}
	runCtx, cancel := context.WithTimeout(ctx, defaultRunNowTimeout)
	defer cancel()
	_, runErr := s.sweeper.RunOnce(runCtx)
	resp, err := s.statusResponse()
	if err != nil {
		if runErr != nil {
			return nil, connect.NewError(connect.CodeInternal, runErr)
		}
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectService) statusResponse() (*apipb.AutoFilerStatusResponse, error) {
	cfg := settings.DefaultSettings().AutoFiler
	if s.settings != nil {
		loaded, err := s.settings.LoadAutoFiler()
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		cfg = loaded
	}
	result := SweepResult{}
	if s.sweeper != nil {
		result = s.sweeper.LastResult()
	}
	open := result.OpenAutoFiled
	remaining := result.RemainingBudget
	if s.sweeper != nil && s.sweeper.Backlog != nil {
		if budget, err := RemainingBudget(s.sweeper.Backlog, cfg.MaxOpenAutoFiled); err == nil {
			remaining = budget
			open = cfg.MaxOpenAutoFiled - budget
		}
	}
	dismissalCount := 0
	if s.dismissals != nil {
		if count, err := s.dismissals.Count(); err == nil {
			dismissalCount = count
		}
	}
	resp := &apipb.AutoFilerStatusResponse{
		Enabled:          cfg.Enabled,
		Mode:             cfg.Mode,
		Strategy:         cfg.Strategy,
		LastCycleTime:    formatTime(result.RanAt),
		LastError:        result.LastError,
		Candidates:       int32(result.Candidates),
		Findings:         int32(result.Findings),
		Created:          int32(result.Created),
		SkippedDismissed: int32(result.SkippedDismissed),
		OpenAutoFiled:    int32(open),
		MaxOpenAutoFiled: int32(cfg.MaxOpenAutoFiled),
		RemainingBudget:  int32(remaining),
		Brake:            brakeToProto(result.Brake),
		DismissalCount:   int32(dismissalCount),
		ReconciledClosed: int32(result.ReconciledClosed),
		ReconciledNoted:  int32(result.ReconciledNoted),
	}
	return resp, nil
}

func (s *ConnectService) DismissSuggestion(ctx context.Context, req *connect.Request[apipb.DismissAutoFilerSuggestionRequest]) (*connect.Response[apipb.DismissAutoFilerSuggestionResponse], error) {
	if req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if s.items == nil || s.reconciler == nil || s.dismissals == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("auto-filer dismissal is not configured"))
	}
	kind, err := backlog.ParseBacklogKind(req.Msg.GetKind())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	item, err := s.items.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, backlog.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("backlog item not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if item.Status != backlog.StatusSuggested {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("only suggested auto-filer items can be dismissed"))
	}
	if !IsAutoFiled(item) || strings.TrimSpace(item.FindingRef) == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("item is not an auto-filed suggestion"))
	}
	reason := strings.TrimSpace(req.Msg.GetReason())
	if reason == "" {
		reason = "Dismissed by operator; auto-filer will remember this finding."
	}
	if err := s.dismissals.Remember(item.FindingRef, ItemRef(item), reason); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	archived, err := s.reconciler.ArchiveItem(ctx, kind, name, reason)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&apipb.DismissAutoFilerSuggestionResponse{
		Item:      backlog.ToProto(archived),
		Dismissed: true,
	}), nil
}

func brakeToProto(state BrakeState) *apipb.AutoFilerBrakeState {
	return &apipb.AutoFilerBrakeState{
		WindowDays:     int32(state.WindowDays),
		Minimum:        int32(state.Minimum),
		Observed:       int32(state.Observed),
		Braked:         state.Braked,
		WindowStartUtc: formatTime(state.WindowStartUTC),
		WindowEndUtc:   formatTime(state.WindowEndUTC),
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
