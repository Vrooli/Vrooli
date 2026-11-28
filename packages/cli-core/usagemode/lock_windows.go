//go:build windows

package usagemode

import "sync"

var processLock sync.Mutex

// lockPath serialises writers inside this process; Windows has no flock and
// the mode file is single-machine, single-operator state.
func lockPath(string) (func(), error) {
	processLock.Lock()
	return processLock.Unlock, nil
}
