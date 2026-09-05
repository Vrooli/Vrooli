package recoverylock

import (
	"errors"
	"fmt"
)

var ErrLockHeld = errors.New("recovery lock held")

// LockHeldError preserves the informational holder written by the process
// that owns the advisory lock while retaining errors.Is compatibility.
type LockHeldError struct {
	Holder string
	Cause  error
}

func (e *LockHeldError) Error() string {
	if e.Holder == "" {
		return fmt.Sprintf("%s: %v", ErrLockHeld, e.Cause)
	}
	return fmt.Sprintf("%s by %s: %v", ErrLockHeld, e.Holder, e.Cause)
}

func (e *LockHeldError) Unwrap() error { return ErrLockHeld }
