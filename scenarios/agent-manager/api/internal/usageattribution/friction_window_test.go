package usageattribution

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/toolfriction"
)

// windowTools serves calls only to windows that reach back to them, like the
// store: a finding seen 20 days ago is not in the default week.
type windowTools []toolfriction.Call

func (w windowTools) ToolCalls(_ context.Context, from, to time.Time) (toolfriction.Window, error) {
	var out []toolfriction.Call
	for _, c := range w {
		if !c.At.Before(from) && c.At.Before(to) {
			out = append(out, c)
		}
	}
	return toolfriction.Window{Calls: out}, nil
}

// A finding is reported and dismissed in the window the caller listed it in:
// listed over an older --since, it is found there; asked about in the default
// week, where it is not, it is an unknown fingerprint.
func TestFrictionReportFindsAFindingInTheWindowItWasListedIn(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	since := now.Add(-30 * 24 * time.Hour)
	service, fingerprint, statuses, _ := failingSearchHub(t, now, since, now.Add(-20*24*time.Hour))
	if _, err := service.ReportFriction(context.Background(), fingerprint, time.Time{}, now); !errors.Is(err, ErrUnknownFinding) {
		t.Fatalf("report in the default week = %v, want ErrUnknownFinding", err)
	}
	if filed, err := service.ReportFriction(context.Background(), fingerprint, since, now); err != nil || filed.BugID == "" {
		t.Fatalf("report = %+v, %v", filed, err)
	}
	if err := service.DismissFriction(context.Background(), fingerprint, since, now); err != nil || statuses.get(fingerprint).Status != StatusDismissed {
		t.Fatalf("dismiss = %v, %+v", err, statuses.get(fingerprint))
	}
}

// Reporting files one bug per fingerprint: concurrent reports are one filing,
// a retry after the status write failed refiles under the same fingerprint
// (the inbox key) and records the bug, and an inbox that cannot be reached is
// ErrFrictionUnavailable with nothing recorded.
func TestFrictionReportFilesOnceAndNamesAnUnreachableInbox(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	service, fingerprint, statuses, inbox := failingSearchHub(t, now, time.Time{}, now.Add(-time.Hour))
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := service.ReportFriction(context.Background(), fingerprint, time.Time{}, now); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if inbox.calls != 1 {
		t.Fatalf("8 concurrent reports filed %d times, want 1", inbox.calls)
	}

	service, fingerprint, statuses, inbox = failingSearchHub(t, now, time.Time{}, now.Add(-time.Hour))
	statuses.fail = errors.New("database is locked")
	if _, err := service.ReportFriction(context.Background(), fingerprint, time.Time{}, now); err == nil {
		t.Fatal("a failed status write must fail the report")
	}
	statuses.fail = nil
	again, err := service.ReportFriction(context.Background(), fingerprint, time.Time{}, now)
	if err != nil || again.BugID != "knw-1" || statuses.get(fingerprint).BugID != "knw-1" || len(inbox.keys) != 1 {
		t.Fatalf("retry = %+v, %v; inbox keys %v; want the one bug recorded", again, err, inbox.keys)
	}

	service, fingerprint, statuses, inbox = failingSearchHub(t, now, time.Time{}, now.Add(-time.Hour))
	inbox.down = true
	if _, err := service.ReportFriction(context.Background(), fingerprint, time.Time{}, now); !errors.Is(err, ErrFrictionUnavailable) || statuses.get(fingerprint).Status != "" {
		t.Fatalf("inbox down = %v, status %+v; want ErrFrictionUnavailable and nothing recorded", err, statuses.get(fingerprint))
	}
}

// failingSearchHub is a service whose window holds one search-hub failure
// finding seen at seen, listed since since, with a status store and an inbox
// that deduplicates on the fingerprint like scenario-qa's.
func failingSearchHub(t *testing.T, now, since, seen time.Time) (Service, string, *lockedStatuses, *keyedInbox) {
	t.Helper()
	var calls windowTools
	for i := 0; i < 3; i++ {
		calls = append(calls, toolfriction.Call{RunID: fmt.Sprintf("r%d", i), CallEventID: fmt.Sprint(i), At: seen, CommandPath: "search-hub query", Owner: "search-hub", Outcome: "failure", Features: toolfriction.Features{Signature: "Error: index locked", ErrorLine: "Error: index locked"}})
	}
	statuses, inbox := &lockedStatuses{m: map[string]ProblemStatus{}}, &keyedInbox{keys: map[string]string{}}
	service := Service{Tools: calls, Statuses: statuses, Bugs: inbox}
	listed, err := service.Friction(context.Background(), since, now)
	if err != nil || len(listed.ToolFindings) != 1 {
		t.Fatalf("listed = %+v, %v", listed.ToolFindings, err)
	}
	return service, listed.ToolFindings[0].Fingerprint, statuses, inbox
}

