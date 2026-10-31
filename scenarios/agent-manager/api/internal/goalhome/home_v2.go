package goalhome

import (
	"errors"
	"io/fs"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Phase-2 goal-home file names (see
// docs/internal/delivery-reliability-research/PHASE2-FORMAT-SPEC.md). The v2
// reader produces the same internal Home/Goal/Epoch structs the supervision
// board already consumes, so the proto/board layer is untouched: an accepted
// unit in LOG.md becomes a synthetic accepted Epoch, STATUS's ## Active becomes
// independent active units, and a ## Reopened entry becomes a reopening Directive.
// FEEDBACK.md is unchanged from v1 and read through the existing FeedbackFile.
const (
	CharterFile = "CHARTER.md"
	StatusFile  = "STATUS.md"
	LogFile     = "LOG.md"
	// GoalFile is only the sentinel an enrollment destination may name
	// ("path:<goal home>/GOAL.md"); no reader opens it.
	GoalFile = "GOAL.md"
)

var (
	// acceptedUnitPattern reads the unit id and the "ACCEPTED <ISO>" head of a
	// "## Accepted units" line. Trailing cells (workers=, yield=, gates=,
	// summary) are parsed from the rest by reusing the epoch acceptance parser.
	acceptedUnitPattern = regexp.MustCompile(`^(\S+)\s*\|\s*ACCEPTED\b`)
	// reopenedPattern reads "<unit-id> | <ISO>" (or "<unit-id> <ISO>") under
	// "## Reopened".
	reopenedPattern = regexp.MustCompile(`^(\S+)\s*[|]?\s*(.*)$`)
)

// LoadV2 reads a Phase-2 goal home (CHARTER.md/STATUS.md/LOG.md/FEEDBACK.md)
// from fsys, whose root is the goal-home directory. Only read failures other
// than a missing file are errors.
func LoadV2(fsys fs.FS) (*Home, error) {
	return loadV2(fsys, false)
}

// LoadBoardProjectionV2 reads the bounded facts used by the supervision board
// from a Phase-2 goal home. A malformed LOG.md yields a partial home
// (EpochsComplete=false) exactly as an unreadable epoch set does in v1, so the
// board never infers accepted/active facts from a partial ledger.
func LoadBoardProjectionV2(fsys fs.FS) (*Home, error) {
	return loadV2(fsys, true)
}

func loadV2(fsys fs.FS, tolerateLogGaps bool) (*Home, error) {
	home := &Home{FileBytes: make(map[string]int), EpochsComplete: true}

	charter, err := home.read(fsys, CharterFile)
	if err != nil {
		return nil, err
	}

	status, err := home.read(fsys, StatusFile)
	if err != nil {
		return nil, err
	}

	feedback, err := home.read(fsys, FeedbackFile)
	if err != nil {
		return nil, err
	}

	logData, err := home.read(fsys, LogFile)
	if err != nil {
		if !tolerateLogGaps {
			return nil, err
		}
		home.EpochsComplete = false
		home.IncompleteEpochFiles = append(home.IncompleteEpochFiles, LogFile)
	}
	return parseV2(home, charter, status, feedback, logData, tolerateLogGaps)
}

// parseV2 assembles a Home from the four goal-home files.
func parseV2(home *Home, charter, status, feedback, logData []byte, tolerateLogGaps bool) (*Home, error) {
	home.Goal = ParseCharter(charter)
	home.Feedback = ParseFeedback(feedback)
	// STATUS supplies the live Current reading, the gradient scoreboard, and the
	// queue/workaround/park inputs that QUEUE.md + WORKAROUNDS.md carried in v1.
	st := parseStatus(status)
	home.Queue = st.queue
	home.QueueData = status
	home.Workarounds = st.workarounds
	home.WorkaroundsKnown = statusWorkaroundsKnown(status)
	home.Scoreboard = st.scoreboard
	home.Warnings = append(home.Warnings, projectionFindings(home.Goal, status)...)
	if st.current != nil {
		home.Goal.Current = st.current
	}

	if slices.Contains(home.Missing, LogFile) || slices.Contains(home.Missing, StatusFile) {
		home.EpochsComplete = false
		home.IncompleteEpochFiles = append(home.IncompleteEpochFiles, LogFile)
	}
	epochs, ok := parseLog(logData)
	if !ok {
		// A malformed LOG is a partial home: the board must not read accepted
		// or active facts from it. Mirror v1's epoch-gap behavior.
		if tolerateLogGaps {
			home.EpochsComplete = false
			home.IncompleteEpochFiles = append(home.IncompleteEpochFiles, LogFile)
		} else {
			return nil, &MalformedLogError{Detail: "LOG.md acceptance must use one ## Accepted units section with parseable, unique unit records"}
		}
	}
	// Active units follow accepted units so the selected summary remains
	// newest-last while preserving all independent worker sets.
	epochs = mergeActiveUnits(epochs, st.active)
	home.Epochs = epochs
	return home, nil
}

// MalformedLogError is returned by LoadV2 when LOG.md cannot be parsed and the
// caller did not request the tolerant board projection.
type MalformedLogError struct{ Detail string }

func (e *MalformedLogError) Error() string { return e.Detail }

// read accounts for every file through one missing/error semantics.
func (h *Home) read(fsys fs.FS, name string) ([]byte, error) {
	data, err := fs.ReadFile(fsys, name)
	if errors.Is(err, fs.ErrNotExist) {
		h.Missing = append(h.Missing, name)
		return nil, nil
	}
	if err == nil {
		h.FileBytes[name] = len(data)
	}
	return data, err
}

// ParseCharter reads CHARTER.md. "- Destination:" maps to the goal destination
// with the same regex v1 uses, and "- Budget:" maps to the new BudgetSeconds
// field (zero when absent or unparseable).
func ParseCharter(data []byte) Goal {
	doc := parseDocument(data)
	goal := Goal{Bytes: len(data), Fields: doc.fields()}
	if m := destinationPattern.FindStringSubmatch(goal.Fields["destination"]); m != nil {
		target, _, _ := ParseNumber(m[3])
		goal.Destination = &Destination{Metric: strings.TrimSpace(m[1]), Op: m[2], Target: target}
	}
	goal.BudgetSeconds = parseBudgetSeconds(goal.Fields["budget"])
	goal.DestinationSet, _ = parseWhen(goal.Fields["destination set"])
	return goal
}

// parseBudgetSeconds reads "- Budget: <duration>" as whole seconds. It accepts
// a Go duration ("72h", "90m") and plain "<n> <unit>" with h/hour, m/min,
// d/day, w/week, s/sec units. It returns 0 when absent or unparseable.
func parseBudgetSeconds(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return int64(d / time.Second)
	}
	value, rest, ok := ParseNumber(raw)
	if !ok {
		return 0
	}
	unit := strings.ToLower(strings.TrimSpace(rest))
	var seconds float64
	switch {
	case unit == "" || strings.HasPrefix(unit, "s"):
		seconds = value
	case strings.HasPrefix(unit, "min") || unit == "m":
		seconds = value * 60
	case strings.HasPrefix(unit, "h"):
		seconds = value * 3600
	case strings.HasPrefix(unit, "d"):
		seconds = value * 86400
	case strings.HasPrefix(unit, "w"):
		seconds = value * 604800
	default:
		return 0
	}
	if seconds < 0 {
		return 0
	}
	return int64(seconds)
}

// statusFacts is everything STATUS.md carries: the queue/workaround/park facts
// v1 split across QUEUE.md + WORKAROUNDS.md, the live Current reading, the one
// active Epoch, and the gradient scoreboard (new in Phase 2).
type statusFacts struct {
	queue       Queue
	workarounds Workarounds
	active      []EpochFile
	current     *Measurement
	scoreboard  []SurfaceProgress
}

// parseStatus reads STATUS.md into statusFacts.
func parseStatus(data []byte) statusFacts {
	doc := parseDocument(data)
	queue := Queue{Bytes: len(data)}
	parseQueueSections(doc, &queue)
	return statusFacts{
		queue:       queue,
		workarounds: parseStatusWorkarounds(doc),
		active:      parseActive(doc),
		current:     parseStatusCurrent(doc),
		scoreboard:  parseScoreboard(doc),
	}
}

// parseScoreboard reads "## Scoreboard" as a Markdown table with columns
// surface | target | state | owner. The header row and the "---" separator row
// are skipped; an owner cell of "-"/"—"/empty means no active owner. A row that
// is not a pipe-delimited table row with at least a surface and state is ignored.
func parseScoreboard(doc *document) []SurfaceProgress {
	s, ok := doc.section("scoreboard")
	if !ok {
		return nil
	}
	var out []SurfaceProgress
	ownerColumn := -1
	for _, l := range s.Body {
		cells, ok := tableRow(l.Text)
		if !ok || len(cells) < 3 || isSeparatorRow(cells) {
			continue
		}
		if strings.EqualFold(cells[0], "surface") {
			for i, cell := range cells {
				if strings.EqualFold(cell, "owner") {
					ownerColumn = i
				}
			}
			continue
		}
		row := SurfaceProgress{Surface: cells[0], Target: cells[1], State: strings.ToLower(cells[2])}
		if row.Surface == "" {
			continue
		}
		if ownerColumn >= 0 && len(cells) > ownerColumn {
			if owner := cells[ownerColumn]; owner != "-" && owner != "—" {
				if ids := WorkerRunIDs(owner); len(ids) == 1 && ids[0] == owner {
					row.OwnerRun = owner
				}
			}
		}
		out = append(out, row)
	}
	return out
}

// tableRow splits "| a | b | c |" into trimmed cells. ok is false for a line
// that is not a pipe-delimited row.
func tableRow(text string) ([]string, bool) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "|") {
		return nil, false
	}
	parts := strings.Split(strings.Trim(trimmed, "|"), "|")
	cells := make([]string, len(parts))
	for i, p := range parts {
		cells[i] = strings.TrimSpace(p)
	}
	return cells, true
}

