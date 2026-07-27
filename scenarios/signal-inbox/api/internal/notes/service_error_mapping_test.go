package notes

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
)

func TestToConnectError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want connect.Code
	}{
		{name: "invalid", err: ErrInvalidNote{Field: "title", Reason: "required"}, want: connect.CodeInvalidArgument},
		{name: "not_found", err: ErrNoteNotFound{ID: "missing"}, want: connect.CodeNotFound},
		{name: "internal", err: errors.New("boom"), want: connect.CodeInternal},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := connect.CodeOf(ToConnectError(tc.err)); got != tc.want {
				t.Fatalf("CodeOf(ToConnectError(%T)) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
