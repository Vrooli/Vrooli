package onboard_test

import (
	"context"
	"testing"

	"vrooli-bridge/internal/clock"
	"vrooli-bridge/internal/onboard"
	"vrooli-bridge/internal/onboard/mocks"

	"github.com/stretchr/testify/require"
)

// workingTreeService wires a service with the working-tree seams (snapshot source
// + node-revision recorder) in addition to the standard fakes.
func workingTreeService(
	repo *mocks.FakeRepository,
	driver *mocks.FakeSSHDriver,
	issuer *mocks.FakeCodeIssuer,
	confirmer *mocks.FakeOnlineConfirmer,
	src *mocks.FakeWorkingTreeSource,
	rec *mocks.FakeNodeRevisionRecorder,
	res onboard.RevisionResolver,
) onboard.Service {
	return onboard.NewService(repo, driver, issuer, confirmer, clock.System{},
		onboard.WithRevisionResolver(res),
		onboard.WithWorkingTreeSource(src),
		onboard.WithArtifactBuilder(&mocks.FakeArtifactBuilder{}),
		onboard.WithNodeRevisionRecorder(rec),
	)
}

const wtBase = "e767613fcadeadbeef00000000000000000000ff"

func TestStart_WorkingTreeShipsTreeAndRecordsDirtyProvenance(t *testing.T) {
	repo := mocks.NewFakeRepository()
	driver := &mocks.FakeSSHDriver{
		RunBootstrapMarkers: successMarkers(testNodeID),
		SyncTreeResult:      onboard.SyncResult{BytesTransferred: 4096, ResolvedDestDir: "/home/deploy/vrooli"},
	}
	issuer := &mocks.FakeCodeIssuer{Code: testCode}
	confirmer := &mocks.FakeOnlineConfirmer{Online: true}
	src := &mocks.FakeWorkingTreeSource{Snapshot_: onboard.WorkingTreeSnapshot{
		BaseHEAD: wtBase,
		Digest:   "digestcafebabe1234",
		RepoDir:  "/cp/repo",
		Files:    []string{"AGENTS.md", "scenarios/vrooli-bridge/bootstrap/bootstrap.sh"},
	}}
	rec := &mocks.FakeNodeRevisionRecorder{}
	// The resolver returns the base HEAD; the calledWorkingTree/calledResolve flags
	// prove which pipeline the service chose.
	res := &fakeRevResolver{resolved: wtBase}
	svc := workingTreeService(repo, driver, issuer, confirmer, src, rec, res)

	in := validInput()
	in.SourceMode = onboard.SourceModeWorkingTree
	in.TargetRevision = "" // default to CP HEAD via the resolver

	dec, err := svc.Start(context.Background(), in)
	require.NoError(t, err)

	op := waitTerminal(t, svc, dec.OpID)
	require.Equal(t, onboard.StateSucceeded, op.State)

	// The resolver's working-tree pipeline ran, not the pinned one.
	require.True(t, res.calledWorkingTree, "working-tree mode must use ResolveWorkingTree")
	require.False(t, res.calledResolve, "working-tree mode must NOT call the pinned Resolve")

	// The tree was snapshotted and shipped exactly once, with the enumerated files.
	require.Equal(t, 1, src.Calls)
	require.Equal(t, 1, driver.SyncTreeCalls)
	require.Equal(t, src.Snapshot_.Files, driver.CapturedSyncTree.Files)
	require.Equal(t, "/cp/repo", driver.CapturedSyncTree.RepoDir)
	require.Equal(t, 1, driver.DetectPlatformCalls)
	require.Equal(t, 1, driver.PushArtifactsCalls)

	// Dirty provenance is persisted on the op: source mode, base, digest, and a
	// TargetRevision that renders "<base>+dirty".
	require.Equal(t, onboard.SourceModeWorkingTree, op.SourceMode)
	require.Equal(t, wtBase, op.BaseRevision)
	require.Equal(t, "digestcafebabe1234", op.WorkingTreeDigest)
	require.Equal(t, wtBase+"+dirty", op.TargetRevision)

	// The bootstrap was pointed at the pre-synced tree (never a clone), with the
	// resolved dest and the digest, and --revision carrying the base (not +dirty).
	require.Contains(t, driver.CapturedArgs, "--source-dir")
	require.Contains(t, driver.CapturedArgs, "/home/deploy/vrooli")
	require.Contains(t, driver.CapturedArgs, "--source-digest")
	require.Contains(t, driver.CapturedArgs, "digestcafebabe1234")
	require.Contains(t, driver.CapturedArgs, wtBase)
	require.Contains(t, driver.CapturedArgs, "--vrooli-bin")
	require.Contains(t, driver.CapturedArgs, "/tmp/artifacts/vrooli")
	require.Contains(t, driver.CapturedArgs, "--bridge-cli")
	require.Contains(t, driver.CapturedArgs, "--agent-bin")

	// A sync-tree step event was persisted.
	_, events, err := svc.GetOp(context.Background(), dec.OpID)
	require.NoError(t, err)
	sawSync := false
	for _, ev := range events {
		if ev.StepID == onboard.StepSyncTree && ev.Status == onboard.StepStatusOK {
			sawSync = true
		}
	}
	require.True(t, sawSync, "expected a sync-tree OK step event")

	// The node record was stamped with the dirty provenance.
	require.Equal(t, 1, rec.Calls)
	require.Equal(t, testNodeID, rec.LastNodeID)
	require.Equal(t, wtBase+"+dirty", rec.LastRevision)
}

