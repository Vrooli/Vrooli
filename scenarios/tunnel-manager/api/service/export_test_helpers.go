package service

import "tunnel-manager/domain"

// SetStateForTest allows integration tests to override internal state.
// This is intentionally not exported as a normal API — it exists only for testing.
func (re *RecoveryEngine) SetStateForTest(mutate func(s *domain.RecoveryState)) {
	re.mu.Lock()
	defer re.mu.Unlock()
	mutate(&re.state)
}
