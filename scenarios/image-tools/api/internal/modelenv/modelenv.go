// Package modelenv makes it structurally impossible for a model-loading
// library to store weights anywhere image-tools does not manage.
//
// Every process image-tools starts that can load models (the Python sidecar,
// iopaint, smoke and build runs, perceptual metrics) receives an environment in
// which each known cache/home variable points under <managed root>/cache and the
// hub libraries run offline. Only the governed acquisition step (Acquisition)
// may reach the network, and it still writes only under the managed root. The
// variables, their library default locations and the packages that need them
// live in one data table (pins.json); adding a library is one row.
package modelenv

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

//go:embed pins.json
var pinsJSON []byte

// CacheSubdir is the directory under the managed root that holds every pinned
// cache. Nothing under it is a catalog model: it is library state that exists
// only because an operator or the governed acquisition step put it there.
const CacheSubdir = "cache"

// Dir is one directory-valued variable.
type Dir struct {
	Var      string   `json:"var"`
	Subdir   string   `json:"subdir"`
	Library  string   `json:"library"`
	Packages []string `json:"packages,omitempty"`
	Defaults []string `json:"defaults,omitempty"`
}

// Flag is one fixed-value variable.
type Flag struct {
	Var     string `json:"var"`
	Value   string `json:"value"`
	Library string `json:"library,omitempty"`
}

// Scrub is a legacy variable removed from every contained environment because
// a library lets it override the pinned directories (or warns about it).
type Scrub struct {
	Var    string `json:"var"`
	Reason string `json:"reason"`
}

// ScanOnly names default locations the stray scan inspects for a library that
// has no relocatable variable. Owner "other" marks software image-tools must
// never modify.
type ScanOnly struct {
	Library  string   `json:"library"`
	Defaults []string `json:"defaults"`
	Owner    string   `json:"owner,omitempty"`
}

// Table is the parsed pins.json.
type Table struct {
	Schema           int        `json:"schema"`
	Doc              string     `json:"doc"`
	Dirs             []Dir      `json:"dirs"`
	Scrub            []Scrub    `json:"scrub"`
	ScanOnly         []ScanOnly `json:"scan_only"`
	Watchlist        []string   `json:"watchlist"`
	Flags            []Flag     `json:"flags"`
	AcquisitionFlags []Flag     `json:"acquisition_flags"`
}

var (
	tableOnce sync.Once
	table     Table
	tableErr  error
)

// Load returns the validated embedded table.
func Load() (Table, error) {
	tableOnce.Do(func() {
		if err := json.Unmarshal(pinsJSON, &table); err != nil {
			tableErr = fmt.Errorf("modelenv: parse pins.json: %w", err)
			return
		}
		tableErr = table.validate()
	})
	return table, tableErr
}

// MustLoad is Load for package-level wiring; the table is embedded, so a parse
// failure is a build defect.
func MustLoad() Table {
	t, err := Load()
	if err != nil {
		panic(err)
	}
	return t
}

func (t Table) validate() error {
	seen := map[string]bool{}
	for _, d := range t.Dirs {
		if strings.TrimSpace(d.Var) == "" || strings.TrimSpace(d.Subdir) == "" {
			return errors.New("modelenv: every dir row needs var and subdir")
		}
		if filepath.IsAbs(d.Subdir) || !filepath.IsLocal(filepath.FromSlash(d.Subdir)) {
			return fmt.Errorf("modelenv: %s subdir %q must be a relative path inside the cache root", d.Var, d.Subdir)
		}
		if seen[d.Var] {
			return fmt.Errorf("modelenv: variable %s is declared twice", d.Var)
		}
		seen[d.Var] = true
	}
	for _, s := range t.Scrub {
		if strings.TrimSpace(s.Var) == "" || seen[s.Var] {
			return fmt.Errorf("modelenv: scrub %q is empty or also pinned", s.Var)
		}
	}
	for _, list := range [][]Flag{t.Flags, t.AcquisitionFlags} {
		for _, f := range list {
			if strings.TrimSpace(f.Var) == "" {
				return errors.New("modelenv: every flag row needs a var")
			}
			if seen[f.Var] {
				return fmt.Errorf("modelenv: variable %s is both a dir and a flag", f.Var)
			}
		}
	}
	return nil
}

var (
	rootMu sync.RWMutex
	root   string
)

// SetRoot declares the managed model root (the directory holding models/,
// adapters/ and staging/). The server calls it once at boot.
func SetRoot(managedRoot string) error {
	if strings.TrimSpace(managedRoot) == "" || !filepath.IsAbs(managedRoot) {
		return fmt.Errorf("modelenv: managed root must be an absolute path, got %q", managedRoot)
	}
	rootMu.Lock()
	root = filepath.Clean(managedRoot)
	rootMu.Unlock()
	return nil
}

// SwapRoot replaces the managed root and returns a function that restores the
// previous one. It exists for tests that need an isolated root.
func SwapRoot(managedRoot string) (restore func()) {
	rootMu.Lock()
	previous := root
	root = ""
	if strings.TrimSpace(managedRoot) != "" {
		root = filepath.Clean(managedRoot)
	}
	rootMu.Unlock()
	return func() {
		rootMu.Lock()
		root = previous
		rootMu.Unlock()
	}
}

