package filepreview

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Listing bounds. A directory is the first preview target whose cost is
// unbounded in a way a file never is: one click on node_modules must not pin a
// core or allocate hundreds of megabytes. Every limit below exists to keep a
// listing as bounded as the 1 MiB inline-text cap already is.
const (
	// DefaultListPageSize is the page size used when a client sends 0.
	DefaultListPageSize = 200
	// MaxListPageSize caps a client-requested page. Larger requests are
	// clamped rather than rejected.
	MaxListPageSize = 1000
	// MaxEntriesScanned is the ceiling on entries read from one directory.
	// Past it the scan stops and the result is flagged truncated.
	MaxEntriesScanned = 50_000
	// StatSortLimit is the largest directory that may be ordered by a
	// stat-requiring sort (size/mtime). Above it the server falls back to a
	// name sort and says so in the result.
	StatSortLimit = 5_000

	// maxChildCounts bounds how many subdirectories on one page have their
	// children counted; each count costs an extra directory read.
	maxChildCounts = 64
	// childCountCeiling is the largest child count reported exactly. A
	// subdirectory with more children reports ChildCountUnknown instead of
	// paying an unbounded read.
	childCountCeiling = 1_000
	// scanChunkSize is how many entries are pulled from the OS per syscall.
	scanChunkSize = 512
)

// ChildCountUnknown marks a subdirectory whose child count was not determined
// — unreadable, past childCountCeiling, or past the per-page count budget.
const ChildCountUnknown int64 = -1

// Sort selects the ordering of a directory page.
//
// The two name sorts are derivable from a single directory scan. The size and
// mtime sorts need metadata for every entry, so they are only honoured below
// StatSortLimit; ListResult.EffectiveSort reports what was actually applied.
type Sort string

const (
	SortDirsFirstName Sort = "dirs_first_name"
	SortName          Sort = "name"
	SortSizeDesc      Sort = "size_desc"
	SortMTimeDesc     Sort = "mtime_desc"
)

// NormalizeSort maps an empty or unrecognized sort onto the default.
func NormalizeSort(s Sort) Sort {
	switch s {
	case SortDirsFirstName, SortName, SortSizeDesc, SortMTimeDesc:
		return s
	default:
		return SortDirsFirstName
	}
}

// needsStat reports whether the sort requires metadata for every entry rather
// than just the names the directory scan already returned.
func (s Sort) needsStat() bool { return s == SortSizeDesc || s == SortMTimeDesc }

// EntryType is the filesystem category of a directory entry, taken from the
// entry itself: a symlink reports as a symlink, never as its target.
type EntryType string

const (
	EntryTypeFile      EntryType = "file"
	EntryTypeDirectory EntryType = "directory"
	EntryTypeSymlink   EntryType = "symlink"
	EntryTypeOther     EntryType = "other"
)

// DirEntryInfo is one rendered child of a listed directory.
type DirEntryInfo struct {
	Name string
	Type EntryType
	// Kind is classified from the extension alone, so building a page costs no
	// file reads. Empty means "determined when the entry is opened" — the
	// resolve that follows a click does sniff content.
	Kind            Kind
	SizeBytes       int64
	ModTimeUnixNano int64
	// CanPreview is false when the entry cannot be opened as a preview: its
	// metadata was unreadable, it is a special file, or it is a broken link.
	CanPreview bool
	// SymlinkTarget is the raw link text, set only for EntryTypeSymlink.
	SymlinkTarget string
	SymlinkBroken bool
	// Mode is the permission string (e.g. "drwxr-xr-x"). Best-effort across
	// platforms: it reports whatever the filesystem exposes.
	Mode string
	// ChildCount is a subdirectory's entry count, or ChildCountUnknown.
	ChildCount int64
}

// ListOptions parameterizes one directory page.
type ListOptions struct {
	Sort       Sort
	ShowHidden bool
	// PageSize is clamped to [1, MaxListPageSize]; 0 selects the default.
	PageSize int
	// PageToken continues a previous page. Empty starts at the beginning.
	PageToken string
}

