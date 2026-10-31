package supervision

// [REQ:REQ-P2-010]

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/repository"
	"agent-manager/internal/toolfriction"

	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type supervisionRunRepo struct {
	repository.RunRepository
	run *domain.Run
}

func (r supervisionRunRepo) Get(context.Context, uuid.UUID) (*domain.Run, error) { return r.run, nil }

type toolFrictionFixture struct {
	score toolfriction.RunScore
	err   error
}

func (f *toolFrictionFixture) RunToolFriction(context.Context, string) (toolfriction.RunScore, error) {
	return f.score, f.err
}

// A subject's friction comes from one source and one rule: the run's score
// over its facts at the current fact version, which also says how far they
// reach. A run with none at that version, or no source at all, is
// unavailable, never a zero.
func TestSubjectFrictionComesFromTheCurrentVersionScore(t *testing.T) {
	id := uuid.New()
	through := time.Now().UTC()
	tools := &toolFrictionFixture{score: toolfriction.RunScore{Commands: 4, OutcomeKnown: 4, Frictional: 1, Failed: 1, Score: .25, Through: through}}
	resolve := func(source ToolFrictionSource) SubjectSummary {
		t.Helper()
		result, err := NewRunSubjectResolver(supervisionRunRepo{run: &domain.Run{ID: id, Status: domain.RunStatusRunning}}, source).Resolve(context.Background(), []*domainpb.WatchSubject{{RunId: id.String()}})
		if err != nil || len(result) != 1 {
			t.Fatalf("%+v %v", result, err)
		}
		return result[0]
	}
	if got := resolve(tools); got.FrictionUnavailable || !got.FrictionThrough.Equal(through) || got.ToolFriction.Score != .25 {
		t.Fatalf("current projection: %+v", got)
	}
	tools.score = toolfriction.RunScore{} // no facts at the current fact version
	if got := resolve(tools); !got.FrictionUnavailable || !got.FrictionThrough.IsZero() {
		t.Fatalf("a run projected only at an older version read as current: %+v", got)
	}
	if got := resolve(nil); !got.FrictionUnavailable {
		t.Fatalf("no source treated as zero friction: %+v", got)
	}
}

// The friction_score trigger is fed by the run's tool friction (the share of
// its commands that failed, guessed a shape or polled): a run over the
// threshold signals; a run with no scored commands, or whose commands mostly
// have unknown outcomes, never does and its friction is unavailable (cannot
// tell, not quiet); a source that fails leaves friction unavailable rather
// than zero.
func TestToolFrictionFeedsTheFrictionTrigger(t *testing.T) {
	id := uuid.New()
	tools := &toolFrictionFixture{score: toolfriction.RunScore{Commands: 10, OutcomeKnown: 10, Failed: 6, Frictional: 6, Score: 0.6, Through: time.Now().UTC()}}
	resolver := NewRunSubjectResolver(supervisionRunRepo{run: &domain.Run{ID: id, Status: domain.RunStatusRunning}}, tools)
	evaluate := func() (SubjectSummary, *domainpb.WatchDecision) {
		t.Helper()
		subjects, err := resolver.Resolve(context.Background(), []*domainpb.WatchSubject{{RunId: id.String()}})
		if err != nil {
			t.Fatal(err)
		}
		watch := &domainpb.CohortWatch{Spec: &domainpb.WatchSpec{Triggers: &domainpb.WatchTriggers{FrictionScore: .5}}}
		decision, err := (TriggerEvaluator{}).Evaluate(context.Background(), EvaluationInput{Watch: watch, Subjects: subjects, Now: time.Now()})
		if err != nil {
			t.Fatal(err)
		}
		return subjects[0], decision
	}
	if subject, decision := evaluate(); subject.ToolFriction.Score != 0.6 || decision.GetClassification() != "friction_threshold" {
		t.Fatalf("subject %+v decision %v, want a friction signal", subject, decision)
	}
	tools.score = toolfriction.RunScore{}
	if subject, decision := evaluate(); decision.GetClassification() == "friction_threshold" || !subject.FrictionUnavailable {
		t.Fatalf("a run with no scored commands must be unavailable and never signal: %+v", subject)
	}
	// 3 of 10 outcomes known: the 7 unknown could all be failures, so a low
	// score is not quiet, and a high one is not evidence either.
	for _, score := range []float64{0.1, 0.9} {
		tools.score = toolfriction.RunScore{Commands: 10, OutcomeKnown: 3, Score: score}
		if subject, decision := evaluate(); !subject.FrictionUnavailable || decision.GetClassification() == "friction_threshold" {
			t.Fatalf("score %.1f over 30%% known outcomes: subject %+v decision %v, want unavailable and no signal", score, subject, decision)
		}
	}
	tools.err = context.DeadlineExceeded
	if subject, _ := evaluate(); !subject.FrictionUnavailable {
		t.Fatalf("a failed read must leave friction unavailable: %+v", subject)
	}
}

