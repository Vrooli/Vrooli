package runsignal

import (
	"encoding/json"
	"flag"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// The precision gate (SPEC 11.2): a labelled corpus of real episode windows and
// a per-pattern precision/recall computation. A pattern below the precision
// floor, or with too few labels, must be Quarantined in its PatternSpec; the
// test fails when an unlisted pattern misses the gate. Corpus provenance and
// label rules: testdata/precision/README.md.

const (
	precisionFloor    = 0.8
	precisionMinLabel = 10
	recallFloor       = 0.8
	recallMinTrue     = 3
	// precisionMinDetectedTrue and wilsonFloor stop a tiny sample passing on
	// a perfect ratio: a pattern needs this many matched true detections and
	// the 95% Wilson lower bound of its precision must clear the floor.
	precisionMinDetectedTrue = 5
	wilsonFloor              = 0.7
)

// wilsonLower is the lower bound of the 95% Wilson score interval.
func wilsonLower(successes, n int) float64 {
	if n == 0 {
		return 0
	}
	const z = 1.96
	p, nf := float64(successes)/float64(n), float64(n)
	denom := 1 + z*z/nf
	centre := p + z*z/(2*nf)
	margin := z * math.Sqrt(p*(1-p)/nf+z*z/(4*nf*nf))
	return (centre - margin) / denom
}

var updatePrecision = flag.Bool("update", false, "rewrite testdata/precision/results.json")

type precisionWindow struct {
	ID           string           `json:"id"`
	Pattern      string           `json:"pattern"`
	Label        string           `json:"label"` // true, false, ambiguous or clean
	Reason       string           `json:"reason"`
	RunID        string           `json:"runId"`
	StartEventID string           `json:"startEventId"`
	EndEventID   string           `json:"endEventId"`
	GoalRun      bool             `json:"goalRun"`
	Events       []precisionEvent `json:"events"`
	// Episodes are further labelled spans in the same window, each for one
	// pattern. The window-level label above stays the primary one.
	Episodes []labelledEpisode `json:"episodes,omitempty"`
	// Relabel names the rule change that moved the label; OldLabel is the label before it.
	Relabel  string `json:"relabel,omitempty"`
	OldLabel string `json:"oldLabel,omitempty"`
}

// blindLabeler marks labels written without reading detector source, from the
// pattern definitions and the evidence alone. They are out-of-sample.
const blindLabeler = "blind-2026-10-30"

// labelledEpisode is one labelled span: a pattern, the events it covers, a
// label (true, false or ambiguous) and why.
type labelledEpisode struct {
	Pattern    string `json:"pattern"`
	StartEvent string `json:"start_event"`
	EndEvent   string `json:"end_event"`
	Label      string `json:"label"`
	Reason     string `json:"reason"`
	Labeler    string `json:"labeler"`
	Relabel    string `json:"relabel,omitempty"`
	OldLabel   string `json:"oldLabel,omitempty"`
}

type precisionEvent struct {
	ID        string          `json:"id"`
	Sequence  int64           `json:"sequence"`
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// PrecisionSplit is one pattern's counts and scores over a subset of windows.
type PrecisionSplit struct {
	LabelledTrue  int `json:"labelledTrue"`
	LabelledFalse int `json:"labelledFalse"`
	Ambiguous     int `json:"ambiguous"`
	DetectedTrue  int `json:"detectedTrue"`
	DetectedFalse int `json:"detectedFalse"`
	// FoundTrue counts labelled-true episodes found at least once (recall's numerator).
	FoundTrue int `json:"foundTrue"`
	// FalsePositives counts detections of this pattern in a verified clean
	// window or overlapping a span labelled false for it.
	FalsePositives int `json:"falsePositives"`
	// Unlabelled counts detections no label covers: not known false, just not
	// yet judged. The gate limits their share of a pattern's detections.
	Unlabelled int     `json:"unlabelled"`
	Precision  float64 `json:"precision"`
	// WilsonLower is the 95% Wilson lower bound of Precision over matched detections.
	WilsonLower float64 `json:"wilsonLower"`
	Recall      float64 `json:"recall"`
}

func (s PrecisionSplit) detections() int {
	return s.DetectedTrue + s.DetectedFalse + s.FalsePositives + s.Unlabelled
}

func (s PrecisionSplit) labels() int { return s.LabelledTrue + s.LabelledFalse }

func (s *PrecisionSplit) score() {
	if flagged := s.DetectedTrue + s.DetectedFalse + s.FalsePositives; flagged > 0 {
		s.Precision = round3(float64(s.DetectedTrue) / float64(flagged))
		s.WilsonLower = round3(wilsonLower(s.DetectedTrue, flagged))
	}
	if s.LabelledTrue > 0 {
		s.Recall = round3(float64(s.FoundTrue) / float64(s.LabelledTrue))
	}
}

// PrecisionResult is one pattern's row in results.json: the totals over all
// windows, and the same scores over the tune split (windows the rules were
// developed against) and the hold-out split (a deterministic 30% by window-id
// hash that rule development must not look at).
type PrecisionResult struct {
	Pattern string `json:"pattern"`
	PrecisionSplit
	Tune    PrecisionSplit `json:"tune"`
	Holdout PrecisionSplit `json:"holdout"`
	// Blind covers only blind-labelled episodes: out-of-sample, since no rule
	// was tuned on them. Its precision is true / (true + false) detections.
	Blind       PrecisionSplit `json:"blind"`
	GateBasis   string         `json:"gateBasis"` // "holdout", or "in-sample" when the hold-out has too few labels
	Quarantined bool           `json:"quarantined"`
}

func (r PrecisionResult) labels() int { return r.LabelledTrue + r.LabelledFalse }

// holdoutMinLabel is the hold-out labels a pattern needs to be gated on the
// hold-out alone.
const holdoutMinLabel = 5

// maxUnlabelledShare is the share of a pattern's detections that may lack a
// label; above it the pattern must be labelled further.
const maxUnlabelledShare = 0.2

func (r PrecisionResult) tooManyUnlabelled() bool {
	d := r.detections()
	return d > 0 && float64(r.Unlabelled) > maxUnlabelledShare*float64(d)
}

// isHoldout puts 30% of windows, chosen by hashing the window id, in the
// hold-out split.
func isHoldout(id string) bool {
	h := fnv.New32a()
	h.Write([]byte(id))
	return h.Sum32()%100 < 30
}

// gated is the split the gate reads: the hold-out once it has enough labels,
// otherwise every label (in-sample).
func (r PrecisionResult) gated() PrecisionSplit {
	if r.GateBasis == "holdout" {
		return r.Holdout
	}
	return r.PrecisionSplit
}

// passesGate is the SPEC 11.2 rule: enough labels overall, enough precision,
// and no lost true episodes once there are enough true labels to measure
// recall, all read from the gated split.
func (r PrecisionResult) passesGate() bool {
	g := r.gated()
	// The hold-out can only tighten the gate: all-label precision must clear
	// the floor too, so a tiny hold-out cannot rescue a pattern.
	if r.DetectedTrue < precisionMinDetectedTrue || r.WilsonLower < wilsonFloor {
		return false
	}
	if r.tooManyUnlabelled() || r.labels() < precisionMinLabel || g.Precision < precisionFloor || r.Precision < precisionFloor {
		return false
	}
	if g.LabelledTrue >= recallMinTrue && g.Recall < recallFloor {
		return false
	}
	return r.LabelledTrue < recallMinTrue || r.Recall >= recallFloor
}

// maxSpanWidthFactor bounds how much wider than the labelled span a detection
// may be and still count as finding it.
const maxSpanWidthFactor = 3

func loadPrecisionCorpus(t *testing.T) []precisionWindow {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "precision", "corpus.json"))
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	var windows []precisionWindow
	if err := json.Unmarshal(raw, &windows); err != nil {
		t.Fatalf("decode corpus: %v", err)
	}
	return windows
}

