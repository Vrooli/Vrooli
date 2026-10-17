package webconsole

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"connectrpc.com/connect"

	terminalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/terminal"
)

// ErrPromptNotSubmitted means the prompt was pasted and Enter was pressed
// (with bounded retries), but the TUI's composer still visibly holds it: the
// directive sits composed-but-unsent. It is distinct from a transport error,
// whose outcome is unknown.
var ErrPromptNotSubmitted = errors.New("prompt pasted but not submitted: the composer still holds it")

// PromptSubmission reports what SendPrompt could confirm about a submit.
type PromptSubmission struct {
	// Verified is true when the composer was seen holding the prompt and then
	// seen without it after Enter. False with a nil error means the composer
	// could not be observed (unreadable screen, or a TUI whose cursor does not
	// sit in its composer): Enter was sent once, exactly as before
	// verification existed, and the outcome is unconfirmed.
	Verified bool
	// EnterPresses counts the Enter keys sent.
	EnterPresses int
}

const (
	// promptPollInterval is the screen-read cadence while confirming a submit.
	promptPollInterval = 150 * time.Millisecond
	// promptIngestBudget bounds the wait for a loaded TUI to finish ingesting
	// the paste before the first Enter. Enter sent mid-ingest is taken as a
	// newline inside the composer (observed with codex under load).
	promptIngestBudget = 5 * time.Second
	// composerWindowRows is how many screen rows, ending at the cursor row, are
	// read as the composer. It covers a wrapped prompt tail plus the blank rows
	// that swallowed Enters leave below it.
	composerWindowRows = 8
	// promptTailRunes is the distinctive prompt suffix searched for on screen.
	promptTailRunes = 32
)

// promptEnterWindows is one confirmation window per Enter attempt, growing so
// a slow render is not mistaken for a swallowed Enter.
var promptEnterWindows = []time.Duration{time.Second, 2 * time.Second, 3 * time.Second}

// pastePlaceholder matches the collapsed form claude and codex show for a
// large paste ("[Pasted text #1 +40 lines]", "[Pasted Content 1234 chars]")
// once whitespace is removed.
var pastePlaceholder = regexp.MustCompile(`\[Pasted[^\[\]]*\]$`)

// SendPrompt implements SessionController. It pastes the prompt via the PTY
// bracketed-paste path (is_paste=true, so a multi-line prompt is delivered as
// one block) and submits it with a named Enter key, which web-console's
// DefaultKeyMap resolves to the carriage return claude/codex/grok TUIs expect.
//
// Submission is verified from the screen rather than assumed after a delay.
// The composer is taken to be the few rows ending at the terminal cursor; it
// "holds" the prompt when that text, with whitespace and box-drawing glyphs
// removed (TUIs wrap and frame input), ends with the prompt's tail or with a
// large-paste placeholder. SendPrompt waits (bounded) for the composer to hold
// the prompt, presses Enter, and confirms the composer no longer holds it —
// a submitted or queued message is no longer the text ending at the cursor.
// While the composer still holds it, Enter is retried (bounded). This assumes
// only that the TUI keeps its cursor in its composer; it uses no
// runner-specific glyphs. When that cannot be observed the result is reported
// unverified, never as success.
func (c *Client) SendPrompt(ctx context.Context, sessionID, prompt, source string) (PromptSubmission, error) {
	var out PromptSubmission
	if _, err := c.terminal.SendInput(ctx, connect.NewRequest(&terminalv1.SendInputRequest{
		SessionId: sessionID,
		Body:      &terminalv1.SendInputRequest_Text{Text: prompt},
		Source:    source,
		IsPaste:   true,
	})); err != nil {
		return out, fmt.Errorf("web-console paste prompt: %w", err)
	}

	tail := promptTail(prompt)
	seen := false
	if tail != "" {
		for range pollCount(promptIngestBudget) {
			holds, readable := c.composerHolds(ctx, sessionID, tail)
			if !readable {
				break
			}
			if holds {
				seen = true
				break
			}
			if err := c.pause(ctx); err != nil {
				return out, err
			}
		}
	}

	for _, window := range promptEnterWindows {
		if _, err := c.terminal.SendInput(ctx, connect.NewRequest(&terminalv1.SendInputRequest{
			SessionId: sessionID,
			Body: &terminalv1.SendInputRequest_Keys{Keys: &terminalv1.KeySequence{
				Keys: []*terminalv1.Key{{Name: "enter"}},
			}},
			Source: source,
		})); err != nil {
			return out, fmt.Errorf("web-console submit prompt: %w", err)
		}
		out.EnterPresses++
		if tail == "" {
			return out, nil
		}
		for range pollCount(window) {
			if err := c.pause(ctx); err != nil {
				return out, err
			}
			holds, readable := c.composerHolds(ctx, sessionID, tail)
			if !readable {
				return out, nil
			}
			if holds {
				seen = true
				continue
			}
			if seen {
				out.Verified = true
				return out, nil
			}
		}
		if !seen {
			// The composer never showed the prompt: nothing to confirm against.
			return out, nil
		}
		// Still composed at the end of the window: Enter was swallowed.
	}
	return out, ErrPromptNotSubmitted
}

// composerHolds reports whether the composer (the rows ending at the cursor)
// ends with tail or a large-paste placeholder. readable is false when the
// screen cannot be read or carries no usable cursor.
func (c *Client) composerHolds(ctx context.Context, sessionID, tail string) (holds, readable bool) {
	resp, err := c.terminal.GetScreen(ctx, connect.NewRequest(&terminalv1.GetScreenRequest{SessionId: sessionID}))
	if err != nil || resp.Msg.GetCursor() == nil {
		return false, false
	}
	rows := strings.Split(resp.Msg.GetPlainText(), "\n")
	y := int(resp.Msg.GetCursor().GetY())
	if y < 0 || y >= len(rows) {
		return false, false
	}
	composer := normalizeScreenText(strings.Join(rows[max(0, y-composerWindowRows+1):y+1], "\n"))
	return strings.HasSuffix(composer, tail) || pastePlaceholder.MatchString(composer), true
}

func (c *Client) pause(ctx context.Context) error {
	if c.wait != nil {
		return c.wait(ctx, promptPollInterval)
	}
	timer := time.NewTimer(promptPollInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func pollCount(budget time.Duration) int {
	return max(1, int(budget/promptPollInterval))
}

// promptTail is the last promptTailRunes of the normalized prompt.
func promptTail(prompt string) string {
	normalized := []rune(normalizeScreenText(prompt))
	return string(normalized[max(0, len(normalized)-promptTailRunes):])
}

// normalizeScreenText removes whitespace and box-drawing/block glyphs so text
// compares equal however a TUI wrapped, indented, or framed it.
func normalizeScreenText(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || (r >= 0x2500 && r <= 0x259F) {
			return -1
		}
		return r
	}, s)
}
