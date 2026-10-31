package measures

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/outcomeledger"
	"agent-manager/internal/usageattribution"
	"agent-manager/internal/usagemodes"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/cli-core/usagemode"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

const (
	// UsageExperiment is the registry name of the Learn comparison measure.
	UsageExperiment = "usage.experiment"
	// pilotDays is how far back a role's pace and variance are measured.
	pilotDays = 30
)

// RoleProposer raises the operator decision for a role-default change and
// reports its state. usagemodes.Notifier implements it.
type RoleProposer interface {
	ProposeRoleDefault(ctx context.Context, p usagemodes.RoleDefaultProposal) (string, error)
	AskState(id string) (state, note string, ok bool)
}

// SetRoleProposer connects role-default proposals. Without it ProposeRoleDefault
// reports unavailable and recommendations show only the exact policy change.
func (h *Handler) SetRoleProposer(proposer RoleProposer) { h.roleProposer = proposer }

// outcomeInputs are the ledger rows and per-run use of a window.
type outcomeInputs struct {
	runs   []outcomeledger.RunFacts
	use    map[string]map[string]outcomeledger.Use
	period usageattribution.PeriodUse
}

// loadOutcomeInputs reads a window through the read model's cache: reading every
// usage row of a long window takes seconds, and each evidence view asks for
// several (usage_outcomes_cache.go).
func (h *Handler) loadOutcomeInputs(ctx context.Context, from, to time.Time) (outcomeInputs, error) {
	ledger, err := h.ledgerReader()
	if err != nil {
		return outcomeInputs{}, err
	}
	service, err := h.usageService()
	if err != nil {
		return outcomeInputs{}, err
	}
	cached, _ := ledger.(*cachedLedger)
	if cached == nil {
		return readOutcomeInputs(ctx, ledger, service, from, to)
	}
	return cached.inputs.get(ctx, h.now().UTC(), from, to, func(ctx context.Context) (outcomeInputs, error) {
		return readOutcomeInputs(ctx, cached.Reader, service, from, to)
	})
}

func readOutcomeInputs(ctx context.Context, ledger outcomeledger.Reader, service usageattribution.Service, from, to time.Time) (outcomeInputs, error) {
	runs, err := ledger.FactsInWindow(ctx, from, to)
	if err != nil {
		return outcomeInputs{}, connect.NewError(connect.CodeInternal, err)
	}
	period, err := service.PeriodUse(ctx, "", from, to)
	if err != nil {
		return outcomeInputs{}, connect.NewError(connect.CodeInternal, err)
	}
	use := make(map[string]map[string]outcomeledger.Use, len(period.Runs))
	for runID, models := range period.Runs {
		use[runID] = make(map[string]outcomeledger.Use, len(models))
		for model, modelUse := range models {
			use[runID][model] = outcomeledger.Use{Points: modelUse.Points, Consumption: usageattribution.Consumption(modelUse.Tokens)}
		}
	}
	return outcomeInputs{runs: runs, use: use, period: period}, nil
}

// rolePilot measures a role's pace and variance from its managed units over
// the last pilotDays. Unknown outcomes never join a comparison, so they do not
// count toward the pace.
func (h *Handler) rolePilot(ctx context.Context, role string, now time.Time) (outcomeledger.Pilot, error) {
	from := now.Add(-pilotDays * 24 * time.Hour)
	in, err := h.loadOutcomeInputs(ctx, from, now)
	if err != nil {
		return outcomeledger.Pilot{}, err
	}
	var managed []outcomeledger.RunFacts
	for _, run := range in.runs {
		if run.Role == role && (run.Path == outcomeledger.PathHeadless || run.Path == outcomeledger.PathInteractive) {
			managed = append(managed, run)
		}
	}
	units, _ := outcomeledger.Assemble(managed, in.use, outcomeledger.ByRole)
	return outcomeledger.PilotFromUnits(units, pilotDays), nil
}

// planPower is the plan simulation: enough that a power near 0.8 is known to about
// 2 points and p-values resolve the corrected level. A plan is cached per pilot, so
// a first read of a new pilot pays for it; operational-stats.md has the full run.
var planPower = outcomeledger.PowerOptions{Sims: 400, Permutations: 399, Resamples: 399}