// isSeparatorRow reports a "| --- | --- |" header separator: every non-empty
// cell is only dashes and colons.
func isSeparatorRow(cells []string) bool {
	seen := false
	for _, c := range cells {
		if c == "" {
			continue
		}
		seen = true
		if strings.Trim(c, "-:") != "" {
			return false
		}
	}
	return seen
}

// parseStatusCurrent reads "## Current: <value> @ <ISO>" (the title carries the
// reading; a "- Current:" body line is also accepted).
func parseStatusCurrent(doc *document) *Measurement {
	raw := ""
	for _, h := range doc.headings {
		if h.Level == 2 && strings.HasPrefix(strings.ToLower(h.Title), "current") {
			_, after, _ := strings.Cut(h.Title, ":")
			raw = strings.TrimSpace(after)
			break
		}
	}
	if raw == "" {
		if s, ok := doc.section("current"); ok {
			raw = strings.TrimSpace(s.Fields()["current"])
		}
	}
	if raw == "" {
		return nil
	}
	value, rest, ok := ParseNumber(raw)
	if !ok {
		return nil
	}
	current := &Measurement{Value: value}
	if _, when, found := strings.Cut(rest, "@"); found {
		if fields := strings.Fields(when); len(fields) > 0 {
			current.At, _ = parseWhen(fields[0])
		}
	}
	return current
}

