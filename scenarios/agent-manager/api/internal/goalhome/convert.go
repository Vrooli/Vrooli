package goalhome

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Phase-2 migration (docs/internal/delivery-reliability-research/PHASE2-FORMAT-SPEC.md).
// ConvertHome turns a v1 goal home (GOAL/QUEUE/WORKAROUNDS/epochs) into the v2
// file set (CHARTER/STATUS/LOG) and self-verifies: it reparses its own output
// and compares the board-relevant facts against the v1 parse. It writes nothing
// when the facts would diverge, so a home is either migrated losslessly or
// flagged — never silently corrupted. FEEDBACK.md is carried unchanged.

// ConvertResult is the outcome of building the v2 files for one home.
type ConvertResult struct {
	Charter []byte
	Status  []byte
	Log     []byte
	// Equivalent is true when the v2 parse of the built files reproduces the v1
	// board facts. When false, Divergence names the first fact that differs and
	// the caller must not write the files.
	Equivalent bool
	Divergence string
}

// ConvertHome builds the v2 files from a v1 home's raw GOAL.md/QUEUE.md/
// WORKAROUNDS.md bytes and its parsed *Home (from Load), then self-verifies.
func ConvertHome(v1 *Home, goalRaw, queueRaw, workaroundsRaw []byte) ConvertResult {
	charter := buildCharter(goalRaw)
	status := buildStatus(v1, queueRaw, workaroundsRaw)
	logData := buildLog(v1)

	v2 := &Home{FileBytes: map[string]int{}, EpochsComplete: true, V2: true}
	v2.Goal = ParseCharter(charter)
	st := parseStatus(status)
	v2.Queue, v2.QueueData, v2.Workarounds, v2.Scoreboard = st.queue, status, st.workarounds, st.scoreboard
	v2.WorkaroundsKnown = statusWorkaroundsKnown(status)
	if st.current != nil {
		v2.Goal.Current = st.current
	}
	v2.Feedback = v1.Feedback // FEEDBACK.md is carried unchanged.
	epochs, ok := parseLog(logData)
	epochs = mergeActiveUnits(epochs, st.active)
	v2.Epochs = epochs

	result := ConvertResult{Charter: charter, Status: status, Log: logData}
	if !ok {
		result.Divergence = "built LOG.md did not reparse"
		return result
	}
	if diff := diffBoardFacts(projectFacts(v1), projectFacts(v2)); diff != "" {
		result.Divergence = diff
		return result
	}
	result.Equivalent = true
	return result
}

// buildCharter is GOAL.md with its "- Current:" field line removed (Current
// moves to STATUS). The board reads only "- Field:" lines from GOAL, so keeping
// them verbatim preserves Destination and Destination-set exactly.
func buildCharter(goalRaw []byte) []byte {
	var kept []string
	for _, line := range strings.Split(string(goalRaw), "\n") {
		if m := fieldPattern.FindStringSubmatch(strings.TrimRight(line, " \t")); m != nil && strings.EqualFold(strings.TrimSpace(m[1]), "current") {
			continue
		}
		kept = append(kept, line)
	}
	return []byte(strings.Join(kept, "\n"))
}

// buildStatus composes STATUS.md: the live Current reading, the one active work
// unit, the queue sections transplanted verbatim from QUEUE.md, and the open
// workarounds transplanted from WORKAROUNDS.md.
func buildStatus(v1 *Home, queueRaw, workaroundsRaw []byte) []byte {
	var b strings.Builder
	title := firstH1(queueRaw)
	if title == "" {
		title = "Status"
	}
	fmt.Fprintf(&b, "# %s\n\n", title)
	if c := v1.Goal.Current; c != nil {
		b.WriteString("## Current: " + formatNumber(c.Value))
		if !c.At.IsZero() {
			b.WriteString(" @ " + c.At.Format(time.RFC3339))
		}
		b.WriteString("\n\n")
	}
	if active := selectActive(v1.Epochs); active != nil {
		status := active.Epoch.Fields["status"]
		workers := active.Epoch.Fields["workers"]
		fmt.Fprintf(&b, "## Active\n\n- unit: %s  status: %s  workers: %s\n\n", active.ID(), status, workers)
	}
	// Queue sections (## Needs operator / ## Handoff / ## Next / ## Waiting on
	// others / ## Censuses) verbatim, so ParseQueue reads the same items.
	b.WriteString(stripFirstH1(queueRaw))
	if !missingName(v1, WorkaroundsFile) {
		b.WriteString("\n\n## Workarounds\n\n")
		b.WriteString(stripFirstH1(workaroundsRaw))
	}
	return []byte(b.String())
}

