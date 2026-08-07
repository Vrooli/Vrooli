package harness

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/api-core/database"

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

func normalizedAbsolutePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(abs)
}

func defaultAdapters(claudeRoot, home string) map[string]AdapterDescriptor {
	return map[string]AdapterDescriptor{
		"claude-code":           {HarnessID: "claude-code", Locations: []string{claudeRoot}, Format: MarkdownPerFile, Extract: wholeMarkdown, Provenance: Provenance{SourceRuntime: "claude-code"}},
		"gemini":                {HarnessID: "gemini", Locations: []string{filepath.Join(home, ".gemini", "GEMINI.md")}, Format: MarkdownSection, Extract: markdownSection("Gemini Added Memories"), Provenance: Provenance{SourceRuntime: "gemini"}},
		"codex":                 {HarnessID: "codex", Locations: []string{filepath.Join(home, ".codex", "AGENTS.md")}, Format: MarkdownBlob, Extract: wholeMarkdown, Provenance: Provenance{SourceRuntime: "codex"}},
		"opencode":              {HarnessID: "opencode", Locations: []string{filepath.Join(home, ".config", "opencode", "AGENTS.md")}, Format: MarkdownBlob, Extract: wholeMarkdown, Provenance: Provenance{SourceRuntime: "opencode"}},
		"grok":                  {HarnessID: "grok", Locations: []string{filepath.Join(home, ".grok", "memory")}, Format: MarkdownPerFile, Extract: wholeMarkdown, Provenance: Provenance{SourceRuntime: "grok"}},
		"antigravity":           {HarnessID: "antigravity", Locations: []string{filepath.Join(home, ".gemini", "antigravity", "brain")}, Format: MarkdownPerFile, Extract: wholeMarkdown, Provenance: Provenance{SourceRuntime: "antigravity"}},
		"swarm-manager-records": {HarnessID: "swarm-manager-records", Locations: []string{filepath.Join(home, ".vrooli", "data", "vrooli", "swarm-manager", "records")}, Format: JSONL, Extract: swarmRecord, Provenance: Provenance{SourceRuntime: "swarm-manager"}},
	}
}

// Runtimes returns every declared import adapter in stable order. Missing
// stores are still returned so maintenance can report an honest per-runtime
// failure instead of silently omitting an unconfigured harness.
func (i *Importer) Runtimes() []string {
	out := make([]string, 0, len(i.adapters))
	for runtime := range i.adapters {
		out = append(out, runtime)
	}
	sort.Strings(out)
	return out
}

// discover walks the adapter's declared locations. managedOnly reports that the
// store exists and every source in it was either a projection target or held
// nothing once the managed block was removed. That is a healthy zero, not a
// parse failure: a runtime whose whole store is the projection would otherwise
// report an import failure on every maintenance tick forever.
func (d AdapterDescriptor) discover(projectionTargets map[string]struct{}) (items []sourceItem, managedOnly bool, err error) {
	var found bool
	var sawSource, sawManaged bool
	for _, location := range d.Locations {
		info, err := os.Stat(location)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, false, fmt.Errorf("read harness store %q: %w", location, err)
		}
		found = true
		if info.IsDir() {
			err = filepath.WalkDir(location, func(path string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() || (!matchesFormat(path, d.Format) && d.HarnessID != "swarm-manager-records") {
					return nil
				}
				sawSource = true
				if _, excluded := projectionTargets[normalizedAbsolutePath(path)]; excluded {
					sawManaged = true
					return nil
				}
				loaded, managed, walkErr2 := d.extractPath(path)
				if walkErr2 != nil {
					return walkErr2
				}
				if managed {
					sawManaged = true
				}
				items = append(items, loaded...)
				return nil
			})
			if err != nil {
				return nil, false, fmt.Errorf("walk harness store %q: %w", location, err)
			}
			continue
		}
		sawSource = true
		if _, excluded := projectionTargets[normalizedAbsolutePath(location)]; excluded {
			sawManaged = true
			continue
		}
		loaded, managed, err := d.extractPath(location)
		if err != nil {
			return nil, false, err
		}
		if managed {
			sawManaged = true
		}
		items = append(items, loaded...)
	}
	if !found {
		return nil, false, fmt.Errorf("harness %q store is not present", d.HarnessID)
	}
	if len(items) == 0 {
		if sawSource && sawManaged {
			return nil, true, nil
		}
		return nil, false, fmt.Errorf("non-empty harness %q store yielded zero importable items", d.HarnessID)
	}
	sort.Slice(items, func(a, b int) bool { return items[a].Path < items[b].Path })
	return items, false, nil
}