// The trigger evaluator and the supervision-evaluate program apply one
// precedence to the same subjects: deadline, event_count, quiet_time, then
// friction. A run whose friction is unavailable (stale projection, events
// after the watermark, or too few known outcomes) only leaves the friction
// trigger; every other trigger still fires on both paths.
func TestTriggerAndProgramAgreeOnEveryTrigger(t *testing.T) {
	source, err := filepath.Abs("../../../.vrooli/program-runtime/supervision-evaluate.py")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	runA, runB := uuid.New(), uuid.New()
	high := toolfriction.RunScore{Commands: 10, OutcomeKnown: 10, Failed: 8, Polled: 3, Frictional: 9, Score: 0.9}
	readable := func(id uuid.UUID, score toolfriction.RunScore) SubjectSummary {
		return SubjectSummary{RunID: id.String(), Status: "running", ToolFriction: score}
	}
	stale := func(id uuid.UUID) SubjectSummary {
		s := readable(id, high)
		s.FrictionUnavailable = true
		return s
	}
	unknownOutcomes := readable(runA, toolfriction.RunScore{Commands: 10, OutcomeKnown: 3, Score: 0.9})
	events := func(n int) []eventlog.CohortEvent {
		out := make([]eventlog.CohortEvent, n)
		for i := range out {
			out[i] = eventlog.CohortEvent{ID: uuid.New(), RunID: runA, Sequence: int64(i + 1), EventType: domain.EventTypeMessage, Timestamp: now}
		}
		return out
	}
	// Both paths name the same decision; the program calls the friction
	// trigger "stalled".
	programClass := map[string]string{"friction_threshold": "stalled"}
	for name, tc := range map[string]struct {
		subjects []SubjectSummary
		events   int
		quiet    bool // last activity is older than the quiet time
		deadline bool
		want     string // Go classification; "quiet" means no trigger fired
	}{
		"readable over threshold":              {subjects: []SubjectSummary{readable(runA, high)}, want: "friction_threshold"},
		"stale projection":                     {subjects: []SubjectSummary{stale(runA)}, want: "quiet"},
		"too few known outcomes":               {subjects: []SubjectSummary{unknownOutcomes}, want: "quiet"},
		"readable beside an unavailable run":   {subjects: []SubjectSummary{stale(runA), readable(runB, high)}, want: "friction_threshold"},
		"quiet time":                           {subjects: []SubjectSummary{readable(runA, toolfriction.RunScore{})}, quiet: true, want: "quiet_time"},
		"quiet time beside unavailable":        {subjects: []SubjectSummary{stale(runA)}, quiet: true, want: "quiet_time"},
		"quiet time beside readable friction":  {subjects: []SubjectSummary{readable(runA, high)}, quiet: true, want: "quiet_time"},
		"quiet time with every run unreadable": {subjects: []SubjectSummary{stale(runA), unknownOutcomes}, quiet: true, want: "quiet_time"},
		"event count":                          {subjects: []SubjectSummary{readable(runA, toolfriction.RunScore{})}, events: 3, want: "event_count"},
		"event count beside unavailable":       {subjects: []SubjectSummary{stale(runA)}, events: 3, want: "event_count"},
		"event count over quiet and friction":  {subjects: []SubjectSummary{readable(runA, high)}, events: 3, quiet: true, want: "event_count"},
		"deadline beside unavailable":          {subjects: []SubjectSummary{stale(runA)}, events: 3, quiet: true, deadline: true, want: "deadline"},
	} {
		updated, deadline := now, now.Add(time.Hour)
		if tc.quiet {
			updated = now.Add(-2 * time.Minute)
		}
		if tc.deadline {
			deadline = now.Add(-time.Second)
		}
		input := EvaluationInput{Subjects: tc.subjects, Events: events(tc.events), Now: now, ProposedCursor: "after"}
		if tc.quiet {
			for i := range input.Events {
				input.Events[i].Timestamp = updated
			}
		}
		input.Watch = &domainpb.CohortWatch{Cursor: &domainpb.WatchCursor{Token: "before"}, UpdatedAt: timestamppb.New(updated), Spec: &domainpb.WatchSpec{PolicyVersion: "test", Triggers: &domainpb.WatchTriggers{
			FrictionScore: .5, EventCount: 3, QuietTime: durationpb.New(time.Minute), Deadline: timestamppb.New(deadline),
		}}}
		trigger, err := (TriggerEvaluator{}).Evaluate(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		built, err := NewProgramRuntimeEvaluator().programInput(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range built["run_tool_friction"].([]any) {
			for _, subject := range tc.subjects {
				if row.(map[string]any)["run_id"] == subject.RunID && !subject.FrictionReadable() {
					t.Errorf("%s: unavailable run %s entered run_tool_friction", name, subject.RunID)
				}
			}
			// A command that failed and polled counts once: 9 had friction, not 11 "failed".
			if summary := row.(map[string]any)["summary"]; row.(map[string]any)["score"] == 0.9 && summary != "9 of 10 commands had friction (a command counts once): 8 failed, 0 guessed a shape, 3 polled" {
				t.Errorf("%s: summary %q", name, summary)
			}
		}
		raw, err := json.Marshal(built)
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("python3", "testdata/supervision_behavior.py", source, "--stdin")
		cmd.Stdin = bytes.NewReader(raw)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("%s: program: %v", name, err)
		}
		var program struct {
			Disposition, Classification string
			RecommendedAction           string `json:"recommended_action"`
		}
		if err := json.Unmarshal(out, &program); err != nil {
			t.Fatalf("%s: %v: %s", name, err, out)
		}
		if trigger.GetClassification() != tc.want {
			t.Errorf("%s: trigger %s, want %s", name, trigger.GetClassification(), tc.want)
		}
		wantProgram := tc.want
		if mapped, ok := programClass[tc.want]; ok {
			wantProgram = mapped
		}
		wantDisposition, wantAction := "signal", "escalate"
		if tc.want == "quiet" {
			wantDisposition, wantAction = "quiet", "park"
		}
		if program.Disposition != wantDisposition || program.Classification != wantProgram || program.RecommendedAction != wantAction {
			t.Errorf("%s: program %s/%s/%s, want %s/%s/%s (trigger %s/%s)", name, program.Disposition, program.Classification, program.RecommendedAction,
				wantDisposition, wantProgram, wantAction, trigger.GetDisposition(), trigger.GetRecommendedAction())
		}
	}
}
