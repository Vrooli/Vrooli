package authz

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
)

func TestStaticTokenAuthorizerAllowsLocalOperatorWhenNotEnforced(t *testing.T) {
	err := AllowLocalOperator().Authorize(context.Background(), OperationExposureExpose, nil)
	require.NoError(t, err)
}

func TestStaticTokenAuthorizerRequiresBearerWhenEnforced(t *testing.T) {
	err := StaticTokenAuthorizer{Enforced: true, Token: "secret"}.Authorize(context.Background(), OperationExposureExpose, nil)
	require.ErrorIs(t, err, ErrTokenRequired)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(ToConnectError(err)))
}

func TestStaticTokenAuthorizerRejectsWrongBearer(t *testing.T) {
	header := http.Header{}
	header.Set("Authorization", "Bearer wrong")

	err := StaticTokenAuthorizer{Enforced: true, Token: "secret"}.Authorize(context.Background(), OperationExposureExpose, header)
	require.ErrorIs(t, err, ErrTokenDenied)
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(ToConnectError(err)))
}

func TestStaticTokenAuthorizerAcceptsBearer(t *testing.T) {
	header := http.Header{}
	header.Set("Authorization", "Bearer secret")

	err := StaticTokenAuthorizer{Enforced: true, Token: "secret"}.Authorize(context.Background(), OperationExposureExpose, header)
	require.NoError(t, err)
}

func TestStaticTokenAuthorizerAcceptsOperatorHeader(t *testing.T) {
	header := http.Header{}
	header.Set(operatorTokenHeader, "secret")

	err := StaticTokenAuthorizer{Enforced: true, Token: "secret"}.Authorize(context.Background(), OperationRecoveryRecover, header)
	require.NoError(t, err)
}

func TestToConnectErrorFallsBackToInternal(t *testing.T) {
	err := ToConnectError(errors.New("boom"))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}
