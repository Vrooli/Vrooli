package files

import (
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
)

// Entry is one indexable file found by a walk (stat only, no content read).
type Entry struct {
	Path      string // corpus-relative, slash-separated
	Size      int64
	MtimeNS   int64
	ChangedNS int64 // inode change time: moves with any content write, even when mtime is restored
	Kind      Kind
	Topic     string
	Feed      string
	MaxBytes  int64
}

// Truncated reports whether only part of the file is read and indexed.
func (e Entry) Truncated() bool { return e.Size > e.MaxBytes }

// WalkStats counts what a walk left out or cut short, so coverage is visible.
type WalkStats struct {
	Private      int // files matching a private glob (operator conversations, personal data)
	Excluded     int // files matching an exclude glob
	Summarised   int // files in run-depth, dataset or code folders that only index summaries
	OversizeJSON int // .json files over max_json_kb
	Truncated    int // indexed files longer than max_file_kb (head, or tail for logs, kept)
}

// WalkResult is one walk: the indexable files, what was left out, and the
// problems met (unreadable folders, malformed feeds.json).
type WalkResult struct {
	Entries  []Entry
	Stats    WalkStats
	Problems []string
}

// Walk lists the indexable files under root (or only topics/<topic>/ when
// topic is set). Nothing in the corpus can make it fail: an unreadable
// folder is skipped and reported, a malformed root feeds.json falls back to
// the defaults and a malformed topic feeds.json indexes only that topic's
// summary files (never the folders it may have excluded), each reported.
func Walk(root, topic string) WalkResult {
	w := walker{root: root, topics: map[string]Policy{}, broken: map[string]bool{}}
	rootPolicy, err := loadPolicy(root, DefaultPolicy())
	if err != nil {
		w.problems = append(w.problems, err.Error()+" (using defaults)")
	}
	w.rootPolicy = rootPolicy
	start := ""
	if topic != "" {
		start = path.Join("topics", topic)
	}
	w.dir(start, false)
	w.collect()
	sort.Slice(w.entries, func(i, j int) bool { return w.entries[i].Path < w.entries[j].Path })
	sort.Strings(w.problems)
	w.problems = slices.Compact(w.problems) // forked walkers report a broken topic policy once each
	return WalkResult{Entries: w.entries, Stats: w.stats, Problems: w.problems}
}

func (s *WalkStats) add(o WalkStats) {
	s.Private += o.Private
	s.Excluded += o.Excluded
	s.Summarised += o.Summarised
	s.OversizeJSON += o.OversizeJSON
	s.Truncated += o.Truncated
}

// collect waits for forked sub-walkers and merges their results.
func (w *walker) collect() {
	w.wg.Wait()
	for _, c := range w.children {
		c.collect()
		w.entries = append(w.entries, c.entries...)
		w.problems = append(w.problems, c.problems...)
		w.stats.add(c.stats)
	}
}

// forkPoint reports whether rel is walked in its own goroutine: each
// top-level folder, each topic and each run. Sub-walkers share no mutable
// state; a forked topic or run reloads its (read-only) policy.
func forkPoint(rel string) bool {
	parts := strings.Split(rel, "/")
	switch {
	case len(parts) == 1:
		return rel != "topics" && rel != ""
	case parts[0] != "topics":
		return false
	case len(parts) == 2:
		return true
	default:
		return len(parts) == 4 && parts[2] == "runs"
	}
}

// fork walks rel in its own goroutine when it is a fork point, which keeps
// a large corpus walk short.
func (w *walker) fork(rel string, inherited bool) bool {
	if !forkPoint(rel) {
		return false
	}
	c := &walker{root: w.root, rootPolicy: w.rootPolicy, topics: map[string]Policy{}, broken: map[string]bool{}, parent: w}
	w.children = append(w.children, c)
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		c.dir(rel, inherited)
	}()
	return true
}

type walker struct {
	root       string
	rootPolicy Policy
	topics     map[string]Policy
	broken     map[string]bool // topics whose feeds.json is malformed
	entries    []Entry
	stats      WalkStats
	problems   []string
	parent     *walker // set on forked sub-walkers
	children   []*walker
	wg         sync.WaitGroup
}

// topicOf splits a corpus path into its topic and the topic-relative rest.
func topicOf(rel string) (string, string) {
	parts := strings.SplitN(rel, "/", 3)
	if len(parts) >= 2 && parts[0] == "topics" {
		if len(parts) == 2 {
			return parts[1], ""
		}
		return parts[1], parts[2]
	}
	return "", rel
}

func (w *walker) policyFor(topic string) Policy {
	if topic == "" {
		return w.rootPolicy
	}
	if p, ok := w.topics[topic]; ok {
		return p
	}
	p, err := loadPolicy(filepath.Join(w.root, "topics", topic), w.rootPolicy)
	if err != nil {
		w.problems = append(w.problems, err.Error()+" (indexing only this topic's README/INDEX/SUMMARY files until it is fixed)")
		p = w.rootPolicy.limits()
		w.broken[topic] = true
	}
	w.topics[topic] = p
	return p
}

func (w *walker) excluded(rel string) bool {
	if anyMatch(w.rootPolicy.Exclude, rel) {
		return true
	}
	topic, inner := topicOf(rel)
	return topic != "" && inner != "" && anyMatch(w.policyFor(topic).Exclude, inner)
}

// private reports whether rel matches the private class in force: the
// topic's own list when its feeds.json sets one, else the corpus list.
func (w *walker) private(rel string) bool {
	topic, inner := topicOf(rel)
	if topic != "" && inner != "" {
		if p := w.policyFor(topic).Private; p != nil {
			return anyMatch(*p, inner)
		}
	}
	globs := DefaultPrivate
	if w.rootPolicy.Private != nil {
		globs = *w.rootPolicy.Private
	}
	return anyMatch(globs, rel)
}

