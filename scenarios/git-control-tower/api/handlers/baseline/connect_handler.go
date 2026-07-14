// Package baseline (handlers) implements the BaselinesService Connect-RPC
// service. It translates proto messages to/from the domain shapes in
// internal/baseline and delegates orchestration to baseline.Service.
//
// Testing rule: handler tests construct *Server with fake adapters + a fake
// RepoResolver. No real git, test-genie, or scenario-auditor is invoked.
package baseline

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"connectrpc.com/connect"

	bl "git-control-tower/internal/baseline"
	"git-control-tower/internal/git"

	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
	"github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines/baselines_v1connect"
)

// snapshotTailCeiling bounds the detached snapshot orchestration so a wedged
// test-genie can never leak a goroutine forever. It must comfortably exceed the
// comprehensive run + pin + manifest write; the reachability probe already
// makes the common unreachable case fail in seconds, so this is a backstop, not
// the normal path.
const snapshotTailCeiling = 30 * time.Minute

// RepoResolver maps an optional explicit repoID (0 = active repo) to a concrete
// (repoID, repoDir) pair the baseline service operates on.
type RepoResolver interface {
	Resolve(ctx context.Context, repoID int64) (int64, string, error)
}

// Server implements baselines_v1connect.BaselinesServiceHandler.
type Server struct {
	svc    *bl.Service
	repos  RepoResolver
	logger *log.Logger
	// finalize, when set, overrides the async snapshot finalize tail (tests run
	// it synchronously to assert the pin lands).
	finalize func(ctx context.Context, pending bl.PendingCapture)
	// finalizeDiffFn, when set, overrides the async diff finalize tail (tests run
	// it synchronously to assert the verdict caches).
	finalizeDiffFn func(ctx context.Context, pending bl.PendingDiff)
	// finalizeCollection, when set, overrides collection-member finalization for
	// deterministic handler tests.
	finalizeCollection func(ctx context.Context, repoID int64, pending bl.PendingCollectionCapture)
	// finalizeCollectionDiff, when set, runs an aggregate child finalizer
	// synchronously in tests.
	finalizeCollectionDiffFn func(ctx context.Context, repoID int64, pending bl.PendingCollectionDiff)
}

// Deps wires the Connect server.
type Deps struct {
	Service *bl.Service
	Repos   RepoResolver
	Logger  *log.Logger
}

// NewServer builds a Server.
func NewServer(d Deps) *Server {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	s := &Server{svc: d.Service, repos: d.Repos, logger: d.Logger}
	s.reattachPendingSnapshots()
	return s
}

// NewHandler returns the (procedure-prefix, http.Handler) pair the router
// mounts.
func NewHandler(d Deps, opts ...connect.HandlerOption) (string, http.Handler) {
	return baselines_v1connect.NewBaselinesServiceHandler(NewServer(d), opts...)
}

// resolveTarget resolves the repo and the effective branch. When requested
// branch is empty and allBranches is false, the current branch is derived from
// the working tree.
func (s *Server) resolveTarget(ctx context.Context, repoID int64, branch string, allBranches bool) (int64, string, string, error) {
	rid, repoDir, err := s.repos.Resolve(ctx, repoID)
	if err != nil {
		return 0, "", "", err
	}
	if allBranches {
		return rid, repoDir, "", nil
	}
	if branch == "" {
		st, gerr := git.Capture(ctx, repoDir)
		if gerr != nil {
			return 0, "", "", gerr
		}
		branch = bl.ResolveStorageBranch(st)
	}
	return rid, repoDir, branch, nil
}

