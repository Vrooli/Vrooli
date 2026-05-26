// Package registry persists the mapping from a Vrooli destination name to the
// kopia repository it represents (backend + config-file path + backend params).
// It lives under the resource state root (never repo-local data/) and is what
// `repo list` reads and what snapshot/policy/maintenance use to resolve a
// repository by name. It stores NO secrets — passphrases and S3 keys live in
// vault.
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Backend enumerates the kopia backends this resource version launches.
const (
	BackendFilesystem = "filesystem"
	BackendS3         = "s3"
)

// Entry describes one registered repository. No secret material is stored here.
type Entry struct {
	Name       string `json:"name"`
	Backend    string `json:"backend"`
	ConfigFile string `json:"config_file"`
	CacheDir   string `json:"cache_dir,omitempty"`
	// Filesystem backend.
	Path string `json:"path,omitempty"`
	// S3 backend.
	Bucket     string `json:"bucket,omitempty"`
	Endpoint   string `json:"endpoint,omitempty"`
	Prefix     string `json:"prefix,omitempty"`
	Region     string `json:"region,omitempty"`
	DisableTLS bool   `json:"disable_tls,omitempty"`
	CreatedAt  string `json:"created_at"`
}

// Registry is a JSON-file-backed set of repository entries keyed by name.
type Registry struct {
	// Path is the registry file location (resolved under the state root).
	Path string
	// Now supplies timestamps; overridable in tests.
	Now func() time.Time
}

// New returns a Registry backed by the given file path.
func New(path string) *Registry {
	return &Registry{Path: path, Now: time.Now}
}

type fileShape struct {
	Version int     `json:"version"`
	Repos   []Entry `json:"repos"`
}

// Load returns all registered entries sorted by name. A missing file is an
// empty registry, not an error.
func (r *Registry) Load() ([]Entry, error) {
	data, err := os.ReadFile(r.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read registry %s: %w", r.Path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, nil
	}
	var shape fileShape
	if err := json.Unmarshal(data, &shape); err != nil {
		return nil, fmt.Errorf("parse registry %s: %w", r.Path, err)
	}
	sort.Slice(shape.Repos, func(i, j int) bool { return shape.Repos[i].Name < shape.Repos[j].Name })
	return shape.Repos, nil
}

// Get returns the entry for a name. found is false when the name is unregistered.
func (r *Registry) Get(name string) (Entry, bool, error) {
	entries, err := r.Load()
	if err != nil {
		return Entry{}, false, err
	}
	for _, e := range entries {
		if e.Name == name {
			return e, true, nil
		}
	}
	return Entry{}, false, nil
}

// Upsert inserts or replaces an entry by name, stamping CreatedAt on first add.
func (r *Registry) Upsert(e Entry) error {
	if strings.TrimSpace(e.Name) == "" {
		return fmt.Errorf("registry entry requires a name")
	}
	entries, err := r.Load()
	if err != nil {
		return err
	}
	now := time.Now
	if r.Now != nil {
		now = r.Now
	}
	replaced := false
	for i := range entries {
		if entries[i].Name == e.Name {
			if e.CreatedAt == "" {
				e.CreatedAt = entries[i].CreatedAt
			}
			entries[i] = e
			replaced = true
			break
		}
	}
	if !replaced {
		if e.CreatedAt == "" {
			e.CreatedAt = now().UTC().Format(time.RFC3339)
		}
		entries = append(entries, e)
	}
	return r.save(entries)
}

// Remove deletes an entry by name. Removing an absent name is a no-op.
func (r *Registry) Remove(name string) error {
	entries, err := r.Load()
	if err != nil {
		return err
	}
	out := entries[:0]
	for _, e := range entries {
		if e.Name != name {
			out = append(out, e)
		}
	}
	return r.save(out)
}

func (r *Registry) save(entries []Entry) error {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	if entries == nil {
		entries = []Entry{}
	}
	data, err := json.MarshalIndent(fileShape{Version: 1, Repos: entries}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode registry: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(r.Path), 0o700); err != nil {
		return fmt.Errorf("create registry dir: %w", err)
	}
	tmp := r.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write registry: %w", err)
	}
	if err := os.Rename(tmp, r.Path); err != nil {
		return fmt.Errorf("commit registry: %w", err)
	}
	return nil
}
