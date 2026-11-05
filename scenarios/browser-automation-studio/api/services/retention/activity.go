package retention

import (
	"fmt"
	"sync"
)

// Evidence readers and producers share this process-local guard with owner
// cleanup. Live execution state remains authoritative across API restarts.
var evidenceActivity = struct {
	sync.Mutex
	users map[string]int
}{users: make(map[string]int)}

func BeginEvidenceActivity(key string) func() {
	evidenceActivity.Lock()
	evidenceActivity.users[key]++
	evidenceActivity.Unlock()
	return func() {
		evidenceActivity.Lock()
		defer evidenceActivity.Unlock()
		evidenceActivity.users[key]--
		if evidenceActivity.users[key] == 0 {
			delete(evidenceActivity.users, key)
		}
	}
}

func EvidenceActive(key string) bool {
	evidenceActivity.Lock()
	defer evidenceActivity.Unlock()
	return evidenceActivity.users[key] > 0
}

// WithInactiveEvidence closes the race between checking activity and removal.
func WithInactiveEvidence(keys []string, remove func() error) error {
	evidenceActivity.Lock()
	defer evidenceActivity.Unlock()
	for _, key := range keys {
		if evidenceActivity.users[key] > 0 {
			return fmt.Errorf("evidence is in use: %s", key)
		}
	}
	return remove()
}
