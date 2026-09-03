package runs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/runmanager"
	"test-genie/internal/shared"
	sharedruns "test-genie/internal/shared/runs"
	"test-genie/internal/targetmodel"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/freshness-go/treedigest"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// StartRun starts a durable, request-decoupled suite run and returns its id and
// ETA synchronously. Plan validation (bad preset/phase) is surfaced up front as
// InvalidArgument; the run itself survives client cancellation.
func (s *Service) StartRun(ctx context.Context, req *connect.Request[runspb.StartRunRequest]) (*connect.Response[runspb.StartRunResponse], error) {
	if s.runManager == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("run manager is not configured"))
	}
	scenario := strings.TrimSpace(req.Msg.GetTarget())
	targetRef := req.Msg.GetTargetRef()
	if targetRef != nil {
		// The admission key is the canonical target expression. The typed target
		// remains on the request and response for consumers that do not use text.
		if expression := targetExpression(targetRef); expression != "" {
			scenario = expression
		}
	}
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or target is required"))
	}
	if strings.TrimSpace(req.Msg.GetSuiteRequestId()) != "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("suite_request_id has been removed; create a remediation job from a completed execution instead"))
	}

	input := execution.SuiteExecutionInput{
		Request: orchestrator.SuiteExecutionRequest{
			ScenarioName:                     scenario,
			Target:                           targetExpression(targetRef),
			Preset:                           strings.TrimSpace(req.Msg.GetPreset()),
			Phases:                           req.Msg.GetPhases(),
			Skip:                             req.Msg.GetSkip(),
			FailFast:                         req.Msg.GetFailFast(),
			DiagnosticsPreset:                strings.TrimSpace(req.Msg.GetDiagnosticsPreset()),
			CaptureProfile:                   strings.TrimSpace(req.Msg.GetCaptureProfile()),
			UIURL:                            strings.TrimSpace(req.Msg.GetUiUrl()),
			APIURL:                           strings.TrimSpace(req.Msg.GetApiUrl()),
			ScenarioPath:                     strings.TrimSpace(req.Msg.GetScenarioPath()),
			LogicalRepoRoot:                  strings.TrimSpace(req.Msg.GetLogicalRepoRoot()),
			LogicalScenarioRelPath:           strings.TrimSpace(req.Msg.GetLogicalScenarioRelPath()),
			RequireGateQuality:               req.Msg.GetRequireGateQuality(),
			CollectionReservationID:          strings.TrimSpace(req.Msg.GetCollectionReservationId()),
			CollectionReservationMemberCount: int(req.Msg.GetCollectionReservationMemberCount()),
			RetainForEvidence:                req.Msg.GetRetainForEvidence(),
			RetentionReason:                  strings.TrimSpace(req.Msg.GetRetentionReason()),
		},
	}
	caller := strings.TrimSpace(req.Header().Get(cliutil.HeaderCaller))
	releasePreview, err := s.runManager.TryAcquirePreviewFor(caller)
	if err != nil {
		return nil, saturationError(err)
	}
	defer releasePreview()
	etaTotal, etaKnown, err := s.prepareAdmission(ctx, &input.Request)
	if err != nil {
		return nil, err
	}
	if runID := s.runManager.CoalescedRunID(input.Request); runID != "" {
		return connect.NewResponse(&runspb.StartRunResponse{RunId: runID, Target: scenario, TargetRef: targetRef, Coalesced: true}), nil
	}

	res, err := s.runManager.Start(runmanager.StartOptions{Input: input, Caller: caller, EstimatedTotalSeconds: etaTotal})
	if err != nil {
		var busy *runmanager.BusyError
		if errors.As(err, &busy) {
			return nil, busyError(busy)
		}
		var saturated *runmanager.SaturatedError
		if errors.As(err, &saturated) {
			return nil, saturationError(saturated)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&runspb.StartRunResponse{
		RunId:                 res.RunID,
		Target:                scenario,
		EstimatedTotalSeconds: int32(etaTotal),
		EtaKnown:              etaKnown,
		Coalesced:             res.Coalesced,
		TargetRef:             targetRef,
	}), nil
}

