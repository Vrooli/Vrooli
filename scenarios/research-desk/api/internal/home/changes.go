package home

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// What changed since the operator last looked (SYSTEM.md §8, rulings B2).

// Look is the snapshot recorded when the operator looks at a commission or
// topic. Deleting it only resets "since".
type Look struct {
	At time.Time `json:"at"`
	// Ledgers maps a ledger path to its byte length.
	Ledgers map[string]int64 `json:"ledgers"`
	// Checkpoints maps a checkpoint path to its modification time.
	Checkpoints map[string]time.Time `json:"checkpoints"`
	// Statuses maps topic, then workstream ID, to its status.
	Statuses map[string]map[string]string `json:"statuses"`
}

// LedgerAddition is text appended to a ledger since the last look.
type LedgerAddition struct {
	Path      string
	Text      string
	Truncated bool
}

// StatusChange is one workstream whose status moved.
type StatusChange struct {
	Topic, Workstream, Before, After string
}

// Changes is the difference between a look, or a given time, and the present.
type Changes struct {
	// Since is the time compared with; zero when there is none.
	Since time.Time
	// Baseline says what Since is, in words (rulings F10); BaselineKind is
	// the same as a code.
	Baseline      string
	BaselineKind  BaselineKind
	Ledger        []LedgerAddition
	Checkpoints   []FileFact
	ShowcaseLines []string
	StatusChanges []StatusChange
	Files         []FileFact
	// Truncated is set when the file walk stopped at its cap.
	Truncated bool
	// Unreadable lists ledgers and folders under the subject that exist but
	// could not be read (rulings E1): changes there are unknown, not none.
	Unreadable []Evidence
}

// BaselineKind is what Changes.Since is.
type BaselineKind string

// Baselines.
const (
	BaselineLook    BaselineKind = "look"
	BaselineAsked   BaselineKind = "asked"
	BaselineLastDay BaselineKind = "last_day"
)

// DefaultWindow is how far back changes go when no look is recorded and no
// time is given (rulings P7).
const DefaultWindow = 24 * time.Hour

// Look subjects.
const (
	LookCommission = "commission"
	LookTopic      = "topic"
)

// Change limits.
const (
	ledgerTextCap = 16 << 10
	walkCap       = 20000
	filesCap      = 200
)

// LookName is the look snapshot's file name in the desk state folder.
func LookName(subject, id string) string { return "looks/" + subject + "-" + id + ".json" }

// scope is what a subject covers.
type scope struct {
	ledgers     []string
	checkpoints []string
	topics      []string
	roots       []string
}

func (s *Snapshot) scope(subject, id string) (scope, error) {
	switch subject {
	case LookCommission:
		c := s.Commission(id)
		if c == nil {
			return scope{}, fmt.Errorf("%w: no commission %q; list them with `research-desk commissions`", ErrNotFound, id)
		}
		sc := scope{ledgers: []string{filepath.Join(c.Dir, LedgerFile)}, roots: []string{c.Dir}}
		for _, run := range c.Runs {
			sc.roots = append(sc.roots, run.Path)
			sc.checkpoints = append(sc.checkpoints, filepath.Join(run.Path, "CHECKPOINT.md"))
		}
		for _, ref := range c.Topics {
			sc.topics = append(sc.topics, ref.Topic)
		}
		return sc, nil
	case LookTopic:
		t := s.Topic(id)
		if t == nil {
			return scope{}, fmt.Errorf("%w: no topic %q; list them with `research-desk topics`", ErrNotFound, id)
		}
		checkpoints, _ := filepath.Glob(filepath.Join(t.Dir, "runs", "*", "CHECKPOINT.md"))
		return scope{ledgers: []string{filepath.Join(t.Dir, LedgerFile)}, checkpoints: checkpoints, topics: []string{t.Slug}, roots: []string{t.Dir}}, nil
	}
	return scope{}, fmt.Errorf("%w: subject must be %s or %s", ErrNotFound, LookCommission, LookTopic)
}

// Look records the present state of a commission or topic. Workstream
// statuses need a snapshot taken with MissionCheck.
func (s *Snapshot) Look(subject, id string) (Look, error) {
	sc, err := s.scope(subject, id)
	if err != nil {
		return Look{}, err
	}
	look := Look{At: s.Now, Ledgers: map[string]int64{}, Checkpoints: map[string]time.Time{}, Statuses: map[string]map[string]string{}}
	for _, path := range sc.ledgers {
		if fact, ok := stat(path); ok {
			look.Ledgers[path] = fact.Bytes
		}
	}
	for _, path := range sc.checkpoints {
		if fact, ok := stat(path); ok {
			look.Checkpoints[path] = fact.Modified
		}
	}
	for _, slug := range sc.topics {
		t := s.Topic(slug)
		if t == nil || !t.Check.Known {
			continue
		}
		statuses := map[string]string{}
		for _, w := range t.Check.Workstreams {
			statuses[w.ID] = w.Status
		}
		look.Statuses[slug] = statuses
	}
	return look, nil
}

