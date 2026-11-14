//go:build !windows

package platform

import "context"

func PrepareWindowsOwnedServiceSubmission(context.Context, OperationIntent) error {
	return ErrUnsupported
}
