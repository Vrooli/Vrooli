package coreactivation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// nativeOverrideItem is the canonical digest input recorded in
// Record.NativeOverridesDigest; its shape is part of the persistent contract.
type nativeOverrideItem struct {
	Path string `json:"path"`
	SHA  string `json:"sha256"`
}

// NativeOverridesDigest binds existing loaded drop-ins without copying their
// contents into receipts or overwriting operator policy during restoration.
// Unknown/missing files and unsupported path encodings remain conflicts.
func NativeOverridesDigest(ctx context.Context, paths string) (string, error) {
	if ctx == nil {
		return "", ErrConflict
	}
	words := strings.Fields(paths)
	if len(words) > 32 {
		return "", fmt.Errorf("native override inventory exceeds its bound")
	}
	items := make([]nativeOverrideItem, 0, len(words))
	seen := map[string]bool{}
	for _, word := range words {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		path := word
		if strings.ContainsAny(word, "\\\"") {
			decoded, err := strconv.Unquote("\"" + word + "\"")
			if err != nil {
				return "", fmt.Errorf("native override path encoding is unsupported")
			}
			path = decoded
		}
		if !filepath.IsAbs(path) || filepath.Clean(path) != path || !strings.HasSuffix(path, ".conf") || seen[path] {
			return "", ErrConflict
		}
		seen[path] = true
		data, err := readFrozen(path, definitionLimit)
		if err != nil {
			return "", fmt.Errorf("native override file is unavailable or unqualified")
		}
		sum := sha256.Sum256(data)
		items = append(items, nativeOverrideItem{path, hex.EncodeToString(sum[:])})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	data, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
