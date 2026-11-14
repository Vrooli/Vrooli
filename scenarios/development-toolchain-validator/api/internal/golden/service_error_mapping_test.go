package golden_test

import (
	"errors"
	"testing"

	"development-toolchain-validator/internal/golden"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
)

func TestToConnectError_MapsSentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want connect.Code
	}{
		{"nil", nil, 0},
		{"invalid", golden.ErrInvalidGolden{Field: "slug", Reason: "required"}, connect.CodeInvalidArgument},
		{"not found", golden.ErrGoldenNotFound{Slug: "g"}, connect.CodeNotFound},
		{"already exists", golden.ErrGoldenAlreadyExists{Slug: "g"}, connect.CodeAlreadyExists},
		{"regenerate failed", golden.ErrRegenerateFailed{Slug: "g", Wrapped: errors.New("boom")}, connect.CodeInternal},
		{"unknown", errors.New("opaque"), connect.CodeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := golden.ToConnectError(tc.err)
			if tc.err == nil {
				require.NoError(t, out)
				return
			}
			require.Error(t, out)
			require.Equal(t, tc.want, connect.CodeOf(out))
		})
	}
}
