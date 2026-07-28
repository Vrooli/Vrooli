package harness

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	// Register the SQLite driver for the read-only Cursor-format adapter.
	_ "modernc.org/sqlite"
)

// Format is the storage shape an adapter reads. Readers produce normalized
// source items; journal persistence deliberately has no format-specific code.
type Format string

const (
	MarkdownPerFile Format = "markdown-per-file"
	MarkdownSection Format = "markdown-section"
	MarkdownBlob    Format = "markdown-blob"
	JSONL           Format = "jsonl"
	SQLite          Format = "sqlite"
)

type (
	AdapterDescriptor struct {
		HarnessID  string
		Locations  []string
		Format     Format
		Extract    Extractor
		Provenance Provenance
	}
	Provenance struct{ SourceRuntime string }
	Extractor  func(path string, data []byte) ([]sourceItem, error)
	sourceItem struct{ Path, Body string }
)

func defaultAdapters(claudeRoot, home string) map[string]AdapterDescriptor {
	return map[string]AdapterDescriptor{
		"claude-code": {HarnessID: "claude-code", Locations: []string{claudeRoot}, Format: MarkdownPerFile, Extract: wholeMarkdown, Provenance: Provenance{SourceRuntime: "claude-code"}},
		"gemini":      {HarnessID: "gemini", Locations: []string{filepath.Join(home, ".gemini", "GEMINI.md")}, Format: MarkdownSection, Extract: markdownSection("Gemini Added Memories"), Provenance: Provenance{SourceRuntime: "gemini"}},
		"codex":       {HarnessID: "codex", Locations: []string{filepath.Join(home, ".codex", "AGENTS.md")}, Format: MarkdownBlob, Extract: wholeMarkdown, Provenance: Provenance{SourceRuntime: "codex"}},
		"opencode":    {HarnessID: "opencode", Locations: []string{filepath.Join(home, ".config", "opencode", "AGENTS.md")}, Format: MarkdownBlob, Extract: wholeMarkdown, Provenance: Provenance{SourceRuntime: "opencode"}},
		"grok":        {HarnessID: "grok", Locations: []string{filepath.Join(home, ".grok", "memory")}, Format: MarkdownPerFile, Extract: wholeMarkdown, Provenance: Provenance{SourceRuntime: "grok"}},
		"antigravity": {HarnessID: "antigravity", Locations: []string{filepath.Join(home, ".gemini", "antigravity", "brain")}, Format: MarkdownPerFile, Extract: wholeMarkdown, Provenance: Provenance{SourceRuntime: "antigravity"}},
	}
}

func (d AdapterDescriptor) discover() ([]sourceItem, error) {
	var items []sourceItem
	var found bool
	for _, location := range d.Locations {
		info, err := os.Stat(location)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read harness store %q: %w", location, err)
		}
		found = true
		if info.IsDir() {
			err = filepath.WalkDir(location, func(path string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() || !matchesFormat(path, d.Format) {
					return nil
				}
				loaded, err := d.extractPath(path)
				if err != nil {
					return err
				}
				items = append(items, loaded...)
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("walk harness store %q: %w", location, err)
			}
			continue
		}
		loaded, err := d.extractPath(location)
		if err != nil {
			return nil, err
		}
		items = append(items, loaded...)
	}
	if !found {
		return nil, fmt.Errorf("harness %q store is not present", d.HarnessID)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("non-empty harness %q store yielded zero importable items", d.HarnessID)
	}
	sort.Slice(items, func(a, b int) bool { return items[a].Path < items[b].Path })
	return items, nil
}

func (d AdapterDescriptor) extractPath(path string) ([]sourceItem, error) {
	if d.Format == SQLite {
		return extractSQLite(path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	items, err := d.Extract(path, b)
	if err != nil {
		return nil, fmt.Errorf("extract %s: %w", path, err)
	}
	return items, nil
}

func matchesFormat(path string, format Format) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch format {
	case MarkdownPerFile, MarkdownSection, MarkdownBlob:
		return ext == ".md"
	case JSONL:
		return ext == ".jsonl"
	case SQLite:
		return ext == ".db" || ext == ".sqlite" || ext == ".sqlite3" || ext == ".vscdb"
	}
	return false
}

func wholeMarkdown(path string, data []byte) ([]sourceItem, error) {
	if body := strings.TrimSpace(string(data)); body != "" {
		return []sourceItem{{Path: path, Body: body}}, nil
	}
	return nil, nil
}

func markdownSection(name string) Extractor {
	return func(path string, data []byte) ([]sourceItem, error) {
		lines := strings.Split(string(data), "\n")
		start := -1
		for n, line := range lines {
			if strings.EqualFold(strings.TrimSpace(strings.TrimLeft(line, "# ")), name) {
				start = n + 1
				break
			}
		}
		if start < 0 {
			return nil, fmt.Errorf("section %q not found", name)
		}
		end := len(lines)
		for n := start; n < len(lines); n++ {
			if strings.HasPrefix(strings.TrimSpace(lines[n]), "#") {
				end = n
				break
			}
		}
		body := strings.TrimSpace(strings.Join(lines[start:end], "\n"))
		if body == "" {
			return nil, fmt.Errorf("section %q is empty", name)
		}
		return []sourceItem{{Path: path + "#" + name, Body: body}}, nil
	}
}

func jsonlItems(path string, data []byte) ([]sourceItem, error) {
	var items []sourceItem
	s := bufio.NewScanner(strings.NewReader(string(data)))
	line := 0
	for s.Scan() {
		line++
		raw := strings.TrimSpace(s.Text())
		if raw == "" {
			continue
		}
		var object map[string]any
		if err := json.Unmarshal([]byte(raw), &object); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		body := ""
		for _, key := range []string{"content", "text", "memory", "description"} {
			if value, ok := object[key].(string); ok {
				body = strings.TrimSpace(value)
				break
			}
		}
		if body == "" {
			return nil, fmt.Errorf("line %d has no supported text field", line)
		}
		items = append(items, sourceItem{Path: fmt.Sprintf("%s#L%d", path, line), Body: body})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func extractSQLite(path string) ([]sourceItem, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	var value string
	if err := db.QueryRow(`SELECT value FROM ItemTable WHERE key='aicontext.personalContext'`).Scan(&value); err != nil {
		return nil, err
	}
	return wholeMarkdown(path+"#aicontext.personalContext", []byte(value))
}
