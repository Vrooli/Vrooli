// Package modelstray finds model bytes that exist outside image-tools' receipts
// and brings them under management.
//
// Containment (internal/modelenv) stops new bytes from escaping. This package
// handles the ones that already did, and the ones that appear inside the managed
// root without a receipt. It scans the default cache locations of every library
// in the pin table, looks for files a catalog entry refers to, and for entries in
// the managed root with no install receipt. Each finding carries the bytes and
// one action: adopt (move into managed storage and record a receipt), reclaim
// (delete a copy that is already managed, or untracked bytes old enough that no
// install can still own them) or report only (software image-tools does not own,
// for example Ollama, ComfyUI or another scenario's Hugging Face models). Nothing
// is deleted or moved by a scan.
package modelstray

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
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"image-tools/internal/fetch"
	"image-tools/internal/modelenv"
)

// Finding kinds.
const (
	KindDefaultCache      = "default_cache"
	KindManagedUntracked  = "managed_untracked"
	KindManagedCache      = "managed_cache"
	KindStaging           = "staging"
	KindExternalReference = "external_reference"
)

// Owners.
const (
	OwnerCatalog = "image_tools_catalog"
	OwnerOther   = "other_application"
	OwnerUnknown = "unknown"
	OwnerManaged = "managed"
)

// Actions.
const (
	ActionAdopt      = "adopt"
	ActionReclaim    = "reclaim"
	ActionReportOnly = "report_only"
	ActionNone       = "none"
)

const (
	defaultUntrackedGrace = 24 * time.Hour
	maxWalkFiles          = 200000
	maxNameMatchFiles     = 5000
)

// Entry is one catalog model or adapter as the scan needs to see it.
type Entry struct {
	Kind      string // "model" or "adapter"
	ID        string
	Source    fetch.Source
	Installed bool
	Path      string
}

// Config wires the scanner to the catalog and the receipts.
type Config struct {
	Root string
	Home string
	// Entries lists every catalog entry with its install state.
	Entries func(ctx context.Context) ([]Entry, error)
	// Adopt records a receipt for a catalog entry whose files already sit at its
	// managed location (kind "model" or "adapter").
	Adopt func(ctx context.Context, kind, id string) error
	Now   func() time.Time
	// UntrackedGrace is how old an unknown entry in the managed root must be
	// before it may be reclaimed (default 24h): younger ones may be an install.
	UntrackedGrace time.Duration
	// Table defaults to the embedded pin table.
	Table *modelenv.Table
}

// Scanner scans, adopts and reclaims.
type Scanner struct{ cfg Config }

// New validates cfg.
func New(cfg Config) (*Scanner, error) {
	if !filepath.IsAbs(cfg.Root) {
		return nil, errors.New("modelstray: managed root must be an absolute path")
	}
	if cfg.Home == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("modelstray: resolve home directory: %w", err)
		}
		cfg.Home = home
	}
	if cfg.Entries == nil {
		return nil, errors.New("modelstray: a catalog entry source is required")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.UntrackedGrace <= 0 {
		cfg.UntrackedGrace = defaultUntrackedGrace
	}
	if cfg.Table == nil {
		t := modelenv.MustLoad()
		cfg.Table = &t
	}
	return &Scanner{cfg: cfg}, nil
}

// Finding is one set of bytes that needs a decision.
type Finding struct {
	ID          string    `json:"id"`
	Path        string    `json:"path"`
	Kind        string    `json:"kind"`
	Library     string    `json:"library,omitempty"`
	Owner       string    `json:"owner"`
	CatalogIDs  []string  `json:"catalog_ids,omitempty"`
	Bytes       int64     `json:"bytes"`
	Files       int       `json:"files"`
	ModTime     time.Time `json:"modified_at,omitempty"`
	Action      string    `json:"action"`
	Destination string    `json:"destination,omitempty"`
	Reason      string    `json:"reason"`
}

// ScannedPath records a location that was inspected, so an empty report is
// distinguishable from a scan that looked nowhere.
type ScannedPath struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Source string `json:"source"`
}

