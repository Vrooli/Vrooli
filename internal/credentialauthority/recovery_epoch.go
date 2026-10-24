package credentialauthority

import "sync"

// RecoveryEpoch is the in-process part of the recovery activation boundary.
// Durable owners persist the returned value beside their authority metadata;
// capabilities carry the value they were issued under and must be rejected
// when it changes.
//
// Epoch zero is never valid. A fresh authority starts at one so that an
// omitted or zero-valued capability cannot accidentally pass validation.
type RecoveryEpoch struct {
	mu      sync.RWMutex
	current uint64
}

// NewRecoveryEpoch creates an epoch counter. Values below one are normalized
// to the first usable epoch for compatibility with fresh stores.
func NewRecoveryEpoch(initial uint64) *RecoveryEpoch {
	if initial == 0 {
		initial = 1
	}
	return &RecoveryEpoch{current: initial}
}

// Current returns the active recovery epoch.
func (e *RecoveryEpoch) Current() uint64 {
	if e == nil {
		return 0
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.current
}

// Advance invalidates capabilities issued under the previous epoch and
// returns the new epoch. Overflow is treated as a hard stop rather than
// wrapping back to a value that an old capability could carry.
func (e *RecoveryEpoch) Advance() uint64 {
	if e == nil {
		return 0
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.current == ^uint64(0) {
		return 0
	}
	e.current++
	return e.current
}

// Accept reports whether a capability was issued for the active epoch.
func (e *RecoveryEpoch) Accept(capabilityEpoch uint64) bool {
	return capabilityEpoch != 0 && capabilityEpoch == e.Current()
}