// SnapshotForBaseline starts ONE comprehensive, durable test-genie run and
// returns its handle IMMEDIATELY — it does not block for the run to finish. The
// pin + manifest write happen on a server-owned goroutine when the run
// completes, so:
//
//   - the caller gets a run id + ETA up front and never sits on a silent block;
//   - a client that disconnects (or Ctrl-Cs) can never abandon a half-pinned,
//     manifest-less baseline — the finalize tail runs under the server-lifetime
//     context (WithoutCancel keeps the request's values, drops its cancellation),
//     bounded by snapshotTailCeiling so a wedged backend can't leak a goroutine.
//
// The caller reattaches through GCT with `baseline snapshot status --wait
// --json`, which finalizes the pin/manifest as the parent workflow authority.
// Raw test-genie wait/follow commands remain diagnostic for the child run.
func (s *Server) SnapshotForBaseline(ctx context.Context, req *connect.Request[baselinesv1.SnapshotForBaselineRequest]) (*connect.Response[baselinesv1.SnapshotForBaselineResponse], error) {
	m := req.Msg
	rid, repoDir, err := s.repos.Resolve(ctx, m.GetRepoId())
	if err != nil {
		return nil, s.wrap("SnapshotForBaseline", err)
	}

	pending, err := s.svc.StartCapture(ctx, bl.CreateRequest{
		RepoID: rid, RepoDir: repoDir, Scenario: m.GetScenario(), Name: m.GetName(),
		Branch: m.GetBranch(), CreatedBy: m.GetCreatedBy(), Reason: m.GetReason(),
	})
	if err != nil {
		var busy *bl.RunBusyError
		if errors.As(err, &busy) {
			return nil, busyConnectError(busy)
		}
		return nil, s.wrap("SnapshotForBaseline", err)
	}

	s.logger.Printf("baselines.SnapshotForBaseline: started comprehensive run scenario=%s name=%s run=%s eta=%ds coalesced=%t (durable; pinning server-side on completion)",
		m.GetScenario(), m.GetName(), pending.Run.RunID, pending.Run.EstimatedTotalSeconds, pending.Run.Coalesced)

	// Finalize on a server-owned context detached from this request.
	s.finalizeSnapshot(ctx, pending)

	return connect.NewResponse(&baselinesv1.SnapshotForBaselineResponse{
		RunId:                 pending.Run.RunID,
		Scenario:              pending.Manifest.Scenario,
		Name:                  pending.Manifest.Name,
		Branch:                pending.Manifest.Branch,
		EstimatedTotalSeconds: int32(pending.Run.EstimatedTotalSeconds),
		EtaKnown:              pending.Run.EtaKnown,
		DirtyWarning:          pending.DirtyWarning,
		Coalesced:             pending.Run.Coalesced,
	}), nil
}

// busyConnectError maps a one-run-per-scenario rejection to FailedPrecondition
// carrying a typed RunBusyInfo detail (shared by the snapshot + diff doors).
func busyConnectError(busy *bl.RunBusyError) error {
	cerr := connect.NewError(connect.CodeFailedPrecondition, busy)
	if detail, derr := connect.NewErrorDetail(&baselinesv1.RunBusyInfo{
		Scenario: busy.Scenario, RunId: busy.RunID, Preset: busy.Preset,
	}); derr == nil {
		cerr.AddDetail(detail)
	}
	return cerr
}

// finalizeSnapshot waits for the durable run to complete, then pins + writes the
// manifest, on a goroutine bound to a server-owned context (WithoutCancel +
// ceiling) so a disconnected client never abandons the pin. Overridable in tests
// to run synchronously.
func (s *Server) finalizeSnapshot(ctx context.Context, pending bl.PendingCapture) {
	run := s.finalize
	if run == nil {
		run = s.finalizeAsync
	}
	run(ctx, pending)
}

func (s *Server) finalizeAsync(ctx context.Context, pending bl.PendingCapture) {
	tailCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), snapshotTailCeiling)
	go func() {
		defer cancel()
		res, err := s.svc.FinalizeCapture(tailCtx, pending)
		if err != nil {
			s.logger.Printf("baselines.SnapshotForBaseline: finalize FAILED scenario=%s name=%s run=%s: %v",
				pending.Manifest.Scenario, pending.Manifest.Name, pending.Run.RunID, err)
			return
		}
		s.logger.Printf("baselines.SnapshotForBaseline: pinned scenario=%s name=%s run=%s",
			pending.Manifest.Scenario, pending.Manifest.Name, res.Manifest.RunID())
	}()
}