// Totals sums findings by action.
type Totals struct {
	StrayBytes       int64 `json:"stray_bytes"`
	AdoptableBytes   int64 `json:"adoptable_bytes"`
	ReclaimableBytes int64 `json:"reclaimable_bytes"`
	ReportOnlyBytes  int64 `json:"report_only_bytes"`
	ManagedCacheByte int64 `json:"managed_cache_bytes"`
}

// Report is a scan result.
type Report struct {
	ScannedAt time.Time     `json:"scanned_at"`
	Root      string        `json:"managed_root"`
	Scanned   []ScannedPath `json:"scanned"`
	Findings  []Finding     `json:"findings"`
	Totals    Totals        `json:"totals"`
}

// Receipt is the durable record of an adopt or reclaim.
type Receipt struct {
	At          time.Time `json:"at"`
	Action      string    `json:"action"`
	FindingID   string    `json:"finding_id"`
	Path        string    `json:"path"`
	Destination string    `json:"destination,omitempty"`
	Bytes       int64     `json:"bytes"`
	Files       int       `json:"files"`
	Library     string    `json:"library,omitempty"`
	Owner       string    `json:"owner"`
	CatalogIDs  []string  `json:"catalog_ids,omitempty"`
	Note        string    `json:"note,omitempty"`
}

// ReceiptsPath is the append-only log of adopt and reclaim actions.
func ReceiptsPath(root string) string {
	return filepath.Join(root, "receipts", "stray-actions.jsonl")
}

func findingID(p string) string {
	sum := sha256.Sum256([]byte(filepath.Clean(p)))
	return hex.EncodeToString(sum[:])[:12]
}

type treeStat struct {
	bytes   int64
	files   int
	modTime time.Time
}

// measure sums regular files without following symlinks (a Hub snapshot's
// symlinks point at blobs counted once), bounded in file count.
func measure(root string) treeStat {
	var t treeStat
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}
		if d.Type().IsRegular() {
			// Directory mtimes move when anything is created or renamed; only
			// file writes say whether bytes are still arriving.
			if info.ModTime().After(t.modTime) {
				t.modTime = info.ModTime()
			}
			t.bytes += info.Size()
			t.files++
			if t.files >= maxWalkFiles {
				return fs.SkipAll
			}
		}
		return nil
	})
	return t
}

type catalogIndex struct {
	repos map[string][]string // lower-case org/name -> entry ids
	files map[string][]string // lower-case basename -> entry ids
	byID  map[string]Entry
}

var hubURL = regexp.MustCompile(`huggingface\.co/([^/]+/[^/]+)/`)

func buildIndex(entries []Entry) catalogIndex {
	idx := catalogIndex{repos: map[string][]string{}, files: map[string][]string{}, byID: map[string]Entry{}}
	add := func(m map[string][]string, key, id string) {
		key = strings.ToLower(key)
		for _, existing := range m[key] {
			if existing == id {
				return
			}
		}
		m[key] = append(m[key], id)
	}
	for _, e := range entries {
		label := e.Kind + ":" + e.ID
		idx.byID[label] = e
		if e.Source.HasRepo() {
			add(idx.repos, e.Source.Repo.RepoID, label)
		}
		for _, c := range e.Source.HubCompanions {
			add(idx.repos, c.RepoID, label)
		}
		urls := []string{e.Source.DownloadURL}
		assets, _ := e.Source.ResolvedAssets()
		for _, a := range assets {
			urls = append(urls, a.URL)
			if a.Filename != "" {
				add(idx.files, path.Base(a.Filename), label)
			}
		}
		for _, u := range urls {
			if m := hubURL.FindStringSubmatch(u); m != nil {
				add(idx.repos, m[1], label)
			}
			if u != "" {
				add(idx.files, path.Base(strings.SplitN(u, "?", 2)[0]), label)
			}
		}
	}
	return idx
}

