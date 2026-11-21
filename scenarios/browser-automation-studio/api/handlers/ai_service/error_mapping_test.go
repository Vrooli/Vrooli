package ai_service

import (
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
)

func TestMapHandlerErrorPreservesGatewaySpendRefusal(t *testing.T) {
	gatewayError := connect.NewError(connect.CodeResourceExhausted, errors.New("gateway spend rejected"))
	gatewayError.Meta().Set("X-Vrooli-Root-Operation-ID", "aabbccddeeff00112233445566778899")
	err := mapHandlerError(gatewayError)
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
	connectError, ok := err.(*connect.Error)
	require.True(t, ok)
	require.Equal(t, "aabbccddeeff00112233445566778899", connectError.Meta().Get("X-Vrooli-Root-Operation-ID"))
}

func TestRequestRootOperationIDPrefersGatewayRecoveryHeader(t *testing.T) {
	header := http.Header{}
	header.Set("Idempotency-Key", " idempotency-id ")
	header.Set("X-Vrooli-Root-Operation-ID", " gateway-recovery-id ")
	require.Equal(t, "gateway-recovery-id", requestRootOperationID(header))
	header.Del("X-Vrooli-Root-Operation-ID")
	require.Equal(t, "idempotency-id", requestRootOperationID(header))
}
