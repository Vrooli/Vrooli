package cliutil

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// freshnessWatchRacyWindow is how far before a manifest's write time an
// observed object must have last changed for its recorded metadata to be
// trusted. It covers coarse filesystem timestamp granularity (FAT records
// mtime in 2 s steps): an object changed inside the window may have changed
// again without moving its timestamps, so the watch refuses to vouch for it.
const freshnessWatchRacyWindow = 2 * time.Second

// FreshnessWatch is the proof that lets a freshness check skip re-enumerating
// its inputs. It records the metadata of every filesystem object the
// enumeration consulted: each directory whose listing it read, each file it
// read or classified, and, for each path it found missing, the nearest
// existing ancestor directory (creating the path changes that directory).
// While every watched object keeps its recorded size, mtime and (where the
// platform reports them) ctime and inode, a fresh enumeration would visit the
// same files with the same content, so the recorded digest still holds.
// Anything else falls back to the full enumeration, which decides precisely.
type FreshnessWatch struct {
	// SpecKey identifies the exact freshness spec the watch was recorded for.
	SpecKey string `json:"spec_key"`
	// ExpansionStable reports that the spec's inputs were already closed over
	// the Go replace closure (WithGoReplaceFreshnessInputs returned them
	// unchanged) and that the files and directories that derivation read are
	// watched too. A holder of the unexpanded input list may then trust the
	// expansion without redoing it.
	ExpansionStable bool                  `json:"expansion_stable,omitempty"`
	Entries         FreshnessWatchEntries `json:"entries"`
}

// FreshnessWatchEntry is one watched filesystem object. Rel is slash-separated
// and relative to the spec's context root (it may climb above it, as shared
// package inputs do).
type FreshnessWatchEntry struct {
	Rel     string
	Dir     bool
	Size    int64
	MTimeNS int64
	CTimeNS int64
	Inode   uint64
}

// FreshnessWatchEntries is decoded by every invocation of an installed CLI,
// so it is encoded as one string of lines (`jq -r .watch.entries` prints
// them), one entry per line: "<f|d> <size> <mtime_ns> <ctime_ns> <inode> <rel>".
// A single string decodes several times faster than a list of objects.
type FreshnessWatchEntries []FreshnessWatchEntry

func (entries FreshnessWatchEntries) MarshalJSON() ([]byte, error) {
	var encoded strings.Builder
	for i, entry := range entries {
		if strings.ContainsAny(entry.Rel, "\n\r") {
			return nil, fmt.Errorf("freshness watch path %q contains a line break", entry.Rel)
		}
		kind := "f"
		if entry.Dir {
			kind = "d"
		}
		if i > 0 {
			encoded.WriteByte('\n')
		}
		encoded.WriteString(kind + " " + strconv.FormatInt(entry.Size, 10) + " " + strconv.FormatInt(entry.MTimeNS, 10) + " " +
			strconv.FormatInt(entry.CTimeNS, 10) + " " + strconv.FormatUint(entry.Inode, 10) + " " + entry.Rel)
	}
	return json.Marshal(encoded.String())
}

func (entries *FreshnessWatchEntries) UnmarshalJSON(data []byte) error {
	lines, err := freshnessWatchLines(data)
	if err != nil {
		return err
	}
	decoded := make(FreshnessWatchEntries, len(lines))
	for i, line := range lines {
		fields := strings.SplitN(line, " ", 6)
		if len(fields) != 6 || (fields[0] != "f" && fields[0] != "d") || fields[5] == "" {
			return fmt.Errorf("malformed freshness watch entry %q", line)
		}
		size, sizeErr := strconv.ParseInt(fields[1], 10, 64)
		mtime, mtimeErr := strconv.ParseInt(fields[2], 10, 64)
		ctime, ctimeErr := strconv.ParseInt(fields[3], 10, 64)
		inode, inodeErr := strconv.ParseUint(fields[4], 10, 64)
		if err := errors.Join(sizeErr, mtimeErr, ctimeErr, inodeErr); err != nil {
			return fmt.Errorf("malformed freshness watch entry %q: %w", line, err)
		}
		decoded[i] = FreshnessWatchEntry{Rel: fields[5], Dir: fields[0] == "d", Size: size, MTimeNS: mtime, CTimeNS: ctime, Inode: inode}
	}
	*entries = decoded
	return nil
}

