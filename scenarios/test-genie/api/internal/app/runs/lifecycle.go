package runs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/freshness-go/treedigest"
	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/runmanager"
	"test-genie/internal/shared"
	sharedruns "test-genie/internal/shared/runs"

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
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	if strings.TrimSpace(req.Msg.GetSuiteRequestId()) != "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("suite_request_id has been removed; create a remediation job from a completed execution instead"))
	}

	input := execution.SuiteExecutionInput{
		Request: orchestrator.SuiteExecutionRequest{
			ScenarioName:           scenario,
			Preset:                 strings.TrimSpace(req.Msg.GetPreset()),
			Phases:                 req.Msg.GetPhases(),
			Skip:                   req.Msg.GetSkip(),
			FailFast:               req.Msg.GetFailFast(),
			DiagnosticsPreset:      strings.TrimSpace(req.Msg.GetDiagnosticsPreset()),
			CaptureProfile:         strings.TrimSpace(req.Msg.GetCaptureProfile()),
			UIURL:                  strings.TrimSpace(req.Msg.GetUiUrl()),
			APIURL:                 strings.TrimSpace(req.Msg.GetApiUrl()),
			ScenarioPath:           strings.TrimSpace(req.Msg.GetScenarioPath()),
			LogicalRepoRoot:        strings.TrimSpace(req.Msg.GetLogicalRepoRoot()),
			LogicalScenarioRelPath: strings.TrimSpace(req.Msg.GetLogicalScenarioRelPath()),
			RequireGateQuality:     req.Msg.GetRequireGateQuality(),
		},
	}
	caller := strings.TrimSpace(req.Header().Get("X-Vrooli-Caller"))
	releasePreview, err := s.runManager.TryAcquirePreviewFor(caller)
	if err != nil {
		return nil, saturationError(err)
	}
	defer releasePreview()
	etaTotal, etaKnown, err := s.previewPlan(ctx, input.Request)
	if err != nil {
		return nil, err
	}
	if err := s.attachAdmissionIdentity(&input.Request); err != nil {
		return nil, err
	}
	if runID := s.runManager.CoalescedRunID(input.Request); runID != "" {
		return connect.NewResponse(&runspb.StartRunResponse{RunId: runID, Scenario: scenario, Coalesced: true}), nil
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
		Scenario:              scenario,
		EstimatedTotalSeconds: int32(etaTotal),
		EtaKnown:              etaKnown,
		Coalesced:             res.Coalesced,
	}), nil
}

func (s *Service) attachAdmissionIdentity(req *orchestrator.SuiteExecutionRequest) error {
	if req == nil {
		return errors.New("run request is required")
	}
	dir, err := s.scenarioDir(req.ScenarioName)
	if err != nil {
		return err
	}
	digest, err := treedigest.Compute(dir)
	if err != nil {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("compute source identity: %w", err))
	}
	if s.planner == nil {
		req.AdmissionTreeDigest = digest
		return nil
	}
	preview, err := s.planner.Preview(context.Background(), *req)
	if err != nil || preview == nil {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("resolve execution identity"))
	}
	req.AdmissionTreeDigest = digest
	req.AdmissionPhaseSetDigest = preview.PhaseSetDigest
	req.AdmissionDescriptorDigest = preview.DescriptorSnapshotDigest
	req.AdmissionConfigurationDigest = preview.ConfigurationFingerprint
	return nil
}

func saturationError(err error) *connect.Error {
	return connect.NewError(connect.CodeResourceExhausted, err)
}

// busyError maps a run-admission rejection to a FailedPrecondition carrying a
// typed RunBusyInfo detail, so callers render wait/abort guidance without
// parsing the error string.
func busyError(busy *runmanager.BusyError) *connect.Error {
	cerr := connect.NewError(connect.CodeFailedPrecondition, busy)
	if detail, derr := connect.NewErrorDetail(&runspb.RunBusyInfo{
		Scenario: busy.Scenario,
		RunId:    busy.RunID,
		Preset:   busy.Preset,
	}); derr == nil {
		cerr.AddDetail(detail)
	}
	return cerr
}

// previewPlan derives the summed plan estimate and surfaces plan validation
// errors (bad preset/phase) as InvalidArgument so a malformed request is
// rejected up front rather than failing inside the run goroutine. A non-fatal
// preview error (e.g. no timing history) yields ETA-unknown and proceeds.
func (s *Service) previewPlan(ctx context.Context, req orchestrator.SuiteExecutionRequest) (int, bool, error) {
	if s.planner == nil {
		return 0, false, nil
	}
	preview, err := s.planner.Preview(ctx, req)
	if err != nil {
		var vErr shared.ValidationError
		if errors.As(err, &vErr) {
			return 0, false, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return 0, false, nil
	}
	if preview == nil {
		return 0, false, nil
	}
	total := preview.Summary.EstimatedDurationSeconds
	return total, total > 0, nil
}

// FollowRun streams canonical run events, replaying history first. Cancelling
// the stream detaches the follower without aborting the run.
func (s *Service) FollowRun(ctx context.Context, req *connect.Request[runspb.FollowRunRequest], stream *connect.ServerStream[runspb.RunEvent]) error {
	if s.runManager == nil {
		return connect.NewError(connect.CodeUnimplemented, errors.New("run manager is not configured"))
	}
	scenario := strings.TrimSpace(req.Msg.GetScenario())
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
	scenario := strings.TrimSpace(req.Msg.GetScenario())
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
	st, err := s.runManager.Abort(strings.TrimSpace(req.Msg.GetScenario()), strings.TrimSpace(req.Msg.GetRunId()))
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
	st, err := s.runManager.Status(strings.TrimSpace(req.Msg.GetScenario()), strings.TrimSpace(req.Msg.GetRunId()))
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
		Scenario:          ev.Scenario,
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
		Scenario:                    st.Scenario,
		Status:                      st.Status,
		ActivePhase:                 st.ActivePhase,
		PhaseIndex:                  int32(st.PhaseIndex),
		PhaseTotal:                  int32(st.PhaseTotal),
		StartedAt:                   startedAt,
		ElapsedSeconds:              st.ElapsedSeconds,
		EstimatedTotalSeconds:       int32(st.EstimatedTotalSeconds),
		EstimatedRemainingSeconds:   int32(st.EstimatedRemainingSeconds),
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