// Root returns the managed model root. Before SetRoot (unit tests, tools that
// never configured a server) it is a private directory under the temp dir, so a
// process that forgot to configure still never writes into a home directory.
func Root() string {
	rootMu.RLock()
	defer rootMu.RUnlock()
	if root != "" {
		return root
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("image-tools-unconfigured-models-%d", os.Getuid()))
}

// CacheRoot is <managed root>/cache.
func CacheRoot() string { return filepath.Join(Root(), CacheSubdir) }

// Pins maps each directory variable to its managed directory.
func (t Table) Pins(managedRoot string) map[string]string {
	out := make(map[string]string, len(t.Dirs))
	base := filepath.Join(managedRoot, CacheSubdir)
	for _, d := range t.Dirs {
		out[d.Var] = filepath.Join(base, filepath.FromSlash(d.Subdir))
	}
	return out
}

// Mode selects which flag set an environment carries.
type Mode int

const (
	// ModeInference is every job-time process: pinned caches, hubs offline,
	// every non-loopback connection refused by the sidecar guard.
	ModeInference Mode = iota
	// ModeAcquisition is the governed fetch step: pinned caches, network allowed.
	ModeAcquisition
)

func (m Mode) String() string {
	if m == ModeAcquisition {
		return "acquisition"
	}
	return "inference"
}

// Overlay returns base with every pin and flag of mode applied. A pin always
// replaces an inherited value, so an operator-exported HF_HOME can never move a
// job's bytes out of the managed root.
func Overlay(base []string, mode Mode) []string { return overlayAt(MustLoad(), base, mode, Root()) }

func overlayAt(t Table, base []string, mode Mode, managedRoot string) []string {
	values := map[string]string{}
	order := []string{}
	for _, item := range base {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if _, dup := values[key]; !dup {
			order = append(order, key)
		}
		values[key] = value
	}
	set := func(key, value string) {
		if _, dup := values[key]; !dup {
			order = append(order, key)
		}
		values[key] = value
	}
	for key, dir := range t.Pins(managedRoot) {
		set(key, dir)
	}
	flags := t.Flags
	if mode == ModeAcquisition {
		flags = t.AcquisitionFlags
	}
	for _, f := range flags {
		set(f.Var, f.Value)
	}
	for _, s := range t.Scrub {
		delete(values, s.Var)
	}
	out := make([]string, 0, len(order))
	for _, key := range order {
		if value, ok := values[key]; ok {
			out = append(out, key+"="+value)
		}
	}
	return out
}

// Environ is Overlay applied to the current process environment.
func Environ(mode Mode) []string { return Overlay(os.Environ(), mode) }

// Apply sets cmdEnv's environment (or the process environment when nil) for
// mode and makes sure the pinned directories exist.
func Apply(cmdEnv []string, mode Mode) []string {
	if cmdEnv == nil {
		cmdEnv = os.Environ()
	}
	EnsureDirs()
	return Overlay(cmdEnv, mode)
}

// EnsureDirs creates the pinned directories so a library that expects its home
// to exist does not fall back to creating a different one.
func EnsureDirs() {
	for _, dir := range MustLoad().Pins(Root()) {
		_ = os.MkdirAll(dir, 0o755)
	}
}

// Report is the diagnostic view served by `image-tools models containment`.
type Report struct {
	ManagedRoot string            `json:"managed_root"`
	CacheRoot   string            `json:"cache_root"`
	Pins        map[string]string `json:"pins"`
	Inference   []string          `json:"inference_flags"`
	Acquisition []string          `json:"acquisition_flags"`
	Libraries   []string          `json:"libraries"`
}

// Describe reports the effective pins for the configured root.
func Describe() Report {
	t := MustLoad()
	r := Report{ManagedRoot: Root(), CacheRoot: CacheRoot(), Pins: t.Pins(Root())}
	for _, f := range t.Flags {
		r.Inference = append(r.Inference, f.Var+"="+f.Value)
	}
	for _, f := range t.AcquisitionFlags {
		r.Acquisition = append(r.Acquisition, f.Var+"="+f.Value)
	}
	for _, d := range t.Dirs {
		r.Libraries = append(r.Libraries, d.Var+" ("+d.Library+")")
	}
	sort.Strings(r.Libraries)
	return r
}

// ExpandHome resolves a leading ~/ in a table default path.
func ExpandHome(path, home string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// DefaultLocations lists every library default location the stray scan checks,
// with the owning library and whether another application owns it.
type DefaultLocation struct {
	Path    string
	Library string
	Var     string
	Other   bool
}

// DefaultLocations expands the table's default paths for home.
func (t Table) DefaultLocations(home string) []DefaultLocation {
	var out []DefaultLocation
	for _, d := range t.Dirs {
		for _, p := range d.Defaults {
			out = append(out, DefaultLocation{Path: ExpandHome(p, home), Library: d.Library, Var: d.Var})
		}
	}
	for _, s := range t.ScanOnly {
		for _, p := range s.Defaults {
			out = append(out, DefaultLocation{Path: ExpandHome(p, home), Library: s.Library, Other: s.Owner == "other"})
		}
	}
	return out
}
