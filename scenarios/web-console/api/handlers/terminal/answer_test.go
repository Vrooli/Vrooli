package terminal

import (
	"context"
	"errors"
	"testing"
	"time"

	"web-console/internal/backend"
	"web-console/session"
)

type fakeAnswerTarget struct {
	activities []session.Activity
	reads      int
	typed      []string
	pressed    []string
	events     []string
}

type fakeResumableAnswerTarget struct {
	prompt              *backend.PendingPrompt
	processingFor       time.Duration
	elapsed             time.Duration
	typed               []string
	pressed             []string
	goalResumed         bool
	afterTypeActivity   *session.Activity
	activityUnavailable bool
}

func (f *fakeResumableAnswerTarget) Activity() (session.Activity, bool) {
	if len(f.typed) == 0 {
		return waitingOn("codex", f.prompt, true), true
	}
	if f.activityUnavailable {
		return session.Activity{}, false
	}
	if f.afterTypeActivity != nil {
		return *f.afterTypeActivity, true
	}
	if f.elapsed < f.processingFor {
		return session.Activity{State: session.ActivityUnknown, Source: session.SourceOutputClock, Harness: "codex"}, true
	}
	if f.goalResumed {
		return session.Activity{State: session.ActivityWorking, Source: session.SourceScreen, Harness: "codex"}, true
	}
	return waitingOn("codex", f.prompt, true), true
}

func (f *fakeResumableAnswerTarget) TypeText(text string) error {
	f.typed = append(f.typed, text)
	return nil
}

func (f *fakeResumableAnswerTarget) PressKey(name string) error {
	f.pressed = append(f.pressed, name)
	if name == "Enter" && f.elapsed >= f.processingFor {
		f.goalResumed = true
	}
	return nil
}

func (f *fakeResumableAnswerTarget) advance(d time.Duration) { f.elapsed += d }

func (f *fakeAnswerTarget) Activity() (session.Activity, bool) {
	f.events = append(f.events, "activity")
	i := f.reads
	if i >= len(f.activities) {
		i = len(f.activities) - 1
	}
	f.reads++
	return f.activities[i], true
}

func (f *fakeAnswerTarget) TypeText(text string) error {
	f.typed = append(f.typed, text)
	f.events = append(f.events, "type:"+text)
	return nil
}

func (f *fakeAnswerTarget) PressKey(name string) error {
	f.pressed = append(f.pressed, name)
	f.events = append(f.events, "key:"+name)
	return nil
}

type fakeReplier struct {
	sessionID string
	reply     string
	calls     int
}

func (f *fakeReplier) ReplyPermission(_ context.Context, sessionID, reply string) error {
	f.calls++
	f.sessionID, f.reply = sessionID, reply
	return nil
}

var instantSettle = answerSettle{Poll: time.Millisecond, Max: 3 * time.Millisecond, Sleep: func(time.Duration) {}}

func colorQuestion(selected string) *backend.PendingPrompt {
	return &backend.PendingPrompt{Kind: "question", Text: "Which color?", Options: []backend.PromptOption{
		{Key: "1", Label: "Red", Selected: selected == "1"},
		{Key: "2", Label: "Blue", Selected: selected == "2"},
	}}
}

func waitingOn(harness string, prompt *backend.PendingPrompt, answerable bool) session.Activity {
	return session.Activity{State: session.ActivityWaiting, Harness: harness, Prompt: prompt, Answerable: answerable}
}

func TestAnswerPromptTypesTheKeyThenEnterWhileThePromptStaysUp(t *testing.T) {
	// [REQ:P0-017i] Claude Code moves the selection on a digit and confirms on
	// Enter; the Enter is sent only while the same prompt is still up.
	prompt := colorQuestion("1")
	target := &fakeAnswerTarget{activities: []session.Activity{waitingOn("claude", prompt, true), waitingOn("claude", colorQuestion("2"), true)}}
	got, err := answerPrompt(context.Background(), "s1", target, nil, PromptAnswer{OptionKey: "2", PromptHash: backend.PromptHash(prompt)}, instantSettle)
	if err != nil {
		t.Fatal(err)
	}
	if len(target.typed) != 1 || target.typed[0] != "2" || len(target.pressed) != 1 || target.pressed[0] != "Enter" {
		t.Fatalf("typed %v pressed %v, want 2 then Enter", target.typed, target.pressed)
	}
	if got.Delivery != DeliveryKeystrokes || got.Answer != "Blue" {
		t.Fatalf("result = %+v, want keystrokes answering Blue", got)
	}
}

func TestAnswerPromptStopsAfterTheKeyWhenTheKeyAnswers(t *testing.T) {
	// [REQ:P0-017i] When the digit alone resolves the prompt, no Enter follows
	// (it would land in whatever the agent shows next).
	prompt := colorQuestion("1")
	target := &fakeAnswerTarget{activities: []session.Activity{waitingOn("claude", prompt, true), {State: session.ActivityWorking, Harness: "claude"}}}
	if _, err := answerPrompt(context.Background(), "s1", target, nil, PromptAnswer{OptionKey: "1", PromptHash: backend.PromptHash(prompt)}, instantSettle); err != nil {
		t.Fatal(err)
	}
	if len(target.typed) != 1 || target.typed[0] != "1" || len(target.pressed) != 0 {
		t.Fatalf("typed %v pressed %v, want only 1", target.typed, target.pressed)
	}
}

