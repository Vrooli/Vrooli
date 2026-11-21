package conversations

import (
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"notification-hub/internal/hub"
)

func TestConnectErrorTellsARevisedAskFromAResolvedOne(t *testing.T) {
	changed := connectError(fmt.Errorf("answer: %w", hub.ErrAskChanged))
	var connectErr *connect.Error
	require.True(t, errors.As(changed, &connectErr))
	require.Equal(t, connect.CodeFailedPrecondition, connectErr.Code())
	require.Equal(t, ReasonAskChanged, connectErr.Meta().Get(ErrorReasonHeader))
	require.Contains(t, connectErr.Message(), "reload it")

	resolved := connectError(fmt.Errorf("%w: ask is already answered", hub.ErrAskResolved))
	require.True(t, errors.As(resolved, &connectErr))
	require.Equal(t, connect.CodeFailedPrecondition, connectErr.Code())
	require.Empty(t, connectErr.Meta().Get(ErrorReasonHeader), "a resolved ask is final; only a revised one is reloaded")
}

func TestConnectErrorTellsAnUnansweredSenderFromAResolvedAsk(t *testing.T) {
	refused := connectError(fmt.Errorf("request context: %w", hub.ErrContextNotAccepted))
	var connectErr *connect.Error
	require.True(t, errors.As(refused, &connectErr))
	require.Equal(t, connect.CodeFailedPrecondition, connectErr.Code())
	require.Equal(t, ReasonContextNotAccepted, connectErr.Meta().Get(ErrorReasonHeader))
}

func TestConnectErrorAWithdrawnOrStaleRequestIsAFailedPreconditionNotAServerFault(t *testing.T) {
	for _, cause := range []error{hub.ErrAskWithdrawn, hub.ErrAskStale} {
		var connectErr *connect.Error
		require.True(t, errors.As(connectError(fmt.Errorf("ask: %w", cause)), &connectErr))
		require.Equal(t, connect.CodeFailedPrecondition, connectErr.Code(), cause.Error())
	}
}
