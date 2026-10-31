package incidents

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func Fingerprint(parts ...string) string {
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part != "" {
			normalized = append(normalized, part)
		}
	}
	sum := sha256.Sum256([]byte(strings.Join(normalized, "|")))
	return "incfp_" + hex.EncodeToString(sum[:12])
}
