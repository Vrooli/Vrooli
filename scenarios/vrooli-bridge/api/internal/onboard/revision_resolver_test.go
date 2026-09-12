package onboard_test

import (
	"context"
	"errors"
	"testing"

	"vrooli-bridge/internal/onboard"
	"vrooli-bridge/internal/onboard/mocks"

	"github.com/vrooli/api-core/schedule"

	"github.com/stretchr/testify/require"
)

// fakeRevResolver is a scriptable onboard.RevisionResolver. It records the last
// requested revision and returns a canned resolution or error.
type fakeRevResolver struct {
	resolved    string
	err         error
	lastRequest string
	// workingTreeErr, when set, is returned by ResolveWorkingTree only (so a test
	// can prove pinned Resolve and working-tree ResolveWorkingTree diverge).
	workingTreeErr error
	// calledResolve / calledWorkingTree record which pipeline the service chose.
	calledResolve     bool
	calledWorkingTree bool
}

func (f *fakeRevResolver) Resolve(_ context.Context, requested string) (string, error) {
	f.lastRequest = requested
	f.calledResolve = true
	if f.err != nil {
		return "", f.err
	}
	if f.resolved != "" {
		return f.resolved, nil
	}
	return requested, nil
}

func (f *fakeRevResolver) ResolveWorkingTree(_ context.Context, requested string) (string, error) {
	f.lastRequest = requested
	f.calledWorkingTree = true
	if f.workingTreeErr != nil {
		return "", f.workingTreeErr
	}
	if f.err != nil {
		return "", f.err
	}
	if f.resolved != "" {
		return f.resolved, nil
	}
	return requested, nil
}

func newResolverService(repo *mocks.FakeRepository, driver *mocks.FakeSSHDriver, issuer *mocks.FakeCodeIssuer, confirmer *mocks.FakeOnlineConfirmer, res onboard.RevisionResolver) onboard.Service {
	return onboard.NewService(repo, driver, issuer, confirmer, schedule.System(), onboard.WithRevisionResolver(res), onboard.WithEnrollmentResolver(fixedEnrollmentResolver{nodeID: testNodeID, paired: true}))
}

// TestStart_OmittedRevisionDefaultsViaResolver is the core phase-6 acceptance:
// StartOnboarding no longer requires target_revision — an omitted revision is
// resolved to the control plane's commit and pinned on the durable op.
func TestStart_OmittedRevisionDefaultsViaResolver(t *testing.T) {
	const cpCommit = "1111111111111111111111111111111111111111"
	repo := mocks.NewFakeRepository()
	driver := &mocks.FakeSSHDriver{RunBootstrapMarkers: successMarkers(testNodeID)}
	res := &fakeRevResolver{resolved: cpCommit}
	svc := newResolverService(repo, driver, &mocks.FakeCodeIssuer{Code: testCode}, &mocks.FakeOnlineConfirmer{Online: true}, res)

	in := validInput()
	in.TargetRevision = "" // omitted — the whole point of the phase
	dec, err := svc.Start(context.Background(), in)
	require.NoError(t, err)
	require.NotEmpty(t, dec.OpID)
	require.Equal(t, "", res.lastRequest, "resolver received the omitted (empty) revision")

	op := waitTerminal(t, svc, dec.OpID)
	require.Equal(t, cpCommit, op.TargetRevision, "op pins to the resolved control-plane commit")
	require.Contains(t, driver.CapturedArgs, cpCommit, "the bootstrap --revision carries the resolved commit")
}

func TestStart_SentinelExpandedViaResolver(t *testing.T) {
	const cpCommit = "2222222222222222222222222222222222222222"
	repo := mocks.NewFakeRepository()
	driver := &mocks.FakeSSHDriver{RunBootstrapMarkers: successMarkers(testNodeID)}
	res := &fakeRevResolver{resolved: cpCommit}
	svc := newResolverService(repo, driver, &mocks.FakeCodeIssuer{Code: testCode}, &mocks.FakeOnlineConfirmer{Online: true}, res)

	in := validInput()
	in.TargetRevision = "@cp"
	dec, err := svc.Start(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, "@cp", res.lastRequest)

	op := waitTerminal(t, svc, dec.OpID)
	require.Equal(t, cpCommit, op.TargetRevision)
}

// TestStart_ResolverErrorFailsClosed asserts a preflight/validation failure from
// the resolver aborts Start before any op is created or host is touched, and the
// password is zeroed.
func TestStart_ResolverErrorFailsClosed(t *testing.T) {
	repo := mocks.NewFakeRepository()
	driver := &mocks.FakeSSHDriver{RunBootstrapMarkers: successMarkers(testNodeID)}
	resErr := errors.New("commit abc123 is not on remote \"origin\"; push it first")
	res := &fakeRevResolver{err: resErr}
	svc := newResolverService(repo, driver, &mocks.FakeCodeIssuer{Code: testCode}, &mocks.FakeOnlineConfirmer{Online: true}, res)

	in := validInput()
	pw := in.Password
	_, err := svc.Start(context.Background(), in)
	require.ErrorIs(t, err, resErr)
	ops, listErr := svc.ListOps(context.Background(), onboard.ListFilter{})
	require.NoError(t, listErr)
	require.Empty(t, ops, "no op is created when preflight fails")
	require.Equal(t, 0, driver.FirstTouchCalls, "no host is touched when preflight fails")
	for _, b := range pw {
		require.Equal(t, byte(0), b, "password must be zeroed on the failure path")
	}
}