func targetExpression(target *commonv1.ValidationTarget) string {
	if target == nil || target.GetKind() == commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_UNSPECIFIED {
		return ""
	}
	names := map[commonv1.ValidationTargetKind]string{
		commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO:      "scenario",
		commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_RESOURCE:      "resource",
		commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TOOL:          "tool",
		commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SAFEGUARD:     "safeguard",
		commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TEAM:          "team",
		commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PACKAGE:       "package",
		commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_CONTROL_PLANE: "control-plane",
		commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_DOCS:          "docs",
		commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PROJECT:       "project",
	}
	name := names[target.GetKind()]
	if name == "" {
		return ""
	}
	return name + ":" + target.GetId()
}

// prepareAdmission computes the complete admission identity and ETA in one
// bounded planner call. The preview token is intentionally held while this
// work runs, so it must honor the caller's context and never invoke a second,
// uncancellable planner preview.
func (s *Service) prepareAdmission(ctx context.Context, req *orchestrator.SuiteExecutionRequest) (int, bool, error) {
	if req == nil {
		return 0, false, errors.New("run request is required")
	}
	dir, err := s.admissionScenarioDir(req)
	if err != nil {
		return 0, false, err
	}
	digest, err := treedigest.Compute(dir)
	if err != nil {
		return 0, false, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("compute source identity: %w", err))
	}
	if s.planner == nil {
		req.AdmissionTreeDigest = digest
		return 0, false, nil
	}
	preview, err := s.planner.Preview(ctx, *req)
	if err != nil {
		var vErr shared.ValidationError
		if errors.As(err, &vErr) {
			return 0, false, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return 0, false, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("resolve execution identity: %w", err))
	}
	if preview == nil {
		return 0, false, connect.NewError(connect.CodeFailedPrecondition, errors.New("resolve execution identity: planner returned no preview"))
	}
	// StartRun is the durable RPC path used by the CLI and control-plane
	// callers. Carry the planner's selected phases and timing guidance into the
	// executor here; the streaming HTTP path applies the same projection in its
	// request handler. Without this, the scheduler sees an empty prediction map
	// and can batch a phase that is already close to its timeout budget.
	applyAdmissionPreview(req, preview)
	req.AdmissionTreeDigest = digest
	req.AdmissionPhaseSetDigest = preview.PhaseSetDigest
	req.AdmissionDescriptorDigest = preview.DescriptorSnapshotDigest
	req.AdmissionConfigurationDigest = preview.ConfigurationFingerprint
	var resources []string
	for _, phase := range preview.Phases {
		resources = append(resources, phase.RequiredResources...)
	}
	req.AdmissionResources = uniqueStrings(resources)
	total := preview.Summary.EstimatedDurationSeconds
	return total, total > 0, nil
}

// applyAdmissionPreview carries the planner's selection and timing guidance
// into the executor without impersonating an explicit phase request.
//
// It used to append every previewed phase onto req.Phases, which silently
// converted a preset request into an explicit phase request. Downstream that is
// a different kind of request: resolveDesiredPhaseList treats explicit phases
// as exact user intent and assigns no preset, so the run recorded
// preset_used=NULL. Git Control Tower's FindReusableRun keys baseline reuse on
// Preset=="comprehensive", so every durable run made itself ineligible for the
// reuse it had just earned. Measured 2026-08-08: runs went from
// preset=comprehensive with 0 requested phases to preset=NULL with 20.
//
// The selection is still carried, because an adaptive profile trims the set to
// a budget and the executor cannot re-derive that. It now travels in
// ResolvedPhases, which narrows selection while keeping the preset name.
func applyAdmissionPreview(req *orchestrator.SuiteExecutionRequest, preview *execution.ExecutionPlanPreview) {
	if req == nil || preview == nil {
		return
	}
	if len(req.Phases) > 0 {
		// The operator named phases explicitly; the planner does not override
		// exact intent.
		return
	}
	for _, phase := range preview.Phases {
		name := strings.TrimSpace(phase.Name)
		if name == "" {
			continue
		}
		req.ResolvedPhases = append(req.ResolvedPhases, name)
		if req.PredictedPhaseDurationsMilliseconds == nil {
			req.PredictedPhaseDurationsMilliseconds = make(map[string]int64)
		}
		req.PredictedPhaseDurationsMilliseconds[strings.ToLower(name)] = int64(phase.EstimatedDurationSeconds) * 1000
	}
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// admissionScenarioDir must use the same physical source selected by the
// orchestrator. Custom scenario paths are used for isolated scratch runs and
// baseline fixtures; hashing the canonical scenario directory here makes the
// admission digest disagree with the execution digest before any phase runs.
func (s *Service) admissionScenarioDir(req *orchestrator.SuiteExecutionRequest) (string, error) {
	if path := strings.TrimSpace(req.ScenarioPath); path != "" {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("resolve scenario path: %w", err))
		}
		info, err := os.Stat(absolute)
		if err != nil {
			return "", connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("inspect scenario path: %w", err))
		}
		if !info.IsDir() {
			return "", connect.NewError(connect.CodeInvalidArgument, errors.New("scenario path must be a directory"))
		}
		return absolute, nil
	}
	if strings.Contains(strings.TrimSpace(req.Target), ":") {
		target, err := targetmodel.Resolve(filepath.Dir(s.scenariosRoot), req.Target)
		if err != nil {
			return "", connect.NewError(connect.CodeInvalidArgument, err)
		}
		return target.Path, nil
	}
	// scenarioDir resolves the artifact root used for run history. Admission
	// identity must instead hash the physical source tree that the executor
	// will validate; hashing ~/.vrooli/test-runs/<scenario> makes every normal
	// run fail before execution when that artifact directory does not exist.
	sourceDir := filepath.Join(s.scenariosRoot, strings.TrimSpace(req.ScenarioName))
	if _, err := os.Stat(filepath.Join(sourceDir, filepath.FromSlash(scenarioManifest))); err != nil {
		return "", connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("inspect scenario source: %w", err))
	}
	return sourceDir, nil
}

