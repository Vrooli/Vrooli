// Package retainedruntime retains explicitly declared runtime bytes. A release is
// not a health receipt, authority grant, or proof of historically loaded modules.
package retainedruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/vrooli/vrooli/internal/config"
)

const limitation = "Captured current declared runtime files, not proof of modules previously loaded by a running process. Selection is not proof of health or authority."

var ErrSelectionConflict = errors.New("retained runtime selection conflict")
var names = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
var ids = regexp.MustCompile(`^[a-f0-9]{64}$`)
var selectionMu sync.Mutex

type Store struct {
	base      string
	afterCopy func()
} // afterCopy is a deterministic mutation seam for package tests.
// CaptureSpec contains launch-contract metadata, never resolved environment values.
// ServiceManifest is normalized and hashed only; its contents are never persisted.
type CaptureSpec struct {
	Scenario, Variant, SourceDir, BuildIdentity, SourceRuntimeIdentity string
	ServiceManifest                                                    []byte
	ForbiddenPaths                                                     []string
}
type Interpreter struct {
	Command string `json:"command"`
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
	Mode    uint32 `json:"mode"`
}
type Release struct {
	SourceDir                                                               string
	ID, Scenario, Variant, RuntimeDir, BuildIdentity, SourceRuntimeIdentity string
	StateContract                                                           string
	ServiceManifestSHA256                                                   string
	Interpreters                                                            []Interpreter
	Paths                                                                   []string
}
type Selection struct {
	SchemaVersion int    `json:"schema_version"`
	Scenario      string `json:"scenario"`
	Variant       string `json:"variant"`
	ReleaseID     string `json:"release_id"`
	OperationID   string `json:"operation_id"`
}
type closure struct {
	StateContract string   `json:"state_contract"`
	Version       int      `json:"version"`
	Paths         []string `json:"paths"`
	Interpreters  []string `json:"interpreters"`
}
type entry struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Mode   uint32 `json:"mode"`
	SHA256 string `json:"sha256,omitempty"`
	Target string `json:"target,omitempty"`
}
type manifest struct {
	SourceDir             string        `json:"source_dir"`
	SchemaVersion         int           `json:"schema_version"`
	Scenario              string        `json:"scenario"`
	Variant               string        `json:"variant"`
	BuildIdentity         string        `json:"build_identity"`
	SourceRuntimeIdentity string        `json:"source_runtime_identity"`
	ServiceManifestSHA256 string        `json:"service_manifest_sha256"`
	Closure               closure       `json:"closure"`
	Files                 []entry       `json:"files"`
	Interpreters          []Interpreter `json:"interpreters"`
	ProvenanceLimitation  string        `json:"provenance_limitation"`
}

func NewStore(base string) *Store { return &Store{base: base} }
func (s *Store) scope(scenario, variant string) (string, error) {
	if !names.MatchString(scenario) || !names.MatchString(variant) || scenario == "." || variant == "." || scenario == ".." || variant == ".." {
		return "", fmt.Errorf("invalid scenario or variant")
	}
	if !filepath.IsAbs(s.base) {
		return "", fmt.Errorf("store base must be absolute")
	}
	return filepath.Join(s.base, scenario, variant), nil
}
func validPath(p string) bool {
	return p != "" && p != "." && !filepath.IsAbs(p) && filepath.ToSlash(filepath.Clean(p)) == p && p != ".." && !strings.HasPrefix(p, "../") && !strings.Contains(p, "\\")
}
func forbidden(p string, extras []string) bool {
	for _, part := range strings.Split(p, "/") {
		if part == ".git" || part == ".env" || strings.HasPrefix(part, ".env.") || part == "credentials" || part == "secrets" {
			return true
		}
	}
	if p == ".vrooli" || strings.HasPrefix(p, ".vrooli/") {
		return true
	}
	for _, x := range extras {
		if p == x || strings.HasPrefix(p, x+"/") {
			return true
		}
	}
	return false
}
func decode(b []byte, v any) error {
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("trailing JSON content")
	}
	return nil
}
func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func fileHash(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	_, e = io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), e
}

