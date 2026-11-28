package usageattribution

import "sync"

// reporting serializes actions on one caller-supplied fingerprint. Waiters are
// registered before blocking, and idle entries are removed after the last
// holder or waiter leaves, so unknown fingerprints cannot grow process state.
var reporting fingerprintLockRegistry

type fingerprintLockRegistry struct {
	mu      sync.Mutex
	entries map[string]*fingerprintLock
}

type fingerprintLock struct {
	mu    sync.Mutex
	users int
}

func lockFingerprint(fingerprint string) func() {
	return reporting.lock(fingerprint)
}

func (registry *fingerprintLockRegistry) lock(key string) func() {
	registry.mu.Lock()
	if registry.entries == nil {
		registry.entries = make(map[string]*fingerprintLock)
	}
	entry := registry.entries[key]
	if entry == nil {
		entry = &fingerprintLock{}
		registry.entries[key] = entry
	}
	entry.users++
	registry.mu.Unlock()

	entry.mu.Lock()
	return func() {
		entry.mu.Unlock()
		registry.mu.Lock()
		entry.users--
		if entry.users == 0 {
			delete(registry.entries, key)
		}
		registry.mu.Unlock()
	}
}
