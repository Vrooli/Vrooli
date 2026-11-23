package control

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	internalcontrol "lighting-control/internal/control"
)

func TestDomainErrorPreservesCancellationStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want connect.Code
	}{
		{name: "caller canceled", err: context.Canceled, want: connect.CodeCanceled},
		{name: "operation deadline", err: context.DeadlineExceeded, want: connect.CodeDeadlineExceeded},
		{name: "invalid desired state", err: internalcontrol.ErrInvalidState, want: connect.CodeInvalidArgument},
		{name: "controller missing", err: internalcontrol.ErrControllerNotFound, want: connect.CodeNotFound},
		{name: "ambiguous zone", err: internalcontrol.ErrAmbiguousZone, want: connect.CodeFailedPrecondition},
		{name: "controller not enrolled", err: internalcontrol.ErrNotEnrolled, want: connect.CodeFailedPrecondition},
		{name: "unsupported operation", err: internalcontrol.ErrUnsupported, want: connect.CodeUnimplemented},
		{name: "missing row", err: sql.ErrNoRows, want: connect.CodeNotFound},
		{name: "backend unavailable", err: internalcontrol.ErrBackendUnavailable, want: connect.CodeUnavailable},
		{name: "protocol mismatch", err: internalcontrol.ErrProtocolMismatch, want: connect.CodeUnavailable},
		{name: "permission denied by backend", err: internalcontrol.ErrPermission, want: connect.CodeUnavailable},
		{name: "backend transport", err: internalcontrol.ErrTransport, want: connect.CodeUnavailable},
		{name: "unexpected failure", err: errors.New("unexpected failure"), want: connect.CodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := connect.CodeOf(domainError(fmt.Errorf("control operation: %w", tt.err)))
			if got != tt.want {
				t.Fatalf("domainError(%v) code = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
