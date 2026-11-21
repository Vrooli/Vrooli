package aimoneyfixtures

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// StaticKeyIssuer is an in-memory fixture issuer, not a signing/JWKS
// implementation. Opaque tokens are deterministic and carry no authority
// outside the tests that hold this instance.
type StaticKeyIssuer struct {
	mu       sync.Mutex
	entries  map[string]keyEntry
	sequence uint64
}
type keyEntry struct {
	subject string
	expires time.Time
	revoked bool
}

func (i *StaticKeyIssuer) Issue(subject string, expires time.Time) (string, error) {
	if subject == "" || expires.IsZero() {
		return "", fmt.Errorf("subject and expiry are required")
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.sequence++
	seed := fmt.Sprintf("fixture-key:%d:%s:%d", i.sequence, subject, expires.UnixNano())
	sum := sha256.Sum256([]byte(seed))
	token := "fixture." + hex.EncodeToString(sum[:])
	if i.entries == nil {
		i.entries = make(map[string]keyEntry)
	}
	i.entries[token] = keyEntry{subject: subject, expires: expires}
	return token, nil
}
func (i *StaticKeyIssuer) Revoke(token string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	entry, ok := i.entries[token]
	if ok {
		entry.revoked = true
		i.entries[token] = entry
	}
}
func (i *StaticKeyIssuer) Valid(token string, now time.Time) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	entry, ok := i.entries[token]
	return ok && !entry.revoked && now.Before(entry.expires)
}
