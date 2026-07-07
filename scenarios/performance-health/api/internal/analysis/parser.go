package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// FileTraceLoader is the real TraceLoader: it reads a captured performance.json
// (a CDP DevTools trace) plus its sibling performance.web-vitals.json and
// produces a per-component count/avg/max table by pairing ⚛ blink.user_timing
// begin/end marks by id2.local, plus long-task / paint / FCP / LCP signals.
//
// It is deliberately robust to a Tier-0 trace (no ⚛ marks): the component table
// is empty but the web-vitals are still parsed. It never errors on a Tier-0
// trace.
//
// When a SymbolLocator is wired and a budget is set, hot components are located
// (component → file:line) and findings are derived. Both are pure: NO AI.
type FileTraceLoader struct {
	// Locator resolves a component name to a "file:line" definition. When nil,
	// findings still emit with their metrics and the definition left empty.
	Locator SymbolLocator
	// BudgetMs is the average-commit-time budget (ms) above which a component is
	// flagged. Zero disables finding derivation (parse-only).
	BudgetMs float64
}

// SymbolLocator resolves a React component name to its definition site
// ("relative/path.tsx:line") within a scenario's UI source tree.
type SymbolLocator interface {
	Locate(scenario, component string) (string, bool)
}

// traceEvent is the subset of a CDP trace event we consume.
type traceEvent struct {
	Cat  string  `json:"cat"`
	Name string  `json:"name"`
	Ph   string  `json:"ph"`
	Ts   float64 `json:"ts"`
	Dur  float64 `json:"dur"`
	Args struct {
		Data map[string]any `json:"data"`
	} `json:"args"`
	ID2 struct {
		Local string `json:"local"`
	} `json:"id2"`
}

// traceFile is the top-level CDP trace shape. A trace is either a bare array of
// events or an object with a `traceEvents` array; we handle both.
type traceFile struct {
	TraceEvents []traceEvent `json:"traceEvents"`
}

// webVitals is the shape written by the injected PerformanceObserver in the
// browser-automation-studio perf capture (the `web-vitals` capture artifact).
type webVitals struct {
	LongTasks []struct {
		Start    float64 `json:"start"`
		Duration float64 `json:"duration"`
		Name     string  `json:"name"`
	} `json:"longTasks"`
	Paint []struct {
		Name  string  `json:"name"`
		Start float64 `json:"start"`
	} `json:"paint"`
	LCP *struct {
		Start float64 `json:"start"`
		Size  float64 `json:"size"`
	} `json:"lcp"`
}

// phaseSuffix matches the React Profiler phase annotation, e.g. " (update)",
// " (mount)", " (nested-update)" appended to a ⚛ mark name.
var phaseSuffix = regexp.MustCompile(`\s*\([a-z-]+\)$`)

// reactMarkPrefix is the ⚛ prefix React's Profiler integration writes on its
// blink.user_timing marks.
const reactMarkPrefix = "⚛"

var browserWorkEvents = map[string]bool{
	"EventDispatch":    true,
	"FunctionCall":     true,
	"Layout":           true,
	"Paint":            true,
	"RasterTask":       true,
	"RunTask":          true,
	"UpdateLayoutTree": true,
}

// Load parses the trace artifact (and its sibling web-vitals file) into a
// Result. The artifact is a filesystem path to performance.json; scenario, when
// non-empty, enables component symbol location.
func (l FileTraceLoader) Load(_ context.Context, scenario, artifact string) (Result, error) {
	raw, err := os.ReadFile(artifact)
	if err != nil {
		return Result{}, fmt.Errorf("analysis: read trace %q: %w", artifact, err)
	}
	events, err := parseTraceEvents(raw)
	if err != nil {
		return Result{}, fmt.Errorf("analysis: parse trace %q: %w", artifact, err)
	}
	res := Result{
		Components:   aggregateComponents(events),
		FrameSummary: summarizeFrames(events),
		BrowserWork:  summarizeBrowserWork(events),
		InputEvents:  summarizeInputEvents(events),
	}

	// Web-vitals: sibling performance.web-vitals.json. Best-effort — a Tier-0
	// capture or a missing observer file must not fail analysis.
	if vitals, ok := readWebVitals(artifact); ok {
		res.LongTaskMs = longTaskTotalMs(vitals)
		res.LongTaskMaxMs = longTaskMaxMs(vitals)
		res.FCPMs = paintMs(vitals, "first-contentful-paint")
		res.LCPMs = lcpMs(vitals)
	}

	// Deterministic finding derivation with symbol location.
	if l.BudgetMs > 0 {
		if l.Locator != nil {
			for i := range res.Components {
				if res.Components[i].Definition != "" {
					continue
				}
				if def, ok := l.Locator.Locate(scenario, res.Components[i].Component); ok {
					res.Components[i].Definition = def
				}
			}
		}
		res.Findings = DeriveFindings(res.Components, l.BudgetMs)
	}
	res.Findings = append(res.Findings, DeriveInteractionFindings(res.FrameSummary, res.BrowserWork, res.InputEvents)...)

	return res, nil
}

