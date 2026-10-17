package goalhome

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Goal-home file names (large-effort-orchestration §3).
const (
	GoalFile        = "GOAL.md"
	QueueFile       = "QUEUE.md"
	FeedbackFile    = "FEEDBACK.md"
	WorkaroundsFile = "WORKAROUNDS.md"
	EpochsDir       = "epochs"
)

// Home is one parsed goal home. A missing file reads as empty and is listed
// in Missing.
type Home struct {
	Goal        Goal
	Queue       Queue
	Feedback    Feedback
	Workarounds Workarounds
	Epochs      []EpochFile
	Missing     []string
	Warnings    []Finding
	// QueueData is QUEUE.md exactly as read.
	QueueData []byte
}

// EpochFile is one parsed epochs/E<n>.md.
type EpochFile struct {
	Name   string
	Number int
	Epoch  *Epoch
}

// Goal is GOAL.md. Destination, Current and DestinationSet come from
// "- Destination: <metric> <op> <n>", "- Current: <n> @ <ISO date>" and
// "- Destination set: <ISO time>"; nil or zero when absent.
type Goal struct {
	Bytes          int
	Fields         map[string]string
	Destination    *Destination
	Current        *Measurement
	DestinationSet time.Time
}

// Destination is the goal's measured target, such as "journeys >= 24".
type Destination struct {
	Metric string  `json:"metric"`
	Op     string  `json:"op"`
	Target float64 `json:"target"`
}

// Measurement is the destination metric's latest reading.
type Measurement struct {
	Value float64   `json:"value"`
	At    time.Time `json:"at,omitempty"`
}

// Queue is QUEUE.md.
type Queue struct {
	Bytes int
	// Handoffs are all level-2 or level-3 headings that name a handoff.
	// A goal home has exactly one, "## Handoff".
	Handoffs        []HandoffSection
	NeedsOperator   []Item
	Next            []Item
	Censuses        []Census
	WaitingOnOthers []Item
}

// HandoffSection is one heading that names a handoff, with its body.
type HandoffSection struct {
	Title string `json:"title"`
	Level int    `json:"level"`
	Line  int    `json:"line"`
	Body  string `json:"-"`
}

// Census is one "- <ISO time> | <scope> | admissible=<slice>|none" line under
// ## Censuses.
type Census struct {
	At         time.Time `json:"at"`
	Scope      string    `json:"scope"`
	Admissible string    `json:"admissible"`
	Line       int       `json:"line"`
}

// Empty is true when the census found no admissible slice.
func (c Census) Empty() bool {
	value := strings.ToLower(strings.TrimSpace(c.Admissible))
	return value == "" || value == "none" || value == "-"
}

// Feedback is FEEDBACK.md. Entries are the "### <ID> — <title> (<date>, <source>)"
// headings before "## Resolved"; OpenBytes is the size of that open part.
type Feedback struct {
	OpenBytes int
	Entries   []FeedbackEntry
}

// FeedbackEntry is one open-part entry and its "Status:" line.
type FeedbackEntry struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Source string `json:"source,omitempty"`
	Status string `json:"status,omitempty"`
	Line   int    `json:"line"`
}

// Open is true when the entry's Status line says open.
func (f FeedbackEntry) Open() bool {
	return strings.HasPrefix(strings.ToLower(f.Status), "open")
}

// Steering is true for feedback the orchestrator must act on: entries from
// the operator or the supervisor (including supervisor audits), and entries
// that name no source.
func (f FeedbackEntry) Steering() bool {
	source := strings.ToLower(f.Source)
	return source == "" || strings.Contains(source, "operator") || strings.Contains(source, "supervisor")
}

// Workarounds is WORKAROUNDS.md. An entry is open when its last " · " field
// starts with "open"; entries after a "## Resolved" heading are closed.
type Workarounds struct {
	OpenBytes int
	Entries   []WorkaroundEntry
}

// WorkaroundEntry is one top-level list entry.
type WorkaroundEntry struct {
	Line  int  `json:"line"`
	Bytes int  `json:"bytes"`
	Open  bool `json:"open"`
}

var (
	epochNamePattern   = regexp.MustCompile(`^E(\d+)\.md$`)
	handoffWord        = regexp.MustCompile(`(?i)\bhandoff\b`)
	destinationPattern = regexp.MustCompile(`^(.+?)\s*(<=|>=|≤|≥|<|>|=)\s*([+\-−]?\d[\d,_]*(?:\.\d+)?)`)
	feedbackSource     = regexp.MustCompile(`\(([^()]*)\)\s*$`)
)

