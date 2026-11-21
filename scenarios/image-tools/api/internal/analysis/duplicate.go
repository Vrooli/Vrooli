package analysis

import (
	"fmt"

	internalops "image-tools/internal/ops"
	"image-tools/internal/percepthash"
)

// DuplicateDetect computes perceptual fingerprints of an image so callers can
// find near-duplicates by comparing hashes (Hamming distance) across a set:
//   - pHash: a 64-bit DCT perceptual hash (robust to re-encode / scale / minor
//     edits) — the primary dedup key.
//   - aHash: a 64-bit average hash (cheaper, more exposure-sensitive).
//
// Pure-Go (no model, no GPU) — the always-runnable analysis floor. The hash
// primitives live in internal/percepthash, shared with the diff comparison
// surface so both agree on what "similar" means.
func DuplicateDetect(src []byte) (DuplicateResult, error) {
	img, _, err := internalops.Decode(src)
	if err != nil {
		return DuplicateResult{}, fmt.Errorf("analysis: decode: %w", err)
	}
	return DuplicateResult{
		PhashHex: fmt.Sprintf("%016x", percepthash.PHash(img)),
		AhashHex: fmt.Sprintf("%016x", percepthash.AHash(img)),
		HashBits: 64,
	}, nil
}