// realPath refuses directory aliases in the store; source symlinks are checked
// separately against the completed captured tree.
func realPath(path string) error {
	for p := filepath.Clean(path); ; p = filepath.Dir(p) {
		st, e := os.Lstat(p)
		if e == nil && st.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink store path: %s", p)
		}
		if e != nil && !os.IsNotExist(e) {
			return e
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	return nil
}
func privateDir(p string) error {
	if e := realPath(p); e != nil {
		return e
	}
	if e := os.MkdirAll(p, 0700); e != nil {
		return e
	}
	st, e := os.Stat(p)
	if e != nil {
		return e
	}
	if st.Mode().Perm() != 0700 {
		return fmt.Errorf("store directory must be private: %s", p)
	}
	return nil
}
func scan(ctx context.Context, root string, paths []string, extras []string) ([]entry, error) {
	seen := map[string]entry{}
	add := func(p string) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if !validPath(p) || forbidden(p, extras) {
			return fmt.Errorf("forbidden runtime path %q", p)
		}
		st, e := os.Lstat(filepath.Join(root, p))
		if e != nil {
			return e
		}
		if st.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
			return fmt.Errorf("special runtime mode: %s", p)
		}
		x := entry{Path: p, Mode: uint32(st.Mode().Perm())}
		switch {
		case st.Mode().IsRegular():
			x.Kind = "file"
			x.SHA256, e = fileHash(filepath.Join(root, p))
		case st.IsDir():
			x.Kind = "dir"
		case st.Mode()&os.ModeSymlink != 0:
			x.Kind = "symlink"
			x.Target, e = os.Readlink(filepath.Join(root, p))
			if filepath.IsAbs(x.Target) || !validPath(filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(p), x.Target)))) {
				return fmt.Errorf("escaping symlink %s", p)
			}
		default:
			return fmt.Errorf("special runtime file %s", p)
		}
		if e != nil {
			return e
		}
		seen[p] = x
		return nil
	}
	for _, p := range paths {
		if !validPath(p) {
			return nil, fmt.Errorf("invalid runtime path %q", p)
		}
		for parent := filepath.Dir(p); parent != "."; parent = filepath.Dir(parent) {
			if e := add(parent); e != nil {
				return nil, e
			}
			if seen[parent].Kind != "dir" {
				return nil, fmt.Errorf("symlink ancestor %s", parent)
			}
		}
		e := filepath.WalkDir(filepath.Join(root, p), func(path string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			rel, e := filepath.Rel(root, path)
			if e != nil {
				return e
			}
			return add(filepath.ToSlash(rel))
		})
		if e != nil {
			return nil, e
		}
	}
	result := make([]entry, 0, len(seen))
	for _, x := range seen {
		result = append(result, x)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, nil
}
func checkLinks(root string, files []entry) error {
	for _, x := range files {
		if x.Kind != "symlink" {
			continue
		}
		p, e := filepath.EvalSymlinks(filepath.Join(root, x.Path))
		if e != nil {
			return e
		}
		rel, e := filepath.Rel(root, p)
		if e != nil || !validPath(filepath.ToSlash(rel)) {
			return fmt.Errorf("symlink outside runtime: %s", x.Path)
		}
	}
	return nil
}
func resolveInterpreters(commands []string) ([]Interpreter, error) {
	out := []Interpreter{}
	seen := map[string]bool{}
	for _, name := range commands {
		if !names.MatchString(name) || seen[name] {
			return nil, fmt.Errorf("invalid interpreter %q", name)
		}
		seen[name] = true
		p, e := exec.LookPath(name)
		if e != nil {
			return nil, e
		}
		p, e = filepath.Abs(p)
		if e != nil {
			return nil, e
		}
		p, e = filepath.EvalSymlinks(p)
		if e != nil {
			return nil, e
		}
		st, e := os.Stat(p)
		if e != nil {
			return nil, e
		}
		if !st.Mode().IsRegular() || st.Mode().Perm()&0111 == 0 {
			return nil, fmt.Errorf("interpreter not executable")
		}
		hash, e := fileHash(p)
		if e != nil {
			return nil, e
		}
		out = append(out, Interpreter{name, p, hash, uint32(st.Mode().Perm())})
	}
	return out, nil
}
func (s *Store) Capture(ctx context.Context, spec CaptureSpec) (Release, error) {
	fail := func(e error) (Release, error) { return Release{}, e }
	scope, e := s.scope(spec.Scenario, spec.Variant)
	if e != nil {
		return fail(e)
	}
	if spec.BuildIdentity == "" || !json.Valid(spec.ServiceManifest) {
		return fail(fmt.Errorf("build identity and JSON service manifest required"))
	}
	for _, p := range spec.ForbiddenPaths {
		if !validPath(p) {
			return fail(fmt.Errorf("invalid forbidden path"))
		}
	}
	source, e := filepath.EvalSymlinks(spec.SourceDir)
	if e != nil {
		return fail(e)
	}
	source, e = filepath.Abs(source)
	if e != nil {
		return fail(e)
	}
	closurePath := filepath.Join(source, ".vrooli/runtime-release.json")
	if e = realPath(closurePath); e != nil {
		return fail(e)
	}
	b, e := os.ReadFile(closurePath)
	if e != nil {
		return fail(e)
	}
	var c closure
	if e = decode(b, &c); e != nil {
		return fail(e)
	}
	if c.Version != 1 || len(c.Paths) == 0 || strings.TrimSpace(c.StateContract) == "" {
		return fail(fmt.Errorf("complete version 1 runtime closure required"))
	}
	files, e := scan(ctx, source, c.Paths, spec.ForbiddenPaths)
	if e != nil {
		return fail(e)
	}
	interpreters, e := resolveInterpreters(c.Interpreters)
	if e != nil {
		return fail(e)
	}
	if e = privateDir(s.base); e != nil {
		return fail(e)
	}
	if e = privateDir(filepath.Dir(scope)); e != nil {
		return fail(e)
	}
	if e = privateDir(scope); e != nil {
		return fail(e)
	}
	stage, e := os.MkdirTemp(scope, ".capture-")
	if e != nil {
		return fail(e)
	}
	defer os.RemoveAll(stage)
	runtime := filepath.Join(stage, "runtime")
	if e = os.Mkdir(runtime, 0700); e != nil {
		return fail(e)
	}
	for _, x := range files {
		if e = ctx.Err(); e != nil {
			return fail(e)
		}
		dst := filepath.Join(runtime, x.Path)
		switch x.Kind {
		case "dir":
			e = os.MkdirAll(dst, 0700)
		case "symlink":
			e = os.Symlink(x.Target, dst)
		case "file":
			e = config.WriteOwnedFileDurableStream(dst, os.FileMode(x.Mode), func(w io.Writer) error {
				f, e := os.Open(filepath.Join(source, x.Path))
				if e != nil {
					return e
				}
				defer f.Close()
				h := sha256.New()
				_, e = io.Copy(io.MultiWriter(w, h), f)
				if e != nil {
					return e
				}
				if hex.EncodeToString(h.Sum(nil)) != x.SHA256 {
					return fmt.Errorf("source changed during capture: %s", x.Path)
				}
				return nil
			})
		}
		if e != nil {
			return fail(e)
		}
	}
	for i := len(files) - 1; i >= 0; i-- {
		x := files[i]
		if x.Kind == "dir" {
			if e = os.Chmod(filepath.Join(runtime, x.Path), os.FileMode(x.Mode)); e != nil {
				return fail(e)
			}
		}
	}
	if s.afterCopy != nil {
		s.afterCopy()
	}
	after, e := scan(ctx, source, c.Paths, spec.ForbiddenPaths)
	if e != nil {
		return fail(e)
	}
	if !reflect.DeepEqual(files, after) {
		return fail(fmt.Errorf("source changed during capture"))
	}
	again, e := os.ReadFile(closurePath)
	if e != nil || !reflect.DeepEqual(b, again) {
		return fail(fmt.Errorf("runtime closure changed during capture"))
	}
	if e = checkLinks(runtime, files); e != nil {
		return fail(e)
	}
	var serviceValue any
	if e = json.Unmarshal(spec.ServiceManifest, &serviceValue); e != nil {
		return fail(e)
	}
	normalized, e := json.Marshal(serviceValue)
	if e != nil {
		return fail(e)
	}
	m := manifest{SourceDir: source, SchemaVersion: 1, Scenario: spec.Scenario, Variant: spec.Variant, BuildIdentity: spec.BuildIdentity, SourceRuntimeIdentity: spec.SourceRuntimeIdentity, ServiceManifestSHA256: digest(normalized), Closure: c, Files: files, Interpreters: interpreters, ProvenanceLimitation: limitation}
	raw, e := json.Marshal(m)
	if e != nil {
		return fail(e)
	}
	id := digest(raw)
	if e = config.WriteOwnedFileDurable(filepath.Join(stage, "manifest.json"), raw, 0600); e != nil {
		return fail(e)
	}
	if e = syncTree(stage); e != nil {
		return fail(e)
	}
	if e = os.Rename(stage, filepath.Join(scope, id)); e != nil {
		if _, v := s.Verify(ctx, spec.Scenario, spec.Variant, id); v != nil {
			return fail(e)
		}
	}
	if e = syncDir(scope); e != nil {
		return fail(e)
	}
	return s.Verify(ctx, spec.Scenario, spec.Variant, id)
}
func syncDir(p string) error {
	f, e := os.Open(p)
	if e != nil {
		return e
	}
	defer f.Close()
	return f.Sync()
}
func syncTree(root string) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return syncDir(p)
		}
		return nil
	})
}
func (s *Store) Verify(ctx context.Context, scenario, variant, id string) (Release, error) {
	var out Release
	scope, e := s.scope(scenario, variant)
	if e != nil {
		return out, e
	}
	if !ids.MatchString(id) {
		return out, fmt.Errorf("invalid release ID")
	}
	dir := filepath.Join(scope, id)
	if e = realPath(dir); e != nil {
		return out, e
	}
	for _, p := range []string{s.base, filepath.Dir(scope), scope, dir} {
		st, err := os.Lstat(p)
		if err != nil {
			return out, err
		}
		if !st.IsDir() || st.Mode().Perm() != 0700 {
			return out, fmt.Errorf("nonprivate release directory: %s", p)
		}
	}
	entries, e := os.ReadDir(dir)
	if e != nil {
		return out, e
	}
	if len(entries) != 2 || entries[0].Name() != "manifest.json" || entries[1].Name() != "runtime" {
		return out, fmt.Errorf("unexpected release entries")
	}
	st, e := os.Lstat(filepath.Join(dir, "manifest.json"))
	if e != nil {
		return out, e
	}
	if !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
		return out, fmt.Errorf("invalid manifest mode")
	}
	raw, e := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if e != nil {
		return out, e
	}
	if digest(raw) != id {
		return out, fmt.Errorf("manifest content identity mismatch")
	}
	var m manifest
	if e = decode(raw, &m); e != nil {
		return out, e
	}
	if !filepath.IsAbs(m.SourceDir) || filepath.Clean(m.SourceDir) != m.SourceDir || m.SchemaVersion != 1 || m.Scenario != scenario || m.Variant != variant || m.ProvenanceLimitation != limitation || m.Closure.Version != 1 || len(m.Closure.Paths) == 0 || strings.TrimSpace(m.Closure.StateContract) == "" || m.BuildIdentity == "" || !ids.MatchString(m.ServiceManifestSHA256) {
		return out, fmt.Errorf("manifest scope or schema mismatch")
	}
	runtime := filepath.Join(dir, "runtime")
	st, e = os.Lstat(runtime)
	if e != nil {
		return out, e
	}
	if !st.IsDir() || st.Mode().Perm() != 0700 {
		return out, fmt.Errorf("invalid runtime root")
	}
	children, e := os.ReadDir(runtime)
	if e != nil {
		return out, e
	}
	paths := []string{}
	for _, child := range children {
		paths = append(paths, child.Name())
	}
	files, e := scan(ctx, runtime, paths, nil)
	if e != nil {
		return out, e
	}
	if !reflect.DeepEqual(files, m.Files) {
		return out, fmt.Errorf("runtime tree differs from manifest")
	}
	if e = checkLinks(runtime, files); e != nil {
		return out, e
	}
	for _, i := range m.Interpreters {
		st, e := os.Lstat(i.Path)
		if e != nil {
			return out, e
		}
		h, e := fileHash(i.Path)
		if e != nil {
			return out, e
		}
		if h != i.SHA256 || uint32(st.Mode().Perm()) != i.Mode || !st.Mode().IsRegular() {
			return out, fmt.Errorf("interpreter changed: %s", i.Command)
		}
	}
	return Release{SourceDir: m.SourceDir, ID: id, Scenario: scenario, Variant: variant, RuntimeDir: runtime, BuildIdentity: m.BuildIdentity, SourceRuntimeIdentity: m.SourceRuntimeIdentity, StateContract: m.Closure.StateContract, ServiceManifestSHA256: m.ServiceManifestSHA256, Interpreters: m.Interpreters, Paths: append([]string(nil), m.Closure.Paths...)}, nil
}