func (s *Server) reattachPendingSnapshots() {
	if s.svc == nil || s.repos == nil {
		return
	}
	go func() {
		ctx := context.Background()
		rid, repoDir, err := s.repos.Resolve(ctx, 0)
		if err != nil {
			s.logger.Printf("baselines.SnapshotForBaseline: pending reattach skipped: %v", err)
			return
		}
		pending, err := s.svc.PendingSnapshotCaptures(rid, repoDir)
		if err != nil {
			s.logger.Printf("baselines.SnapshotForBaseline: pending reattach failed: %v", err)
			return
		}
		for _, p := range pending {
			s.logger.Printf("baselines.SnapshotForBaseline: reattaching pending snapshot scenario=%s name=%s run=%s",
				p.Manifest.Scenario, p.Manifest.Name, p.Run.RunID)
			s.finalizeSnapshot(ctx, p)
		}
	}()
}

func (s *Server) GetBaseline(ctx context.Context, req *connect.Request[baselinesv1.GetBaselineRequest]) (*connect.Response[baselinesv1.GetBaselineResponse], error) {
	m := req.Msg
	rid, _, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("GetBaseline", err)
	}
	manifest, err := s.svc.Get(ctx, rid, m.GetScenario(), branch, m.GetName())
	if err != nil {
		return nil, s.wrap("GetBaseline", err)
	}
	return connect.NewResponse(&baselinesv1.GetBaselineResponse{Baseline: manifestToProto(manifest)}), nil
}

func (s *Server) GetSnapshotStatus(ctx context.Context, req *connect.Request[baselinesv1.GetSnapshotStatusRequest]) (*connect.Response[baselinesv1.GetSnapshotStatusResponse], error) {
	m := req.Msg
	rid, repoDir, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("GetSnapshotStatus", err)
	}
	st, err := s.svc.GetSnapshotStatus(ctx, bl.SnapshotStatusRequest{
		RepoID: rid, RepoDir: repoDir, Scenario: m.GetScenario(),
		Branch: branch, Name: m.GetName(), RunID: m.GetRunId(), Wait: m.GetWait(),
	})
	if err != nil {
		return nil, s.wrap("GetSnapshotStatus", err)
	}
	out := &baselinesv1.GetSnapshotStatusResponse{
		Status:                      st.Status,
		Scenario:                    st.Scenario,
		Name:                        st.Name,
		Branch:                      st.Branch,
		RunId:                       st.RunID,
		RunStatus:                   st.RunStatus,
		Error:                       st.Error,
		SimilarBaselines:            st.SimilarBaselines,
		RecommendedNextCheckSeconds: int32(st.RecommendedNextCheckSeconds),
	}
	if st.Baseline != nil {
		out.Baseline = manifestToProto(*st.Baseline)
	}
	return connect.NewResponse(out), nil
}

func (s *Server) ListBaselines(ctx context.Context, req *connect.Request[baselinesv1.ListBaselinesRequest]) (*connect.Response[baselinesv1.ListBaselinesResponse], error) {
	m := req.Msg
	rid, _, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), m.GetAllBranches())
	if err != nil {
		return nil, s.wrap("ListBaselines", err)
	}
	manifests, err := s.svc.List(ctx, rid, m.GetScenario(), branch)
	if err != nil {
		return nil, s.wrap("ListBaselines", err)
	}
	out := &baselinesv1.ListBaselinesResponse{Baselines: make([]*baselinesv1.BaselineManifest, 0, len(manifests))}
	for _, mf := range manifests {
		out.Baselines = append(out.Baselines, manifestToProto(mf))
	}
	return connect.NewResponse(out), nil
}