func resumablePrompt() *backend.PendingPrompt {
	return &backend.PendingPrompt{Kind: "resumable_interruption", Text: "Goal stalled", Options: []backend.PromptOption{
		{Key: "/goal resume", Label: "Resume"},
	}}
}

func TestAnswerPromptResumesAfterTransientUnknownSettles(t *testing.T) {
	// The typed command first produces only an output-clock unknown state. Once
	// the same stalled prompt is visible again, Enter submits it and the fake
	// terminal reports that the goal resumed. The assertion is about the
	// resulting transition, not a particular number of polls.
	prompt := resumablePrompt()
	target := &fakeResumableAnswerTarget{prompt: prompt, processingFor: 2 * time.Millisecond}
	settle := answerSettle{Poll: time.Millisecond, Max: 5 * time.Millisecond, Sleep: target.advance}

	got, err := answerPrompt(context.Background(), "s1", target, nil, PromptAnswer{OptionKey: "/goal resume", PromptHash: backend.PromptHash(prompt)}, settle)
	if err != nil {
		t.Fatal(err)
	}
	if !target.goalResumed {
		t.Fatalf("resume command was not accepted: elapsed=%s pressed=%v", target.elapsed, target.pressed)
	}
	if len(target.typed) != 1 || target.typed[0] != "/goal resume" || len(target.pressed) != 1 || target.pressed[0] != "Enter" {
		t.Fatalf("typed=%v pressed=%v, want one resume command followed by one Enter", target.typed, target.pressed)
	}
	if got.Delivery != DeliveryKeystrokes || got.Answer != "Resume" {
		t.Fatalf("result = %+v, want keystroke recovery answer", got)
	}
}

func TestAnswerPromptDoesNotEnterAfterPositiveProgressEvidence(t *testing.T) {
	prompt := resumablePrompt()
	cases := []struct {
		name     string
		activity session.Activity
	}{
		{"hook reports working", session.Activity{State: session.ActivityWorking, Source: session.SourceHook, Harness: "codex"}},
		{"harness event reports working", session.Activity{State: session.ActivityWorking, Source: session.SourceHarnessEvent, Harness: "codex"}},
		{"new prompt is visible", waitingOn("codex", colorQuestion("1"), true)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := &fakeResumableAnswerTarget{prompt: prompt, afterTypeActivity: &tc.activity}
			settle := answerSettle{Poll: time.Millisecond, Max: 5 * time.Millisecond, Sleep: target.advance}
			got, err := answerPrompt(context.Background(), "s1", target, nil, PromptAnswer{OptionKey: "/goal resume", PromptHash: backend.PromptHash(prompt)}, settle)
			if err != nil {
				t.Fatal(err)
			}
			if len(target.pressed) != 0 || got.Answer != "Resume" {
				t.Fatalf("pressed=%v result=%+v, want no Enter after progress evidence", target.pressed, got)
			}
		})
	}
}

func TestAnswerPromptWaitsOutScreenOnlyWorkingBeforeSubmittingResumableCommand(t *testing.T) {
	prompt := resumablePrompt()
	screenWorking := session.Activity{State: session.ActivityWorking, Source: session.SourceScreen, Harness: "codex"}
	target := &fakeResumableAnswerTarget{prompt: prompt, afterTypeActivity: &screenWorking}
	settle := answerSettle{Poll: time.Millisecond, Max: 5 * time.Millisecond, Sleep: target.advance}

	got, err := answerPrompt(context.Background(), "s1", target, nil, PromptAnswer{OptionKey: "/goal resume", PromptHash: backend.PromptHash(prompt)}, settle)
	if err != nil {
		t.Fatal(err)
	}
	if len(target.pressed) != 1 || target.pressed[0] != "Enter" {
		t.Fatalf("pressed=%v, want one Enter after screen-only progress remained inconclusive", target.pressed)
	}
	if !target.goalResumed {
		t.Fatalf("resume command was not accepted after Enter: elapsed=%s pressed=%v", target.elapsed, target.pressed)
	}
	if target.elapsed < settle.Max {
		t.Fatalf("elapsed=%s, want bounded settle of at least %s before Enter", target.elapsed, settle.Max)
	}
	if got.Delivery != DeliveryKeystrokes || got.Answer != "Resume" {
		t.Fatalf("result=%+v, want delivered resumable command", got)
	}
}

