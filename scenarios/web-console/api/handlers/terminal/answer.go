package terminal

import (
	"context"
	"fmt"
	"time"

	"web-console/internal/backend"
	"web-console/session"
)

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#maturity-levels-per-harness

// PromptAnswer answers the prompt a session's agent is showing, as the caller
// saw it (identified by PromptHash).
type PromptAnswer struct {
	OptionKey  string
	PromptHash string
	// Cancel dismisses the prompt (Escape) instead of choosing an option.
	Cancel bool
}

// How an answer reached the agent.
const (
	DeliveryKeystrokes = "keystrokes"
	DeliveryHarnessAPI = "harness_api"
)

// PromptAnswerResult reports how the answer was delivered and what it chose.
type PromptAnswerResult struct {
	Delivery string
	// Answer is the chosen option's label; empty for a cancel.
	Answer string
}

// PromptReplier answers through a harness's own API (OpenCode permissions).
type PromptReplier interface {
	ReplyPermission(ctx context.Context, sessionID, reply string) error
}

// answerSource tags answer keystrokes on the session's input lane.
const answerSource = "prompt_answer"

// answerTarget is the slice of a session an answer needs.
type answerTarget interface {
	Activity() (session.Activity, bool)
	TypeText(text string) error
	PressKey(name string) error
}

// answerSettle bounds how long an answer watches the prompt after typing an
// option's key. Claude Code moves the selection on a digit and confirms on
// Enter; a box that already resolved on the digit must not get the Enter, which
// would land in whatever the agent shows next.
type answerSettle struct {
	Poll  time.Duration
	Max   time.Duration
	Sleep func(time.Duration)
}

var defaultAnswerSettle = answerSettle{Poll: 50 * time.Millisecond, Max: 600 * time.Millisecond, Sleep: time.Sleep}

// answerPrompt sends an answer only when the session is waiting on the prompt
// the caller saw, answering is on for it, and the option is on it.
func answerPrompt(ctx context.Context, sessionID string, target answerTarget, replier PromptReplier, answer PromptAnswer, settle answerSettle) (PromptAnswerResult, error) {
	activity, ok := target.Activity()
	if !ok || activity.State != session.ActivityWaiting || activity.Prompt == nil {
		return PromptAnswerResult{}, fmt.Errorf("the session is not waiting on a prompt: %w", ErrFailedPrecondition)
	}
	prompt := activity.Prompt
	if backend.PromptHash(prompt) != answer.PromptHash {
		return PromptAnswerResult{}, fmt.Errorf("the prompt changed since it was shown: %w", ErrFailedPrecondition)
	}
	if !activity.Answerable {
		return PromptAnswerResult{}, fmt.Errorf("answering is off for this harness or version: %w", ErrFailedPrecondition)
	}
	if answer.Cancel {
		if !prompt.Cancellable {
			return PromptAnswerResult{}, fmt.Errorf("this prompt cannot be dismissed: %w", ErrInvalidArgument)
		}
		if err := target.PressKey("Escape"); err != nil {
			return PromptAnswerResult{}, err
		}
		return PromptAnswerResult{Delivery: DeliveryKeystrokes}, nil
	}
	var chosen *backend.PromptOption
	for i := range prompt.Options {
		if prompt.Options[i].Key == answer.OptionKey {
			chosen = &prompt.Options[i]
			break
		}
	}
	if chosen == nil {
		return PromptAnswerResult{}, fmt.Errorf("option %q is not on the prompt: %w", answer.OptionKey, ErrInvalidArgument)
	}
	if activity.Harness == "opencode" {
		if replier == nil {
			return PromptAnswerResult{}, fmt.Errorf("no reply path for OpenCode: %w", ErrFailedPrecondition)
		}
		if err := replier.ReplyPermission(ctx, sessionID, chosen.Key); err != nil {
			return PromptAnswerResult{}, fmt.Errorf("OpenCode reply: %w", err)
		}
		return PromptAnswerResult{Delivery: DeliveryHarnessAPI, Answer: chosen.Label}, nil
	}
	if err := target.TypeText(chosen.Key); err != nil {
		return PromptAnswerResult{}, err
	}
	for waited := time.Duration(0); waited < settle.Max; waited += settle.Poll {
		settle.Sleep(settle.Poll)
		now, ok := target.Activity()
		if !ok || now.State != session.ActivityWaiting || backend.PromptHash(now.Prompt) != answer.PromptHash {
			return PromptAnswerResult{Delivery: DeliveryKeystrokes, Answer: chosen.Label}, nil
		}
	}
	if err := target.PressKey("Enter"); err != nil {
		return PromptAnswerResult{}, err
	}
	return PromptAnswerResult{Delivery: DeliveryKeystrokes, Answer: chosen.Label}, nil
}

// sessionAnswerTarget answers through a live session's activity and input lane.
type sessionAnswerTarget struct {
	sess *session.Session
}

func (t sessionAnswerTarget) Activity() (session.Activity, bool) {
	detector := t.sess.ActivityDetector()
	if detector == nil {
		return session.Activity{}, false
	}
	return detector.Current(), true
}

func (t sessionAnswerTarget) TypeText(text string) error {
	_, err := t.sess.SendInputCountWithKeyMap(session.InputText(text).WithSource(answerSource), DefaultKeyMap{})
	return err
}

func (t sessionAnswerTarget) PressKey(name string) error {
	_, err := t.sess.SendInputCountWithKeyMap(session.InputKeys(session.Key{Name: name}).WithSource(answerSource), DefaultKeyMap{})
	return err
}

// AnswerPrompt answers the prompt the session's agent is showing.
func (a *Adapter) AnswerPrompt(ctx context.Context, sessionID string, answer PromptAnswer) (PromptAnswerResult, error) {
	sess, err := a.lookup(sessionID)
	if err != nil {
		return PromptAnswerResult{}, err
	}
	return answerPrompt(ctx, sessionID, sessionAnswerTarget{sess: sess}, a.Replier, answer, defaultAnswerSettle)
}