// extractPath reads one source. managed reports that the source held content
// before the managed block was removed and nothing after it, which is a healthy
// zero rather than a parse failure.
func (d AdapterDescriptor) extractPath(path string) (items []sourceItem, managed bool, err error) {
	if d.Format == SQLite {
		out, err := extractSQLite(path)
		return out, false, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false, fmt.Errorf("read %s: %w", path, err)
	}
	// Older whole-file projections predate managed markers. Keep their
	// generated-only guard for relocated stores; current projections are
	// removed by stripManagedWakeBlock below.
	if strings.HasPrefix(string(b), generatedHeader) {
		return nil, true, nil
	}
	extracted, err := d.Extract(path, b)
	if err != nil {
		return nil, false, fmt.Errorf("extract %s: %w", path, err)
	}
	for _, item := range extracted {
		item.Body = stripManagedWakeBlock(item.Body)
		if strings.TrimSpace(item.Body) == "" {
			// The source existed and carried only this service's own output.
			managed = true
			continue
		}
		items = append(items, item)
	}
	return items, managed, nil
}

// stripManagedWakeBlock removes generated wake content while preserving all
// surrounding native memory. A file can contain several managed blocks after
// repeated migrations, so every complete marker pair is removed.
func stripManagedWakeBlock(body string) string {
	for {
		start := strings.Index(body, wakeStart)
		if start < 0 {
			return body
		}
		afterStart := body[start+len(wakeStart):]
		end := strings.Index(afterStart, wakeEnd)
		if end < 0 {
			return body
		}
		body = body[:start] + afterStart[end+len(wakeEnd):]
	}
}

func matchesFormat(path string, format Format) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch format {
	case MarkdownPerFile, MarkdownSection, MarkdownBlob:
		return ext == ".md"
	case JSONL:
		return ext == ".jsonl" || ext == ".json"
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
	db, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: "file:" + path + "?mode=ro", MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		return nil, err
	}
	defer db.Close()
	var value string
	if err := db.Primary().QueryRowContext(context.Background(), `SELECT value FROM ItemTable WHERE key='aicontext.personalContext'`).Scan(&value); err != nil {
		return nil, err
	}
	return wholeMarkdown(path+"#aicontext.personalContext", []byte(value))
}

// swarmRecord is intentionally a read-only compatibility adapter. It imports
// the durable narrative, not swarm-manager's relational linkage fields.
func swarmRecord(path string, data []byte) ([]sourceItem, error) {
	var r struct {
		ID, Trigger, Approach, Evidence, Outcome string
		Scenario                                 string `json:"scenario"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	// The directory also contains private drafts and adjacent runtime metadata.
	// They are not published historical records, so ignore them.
	if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.Trigger) == "" || strings.TrimSpace(r.Approach) == "" || strings.TrimSpace(r.Outcome) == "" {
		return nil, nil
	}
	body := fmt.Sprintf("Work record from swarm-manager (%s)\n\nTrigger: %s\nApproach: %s\nEvidence: %s\nOutcome: %s", r.Scenario, r.Trigger, r.Approach, r.Evidence, r.Outcome)
	return []sourceItem{{Path: path, Body: body}}, nil
}
