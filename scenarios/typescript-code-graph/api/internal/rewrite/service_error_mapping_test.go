package rewrite

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"typescript-code-graph/internal/sidecar"
)

func TestErrorToConnectCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want connect.Code
	}{
		{"invalid_input", RewriteError{Kind: RewriteErrorInvalidInput}, connect.CodeInvalidArgument},
		{"invalid_operation", RewriteError{Kind: RewriteErrorInvalidOperation}, connect.CodeInvalidArgument},
		{"plan_not_found", RewriteError{Kind: RewriteErrorPlanNotFound}, connect.CodeFailedPrecondition},
		{"sidecar_unavailable", RewriteError{Kind: RewriteErrorSidecarUnavailable}, connect.CodeUnavailable},
		{"sidecar_timeout", RewriteError{Kind: RewriteErrorSidecarTimeout}, connect.CodeDeadlineExceeded},
		{"internal", RewriteError{Kind: RewriteErrorInternal}, connect.CodeInternal},
		{"ctx_deadline", context.DeadlineExceeded, connect.CodeDeadlineExceeded},
		{"ctx_canceled", context.Canceled, connect.CodeCanceled},
		{"random", errors.New("other"), connect.CodeInternal},
		{"nil", nil, connect.Code(0)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ErrorToConnectCode(tc.err); got != tc.want {
				t.Errorf("ErrorToConnectCode = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFromSidecarError_Classifies(t *testing.T) {
	cases := []struct {
		name string
		in   error
		kind RewriteErrorKind
	}{
		{"unavailable", sidecar.ErrSidecarUnavailable, RewriteErrorSidecarUnavailable},
		{"permanent", sidecar.ErrSidecarPermanentlyUnhealthy, RewriteErrorSidecarUnavailable},
		{"timeout", sidecar.ErrSidecarTimeout, RewriteErrorSidecarTimeout},
		{"deadline", context.DeadlineExceeded, RewriteErrorSidecarTimeout},
		{"typed_rewrite", &sidecar.RewriteError{Kind: "anything", Message: "boom"}, RewriteErrorInternal},
		{"unknown", errors.New("???"), RewriteErrorInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := fromSidecarError("/abs", "id", tc.in)
			var re RewriteError
			if !errors.As(err, &re) || re.Kind != tc.kind {
				t.Errorf("want kind %s, got %v", tc.kind, err)
			}
		})
	}
}

func TestToConnectError(t *testing.T) {
	if ToConnectError(nil) != nil {
		t.Error("nil in should produce nil out")
	}
	err := ToConnectError(RewriteError{Kind: RewriteErrorPlanNotFound})
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("expected CodeFailedPrecondition, got %v", connect.CodeOf(err))
	}
}
