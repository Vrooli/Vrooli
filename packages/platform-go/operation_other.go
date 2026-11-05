//go:build !linux

package platform

import "context"

func startOwnedOperation(context.Context, OperationIntent, OperationSpec) (*OwnedOperation, error) {
	return nil, ErrUnsupported
}

func lookupOwnedOperation(context.Context, OperationIntent) (OperationRef, error) {
	return OperationRef{}, ErrUnsupported
}

func inspectOwnedOperation(context.Context, OperationRef) (OperationObservation, error) {
	return OperationObservation{}, ErrUnsupported
}

func stopOwnedOperation(context.Context, OperationRef) (OperationStopReceipt, error) {
	return OperationStopReceipt{}, ErrUnsupported
}

func validateNativeOperationRef(OperationRef) error { return ErrUnsupported }

func releaseOwnedOperation(context.Context, OperationRef) error { return ErrUnsupported }