// StartDiff resolves the comprehensive run the baseline will be diffed against
// and returns its handle IMMEDIATELY — it does not block for the run. The diff
// verdict is computed + cached on a server-owned goroutine when the run
// completes (durable across client disconnect), so the caller follows the run
// and resolves the verdict with GetDiffResult. Mirrors SnapshotForBaseline.
func (s *Server) StartDiff(ctx context.Context, req *connect.Request[baselinesv1.StartDiffRequest]) (*connect.Response[baselinesv1.StartDiffResponse], error) {
	m := req.Msg
	rid, repoDir, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("StartDiff", err)
	}
	out, err := s.svc.StartDiff(ctx, bl.StartDiffRequest{
		RepoID: rid, RepoDir: repoDir, Scenario: m.GetScenario(),
		Branch: branch, Name: m.GetName(),
	})
	if err != nil {
		return nil, s.wrapStartDiff(err)
	}

	s.logger.Printf("baselines.StartDiff: scenario=%s name=%s run=%s coalesced=%t reused=%t (durable; verdict cached server-side on completion)",
		m.GetScenario(), m.GetName(), out.RunID, out.Coalesced, out.ReusedRun)

	// Finalize (await + compute + cache) on a server-owned context detached from
	// this request, so a disconnected client never abandons the verdict.
	s.finalizeDiff(ctx, out.Pending)

	return connect.NewResponse(&baselinesv1.StartDiffResponse{
		RunId:                 out.RunID,
		Scenario:              out.Scenario,
		Name:                  out.Name,
		Branch:                out.Branch,
		EstimatedTotalSeconds: int32(out.EstimatedTotalSeconds),
		EtaKnown:              out.EtaKnown,
		Coalesced:             out.Coalesced,
		ReusedRun:             out.ReusedRun,
		ReusedSha:             out.ReusedSha,
		DirtyWarning:          out.DirtyWarning,
	}), nil
}

// finalizeDiff runs FinalizeDiff on a goroutine bound to a server-owned context
// (WithoutCancel + ceiling) so a disconnected client never abandons the cached
// verdict. Overridable in tests to run synchronously.
func (s *Server) finalizeDiff(ctx context.Context, pending bl.PendingDiff) {
	run := s.finalizeDiffFn
	if run == nil {
		run = s.finalizeDiffAsync
	}
	run(ctx, pending)
}

func (s *Server) finalizeDiffAsync(ctx context.Context, pending bl.PendingDiff) {
	tailCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), snapshotTailCeiling)
	go func() {
		defer cancel()
		cd, err := s.svc.FinalizeDiff(tailCtx, pending)
		if err != nil {
			s.logger.Printf("baselines.StartDiff: finalize FAILED scenario=%s name=%s run=%s: %v",
				pending.Scenario, pending.Name, pending.CurRunID, err)
			return
		}
		verdict := ""
		if cd.Result != nil {
			verdict = string(cd.Result.Verdict)
		}
		s.logger.Printf("baselines.StartDiff: cached verdict scenario=%s name=%s run=%s verdict=%s",
			pending.Scenario, pending.Name, pending.CurRunID, verdict)
	}()
}

// GetDiffResult returns the cached diff verdict for a (baseline, run), or its
// in-flight status when the run is still executing.
func (s *Server) GetDiffResult(ctx context.Context, req *connect.Request[baselinesv1.GetDiffResultRequest]) (*connect.Response[baselinesv1.GetDiffResultResponse], error) {
	started := time.Now()
	m := req.Msg
	rid, repoDir, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("GetDiffResult", err)
	}
	cd, nextCheck, err := s.svc.GetDiffResult(ctx, bl.GetDiffResultRequest{
		RepoID: rid, RepoDir: repoDir, Scenario: m.GetScenario(),
		Branch: branch, Name: m.GetName(), RunID: m.GetRunId(),
		Wait: m.GetWait(), Latest: m.GetLatest(),
	})
	if err != nil {
		s.logger.Printf("baselines.GetDiffResult: scenario=%s name=%s run=%s latest=%t wait=%t status=error duration=%s err=%v",
			m.GetScenario(), m.GetName(), m.GetRunId(), m.GetLatest(), m.GetWait(), time.Since(started), err)
		return nil, s.wrap("GetDiffResult", err)
	}
	runID := cd.RunID
	if runID == "" {
		runID = m.GetRunId()
	}
	verdict := ""
	if cd.Result != nil {
		verdict = string(cd.Result.Verdict)
	}
	s.logger.Printf("baselines.GetDiffResult: scenario=%s name=%s run=%s latest=%t wait=%t status=%s verdict=%s next_check=%d duration=%s",
		m.GetScenario(), m.GetName(), runID, m.GetLatest(), m.GetWait(), cd.Status, verdict, nextCheck, time.Since(started))
	out := &baselinesv1.GetDiffResultResponse{
		Status:                      cd.Status,
		Error:                       cd.Error,
		RunId:                       runID,
		RecommendedNextCheckSeconds: int32(nextCheck),
	}
	if cd.Result != nil {
		out.Diff = diffResultToProto(*cd.Result)
	}
	return connect.NewResponse(out), nil
}