// parseActive reads "## Active" into independent non-accepted units whose Fields hold
// "status" and "workers", so the board's active_units and selected active_unit/active_status/
// active_workers resolve exactly as from a v1 active epoch file. The section
// body is a list of "unit: <id>  status: <text>  workers: <uuids>" cells.
func parseActive(doc *document) []EpochFile {
	s, ok := doc.section("active")
	if !ok {
		return nil
	}
	var active []EpochFile
	for _, item := range s.items() {
		fields := inlineFields(item.Text)
		unit := fields["unit"]
		status := fields["status"]
		workers := fields["workers"]
		if unit == "" || strings.ContainsAny(unit, " \t;`|") {
			continue
		}
		epoch := &Epoch{Fields: map[string]string{}}
		if status != "" {
			epoch.Fields["status"] = status
		} else {
			// A unit named under ## Active with no explicit status is active work;
			// keep it visible to the board as such.
			epoch.Fields["status"] = "in-progress"
		}
		if workers != "" {
			epoch.Fields["workers"] = workers
		}
		active = append(active, unitFile(unit, epoch))
	}
	return active
}

func mergeActiveUnits(units, active []EpochFile) []EpochFile {
	for _, a := range active {
		found := false
		for i := range units {
			if units[i].ID() != a.ID() {
				continue
			}
			found = true
			// LOG is authoritative for acceptance. STATUS can refine reopened work.
			if !units[i].Epoch.IsAccepted() {
				units[i].Epoch.Fields = a.Epoch.Fields
			}
			break
		}
		if !found {
			units = append(units, a)
		}
	}
	return units
}

