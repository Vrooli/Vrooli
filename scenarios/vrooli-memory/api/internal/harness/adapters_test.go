package harness

import (
	"database/sql"
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
	_, err = d.discover()
	require.ErrorContains(t, err, "extract")
}

func TestGeneratedProjectionIsNeverAnImportItem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MEMORY.md")
	require.NoError(t, os.WriteFile(path, []byte(generatedHeader+"# Unified Vrooli Memory\n"), 0o600))
	d := AdapterDescriptor{HarnessID: "test", Locations: []string{path}, Format: MarkdownBlob, Extract: wholeMarkdown}
	_, err := d.discover()
	require.ErrorContains(t, err, "yielded zero importable items")
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
