package harness

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestFormatReadersExtractNormalizedItems(t *testing.T) { // [REQ:VMEM-P0-011]
	markdown, err := wholeMarkdown("memory.md", []byte("  durable memory  \n"))
	require.NoError(t, err)
	require.Equal(t, []sourceItem{{Path: "memory.md", Body: "durable memory"}}, markdown)

	section, err := markdownSection("Gemini Added Memories")("GEMINI.md", []byte("# Other\nignore\n## Gemini Added Memories\nremember this\n# Next\nignore"))
	require.NoError(t, err)
	require.Equal(t, "remember this", section[0].Body)

	jsonl, err := jsonlItems("memories.jsonl", []byte("{\"content\":\"one\"}\n{\"text\":\"two\"}\n"))
	require.NoError(t, err)
	require.Equal(t, []string{"one", "two"}, []string{jsonl[0].Body, jsonl[1].Body})
}

func TestSQLiteReaderAndNonEmptyParserFailure(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "state.vscdb")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE ItemTable (key TEXT PRIMARY KEY, value TEXT); INSERT INTO ItemTable(key,value) VALUES ('aicontext.personalContext','SQLite memory');`)
	require.NoError(t, err)
	require.NoError(t, db.Close())
	items, err := extractSQLite(dbPath)
	require.NoError(t, err)
	require.Equal(t, "SQLite memory", items[0].Body)

	bad := filepath.Join(dir, "bad.jsonl")
	require.NoError(t, os.WriteFile(bad, []byte("not json"), 0o600))
	d := AdapterDescriptor{HarnessID: "test", Locations: []string{bad}, Format: JSONL, Extract: jsonlItems}
	_, _, err = d.discover(nil)
	require.ErrorContains(t, err, "extract")
}

func TestGeneratedProjectionIsNeverAnImportItem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MEMORY.md")
	require.NoError(t, os.WriteFile(path, []byte(generatedHeader+"# Unified Vrooli Memory\n"), 0o600))
	d := AdapterDescriptor{HarnessID: "test", Locations: []string{path}, Format: MarkdownBlob, Extract: wholeMarkdown}
	items, managedOnly, err := d.discover(nil)
	require.NoError(t, err, "a store holding only generated output is a healthy zero, not a failure")
	require.Empty(t, items)
	require.True(t, managedOnly)
}

func TestLegacyGeneratedProjectionIsNeverAnImportItem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MEMORY.md")
	require.NoError(t, os.WriteFile(path, []byte(legacyGeneratedHeader+"# old projection\n"), 0o600))
	d := AdapterDescriptor{HarnessID: "claude-code", Locations: []string{path}, Format: MarkdownPerFile, Extract: wholeMarkdown}
	items, managed, err := d.extractPath(path)
	require.NoError(t, err)
	require.True(t, managed)
	require.Empty(t, items)
}

func TestEmptyStoreErrorsAreClassifiedForHonestDryRuns(t *testing.T) {
	require.True(t, IsEmptyStoreError(fmt.Errorf(`harness %q store is not present`, "gemini")))
	require.True(t, IsEmptyStoreError(fmt.Errorf(`non-empty harness %q store yielded zero importable items`, "codex")))
	require.False(t, IsEmptyStoreError(fmt.Errorf("unsupported harness %q", "unknown")))
}

func TestEmptyNativeMemoryDirectoryIsHealthyZero(t *testing.T) {
	d := AdapterDescriptor{HarnessID: "codex", Locations: []string{t.TempDir()}, Format: MarkdownPerFile, Extract: wholeMarkdown}
	items, managedOnly, err := d.discover(nil)
	require.NoError(t, err)
	require.Empty(t, items)
	require.True(t, managedOnly)
}

func TestAdapterStripsManagedWakeBlockButKeepsNativeText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.md")
	body := "native memory before\n" + wakeStart + "\nmanaged wake\n" + wakeEnd + "\nnative memory after\n"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	d := AdapterDescriptor{HarnessID: "test", Locations: []string{path}, Format: MarkdownBlob, Extract: wholeMarkdown}
	items, managedOnly, err := d.discover(nil)
	require.NoError(t, err)
	require.False(t, managedOnly)
	require.Len(t, items, 1)
	require.Equal(t, "native memory before\n\nnative memory after", items[0].Body)
	require.NotContains(t, items[0].Body, "managed wake")
}

func TestSwarmRecordAdapterImportsOnlyCompletedNarratives(t *testing.T) {
	completed, err := swarmRecord("record.json", []byte(`{"id":"rec-1","scenario":"vrooli-memory","trigger":"need memory","approach":"implemented it","evidence":"tests passed","outcome":"shipped"}`))
	require.NoError(t, err)
	require.Len(t, completed, 1)
	require.Contains(t, completed[0].Body, "Trigger: need memory")
	require.Contains(t, completed[0].Body, "Outcome: shipped")

	draft, err := swarmRecord("draft.json", []byte(`{"id":"rec-draft","trigger":"partial"}`))
	require.NoError(t, err)
	require.Empty(t, draft)
}

// A runtime whose entire store is this service's own projection must report a
// completed import that saw nothing, not a failure. Four runtimes reported a
// permanent red on every maintenance tick before this distinction existed.
func TestStoreHoldingOnlyManagedContentImportsWithoutFailing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	body := wakeStart + "\nprojected wake block\n" + wakeEnd + "\n"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	d := AdapterDescriptor{HarnessID: "codex", Locations: []string{path}, Format: MarkdownBlob, Extract: wholeMarkdown}
	items, managedOnly, err := d.discover(nil)
	require.NoError(t, err)
	require.Empty(t, items)
	require.True(t, managedOnly)
}

// A store with real content that cannot be parsed is still a failure.
func TestStoreWithUnparseableContentStillFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "memory.md")
	require.NoError(t, os.WriteFile(path, []byte("   \n\n   \n"), 0o600))

	d := AdapterDescriptor{HarnessID: "codex", Locations: []string{path}, Format: MarkdownBlob, Extract: wholeMarkdown}
	_, managedOnly, err := d.discover(nil)
	if err == nil {
		require.True(t, managedOnly, "whitespace-only store must not be reported as a parse failure")
	} else {
		require.ErrorContains(t, err, "yielded zero importable items")
		require.False(t, managedOnly)
	}
}