// freshnessWatchLines splits the encoded entries string. Line breaks are the
// only escapes the writer normally emits, so the raw literal is split on them
// directly; any other escape takes the general JSON path.
func freshnessWatchLines(data []byte) ([]string, error) {
	raw, quoted := bytes.CutPrefix(data, []byte(`"`))
	raw, closed := bytes.CutSuffix(raw, []byte(`"`))
	if quoted && closed && onlyLineBreakEscapes(raw) {
		if len(raw) == 0 {
			return nil, nil
		}
		return strings.Split(string(raw), `\n`), nil
	}
	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return nil, err
	}
	if encoded == "" {
		return nil, nil
	}
	return strings.Split(encoded, "\n"), nil
}

func onlyLineBreakEscapes(raw []byte) bool {
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '"':
			return false
		case '\\':
			if i+1 >= len(raw) || raw[i+1] != 'n' {
				return false
			}
			i++
		}
	}
	return true
}

// freshnessObserver records what one enumeration consulted. A nil observer
// records nothing, so the enumeration code calls it unconditionally.
type freshnessObserver struct {
	root    string
	entries map[string]FreshnessWatchEntry
	// unwatchable is set when the enumeration consulted state a watch cannot
	// represent (git's index and ignore rules, the Go toolchain's build
	// constraints); no watch is produced then.
	unwatchable bool
}

func newFreshnessObserver(spec FreshnessSpec) *freshnessObserver {
	return &freshnessObserver{root: freshnessContextRoot(spec), entries: map[string]FreshnessWatchEntry{}}
}

func freshnessContextRoot(spec FreshnessSpec) string {
	if root := cleanAbs(spec.ContextRoot); root != "" {
		return root
	}
	return cleanAbs(spec.SourceRoot)
}

// markUnwatchable records that the enumeration depended on something other
// than the filesystem metadata the watch can verify.
func (o *freshnessObserver) markUnwatchable() {
	if o != nil {
		o.unwatchable = true
	}
}

// listed records a directory whose listing the enumeration consumed.
func (o *freshnessObserver) listed(dir string) {
	o.probed(dir)
}

// read records a file whose content or classification the enumeration used,
// from the FileInfo the walk already holds.
func (o *freshnessObserver) read(path string, info fs.FileInfo) {
	if o == nil {
		return
	}
	o.record(path, info)
}

// probed records whatever is at path now; a missing path is represented by its
// nearest existing ancestor, whose listing changes when the path appears.
func (o *freshnessObserver) probed(path string) {
	if o == nil {
		return
	}
	path = filepath.Clean(path)
	for {
		info, err := os.Lstat(path)
		if err == nil {
			o.record(path, info)
			return
		}
		// Missing, or a file where a directory was expected: either way the
		// enumeration saw nothing here, and creating it changes an ancestor.
		parent := filepath.Dir(path)
		if parent == path {
			o.unwatchable = true
			return
		}
		path = parent
	}
}

// globbed records the directories filepath.Glob reads to expand pattern.
func (o *freshnessObserver) globbed(pattern string) {
	if o == nil {
		return
	}
	dir := filepath.Dir(filepath.Clean(pattern))
	if !hasGlobPattern(dir) {
		o.listed(dir)
		return
	}
	o.globbed(dir)
	matches, err := filepath.Glob(dir)
	if err != nil {
		o.unwatchable = true
		return
	}
	for _, match := range matches {
		o.probed(match)
	}
}

