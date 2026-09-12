package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
)

func StableFileID(path string) string {
	return stableID("file", canonicalPath(path))
}

func StableSymbolID(language, qualifiedName, kind string) string {
	return stableID("symbol", strings.ToLower(strings.TrimSpace(language)), strings.TrimSpace(kind), strings.TrimSpace(qualifiedName))
}

func StableContractID(kind, fullName string) string {
	return stableID("contract", strings.TrimSpace(kind), strings.TrimSpace(fullName))
}

// StableContentAnchorID derives identity from semantic ownership and a
// whitespace-normalized declaration signature. Mutable line ranges are not
// identity inputs, so unrelated insertions do not change the anchor.
func StableContentAnchorID(language, owner, signature string) string {
	return stableID("anchor", strings.ToLower(strings.TrimSpace(language)), strings.TrimSpace(owner), strings.Join(strings.Fields(signature), " "))
}

func stableID(parts ...string) string {
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return parts[0] + ":" + hex.EncodeToString(digest[:16])
}

func canonicalPath(path string) string {
	clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	return strings.TrimPrefix(clean, "./")
}
