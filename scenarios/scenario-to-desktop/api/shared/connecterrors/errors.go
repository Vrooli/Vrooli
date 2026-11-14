// Package connecterrors preserves the scenario remediation contract on
// Connect-RPC failures.
package connecterrors

import (
	"context"
	"encoding/json"
	stderrors "errors"

	"connectrpc.com/connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
	domainerrors "scenario-to-desktop-api/shared/errors"
)

// Interceptor attaches one ErrorEnvelope to every unary Connect failure. This
// keeps transport codes useful while giving UI and agent clients a single,
// typed remediation payload regardless of which domain emitted the error.
func Interceptor() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			response, err := next(ctx, req)
			if err == nil {
				return response, nil
			}
			return response, WithEnvelope(err)
		}
	})
}

// WithEnvelope preserves an existing Connect status and details, adding the
// scenario ErrorEnvelope exactly once. Non-Connect errors are made internal
// failures with a safe, actionable default recovery.
func WithEnvelope(err error) error {
	connectErr := new(connect.Error)
	if !stderrors.As(err, &connectErr) {
		connectErr = connect.NewError(connect.CodeInternal, err)
	}
	for _, detail := range connectErr.Details() {
		if detail.Type() == "vrooli.scenario_to_desktop.v1.shared.ErrorEnvelope" {
			return connectErr
		}
	}

	domainErr, ok := domainerrors.IsDomainError(err)
	if !ok {
		domainErr = domainerrors.ErrInternal("desktop operation failed").WithRecovery(domainerrors.RecoveryContactSupport, "Review the server logs and contact support if the problem persists")
	}
	domainErr = domainErr.EnrichRecovery()
	detail, detailErr := connect.NewErrorDetail(envelope(domainErr))
	if detailErr == nil {
		connectErr.AddDetail(detail)
	}
	return connectErr
}

func envelope(err *domainerrors.DomainError) *sharedv1.ErrorEnvelope {
	details := make(map[string]string, len(err.Details))
	for key, value := range err.Details {
		encoded, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			details[key] = "[unavailable]"
			continue
		}
		details[key] = string(encoded)
	}
	result := &sharedv1.ErrorEnvelope{
		Code:         string(err.Code),
		Category:     string(err.Category()),
		Recovery:     string(err.GetRecovery()),
		RecoveryHint: err.RecoveryHint,
		Details:      details,
		ManualSteps:  append([]string(nil), err.ManualSteps...),
	}
	if err.RetryStrategy != nil {
		result.RetryStrategy = marshal(err.RetryStrategy)
	}
	if err.AutoFix != nil {
		result.AutoFix = marshal(err.AutoFix)
	}
	if err.Diagnostic != nil {
		result.Diagnostic = marshal(err.Diagnostic)
	}
	return result
}

func marshal(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(encoded)
}