func saturationError(err error) *connect.Error {
	cerr := connect.NewError(connect.CodeResourceExhausted, err)
	var saturated *runmanager.SaturatedError
	if errors.As(err, &saturated) {
		if detail, derr := connect.NewErrorDetail(&runspb.AdmissionSaturation{
			LimitKind:         admissionLimitKind(saturated.Limit),
			Occupancy:         int32(saturated.Occupancy),
			ConfiguredLimit:   int32(saturated.ConfiguredLimit),
			FifoPosition:      int32(saturated.FIFOPosition),
			RetryAfterSeconds: int32(saturated.RetryAfterSeconds),
		}); derr == nil {
			cerr.AddDetail(detail)
		}
	}
	return cerr
}

func admissionLimitKind(limit string) runspb.AdmissionLimitKind {
	switch limit {
	case "queued run capacity":
		return runspb.AdmissionLimitKind_ADMISSION_LIMIT_KIND_GLOBAL_QUEUE
	case "caller queued run capacity":
		return runspb.AdmissionLimitKind_ADMISSION_LIMIT_KIND_CALLER_QUEUE
	case "reservation queued run capacity":
		return runspb.AdmissionLimitKind_ADMISSION_LIMIT_KIND_RESERVATION_QUEUE
	case "preview capacity":
		return runspb.AdmissionLimitKind_ADMISSION_LIMIT_KIND_GLOBAL_PREVIEW
	case "caller preview capacity":
		return runspb.AdmissionLimitKind_ADMISSION_LIMIT_KIND_CALLER_PREVIEW
	default:
		return runspb.AdmissionLimitKind_ADMISSION_LIMIT_KIND_UNSPECIFIED
	}
}

// busyError maps a run-admission rejection to a FailedPrecondition carrying a
// typed RunBusyInfo detail, so callers render wait/abort guidance without
// parsing the error string.
func busyError(busy *runmanager.BusyError) *connect.Error {
	cerr := connect.NewError(connect.CodeFailedPrecondition, busy)
	if detail, derr := connect.NewErrorDetail(&runspb.RunBusyInfo{
		Target: busy.Scenario,
		RunId:  busy.RunID,
		Preset: busy.Preset,
	}); derr == nil {
		cerr.AddDetail(detail)
	}
	return cerr
}

