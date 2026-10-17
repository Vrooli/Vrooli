// Package storeunlock holds this node's credential-store passphrase in memory.
//
// The control plane escrows the passphrase and re-delivers it, sealed to this
// node's encryption key, every time the agent connects — including after a
// reboot. The agent keeps it here, never on disk, and serves it to local Vrooli
// processes through the credential unlock socket so a headless node's store
// opens with nobody present.
package storeunlock

import (
	"strings"
	"sync"

	"github.com/vrooli/platform-go/credentialunlock"
)

// Holder is safe for concurrent use.
type Holder struct {
	mu    sync.Mutex
	value []byte
}

// Owns reports whether a pushed grant carries this node's store passphrase.
func Owns(logicalID, field string) bool {
	return strings.HasPrefix(strings.TrimSpace(logicalID), credentialunlock.NodeStoreLogicalIDPrefix) && strings.TrimSpace(field) == "passphrase"
}

// Put replaces the held passphrase, zeroing the previous one.
func (h *Holder) Put(value []byte) {
	if h == nil || len(value) == 0 {
		return
	}
	copied := append([]byte(nil), value...)
	h.mu.Lock()
	defer h.mu.Unlock()
	zero(h.value)
	h.value = copied
}

// Get returns a copy of the held passphrase; the caller must not retain it.
func (h *Holder) Get() ([]byte, bool) {
	if h == nil {
		return nil, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.value) == 0 {
		return nil, false
	}
	return append([]byte(nil), h.value...), true
}

// Held reports whether a passphrase is held, without copying it.
func (h *Holder) Held() bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.value) > 0
}

// Clear zeroes and forgets the held passphrase.
func (h *Holder) Clear() {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	zero(h.value)
	h.value = nil
}

func zero(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
