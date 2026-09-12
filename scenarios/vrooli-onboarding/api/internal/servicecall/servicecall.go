// Package servicecall contains small typed helpers for transport-free service
// seams. It keeps missing dependency behavior consistent without duplicating
// the same nil-guard in every domain method.
package servicecall

import "context"

// Invoke calls an injected service function when it is available. A missing
// function is a composition error and retains the historical context.Canceled
// sentinel used by onboarding service seams.
func Invoke[T any](available bool, call func() (T, error), zero T) (T, error) {
	if !available {
		return zero, context.Canceled
	}
	return call()
}