// FollowRun streams canonical run events, replaying history first. Cancelling
// the stream detaches the follower without aborting the run.
func (s *Service) FollowRun(ctx context.Context, req *connect.Request[runspb.FollowRunRequest], stream *connect.ServerStream[runspb.RunEvent]) error {
	if s.runManager == nil {
		return connect.NewError(connect.CodeUnimplemented, errors.New("run manager is not configured"))
	}
	scenario := strings.TrimSpace(req.Msg.GetTarget())
	runID := strings.TrimSpace(req.Msg.GetRunId())
	// suppress_heartbeats is a per-follower filter: the manager keeps publishing
	// heartbeats to the shared broadcaster (other followers — e.g. browser SSE —
	// still receive them); we just skip them on THIS stream.
	suppressHeartbeats := req.Msg.GetSuppressHeartbeats()
	replay, ch, err := s.runManager.FollowAfter(ctx, scenario, runID, req.Msg.GetAfterSequence())
	if err != nil {
		return mapRunError(err)
	}
	for _, ev := range replay {
		if !keepFollowEvent(ev, suppressHeartbeats) {
			continue
		}
		if err := stream.Send(toRunEvent(ev)); err != nil {
			return err
		}
	}
	for ev := range ch {
		if !keepFollowEvent(ev, suppressHeartbeats) {
			continue
		}
		if err := stream.Send(toRunEvent(ev)); err != nil {
			return err
		}
	}
	return nil
}

// keepFollowEvent reports whether an event is forwarded to a follower. When
// suppressHeartbeats is set, phase_heartbeat keep-alives are dropped; every other
// event (phase transitions and the terminal run_completed) always passes, so the
// filter can never silence progress or the verdict.
func keepFollowEvent(ev runmanager.Event, suppressHeartbeats bool) bool {
	return !(suppressHeartbeats && ev.Kind == runmanager.EventPhaseHeartbeat)
}

// WaitRun blocks until terminal or the optional timeout, returning the live
// status. A timeout returns the current (non-terminal) snapshot with
// timed_out=true; the run keeps executing.
func (s *Service) WaitRun(ctx context.Context, req *connect.Request[runspb.WaitRunRequest]) (*connect.Response[runspb.WaitRunResponse], error) {
	if s.runManager == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("run manager is not configured"))
	}
	scenario := strings.TrimSpace(req.Msg.GetTarget())
	runID := strings.TrimSpace(req.Msg.GetRunId())

	waitCtx := ctx
	if secs := req.Msg.GetTimeoutSeconds(); secs > 0 {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, time.Duration(secs)*time.Second)
		defer cancel()
	}

	st, err := s.runManager.Wait(waitCtx, scenario, runID)
	timedOut := false
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			timedOut = true
		case errors.Is(err, sharedruns.ErrRunNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	if isTerminalStatus(st.Status) {
		timedOut = false
	}
	response := &runspb.WaitRunResponse{
		Status:                        toLiveStatus(st),
		TimedOut:                      timedOut,
		TerminalSnapshotSchemaVersion: int32(st.TerminalSnapshotSchemaVersion),
		DegradedReasons:               append([]string(nil), st.DegradedReasons...),
	}
	if st.TerminalRecord != nil {
		response.TerminalRun = toTerminalRunInfo(*st.TerminalRecord, st.Result, st.DescriptorSnapshot)
	}
	return connect.NewResponse(response), nil
}

// AbortRun cancels a running run and reports its terminal aborted status.
func (s *Service) AbortRun(ctx context.Context, req *connect.Request[runspb.AbortRunRequest]) (*connect.Response[runspb.AbortRunResponse], error) {
	if s.runManager == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("run manager is not configured"))
	}
	st, err := s.runManager.Abort(strings.TrimSpace(req.Msg.GetTarget()), strings.TrimSpace(req.Msg.GetRunId()))
	if err != nil {
		return nil, mapRunError(err)
	}
	return connect.NewResponse(&runspb.AbortRunResponse{Status: toLiveStatus(st)}), nil
}

// GetRunStatus returns a point-in-time live snapshot.
func (s *Service) GetRunStatus(ctx context.Context, req *connect.Request[runspb.GetRunStatusRequest]) (*connect.Response[runspb.RunLiveStatus], error) {
	if s.runManager == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("run manager is not configured"))
	}
	st, err := s.runManager.Status(strings.TrimSpace(req.Msg.GetTarget()), strings.TrimSpace(req.Msg.GetRunId()))
	if err != nil {
		return nil, mapRunError(err)
	}
	return connect.NewResponse(toLiveStatus(st)), nil
}