func parsePrecisionTime(t *testing.T, s string) time.Time {
	t.Helper()
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05"} {
		if ts, err := time.Parse(layout, s); err == nil {
			return ts
		}
	}
	t.Fatalf("unparseable timestamp %q", s)
	return time.Time{}
}

// rebuildPrecisionEvents turns corpus events back into the domain events the
// production path reads.
func rebuildPrecisionEvents(t *testing.T, w precisionWindow) []*domain.RunEvent {
	t.Helper()
	runID := uuid.MustParse(w.RunID)
	events := make([]*domain.RunEvent, 0, len(w.Events))
	for _, e := range w.Events {
		var payload domain.EventPayload
		switch domain.RunEventType(e.Type) {
		case domain.EventTypeMessage:
			payload = &domain.MessageEventData{}
		case domain.EventTypeToolCall:
			payload = &domain.ToolCallEventData{}
		case domain.EventTypeToolResult:
			payload = &domain.ToolResultEventData{}
		case domain.EventTypeStatus:
			payload = &domain.StatusEventData{}
		case domain.EventTypeGoalStatusChanged:
			payload = &domain.GoalStatusChangedEventData{}
		case domain.EventTypeError:
			payload = &domain.ErrorEventData{}
		default:
			continue
		}
		if err := json.Unmarshal(e.Data, payload); err != nil {
			t.Fatalf("%s: decode %s event: %v", w.ID, e.Type, err)
		}
		events = append(events, &domain.RunEvent{
			ID: uuid.MustParse(e.ID), RunID: runID, Sequence: e.Sequence,
			EventType: domain.RunEventType(e.Type), Timestamp: parsePrecisionTime(t, e.Timestamp), Data: payload,
		})
	}
	return events
}

