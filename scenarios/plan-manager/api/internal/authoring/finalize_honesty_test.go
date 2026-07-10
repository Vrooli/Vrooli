package authoring_test

import (
	"context"
	"testing"

	"plan-manager/internal/authoring"
	internalplans "plan-manager/internal/plans"

	"github.com/stretchr/testify/require"
)

// TestFinalizeReportsComputedMirrorAndStoreIdentity covers the honest-finalize
// contract: the response carries the COMPUTED mirror publish result (threaded
// from the write, not the read-model default), the physical store path, the
// stamped workspace, and a finalize timestamp.
func TestFinalizeReportsComputedMirrorAndStoreIdentity(t *testing.T) {
	writer := &fakePlanWriter{mirror: internalplans.RenderedPlanMirror{
		Path:         "/home/user/.vrooli/plans/improve-widget.md",
		RelativePath: "improve-widget.md",
		Status:       internalplans.RenderedMirrorStatusFresh,
	}}
	svc := newService(t, authoring.Deps{Writer: writer, StorePath: "/data/plan-manager.db"})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Honest finalize", "honest-finalize", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	result, _, err := svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{WorkspaceRoot: "/repo/root"})
	require.NoError(t, err)
	require.False(t, result.AlreadyFinalized)
	require.Equal(t, "/data/plan-manager.db", result.StorePath)
	require.NotEmpty(t, result.FinalizedAt)
	require.Equal(t, internalplans.RenderedMirrorStatusFresh, result.Mirror.Status)
	require.Equal(t, "improve-widget.md", result.Mirror.RelativePath)
	require.Equal(t, "/repo/root", writer.created.WorkspaceRoot,
		"finalize must stamp the caller's workspace root on the draft it persists")
}

func TestFinalizeReadbackUsesFinalizedWorkspace(t *testing.T) {
	writer := &fakePlanWriter{}
	reader := &fakePlanReader{plans: map[string]internalplans.Plan{
		"plan-finalized": {ID: "plan-finalized", WorkspaceRoot: "/repo/workspace-a"},
	}}
	svc := newService(t, authoring.Deps{Writer: writer, Reader: reader})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Scoped finalize", "", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	_, _, err = svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{WorkspaceRoot: "/repo/workspace-a"})
	require.NoError(t, err)
	require.Equal(t, []string{"/repo/workspace-a"}, reader.getWorkspaces)
	require.Equal(t, []string{"/repo/workspace-a"}, reader.renderWorkspaces)
}

// TestFinalizeUnreportedMirrorSurfacesAsWriteFailed: a plans store that does
// not report a mirror publish result (default/unknown status) means no mirror
// file was written — finalize must surface write_failed with the reason, never
// echo the silent default.
func TestFinalizeUnreportedMirrorSurfacesAsWriteFailed(t *testing.T) {
	for _, status := range []internalplans.RenderedMirrorStatus{
		internalplans.RenderedMirrorStatusUnspecified,
		internalplans.RenderedMirrorStatusUnknown,
	} {
		writer := &fakePlanWriter{mirror: internalplans.RenderedPlanMirror{Status: status}}
		svc := newService(t, authoring.Deps{Writer: writer})
		ctx := context.Background()
		sess, _, err := svc.StartSession(ctx, "Silent mirror", "", "")
		require.NoError(t, err)
		fillMandatory(t, svc, sess.ID)

		result, _, err := svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{})
		require.NoError(t, err)
		require.Equal(t, internalplans.RenderedMirrorStatusWriteFailed, result.Mirror.Status,
			"unreported mirror state (%q) must surface as write_failed", status)
		require.Contains(t, result.Mirror.LastError, "no mirror file was written")
	}
}

// TestFinalizeThreadsWriteFailedMirror: an explicit write_failed publish result
// (e.g. read-only plans dir) passes through with its error; the plan is still
// persisted (SQLite is SSOT) so finalize succeeds — loudly, not silently.
func TestFinalizeThreadsWriteFailedMirror(t *testing.T) {
	writer := &fakePlanWriter{mirror: internalplans.RenderedPlanMirror{
		Path:      "/home/user/.vrooli/plans/broken.md",
		Status:    internalplans.RenderedMirrorStatusWriteFailed,
		LastError: "open /home/user/.vrooli/plans/broken.md: permission denied",
	}}
	svc := newService(t, authoring.Deps{Writer: writer})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Broken mirror", "", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	result, _, err := svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{})
	require.NoError(t, err, "a failed mirror write must not fail finalize — the plan row persisted")
	require.Equal(t, internalplans.RenderedMirrorStatusWriteFailed, result.Mirror.Status)
	require.Contains(t, result.Mirror.LastError, "permission denied")
	require.NotEmpty(t, result.Plan.ID)
}

// TestFinalizeIdempotentRerunReportsAlreadyFinalized: re-running finalize on a
// finalized session must say so explicitly instead of returning
// indistinguishable happy output.
func TestFinalizeIdempotentRerunReportsAlreadyFinalized(t *testing.T) {
	writer := &fakePlanWriter{}
	reader := &fakePlanReader{plans: map[string]internalplans.Plan{
		"plan-finalized": {
			ID:     "plan-finalized",
			Slug:   "already-done",
			Title:  "Already done",
			Status: internalplans.PlanStatusDraft,
		},
	}}
	svc := newService(t, authoring.Deps{Writer: writer, Reader: reader, StorePath: "/data/plan-manager.db"})
	ctx := context.Background()
	sess, _, err := svc.StartSession(ctx, "Already done", "already-done", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	first, _, err := svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{})
	require.NoError(t, err)
	require.False(t, first.AlreadyFinalized)

	second, _, err := svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{})
	require.NoError(t, err)
	require.True(t, second.AlreadyFinalized, "re-run must be marked as the idempotent short-circuit")
	require.Equal(t, first.Plan.ID, second.Plan.ID)
	require.NotEmpty(t, second.FinalizedAt)
	require.Equal(t, "/data/plan-manager.db", second.StorePath)
	require.Equal(t, 1, writer.calls)
}
