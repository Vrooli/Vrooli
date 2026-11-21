package ledger

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
)

type ownerFlags map[string]bool

// stubOperation implements only what actAsOwner reads.
type stubOperation struct {
	cliapp.OperationContext
	flags ownerFlags
}

func (s stubOperation) BoolFlag(name string) bool { return s.flags[name] }

func withOwnerSeams(t *testing.T, eligible func(context.Context) error, exchange func(context.Context) (string, error)) {
	t.Helper()
	prevEligible, prevExchange := ownerEligible, ownerExchange
	ownerEligible, ownerExchange = eligible, exchange
	t.Cleanup(func() { ownerEligible, ownerExchange = prevEligible, prevExchange })
}

func neverCalled(t *testing.T) (func(context.Context) error, func(context.Context) (string, error)) {
	return func(context.Context) error { t.Fatal("owner eligibility must not be checked"); return nil },
		func(context.Context) (string, error) { t.Fatal("owner session must not be requested"); return "", nil }
}

// Without --local-owner the CLI presents nothing: an owner session is never an
// automatic fallback.
func TestActAsOwnerIsNeverAutomatic(t *testing.T) {
	eligible, exchange := neverCalled(t)
	withOwnerSeams(t, eligible, exchange)
	request := connect.NewRequest(&struct{}{})
	require.NoError(t, actAsOwner(stubOperation{flags: ownerFlags{}}, request))
	require.Empty(t, request.Header().Get("Authorization"))
}

func TestActAsOwnerPresentsTheOwnerSessionWhenAsked(t *testing.T) {
	withOwnerSeams(t, func(context.Context) error { return nil }, func(context.Context) (string, error) { return "owner-session", nil })
	request := connect.NewRequest(&struct{}{})
	require.NoError(t, actAsOwner(stubOperation{flags: ownerFlags{localOwnerFlag: true}}, request))
	require.Equal(t, "Bearer owner-session", request.Header().Get("Authorization"))
}

// Inside an agent run the broker proves an identity, so the owner session is
// refused before any exchange is attempted.
func TestActAsOwnerIsRefusedInsideAnAgentRun(t *testing.T) {
	withOwnerSeams(t,
		func(context.Context) error { return errors.New("this process is an identified agent run") },
		func(context.Context) (string, error) { t.Fatal("no exchange inside an agent run"); return "", nil })
	request := connect.NewRequest(&struct{}{})
	err := actAsOwner(stubOperation{flags: ownerFlags{localOwnerFlag: true}}, request)
	require.ErrorContains(t, err, "outside an agent run")
	require.Empty(t, request.Header().Get("Authorization"))
}

func TestActAsOwnerReportsAnUnavailableAuthenticator(t *testing.T) {
	withOwnerSeams(t, func(context.Context) error { return nil }, func(context.Context) (string, error) { return "", errors.New("socket closed") })
	err := actAsOwner(stubOperation{flags: ownerFlags{localOwnerFlag: true}}, connect.NewRequest(&struct{}{}))
	require.ErrorContains(t, err, "local authenticator")
	require.ErrorContains(t, err, "socket closed")
}

// The refusal says what to do, and keeps the code for a caller that checks it.
func TestExplainRefusalSaysWhatToDo(t *testing.T) {
	unauthenticated := connect.NewError(connect.CodeUnauthenticated, errors.New("no identity was presented"))
	err := explainRefusal(unauthenticated, false)
	require.ErrorContains(t, err, "--local-owner")
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.ErrorContains(t, explainRefusal(unauthenticated, true), "could not verify the owner session")

	denied := connect.NewError(connect.CodePermissionDenied, errors.New("not the owner"))
	require.ErrorContains(t, explainRefusal(denied, false), "refused this identity")
	asOwner := explainRefusal(denied, true)
	require.ErrorContains(t, asOwner, "owner-binding policy")
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(asOwner))

	other := connect.NewError(connect.CodeNotFound, errors.New("x"))
	require.Equal(t, other, explainRefusal(other, true))
}
