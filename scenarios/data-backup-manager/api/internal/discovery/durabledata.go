package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DurableData is the discovery domain's local mirror of a resource manifest's
// `durable_data` block (shared shape: .vrooli/schemas/common.schema.json
// #/definitions/durableDataEntry, enforced platform-side by
// internal/resources/manifest). It is decoded directly from the manifest path
// the ResourceEnumerator reports — DBM deliberately keeps its own mirror rather
// than importing the vrooli-core type, to respect the scenario boundary.
type DurableData struct {
	Base    string                      `json:"base,omitempty"`
	Entries map[string]DurableDataEntry `json:"entries"`
}

// DurableDataEntry is one declared durable location under a resource's base.
type DurableDataEntry struct {
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	Regenerable bool   `json:"regenerable"`
	Format      string `json:"format,omitempty"`
	Sensitive   bool   `json:"sensitive,omitempty"`
	Rationale   string `json:"rationale,omitempty"`
}

// baseTokens are the home-relative tokens a durable_data base may start with.
var baseTokens = []string{"$HOME", "~", "%USERPROFILE%"}

// loadDurableData reads a resource manifest and returns its durable_data block,
// or (nil, nil) when the manifest is unreadable, unparseable, or declares none.
// It never errors: a malformed manifest simply contributes no candidates, so
// one bad resource never breaks discovery.
func loadDurableData(manifestPath string) *DurableData {
	if strings.TrimSpace(manifestPath) == "" {
		return nil
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil
	}
	var doc struct {
		DurableData *DurableData `json:"durable_data"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil
	}
	if doc.DurableData == nil || len(doc.DurableData.Entries) == 0 {
		return nil
	}
	return doc.DurableData
}

// resolveBase expands a durable_data base against the operator's home dir. A
// leading $HOME / ~ / %USERPROFILE% token (or an empty base) resolves to home;
// the remainder joins beneath it. A base with no recognized token is rejected
// (ok=false) so a misdeclaration can never point the scanner at an arbitrary
// absolute path.
func resolveBase(base, home string) (string, bool) {
	base = strings.TrimSpace(base)
	if base == "" {
		return home, true
	}
	if strings.Contains(base, "\\") {
		return "", false
	}
	for _, tok := range baseTokens {
		if base == tok {
			return home, true
		}
		if strings.HasPrefix(base, tok+"/") {
			rel := strings.TrimPrefix(base, tok+"/")
			if hasParentTraversal(rel) {
				return "", false
			}
			return filepath.Join(home, filepath.FromSlash(rel)), true
		}
	}
	return "", false
}

// hasParentTraversal reports whether a slash path contains a ".." segment.
func hasParentTraversal(p string) bool {
	for _, part := range strings.Split(p, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}
