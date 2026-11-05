package measures

import (
	"context"
	"errors"
	"sort"
	"time"

	"agent-manager/internal/usageattribution"
	"agent-manager/internal/usagemodes"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/cli-core/usagemode"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

// SetUsageModes connects the usage mode service. Without it the mode
// procedures report unavailable.
func (h *Handler) SetUsageModes(service *usagemodes.Service) { h.usageModes = service }

// RebaseResult is one run's outcome of an explicit usage-mode rebase.
type RebaseResult struct {
	RunID                 string
	Status                string
	Reason                string
	FromModel             string
	ToModel               string
	EstimatedRereadTokens int64
}

// RunRebaser moves managed runs to a mode's lower model at their next resume.
// Agent Manager's orchestrator implements it through the wiring adapter.
type RunRebaser interface {
	RebaseRuns(ctx context.Context, modeID string, runIDs []string, dryRun bool) ([]RebaseResult, error)
}

// SetRunRebaser connects explicit active-run switching. Without it
// RebaseUsageModeRuns reports unavailable.
func (h *Handler) SetRunRebaser(rebaser RunRebaser) { h.runRebaser = rebaser }

func (h *Handler) ExtendUsageMode(ctx context.Context, req *connect.Request[measurepb.ExtendUsageModeRequest]) (*connect.Response[measurepb.ExtendUsageModeResponse], error) {
	if _, err := requireModeOperator(ctx); err != nil {
		return nil, err
	}
	service, err := h.modes()
	if err != nil {
		return nil, err
	}
	mode, err := service.Extend(ctx, req.Msg.GetId(), req.Msg.GetTo())
	if err != nil {
		return nil, modeError(err)
	}
	return connect.NewResponse(&measurepb.ExtendUsageModeResponse{Mode: protoUsageMode(mode)}), nil
}

func (h *Handler) RebaseUsageModeRuns(ctx context.Context, req *connect.Request[measurepb.RebaseUsageModeRunsRequest]) (*connect.Response[measurepb.RebaseUsageModeRunsResponse], error) {
	if _, err := requireModeOperator(ctx); err != nil {
		return nil, err
	}
	if h.runRebaser == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("switching running work is unavailable"))
	}
	results, err := h.runRebaser.RebaseRuns(ctx, req.Msg.GetModeId(), req.Msg.GetRunIds(), req.Msg.GetDryRun())
	if err != nil {
		return nil, modeError(err)
	}
	response := &measurepb.RebaseUsageModeRunsResponse{DryRun: req.Msg.GetDryRun()}
	for _, result := range results {
		response.Results = append(response.Results, &measurepb.UsageModeRebaseResult{
			RunId: result.RunID, Status: result.Status, Reason: result.Reason, FromModel: result.FromModel, ToModel: result.ToModel,
			EstimatedRereadTokens: result.EstimatedRereadTokens,
		})
	}
	return connect.NewResponse(response), nil
}

func (h *Handler) modes() (*usagemodes.Service, error) {
	if h.usageModes == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("usage modes are unavailable"))
	}
	return h.usageModes, nil
}

// requireModeOperator is the standing-grant write gate: a verified independent
// human with agent-manager:write. Agent identities and services are refused.
func requireModeOperator(ctx context.Context) (string, error) {
	principal, err := authn.RequireIndependentHuman(ctx)
	if err == nil {
		_, err = authn.RequireCapability(ctx, "agent-manager:write")
	}
	if err != nil {
		return "", connect.NewError(connect.CodePermissionDenied, errors.New("usage modes can be started or ended only by a verified operator, not an agent"))
	}
	return principal.Subject, nil
}

