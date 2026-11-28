package binaryfetch

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
)

// VerifyArtifact verifies the bytes that will be launched. File artifacts use
// a byte digest; directory artifacts use the deterministic tree digest.
func VerifyArtifact(path, layout, expected string) error {
	return VerifyArtifactContext(context.Background(), path, layout, expected)
}

// VerifyArtifactContext verifies the same bytes with streamed cancellation.
func VerifyArtifactContext(ctx context.Context, path, layout, expected string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
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
		actual, err = TreeDigestContext(ctx, path)
		if err != nil {
			return err
		}
	} else {
		var err error
		actual, err = FileDigestContext(ctx, path)
		if err != nil {
			return err
		}
	}

	if actual != expected {
		return fmt.Errorf("artifact checksum mismatch for %s: got %s want %s", path, actual, expected)
	}
	return nil
}
