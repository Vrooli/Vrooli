package workspaces

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// newID returns a prefixed 128-bit random identity. crypto/rand is the source
// because ids are addressable URLs and foreign keys: a predictable counter
// would leak creation order and invite collisions across imports, forks and a
// future offline draft queue.
func newID(prefix string) (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("read crypto/rand: %w", err)
	}
	return prefix + hex.EncodeToString(bytes[:]), nil
}
