package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"scenario-to-cloud/domain"
)

// Digest returns sha256 over the canonical JSON of the closure with its own
// Digest field cleared. encoding/json emits struct fields in declaration
// order and map keys sorted, and every slice in the closure is sorted by the
// resolver, so equal closures produce equal digests.
func Digest(closure domain.Closure) (string, error) {
	closure.Digest = ""
	data, err := json.Marshal(closure)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// Verify recomputes the digest and reports whether it matches the stored one.
func Verify(closure domain.Closure) (bool, error) {
	expected, err := Digest(closure)
	if err != nil {
		return false, err
	}
	return expected == closure.Digest, nil
}
