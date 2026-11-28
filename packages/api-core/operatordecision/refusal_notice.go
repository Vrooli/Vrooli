package operatordecision

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/vrooli/api-core/hubsender"
	"github.com/vrooli/api-core/serviceprincipal"
	"github.com/vrooli/api-core/servicetoken"
)

// NoticeSender sends the operator a notice through Notification Hub.
// *hubsender.Client, built for the scenario's own sender principal, is the
// production implementation.
type NoticeSender interface {
	Send(ctx context.Context, notice hubsender.Notice) error
}

const (
	// DefaultNoticeBurst is how many refusals of one scenario are reported one by
	// one inside a window before the rest are folded into one summary.
	DefaultNoticeBurst = 5
	// DefaultNoticeWindow is how long that count runs.
	DefaultNoticeWindow = 10 * time.Minute
	// noticeSendTimeout bounds the Hub call a refusal waits for.
	noticeSendTimeout = 5 * time.Second
	// maxNoticeQuestion bounds the ask text quoted in a notice, in characters.
	maxNoticeQuestion = 100
	// maxRememberedRefusals bounds the refusals remembered as reported. Forgetting
	// them is safe: the Hub applies the same idempotency key to the same notice.
	maxRememberedRefusals = 1024
)

// RefusalNoticeConfig configures a RefusalNotifier.
type RefusalNoticeConfig struct {
	// Source is the scenario whose webhook refused the resolutions.
	Source string
	// Sender sends the notice. Without one the notifier only logs.
	Sender NoticeSender
	// MaxPerWindow is how many refusals are reported individually per Window.
	// Zero means DefaultNoticeBurst.
	MaxPerWindow int
	// Window is the burst window. Zero means DefaultNoticeWindow.
	Window time.Duration
	// AskLink is the context link of the notice: where the operator opens the ask,
	// as the Hub serves it. Nil means "/asks/<id>".
	AskLink func(askID string) string
	// Logf receives what could not be sent. log.Printf by default.
	Logf func(format string, args ...any)
	// Now replaces the clock.
	Now func() time.Time
}

// RefusalNoticeStats counts what a RefusalNotifier did, since it started.
type RefusalNoticeStats struct {
	// Sent are notices about one refused resolution.
	Sent uint64
	// Summaries are notices that folded a burst of refusals into one.
	Summaries uint64
	// Suppressed refusals were not reported on their own: they came after the
	// burst limit of their window.
	Suppressed uint64
	// Failed sends. A refusal whose notice failed is reported again when the bus
	// redelivers it.
	Failed uint64
}

// RefusalNotifier tells the operator that something tried to apply an answer they
// did not give. A scenario passes Notify as WebhookConfig.OnRefused.
//
// A refused resolution is a security event, not a retry: it must reach the
// operator once, and a forger spraying asks must not flood their phone. The
// notifier therefore sends one notice per scenario, ask, reason and question,
// whatever number of times the bus redelivers the refusal (its idempotency key
// makes that hold across restarts too), and after MaxPerWindow notices in a
// window sends a single summary instead of the rest. The text names the ask by
// the question the Hub holds, never by anything the event body says.
type RefusalNotifier struct {
	cfg RefusalNoticeConfig

	mu         sync.Mutex
	reported   map[string]struct{}
	window     time.Time
	sent       int
	summarized bool
	stats      RefusalNoticeStats
}

