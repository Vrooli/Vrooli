// Package cleanupplan contains the stable hash primitive shared by the local
// uninstall planner and Bridge's durable cleanup domain.
package cleanupplan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// HashResolvedArtifacts hashes only the resolved Remove, Keep, and Cannot
// attribute lists. Request metadata, plan ids, timestamps, and transport are
// intentionally excluded: the authorization subject is the artifact set the
// operator reviewed, not the request that happened to produce it.
func HashResolvedArtifacts(remove, keep, cannotAttribute any) string {
	payload := struct {
		Remove          any `json:"remove"`
		Keep            any `json:"keep"`
		CannotAttribute any `json:"cannot_attribute"`
	}{Remove: remove, Keep: keep, CannotAttribute: cannotAttribute}
	data, _ := json.Marshal(payload)
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
