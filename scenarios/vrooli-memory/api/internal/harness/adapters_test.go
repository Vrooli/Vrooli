package harness

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestFormatReadersExtractNormalizedItems(t *testing.T) {
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