type lockedStatuses struct {
	mu   sync.Mutex
	m    map[string]ProblemStatus
	fail error
}

func (s *lockedStatuses) FrictionStatuses(context.Context) (map[string]ProblemStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return maps.Clone(s.m), nil
}

func (s *lockedStatuses) SetFrictionStatus(_ context.Context, status ProblemStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail != nil {
		return s.fail
	}
	s.m[status.Fingerprint] = status
	return nil
}

func (s *lockedStatuses) get(fingerprint string) ProblemStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.m[fingerprint]
}

// keyedInbox returns the existing bug for a fingerprint it already holds, as
// scenario-qa does for an idempotency key; calls counts every filing attempt.
type keyedInbox struct {
	mu    sync.Mutex
	keys  map[string]string
	calls int
	down  bool
}

func (b *keyedInbox) FileFrictionBug(_ context.Context, bug FrictionBug) (string, error) {
	time.Sleep(time.Millisecond) // widen the window two unserialized reports would race in
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.down {
		return "", errors.New("connection refused")
	}
	b.calls++
	if id, ok := b.keys[bug.Fingerprint]; ok {
		return id, nil
	}
	b.keys[bug.Fingerprint] = fmt.Sprintf("knw-%d", len(b.keys)+1)
	return b.keys[bug.Fingerprint], nil
}

// The tool window has one lookback rule: the default week, an older start
// clamped to MaxLookback, and a start at or after now refused.
func TestToolWindowStartClampsAndRefuses(t *testing.T) {
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		since, want time.Time
		err         bool
	}{
		{time.Time{}, now.Add(-ToolFrictionWindow), false},
		{now.Add(-24 * time.Hour), now.Add(-24 * time.Hour), false},
		{now.Add(-400 * 24 * time.Hour), now.Add(-MaxLookback), false},
		{now, time.Time{}, true},
		{now.Add(time.Hour), time.Time{}, true},
	} {
		got, err := ToolWindowStart(c.since, now)
		if (err != nil) != c.err || !got.Equal(c.want) {
			t.Errorf("ToolWindowStart(%v) = %v, %v; want %v (error %v)", c.since, got, err, c.want, c.err)
		}
	}
	if _, err := (Service{}).Friction(context.Background(), now.Add(time.Hour), now); err == nil {
		t.Error("a window starting after now must be refused, not read as empty")
	}
}

// The recommendations and the friction list share one materiality rule over
// one score: a tool action's score is its finding's, and the action is listed
// exactly when the finding is material. A degradation under 1% of the Vrooli
// CLI calls that a third of the runs hit is listed, priced in its calls: the
// blocking waits it held inline are stated, never an effect in minutes.
func TestActionsListExactlyTheMaterialFindingsByTheirScore(t *testing.T) {
	var calls []toolfriction.Call
	at := time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)
	ms := int64(90_000)
	for run := 0; run < 30; run++ {
		id := fmt.Sprintf("r%02d", run)
		for i := 0; i < 60; i++ {
			calls = append(calls, toolfriction.Call{RunID: id, CallEventID: fmt.Sprintf("%s-%d", id, i), At: at, CommandPath: "agent-manager run list", Owner: "agent-manager", Outcome: "success", DurationMS: 60_000, HasDuration: true})
		}
		if run%3 == 0 {
			calls = append(calls, toolfriction.Call{
				RunID: id, CallEventID: id + "-park", At: at, CommandPath: "test-genie runs wait", Owner: "test-genie", Outcome: "success", DurationMS: ms, HasDuration: true,
				Features: toolfriction.Features{Notice: "agent-manager park unavailable (runner type is unavailable) — waiting inline instead", NoticeLine: "agent-manager park unavailable (runner type is unavailable) — waiting inline instead"},
			})
		}
	}
	tools := toolfriction.Analyze(toolfriction.Window{Calls: calls})
	report := Recommend(Inputs{Sources: Sources{Calls: calls}})
	for _, f := range tools.Findings {
		var action *Action
		for i, a := range report.Actions {
			for _, fp := range a.Fingerprints {
				if fp == f.Fingerprint {
					action = &report.Actions[i]
				}
			}
		}
		if f.Material() != (action != nil) {
			t.Errorf("%s %s: material %v (score %.4f) but listed %v", f.Kind, f.Owner, f.Material(), f.Score, action != nil)
		}
		if action != nil && action.Score != f.Score {
			t.Errorf("%s: action score %.6f ≠ finding score %.6f", f.Kind, action.Score, f.Score)
		}
		if f.Kind == toolfriction.KindDegraded {
			if f.Owner != "agent-manager" || f.CostShare >= toolfriction.MaterialScore || action == nil || action.Owner != "agent-manager" || action.Effect.Unit != UnitCalls || !strings.Contains(action.ScoreBasis, "Vrooli CLI calls") || !strings.Contains(action.Rationale, "15m 00s held inline by those blocking waits (the wait itself, not a saving)") {
				t.Errorf("park = %+v listed as %+v", f, action)
			}
		}
	}
}

