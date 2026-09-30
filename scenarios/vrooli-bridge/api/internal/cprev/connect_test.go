package cprev_test

import (
	"errors"
	"testing"

	"vrooli-bridge/internal/cprev"

	"connectrpc.com/connect"
)

func TestConnectError_Mapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want connect.Code
	}{
		{"unsafe ref → invalid argument", cprev.ErrUnsafeRevision{Revision: "HEAD~1", Reason: "bad"}, connect.CodeInvalidArgument},
		{"not pushed → failed precondition", cprev.ErrNotPushed{Commit: "abc", Remote: "origin"}, connect.CodeFailedPrecondition},
		{"no CP commit → failed precondition", cprev.ErrNoControlPlaneCommit{}, connect.CodeFailedPrecondition},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cprev.ConnectError(tc.err)
			if got == nil {
				t.Fatalf("ConnectError(%v) = nil, want a %v error", tc.err, tc.want)
			}
			if connect.CodeOf(got) != tc.want {
				t.Fatalf("code = %v, want %v", connect.CodeOf(got), tc.want)
			}
			// The original message must survive so the client sees the guidance.
			if got.Error() == "" {
				t.Fatalf("mapped error has no message")
			}
		})
	}
}

func TestConnectError_PassesThroughNonCprevErrors(t *testing.T) {
	if got := cprev.ConnectError(errors.New("some domain error")); got != nil {
		t.Fatalf("ConnectError(non-cprev) = %v, want nil (caller falls back to domain mapping)", got)
	}
	if got := cprev.ConnectError(nil); got != nil {
		t.Fatalf("ConnectError(nil) = %v, want nil", got)
	}
}