// wrapStartDiff maps a StartDiff rejection. A divergent in-flight run surfaces as
// FailedPrecondition carrying RunBusyInfo so the CLI renders wait/abort guidance.
func (s *Server) wrapStartDiff(err error) error {
	var busy *bl.RunBusyError
	if errors.As(err, &busy) {
		return busyConnectError(busy)
	}
	return s.wrap("StartDiff", err)
}

func (s *Server) DeleteBaseline(ctx context.Context, req *connect.Request[baselinesv1.DeleteBaselineRequest]) (*connect.Response[baselinesv1.DeleteBaselineResponse], error) {
	m := req.Msg
	rid, _, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("DeleteBaseline", err)
	}
	if err := s.svc.Delete(ctx, rid, m.GetScenario(), branch, m.GetName()); err != nil {
		return nil, s.wrap("DeleteBaseline", err)
	}
	return connect.NewResponse(&baselinesv1.DeleteBaselineResponse{Deleted: true}), nil
}

// StartCollectionCapture creates or resumes a durable multi-scenario capture.
// It returns after child runs are started; each child remains finalized by the
// server even if this transport disconnects.
func (s *Server) StartCollectionCapture(ctx context.Context, req *connect.Request[baselinesv1.StartCollectionCaptureRequest]) (*connect.Response[baselinesv1.StartCollectionCaptureResponse], error) {
	m := req.Msg
	rid, repoDir, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("StartCollectionCapture", err)
	}
	targets := make([]bl.CollectionTarget, 0, len(m.GetTargets()))
	for _, target := range m.GetTargets() {
		targets = append(targets, bl.CollectionTarget{Scenario: target.GetScenario(), BaselineName: target.GetBaselineName(), Required: target.GetRequired()})
	}
	started, err := s.svc.StartCollectionCapture(ctx, bl.StartCollectionCaptureRequest{
		RepoID: rid, RepoDir: repoDir, Branch: branch, Name: m.GetName(), Targets: targets, PathSelections: m.GetPathSelections(), PathPolicy: bl.PathSnapshotPolicy{IncludeIgnored: m.GetIncludeIgnored(), RetainContent: m.GetRetainContent()}, CreatedBy: m.GetCreatedBy(), Reason: m.GetReason(),
	})
	if err != nil {
		var policyErr *bl.PathSnapshotPolicyError
		if errors.As(err, &policyErr) {
			return nil, pathSnapshotPolicyConnectError(policyErr)
		}
		return nil, s.wrap("StartCollectionCapture", err)
	}
	for _, pending := range started.Pending {
		s.finalizeCollectionCapture(ctx, rid, pending)
	}
	return connect.NewResponse(&baselinesv1.StartCollectionCaptureResponse{Collection: collectionToProto(started.Collection), Resumed: started.Resumed}), nil
}

