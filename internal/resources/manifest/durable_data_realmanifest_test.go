package manifest_test

import (
	"path/filepath"
	"testing"

	manifest "github.com/vrooli/vrooli/internal/resources/manifest"
)

// TestCodingAgentManifestsDeclareDurableData is the anti-drift guard for the
// per-resource durable-data feature: it loads the real coding-agent manifests
// through the production validator and asserts each declares durable_data with
// the expected curated entries and that credential entries are flagged
// sensitive. Removing or breaking a durable_data block fails here.
func TestCodingAgentManifestsDeclareDurableData(t *testing.T) {
	repoRoot := findRepoRoot(t)
	cases := []struct {
		name        string
		base        string
		wantKeys    []string
		sensitive   []string
		sqliteEntry string // entry key expected to declare format: sqlite (optional)
	}{
		{
			name:      "claude-code",
			base:      "$HOME/.claude",
			wantKeys:  []string{"history", "projects", "file-history", "plans", "credentials"},
			sensitive: []string{"credentials"},
		},
		{
			name:        "codex",
			base:        "$HOME/.codex",
			wantKeys:    []string{"sessions", "history", "state", "config", "memories", "auth"},
			sensitive:   []string{"auth"},
			sqliteEntry: "state",
		},
		{
			name:     "opencode",
			base:     "$HOME",
			wantKeys: []string{"config", "storage"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(repoRoot, "resources", tc.name, "resource.json")
			m, err := manifest.Load(path)
			if err != nil {
				t.Fatalf("load %s: %v", path, err)
			}
			if m.DurableData == nil {
				t.Fatalf("%s: expected durable_data block", tc.name)
			}
			if m.DurableData.Base != tc.base {
				t.Errorf("%s: base = %q, want %q", tc.name, m.DurableData.Base, tc.base)
			}
			for _, key := range tc.wantKeys {
				entry, ok := m.DurableData.Entries[key]
				if !ok {
					t.Errorf("%s: missing durable_data entry %q", tc.name, key)
					continue
				}
				if entry.Regenerable {
					t.Errorf("%s: entry %q must be non-regenerable to be backed up", tc.name, key)
				}
			}
			for _, key := range tc.sensitive {
				if entry, ok := m.DurableData.Entries[key]; !ok || !entry.Sensitive {
					t.Errorf("%s: entry %q must be present and sensitive", tc.name, key)
				}
			}
			if tc.sqliteEntry != "" {
				if entry, ok := m.DurableData.Entries[tc.sqliteEntry]; !ok || entry.Format != "sqlite" {
					t.Errorf("%s: entry %q must declare format sqlite", tc.name, tc.sqliteEntry)
				}
			}
		})
	}
}