// planFor is the expected time to the planned look.
func planFor(pilot outcomeledger.Pilot, split int32, effect float64) outcomeledger.Plan {
	share := 0.5
	if split > 0 && split < 100 {
		share = float64(split) / 100
	}
	key := fmt.Sprintf("%.4f|%.4f|%.3f|%.3f|%.1f|%d", pilot.UnitsPerDay, pilot.CostCV, pilot.AcceptedShare, pilot.CompletedShare, share, pilot.Units)
	planCache.Lock()
	defer planCache.Unlock()
	if hit, ok := planCache.plans[key+fmt.Sprintf("|%.2f", effect)]; ok {
		return hit
	}
	cfg := outcomeledger.DefaultExperimentConfig()
	plan := outcomeledger.PlanExperiment(pilot, share, effect, cfg.MinEffect, 0.8, cfg.MinAcceptedPerArm, planPower)
	if len(planCache.plans) > 64 {
		planCache.plans = map[string]outcomeledger.Plan{}
	}
	planCache.plans[key+fmt.Sprintf("|%.2f", effect)] = plan
	return plan
}

// planCache keeps a plan per pilot: the simulation is deterministic for a given
// pilot, and the evidence view asks for it on every refresh. Holding the lock
// through the simulation also stops two requests simulating the same pilot.
var planCache = struct {
	sync.Mutex
	plans map[string]outcomeledger.Plan
}{plans: map[string]outcomeledger.Plan{}}

func protoHorizon(h outcomeledger.Horizon) *measurepb.UsageExperimentHorizon {
	return &measurepb.UsageExperimentHorizon{Reached: h.Reached, Days: h.Days, LowDays: h.LowDays, HighDays: h.HighDays, PerArm: int64(h.PerArm)}
}

func protoPlan(plan outcomeledger.Plan) *measurepb.UsageExperimentPlan {
	cfg := outcomeledger.DefaultExperimentConfig()
	return &measurepb.UsageExperimentPlan{
		UnitsPerDay: plan.Pilot.UnitsPerDay, CostCv: plan.Pilot.CostCV, VarianceAssumed: plan.Pilot.Assumed, PilotUnits: int64(plan.Pilot.Units), Effect: plan.Effect,
		Detect: protoHorizon(plan.Detect), Same: protoHorizon(plan.Same), DetectableIn_90Days: plan.DetectableIn90Days, Power: plan.Power, Notes: plan.Notes,
		MinAcceptedPerArm: int64(cfg.MinAcceptedPerArm), MinEffect: cfg.MinEffect, ReworkMargin: cfg.ReworkMargin,
		AcceptedShare: plan.Pilot.AcceptedShare, CompletedShare: plan.Pilot.CompletedShare, PowerAtLook: plan.PowerAtLook, SameAtLook: plan.SameAtLook,
		CompletedOnly: protoHorizon(plan.CompletedOnly), CompletedOnlyPower: plan.CompletedPower, AlphaPerTest: plan.AlphaPerTest,
	}
}

func (h *Handler) PlanUsageExperiment(ctx context.Context, req *connect.Request[measurepb.PlanUsageExperimentRequest]) (*connect.Response[measurepb.PlanUsageExperimentResponse], error) {
	service, err := h.modes()
	if err != nil {
		return nil, err
	}
	runner, role := usagemode.NormalizeRunner(req.Msg.GetRunner()), strings.TrimSpace(req.Msg.GetRole())
	if (runner != "claude" && runner != "codex") || role == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("pass a runner (claude or codex) and one role"))
	}
	if service.RoleExempt(ctx, runner, role) {
		return connect.NewResponse(&measurepb.PlanUsageExperimentResponse{Refusal: "role " + role + " is exempt from usage modes and is never randomized"}), nil
	}
	pilot, err := h.rolePilot(ctx, role, h.now().UTC())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&measurepb.PlanUsageExperimentResponse{Plan: protoPlan(planFor(pilot, req.Msg.GetSplitPercent(), req.Msg.GetEffect()))}), nil
}