// detectionSpan is the event-index range an episode covers in a window; ok is
// false when none of its events is in the window.
func detectionSpan(ep FrictionEpisode, index map[string]int) (lo, hi int, ok bool) {
	lo, hi = int(^uint(0)>>1), -1
	for _, id := range append([]string{ep.StartEventID, ep.EndEventID}, ep.EvidenceEventIDs...) {
		if i, found := index[id]; found {
			lo, hi = min(lo, i), max(hi, i)
		}
	}
	return lo, hi, hi >= 0
}

// windowDetections runs the production path over a window and returns its
// episodes with their spans.
func windowDetections(t *testing.T, w precisionWindow) (episodes []FrictionEpisode, spans [][2]int) {
	t.Helper()
	events := rebuildPrecisionEvents(t, w)
	facts := DeriveInvocationFacts(events)
	index := make(map[string]int, len(events))
	for i, e := range events {
		index[e.ID.String()] = i
	}
	for _, ep := range DeriveEpisodes(facts, events, WithGoalRun(w.GoalRun)) {
		if lo, hi, ok := detectionSpan(ep, index); ok {
			episodes = append(episodes, ep)
			spans = append(spans, [2]int{lo, hi})
		}
	}
	return episodes, spans
}

// labelledSpans lists every labelled episode of a window with its event-index
// range: the primary window label and any extra episodes.
type spanLabel struct {
	pattern, label string
	lo, hi         int
	blind          bool
}

func windowLabels(t *testing.T, w precisionWindow) []spanLabel {
	t.Helper()
	var out []spanLabel
	if w.Label == "true" || w.Label == "false" || w.Label == "ambiguous" {
		lo, hi := labelledSpan(w)
		out = append(out, spanLabel{pattern: w.Pattern, label: w.Label, lo: lo, hi: hi})
	}
	for _, e := range w.Episodes {
		lo, hi := indexOfEvent(w, e.StartEvent), indexOfEvent(w, e.EndEvent)
		if lo < 0 || hi < lo {
			t.Fatalf("%s: episode %s %s..%s is not in the window", w.ID, e.Pattern, e.StartEvent, e.EndEvent)
		}
		out = append(out, spanLabel{pattern: e.Pattern, label: e.Label, lo: lo, hi: hi, blind: e.Labeler == blindLabeler})
	}
	return out
}

func indexOfEvent(w precisionWindow, id string) int {
	for i, e := range w.Events {
		if e.ID == id {
			return i
		}
	}
	return -1
}