func modeError(err error) error {
	if errors.Is(err, usagemodes.ErrInvalid) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func (h *Handler) StartUsageMode(ctx context.Context, req *connect.Request[measurepb.StartUsageModeRequest]) (*connect.Response[measurepb.StartUsageModeResponse], error) {
	operator, err := requireModeOperator(ctx)
	if err != nil {
		return nil, err
	}
	service, err := h.modes()
	if err != nil {
		return nil, err
	}
	m := req.Msg
	result, err := service.Start(ctx, usagemodes.StartRequest{
		Preset: m.GetPreset(), Model: m.GetModel(), Effort: m.GetEffort(), ContextLimitTokens: int(m.GetContextLimitTokens()),
		Runner: m.GetRunner(), Role: m.GetRole(), Profile: m.GetProfile(), Until: m.GetUntil(), Label: m.GetLabel(), StartedBy: operator,
	})
	if err != nil {
		return nil, modeError(err)
	}
	return connect.NewResponse(&measurepb.StartUsageModeResponse{Mode: protoUsageMode(result.Mode), ReplacedIds: result.ReplacedIDs}), nil
}

func (h *Handler) EndUsageMode(ctx context.Context, req *connect.Request[measurepb.EndUsageModeRequest]) (*connect.Response[measurepb.EndUsageModeResponse], error) {
	if _, err := requireModeOperator(ctx); err != nil {
		return nil, err
	}
	service, err := h.modes()
	if err != nil {
		return nil, err
	}
	ended, missing, err := service.End(req.Msg.GetIds(), req.Msg.GetAll())
	if err != nil {
		return nil, modeError(err)
	}
	return connect.NewResponse(&measurepb.EndUsageModeResponse{EndedIds: ended, NotFoundIds: missing}), nil
}

// reachNotes say plainly what a mode can and cannot change.
var reachNotes = []string{
	"Agent Manager runs with a role: a mode applies to each new run and is recorded as usage_mode:<id> in its selection reason. Running and resumed runs keep their settings.",
	"Operator sessions started through the harness shim or `vrooli agent launch`: a mode applies on the next launch only; open sessions are not changed. Role and profile scopes do not reach them.",
	"Desktop apps and harnesses started outside the shim cannot be reached; their use is still imported and measured, and it may sit in the unattributed share.",
	"Points are estimates from per-window calibration. The burn shown is observed use since the mode started, not a saving.",
}

func (h *Handler) ListUsageModes(ctx context.Context, _ *connect.Request[measurepb.ListUsageModesRequest]) (*connect.Response[measurepb.ListUsageModesResponse], error) {
	service, err := h.modes()
	if err != nil {
		return nil, err
	}
	now := h.now().UTC()
	response := &measurepb.ListUsageModesResponse{GeneratedAt: now.Format(time.RFC3339), ReachNotes: reachNotes, ModeFile: service.File()}
	if h.ledger != nil {
		// Ended modes stay on record, so a window chart can shade them too.
		if periods, err := h.ledger.ModePeriods(ctx); err == nil {
			labels := map[string]string{}
			for _, active := range service.List() {
				labels[active.ID] = active.Label
			}
			response.Periods = protoModePeriods(periods, labels, now.Add(-35*24*time.Hour), now)
		}
	}
	attribution, attributionErr := h.currentAttribution(ctx, time.Time{}, now)
	for _, mode := range service.List() {
		status := &measurepb.UsageModeStatus{Mode: protoUsageMode(mode)}
		if mode.ExpiresAt != nil {
			status.SecondsLeft = int64(mode.ExpiresAt.Sub(now).Seconds())
		}
		if burn, err := h.modeBurn(ctx, mode, now); err == nil {
			status.Burn = burn
		}
		response.Modes = append(response.Modes, status)
	}
	if attributionErr == nil {
		for _, result := range attribution.Results {
			reach := usageattribution.ReachOf(result, attribution.Runs)
			response.Reach = append(response.Reach, protoReach(result, reach))
		}
	}
	return connect.NewResponse(response), nil
}

func (h *Handler) currentAttribution(ctx context.Context, since, now time.Time) (usageattribution.Attribution, error) {
	service, err := h.usageService()
	if err != nil {
		return usageattribution.Attribution{}, err
	}
	return service.Current(ctx, "", since, now)
}

// modeBurn is each window's observed increase since the mode started, from the
// runner providers the mode covers (the same attribution `usage burners
// --since` reads).
func (h *Handler) modeBurn(ctx context.Context, mode usagemode.Mode, now time.Time) ([]*measurepb.UsageModeBurn, error) {
	attribution, err := h.currentAttribution(ctx, mode.StartedAt, now)
	if err != nil {
		return nil, err
	}
	covered := map[string]bool{}
	for runner := range mode.Runners {
		covered[usageattribution.ProviderFor(usagemode.NormalizeRunner(runner))] = true
	}
	if mode.Learn != nil {
		for runner := range mode.Learn.Runners {
			covered[usageattribution.ProviderFor(usagemode.NormalizeRunner(runner))] = true
		}
	}
	var burn []*measurepb.UsageModeBurn
	for _, result := range attribution.Results {
		if !covered[result.Window.Provider] {
			continue
		}
		burn = append(burn, &measurepb.UsageModeBurn{
			Provider: result.Window.Provider, Pool: result.Window.Pool, Window: result.Window.Name,
			ObservedIncreasePoints: result.Observed, AttributedPoints: result.Attributed(), UnattributedPoints: result.Unattributed,
		})
	}
	return burn, nil
}

func protoReach(result usageattribution.Result, reach usageattribution.Reach) *measurepb.UsageModeReachRow {
	row := &measurepb.UsageModeReachRow{
		Provider: result.Window.Provider, Pool: result.Window.Pool, Window: result.Window.Name, ResetsAt: rfc3339(result.Window.ResetAt),
		ObservedIncreasePoints: reach.Observed, OperatorPoints: reach.Operator, ProfilePoints: reach.Profile, UnattributedPoints: reach.Unattributed,
	}
	if reach.Observed > 0 {
		row.OperatorShare, row.ProfileShare, row.UnattributedShare = reach.Operator/reach.Observed, reach.Profile/reach.Observed, reach.Unattributed/reach.Observed
	}
	return row
}

func protoUsageMode(m usagemode.Mode) *measurepb.UsageMode {
	out := &measurepb.UsageMode{
		Id: m.ID, Label: m.Label, Kind: m.Kind, StartedAt: m.StartedAt.UTC().Format(time.RFC3339), StartedBy: m.StartedBy, Source: m.Source,
		Scope:     &measurepb.UsageModeScope{Runner: m.Scope.Runner, Role: m.Scope.Role, Profile: m.Scope.Profile},
		UntilKind: m.Until.Kind, UntilProvider: m.Until.Provider, UntilWindowMinutes: int64(m.Until.WindowMinutes), ExemptRoles: m.ExemptRoles,
	}
	if m.ExpiresAt != nil {
		out.ExpiresAt = m.ExpiresAt.UTC().Format(time.RFC3339)
	}
	runners := make([]string, 0, len(m.Runners))
	for runner := range m.Runners {
		runners = append(runners, runner)
	}
	sort.Strings(runners)
	for _, runner := range runners {
		s := m.Runners[runner]
		out.Settings = append(out.Settings, protoModeSettings(runner, s))
	}
	if m.Learn != nil {
		learn := &measurepb.UsageModeLearn{SplitPercent: int32(m.Learn.Split() * 100), Paused: m.Learn.Paused, PausedReason: m.Learn.PausedReason}
		if m.Learn.Sessions != nil && m.Learn.Sessions.OptIn {
			learn.SessionsOptIn, learn.BlockDays = true, int32(max(m.Learn.Sessions.BlockDays, usagemode.DefaultBlockDays))
		}
		keys := make([]string, 0, len(m.Learn.Runners))
		for runner := range m.Learn.Runners {
			keys = append(keys, runner)
		}
		sort.Strings(keys)
		for _, runner := range keys {
			learn.Challenger = append(learn.Challenger, protoModeSettings(runner, m.Learn.Runners[runner]))
		}
		out.Learn = learn
	}
	return out
}

func protoModeSettings(runner string, s usagemode.RunnerSettings) *measurepb.UsageModeSettings {
	return &measurepb.UsageModeSettings{Runner: runner, ModelMap: s.ModelMap, SessionModel: s.SessionModel, Effort: s.Effort, ContextLimitTokens: int64(s.ContextLimitTokens)}
}
