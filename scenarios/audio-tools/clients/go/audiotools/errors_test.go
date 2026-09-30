package audiotools

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
)

func TestNormalizeError_NilStaysNil(t *testing.T) {
	require.NoError(t, NormalizeError(nil))
}

func TestNormalizeError_MapsConnectCodes(t *testing.T) {
	cases := []struct {
		code     connect.Code
		sentinel error
	}{
		{connect.CodeUnavailable, ErrUnavailable},
		{connect.CodeResourceExhausted, ErrInsufficientCredits},
		{connect.CodeInvalidArgument, ErrInvalidArgument},
		{connect.CodeNotFound, ErrNotFound},
		{connect.CodeFailedPrecondition, ErrFailedPrecondition},
		{connect.CodeInternal, ErrInternal},
		{connect.CodeUnknown, ErrInternal},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.code.String(), func(t *testing.T) {
			cerr := connect.NewError(tc.code, errors.New("boom"))
			got := NormalizeError(cerr)
			require.True(t, errors.Is(got, tc.sentinel), "want %v, got %v", tc.sentinel, got)
		})
	}
}

func TestNormalizeError_NonConnectErrorWrapsInternal(t *testing.T) {
	raw := errors.New("io: closed")
	got := NormalizeError(raw)
	require.True(t, errors.Is(got, ErrInternal))
	require.True(t, errors.Is(got, raw))
}
