//go:build !linux

package platform

import "context"

func captureServiceScope(context.Context, ScopeRef, int, string) (ServiceScopeRef, error) {
	return ServiceScopeRef{}, ErrUnsupported
}

func stopServiceScope(context.Context, ServiceScopeRef) (ServiceScopeStopReceipt, error) {
	return ServiceScopeStopReceipt{}, ErrUnsupported
}

func inspectServiceScope(context.Context, ServiceScopeRef) (ServiceScopeObservation, error) {
	return ServiceScopeObservation{}, ErrUnsupported
}
