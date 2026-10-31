package usageattribution

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/transcriptindex"
)

// Group-by dimensions.
const (
	BySession  = "session"
	ByRun      = "run"
	ByProfile  = "profile"
	ByWorkload = "workload"
	ByModel    = "model"
	ByRunner   = "runner"
	ByModality = "modality"
	ByToolKind = "tool_kind"
)

// Dimensions lists every group-by value.
var Dimensions = []string{BySession, ByRun, ByProfile, ByWorkload, ByModel, ByRunner, ByModality, ByToolKind}

// ValidDimension reports whether by is a group-by value.
func ValidDimension(by string) bool {
	for _, dimension := range Dimensions {
		if dimension == by {
			return true
		}
	}
	return false
}

const (
	operatorProfile  = "operator"
	operatorWorkload = "operator-session"
	maxBurnerRunIDs  = 10
)

// Burner is one consumer of a window.
type Burner struct {
	Key, Label string
	Points     float64
	Share      float64
	Tokens     domain.UsageCounters
	Weighted   float64
	Sessions   int
	Runs       int
	// Requests counts provider requests (one per model call), not
	// conversational turns.
	Requests  int
	RunIDs    []string
	Runner    string
	Model     string
	ProfileID string
	Workload  string
	FirstAt   time.Time
	LastAt    time.Time
	Facts     transcriptindex.TurnFacts
	Source    string
}

type burnerAccumulator struct {
	Burner
	sessions, runs  map[string]bool
	runPoints       map[string]float64
	runners, models map[string]bool
	profiles        map[string]bool
	workloads       map[string]bool
	sources         map[string]bool
}

// Group folds a window's attributed turns into burners by dimension, largest
// first. runID, when set, keeps only that run's turns. A tool_kind turn is
// split across its tool kinds by call count; every other dimension gives a
// turn one key, so the burners' points always sum to the attributed points of
// the turns kept.
func Group(result Result, by string, runs map[string]Run, runID string) []Burner {
	groups := map[string]*burnerAccumulator{}
	for i, turn := range result.Turns {
		if runID != "" && turn.RunID != runID {
			continue
		}
		run := runs[turn.RunID]
		for _, part := range keysFor(by, turn, run) {
			group := groups[part.key]
			if group == nil {
				group = &burnerAccumulator{Burner: Burner{Key: part.key, Label: part.label}, sessions: map[string]bool{}, runs: map[string]bool{}, runPoints: map[string]float64{}, runners: map[string]bool{}, models: map[string]bool{}, profiles: map[string]bool{}, workloads: map[string]bool{}, sources: map[string]bool{}}
				groups[part.key] = group
			}
			group.add(turn, run, result.Points[i]*part.fraction, result.Weighted[i]*part.fraction, part.fraction)
		}
	}
	burners := make([]Burner, 0, len(groups))
	for _, group := range groups {
		burners = append(burners, group.finish(result.Observed))
	}
	sort.Slice(burners, func(i, j int) bool {
		if burners[i].Points != burners[j].Points {
			return burners[i].Points > burners[j].Points
		}
		return burners[i].Key < burners[j].Key
	})
	return burners
}

type groupKey struct {
	key, label string
	fraction   float64
}

