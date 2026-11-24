package domain

import (
	"crypto/sha256"
	"encoding/hex"
)

// IsSHA256Hex reports whether value is a bare 64-character hexadecimal
// SHA-256 digest. Prefix normalization belongs to the caller.
func IsSHA256Hex(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