func (w *walker) included(rel string) bool {
	if anyMatch(w.rootPolicy.Include, rel) {
		return true
	}
	topic, inner := topicOf(rel)
	return topic != "" && inner != "" && anyMatch(w.policyFor(topic).Include, inner)
}

func (w *walker) mayInclude(rel string) bool {
	for _, g := range w.rootPolicy.Include {
		if mayContain(g, rel) {
			return true
		}
	}
	topic, inner := topicOf(rel)
	if topic == "" || inner == "" {
		return false
	}
	for _, g := range w.policyFor(topic).Include {
		if mayContain(g, inner) {
			return true
		}
	}
	return false
}

// runDepth is the number of folders below runs/<run>/ that rel sits at, or
// -1 when rel is not inside a run.
func runDepth(rel string) int {
	parts := strings.Split(rel, "/")
	for i, part := range parts {
		if part == "runs" && i+1 < len(parts) {
			return len(parts) - (i + 2)
		}
	}
	return -1
}

// dir indexes one folder. inherited is true below a run-depth or dataset
// folder: such subtrees are still entered, but only their README, INDEX
// and SUMMARY files (and Include matches) are indexed. Code trees and
// pruned folders are not entered at all.
func (w *walker) dir(rel string, inherited bool) {
	items, err := os.ReadDir(filepath.Join(w.root, filepath.FromSlash(rel)))
	if err != nil {
		w.problems = append(w.problems, "unreadable folder: "+rel)
		return
	}
	topic, _ := topicOf(rel)
	policy := w.policyFor(topic)
	files, dirs := sortEntries(items)
	code := rel != "" && isCodeTree(items)
	summarised := inherited || code || w.summarises(rel, topic, policy, len(files))
	for _, item := range files {
		child := path.Join(rel, item.Name())
		if !w.skipped(child, summarised) {
			w.add(item, child, topic, policy)
		}
	}
	for _, name := range dirs {
		child := path.Join(rel, name)
		if w.enters(child, code) && !w.fork(child, summarised) {
			w.dir(child, summarised)
		}
	}
}

// summarises reports whether a folder indexes only its summary files: its
// topic's feeds.json is broken, it holds a generated dataset, or it is deeper
// than the run depth.
func (w *walker) summarises(rel, topic string, policy Policy, files int) bool {
	return w.broken[topic] || files > policy.DatasetFiles || runDepth(rel) >= policy.RunDepth
}

// enters reports whether the walk descends into a folder: never into an
// excluded one, and into a code tree's subfolders only where an include rule
// could match.
func (w *walker) enters(child string, code bool) bool {
	return !w.excluded(child) && (!code || w.mayInclude(child))
}

// isCodeTree reports a checked-out source tree: code is Code Facts' corpus.
func isCodeTree(items []os.DirEntry) bool {
	return slices.ContainsFunc(items, func(item os.DirEntry) bool { return slices.Contains(codeMarkers, item.Name()) })
}

// sortEntries splits a folder into indexable files and enterable folders.
// Links are never followed out of the corpus.
func sortEntries(items []os.DirEntry) (files []os.DirEntry, dirs []string) {
	for _, item := range items {
		name := item.Name()
		switch {
		case item.Type()&os.ModeSymlink != 0:
		case item.IsDir():
			if !prunedDirs[name] && !strings.HasPrefix(name, ".") {
				dirs = append(dirs, name)
			}
		case item.Type().IsRegular() && name != PolicyFile && kindOf(name) != "":
			files = append(files, item)
		}
	}
	return files, dirs
}

func kindOf(name string) Kind { return kindByExt[strings.ToLower(filepath.Ext(name))] }

// skipped counts and reports a file that policy keeps out of the index.
func (w *walker) skipped(child string, summarised bool) bool {
	switch {
	case w.private(child):
		w.stats.Private++
	case w.excluded(child):
		w.stats.Excluded++
	case summarised && !summaryFile(path.Base(child)) && !w.included(child):
		w.stats.Summarised++
	default:
		return false
	}
	return true
}

// add records one indexable file, counting oversize JSON (skipped) and files
// that will be cut short.
func (w *walker) add(item os.DirEntry, child, topic string, policy Policy) {
	info, err := item.Info()
	if err != nil {
		return
	}
	kind := kindOf(item.Name())
	if kind == KindJSON && info.Size() > int64(policy.MaxJSONKB)*1024 {
		w.stats.OversizeJSON++
		return
	}
	e := Entry{
		Path: child, Size: info.Size(), MtimeNS: info.ModTime().UnixNano(), ChangedNS: changeTime(info),
		Kind: kind, Topic: topic, Feed: w.feedOf(child), MaxBytes: int64(policy.MaxFileKB) * 1024,
	}
	if e.Truncated() {
		w.stats.Truncated++
	}
	w.entries = append(w.entries, e)
}

// feedOf names the feed a file belongs to: a topic feeds.json feed if one
// matches, else the file's top folder inside its topic ("(root)" for files
// directly in the topic or corpus root).
func (w *walker) feedOf(rel string) string {
	topic, inner := topicOf(rel)
	if topic == "" {
		inner = rel
	} else {
		feeds := w.policyFor(topic).Feeds
		names := make([]string, 0, len(feeds))
		for name := range feeds {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if anyMatch(feeds[name], inner) {
				return name
			}
		}
	}
	if i := strings.Index(inner, "/"); i > 0 {
		return inner[:i]
	}
	return "(root)"
}