func summarizeFrames(events []traceEvent) FrameSummary {
	if len(events) == 0 {
		return FrameSummary{}
	}
	var (
		minTs float64
		maxTs float64
		seen  bool
		out   FrameSummary
	)
	for _, e := range events {
		if !seen || e.Ts < minTs {
			minTs = e.Ts
		}
		endTs := e.Ts + e.Dur
		if !seen || endTs > maxTs {
			maxTs = endTs
		}
		seen = true
		switch e.Name {
		case "BeginFrame":
			out.BeginFrameCount++
		case "DrawFrame":
			out.DrawnFrameCount++
		case "DroppedFrame":
			out.DroppedFrameCount++
		}
	}
	if !seen {
		return out
	}
	out.TraceDurationMs = round1((maxTs - minTs) / 1000.0)
	if out.TraceDurationMs > 0 {
		out.ApproxDrawnFPS = round1(float64(out.DrawnFrameCount) / (out.TraceDurationMs / 1000.0))
	}
	totalTerminalFrames := out.DrawnFrameCount + out.DroppedFrameCount
	if totalTerminalFrames > 0 {
		out.DroppedFrameRate = round1(float64(out.DroppedFrameCount) / float64(totalTerminalFrames))
	}
	return out
}

func summarizeBrowserWork(events []traceEvent) []EventSummary {
	agg := map[string]*eventAcc{}
	for _, e := range events {
		if !browserWorkEvents[e.Name] {
			continue
		}
		addEventDuration(agg, e.Name, e.Dur)
	}
	return eventSummaries(agg)
}

func summarizeInputEvents(events []traceEvent) []EventSummary {
	agg := map[string]*eventAcc{}
	for _, e := range events {
		if e.Name != "EventDispatch" {
			continue
		}
		addEventDuration(agg, inputEventType(e), e.Dur)
	}
	return eventSummaries(agg)
}

type eventAcc struct {
	count int
	total float64
	max   float64
}

func addEventDuration(agg map[string]*eventAcc, name string, durUs float64) {
	if name == "" {
		return
	}
	if durUs < 0 {
		durUs = 0
	}
	a := agg[name]
	if a == nil {
		a = &eventAcc{}
		agg[name] = a
	}
	a.count++
	a.total += durUs
	if durUs > a.max {
		a.max = durUs
	}
}

func eventSummaries(agg map[string]*eventAcc) []EventSummary {
	out := make([]EventSummary, 0, len(agg))
	for name, a := range agg {
		totalMs := a.total / 1000.0
		avgMs := 0.0
		if a.count > 0 {
			avgMs = totalMs / float64(a.count)
		}
		out = append(out, EventSummary{
			Name:    name,
			Count:   a.count,
			TotalMs: round1(totalMs),
			MaxMs:   round1(a.max / 1000.0),
			AvgMs:   round1(avgMs),
		})
	}
	sortEventSummaries(out)
	return out
}

func sortEventSummaries(s []EventSummary) {
	sort.Slice(s, func(i, j int) bool {
		if s[i].TotalMs != s[j].TotalMs {
			return s[i].TotalMs > s[j].TotalMs
		}
		return s[i].Name < s[j].Name
	})
}