func (o *freshnessObserver) record(path string, info fs.FileInfo) {
	abs := filepath.Clean(path)
	if _, seen := o.entries[abs]; seen {
		return
	}
	rel, err := filepath.Rel(o.root, abs)
	if err != nil {
		o.unwatchable = true
		return
	}
	ctime, inode := statChangeToken(info)
	o.entries[abs] = FreshnessWatchEntry{
		Rel:     filepath.ToSlash(rel),
		Dir:     info.IsDir(),
		Size:    info.Size(),
		MTimeNS: info.ModTime().UnixNano(),
		CTimeNS: ctime,
		Inode:   inode,
	}
}

// watch returns the recorded proof for spec, or nil when the enumeration was
// not fully observable.
func (o *freshnessObserver) watch(spec FreshnessSpec, expansionStable bool) *FreshnessWatch {
	if o == nil || o.unwatchable || len(o.entries) == 0 {
		return nil
	}
	entries := make(FreshnessWatchEntries, 0, len(o.entries))
	for _, entry := range o.entries {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Rel < entries[j].Rel })
	return &FreshnessWatch{SpecKey: freshnessSpecKey(spec), ExpansionStable: expansionStable, Entries: entries}
}

// Holds reports whether every watched object still carries its recorded
// metadata and none of it changed within the racy window before writtenAtNS.
// It reads only metadata: one lstat per watched object, spread over a few
// workers because the cost is syscall latency. A false result proves nothing;
// callers fall back to a full enumeration.
func (w *FreshnessWatch) Holds(spec FreshnessSpec, writtenAtNS int64) bool {
	if w == nil || w.SpecKey == "" || len(w.Entries) == 0 || w.SpecKey != freshnessSpecKey(spec) {
		return false
	}
	trustedBefore := writtenAtNS - int64(freshnessWatchRacyWindow)
	for _, entry := range w.Entries {
		if entry.MTimeNS >= trustedBefore || entry.CTimeNS >= trustedBefore {
			return false
		}
	}
	root := freshnessContextRoot(spec) + string(filepath.Separator)
	workers := min(freshnessWatchWorkers, runtime.GOMAXPROCS(0), (len(w.Entries)+freshnessWatchMinPerWorker-1)/freshnessWatchMinPerWorker)
	var changed atomic.Bool
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := worker; i < len(w.Entries) && !changed.Load(); i += workers {
				if !w.Entries[i].unchanged(root) {
					changed.Store(true)
				}
			}
		}(worker)
	}
	wg.Wait()
	return !changed.Load()
}

const (
	freshnessWatchWorkers      = 8
	freshnessWatchMinPerWorker = 128
)

func (entry FreshnessWatchEntry) unchanged(root string) bool {
	info, err := os.Lstat(filepath.Clean(root + filepath.FromSlash(entry.Rel)))
	if err != nil || info.IsDir() != entry.Dir || info.Size() != entry.Size || info.ModTime().UnixNano() != entry.MTimeNS {
		return false
	}
	ctime, inode := statChangeToken(info)
	return ctime == entry.CTimeNS && inode == entry.Inode
}

// freshnessSpecKey canonically identifies everything about a spec that shapes
// its enumeration. Roots are absolute; skip lists are order-insensitive.
func freshnessSpecKey(spec FreshnessSpec) string {
	hasher := sha256.New()
	fmt.Fprintf(hasher, "freshness-spec/v1\nsource=%s\ncontext=%s\ncase-insensitive=%t\n", cleanAbs(spec.SourceRoot), freshnessContextRoot(spec), spec.CaseInsensitive)
	for _, input := range trimNonEmpty(spec.Inputs) {
		fmt.Fprintf(hasher, "input=%s\n", input)
	}
	for _, skip := range sortedUniqueNonEmpty(spec.SkipFiles) {
		fmt.Fprintf(hasher, "skip-file=%s\n", skip)
	}
	for _, suffix := range sortedUniqueNonEmpty(spec.SkipSuffixes) {
		fmt.Fprintf(hasher, "skip-suffix=%s\n", suffix)
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func sortedUniqueNonEmpty(values []string) []string {
	out := trimNonEmpty(values)
	slices.Sort(out)
	return slices.Compact(out)
}