func (h *Handler) StartUsageExperiment(ctx context.Context, req *connect.Request[measurepb.StartUsageExperimentRequest]) (*connect.Response[measurepb.StartUsageExperimentResponse], error) {
	operator, err := requireModeOperator(ctx)
	if err != nil {
		return nil, err
	}
	service, err := h.modes()
	if err != nil {
		return nil, err
	}
	m := req.Msg
	mode, err := service.StartLearn(ctx, usagemodes.LearnRequest{
		Runner: m.GetRunner(), Role: m.GetRole(), Profile: m.GetProfile(), Model: m.GetModel(), Until: m.GetUntil(), Label: m.GetLabel(),
		SplitPercent: int(m.GetSplitPercent()), Sessions: m.GetSessions(), BlockDays: int(m.GetBlockDays()), StartedBy: operator,
	})
	if err != nil {
		return nil, modeError(err)
	}
	response := &measurepb.StartUsageExperimentResponse{Mode: protoUsageMode(mode)}
	if pilot, err := h.rolePilot(ctx, mode.Scope.Role, h.now().UTC()); err == nil {
		response.Plan = protoPlan(planFor(pilot, int32(mode.Learn.SplitPercent), 0))
	}
	response.Notes = append(response.Notes, "New managed runs of "+mode.Scope.Role+" are assigned at random to the incumbent or the challenger and the arm is recorded in the run's policy snapshot.",
		"A save mode that applies, or a forecast that a window will run out, pauses assignment; work already assigned keeps its arm.")
	if mode.Learn.Sessions != nil {
		response.Notes = append(response.Notes, "Operator sessions are included as randomized day blocks and are labelled at launch; a full-power launch is allowed once and counted as an exception in its assigned arm.")
	}
	return connect.NewResponse(response), nil
}

