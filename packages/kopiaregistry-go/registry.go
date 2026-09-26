// Package kopiaregistry owns the non-secret registry shared by resource-kopia
// and the control-plane credential inventory.
//
// The registry contains repository names and backend metadata only. Repository
// passphrases live in the credential authority under a per-repository identity.
package kopiaregistry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	BackendFilesystem = "filesystem"
	BackendS3         = "s3"
	PassphraseField   = "repository-passphrase"
)

// Entry describes one registered repository. It contains no credential value.
type Entry struct {
	Name       string `json:"name"`
	Backend    string `json:"backend"`
	ConfigFile string `json:"config_file"`
	CacheDir   string `json:"cache_dir,omitempty"`
	Path       string `json:"path,omitempty"`
	Bucket     string `json:"bucket,omitempty"`
	Endpoint   string `json:"endpoint,omitempty"`
	Prefix     string `json:"prefix,omitempty"`
	Region     string `json:"region,omitempty"`
	DisableTLS bool   `json:"disable_tls,omitempty"`
	CreatedAt  string `json:"created_at"`
}

// Registry is a JSON-file-backed set of repository entries keyed by name.
type Registry struct {
	Path string
	Now  func() time.Time
}

type fileShape struct {
	Version int     `json:"version"`
	Repos   []Entry `json:"repos"`
}

// New returns a registry backed by path.
func New(path string) *Registry { return &Registry{Path: path, Now: time.Now} }

// RegistryPath returns the declared kopia state location plus registry.json.
// KOPIA_STATE_DIR is the relocation lever for the resource's storage entry;
// the default mirrors resources/kopia/resource.json on each supported host.
func RegistryPath() string {
	stateRoot := strings.TrimSpace(os.Getenv("KOPIA_STATE_DIR"))
	if stateRoot == "" {
		stateRoot = defaultStateRoot()
	}
	return RegistryPathForStateRoot(stateRoot)
}

func defaultStateRoot() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		home = "."
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "vrooli", "resources", "kopia")
	case "windows":
		return filepath.Join(home, "AppData", "Local", "vrooli", "resources", "kopia")
	default:
		base := strings.TrimSpace(os.Getenv("XDG_STATE_HOME"))
		if base == "" {
			base = filepath.Join(home, ".local", "state")
		}
		return filepath.Join(base, "vrooli", "resources", "kopia")
	}
}

// RegistryPathForStateRoot appends the registry filename to a resolved state
// storage entry. Callers that already use the storage resolver should use this
// helper instead of rebuilding the filename.
func RegistryPathForStateRoot(stateRoot string) string {
	return filepath.Join(strings.TrimSpace(stateRoot), "registry.json")
}

// PassphraseIdentity returns the durable authority identity for one repository.
func PassphraseIdentity(repo string) (credentialauthority.Identity, error) {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return "", fmt.Errorf("repository name is required for its credential identity")
	}
	if strings.ContainsAny(repo, `/\\`) || repo == "." || repo == ".." {
		return "", fmt.Errorf("repository name %q cannot contain path separators", repo)
	}
	return credentialauthority.ParseIdentity("vrooli/kopia/" + repo)
}

// Load returns all entries sorted by name. A missing file is an empty registry.
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

// Get returns one entry and whether it exists.
func (r *Registry) Get(name string) (Entry, bool, error) {
	entries, err := r.Load()
	if err != nil {
		return Entry{}, false, err
	}
	for _, entry := range entries {
		if entry.Name == name {
			return entry, true, nil
		}
	}
	return Entry{}, false, nil
}

// Upsert inserts or replaces one entry.
func (r *Registry) Upsert(entry Entry) error {
	if strings.TrimSpace(entry.Name) == "" {
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
		if entries[i].Name == entry.Name {
			if entry.CreatedAt == "" {
				entry.CreatedAt = entries[i].CreatedAt
			}
			entries[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		if entry.CreatedAt == "" {
			entry.CreatedAt = now().UTC().Format(time.RFC3339)
		}
		entries = append(entries, entry)
	}
	return r.save(entries)
}

// Remove deletes one entry. Removing a missing entry is a no-op.
func (r *Registry) Remove(name string) error {
	entries, err := r.Load()
	if err != nil {
		return err
	}
	out := entries[:0]
	for _, entry := range entries {
		if entry.Name != name {
			out = append(out, entry)
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
	if err := os.MkdirAll(filepath.Dir(r.Path), 0o700); err != nil { //nolint:mnd // private registry directory mode
		return fmt.Errorf("create registry dir: %w", err)
	}
	tmp := r.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil { //nolint:mnd // private registry file mode
		return fmt.Errorf("write registry: %w", err)
	}
	if err := os.Rename(tmp, r.Path); err != nil {
		return fmt.Errorf("commit registry: %w", err)
	}
	return nil
}