func unitFile(id string, epoch *Epoch) EpochFile {
	number, _ := strconv.Atoi(strings.TrimPrefix(id, "E"))
	return EpochFile{UnitID: id, Number: number, Name: id + ".md", Epoch: epoch}
}

// inlineKeyPattern matches a field key: a single bareword (no spaces), so a
// hyphenated value like "in-progress" is not glued onto the following key.
var inlineKeyPattern = regexp.MustCompile(`([A-Za-z][A-Za-z_-]*):\s*`)

// inlineFields reads "unit: <id>  status: <text>  workers: <a>,<b>" from one
// list line. A key is a bareword immediately followed by ":"; its value runs to
// the next such key (or end of line). The list marker is stripped first.
func inlineFields(text string) map[string]string {
	if m := listItemPattern.FindStringSubmatch(strings.TrimSpace(text)); m != nil {
		text = m[1]
	}
	out := map[string]string{}
	keys := inlineKeyPattern.FindAllStringSubmatchIndex(text, -1)
	for i, m := range keys {
		key := strings.ToLower(strings.TrimSpace(text[m[2]:m[3]]))
		end := len(text)
		if i+1 < len(keys) {
			end = keys[i+1][0]
		}
		value := strings.TrimRight(strings.TrimSpace(text[m[1]:end]), ",")
		if _, seen := out[key]; !seen {
			out[key] = strings.TrimSpace(value)
		}
	}
	return out
}

// parseStatusWorkarounds reads the "## Workarounds" section and parses its
// entries through the same ParseWorkarounds rules the v1 WORKAROUNDS.md uses, so
// an entry's open/closed verdict is identical before and after migration. The
// section body is handed to ParseWorkarounds verbatim.
func parseStatusWorkarounds(doc *document) Workarounds {
	s, ok := doc.section("workarounds")
	if !ok {
		return Workarounds{}
	}
	var b strings.Builder
	for _, l := range s.Body {
		b.WriteString(l.Text)
		b.WriteByte('\n')
	}
	return ParseWorkarounds([]byte(b.String()))
}

func statusWorkaroundsKnown(data []byte) bool {
	s, ok := parseDocument(data).section("workarounds")
	return ok && !strings.EqualFold(s.Fields()["assessment"], "unassessed")
}