func keysFor(by string, turn Turn, run Run) []groupKey {
	one := func(key, label string) []groupKey {
		if label == "" {
			label = key
		}
		return []groupKey{{key: key, label: shortenUUIDs(label), fraction: 1}}
	}
	switch by {
	case ByRun:
		if run.Label == "" {
			return one(turn.RunID, workloadLabel(run.Workload, "Run")+" · "+uuidStub(turn.RunID))
		}
		return one(turn.RunID, run.Label)
	case ByProfile:
		if run.ProfileID == "" {
			return one(operatorProfile, "Operator sessions (no profile)")
		}
		return one(run.ProfileID, run.ProfileName)
	case ByWorkload:
		if run.Workload == "" {
			return one(operatorWorkload, "Operator sessions (no workload)")
		}
		return one(run.Workload, workloadLabel(run.Workload, run.Workload))
	case ByModel:
		model := turn.Model
		if model == "" {
			model = "unknown"
		}
		return one(model, "")
	case ByRunner:
		return one(turn.Runner, "")
	case ByModality:
		return one(Modality(turn.Facts), "")
	case ByToolKind:
		calls := 0
		for _, count := range turn.Facts.ToolKinds {
			calls += count
		}
		if calls == 0 {
			return one("none", "No tool calls")
		}
		kinds := make([]string, 0, len(turn.Facts.ToolKinds))
		for kind := range turn.Facts.ToolKinds {
			kinds = append(kinds, kind)
		}
		sort.Strings(kinds)
		parts := make([]groupKey, 0, len(kinds))
		for _, kind := range kinds {
			parts = append(parts, groupKey{key: kind, label: kind, fraction: float64(turn.Facts.ToolKinds[kind]) / float64(calls)})
		}
		return parts
	}
	session := run.Session
	if session == "" {
		session = turn.RunID
	}
	return one(session, run.SessionLabel)
}

// Modality is a turn's primary modality: the non-text work it carried, or
// text.
func Modality(facts transcriptindex.TurnFacts) string {
	switch {
	case facts.ImagesOut > 0:
		return "image_out"
	case facts.ImagesIn > 0:
		return "image_in"
	case facts.AudioIn > 0:
		return "audio_in"
	case facts.WebSearches > 0:
		return "web_search"
	case facts.WebFetches > 0:
		return "web_fetch"
	}
	return "text"
}

func (g *burnerAccumulator) add(turn Turn, run Run, points, weighted, fraction float64) {
	g.Points += points
	g.Weighted += weighted
	g.Tokens = addCounters(g.Tokens, scaleCounters(turn.Tokens, fraction))
	g.Requests += turn.Reports
	g.runs[turn.RunID] = true
	g.runPoints[turn.RunID] += points
	session := run.Session
	if session == "" {
		session = turn.RunID
	}
	g.sessions[session] = true
	g.runners[turn.Runner] = true
	g.models[turn.Model] = true
	g.profiles[run.ProfileID] = true
	g.workloads[run.Workload] = true
	source := run.Source
	if source == "" {
		source = SourceIndexed
	}
	g.sources[source] = true
	if g.FirstAt.IsZero() || turn.At.Before(g.FirstAt) {
		g.FirstAt = turn.At
	}
	if turn.At.After(g.LastAt) {
		g.LastAt = turn.At
	}
	if fraction == 1 {
		g.Facts = addFacts(g.Facts, turn.Facts)
	}
}

func (g *burnerAccumulator) finish(observed float64) Burner {
	b := g.Burner
	if observed > 0 {
		b.Share = b.Points / observed
	}
	b.Sessions, b.Runs = len(g.sessions), len(g.runs)
	for id := range g.runs {
		b.RunIDs = append(b.RunIDs, id)
	}
	sort.Slice(b.RunIDs, func(i, j int) bool {
		if g.runPoints[b.RunIDs[i]] != g.runPoints[b.RunIDs[j]] {
			return g.runPoints[b.RunIDs[i]] > g.runPoints[b.RunIDs[j]]
		}
		return b.RunIDs[i] < b.RunIDs[j]
	})
	if len(b.RunIDs) > maxBurnerRunIDs {
		b.RunIDs = b.RunIDs[:maxBurnerRunIDs]
	}
	b.Runner, b.Model, b.ProfileID, b.Workload = sole(g.runners), sole(g.models), sole(g.profiles), sole(g.workloads)
	b.Source = sole(g.sources)
	if len(g.sources) > 1 {
		b.Source = SourceMixed
	}
	return b
}

func sole(values map[string]bool) string {
	if len(values) != 1 {
		return ""
	}
	for value := range values {
		return value
	}
	return ""
}