func labelledSpan(w precisionWindow) (lo, hi int) {
	lo, hi = indexOfEvent(w, w.StartEventID), indexOfEvent(w, w.EndEventID)
	// A windowed-out end (very long episodes are cut in the middle) still
	// covers everything from its start.
	if hi < 0 {
		hi = len(w.Events) - 1
	}
	if lo < 0 {
		lo = 0
	}
	return lo, hi
}

// detectionOutcome is how one detection scores against a window's labels.
type detectionOutcome int

const (
	outcomeFound      detectionOutcome = iota // finds a labelled-true or labelled-false episode
	outcomeDuplicate                          // a second detection of an already found episode
	outcomeAmbiguous                          // overlaps an ambiguous span
	outcomeClean                              // in a verified clean window: false positive
	outcomeFalseSpan                          // overlaps a span labelled false
	outcomeUnlabelled                         // no label covers it
)

// classify scores each detection of a window. found[i] is the label index a
// found detection matched.
func classify(w precisionWindow, labels []spanLabel, episodes []FrictionEpisode, spans [][2]int) (outcomes []detectionOutcome, matched []int) {
	hit := make([]bool, len(labels))
	foundSpans := make(map[int][][2]int, len(labels))
	outcomes, matched = make([]detectionOutcome, len(episodes)), make([]int, len(episodes))
	for i, ep := range episodes {
		matched[i] = -1
		outcomes[i] = outcomeUnlabelled
		overlap := func(l spanLabel) bool { return l.pattern == ep.Pattern && spans[i][0] <= l.hi && spans[i][1] >= l.lo }
		// A detection finds a labelled episode only if it overlaps the span
		// and is not much wider than it.
		// Prefer a label no detection has found yet, so two detections of two
		// labelled episodes both count.
		for _, wantUnhit := range []bool{true, false} {
			for li, l := range labels {
				if (l.label == "true" || l.label == "false") && overlap(l) && hit[li] != wantUnhit &&
					spans[i][1]-spans[i][0]+1 <= maxSpanWidthFactor*(l.hi-l.lo+1) {
					matched[i] = li
					outcomes[i] = outcomeFound
					// Two detections over the same events report one thing twice;
					// distinct events inside one labelled span (several failing
					// calls) are each a real detection.
					for _, seen := range foundSpans[li] {
						if seen == spans[i] {
							outcomes[i] = outcomeDuplicate
						}
					}
					foundSpans[li] = append(foundSpans[li], spans[i])
					hit[li] = true
					break
				}
			}
			if matched[i] >= 0 {
				break
			}
		}
		if outcomes[i] == outcomeFound || outcomes[i] == outcomeDuplicate {
			continue
		}
		switch {
		case w.Label == "clean":
			outcomes[i] = outcomeClean
		default:
			for _, l := range labels {
				if overlap(l) && l.label == "false" {
					outcomes[i] = outcomeFalseSpan
					break
				}
				if overlap(l) && l.label == "ambiguous" {
					outcomes[i] = outcomeAmbiguous
				}
			}
		}
	}
	return outcomes, matched
}