// buildLog composes LOG.md: one "## Accepted units" line per accepted epoch
// (preserving its ACCEPTED timestamp, yield, gates and worker UUIDs) and one
// "## Reopened" entry per epoch whose acceptance was reopened by a later
// directive.
func buildLog(v1 *Home) []byte {
	var accepted, reopened []string
	for _, ef := range v1.Epochs {
		acc := ef.Epoch.LastAcceptance()
		if acc == nil {
			continue
		}
		raw := strings.TrimSpace(acc.Raw)
		if !strings.HasPrefix(raw, "ACCEPTED") {
			raw = "ACCEPTED " + raw
		}
		line := fmt.Sprintf("- %s | %s", ef.ID(), raw)
		if workers := strings.TrimSpace(ef.Epoch.Fields["workers"]); workers != "" {
			line += " | workers=" + workers
		}
		accepted = append(accepted, line)
		// A reopened epoch has an acceptance plus a later directive; the board
		// treats it as active, so record the reopen so v2 reconstructs it.
		if !ef.Epoch.IsAccepted() {
			if at := reopenTime(ef.Epoch, acc.At); !at.IsZero() {
				reopened = append(reopened, fmt.Sprintf("- %s | %s", ef.ID(), at.Format(time.RFC3339)))
			}
		}
	}
	var b strings.Builder
	b.WriteString("# Accepted-unit ledger\n\n## Accepted units\n\n")
	if len(accepted) == 0 {
		b.WriteString("(none yet)\n")
	} else {
		b.WriteString(strings.Join(accepted, "\n") + "\n")
	}
	b.WriteString("\n## Reopened\n\n")
	if len(reopened) == 0 {
		b.WriteString("(none)\n")
	} else {
		b.WriteString(strings.Join(reopened, "\n") + "\n")
	}
	return []byte(b.String())
}

// reopenTime is the time of the earliest directive after the acceptance, which
// is what reopened the epoch.
func reopenTime(e *Epoch, acceptedAt time.Time) time.Time {
	var best time.Time
	for _, d := range e.Directives {
		if d.At.After(acceptedAt) && (best.IsZero() || d.At.Before(best)) {
			best = d.At
		}
	}
	return best
}

// selectActive picks the one non-accepted epoch the board would show as active:
// highest priority, newest-last on a tie. It mirrors the supervisor's
// goalHomeEpochPriority so the migrated ## Active holds the same unit.
func selectActive(epochs []EpochFile) *EpochFile {
	var active *EpochFile
	priority := 0
	for i := len(epochs) - 1; i >= 0; i-- {
		ef := epochs[i]
		if ef.Epoch.IsAccepted() {
			continue
		}
		p := epochPriority(ef.Epoch)
		if active == nil || p > priority {
			copy := ef
			active, priority = &copy, p
		}
	}
	return active
}

// epochPriority mirrors supervision.goalHomeEpochPriority.
func epochPriority(e *Epoch) int {
	if e != nil && e.LastAcceptance() != nil && !e.IsAccepted() {
		return 2
	}
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(e.Fields["status"])))
	if len(fields) == 0 {
		return 0
	}
	switch fields[0] {
	case "in-progress", "parked", "active":
		return 2
	case "admitted":
		return 1
	default:
		return 0
	}
}

// projectedFacts are the board-relevant, spend-independent facts used to verify
// a migration is lossless.
type projectedFacts struct {
	destination      string
	current          string
	activeID         string
	activeStatus     string
	activeWorkers    []string
	lastAcceptedAt   string
	lastAcceptedID   string
	lastYield        string
	acceptedWorkers  []string // per accepted unit, newest-first: sorted worker UUIDs joined
	acceptedIDs      []string
	parkWithoutChild string
	parkWithChild    string
	needsOperator    []string
	openWorkarounds  int
	openFeedback     []string
	lintCodes        []string
}

// projectFacts computes the board facts a home produces, independent of live
// run spend. It mirrors supervision.goalHomeFileFacts over the parsed epochs.
func projectFacts(home *Home) projectedFacts {
	f := projectedFacts{openWorkarounds: -1}
	if d := home.Goal.Destination; d != nil {
		f.destination = fmt.Sprintf("%s|%s|%s", d.Metric, d.Op, formatNumber(d.Target))
	}
	if c := home.Goal.Current; c != nil {
		f.current = formatNumber(c.Value) + "@" + c.At.Format(time.RFC3339)
	}
	if active := selectActive(home.Epochs); active != nil {
		f.activeID = active.ID()
		f.activeStatus = active.Epoch.Fields["status"]
		f.activeWorkers = WorkerRunIDs(active.Epoch.Fields["workers"])
	}
	for i := len(home.Epochs) - 1; i >= 0; i-- {
		ef := home.Epochs[i]
		if !ef.Epoch.IsAccepted() {
			continue
		}
		if f.lastAcceptedAt == "" {
			f.lastAcceptedID = ef.ID()
			if acc := ef.Epoch.LastAcceptance(); acc != nil {
				f.lastAcceptedAt = acc.At.Format(time.RFC3339)
				if acc.Yield != nil {
					f.lastYield = acc.Yield.String()
				}
			}
		}
		workers := WorkerRunIDs(ef.Epoch.Fields["workers"])
		f.acceptedIDs = append(f.acceptedIDs, ef.ID())
		sort.Strings(workers)
		f.acceptedWorkers = append(f.acceptedWorkers, strings.Join(workers, ","))
	}
	for _, item := range home.Queue.NeedsOperator {
		first, _, _ := strings.Cut(item.Text, "\n")
		f.needsOperator = append(f.needsOperator, strings.TrimSpace(first))
	}
	if home.V2 && home.WorkaroundsKnown || !home.V2 && !missingName(home, WorkaroundsFile) {
		open := 0
		for _, e := range home.Workarounds.Entries {
			if e.Open {
				open++
			}
		}
		f.openWorkarounds = open
	}
	for _, e := range home.Feedback.Entries {
		if e.Open() {
			f.openFeedback = append(f.openFeedback, e.ID)
		}
	}
	for _, finding := range Lint(home).Findings {
		// resume-budget and board-file-size-advisory are byte-size heuristics; the
		// v2 file set legitimately has a different size (STATUS carries current,
		// active and workarounds), so they are not attribution facts to match.
		if finding.Blocking && finding.Code != "resume-budget" {
			f.lintCodes = append(f.lintCodes, finding.Code)
		}
	}
	sort.Strings(f.needsOperator)
	sort.Strings(f.openFeedback)
	sort.Strings(f.lintCodes)
	for _, check := range []struct {
		children []string
		target   *string
	}{{nil, &f.parkWithoutChild}, {[]string{"live-child"}, &f.parkWithChild}} {
		d := DecidePark(check.children, home.Queue, 0)
		*check.target = fmt.Sprintf("%t|%s|%s", d.Allowed, d.Rule, d.Timeout)
	}
	return f
}