// Changes compares the present with a given time, else with the previous
// look, else with 24 hours ago. A time gives files and checkpoints only:
// ledger text and workstream statuses need the sizes and statuses a look
// records.
func (s *Snapshot) Changes(prev *Look, since time.Time, subject, id string) (Changes, error) {
	now, err := s.Look(subject, id)
	if err != nil {
		return Changes{}, err
	}
	sc, _ := s.scope(subject, id)
	kind, baseline := BaselineAsked, "since "+since.UTC().Format(time.RFC3339)+" as asked"
	if since.IsZero() && prev == nil {
		since = s.Now.Add(-DefaultWindow)
		kind, baseline = BaselineLastDay, "no look recorded; the last 24 hours, since "+since.UTC().Format(time.RFC3339)
	}
	switch {
	case !since.IsZero():
		out := Changes{Since: since, BaselineKind: kind, Baseline: baseline + "; ledger text and workstream status changes need a recorded look"}
		for _, path := range sc.checkpoints {
			if mod, ok := now.Checkpoints[path]; ok && mod.After(since) {
				if fact, ok := stat(path); ok {
					out.Checkpoints = append(out.Checkpoints, fact)
				}
			}
		}
		out.Files, out.Truncated, out.Unreadable = modifiedSince(sc.roots, since)
		return out, nil
	}
	out := Changes{Since: prev.At, BaselineKind: BaselineLook, Baseline: "the look recorded at " + prev.At.UTC().Format(time.RFC3339)}
	for _, path := range sc.ledgers {
		size, ok := now.Ledgers[path]
		before := prev.Ledgers[path]
		if !ok || size == before {
			continue
		}
		if size < before {
			before = 0 // rewritten: report it whole
		}
		add, err := readRange(path, before, size)
		if err != nil {
			out.Unreadable = append(out.Unreadable, Unreadable(path, err))
			continue
		}
		out.Ledger = append(out.Ledger, add)
		for _, line := range strings.Split(add.Text, "\n") {
			if strings.Contains(strings.ToLower(line), "showcase") {
				out.ShowcaseLines = append(out.ShowcaseLines, strings.TrimSpace(line))
			}
		}
	}
	for _, path := range sc.checkpoints {
		mod, ok := now.Checkpoints[path]
		if old, had := prev.Checkpoints[path]; ok && (!had || mod.After(old)) {
			if fact, ok := stat(path); ok {
				out.Checkpoints = append(out.Checkpoints, fact)
			}
		}
	}
	for topic, statuses := range now.Statuses {
		before, known := prev.Statuses[topic]
		if !known {
			continue
		}
		for ws, status := range statuses {
			if old := before[ws]; old != status {
				out.StatusChanges = append(out.StatusChanges, StatusChange{Topic: topic, Workstream: ws, Before: old, After: status})
			}
		}
	}
	sort.Slice(out.StatusChanges, func(i, j int) bool {
		a, b := out.StatusChanges[i], out.StatusChanges[j]
		return a.Topic+"/"+a.Workstream < b.Topic+"/"+b.Workstream
	})
	files, truncated, unreadable := modifiedSince(sc.roots, prev.At)
	out.Files, out.Truncated, out.Unreadable = files, truncated, append(out.Unreadable, unreadable...)
	return out, nil
}

func readRange(path string, from, to int64) (LedgerAddition, error) {
	add := LedgerAddition{Path: path}
	if to-from > ledgerTextCap {
		to, add.Truncated = from+ledgerTextCap, true
	}
	f, err := os.Open(path)
	if err != nil {
		return add, err
	}
	defer f.Close()
	buf := make([]byte, to-from)
	n, err := f.ReadAt(buf, from)
	if err != nil && !errors.Is(err, io.EOF) {
		return add, err
	}
	add.Text = strings.ToValidUTF8(string(buf[:n]), "")
	return add, nil
}

// modifiedSince lists files under roots modified after t, newest first, and
// the folders it could not read. A missing root has no files.
func modifiedSince(roots []string, t time.Time) ([]FileFact, bool, []Evidence) {
	var (
		out        []FileFact
		unreadable []Evidence
		visited    int
		capped     bool
		seen       = map[string]bool{}
	)
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if !(path == root && absent(err)) {
					unreadable = append(unreadable, Unreadable(path, err))
				}
				return nil
			}
			if visited++; visited > walkCap {
				capped = true
				return io.EOF
			}
			if strings.HasPrefix(d.Name(), ".") && path != root {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() || seen[path] {
				return nil
			}
			seen[path] = true
			info, err := d.Info()
			if err == nil && info.Mode().IsRegular() && info.ModTime().After(t) {
				out = append(out, FileFact{Name: d.Name(), Path: path, Bytes: info.Size(), Modified: info.ModTime()})
			}
			return nil
		})
		if capped {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Modified.After(out[j].Modified) })
	if len(out) > filesCap {
		out, capped = out[:filesCap], true
	}
	return out, capped, unreadable
}