func computePrecision(t *testing.T, windows []precisionWindow) []PrecisionResult {
	t.Helper()
	byPattern := map[string]*PrecisionResult{}
	row := func(p string) *PrecisionResult {
		if byPattern[p] == nil {
			byPattern[p] = &PrecisionResult{Pattern: p}
		}
		return byPattern[p]
	}
	for _, d := range EpisodeDetectors() {
		row(d.Identifier())
	}
	for _, w := range windows {
		episodes, spans := windowDetections(t, w)
		// bump applies a change to the pattern's totals and to its split, and
		// to the blind split when the label is blind.
		bump := func(pattern string, blind bool, f func(*PrecisionSplit)) {
			r := row(pattern)
			f(&r.PrecisionSplit)
			if isHoldout(w.ID) {
				f(&r.Holdout)
			} else {
				f(&r.Tune)
			}
			if blind {
				f(&r.Blind)
			}
		}
		switch w.Label {
		case "clean", "true", "false", "ambiguous":
		default:
			t.Fatalf("%s: unknown label %q", w.ID, w.Label)
		}
		labels := windowLabels(t, w)
		for _, l := range labels {
			switch l.label {
			case "true":
				bump(l.pattern, l.blind, func(s *PrecisionSplit) { s.LabelledTrue++ })
			case "false":
				bump(l.pattern, l.blind, func(s *PrecisionSplit) { s.LabelledFalse++ })
			case "ambiguous":
				bump(l.pattern, l.blind, func(s *PrecisionSplit) { s.Ambiguous++ })
			default:
				t.Fatalf("%s: unknown episode label %q", w.ID, l.label)
			}
		}
		outcomes, matched := classify(w, labels, episodes, spans)
		counted := map[int]bool{}
		for i, ep := range episodes {
			switch outcomes[i] {
			case outcomeFound:
				l := labels[matched[i]]
				if l.label == "true" {
					bump(ep.Pattern, l.blind, func(s *PrecisionSplit) { s.DetectedTrue++ })
					if !counted[matched[i]] {
						counted[matched[i]] = true
						bump(ep.Pattern, l.blind, func(s *PrecisionSplit) { s.FoundTrue++ })
					}
				} else {
					bump(ep.Pattern, l.blind, func(s *PrecisionSplit) { s.DetectedFalse++ })
				}
			case outcomeClean, outcomeFalseSpan, outcomeDuplicate:
				// A second episode over one labelled span is a duplicate the reader
				// would see as another problem, so it counts against precision.
				bump(ep.Pattern, false, func(s *PrecisionSplit) { s.FalsePositives++ })
			case outcomeUnlabelled, outcomeAmbiguous:
				bump(ep.Pattern, false, func(s *PrecisionSplit) { s.Unlabelled++ })
			}
		}
	}
	out := make([]PrecisionResult, 0, len(byPattern))
	for _, r := range byPattern {
		r.score()
		r.Tune.score()
		r.Holdout.score()
		r.Blind.score()
		r.GateBasis = "in-sample"
		if r.Holdout.labels() >= holdoutMinLabel {
			r.GateBasis = "holdout"
		}
		spec, _ := PatternByID(r.Pattern)
		r.Quarantined = spec.Quarantined != ""
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Pattern < out[j].Pattern })
	return out
}

func round3(v float64) float64 { return float64(int(v*1000+0.5)) / 1000 }

func TestPrecisionCorpusIsWellFormed(t *testing.T) {
	windows := loadPrecisionCorpus(t)
	ids := map[string]bool{}
	for _, w := range windows {
		if ids[w.ID] {
			t.Fatalf("duplicate window id %s", w.ID)
		}
		ids[w.ID] = true
		if strings.TrimSpace(w.Reason) == "" {
			t.Errorf("%s: label without a reason", w.ID)
		}
		switch w.Label {
		case "true", "false", "ambiguous", "clean":
		default:
			t.Errorf("%s: unknown label %q", w.ID, w.Label)
		}
		if len(w.Events) == 0 || len(w.Events) > 120 {
			t.Errorf("%s: %d events, want 1..120", w.ID, len(w.Events))
		}
	}
	if info, err := os.Stat(filepath.Join("testdata", "precision", "corpus.json")); err == nil && info.Size() > 3<<20 {
		t.Errorf("corpus is %d bytes, limit is 3 MiB", info.Size())
	}
}

// Quarantine reasons point at results.json instead of quoting numbers that
// would go stale when the corpus or detectors change.
func TestQuarantineReasonsQuoteNoNumbers(t *testing.T) {
	for _, spec := range PatternSpecs() {
		if strings.ContainsAny(spec.Quarantined, "0123456789") {
			t.Errorf("%s: quarantine reason %q quotes a number; point at testdata/precision/results.json", spec.ID, spec.Quarantined)
		}
	}
}