// ListResult is one bounded page of a directory.
type ListResult struct {
	ResolvedPath string
	// ParentPath is empty at a filesystem root.
	ParentPath    string
	Entries       []DirEntryInfo
	TotalEntries  int
	Truncated     bool
	NextPageToken string
	EffectiveSort Sort
	Warnings      []string
}

// pageToken is the opaque continuation payload. DirModTimeUnixNano is the
// stale guard: offset paging over a directory that changed mid-walk would
// silently skip or duplicate entries, so a mismatch fails the page instead.
type pageToken struct {
	Sort       Sort   `json:"s"`
	ShowHidden bool   `json:"h"`
	Offset     int    `json:"o"`
	DirModTime int64  `json:"m"`
	Version    uint8  `json:"v"`
	Path       string `json:"p"`
}

// pageTokenVersion guards against decoding a token from an older encoding.
const pageTokenVersion uint8 = 1

// ListDirectory returns one bounded, sorted page of resolvedPath's entries.
// resolvedPath must come from a Target the resolver produced (in practice, via
// an opaque preview id) — this function never resolves a raw client path.
func (r *Resolver) ListDirectory(resolvedPath string, opts ListOptions) (*ListResult, error) {
	info, err := os.Stat(resolvedPath)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			return nil, newError(CodeNotFound, "Referenced directory no longer exists")
		case os.IsPermission(err):
			return nil, newError(CodeNotAllowed, "Referenced directory is not readable")
		default:
			return nil, newError(CodeInternal, "Failed to stat referenced directory")
		}
	}
	if !info.IsDir() {
		return nil, newError(CodeNotPreviewable, "Referenced path is not a directory")
	}

	requestedSort := NormalizeSort(opts.Sort)
	showHidden := opts.ShowHidden
	offset := 0

	if opts.PageToken != "" {
		tok, tokErr := decodePageToken(opts.PageToken)
		if tokErr != nil {
			return nil, tokErr
		}
		// The token owns the read it continues. A client that changes sort or
		// filter must restart at page one; failing loudly here beats stitching
		// together pages from two different orderings.
		if tok.Sort != requestedSort || tok.ShowHidden != showHidden || tok.Path != resolvedPath {
			return nil, newError(CodeInvalid, "Page token does not match the requested directory, sort, or filter")
		}
		if tok.DirModTime != info.ModTime().UnixNano() {
			return nil, newError(CodeStale, "Directory changed while paging; reload the listing")
		}
		offset = tok.Offset
	}

	pageSize := opts.PageSize
	switch {
	case pageSize <= 0:
		pageSize = DefaultListPageSize
	case pageSize > MaxListPageSize:
		pageSize = MaxListPageSize
	}

	scanned, truncated, err := scanDirectory(resolvedPath, showHidden)
	if err != nil {
		return nil, err
	}

	result := &ListResult{
		ResolvedPath:  resolvedPath,
		ParentPath:    parentPath(resolvedPath),
		TotalEntries:  len(scanned),
		Truncated:     truncated,
		EffectiveSort: requestedSort,
		Entries:       []DirEntryInfo{},
	}
	if truncated {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"Directory has more than %d entries; only the first %d are listed.",
			MaxEntriesScanned, MaxEntriesScanned))
	}

	// An expensive sort degrades visibly rather than silently: above the
	// threshold it would cost one stat per entry across the whole directory.
	if requestedSort.needsStat() && len(scanned) > StatSortLimit {
		result.EffectiveSort = SortDirsFirstName
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"Sorting by size or date needs details for every entry and is limited to %d; sorted by name instead.",
			StatSortLimit))
	}

	if result.EffectiveSort.needsStat() {
		for i := range scanned {
			scanned[i].loadInfo()
		}
	}
	sortScanned(scanned, result.EffectiveSort)

	if offset < 0 || offset > len(scanned) {
		return nil, newError(CodeInvalid, "Page token is out of range")
	}
	end := offset + pageSize
	if end > len(scanned) {
		end = len(scanned)
	}
	page := scanned[offset:end]

	result.Entries = make([]DirEntryInfo, 0, len(page))
	childCountBudget := maxChildCounts
	for i := range page {
		result.Entries = append(result.Entries, describeEntry(resolvedPath, &page[i], showHidden, &childCountBudget))
	}

	if end < len(scanned) {
		token, encErr := encodePageToken(pageToken{
			Sort:       result.EffectiveSort,
			ShowHidden: showHidden,
			Offset:     end,
			DirModTime: info.ModTime().UnixNano(),
			Version:    pageTokenVersion,
			Path:       resolvedPath,
		})
		if encErr != nil {
			return nil, newError(CodeInternal, "Failed to build the next-page token")
		}
		result.NextPageToken = token
	}

	return result, nil
}