func missingName(home *Home, name string) bool {
	for _, m := range home.Missing {
		if m == name {
			return true
		}
	}
	return false
}

// diffBoardFacts returns the first fact that differs between a v1 home and its
// v2 conversion, or "" when they are equivalent for the board.
func diffBoardFacts(a, b projectedFacts) string {
	if a.destination != b.destination {
		return fmt.Sprintf("destination %q vs %q", a.destination, b.destination)
	}
	if a.current != b.current {
		return fmt.Sprintf("current %q vs %q", a.current, b.current)
	}
	if a.activeID != b.activeID || a.lastAcceptedID != b.lastAcceptedID {
		return fmt.Sprintf("unit identities active=%q/%q accepted=%q/%q", a.activeID, b.activeID, a.lastAcceptedID, b.lastAcceptedID)
	}
	if a.activeStatus != b.activeStatus {
		return fmt.Sprintf("active status %q vs %q", a.activeStatus, b.activeStatus)
	}
	if strings.Join(a.activeWorkers, ",") != strings.Join(b.activeWorkers, ",") {
		return fmt.Sprintf("active workers %v vs %v", a.activeWorkers, b.activeWorkers)
	}
	if a.lastAcceptedAt != b.lastAcceptedAt {
		return fmt.Sprintf("last accepted at %q vs %q", a.lastAcceptedAt, b.lastAcceptedAt)
	}
	if a.lastYield != b.lastYield {
		return fmt.Sprintf("last yield %q vs %q", a.lastYield, b.lastYield)
	}
	if strings.Join(a.acceptedWorkers, ";") != strings.Join(b.acceptedWorkers, ";") {
		return fmt.Sprintf("accepted worker sets %v vs %v", a.acceptedWorkers, b.acceptedWorkers)
	}
	if strings.Join(a.acceptedIDs, ";") != strings.Join(b.acceptedIDs, ";") {
		return fmt.Sprintf("accepted unit identities %v vs %v", a.acceptedIDs, b.acceptedIDs)
	}
	if a.parkWithoutChild != b.parkWithoutChild || a.parkWithChild != b.parkWithChild {
		return "park-policy behavior changed"
	}
	if strings.Join(a.needsOperator, "|") != strings.Join(b.needsOperator, "|") {
		return fmt.Sprintf("needs-operator %v vs %v", a.needsOperator, b.needsOperator)
	}
	if a.openWorkarounds != b.openWorkarounds {
		return fmt.Sprintf("open workarounds %d vs %d", a.openWorkarounds, b.openWorkarounds)
	}
	if strings.Join(a.openFeedback, ",") != strings.Join(b.openFeedback, ",") {
		return fmt.Sprintf("open feedback %v vs %v", a.openFeedback, b.openFeedback)
	}
	if strings.Join(a.lintCodes, ",") != strings.Join(b.lintCodes, ",") {
		return fmt.Sprintf("lint findings %v vs %v", a.lintCodes, b.lintCodes)
	}
	return ""
}

// firstH1 is the text of the first "# " heading, or "".
func firstH1(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		if m := headingPattern.FindStringSubmatch(line); m != nil && len(m[1]) == 1 {
			return strings.TrimSpace(m[2])
		}
	}
	return ""
}

// stripFirstH1 drops the first "# " heading line and any blank lines directly
// after it, returning the rest of the document.
func stripFirstH1(data []byte) string {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if m := headingPattern.FindStringSubmatch(line); m != nil && len(m[1]) == 1 {
			rest := lines[i+1:]
			for len(rest) > 0 && strings.TrimSpace(rest[0]) == "" {
				rest = rest[1:]
			}
			return strings.Join(rest, "\n")
		}
	}
	return strings.TrimLeft(string(data), "\n")
}