func inside(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func resolved(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return filepath.Clean(p)
}

var expandDirs = map[string]bool{"hub": true, "models": true, "checkpoints": true}

func unitsOf(root string) []string {
	info, err := os.Lstat(root)
	if err != nil {
		return nil
	}
	if !info.IsDir() {
		return []string{root}
	}
	var out []string
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		child := filepath.Join(root, e.Name())
		if e.IsDir() && expandDirs[e.Name()] {
			grand, _ := os.ReadDir(child)
			for _, g := range grand {
				out = append(out, filepath.Join(child, g.Name()))
			}
			continue
		}
		out = append(out, child)
	}
	return out
}

var hubRepoDir = regexp.MustCompile(`^(models|datasets|spaces)--(.+)$`)

func hubRepoID(name string) (string, bool) {
	m := hubRepoDir.FindStringSubmatch(name)
	if m == nil || m[1] != "models" {
		return "", false
	}
	org, repo, ok := strings.Cut(m[2], "--")
	if !ok || org == "" || repo == "" {
		return "", false
	}
	return org + "/" + repo, true
}

func sorted(ids []string) []string { sort.Strings(ids); return ids }

// Scan inspects without changing anything.
func (s *Scanner) Scan(ctx context.Context) (Report, error) {
	entries, err := s.cfg.Entries(ctx)
	if err != nil {
		return Report{}, fmt.Errorf("modelstray: read the catalog: %w", err)
	}
	idx := buildIndex(entries)
	root := resolved(s.cfg.Root)
	report := Report{ScannedAt: s.cfg.Now().UTC(), Root: s.cfg.Root, Findings: []Finding{}, Scanned: []ScannedPath{}}
	pins := s.cfg.Table.Pins(s.cfg.Root)

	seen := map[string]bool{}
	for _, loc := range s.cfg.Table.DefaultLocations(s.cfg.Home) {
		p := filepath.Clean(loc.Path)
		if seen[p] {
			continue
		}
		seen[p] = true
		_, statErr := os.Lstat(p)
		report.Scanned = append(report.Scanned, ScannedPath{Path: p, Exists: statErr == nil, Source: loc.Library})
		if statErr != nil || inside(root, resolved(p)) {
			continue
		}
		for _, unit := range unitsOf(p) {
			if f, ok := s.classifyDefault(unit, p, loc, pins, idx); ok {
				report.Findings = append(report.Findings, f)
			}
		}
	}

	report.Findings = append(report.Findings, s.scanManaged(ctx, root, entries, &report)...)
	sort.SliceStable(report.Findings, func(i, j int) bool {
		if report.Findings[i].Bytes != report.Findings[j].Bytes {
			return report.Findings[i].Bytes > report.Findings[j].Bytes
		}
		return report.Findings[i].Path < report.Findings[j].Path
	})
	for _, f := range report.Findings {
		switch f.Action {
		case ActionAdopt:
			report.Totals.AdoptableBytes += f.Bytes
		case ActionReclaim:
			report.Totals.ReclaimableBytes += f.Bytes
		case ActionReportOnly:
			report.Totals.ReportOnlyBytes += f.Bytes
		}
		if f.Kind == KindManagedCache {
			report.Totals.ManagedCacheByte += f.Bytes
		} else if f.Kind != KindStaging {
			report.Totals.StrayBytes += f.Bytes
		}
	}
	return report, nil
}