// ReadSelection reads the intent pointer, not a health or integrity receipt.
// Consumers must Verify its nonempty ReleaseID before using any runtime bytes.
func (s *Store) ReadSelection(scenario, variant string) (Selection, error) {
	scope, e := s.scope(scenario, variant)
	if e != nil {
		return Selection{}, e
	}
	p := filepath.Join(scope, "selection.json")
	if e = realPath(p); e != nil {
		return Selection{}, e
	}
	b, e := os.ReadFile(p)
	if os.IsNotExist(e) {
		return Selection{SchemaVersion: 1, Scenario: scenario, Variant: variant}, nil
	}
	if e != nil {
		return Selection{}, e
	}
	st, e := os.Lstat(p)
	if e != nil {
		return Selection{}, e
	}
	if !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
		return Selection{}, fmt.Errorf("invalid selection file mode")
	}
	var sel Selection
	if e = decode(b, &sel); e != nil {
		return sel, e
	}
	if sel.SchemaVersion != 1 || sel.Scenario != scenario || sel.Variant != variant || (sel.ReleaseID != "" && !ids.MatchString(sel.ReleaseID)) {
		return Selection{}, fmt.Errorf("invalid selection")
	}
	return sel, nil
}

// Select requires the caller's cross-process scenario lock. CAS compares IDs,
// including the empty working-source selection. It never launches anything.
func (s *Store) Select(ctx context.Context, scenario, variant, id, expected, operation string) (Selection, error) {
	selectionMu.Lock()
	defer selectionMu.Unlock()
	if e := ctx.Err(); e != nil {
		return Selection{}, e
	}
	if strings.TrimSpace(operation) == "" {
		return Selection{}, fmt.Errorf("operation ID required")
	}
	scope, e := s.scope(scenario, variant)
	if e != nil {
		return Selection{}, e
	}
	current, e := s.ReadSelection(scenario, variant)
	if e != nil {
		return Selection{}, e
	}
	if current.ReleaseID != expected {
		return Selection{}, ErrSelectionConflict
	}
	if id != "" {
		if _, e = s.Verify(ctx, scenario, variant, id); e != nil {
			return Selection{}, e
		}
	}
	for _, p := range []string{s.base, filepath.Dir(scope), scope} {
		if e = privateDir(p); e != nil {
			return Selection{}, e
		}
	}
	if e = ctx.Err(); e != nil {
		return Selection{}, e
	}
	sel := Selection{1, scenario, variant, id, operation}
	b, e := json.Marshal(sel)
	if e != nil {
		return Selection{}, e
	}
	if e = config.WriteOwnedFileDurable(filepath.Join(scope, "selection.json"), b, 0600); e != nil {
		return Selection{}, e
	}
	return sel, nil
}