func (s *Server) EstimatePathSnapshot(ctx context.Context, req *connect.Request[baselinesv1.EstimatePathSnapshotRequest]) (*connect.Response[baselinesv1.EstimatePathSnapshotResponse], error) {
	m := req.Msg
	_, repoDir, _, err := s.resolveTarget(ctx, m.GetRepoId(), "", false)
	if err != nil {
		return nil, s.wrap("EstimatePathSnapshot", err)
	}
	estimate, err := s.svc.EstimatePathSnapshot(ctx, repoDir, m.GetSelections(), bl.PathSnapshotPolicy{IncludeIgnored: m.GetIncludeIgnored(), RetainContent: m.GetRetainContent()})
	if err != nil {
		return nil, s.wrap("EstimatePathSnapshot", err)
	}
	return connect.NewResponse(&baselinesv1.EstimatePathSnapshotResponse{Estimate: pathSnapshotEstimateToProto(estimate)}), nil
}

// ExtendCollection appends newly discovered, pre-edit scenarios to an existing
// immutable collection. Existing members can never be changed through this RPC.
func (s *Server) ExtendCollection(ctx context.Context, req *connect.Request[baselinesv1.ExtendCollectionRequest]) (*connect.Response[baselinesv1.ExtendCollectionResponse], error) {
	m := req.Msg
	rid, repoDir, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("ExtendCollection", err)
	}
	targets := make([]bl.CollectionTarget, 0, len(m.GetTargets()))
	for _, target := range m.GetTargets() {
		targets = append(targets, bl.CollectionTarget{Scenario: target.GetScenario(), BaselineName: target.GetBaselineName(), Required: target.GetRequired()})
	}
	started, err := s.svc.ExtendCollection(ctx, bl.ExtendCollectionRequest{RepoID: rid, RepoDir: repoDir, Branch: branch, Name: m.GetName(), Targets: targets, CreatedBy: m.GetCreatedBy(), Reason: m.GetReason()})
	if err != nil {
		return nil, s.wrap("ExtendCollection", err)
	}
	for _, pending := range started.Pending {
		s.finalizeCollectionCapture(ctx, rid, pending)
	}
	return connect.NewResponse(&baselinesv1.ExtendCollectionResponse{Collection: collectionToProto(started.Collection), AddedScenarios: started.AddedScenarios, Resumed: started.Resumed}), nil
}

func (s *Server) finalizeCollectionCapture(ctx context.Context, repoID int64, pending bl.PendingCollectionCapture) {
	if s.finalizeCollection != nil {
		s.finalizeCollection(ctx, repoID, pending)
		return
	}
	tailCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), snapshotTailCeiling)
	go func() {
		defer cancel()
		if _, err := s.svc.FinalizeCollectionCapture(tailCtx, repoID, pending); err != nil {
			s.logger.Printf("baselines.StartCollectionCapture: finalize failed collection=%s scenario=%s run=%s: %v", pending.CollectionName, pending.Scenario, pending.Pending.Run.RunID, err)
		}
	}()
}

func (s *Server) GetCollection(ctx context.Context, req *connect.Request[baselinesv1.GetCollectionRequest]) (*connect.Response[baselinesv1.GetCollectionResponse], error) {
	m := req.Msg
	rid, _, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("GetCollection", err)
	}
	var collection bl.CollectionManifest
	if m.GetWait() {
		collection, err = s.svc.ResumeCollectionCapture(ctx, rid, branch, m.GetName())
	} else {
		// A non-wait read must still project already-terminal children. This
		// recovers a collection whose detached finalizer was interrupted by an
		// API restart without turning ordinary status reads into waits.
		collection, err = s.svc.ReconcileCollectionCapture(ctx, rid, branch, m.GetName())
	}
	if err != nil {
		return nil, s.wrap("GetCollection", err)
	}
	return connect.NewResponse(&baselinesv1.GetCollectionResponse{Collection: collectionToProto(collection)}), nil
}

