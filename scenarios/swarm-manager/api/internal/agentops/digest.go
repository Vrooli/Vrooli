package agentops

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
)

// digestPattern is the canonical shape every pinned digest in this layer takes:
// a sha256 hex content hash. Provenance, bindings, and workflow records all pin
// digests in this form so a mismatch at resume time is a typed, detectable
// tamper/staleness signal rather than a silent divergence.
var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// IsWellFormedDigest reports whether s is a canonical sha256:<64-hex> digest.
func IsWellFormedDigest(s string) bool { return digestPattern.MatchString(s) }

// CanonicalDigest returns the canonical sha256 content digest of a raw JSON
// document. The bytes are first decoded and re-encoded so that key ordering and
// insignificant whitespace never change the digest: two byte-different but
// semantically identical documents hash identically. This is the single hashing
// primitive the agent-operations layer pins (compiled-mode, prompt-catalog,
// caller-input, provenance, policy revisions) so reproducibility checks compare
// like with like.
func CanonicalDigest(raw []byte) (string, error) {
	var doc any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		return "", fmt.Errorf("canonical digest: decode: %w", err)
	}
	canonical, err := json.Marshal(doc) // Go marshals map keys in sorted order.
	if err != nil {
		return "", fmt.Errorf("canonical digest: re-encode: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return fmt.Sprintf("sha256:%x", sum), nil
}

// DigestOf marshals v and returns its canonical digest. It is the convenience
// form used when hashing an in-memory contract value rather than raw bytes.
func DigestOf(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("digest of value: marshal: %w", err)
	}
	return CanonicalDigest(raw)
}