// scannedEntry pairs a directory entry with its lazily-loaded metadata and a
// precomputed lower-case sort key (so comparisons don't allocate per call).
type scannedEntry struct {
	name    string
	lower   string
	entry   fs.DirEntry
	info    fs.FileInfo
	infoErr error
	loaded  bool
}

// loadInfo fills in the entry's metadata, at most once. On Unix this costs an
// lstat; on Windows the directory scan already carried it.
func (s *scannedEntry) loadInfo() {
	if s.loaded {
		return
	}
	s.loaded = true
	s.info, s.infoErr = s.entry.Info()
}

// isDir reports whether the entry is a directory, without following symlinks.
func (s *scannedEntry) isDir() bool { return s.entry.IsDir() }

// scanDirectory reads a directory in bounded chunks, applying the hidden
// filter as it goes so memory stays proportional to what will be shown.
func scanDirectory(dir string, showHidden bool) ([]scannedEntry, bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		if os.IsPermission(err) {
			return nil, false, newError(CodeNotAllowed, "Referenced directory is not readable")
		}
		if os.IsNotExist(err) {
			return nil, false, newError(CodeNotFound, "Referenced directory no longer exists")
		}
		return nil, false, newError(CodeInternal, "Failed to open referenced directory")
	}
	defer f.Close()

	out := make([]scannedEntry, 0, scanChunkSize)
	truncated := false
	for {
		batch, readErr := f.ReadDir(scanChunkSize)
		for _, de := range batch {
			if !showHidden && entryHidden(de) {
				continue
			}
			name := de.Name()
			out = append(out, scannedEntry{name: name, lower: strings.ToLower(name), entry: de})
		}
		if len(out) >= MaxEntriesScanned {
			out = out[:MaxEntriesScanned]
			truncated = true
			break
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return nil, false, newError(CodeInternal, "Failed to read referenced directory")
		}
		if len(batch) == 0 {
			break
		}
	}
	return out, truncated, nil
}

// sortScanned orders the scan in place. Both name orders are total and
// deterministic: case-insensitive first so listings read naturally, then
// byte-wise so two names differing only in case never swap between calls.
func sortScanned(entries []scannedEntry, s Sort) {
	switch s {
	case SortName:
		sort.SliceStable(entries, func(i, j int) bool {
			return compareNames(&entries[i], &entries[j]) < 0
		})
	case SortSizeDesc:
		sort.SliceStable(entries, func(i, j int) bool {
			si, sj := entrySize(&entries[i]), entrySize(&entries[j])
			if si != sj {
				return si > sj
			}
			return compareNames(&entries[i], &entries[j]) < 0
		})
	case SortMTimeDesc:
		sort.SliceStable(entries, func(i, j int) bool {
			mi, mj := entryModTime(&entries[i]), entryModTime(&entries[j])
			if mi != mj {
				return mi > mj
			}
			return compareNames(&entries[i], &entries[j]) < 0
		})
	default: // SortDirsFirstName
		sort.SliceStable(entries, func(i, j int) bool {
			di, dj := entries[i].isDir(), entries[j].isDir()
			if di != dj {
				return di
			}
			return compareNames(&entries[i], &entries[j]) < 0
		})
	}
}

func compareNames(a, b *scannedEntry) int {
	if a.lower != b.lower {
		return strings.Compare(a.lower, b.lower)
	}
	return strings.Compare(a.name, b.name)
}

// entrySize returns the entry's own byte size, or 0 when unreadable. Only
// called for stat-requiring sorts, where loadInfo already ran.
func entrySize(e *scannedEntry) int64 {
	if e.info == nil {
		return 0
	}
	return e.info.Size()
}

func entryModTime(e *scannedEntry) int64 {
	if e.info == nil {
		return 0
	}
	return e.info.ModTime().UnixNano()
}