// StartCollectionDiff starts durable child diffs for the explicit selection.
// The immediate response is intentionally a handle report, not a fabricated
// final verdict; callers reattach to each returned child run through the normal
// durable diff status endpoint while the server-owned finalizers persist them.
func (s *Server) StartCollectionDiff(ctx context.Context, req *connect.Request[baselinesv1.StartCollectionDiffRequest]) (*connect.Response[baselinesv1.StartCollectionDiffResponse], error) {
	m := req.Msg
	rid, repoDir, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("StartCollectionDiff", err)
	}
	started, err := s.svc.StartCollectionDiff(ctx, bl.StartCollectionDiffRequest{RepoID: rid, RepoDir: repoDir, Branch: branch, Name: m.GetName(), OperationID: m.GetOperationId(), Scenarios: m.GetScenarios()})
	if err != nil {
		return nil, s.wrap("StartCollectionDiff", err)
	}
	for _, pending := range started.Pending {
		s.finalizeCollectionDiff(ctx, rid, pending)
	}
	aggregate := bl.AggregateCollectionDiff(started.Collection, started.Members)
	out := &baselinesv1.StartCollectionDiffResponse{Collection: collectionToProto(started.Collection), Classification: string(aggregate.Verdict), OperationId: started.Operation.ID}
	for _, member := range started.Members {
		out.Members = append(out.Members, collectionDiffMemberToProto(member))
	}
	return connect.NewResponse(out), nil
}

func (s *Server) GetCollectionDiff(ctx context.Context, req *connect.Request[baselinesv1.GetCollectionDiffRequest]) (*connect.Response[baselinesv1.GetCollectionDiffResponse], error) {
	m := req.Msg
	rid, _, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("GetCollectionDiff", err)
	}
	collection, operation, err := s.svc.GetCollectionDiff(ctx, rid, branch, m.GetName(), m.GetOperationId(), m.GetWait())
	if err != nil {
		return nil, s.wrap("GetCollectionDiff", err)
	}
	aggregate := operation.Aggregate(collection)
	out := &baselinesv1.GetCollectionDiffResponse{Collection: collectionToProto(collection), Classification: string(aggregate.Verdict), OperationId: operation.ID}
	for _, member := range operation.Members {
		out.Members = append(out.Members, collectionDiffMemberToProto(member))
	}
	return connect.NewResponse(out), nil
}

func (s *Server) finalizeCollectionDiff(ctx context.Context, repoID int64, pending bl.PendingCollectionDiff) {
	if s.finalizeCollectionDiffFn != nil {
		s.finalizeCollectionDiffFn(ctx, repoID, pending)
		return
	}
	tailCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), snapshotTailCeiling)
	go func() {
		defer cancel()
		if _, err := s.svc.FinalizeCollectionDiff(tailCtx, repoID, pending); err != nil {
			s.logger.Printf("baselines.StartCollectionDiff: finalize failed collection=%s operation=%s scenario=%s: %v", pending.CollectionName, pending.OperationID, pending.Scenario, err)
		}
	}()
}

func (s *Server) DeleteCollection(ctx context.Context, req *connect.Request[baselinesv1.DeleteCollectionRequest]) (*connect.Response[baselinesv1.DeleteCollectionResponse], error) {
	m := req.Msg
	rid, _, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("DeleteCollection", err)
	}
	if err := s.svc.DeleteCollection(ctx, rid, branch, m.GetName()); err != nil {
		return nil, s.wrap("DeleteCollection", err)
	}
	return connect.NewResponse(&baselinesv1.DeleteCollectionResponse{Deleted: true}), nil
}

func (s *Server) CapturePathSnapshot(ctx context.Context, req *connect.Request[baselinesv1.CapturePathSnapshotRequest]) (*connect.Response[baselinesv1.CapturePathSnapshotResponse], error) {
	m := req.Msg
	rid, repoDir, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("CapturePathSnapshot", err)
	}
	captured, err := s.svc.CapturePathSnapshot(ctx, bl.CapturePathSnapshotRequest{RepoID: rid, RepoDir: repoDir, Branch: branch, Name: m.GetName(), Selections: m.GetSelections(), Retention: time.Duration(m.GetRetentionSeconds()) * time.Second, Policy: bl.PathSnapshotPolicy{IncludeIgnored: m.GetIncludeIgnored(), RetainContent: m.GetRetainContent()}})
	if err != nil {
		var policyErr *bl.PathSnapshotPolicyError
		if errors.As(err, &policyErr) {
			return nil, pathSnapshotPolicyConnectError(policyErr)
		}
		return nil, s.wrap("CapturePathSnapshot", err)
	}
	return connect.NewResponse(&baselinesv1.CapturePathSnapshotResponse{Snapshot: pathSnapshotToProto(captured.Snapshot), Resumed: captured.Resumed}), nil
}

