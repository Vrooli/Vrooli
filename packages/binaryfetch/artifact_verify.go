package binaryfetch

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// VerifyArtifact verifies the bytes that will be launched. File artifacts use
// a byte digest; directory artifacts use the deterministic tree digest.
func VerifyArtifact(path, layout, expected string) error {
	expected = strings.ToLower(strings.TrimSpace(expected))
	if len(expected) != 64 {
		return fmt.Errorf("artifact digest must be a 64-character SHA-256 value")
	}
	if _, err := hex.DecodeString(expected); err != nil {
		return fmt.Errorf("artifact digest is not hexadecimal: %w", err)
	}
	var actual string
	if strings.EqualFold(strings.TrimSpace(layout), "dir") {
		var err error
		actual, err = TreeDigest(path)
		if err != nil {
			return err
		}
	} else {
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(body)
		actual = hex.EncodeToString(sum[:])
	}
	if actual != expected {
		return fmt.Errorf("artifact checksum mismatch for %s: got %s want %s", path, actual, expected)
	}
	return nil
}
