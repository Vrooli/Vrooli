package agentmanager

import (
	"errors"
	"fmt"

	"connectrpc.com/connect"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

// ApprovalPendingError means Agent Manager requested operator approval and did
// not start the requested task or run. Callers must not treat it as accepted.
type ApprovalPendingError struct {
	Subject string
	Detail  string
	cause   error
}

func (e *ApprovalPendingError) Error() string {
	if e == nil {
		return "agent-manager authorization approval pending; operation was not started"
	}
	if e.Detail == "" {
		return "agent-manager authorization approval pending; operation was not started"
	}
	return fmt.Sprintf("agent-manager authorization approval pending; operation was not started: %s", e.Detail)
}

func (e *ApprovalPendingError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func approvalPending(err error) error {
	var rpcErr *connect.Error
	if !errors.As(err, &rpcErr) || rpcErr.Code() != connect.CodeFailedPrecondition {
		return err
	}
	for _, detail := range rpcErr.Details() {
		message, decodeErr := detail.Value()
		if decodeErr != nil {
			continue
		}
		pending, ok := message.(*apipb.AuthorizationApprovalPending)
		if ok {
			return &ApprovalPendingError{Subject: pending.GetSubject(), Detail: pending.GetDetail(), cause: err}
		}
	}
	return err
}