func TestStart_PinnedModeNeverShipsTree(t *testing.T) {
	repo := mocks.NewFakeRepository()
	driver := &mocks.FakeSSHDriver{RunBootstrapMarkers: successMarkers(testNodeID)}
	src := &mocks.FakeWorkingTreeSource{}
	rec := &mocks.FakeNodeRevisionRecorder{}
	res := &fakeRevResolver{resolved: "1111111111111111111111111111111111111111"}
	svc := workingTreeService(repo, driver, &mocks.FakeCodeIssuer{Code: testCode}, &mocks.FakeOnlineConfirmer{Online: true}, src, rec, res)

	in := validInput()
	in.SourceMode = onboard.SourceModePinned

	dec, err := svc.Start(context.Background(), in)
	require.NoError(t, err)
	op := waitTerminal(t, svc, dec.OpID)
	require.Equal(t, onboard.StateSucceeded, op.State)

	require.True(t, res.calledResolve, "pinned mode uses the pinned Resolve")
	require.False(t, res.calledWorkingTree)
	require.Equal(t, 0, src.Calls, "pinned mode must not snapshot the working tree")
	require.Equal(t, 0, driver.SyncTreeCalls, "pinned mode must not ship a tree")
	require.Equal(t, 0, driver.DetectPlatformCalls, "pinned mode does not cross-build live-tree artifacts")
	require.Equal(t, 0, driver.PushArtifactsCalls, "pinned mode does not transfer live-tree artifacts")
	require.NotContains(t, driver.CapturedArgs, "--source-dir")
	require.Equal(t, onboard.SourceModePinned, op.SourceMode)
	require.Empty(t, op.WorkingTreeDigest)
	// Pinned mode still stamps the node with the pinned commit.
	require.Equal(t, "1111111111111111111111111111111111111111", rec.LastRevision)
}

func TestStart_WorkingTreeWithoutSourceIsRefused(t *testing.T) {
	// No WorkingTreeSource wired: a working-tree request must fail loudly at Start
	// rather than silently degrading to a pinned clone.
	repo := mocks.NewFakeRepository()
	driver := &mocks.FakeSSHDriver{RunBootstrapMarkers: successMarkers(testNodeID)}
	res := &fakeRevResolver{resolved: wtBase}
	svc := onboard.NewService(repo, driver, &mocks.FakeCodeIssuer{Code: testCode}, &mocks.FakeOnlineConfirmer{Online: true}, clock.System{},
		onboard.WithRevisionResolver(res))

	in := validInput()
	in.SourceMode = onboard.SourceModeWorkingTree
	_, err := svc.Start(context.Background(), in)
	require.Error(t, err)
	var invalid onboard.ErrInvalid
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "source_mode", invalid.Field)
}
