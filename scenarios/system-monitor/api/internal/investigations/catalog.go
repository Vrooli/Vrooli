package investigations

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// ExecutionMode is the only supported execution boundary for a catalog entry.
type ExecutionMode string

const (
	ModeNative ExecutionMode = "native"
	ModeShell  ExecutionMode = "shell"
)

// Entry is a portable investigation catalog entry. Built-ins are immutable;
// operator entries may replace a built-in with the same id.
type Entry struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Category      string        `json:"category"`
	Mode          ExecutionMode `json:"mode"`
	Query         string        `json:"query,omitempty"`
	ScriptFile    string        `json:"script_file,omitempty"`
	RequiredTools []string      `json:"required_tools,omitempty"`
	Platforms     []string      `json:"platforms"`
	Enabled       bool          `json:"enabled"`
	Source        string        `json:"source"`
}

type Catalog struct {
	entries map[string]Entry
	shell   map[string][]byte
}

//go:embed catalog/entries.json catalog/shell/*.sh
var builtinFS embed.FS

func decodeEntries(data []byte, source string) ([]Entry, error) {
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("decode %s catalog: %w", source, err)
	}
	for i := range entries {
		entries[i].ID = strings.TrimSpace(entries[i].ID)
		entries[i].Source = source
		if entries[i].ID == "" {
			return nil, fmt.Errorf("decode %s catalog: entry %d has no id", source, i)
		}
		if entries[i].Mode != ModeNative && entries[i].Mode != ModeShell {
			return nil, fmt.Errorf("decode %s catalog: entry %q has invalid mode %q", source, entries[i].ID, entries[i].Mode)
		}
		if len(entries[i].Platforms) == 0 {
			return nil, fmt.Errorf("decode %s catalog: entry %q has no platforms", source, entries[i].ID)
		}
	}
	return entries, nil
}

// Load loads the embedded product catalog and the optional operator overlay.
// A missing overlay is valid; a missing or empty built-in catalog is not.
func Load(stateDir string) (Catalog, error) {
	data, err := builtinFS.ReadFile("catalog/entries.json")
	if err != nil {
		return Catalog{}, fmt.Errorf("read built-in investigation catalog: %w", err)
	}
	builtins, err := decodeEntries(data, "builtin")
	if err != nil {
		return Catalog{}, err
	}
	if len(builtins) == 0 {
		return Catalog{}, errors.New("built-in investigation catalog is empty: rebuild the system-monitor binary with catalog entries")
	}

	catalog := Catalog{entries: make(map[string]Entry, len(builtins)), shell: make(map[string][]byte)}
	for _, entry := range builtins {
		catalog.entries[entry.ID] = entry
	}

	overlayDir := filepath.Join(stateDir, "investigations")
	entries, err := os.ReadDir(overlayDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return Catalog{}, fmt.Errorf("read operator investigation catalog %q: %w", overlayDir, err)
		}
	} else {
		for _, item := range entries {
			if item.IsDir() || filepath.Ext(item.Name()) != ".json" {
				continue
			}
			path := filepath.Join(overlayDir, item.Name())
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return Catalog{}, fmt.Errorf("read operator investigation entry %q: %w", path, readErr)
			}
			operatorEntries, decodeErr := decodeEntries(content, "operator")
			if decodeErr != nil {
				return Catalog{}, fmt.Errorf("operator catalog %q: %w", path, decodeErr)
			}
			for _, entry := range operatorEntries {
				catalog.entries[entry.ID] = entry
			}
		}
	}

	for _, entry := range catalog.entries {
		if entry.Mode != ModeShell {
			continue
		}
		name := entry.ScriptFile
		if name == "" {
			name = entry.ID + ".sh"
		}
		content, readErr := builtinFS.ReadFile(filepath.ToSlash(filepath.Join("catalog", "shell", name)))
		if readErr == nil {
			catalog.shell[entry.ID] = content
		}
	}
	return catalog, nil
}

// LoadBuiltin is a test seam that validates an alternate embedded filesystem.
func LoadBuiltin(fsys fs.FS) (Catalog, error) {
	data, err := fs.ReadFile(fsys, "catalog/entries.json")
	if err != nil {
		return Catalog{}, fmt.Errorf("read built-in investigation catalog: %w", err)
	}
	entries, err := decodeEntries(data, "builtin")
	if err != nil {
		return Catalog{}, err
	}
	if len(entries) == 0 {
		return Catalog{}, errors.New("built-in investigation catalog is empty")
	}
	catalog := Catalog{entries: make(map[string]Entry, len(entries)), shell: make(map[string][]byte)}
	for _, entry := range entries {
		catalog.entries[entry.ID] = entry
	}
	return catalog, nil
}

func (c Catalog) Entries() []Entry {
	result := make([]Entry, 0, len(c.entries))
	for _, entry := range c.entries {
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (c Catalog) Get(id string) (Entry, bool)    { entry, ok := c.entries[id]; return entry, ok }
func (c Catalog) Shell(id string) ([]byte, bool) { data, ok := c.shell[id]; return data, ok }

func (e Entry) SupportsCurrentPlatform() bool {
	want := runtime.GOOS
	if want == "darwin" {
		want = "macos"
	}
	for _, platform := range e.Platforms {
		if strings.EqualFold(platform, want) {
			return true
		}
	}
	return false
}