func pathSnapshotPolicyConnectError(policyErr *bl.PathSnapshotPolicyError) error {
	cerr := connect.NewError(connect.CodeFailedPrecondition, policyErr)
	if detail, err := connect.NewErrorDetail(&baselinesv1.PathSnapshotPolicyViolation{Estimate: pathSnapshotEstimateToProto(policyErr.Estimate)}); err == nil {
		cerr.AddDetail(detail)
	}
	return cerr
}

func (s *Server) GetPathSnapshot(ctx context.Context, req *connect.Request[baselinesv1.GetPathSnapshotRequest]) (*connect.Response[baselinesv1.GetPathSnapshotResponse], error) {
	m := req.Msg
	rid, _, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("GetPathSnapshot", err)
	}
	snapshot, err := s.svc.StorageLoadPathSnapshot(rid, branch, m.GetName())
	if err != nil {
		return nil, s.wrap("GetPathSnapshot", err)
	}
	return connect.NewResponse(&baselinesv1.GetPathSnapshotResponse{Snapshot: pathSnapshotToProto(snapshot)}), nil
}

func (s *Server) DiffPathSnapshots(ctx context.Context, req *connect.Request[baselinesv1.DiffPathSnapshotsRequest]) (*connect.Response[baselinesv1.DiffPathSnapshotsResponse], error) {
	m := req.Msg
	rid, _, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("DiffPathSnapshots", err)
	}
	before, err := s.svc.StorageLoadPathSnapshot(rid, branch, m.GetBeforeName())
	if err != nil {
		return nil, s.wrap("DiffPathSnapshots", err)
	}
	after, err := s.svc.StorageLoadPathSnapshot(rid, branch, m.GetAfterName())
	if err != nil {
		return nil, s.wrap("DiffPathSnapshots", err)
	}
	out := &baselinesv1.DiffPathSnapshotsResponse{Classification: "informational-source-evidence"}
	deltas, err := bl.FilterSourceDeltas(bl.DiffPathSnapshots(before, after), m.GetSelections())
	if err != nil {
		return nil, s.wrap("DiffPathSnapshots", err)
	}
	for _, delta := range deltas {
		out.Deltas = append(out.Deltas, sourceDeltaToProto(delta))
	}
	return connect.NewResponse(out), nil
}

func (s *Server) DeletePathSnapshot(ctx context.Context, req *connect.Request[baselinesv1.DeletePathSnapshotRequest]) (*connect.Response[baselinesv1.DeletePathSnapshotResponse], error) {
	m := req.Msg
	rid, _, branch, err := s.resolveTarget(ctx, m.GetRepoId(), m.GetBranch(), false)
	if err != nil {
		return nil, s.wrap("DeletePathSnapshot", err)
	}
	if err := s.svc.DeletePathSnapshot(ctx, rid, branch, m.GetName()); err != nil {
		return nil, s.wrap("DeletePathSnapshot", err)
	}
	return connect.NewResponse(&baselinesv1.DeletePathSnapshotResponse{Deleted: true}), nil
}

// wrap maps domain errors to Connect codes and logs internal ones.
func (s *Server) wrap(op string, err error) error {
	switch {
	case errors.Is(err, bl.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, bl.ErrAlreadyExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, bl.ErrSnapshotQuota):
		return connect.NewError(connect.CodeResourceExhausted, err)
	}
	s.logger.Printf("baselines.%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, err)
}