// parseLog reads LOG.md. "## Accepted units" lines each become one accepted
// Epoch; "## Reopened" entries become a reopening Directive on the matching
// unit (dated after its ACCEPTED ts) so IsAccepted()/ReopenedBy() behave as v1.
// ok is false when an "## Accepted units" line is present but unparseable, so
// the caller treats the home as partial.
func parseLog(data []byte) ([]EpochFile, bool) {
	doc := parseDocument(data)
	acceptedSections := 0
	for _, s := range doc.sections(2, 2) {
		if s.lowerTitle() == "accepted units" {
			acceptedSections++
		}
		for _, item := range s.items() {
			if acceptedUnitPattern.MatchString(item.Text) && s.lowerTitle() != "accepted units" {
				return nil, false
			}
		}
	}
	if acceptedSections > 1 {
		return nil, false
	}
	type unit struct {
		id    string
		epoch *Epoch
	}
	var units []unit
	byID := map[string]*Epoch{}

	if s, ok := doc.section("accepted units"); ok {
		for _, item := range s.items() {
			line := strings.TrimSpace(item.Text)
			m := acceptedUnitPattern.FindStringSubmatch(line)
			if m == nil {
				return nil, false
			}
			id := m[1]
			epoch := acceptedEpochFromLogLine(id, line)
			if epoch == nil {
				return nil, false
			}
			if prior := byID[id]; prior != nil {
				// Reaccepting reopened work is still one stable unit. Preserve
				// its acceptance history and project the latest owning workers.
				prior.Acceptances = append(prior.Acceptances, epoch.Acceptances...)
				prior.Fields = epoch.Fields
				for i := range units {
					if units[i].id == id {
						units = append(units[:i], units[i+1:]...)
						break
					}
				}
				units = append(units, unit{id: id, epoch: prior})
				continue
			}
			units = append(units, unit{id: id, epoch: epoch})
			byID[id] = epoch
		}
	}

	if s, ok := doc.section("reopened"); ok {
		for _, item := range s.items() {
			m := reopenedPattern.FindStringSubmatch(strings.TrimSpace(item.Text))
			if m == nil {
				continue
			}
			id := m[1]
			at, ok := parseWhen(firstField(m[2]))
			if !ok {
				continue
			}
			epoch, known := byID[id]
			if !known {
				continue
			}
			accepted := epoch.LastAcceptance()
			if accepted == nil || !at.After(accepted.At) {
				continue
			}
			// A directive dated after acceptance is what ReopenedBy() reads.
			epoch.Directives = append(epoch.Directives, Directive{ID: "D1", At: at, Text: "reopened"})
		}
	}

	out := make([]EpochFile, 0, len(units))
	for _, u := range units {
		out = append(out, unitFile(u.id, u.epoch))
	}
	return out, true
}

// acceptedEpochFromLogLine builds a synthetic accepted Epoch from one
// "<id> | ACCEPTED <ISO> | workers=<uuids> | yield=<±n> <unit> | gates=... |
// <summary>" line. workers goes into Fields["workers"] so the board's
// per-accepted-unit spend averaging finds the same UUIDs; yield/gates go into
// the ACCEPTED acceptance line.
func acceptedEpochFromLogLine(id, line string) *Epoch {
	cells := strings.Split(line, "|")
	if len(cells) < 2 {
		return nil
	}
	epoch := &Epoch{Fields: map[string]string{"status": "accepted"}}
	// Build the acceptance text the v1 parser understands. Drop the leading id
	// cell; keep the ACCEPTED head and the yield/gates cells it reads.
	var acc []string
	for _, cell := range cells[1:] {
		cell = strings.TrimSpace(cell)
		if key, value, ok := strings.Cut(cell, "="); ok {
			switch strings.ToLower(strings.TrimSpace(key)) {
			case "workers":
				epoch.Fields["workers"] = strings.TrimSpace(value)
				continue
			case "yield":
				acc = append(acc, "yield="+strings.TrimSpace(value))
				continue
			case "gates":
				acc = append(acc, "gates="+strings.TrimSpace(value))
				continue
			}
		}
		acc = append(acc, cell)
	}
	raw := strings.Join(acc, " | ")
	if !strings.HasPrefix(raw, "ACCEPTED") {
		return nil
	}
	epoch.Acceptances = append(epoch.Acceptances, parseAcceptance(raw))
	return epoch
}

func firstField(raw string) string {
	raw = strings.Trim(strings.TrimSpace(raw), "|")
	if fields := strings.Fields(raw); len(fields) > 0 {
		return fields[0]
	}
	return ""
}

// Fields reads top-level "- Field: value" lines inside a single section body.
func (s section) Fields() map[string]string {
	fields := map[string]string{}
	for _, l := range s.Body {
		if m := fieldPattern.FindStringSubmatch(strings.TrimRight(l.Text, " \t")); m != nil {
			key := strings.ToLower(m[1])
			if _, seen := fields[key]; !seen {
				fields[key] = strings.TrimSpace(m[2])
			}
		}
	}
	return fields
}
