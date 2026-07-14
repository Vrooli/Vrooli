package onboard_test

import (
	"context"
	"testing"

	onboardhandler "vrooli-bridge/handlers/onboard"
	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/clock"
	"vrooli-bridge/internal/cprev"
	"vrooli-bridge/internal/onboard"
	"vrooli-bridge/internal/onboard/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	onboardv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard"
)

// stubResolver is a handler-test onboard.RevisionResolver.
type stubResolver struct {
	resolved string
	err      error
	last     string
}

func (s *stubResolver) Resolve(_ context.Context, requested string) (string, error) {
	s.last = requested
	if s.err != nil {
		return "", s.err
	}
	return s.resolved, nil
}

// startHandler is the slice of the onboard Connect handler these tests drive
// (NewConnectHandler returns an unexported concrete type).
type startHandler interface {
	StartOnboarding(context.Context, *connect.Request[onboardv1.StartOnboardingRequest]) (*connect.Response[onboardv1.StartOnboardingResponse], error)
}

func handlerWithResolver(res onboard.RevisionResolver) startHandler {
	svc := onboard.NewService(
		mocks.NewFakeRepository(),
		&mocks.FakeSSHDriver{},
		&mocks.FakeCodeIssuer{Code: "PAIRINGCODEABCDEF0123456789ABCDEF"},
		&mocks.FakeOnlineConfirmer{Online: true},
		clock.System{},
		onboard.WithRevisionResolver(res),
	)
	return onboardhandler.NewConnectHandler(onboardhandler.Deps{Service: svc})
}

func dryRunRequest() *connect.Request[onboardv1.StartOnboardingRequest] {
	req := connect.NewRequest(&onboardv1.StartOnboardingRequest{
		Host:            "web-01.example.com",
		User:            "deploy",
		SshPassword:     "pw",
		NodeName:        "web-01",
		ControlPlaneUrl: "https://cp.example.com",
		// TargetRevision deliberately omitted.
	})
	req.Header().Set("X-Dry-Run", "true")
	return req
}

// TestStartOnboarding_OmittedRevisionAcceptedAtBoundary is the phase-6 handler
// acceptance: with the resolver wired, StartOnboarding no longer 400s on a
// missing target_revision — it defaults it and (here, dry-run) validates.
func TestStartOnboarding_OmittedRevisionAcceptedAtBoundary(t *testing.T) {
	res := &stubResolver{resolved: "1111111111111111111111111111111111111111"}
	h := handlerWithResolver(res)
	ownerCtx := auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1"})

	resp, err := h.StartOnboarding(ownerCtx, dryRunRequest())
	require.NoError(t, err, "omitted revision must be accepted, not rejected")
	require.True(t, resp.Msg.GetDryRun())
	require.Equal(t, "", res.last, "resolver saw the omitted (empty) revision")
}

// TestStartOnboarding_MetacharRevisionFriendlyRejection asserts a relative ref is
// rejected at the API boundary with InvalidArgument (not an opaque privsep
// failure on the node).
func TestStartOnboarding_MetacharRevisionFriendlyRejection(t *testing.T) {
	res := &stubResolver{err: cprev.ErrUnsafeRevision{Revision: "HEAD~1", Reason: "relative ref"}}
	h := handlerWithResolver(res)
	ownerCtx := auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1"})

	_, err := h.StartOnboarding(ownerCtx, dryRunRequest())
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Contains(t, err.Error(), "HEAD~1")
}

// TestStartOnboarding_UnpushedRevisionPreconditionFailure asserts an unpushed
// commit surfaces as FailedPrecondition with push-first guidance.
func TestStartOnboarding_UnpushedRevisionPreconditionFailure(t *testing.T) {
	res := &stubResolver{err: cprev.ErrNotPushed{Commit: "abc123", Remote: "origin"}}
	h := handlerWithResolver(res)
	ownerCtx := auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1"})

	_, err := h.StartOnboarding(ownerCtx, dryRunRequest())
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.Contains(t, err.Error(), "push it first")
}