func (s *Scanner) classifyDefault(unit, locRoot string, loc modelenv.DefaultLocation, pins map[string]string, idx catalogIndex) (Finding, bool) {
	st := measure(unit)
	if st.files == 0 {
		return Finding{}, false
	}
	f := Finding{ID: findingID(unit), Path: unit, Kind: KindDefaultCache, Library: loc.Library, Bytes: st.bytes, Files: st.files, ModTime: st.modTime, Owner: OwnerUnknown, Action: ActionReportOnly}
	if loc.Other {
		f.Owner = OwnerOther
		f.Reason = "owned by another application (" + loc.Library + "); image-tools never modifies it"
		return f, true
	}
	var ids []string
	base := filepath.Base(unit)
	if repo, ok := hubRepoID(base); ok {
		// A Hub cache directory names the repository it holds, so only a
		// catalog entry that names that repository can claim it. File names
		// are never a basis: every transformers repo has a config.json.
		ids = idx.repos[strings.ToLower(repo)]
	} else {
		ids = s.matchFiles(unit, idx)
	}
	if len(ids) == 0 {
		f.Reason = "no image-tools catalog entry refers to this; it may belong to another scenario or tool, so it is reported and left alone"
		return f, true
	}
	f.Owner, f.CatalogIDs = OwnerCatalog, sorted(ids)
	managedDir := pins[loc.Var]
	if loc.Var == "" || managedDir == "" {
		f.Reason = "a catalog entry refers to this, but " + loc.Library + " has no relocatable cache variable: reported only"
		return f, true
	}
	rel, err := filepath.Rel(locRoot, unit)
	if err != nil {
		return f, true
	}
	f.Destination = filepath.Join(managedDir, rel)
	if existing := measure(f.Destination); existing.files > 0 {
		if existing.bytes >= st.bytes {
			f.Action = ActionReclaim
			f.Reason = fmt.Sprintf("the managed copy at %s already holds at least these %d bytes: this duplicate can be reclaimed", f.Destination, st.bytes)
		} else {
			f.Reason = "a smaller managed copy exists at the destination: resolve by hand (adopt would overwrite it)"
		}
		return f, true
	}
	f.Action = ActionAdopt
	f.Reason = "catalog entry " + strings.Join(f.CatalogIDs, ", ") + " refers to these files: move them into managed storage and record a receipt"
	return f, true
}

// genericFileNames are names shared by unrelated models (any transformers repo
// has a config.json), so they never show that a catalog entry owns a file.
var genericFileNames = map[string]bool{
	"config.json": true, "model.safetensors": true, "pytorch_model.bin": true, "model.onnx": true,
	"tokenizer.json": true, "tokenizer_config.json": true, "vocab.json": true, "merges.txt": true,
	"vocab.txt": true, "special_tokens_map.json": true, "preprocessor_config.json": true,
	"generation_config.json": true, "model.ckpt": true, "model.bin": true, "readme.md": true,
	"diffusion_pytorch_model.safetensors": true, "scheduler_config.json": true, "model_index.json": true,
}

func (s *Scanner) matchFiles(unit string, idx catalogIndex) []string {
	var ids []string
	count := 0
	_ = filepath.WalkDir(unit, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if count++; count > maxNameMatchFiles {
			return fs.SkipAll
		}
		if genericFileNames[strings.ToLower(d.Name())] {
			return nil
		}
		ids = append(ids, idx.files[strings.ToLower(d.Name())]...)
		return nil
	})
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

var managedTop = map[string]bool{"models": true, "adapters": true, "staging": true, "blobs": true, "deployment": true, "cache": true, "receipts": true}

