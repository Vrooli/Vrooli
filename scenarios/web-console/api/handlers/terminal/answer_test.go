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
}

func (f *fakeAnswerTarget) Activity() (session.Activity, bool) {
	i := f.reads
	if i >= len(f.activities) {
		i = len(f.activities) - 1
	}
	f.reads++
	return f.activities[i], true
}

func (f *fakeAnswerTarget) TypeText(text string) error {
	f.typed = append(f.typed, text)
	return nil
}

func (f *fakeAnswerTarget) PressKey(name string) error {
	f.pressed = append(f.pressed, name)
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
