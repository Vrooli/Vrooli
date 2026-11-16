package operatordecision

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// SubjectKeyFrom turns a producer's own identity for a situation (a work item,
// a goal home plus a decision id) into a subject key the Hub accepts. A name the
// rule already accepts is used as it is, so the key stays readable and is the
// same on every call. Any other name has its unacceptable characters replaced
// and its length cut, then carries a hash of the whole name, so two different
// names never share a subject.
func SubjectKeyFrom(name string) string {
	clean := strings.Map(func(r rune) rune {
		if r < 0x80 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._:/@-", r)) {
			return r
		}
		return '_'
	}, name)
	if clean == name && len(name) <= 128 && SubjectKeyRule.MatchString(name) {
		return name
	}
	sum := sha256.Sum256([]byte(name))
	if len(clean) > 100 {
		clean = clean[:100]
	}
	readable := strings.TrimLeft(clean, "._:/@-")
	if readable == "" {
		// A name made only of punctuation still has to start with a letter or digit.
		readable = "key"
	}
	return readable + "-" + hex.EncodeToString(sum[:8])
}