func TestPrecisionGate(t *testing.T) {
	windows := loadPrecisionCorpus(t)
	results := computePrecision(t, windows)

	var table strings.Builder
	fmt.Fprintf(&table, "%-27s %4s %5s %4s %6s %6s %4s %5s | %-17s | %-13s | %-17s | %-17s %s\n", "pattern", "true", "false", "amb", "detT", "detF", "FP", "unlab", "all P/R", "blind T/F P", "tune P/R (n)", "holdout P/R (n)", "gate")
	pr := func(s PrecisionSplit) string { return fmt.Sprintf("%.2f/%.2f (%d)", s.Precision, s.Recall, s.labels()) }
	for _, r := range results {
		status := "PASS " + r.GateBasis
		switch {
		case r.Quarantined:
			spec, _ := PatternByID(r.Pattern)
			status = "quarantined (" + spec.Quarantined + ")"
		case !r.passesGate():
			status = "FAIL " + r.GateBasis
		}
		fmt.Fprintf(&table, "%-27s %4d %5d %4d %6d %6d %4d %5d | %-17s | %-13s | %-17s | %-17s %s\n", r.Pattern, r.LabelledTrue, r.LabelledFalse, r.Ambiguous, r.DetectedTrue, r.DetectedFalse, r.FalsePositives, r.Unlabelled,
			fmt.Sprintf("%.2f/%.2f w%.2f", r.Precision, r.Recall, r.WilsonLower), fmt.Sprintf("%d/%d %.2f", r.Blind.LabelledTrue, r.Blind.LabelledFalse, r.Blind.Precision), pr(r.Tune), pr(r.Holdout), status)
	}
	t.Logf("per-pattern precision over %d windows (gate reads the hold-out split when it has >=%d labels, else in-sample)\n%s", len(windows), holdoutMinLabel, table.String())

	for _, r := range results {
		g := r.gated()
		switch {
		case r.Quarantined && r.passesGate():
			t.Logf("%s now passes the gate and can leave quarantine in patterns.go", r.Pattern)
		case !r.Quarantined && r.tooManyUnlabelled():
			t.Errorf("%s has %d unlabelled of %d detections (limit %.0f%%); label them blind and add them to corpus.json", r.Pattern, r.Unlabelled, r.detections(), maxUnlabelledShare*100)
		case !r.Quarantined && r.labels() < precisionMinLabel:
			t.Errorf("%s has %d labelled episodes (need %d); quarantine it in patterns.go or label more", r.Pattern, r.labels(), precisionMinLabel)
		case !r.Quarantined && r.DetectedTrue < precisionMinDetectedTrue:
			t.Errorf("%s has %d matched true detections (need %d); quarantine it in patterns.go or label more", r.Pattern, r.DetectedTrue, precisionMinDetectedTrue)
		case !r.Quarantined && r.WilsonLower < wilsonFloor:
			t.Errorf("%s Wilson lower bound %.3f is below %.1f; quarantine it in patterns.go or label more", r.Pattern, r.WilsonLower, wilsonFloor)
		case !r.Quarantined && (g.Precision < precisionFloor || r.Precision < precisionFloor):
			t.Errorf("%s %s precision %.3f (all labels %.3f) is below %.1f; quarantine it in patterns.go or fix the detector", r.Pattern, r.GateBasis, g.Precision, r.Precision, precisionFloor)
		case !r.Quarantined && g.LabelledTrue >= recallMinTrue && g.Recall < recallFloor:
			t.Errorf("%s %s recall %.3f on %d true labels is below %.1f", r.Pattern, r.GateBasis, g.Recall, g.LabelledTrue, recallFloor)
		case !r.Quarantined && r.LabelledTrue >= recallMinTrue && r.Recall < recallFloor:
			t.Errorf("%s all-label recall %.3f on %d true labels is below %.1f", r.Pattern, r.Recall, r.LabelledTrue, recallFloor)
		}
	}
	golden := filepath.Join("testdata", "precision", "results.json")
	encoded, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	encoded = append(encoded, '\n')
	if *updatePrecision {
		if err := os.WriteFile(golden, encoded, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read %s (regenerate with -update): %v", golden, err)
	}
	if string(want) != string(encoded) {
		t.Errorf("precision results changed; review the table above and regenerate with: go test ./internal/runsignal -run Precision -update\nwant:\n%s\ngot:\n%s", want, encoded)
	}
}

// TestPrecisionDumpUnlabelled writes every unlabelled detection of the listed
// patterns, with surrounding events, for blind labelling. It runs only when
// PRECISION_DUMP names an output file; PRECISION_DUMP_PATTERNS is a comma list.
// A pattern with more than dumpSampleCap detections is dumped as a
// deterministic sample ordered by an FNV hash of the detection key.
func TestPrecisionDumpUnlabelled(t *testing.T) {
	out := os.Getenv("PRECISION_DUMP")
	if out == "" {
		t.Skip("set PRECISION_DUMP=<file> to dump unlabelled detections")
	}
	want := map[string]bool{}
	for _, p := range strings.Split(os.Getenv("PRECISION_DUMP_PATTERNS"), ",") {
		want[p] = true
	}
	const dumpSampleCap = 40
	type item struct{ key, text string }
	byPattern := map[string][]item{}
	for _, w := range loadPrecisionCorpus(t) {
		events := rebuildPrecisionEvents(t, w)
		episodes, spans := windowDetections(t, w)
		outcomes, _ := classify(w, windowLabels(t, w), episodes, spans)
		for i, ep := range episodes {
			if outcomes[i] != outcomeUnlabelled || !want[ep.Pattern] {
				continue
			}
			key := fmt.Sprintf("%s|%s|%s|%s", ep.Pattern, w.ID, ep.StartEventID, ep.EndEventID)
			var b strings.Builder
			fmt.Fprintf(&b, "### %s  span %d..%d of %d  (window label: %s/%s, goalRun=%v)\n", key, spans[i][0], spans[i][1], len(events), w.Pattern, w.Label, w.GoalRun)
			for j := max(0, spans[i][0]-3); j <= min(len(events)-1, spans[i][1]+3); j++ {
				if j > spans[i][0]+6 && j < spans[i][1]-2 {
					if j == spans[i][0]+7 {
						b.WriteString("   ...\n")
					}
					continue
				}
				marker := " "
				if j >= spans[i][0] && j <= spans[i][1] {
					marker = "*"
				}
				fmt.Fprintf(&b, "%s%d %s\n", marker, j, dumpEventLine(w.Events[j]))
			}
			byPattern[ep.Pattern] = append(byPattern[ep.Pattern], item{key, b.String()})
		}
	}
	var all strings.Builder
	for _, p := range strings.Split(os.Getenv("PRECISION_DUMP_PATTERNS"), ",") {
		items := byPattern[p]
		hash := func(s string) uint32 { h := fnv.New32a(); h.Write([]byte(s)); return h.Sum32() }
		sort.Slice(items, func(i, j int) bool { return hash(items[i].key) < hash(items[j].key) })
		fmt.Fprintf(&all, "\n===== %s: %d unlabelled", p, len(items))
		if len(items) > dumpSampleCap {
			items = items[:dumpSampleCap]
			fmt.Fprintf(&all, " (deterministic sample of %d)", dumpSampleCap)
		}
		all.WriteString(" =====\n")
		for _, it := range items {
			all.WriteString(it.text)
		}
	}
	if err := os.WriteFile(out, []byte(all.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func dumpHead(v any, n int) string {
	s := strings.Join(strings.Fields(fmt.Sprint(v)), " ")
	if len(s) > n {
		s = s[:n] + "~"
	}
	return s
}

func dumpEventLine(e precisionEvent) string {
	var d map[string]any
	_ = json.Unmarshal(e.Data, &d)
	switch e.Type {
	case "tool_call":
		in, _ := json.Marshal(d["input"])
		return fmt.Sprintf("CALL %v %s", d["toolName"], dumpHead(string(in), 130))
	case "tool_result":
		return fmt.Sprintf("RES exit=%v ok=%v %s", d["exitCode"], d["success"], dumpHead(d["output"], 110))
	case "message":
		return fmt.Sprintf("MSG %v: %s", d["role"], dumpHead(d["content"], 150))
	default:
		return fmt.Sprintf("%s %s", strings.ToUpper(e.Type), dumpHead(string(e.Data), 80))
	}
}