// describeEntry builds the rendered form of one page entry. This is the only
// place that pays per-entry syscalls, and it runs for the page alone —
// never for the whole directory.
func describeEntry(dir string, e *scannedEntry, showHidden bool, childCountBudget *int) DirEntryInfo {
	e.loadInfo()

	out := DirEntryInfo{
		Name:       e.name,
		Type:       entryTypeOf(e.entry),
		ChildCount: ChildCountUnknown,
	}
	if e.infoErr != nil || e.info == nil {
		// The entry vanished or its metadata is unreadable between the scan
		// and now. Show the name; refuse to pretend it can be opened.
		out.CanPreview = false
		return out
	}

	out.SizeBytes = e.info.Size()
	out.ModTimeUnixNano = e.info.ModTime().UnixNano()
	out.Mode = e.info.Mode().String()

	full := filepath.Join(dir, e.name)

	switch out.Type {
	case EntryTypeDirectory:
		out.Kind = KindDirectory
		out.CanPreview = true
		if *childCountBudget > 0 {
			*childCountBudget--
			out.ChildCount = countChildren(full, showHidden)
		}

	case EntryTypeSymlink:
		if target, err := os.Readlink(full); err == nil {
			out.SymlinkTarget = target
		}
		// One stat decides both whether the link resolves and, when it points
		// at a directory, which renderer a click should reach.
		st, err := os.Stat(full)
		if err != nil {
			out.SymlinkBroken = true
			out.CanPreview = false
			break
		}
		out.CanPreview = true
		if st.IsDir() {
			out.Kind = KindDirectory
		} else if k, ok := classifyByExtension(e.name); ok {
			out.Kind = k
		}

	case EntryTypeFile:
		out.CanPreview = true
		if k, ok := classifyByExtension(e.name); ok {
			out.Kind = k
		}

	default: // EntryTypeOther — sockets, FIFOs, device nodes.
		// Never previewable: opening a FIFO blocks until a writer appears.
		out.CanPreview = false
	}

	return out
}

// entryTypeOf categorizes an entry from its own mode bits, without following
// symlinks. Reliable on every platform Go supports: when the OS cannot report
// a type from the directory scan alone, the runtime fills it in.
func entryTypeOf(de fs.DirEntry) EntryType {
	mode := de.Type()
	switch {
	case mode&fs.ModeSymlink != 0:
		return EntryTypeSymlink
	case mode.IsDir():
		return EntryTypeDirectory
	case mode.IsRegular():
		return EntryTypeFile
	default:
		return EntryTypeOther
	}
}

// countChildren returns how many entries a subdirectory holds, honouring the
// same hidden filter as the parent listing so the number matches what a click
// would show. Returns ChildCountUnknown when unreadable or past the ceiling.
func countChildren(dir string, showHidden bool) int64 {
	f, err := os.Open(dir)
	if err != nil {
		return ChildCountUnknown
	}
	defer f.Close()

	var n int64
	for {
		batch, readErr := f.ReadDir(scanChunkSize)
		for _, de := range batch {
			if !showHidden && entryHidden(de) {
				continue
			}
			n++
		}
		if n > childCountCeiling {
			return ChildCountUnknown
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return n
			}
			return ChildCountUnknown
		}
		if len(batch) == 0 {
			return n
		}
	}
}

// parentPath returns the containing directory, or "" when p is already a
// filesystem root. filepath.Dir is its own fixed point at a root on every
// platform, including Windows drive letters and UNC shares.
func parentPath(p string) string {
	parent := filepath.Dir(p)
	if parent == p {
		return ""
	}
	return parent
}

func encodePageToken(t pageToken) (string, error) {
	raw, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodePageToken(s string) (pageToken, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return pageToken{}, newError(CodeInvalid, "Page token is malformed")
	}
	var t pageToken
	if err := json.Unmarshal(raw, &t); err != nil {
		return pageToken{}, newError(CodeInvalid, "Page token is malformed")
	}
	if t.Version != pageTokenVersion {
		return pageToken{}, newError(CodeInvalid, "Page token is from an unsupported version; reload the listing")
	}
	return t, nil
}