func TestAnswerPromptCancellationDoesNotSubmitResumableCommand(t *testing.T) {
	prompt := resumablePrompt()
	target := &fakeResumableAnswerTarget{prompt: prompt, processingFor: time.Second}
	ctx, cancel := context.WithCancel(context.Background())
	settle := answerSettle{Poll: time.Millisecond, Max: 5 * time.Millisecond, Sleep: func(time.Duration) { cancel() }}
	_, err := answerPrompt(ctx, "s1", target, nil, PromptAnswer{OptionKey: "/goal resume", PromptHash: backend.PromptHash(prompt)}, settle)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v, want context canceled", err)
	}
	if len(target.pressed) != 0 {
		t.Fatalf("pressed=%v, want no Enter after cancellation", target.pressed)
	}
}

func TestAnswerPromptDoesNotSubmitWhenActivityBecomesUnavailable(t *testing.T) {
	prompt := resumablePrompt()
	target := &fakeResumableAnswerTarget{prompt: prompt, activityUnavailable: true}
	settle := answerSettle{Poll: time.Millisecond, Max: 5 * time.Millisecond, Sleep: target.advance}
	got, err := answerPrompt(context.Background(), "s1", target, nil, PromptAnswer{OptionKey: "/goal resume", PromptHash: backend.PromptHash(prompt)}, settle)
	if err != nil {
		t.Fatal(err)
	}
	if len(target.pressed) != 0 || got.Answer != "Resume" {
		t.Fatalf("pressed=%v result=%+v, want no Enter when activity becomes unavailable", target.pressed, got)
	}
}

func TestAnswerPromptRefusesWhatItCannotAnswer(t *testing.T) {
	// [REQ:P0-017i] Nothing is typed unless the session is waiting on the
	// prompt the caller saw, answering is on for it, and the option exists.
	prompt := colorQuestion("1")
	hash := backend.PromptHash(prompt)
	cases := []struct {
		name     string
		activity session.Activity
		answer   PromptAnswer
		want     error
	}{
		{"not waiting", session.Activity{State: session.ActivityIdle, Harness: "claude"}, PromptAnswer{OptionKey: "1", PromptHash: hash}, ErrFailedPrecondition},
		{"prompt changed", waitingOn("claude", prompt, true), PromptAnswer{OptionKey: "1", PromptHash: "stale"}, ErrFailedPrecondition},
		{"not answerable", waitingOn("claude", prompt, false), PromptAnswer{OptionKey: "1", PromptHash: hash}, ErrFailedPrecondition},
		{"unknown option", waitingOn("claude", prompt, true), PromptAnswer{OptionKey: "9", PromptHash: hash}, ErrInvalidArgument},
		{"cancel not offered", waitingOn("claude", prompt, true), PromptAnswer{Cancel: true, PromptHash: hash}, ErrInvalidArgument},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := &fakeAnswerTarget{activities: []session.Activity{tc.activity}}
			if _, err := answerPrompt(context.Background(), "s1", target, nil, tc.answer, instantSettle); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if len(target.typed)+len(target.pressed) != 0 {
				t.Fatalf("typed %v pressed %v, want nothing sent", target.typed, target.pressed)
			}
		})
	}
}

func TestAnswerPromptCancelPressesEscape(t *testing.T) {
	// [REQ:P0-017i] A prompt that offers "Esc to cancel" can be dismissed.
	prompt := colorQuestion("1")
	prompt.Cancellable = true
	target := &fakeAnswerTarget{activities: []session.Activity{waitingOn("claude", prompt, true)}}
	if _, err := answerPrompt(context.Background(), "s1", target, nil, PromptAnswer{Cancel: true, PromptHash: backend.PromptHash(prompt)}, instantSettle); err != nil {
		t.Fatal(err)
	}
	if len(target.pressed) != 1 || target.pressed[0] != "Escape" || len(target.typed) != 0 {
		t.Fatalf("typed %v pressed %v, want Escape only", target.typed, target.pressed)
	}
}

func TestAnswerPromptRepliesToOpenCodeThroughItsAPI(t *testing.T) {
	// [REQ:P0-017i] OpenCode permissions are answered through the harness's
	// permission API, never with keystrokes.
	prompt := &backend.PendingPrompt{Kind: "permission", Text: "bash git status", Options: []backend.PromptOption{
		{Key: "once", Label: "Allow once"}, {Key: "always", Label: "Always allow"}, {Key: "reject", Label: "Reject"},
	}}
	target := &fakeAnswerTarget{activities: []session.Activity{waitingOn("opencode", prompt, true)}}
	replier := &fakeReplier{}
	got, err := answerPrompt(context.Background(), "wc1", target, replier, PromptAnswer{OptionKey: "once", PromptHash: backend.PromptHash(prompt)}, instantSettle)
	if err != nil {
		t.Fatal(err)
	}
	if replier.calls != 1 || replier.sessionID != "wc1" || replier.reply != "once" {
		t.Fatalf("replier = %+v, want one reply once for wc1", replier)
	}
	if len(target.typed)+len(target.pressed) != 0 || got.Delivery != DeliveryHarnessAPI || got.Answer != "Allow once" {
		t.Fatalf("typed %v pressed %v result %+v, want an API answer only", target.typed, target.pressed, got)
	}
}