// Load reads a goal home from fsys, whose root is the goal-home directory.
// Only read failures other than a missing file are errors.
func Load(fsys fs.FS) (*Home, error) {
	home := &Home{}
	read := func(name string) ([]byte, error) {
		data, err := fs.ReadFile(fsys, name)
		if errors.Is(err, fs.ErrNotExist) {
			home.Missing = append(home.Missing, name)
			return nil, nil
		}
		return data, err
	}
	goal, err := read(GoalFile)
	if err != nil {
		return nil, err
	}
	home.Goal = ParseGoal(goal)
	if home.QueueData, err = read(QueueFile); err != nil {
		return nil, err
	}
	home.Queue = ParseQueue(home.QueueData)
	feedback, err := read(FeedbackFile)
	if err != nil {
		return nil, err
	}
	home.Feedback = ParseFeedback(feedback)
	workarounds, err := read(WorkaroundsFile)
	if err != nil {
		return nil, err
	}
	home.Workarounds = ParseWorkarounds(workarounds)
	entries, err := fs.ReadDir(fsys, EpochsDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	for _, entry := range entries {
		m := epochNamePattern.FindStringSubmatch(entry.Name())
		if m == nil || entry.IsDir() {
			continue
		}
		data, err := fs.ReadFile(fsys, path.Join(EpochsDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		epoch, err := ParseEpoch(bytes.NewReader(data))
		if err != nil {
			home.Warnings = append(home.Warnings, Finding{Code: "epoch-unreadable", File: path.Join(EpochsDir, entry.Name()), Detail: err.Error()})
			continue
		}
		number, _ := strconv.Atoi(m[1])
		home.Epochs = append(home.Epochs, EpochFile{Name: entry.Name(), Number: number, Epoch: epoch})
	}
	sort.Slice(home.Epochs, func(i, j int) bool { return home.Epochs[i].Number < home.Epochs[j].Number })
	return home, nil
}

// EpochHome returns the goal home of an epoch file at <home>/epochs/E<n>.md.
func EpochHome(epochPath string) (string, bool) {
	dir := filepath.Dir(epochPath)
	if filepath.Base(dir) != EpochsDir {
		return "", false
	}
	return filepath.Dir(dir), true
}

// ParseGoal reads GOAL.md.
func ParseGoal(data []byte) Goal {
	doc := parseDocument(data)
	goal := Goal{Bytes: len(data), Fields: doc.fields()}
	if m := destinationPattern.FindStringSubmatch(goal.Fields["destination"]); m != nil {
		target, _, _ := ParseNumber(m[3])
		goal.Destination = &Destination{Metric: strings.TrimSpace(m[1]), Op: m[2], Target: target}
	}
	if raw := goal.Fields["current"]; raw != "" {
		if value, rest, ok := ParseNumber(raw); ok {
			current := &Measurement{Value: value}
			if _, when, found := strings.Cut(rest, "@"); found {
				if fields := strings.Fields(when); len(fields) > 0 {
					current.At, _ = parseWhen(fields[0])
				}
			}
			goal.Current = current
		}
	}
	goal.DestinationSet, _ = parseWhen(goal.Fields["destination set"])
	return goal
}

// ParseQueue reads QUEUE.md.
func ParseQueue(data []byte) Queue {
	doc := parseDocument(data)
	queue := Queue{Bytes: len(data)}
	for _, s := range doc.sections(2, 3) {
		if handoffWord.MatchString(s.Title) {
			queue.Handoffs = append(queue.Handoffs, HandoffSection{Title: s.Title, Level: s.Level, Line: s.Line, Body: s.bodyText()})
		}
	}
	if s, ok := doc.section("needs operator"); ok {
		queue.NeedsOperator = s.items()
	}
	if s, ok := doc.section("next"); ok {
		queue.Next = s.items()
	}
	if s, ok := doc.section("waiting on others"); ok {
		queue.WaitingOnOthers = s.items()
	}
	if s, ok := doc.section("censuses"); ok {
		for _, item := range s.items() {
			queue.Censuses = append(queue.Censuses, parseCensus(item))
		}
		sort.SliceStable(queue.Censuses, func(i, j int) bool { return queue.Censuses[i].At.Before(queue.Censuses[j].At) })
	}
	return queue
}

func parseCensus(item Item) Census {
	census := Census{Line: item.Line}
	cells := strings.Split(item.Text, "|")
	census.At, _ = parseWhen(cells[0])
	for i, cell := range cells[1:] {
		cell = strings.TrimSpace(cell)
		if key, value, ok := strings.Cut(cell, "="); ok && strings.EqualFold(strings.TrimSpace(key), "admissible") {
			census.Admissible = strings.TrimSpace(value)
		} else if i == 0 {
			census.Scope = cell
		}
	}
	return census
}

// ReadyNext lists the ## Next items marked [ready].
func (q Queue) ReadyNext() []Item {
	var ready []Item
	for _, item := range q.Next {
		if item.Ready {
			ready = append(ready, item)
		}
	}
	return ready
}

// HasReaimDecision is true when ## Needs operator holds an item tagged [re-aim].
func (q Queue) HasReaimDecision() bool {
	for _, item := range q.NeedsOperator {
		if strings.Contains(strings.ToLower(item.Text), "[re-aim]") {
			return true
		}
	}
	return false
}

// ParseFeedback reads FEEDBACK.md.
func ParseFeedback(data []byte) Feedback {
	doc := parseDocument(data)
	feedback := Feedback{OpenBytes: len(data)}
	if offset := doc.offsetOf("resolved"); offset >= 0 {
		feedback.OpenBytes = offset
	}
	for _, s := range doc.sections(3, 3) {
		if s.Start >= feedback.OpenBytes {
			break
		}
		entry := FeedbackEntry{Title: s.Title, Line: s.Line}
		entry.ID, _, _ = strings.Cut(s.Title, " ")
		if m := feedbackSource.FindStringSubmatch(s.Title); m != nil {
			parts := strings.Split(m[1], ",")
			entry.Source = strings.TrimSpace(parts[len(parts)-1])
			if _, isDate := parseWhen(entry.Source); isDate {
				entry.Source = ""
			}
		}
		for _, l := range s.Body {
			text := strings.TrimSpace(strings.ReplaceAll(l.Text, "*", ""))
			if len(text) > 7 && strings.EqualFold(text[:7], "status:") {
				entry.Status = strings.TrimSpace(text[7:])
				break
			}
		}
		feedback.Entries = append(feedback.Entries, entry)
	}
	return feedback
}

// ParseWorkarounds reads WORKAROUNDS.md. Open bytes are the text before the
// first entry plus every open entry, or everything before "## Resolved"
// when the file has that section.
func ParseWorkarounds(data []byte) Workarounds {
	doc := parseDocument(data)
	workarounds := Workarounds{}
	resolvedAt := doc.offsetOf("resolved")
	limit := len(data)
	if resolvedAt >= 0 {
		limit = resolvedAt
	}
	var current *WorkaroundEntry
	var text []string
	preamble := -1
	finish := func(end int) {
		if current == nil {
			return
		}
		current.Bytes = end - doc.lines[current.Line-1].Start
		fields := strings.Split(strings.Join(text, " "), " · ")
		status := strings.ToLower(strings.TrimSpace(fields[len(fields)-1]))
		current.Open = len(fields) > 1 && strings.HasPrefix(status, "open")
		workarounds.Entries = append(workarounds.Entries, *current)
		current, text = nil, nil
	}
	for _, l := range doc.lines {
		if l.Start >= limit {
			break
		}
		if strings.HasPrefix(l.Text, "#") {
			finish(l.Start)
			continue
		}
		if listItemPattern.MatchString(l.Text) {
			finish(l.Start)
			if preamble < 0 {
				preamble = l.Start
			}
			current = &WorkaroundEntry{Line: l.Num}
		}
		if current != nil {
			text = append(text, strings.TrimSpace(l.Text))
		}
	}
	finish(limit)
	switch {
	case resolvedAt >= 0:
		workarounds.OpenBytes = resolvedAt
	case preamble < 0:
		workarounds.OpenBytes = len(data)
	default:
		workarounds.OpenBytes = preamble
		for _, entry := range workarounds.Entries {
			if entry.Open {
				workarounds.OpenBytes += entry.Bytes
			}
		}
	}
	return workarounds
}

// namedIn reports whether text names a feedback ID, in full ("BAS-FB-064") or
// without its goal prefix ("FB-064").
func namedIn(text, id string) bool {
	if id == "" {
		return false
	}
	candidates := []string{id}
	if parts := strings.Split(id, "-"); len(parts) > 2 {
		candidates = append(candidates, strings.Join(parts[1:], "-"))
	}
	for _, candidate := range candidates {
		pattern := regexp.MustCompile(`(?i)(^|[^A-Za-z0-9-])` + regexp.QuoteMeta(candidate) + `($|[^A-Za-z0-9])`)
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}

func describeHandoffs(handoffs []HandoffSection) string {
	parts := make([]string, 0, len(handoffs))
	for _, h := range handoffs {
		parts = append(parts, fmt.Sprintf("%q (line %d)", strings.Repeat("#", h.Level)+" "+h.Title, h.Line))
	}
	return strings.Join(parts, ", ")
}