func inputEventType(e traceEvent) string {
	for _, key := range []string{"type", "eventType"} {
		if v, ok := e.Args.Data[key]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return "unknown"
}

// parseTraceEvents decodes either a bare event array or a {traceEvents:[...]}
// object.
func parseTraceEvents(raw []byte) ([]traceEvent, error) {
	trimmed := strings.TrimSpace(string(raw))
	if strings.HasPrefix(trimmed, "[") {
		var evs []traceEvent
		if err := json.Unmarshal(raw, &evs); err != nil {
			return nil, err
		}
		return evs, nil
	}
	var tf traceFile
	if err := json.Unmarshal(raw, &tf); err != nil {
		return nil, err
	}
	return tf.TraceEvents, nil
}

// aggregateComponents pairs ⚛ blink.user_timing begin/end marks by id2.local and
// rolls them up per component (name with the phase suffix stripped). Trace `ts`
// is in microseconds; the table is reported in milliseconds.
//
// Robust to a Tier-0 trace: zero ⚛ marks → empty slice (never an error).
func aggregateComponents(events []traceEvent) []ComponentTiming {
	type acc struct {
		count int
		total float64 // microseconds
		max   float64 // microseconds
	}
	begins := map[string]traceEvent{}
	agg := map[string]*acc{}

	for _, e := range events {
		if !strings.Contains(e.Cat, "blink.user_timing") {
			continue
		}
		if !strings.HasPrefix(e.Name, reactMarkPrefix) {
			continue
		}
		key := e.ID2.Local
		if key == "" {
			continue
		}
		switch e.Ph {
		case "b":
			begins[key] = e
		case "e":
			b, ok := begins[key]
			if !ok {
				continue
			}
			delete(begins, key)
			durUs := e.Ts - b.Ts
			if durUs < 0 {
				durUs = 0
			}
			name := componentName(b.Name)
			a := agg[name]
			if a == nil {
				a = &acc{}
				agg[name] = a
			}
			a.count++
			a.total += durUs
			if durUs > a.max {
				a.max = durUs
			}
		}
	}

	out := make([]ComponentTiming, 0, len(agg))
	for name, a := range agg {
		avgMs := 0.0
		if a.count > 0 {
			avgMs = (a.total / float64(a.count)) / 1000.0
		}
		out = append(out, ComponentTiming{
			Component:   name,
			CommitCount: a.count,
			AvgMs:       round1(avgMs),
			MaxMs:       round1(a.max / 1000.0),
		})
	}
	sortComponents(out)
	return out
}

// componentName strips the ⚛ prefix and the trailing " (phase)" annotation,
// e.g. "⚛ App (update)" → "App".
func componentName(raw string) string {
	name := strings.TrimSpace(strings.TrimPrefix(raw, reactMarkPrefix))
	name = phaseSuffix.ReplaceAllString(name, "")
	return strings.TrimSpace(name)
}

// readWebVitals reads the sibling performance.web-vitals.json for a trace path.
func readWebVitals(tracePath string) (webVitals, bool) {
	dir := filepath.Dir(tracePath)
	base := filepath.Base(tracePath)

	candidates := []string{
		// Trace-name-derived sibling first: "<trace>.json" →
		// "<trace>.web-vitals.json". This keeps two traces in the same directory
		// (e.g. baseline + candidate) from sharing one vitals file.
		filepath.Join(dir, strings.TrimSuffix(base, ".json")+".web-vitals.json"),
		// Canonical BAS capture layout fallback.
		filepath.Join(dir, "performance.web-vitals.json"),
	}
	for _, c := range candidates {
		raw, err := os.ReadFile(c)
		if err != nil {
			continue
		}
		var wv webVitals
		if err := json.Unmarshal(raw, &wv); err != nil {
			continue
		}
		return wv, true
	}
	return webVitals{}, false
}

func longTaskTotalMs(wv webVitals) int64 {
	var total float64
	for _, lt := range wv.LongTasks {
		total += lt.Duration
	}
	return int64(math.Round(total))
}

func longTaskMaxMs(wv webVitals) float64 {
	var max float64
	for _, lt := range wv.LongTasks {
		if lt.Duration > max {
			max = lt.Duration
		}
	}
	return round1(max)
}

func paintMs(wv webVitals, name string) int64 {
	for _, p := range wv.Paint {
		if p.Name == name {
			return int64(math.Round(p.Start))
		}
	}
	return 0
}

func lcpMs(wv webVitals) int64 {
	if wv.LCP == nil {
		return 0
	}
	return int64(math.Round(wv.LCP.Start))
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