func scaleCounters(c domain.UsageCounters, fraction float64) domain.UsageCounters {
	if fraction == 1 {
		return c
	}
	scale := func(v int) int { return int(float64(v)*fraction + 0.5) }
	return domain.UsageCounters{InputTokens: scale(c.InputTokens), OutputTokens: scale(c.OutputTokens), CacheReadTokens: scale(c.CacheReadTokens), CacheCreationTokens: scale(c.CacheCreationTokens)}
}

func addFacts(a, b transcriptindex.TurnFacts) transcriptindex.TurnFacts {
	a.ImagesIn += b.ImagesIn
	a.ImagesInUser += b.ImagesInUser
	a.ImagesInHighDetail += b.ImagesInHighDetail
	a.ImageBytesIn += b.ImageBytesIn
	a.ImagesOut += b.ImagesOut
	a.WebSearches += b.WebSearches
	a.WebFetches += b.WebFetches
	a.AudioIn += b.AudioIn
	a.ToolCalls += b.ToolCalls
	for kind, count := range b.ToolKinds {
		if a.ToolKinds == nil {
			a.ToolKinds = map[string]int{}
		}
		a.ToolKinds[kind] += count
	}
	if b.CreditsBalance != "" {
		a.CreditsBalance = b.CreditsBalance
	}
	return a
}

var uuidPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

// uuidStub abbreviates an id for a label: its first four characters and an
// ellipsis.
func uuidStub(id string) string {
	if len(id) > 4 {
		return id[:4] + "…"
	}
	return id
}

// shortenUUIDs keeps a raw UUID out of a label.
func shortenUUIDs(label string) string {
	return uuidPattern.ReplaceAllStringFunc(label, uuidStub)
}

// workloadLabel names a kind:key workload in words ("Workflow draft"); other
// is the label when the workload is empty.
func workloadLabel(workload, other string) string {
	kind, key, found := strings.Cut(workload, ":")
	if !found {
		return other
	}
	switch kind {
	case "workflow_node":
		return "Workflow " + key
	case "interactive":
		return "Interactive " + key
	case "adhoc":
		return "Ad hoc " + key
	}
	return kind + " " + key
}

// ShortID abbreviates a UUID-like key for labels.
func ShortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func pts(value float64) string { return fmt.Sprintf("≈%.1f pts", value) }

// MeteredLine is work the provider meters outside the token counters (AM-3).
type MeteredLine struct {
	Kind   string
	Count  int
	State  string // known | estimated | unknown
	Amount string
	Note   string
}

// MeteredLines lists a window's work the provider meters outside the token
// counters: Codex credits, web searches and generated images. None of it is in
// Points. A Vrooli scenario call is local work, not provider-metered: its
// count, failures and output weight are in the tool friction profiles.
func MeteredLines(result Result) []MeteredLine {
	var webSearches, imagesOut int
	var firstCredits, lastCredits string
	for _, turn := range result.Turns {
		webSearches += turn.Facts.WebSearches
		imagesOut += turn.Facts.ImagesOut
		if turn.Facts.CreditsBalance != "" {
			if firstCredits == "" {
				firstCredits = turn.Facts.CreditsBalance
			}
			lastCredits = turn.Facts.CreditsBalance
		}
	}
	var lines []MeteredLine
	if lastCredits != "" {
		line := MeteredLine{Kind: "codex_credits", State: "known", Amount: "0", Note: "credits balance " + lastCredits + " unchanged in this window"}
		if firstCredits != lastCredits {
			line.Amount, line.Note = firstCredits+" → "+lastCredits, "credits balance moved in this window"
		}
		lines = append(lines, line)
	}
	if webSearches > 0 {
		lines = append(lines, MeteredLine{Kind: "web_search", Count: webSearches, State: "unknown", Note: "searches may be metered outside the token counters; the plan charge is not reported"})
	}
	if imagesOut > 0 {
		lines = append(lines, MeteredLine{Kind: "image_generation", Count: imagesOut, State: "unknown", Note: "generated images draw on shared limits or credits; the transcript records no charge"})
	}
	return lines
}