func toRunEvent(ev runmanager.Event) *runspb.RunEvent {
	return &runspb.RunEvent{
		Sequence:          ev.Sequence,
		Event:             ev.Kind,
		ElapsedSeconds:    ev.ElapsedSeconds,
		RunId:             ev.RunID,
		Target:            ev.Scenario,
		ArtifactDir:       ev.ArtifactDir,
		Preset:            ev.Preset,
		Phase:             ev.Phase,
		PhaseIndex:        int32(ev.PhaseIndex),
		PhaseTotal:        int32(ev.PhaseTotal),
		Status:            ev.Status,
		DurationSeconds:   int32(ev.DurationSeconds),
		QuietSeconds:      ev.QuietSeconds,
		Message:           ev.Message,
		Success:           ev.Success,
		Verdict:           ev.Verdict,
		Error:             ev.Error,
		PhasePresentation: ev.PhasePresentation,
		FindingsSummary:   ev.FindingsSummary,
	}
}

func toLiveStatus(st runmanager.LiveStatus) *runspb.RunLiveStatus {
	startedAt := ""
	if !st.StartedAt.IsZero() {
		startedAt = st.StartedAt.UTC().Format(time.RFC3339)
	}
	lastProgressAt := startedAt
	if !st.LastProgressAt.IsZero() {
		lastProgressAt = st.LastProgressAt.UTC().Format(time.RFC3339)
	}
	standing := &commonv1.OperationStanding{
		Lifecycle:                 testGenieLifecycle(st),
		TerminalOutcome:           testGenieOutcome(st),
		Owner:                     "test-genie",
		OperationId:               st.RunID,
		StartedAt:                 startedAt,
		LastProgressAt:            lastProgressAt,
		ElapsedSeconds:            st.ElapsedSeconds,
		EstimatedRemainingSeconds: int32(st.EstimatedRemainingSeconds),
		EtaKnown:                  st.ETAKnown,
		Directive:                 testGenieDirective(st),
		RecommendedWaitSeconds:    int32(st.RecommendedNextCheckSeconds),
		ActivePhase:               st.ActivePhase,
	}
	if standing.GetLifecycle() != "terminal" && st.Scenario != "" && st.RunID != "" {
		standing.ReattachCommand = "test-genie runs wait --json " + st.Scenario + " " + st.RunID
	}
	return &runspb.RunLiveStatus{
		RunId:                       st.RunID,
		Target:                      st.Scenario,
		Status:                      st.Status,
		ActivePhase:                 st.ActivePhase,
		PhaseIndex:                  int32(st.PhaseIndex),
		PhaseTotal:                  int32(st.PhaseTotal),
		StartedAt:                   startedAt,
		ElapsedSeconds:              st.ElapsedSeconds,
		EstimatedTotalSeconds:       int32(st.EstimatedTotalSeconds),
		EstimatedRemainingSeconds:   int32(st.EstimatedRemainingSeconds),
		QueuePosition:               int32(st.QueuePosition),
		EstimatedQueueWaitSeconds:   int32(st.EstimatedQueueWaitSeconds),
		EtaKnown:                    st.ETAKnown,
		RecommendedNextCheckSeconds: int32(st.RecommendedNextCheckSeconds),
		Verdict:                     st.Verdict,
		Success:                     st.Success,
		Error:                       st.Error,
		Active:                      st.Active,
		TerminalPresentations:       st.TerminalPresentations,
		TerminalFindingsSummaries:   st.TerminalFindingsSummaries,
		DegradedReasons:             append([]string(nil), st.DegradedReasons...),
		Standing:                    standing,
	}
}

func testGenieLifecycle(st runmanager.LiveStatus) string {
	if isTerminalStatus(st.Status) {
		return "terminal"
	}
	if st.Status == "queued" || !st.Active {
		return "queued"
	}
	if st.ActivePhase == "" || strings.HasPrefix(st.ActivePhase, "preparing:") {
		return "preparing"
	}
	return "executing"
}

func testGenieOutcome(st runmanager.LiveStatus) string {
	switch st.Status {
	case "passed":
		return "passed"
	case "failed":
		return "failed"
	case "aborted":
		return "aborted"
	default:
		return ""
	}
}

func testGenieDirective(st runmanager.LiveStatus) string {
	if !isTerminalStatus(st.Status) {
		return "wait"
	}
	if st.Status == "passed" {
		return ""
	}
	return "inspect"
}

func isTerminalStatus(status string) bool {
	switch status {
	case sharedruns.StatusPassed, sharedruns.StatusFailed, sharedruns.StatusAborted:
		return true
	default:
		return false
	}
}