// learnPeriods lists the recorded learn periods, optionally one id.
func (h *Handler) learnPeriods(ctx context.Context, id string) ([]outcomeledger.ModePeriod, error) {
	ledger, err := h.ledgerReader()
	if err != nil {
		return nil, err
	}
	periods, err := ledger.ModePeriods(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var out []outcomeledger.ModePeriod
	for _, period := range periods {
		if period.Kind == usagemode.KindLearn && (id == "" || period.ID == id) {
			out = append(out, period)
		}
	}
	return out, nil
}

func (h *Handler) UsageExperiment(ctx context.Context, req *connect.Request[measurepb.UsageExperimentRequest]) (*connect.Response[measurepb.UsageExperimentResponse], error) {
	now := h.now().UTC()
	periods, err := h.learnPeriods(ctx, strings.TrimSpace(req.Msg.GetModeId()))
	if err != nil {
		return nil, err
	}
	active := map[string]usagemode.Mode{}
	if h.usageModes != nil {
		for _, mode := range h.usageModes.List() {
			active[mode.ID] = mode
		}
	}
	response := &measurepb.UsageExperimentResponse{GeneratedAt: now.Format(time.RFC3339), DefinitionId: definitionID(UsageExperiment)}
	var known int64
	var from, to time.Time
	for _, period := range periods {
		experiment, err := h.experiment(ctx, period, active[period.ID], now)
		if err != nil {
			return nil, err
		}
		known += experiment.GetIncumbent().GetKnown() + experiment.GetChallenger().GetKnown()
		if from.IsZero() || period.StartedAt.Before(from) {
			from = period.StartedAt
		}
		response.Experiments = append(response.Experiments, experiment)
	}
	if to = now; from.IsZero() {
		from = now
	}
	response.Validity = h.validityForSample(known)
	response.Provenance = usageAttributionProvenance(definitionFor(UsageExperiment).SourceTable, int64(len(periods)), from, to)
	return connect.NewResponse(response), nil
}

func protoArm(arm outcomeledger.ArmResult) *measurepb.UsageExperimentArm {
	return &measurepb.UsageExperimentArm{
		Arm: arm.Arm, Units: int64(arm.Units), Known: int64(arm.Known), Accepted: int64(arm.Accepted), Exceptions: int64(arm.Exceptions),
		PointsPerAccepted: protoRange(arm.Row.PointsPerAccepted), ReworkRate: protoRange(arm.Row.ReworkRate),
		Completed: int64(arm.Completed), AssumedAccepted: int64(arm.AssumedAccepted), Unknown: int64(arm.Unknown), TotalPoints: arm.TotalPoints, UnknownPoints: arm.UnknownPoints,
		PointsPerCompleted: protoRange(arm.Row.PointsPerCompleted), Clusters: int64(arm.Clusters),
	}
}

// comparisonStarts lists when every recorded learn comparison started: the family
// the corrected level divides over.
func (h *Handler) comparisonStarts(ctx context.Context) ([]time.Time, error) {
	periods, err := h.learnPeriods(ctx, "")
	if err != nil {
		return nil, err
	}
	starts := make([]time.Time, len(periods))
	for i, period := range periods {
		starts[i] = period.StartedAt
	}
	return starts, nil
}

// experiment analyses one learn period by assigned arm.
func (h *Handler) experiment(ctx context.Context, period outcomeledger.ModePeriod, mode usagemode.Mode, now time.Time) (*measurepb.UsageExperiment, error) {
	end := period.EndedAt
	if end.IsZero() || end.After(now) {
		end = now
	}
	in, err := h.loadOutcomeInputs(ctx, period.StartedAt, end)
	if err != nil {
		return nil, err
	}
	exceptions := map[string]int{}
	for _, run := range in.runs {
		if run.ArmMode == period.ID && run.ArmException != "" {
			exceptions[run.Arm]++
		}
	}
	units, samples := outcomeledger.Assemble(outcomeledger.ArmWork(in.runs, period.ID), in.use, outcomeledger.ByArm)
	cfg := outcomeledger.DefaultExperimentConfig()
	if period.BlockDays > 0 {
		cfg.BlockDays = period.BlockDays
	}
	cfg.ComparisonStarts, err = h.comparisonStarts(ctx)
	if err != nil {
		return nil, err
	}
	report := outcomeledger.AnalyzeExperiment(period.ID, units, samples, exceptions, cfg)

	out := &measurepb.UsageExperiment{
		ModeId: period.ID, Label: mode.Label, Role: period.Role, Runner: period.Runner, Design: report.Design, SplitPercent: int32(period.Split*100 + 0.5),
		SessionsOptIn: period.SessionOptIn, BlockDays: int32(period.BlockDays), Active: !mode.StartedAt.IsZero(),
		StartedAt: rfc3339(period.StartedAt), EndsAt: rfc3339(period.ExpiresAt),
		Incumbent: protoArm(report.Incumbent), Challenger: protoArm(report.Challenger),
		CostRatio: protoRange(report.Cost), CostPValue: report.CostP, ReworkDifference: protoRange(report.ReworkDiff), ReworkPValue: report.ReworkP,
		Verdict: report.Verdict, Reason: report.Reason, NeedAccepted: int64(report.NeedAccepted),
		LookReached: report.LookReached, PlannedAccepted: int64(report.Config.MinAcceptedPerArm), LookAt: rfc3339(report.LookAt), AlphaPerTest: report.AlphaPerTest, Comparisons: int64(report.Comparisons),
		SpanningUnits: int64(report.SpanningUnits), UnitsAfterLook: int64(report.UnitsAfterLook), UnknownRateDifference: report.UnknownRateGap, CostBound: protoRange(report.CostBound),
		DecisionRange: protoRange(report.DecisionRange), DecisionLevel: report.DecisionLevel,
	}
	if mode.Learn != nil {
		out.Paused, out.PausedReason = mode.Learn.Paused, mode.Learn.PausedReason
		if settings, ok := mode.ChallengerSettings(period.Runner); ok {
			for from, to := range settings.ModelMap {
				out.IncumbentModel, out.ChallengerModel = from, to
				break
			}
		}
	}
	if out.ChallengerModel == "" && len(period.Targets) > 0 {
		out.ChallengerModel = period.Targets[0]
	}
	if period.Role != "" {
		if pilot, err := h.rolePilot(ctx, period.Role, now); err == nil {
			out.Plan = protoPlan(planFor(pilot, out.SplitPercent, 0))
		}
	}
	out.Recommendation = h.recommend(ctx, out, report)
	return out, nil
}

// recommend turns a verdict into a role-default recommendation. It only ever
// proposes: cheaper looks better and about the same support a change, the
// other verdicts (inconclusive and withheld among them) do not. The policy change is shown exactly, never applied.
func (h *Handler) recommend(ctx context.Context, experiment *measurepb.UsageExperiment, report outcomeledger.ExperimentReport) *measurepb.UsageRoleRecommendation {
	recommendation := &measurepb.UsageRoleRecommendation{
		Role: experiment.GetRole(), Runner: experiment.GetRunner(), FromModel: experiment.GetIncumbentModel(), ToModel: experiment.GetChallengerModel(), Verdict: report.Verdict, Evidence: report.Reason,
	}
	switch report.Verdict {
	case outcomeledger.ExperimentCheaperBetter, outcomeledger.ExperimentAboutSame:
	default:
		return recommendation
	}
	if recommendation.FromModel == "" || recommendation.ToModel == "" {
		recommendation.PolicyNote = "the incumbent and challenger models are known only while the comparison is active"
		return recommendation
	}
	recommendation.Available = true
	if h.usageModes != nil {
		change := h.usageModes.RoleDefaultChange(ctx, recommendation.Runner, recommendation.Role, recommendation.FromModel, recommendation.ToModel)
		recommendation.PolicyFile, recommendation.PolicyNote = change.File, change.Note
		for _, op := range change.Ops {
			recommendation.PolicyOps = append(recommendation.PolicyOps, fmt.Sprintf("%s %s: %q -> %q", op.Op, op.Path, op.From, op.Value))
		}
	}
	recommendation.DecisionId = operatordecision.CorrelationID("umr", experiment.GetModeId(), recommendation.Role, recommendation.ToModel)
	if h.roleProposer != nil {
		if state, note, ok := h.roleProposer.AskState(recommendation.DecisionId); ok {
			recommendation.DecisionState, recommendation.DecisionNote = state, note
		}
	}
	return recommendation
}

func (h *Handler) ProposeRoleDefault(ctx context.Context, req *connect.Request[measurepb.ProposeRoleDefaultRequest]) (*connect.Response[measurepb.ProposeRoleDefaultResponse], error) {
	if _, err := requireModeOperator(ctx); err != nil {
		return nil, err
	}
	if h.roleProposer == nil || h.usageModes == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("role-default proposals are unavailable"))
	}
	periods, err := h.learnPeriods(ctx, strings.TrimSpace(req.Msg.GetModeId()))
	if err != nil {
		return nil, err
	}
	if len(periods) != 1 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no comparison %q", req.Msg.GetModeId()))
	}
	active := map[string]usagemode.Mode{}
	for _, mode := range h.usageModes.List() {
		active[mode.ID] = mode
	}
	now := h.now().UTC()
	experiment, err := h.experiment(ctx, periods[0], active[periods[0].ID], now)
	if err != nil {
		return nil, err
	}
	recommendation := experiment.GetRecommendation()
	if !recommendation.GetAvailable() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("the comparison reads %q, which does not support changing a role default: %s", experiment.GetVerdict(), experiment.GetReason()))
	}
	change := h.usageModes.RoleDefaultChange(ctx, recommendation.GetRunner(), recommendation.GetRole(), recommendation.GetFromModel(), recommendation.GetToModel())
	id, err := h.roleProposer.ProposeRoleDefault(ctx, usagemodes.RoleDefaultProposal{
		ModeID: experiment.GetModeId(), Runner: recommendation.GetRunner(), Role: recommendation.GetRole(), From: recommendation.GetFromModel(), To: recommendation.GetToModel(),
		Verdict: experiment.GetVerdict(), Evidence: recommendation.GetEvidence(), Change: change,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	recommendation.DecisionId = id
	recommendation.DecisionState = "pending"
	return connect.NewResponse(&measurepb.ProposeRoleDefaultResponse{Recommendation: recommendation}), nil
}

// ReadyEvidence implements usagemodes.EvidenceSource over the run ledger: a
// learn comparison is ready when its verdict is anything but not enough data,
// and a save mode when its model comparison reached a verdict inside the mode's
// own period.
func (h *Handler) ReadyEvidence(ctx context.Context, now time.Time) ([]usagemodes.EvidenceReport, error) {
	ledger, err := h.ledgerReader()
	if err != nil {
		return nil, err
	}
	periods, err := ledger.ModePeriods(ctx)
	if err != nil {
		return nil, err
	}
	labels := map[string]string{}
	active := map[string]usagemode.Mode{}
	if h.usageModes != nil {
		for _, mode := range h.usageModes.List() {
			labels[mode.ID], active[mode.ID] = mode.Label, mode
		}
	}
	var reports []usagemodes.EvidenceReport
	for _, period := range periods {
		if now.Sub(period.StartedAt) > outcomeledger.Lookback {
			continue
		}
		label := labels[period.ID]
		if label == "" {
			label = period.ID
		}
		if period.Kind == usagemode.KindLearn {
			experiment, err := h.experiment(ctx, period, active[period.ID], now)
			if err != nil {
				return reports, err
			}
			if experiment.GetVerdict() != outcomeledger.ExperimentNotEnough {
				reports = append(reports, usagemodes.EvidenceReport{ModeID: period.ID, Label: label, Line: learnReadyLine(experiment)})
			}
			continue
		}
		line, ok, err := h.saveReadyLine(ctx, period, now)
		if err != nil {
			return reports, err
		}
		if ok {
			reports = append(reports, usagemodes.EvidenceReport{ModeID: period.ID, Label: label, Line: line})
		}
	}
	sort.Slice(reports, func(i, j int) bool { return reports[i].ModeID < reports[j].ModeID })
	return reports, nil
}

func learnReadyLine(experiment *measurepb.UsageExperiment) string {
	words := map[string]string{
		outcomeledger.ExperimentCheaperBetter: "the cheaper model looks better",
		outcomeledger.ExperimentAboutSame:     "the two models look about the same",
		outcomeledger.ExperimentHigherBetter:  "the higher model looks better",
		outcomeledger.ExperimentInconclusive:  "the planned look settled nothing",
		outcomeledger.ExperimentWithheld:      "the verdict is withheld",
	}
	return fmt.Sprintf("%s for %s: %s. %s", designWords(experiment.GetDesign()), experiment.GetRole(), words[experiment.GetVerdict()], experiment.GetReason())
}

// designWords names a design from its recorded basis.
func designWords(design string) string {
	switch design {
	case outcomeledger.DesignRandomized:
		return "Randomized comparison"
	case outcomeledger.DesignDayBlockRandomized:
		return "Day-block randomized comparison"
	}
	return "Comparison with no assignment on record"
}

func (h *Handler) saveReadyLine(ctx context.Context, period outcomeledger.ModePeriod, now time.Time) (string, bool, error) {
	end := period.EndedAt
	if end.IsZero() || end.After(now) {
		end = now
	}
	in, err := h.loadOutcomeInputs(ctx, period.StartedAt, end)
	if err != nil {
		return "", false, err
	}
	units, samples := outcomeledger.Assemble(in.runs, in.use, outcomeledger.ByModel)
	report := outcomeledger.Analyze(units, samples, outcomeledger.Config{})
	for _, row := range report.Rows {
		if row.Status != outcomeledger.StatusEnough || !row.CheaperBy.Available || row.CheaperBy.Value <= 0 || row.CheaperBy.Low <= 0 || row.CheaperBy.High <= 0 {
			continue
		}
		switch row.Verdict {
		case outcomeledger.VerdictCheaper, outcomeledger.VerdictMoreExpensive, outcomeledger.VerdictAboutSame:
			return fmt.Sprintf("Observational result (%s work): %s cost %.1fx as much per accepted outcome as %s (range %.1fx to %.1fx). It compares periods as they happened, so treat it as a clue.",
				row.Stratum, row.Key, 1/row.CheaperBy.Value, row.Baseline, 1/row.CheaperBy.High, 1/row.CheaperBy.Low), true, nil
		}
	}
	return "", false, nil
}