func (s *Scanner) scanManaged(ctx context.Context, root string, entries []Entry, report *Report) []Finding {
	var out []Finding
	report.Scanned = append(report.Scanned, ScannedPath{Path: s.cfg.Root, Exists: true, Source: "managed model root"})
	tracked := map[string]bool{}
	for _, e := range entries {
		if !e.Installed || e.Path == "" {
			continue
		}
		if !inside(root, resolved(e.Path)) {
			st := measure(e.Path)
			out = append(out, Finding{ID: findingID(e.Path), Path: e.Path, Kind: KindExternalReference, Owner: OwnerCatalog, CatalogIDs: []string{e.Kind + ":" + e.ID}, Bytes: st.bytes, Files: st.files, ModTime: st.modTime, Action: ActionReportOnly,
				Reason: "this entry's receipt points outside the managed root (a local-path entry): the bytes are referenced, not owned or moved; copy them into a managed install if they should be retained with the rest"})
			continue
		}
		tracked[filepath.Clean(resolved(e.Path))] = true
	}
	catalogByDir := map[string]Entry{}
	for _, e := range entries {
		catalogByDir[e.Kind+"/"+e.ID] = e
	}
	for _, sub := range []struct{ dir, kind string }{{"models", "model"}, {"adapters", "adapter"}} {
		dir := filepath.Join(s.cfg.Root, sub.dir)
		report.Scanned = append(report.Scanned, ScannedPath{Path: dir, Exists: dirExists(dir), Source: "managed " + sub.kind + " directory"})
		children, _ := os.ReadDir(dir)
		for _, c := range children {
			p := filepath.Join(dir, c.Name())
			if tracked[filepath.Clean(resolved(p))] {
				continue
			}
			st := measure(p)
			f := Finding{ID: findingID(p), Path: p, Kind: KindManagedUntracked, Library: "image-tools " + sub.kind + " storage", Owner: OwnerUnknown, Bytes: st.bytes, Files: st.files, ModTime: st.modTime, Action: ActionReportOnly}
			if entry, ok := catalogByDir[sub.kind+"/"+c.Name()]; ok && c.IsDir() && s.cfg.Adopt != nil {
				f.Owner, f.CatalogIDs, f.Action = OwnerCatalog, []string{entry.Kind + ":" + entry.ID}, ActionAdopt
				f.Reason = "files for catalog " + sub.kind + " " + entry.ID + " sit at its managed location with no install receipt: verify against the catalog and record one"
			} else if age := s.cfg.Now().Sub(st.modTime); st.files > 0 && age >= s.cfg.UntrackedGrace {
				f.Action = ActionReclaim
				f.Reason = fmt.Sprintf("no catalog entry or receipt owns this and nothing has written to it for %s: it can be reclaimed", age.Round(time.Hour))
			} else {
				f.Reason = "no receipt owns this, but it is recent or empty: it may be an install or leftover in progress, so it is reported only"
			}
			out = append(out, f)
		}
	}
	top, _ := os.ReadDir(s.cfg.Root)
	for _, c := range top {
		p := filepath.Join(s.cfg.Root, c.Name())
		switch {
		case c.Name() == "cache" && c.IsDir():
			caches, _ := os.ReadDir(p)
			for _, cc := range caches {
				cp := filepath.Join(p, cc.Name())
				st := measure(cp)
				if st.files == 0 {
					continue
				}
				out = append(out, Finding{ID: findingID(cp), Path: cp, Kind: KindManagedCache, Library: "pinned library cache", Owner: OwnerManaged, Bytes: st.bytes, Files: st.files, ModTime: st.modTime, Action: ActionNone,
					Reason: "library cache under the managed root; counted here because it has no install receipt by design"})
			}
		case c.Name() == "staging" && c.IsDir():
			st := measure(p)
			if st.files > 0 {
				out = append(out, Finding{ID: findingID(p), Path: p, Kind: KindStaging, Library: "acquisition staging", Owner: OwnerManaged, Bytes: st.bytes, Files: st.files, ModTime: st.modTime, Action: ActionNone,
					Reason: "in-flight or abandoned acquisition staging; `image-tools models storage` owns reclaiming abandoned staging"})
			}
		case managedTop[c.Name()]:
		default:
			st := measure(p)
			out = append(out, Finding{ID: findingID(p), Path: p, Kind: KindManagedUntracked, Library: "managed root", Owner: OwnerUnknown, Bytes: st.bytes, Files: st.files, ModTime: st.modTime, Action: ActionReportOnly,
				Reason: "an entry in the managed root that image-tools does not create: reported only"})
		}
	}
	return out
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func (s *Scanner) find(ctx context.Context, id string) (Finding, error) {
	report, err := s.Scan(ctx)
	if err != nil {
		return Finding{}, err
	}
	for _, f := range report.Findings {
		if f.ID == id {
			return f, nil
		}
	}
	return Finding{}, fmt.Errorf("modelstray: no finding %q in a fresh scan (it may already have been handled); run the scan again", id)
}

// Adopt brings one finding under management and records a receipt.
func (s *Scanner) Adopt(ctx context.Context, id string) (Receipt, error) {
	f, err := s.find(ctx, id)
	if err != nil {
		return Receipt{}, err
	}
	if f.Action != ActionAdopt {
		return Receipt{}, fmt.Errorf("modelstray: finding %s cannot be adopted (action %s): %s", id, f.Action, f.Reason)
	}
	receipt := Receipt{At: s.cfg.Now().UTC(), Action: ActionAdopt, FindingID: f.ID, Path: f.Path, Destination: f.Destination, Bytes: f.Bytes, Files: f.Files, Library: f.Library, Owner: f.Owner, CatalogIDs: f.CatalogIDs}
	if f.Kind == KindManagedUntracked {
		kind, entryID, _ := strings.Cut(f.CatalogIDs[0], ":")
		if s.cfg.Adopt == nil {
			return Receipt{}, errors.New("modelstray: receipt adoption is not wired")
		}
		if err := s.cfg.Adopt(ctx, kind, entryID); err != nil {
			return Receipt{}, fmt.Errorf("modelstray: adopt %s %s: %w", kind, entryID, err)
		}
		receipt.Destination = f.Path
		receipt.Note = "install receipt recorded in place; run `image-tools models install " + entryID + "` to revalidate the runtime"
	} else {
		if err := fetch.EnsureInsideRoot(s.cfg.Root, f.Destination); err != nil {
			return Receipt{}, err
		}
		if err := moveTree(f.Path, f.Destination); err != nil {
			return Receipt{}, fmt.Errorf("modelstray: adopt %s: %w", f.Path, err)
		}
	}
	return receipt, s.record(receipt)
}

// Reclaim deletes one finding that is safe to delete and records a receipt.
func (s *Scanner) Reclaim(ctx context.Context, id string) (Receipt, error) {
	f, err := s.find(ctx, id)
	if err != nil {
		return Receipt{}, err
	}
	if f.Action != ActionReclaim {
		return Receipt{}, fmt.Errorf("modelstray: finding %s cannot be reclaimed (action %s): %s", id, f.Action, f.Reason)
	}
	if f.Kind == KindDefaultCache {
		if existing := measure(f.Destination); existing.bytes < f.Bytes {
			return Receipt{}, fmt.Errorf("modelstray: refusing to delete %s: the managed copy at %s shrank since the scan", f.Path, f.Destination)
		}
	}
	if err := os.RemoveAll(f.Path); err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return Receipt{}, fmt.Errorf("modelstray: %s is protected (%v); request a storage-manager deletion grant for that path and retry", f.Path, err)
		}
		return Receipt{}, fmt.Errorf("modelstray: reclaim %s: %w", f.Path, err)
	}
	if _, err := os.Lstat(f.Path); !os.IsNotExist(err) {
		return Receipt{}, fmt.Errorf("modelstray: %s still exists after reclaim", f.Path)
	}
	receipt := Receipt{At: s.cfg.Now().UTC(), Action: ActionReclaim, FindingID: f.ID, Path: f.Path, Destination: f.Destination, Bytes: f.Bytes, Files: f.Files, Library: f.Library, Owner: f.Owner, CatalogIDs: f.CatalogIDs}
	return receipt, s.record(receipt)
}

func (s *Scanner) record(r Receipt) error {
	p := ReceiptsPath(s.cfg.Root)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	line, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = file.Write(append(line, '\n'))
	return err
}

// moveTree renames src to dst, falling back to copy, verify and remove when the
// two are on different filesystems. It refuses to overwrite.
func moveTree(src, dst string) error {
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("destination %s already exists", dst)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else if !isCrossDevice(err) {
		return err
	}
	want := measure(src)
	if err := copyTree(src, dst); err != nil {
		_ = os.RemoveAll(dst)
		return err
	}
	if got := measure(dst); got.bytes != want.bytes || got.files != want.files {
		_ = os.RemoveAll(dst)
		return fmt.Errorf("copy verification failed: %d bytes in %d files, want %d in %d", got.bytes, got.files, want.bytes, want.files)
	}
	return os.RemoveAll(src)
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case d.Type()&fs.ModeSymlink != 0:
			link, err := os.Readlink(p)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case d.Type().IsRegular():
			in, err := os.Open(p)
			if err != nil {
				return err
			}
			defer in.Close()
			out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, in); err != nil {
				out.Close()
				return err
			}
			return out.Close()
		}
		return nil
	})
}