// NewRefusalNotifier builds the notifier.
func NewRefusalNotifier(cfg RefusalNoticeConfig) *RefusalNotifier {
	if cfg.MaxPerWindow <= 0 {
		cfg.MaxPerWindow = DefaultNoticeBurst
	}
	if cfg.Window <= 0 {
		cfg.Window = DefaultNoticeWindow
	}
	if cfg.AskLink == nil {
		cfg.AskLink = func(askID string) string { return "/asks/" + askID }
	}
	if cfg.Logf == nil {
		cfg.Logf = log.Printf
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &RefusalNotifier{cfg: cfg, reported: map[string]struct{}{}}
}

// Stats is what the notifier has done since it started.
func (n *RefusalNotifier) Stats() RefusalNoticeStats {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.stats
}

// Notify reports one refusal. It is WebhookConfig.OnRefused.
func (n *RefusalNotifier) Notify(ctx context.Context, refusal Refusal) {
	key := n.refusalKey(refusal)
	now := n.cfg.Now()

	n.mu.Lock()
	if _, done := n.reported[key]; done {
		n.mu.Unlock()
		return
	}
	if n.window.IsZero() || now.Sub(n.window) >= n.cfg.Window {
		n.window, n.sent, n.summarized = now, 0, false
	}
	var notice hubsender.Notice
	summary := false
	switch {
	case n.sent < n.cfg.MaxPerWindow:
		n.sent++
		if len(n.reported) >= maxRememberedRefusals {
			n.reported = map[string]struct{}{}
		}
		n.reported[key] = struct{}{}
		notice = n.refusalNotice(key, refusal)
	case !n.summarized:
		n.summarized, summary = true, true
		n.stats.Suppressed++
		notice = n.summaryNotice()
	default:
		n.stats.Suppressed++
		n.mu.Unlock()
		return
	}
	window := n.window
	n.mu.Unlock()

	if summary {
		notice.IdempotencyKey = fmt.Sprintf("refused-resolution-burst:%s:%d", n.cfg.Source, window.Unix())
	}
	err := n.send(ctx, notice)

	n.mu.Lock()
	defer n.mu.Unlock()
	switch {
	case err == nil && summary:
		n.stats.Summaries++
	case err == nil:
		n.stats.Sent++
	default:
		n.stats.Failed++
		if summary {
			n.summarized = false
		} else {
			delete(n.reported, key)
			n.sent--
		}
		n.cfg.Logf("operatordecision: could not tell the operator about a refused resolution for ask %s: %v", refusal.AskID, err)
	}
}

func (n *RefusalNotifier) send(ctx context.Context, notice hubsender.Notice) error {
	if n.cfg.Sender == nil {
		return errors.New("no notice sender is configured")
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), noticeSendTimeout)
	defer cancel()
	err := n.cfg.Sender.Send(ctx, notice)
	if errors.Is(err, servicetoken.ErrNotIssued) {
		return fmt.Errorf("%s has no notification sender credential yet; the operator closes this with `%s`: %w", n.cfg.Source, serviceprincipal.IssueCommand, err)
	}
	return err
}

// refusalKey names one refusal. The question is part of it so that an open ask a
// producer revised is a new refusal, not a repeat with different terms, which the
// Hub's idempotency would refuse.
func (n *RefusalNotifier) refusalKey(r Refusal) string {
	sum := sha256.Sum256([]byte(tidy(r.Question)))
	return fmt.Sprintf("refused-resolution:%s:%s:%s:%s", n.cfg.Source, r.AskID, r.Reason, hex.EncodeToString(sum[:4]))
}

var refusalReasonText = map[string]string{
	ReasonMismatch: "the answer or another field in the event differs from the Hub's record of the ask",
	ReasonAskOpen:  "the event says the ask is closed, and the Hub still has it open",
}

func (n *RefusalNotifier) refusalNotice(key string, r Refusal) hubsender.Notice {
	subject := "ask " + r.AskID
	if question := truncate(tidy(r.Question), maxNoticeQuestion); question != "" {
		subject = "\"" + question + "\""
	}
	reason := refusalReasonText[r.Reason]
	if reason == "" {
		reason = "the Hub does not confirm it (" + r.Reason + ")"
	}
	return hubsender.Notice{
		Title: "An answer you didn't give was refused",
		Body: fmt.Sprintf("Something tried to apply an answer you didn't give for %s. It was refused; nothing changed.\n\nTechnical details: %s refused a resolution of ask %s because %s.",
			subject, n.cfg.Source, r.AskID, reason),
		ContextURL:       n.cfg.AskLink(r.AskID),
		Urgency:          "high",
		SensitivityLabel: "private",
		IdempotencyKey:   key,
	}
}

func (n *RefusalNotifier) summaryNotice() hubsender.Notice {
	return hubsender.Notice{
		Title: "Many answers you didn't give were refused",
		Body: fmt.Sprintf("Something keeps trying to apply answers you didn't give to %s. The first %d were reported on their own; the rest in this period are refused the same way and are not reported one by one. Nothing changed.\n\nTechnical details: %s refused more than %d resolutions within %s that the Hub's records contradict; its log and webhook counters list them.",
			n.cfg.Source, n.cfg.MaxPerWindow, n.cfg.Source, n.cfg.MaxPerWindow, n.cfg.Window),
		Urgency:          "high",
		SensitivityLabel: "private",
	}
}

// tidy makes ask text one line of printable characters.
func tidy(text string) string {
	return strings.Join(strings.FieldsFunc(text, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }), " ")
}

func truncate(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return strings.TrimSpace(string(runes[:limit])) + "…"
}