// In runs whose facts record no compaction segments (read by an older
// transcript parser), R-skill counts a skill's repeated loads without claiming
// they were still in context.
func TestSkillWeightSaysResidencyUnknownWithoutSegments(t *testing.T) {
	var calls []toolfriction.Call
	for i := 0; i < 3; i++ {
		for j := 0; j < 2; j++ {
			calls = append(calls, toolfriction.Call{RunID: fmt.Sprintf("r%d", i), CommandPath: "prompt-manager skill read", Owner: "prompt-manager", SkillID: "big", Residency: 100, ResultTokens: 4_000, TokenBasis: "estimated"})
		}
	}
	for _, segments := range []bool{false, true} {
		for i := range calls {
			calls[i].OlderParser = !segments
		}
		rows, _ := skillWeights(calls, toolfriction.Analyze(toolfriction.Window{Calls: calls}).Totals)
		if len(rows) != 1 {
			t.Fatalf("rows = %+v", rows)
		}
		claims := strings.Contains(rows[0].Rationale, "while it was still in context")
		if claims == !segments || segments == strings.Contains(rows[0].Rationale, "residency unknown") {
			t.Errorf("segments %v: %s", segments, rows[0].Rationale)
		}
	}
}

// A skill read whose output went through a filter never put the skill in
// context: R-skill counts neither its load nor its tokens, and its command's
// output weight is still R-skill's to state.
func TestSkillWeightSkipsFilteredReads(t *testing.T) {
	var calls []toolfriction.Call
	for i := 0; i < 3; i++ {
		load := toolfriction.Call{RunID: fmt.Sprintf("r%d", i), CommandPath: "prompt-manager skill read", Owner: "prompt-manager", SkillID: "big", Residency: 100, ResultTokens: 4_000, TokenBasis: "estimated"}
		filtered := load
		filtered.Filtered, filtered.ResultTokens = true, 300
		calls = append(calls, load, filtered)
	}
	rows, _ := skillWeights(calls, toolfriction.Analyze(toolfriction.Window{Calls: calls}).Totals)
	if len(rows) != 1 || !strings.HasPrefix(rows[0].Rationale, "Loaded 3 times in 3 sessions, ≈4000 tokens a load") || strings.Contains(rows[0].Rationale, "1 loads repeated") {
		t.Fatalf("rows = %+v", rows)
	}
	if !skillPaths(calls)["prompt-manager skill read"] {
		t.Error("skill loads' output weight is R-skill's")
	}
}

// A providers-degraded summary whose promised notes were never printed is an
// action for that CLI to print them, not to "fix what makes it report" the
// summary line.
func TestUnprintedProviderNotesAskForTheNotes(t *testing.T) {
	line := "Searched <n> corpora; <n> total hit(s) (some providers degraded — see notes below). [degraded provider not named in the output read]"
	report := toolfriction.Report{Totals: toolfriction.Totals{VrooliCalls: 800}, Findings: []toolfriction.Finding{
		{Fingerprint: "n1", Kind: toolfriction.KindDegraded, Owner: "search-hub", CommandPath: "search-hub query", Calls: 745, Runs: 77, Unit: toolfriction.UnitCalls, Cost: 389, ErrorLine: line, Shares: []toolfriction.Share{{Kind: toolfriction.KindDegraded, Unit: toolfriction.UnitCalls, Cost: 389, Share: 0.48}}},
	}}
	actions := toolActions(report, nil, nil, 265)
	if len(actions) != 1 || actions[0].Title != "Make search-hub print the degraded-provider notes it promises" || !strings.Contains(actions[0].Text, "never named the degraded providers (745 calls in 77 runs)") {
		t.Fatalf("actions = %+v", actions)
	}
}
