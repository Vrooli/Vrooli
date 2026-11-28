package home

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// What changed since the operator last looked (SYSTEM.md §8).

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

// LedgerAddition is text appended to a ledger since the last look, or,
// with no look, the sections under its headings dated after the time
// compared with (FromDatedHeadings).
type LedgerAddition struct {
	Path              string
	Text              string
	Truncated         bool
	FromDatedHeadings bool
}

// StatusChange is one workstream whose status moved.
type StatusChange struct {
	Topic, Workstream, Before, After string
}

// Changes is the difference between a look, or a given time, and the present.
type Changes struct {
	// Since is the time compared with; zero when there is none.
	Since time.Time
	// Baseline says what Since is, in words; BaselineKind is
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
	// could not be read: changes there are unknown, not none.
	Unreadable []Evidence
	// NotShown says, in plain words, what this comparison cannot show.
	NotShown []string
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
// time is given.
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
			return scope{}, WithCommand(fmt.Errorf("%w: no commission %q", ErrNotFound, id), "research-desk commissions")
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
			return scope{}, WithCommand(fmt.Errorf("%w: no topic %q", ErrNotFound, id), "research-desk topics")
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
// look, else with 24 hours ago. A time gives files, checkpoints and the
// ledger sections under headings dated after it; workstream status changes
// need the statuses a look records, and NotShown says so.
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
		out := Changes{Since: since, BaselineKind: kind, Baseline: baseline}
		out.NotShown = append(out.NotShown, "Workstream status changes need a recorded look; mark this seen to start one.")
		for _, path := range sc.ledgers {
			out.datedLedger(path, since)
		}
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
		out.addLedger(add)
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

// addLedger keeps a ledger addition and its showcase lines.
func (c *Changes) addLedger(add LedgerAddition) {
	c.Ledger = append(c.Ledger, add)
	for _, line := range strings.Split(add.Text, "\n") {
		if strings.Contains(strings.ToLower(line), "showcase") {
			c.ShowcaseLines = append(c.ShowcaseLines, strings.TrimSpace(line))
		}
	}
}

// ledgerDate is a date in a ledger heading, with an optional time and zone
// (UTC when none is given).
var ledgerDate = regexp.MustCompile(`\b(20\d\d-\d\d-\d\d)(?:[T ](\d\d:\d\d)(:\d\d(?:\.\d+)?)?(Z|[+-]\d\d:\d\d)?)?`)

// datedLedger adds, with no recorded look, the sections of a ledger under
// headings dated after since: a heading with only a date counts when that
// day reaches past since. A ledger without dated headings is named in
// NotShown, since its new text cannot be told from its old.
func (c *Changes) datedLedger(path string, since time.Time) {
	data, err := os.ReadFile(path)
	switch {
	case absent(err):
		return
	case err != nil:
		c.Unreadable = append(c.Unreadable, Unreadable(path, err))
		return
	}
	var (
		kept    []string
		dated   int
		level   int // the open section's heading level; 0 when none is kept
		include bool
	)
	for _, line := range strings.Split(strings.ToValidUTF8(string(data), ""), "\n") {
		if n := headingLevel(line); n > 0 {
			at, ok := headingTime(line)
			switch {
			case ok:
				dated++
				level, include = n, laterThan(at, since)
			case level > 0 && n <= level:
				level, include = 0, false
			}
		}
		if include {
			kept = append(kept, line)
		}
	}
	if dated == 0 {
		c.NotShown = append(c.NotShown, path+" has no dated headings, so what was added to it cannot be told apart; mark this seen to start a look.")
		return
	}
	if len(kept) == 0 {
		return
	}
	add := LedgerAddition{Path: path, Text: strings.Join(kept, "\n"), FromDatedHeadings: true}
	if len(add.Text) > ledgerTextCap {
		add.Text, add.Truncated = strings.ToValidUTF8(add.Text[:ledgerTextCap], ""), true
	}
	c.addLedger(add)
}

// datedAt is a heading's time; Day is set when it gave a date only.
type datedAt struct {
	At  time.Time
	Day bool
}

func headingLevel(line string) int {
	n := 0
	for n < len(line) && n < 6 && line[n] == '#' {
		n++
	}
	if n == 0 || n >= len(line) || line[n] != ' ' {
		return 0
	}
	return n
}

func headingTime(line string) (datedAt, bool) {
	m := ledgerDate.FindStringSubmatch(line)
	if m == nil {
		return datedAt{}, false
	}
	if m[2] == "" {
		day, err := time.Parse(time.DateOnly, m[1])
		return datedAt{At: day, Day: true}, err == nil
	}
	sec, zone := m[3], m[4]
	if sec == "" {
		sec = ":00"
	}
	if zone == "" {
		zone = "Z"
	}
	at, err := time.Parse(time.RFC3339, m[1]+"T"+m[2]+sec+zone)
	return datedAt{At: at}, err == nil
}

func laterThan(d datedAt, since time.Time) bool {
	if d.Day {
		return d.At.Add(24 * time.Hour).After(since)
	}
	return d.At.After(since)
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
