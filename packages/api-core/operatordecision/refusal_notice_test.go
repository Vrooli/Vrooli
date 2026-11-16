package operatordecision

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/hubsender"
	"github.com/vrooli/api-core/serviceprincipal"
	"github.com/vrooli/api-core/servicetoken"
)

const genuineQuestion = "Delete the staging backups older than 30 days?"

// noticeLog is a NoticeSender that records what the operator would be sent.
type noticeLog struct {
	mu      sync.Mutex
	notices []hubsender.Notice
	err     error
}

func (l *noticeLog) Send(_ context.Context, notice hubsender.Notice) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.err != nil {
		return l.err
	}
	// The Hub applies one idempotency key to one notice, whatever the sends.
	for _, sent := range l.notices {
		if sent.IdempotencyKey == notice.IdempotencyKey && notice.IdempotencyKey != "" {
			return nil
		}
	}
	l.notices = append(l.notices, notice)
	return nil
}

func (l *noticeLog) setErr(err error) {
	l.mu.Lock()
	l.err = err
	l.mu.Unlock()
}

func (l *noticeLog) sent() []hubsender.Notice {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]hubsender.Notice(nil), l.notices...)
}

type noticeFixture struct {
	*verifyFixture
	notices  *noticeLog
	notifier *RefusalNotifier
	now      time.Time
	logs     []string
}

func newNoticeFixture(t *testing.T, asks ...askreader.Ask) *noticeFixture {
	t.Helper()
	f := &noticeFixture{verifyFixture: newVerifyFixture(t, asks...), notices: &noticeLog{}, now: time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)}
	f.notifier = NewRefusalNotifier(RefusalNoticeConfig{
		Source: "storage-manager", Sender: f.notices, Now: func() time.Time { return f.now },
		Logf: func(format string, args ...any) { f.logs = append(f.logs, fmt.Sprintf(format, args...)) },
	})
	f.webhook.cfg.OnRefused = f.notifier.Notify
	return f
}

// questionAsk is the Hub's record of an ask with its own question.
func questionAsk(id string, mutate func(*askreader.Ask)) askreader.Ask {
	ask := answeredGrant()
	ask.ID, ask.Question = id, genuineQuestion
	ask.RequesterRef, ask.SubjectKey = "evt-"+id, "grant:"+id
	if mutate != nil {
		mutate(&ask)
	}
	return ask
}

// forgeAnswer is a body that claims the operator approved ask.
func forgeAnswer(ask askreader.Ask) map[string]any {
	claim := claimOf(ask)
	claim["answer"], claim["answered_by"] = "approve-everything", "mallory"
	claim["question"] = "Totally routine, nothing to see here"
	return claim
}

func TestARefusedResolutionIsReportedOnceAcrossSixRedeliveries(t *testing.T) {
	ask := questionAsk("ask-1", nil)
	f := newNoticeFixture(t, ask)
	for range 6 {
		if rec := f.post(AskResolvedEventType, forgeAnswer(ask)); rec.Code != http.StatusConflict {
			t.Fatalf("status %d, want 409 on every delivery", rec.Code)
		}
	}
	sent := f.notices.sent()
	if len(sent) != 1 {
		t.Fatalf("%d notices for 6 redeliveries of one refusal, want 1", len(sent))
	}
	if stats := f.notifier.Stats(); stats.Sent != 1 || stats.Suppressed != 0 || stats.Failed != 0 {
		t.Fatalf("stats = %+v", stats)
	}
	// A different reason on the same ask is a different event: the pending ask.
	pending := ask
	pending.State, pending.Answer, pending.AnsweredBy = "pending", "", ""
	f.hub.asks[ask.ID] = pending
	f.post(AskResolvedEventType, claimOf(ask))
	if got := len(f.notices.sent()); got != 2 {
		t.Fatalf("%d notices after a second reason on the ask, want 2", got)
	}
}

// What the operator reads is the Hub's record, never a field of the body a forger
// controls: not its question, not the answer it claims.
func TestTheNoticeTextComesFromTheHubsRecordAndSaysWhatItMeans(t *testing.T) {
	ask := questionAsk("ask-1", nil)
	f := newNoticeFixture(t, ask)
	f.post(AskResolvedEventType, forgeAnswer(ask))
	sent := f.notices.sent()
	if len(sent) != 1 {
		t.Fatalf("notices = %d", len(sent))
	}
	n := sent[0]
	for _, want := range []string{
		"Something tried to apply an answer you didn't give for \"" + genuineQuestion + "\". It was refused; nothing changed.",
		"Technical details: storage-manager refused a resolution of ask ask-1",
	} {
		if !strings.Contains(n.Body, want) {
			t.Errorf("body lacks %q:\n%s", want, n.Body)
		}
	}
	for _, forged := range []string{"Totally routine", "approve-everything", "mallory"} {
		if strings.Contains(n.Title+n.Body, forged) {
			t.Errorf("the notice repeats %q from the event body", forged)
		}
	}
	if n.ContextURL != "/asks/ask-1" {
		t.Errorf("context link = %q, want the ask's page on the Hub", n.ContextURL)
	}
	if n.Urgency != "high" || n.SensitivityLabel != "private" || n.Title == "" {
		t.Fatalf("notice = %+v, want high urgency, normal (private) sensitivity and a title", n)
	}
	if !strings.HasPrefix(n.IdempotencyKey, "refused-resolution:storage-manager:ask-1:mismatch:") {
		t.Fatalf("idempotency key = %q", n.IdempotencyKey)
	}
}

func TestALongOrMultiLineQuestionIsQuotedTruncatedOnOneLine(t *testing.T) {
	ask := questionAsk("ask-1", func(a *askreader.Ask) {
		a.Question = "Delete\nthe\tstaging backups?  " + strings.Repeat("very long ", 40)
	})
	f := newNoticeFixture(t, ask)
	f.post(AskResolvedEventType, forgeAnswer(ask))
	body := f.notices.sent()[0].Body
	first := strings.SplitN(body, "\n", 2)[0]
	if !strings.Contains(first, "\"Delete the staging backups? very long") || !strings.Contains(first, "…\"") || len([]rune(first)) > 260 {
		t.Fatalf("first line = %q", first)
	}
}

func TestAnAskWithoutAQuestionIsNamedById(t *testing.T) {
	ask := questionAsk("ask-9", func(a *askreader.Ask) { a.Question = "" })
	f := newNoticeFixture(t, ask)
	f.post(AskResolvedEventType, forgeAnswer(ask))
	if body := f.notices.sent()[0].Body; !strings.Contains(body, "for ask ask-9. It was refused") {
		t.Fatalf("body = %q", body)
	}
}

// A revised open ask has new terms; reusing the old key would be refused by the
// Hub for different immutable terms, so it is its own refusal.
func TestAnAskRevisedBetweenRefusalsIsReportedAgain(t *testing.T) {
	ask := questionAsk("ask-1", nil)
	f := newNoticeFixture(t, ask)
	f.post(AskResolvedEventType, forgeAnswer(ask))
	revised := ask
	revised.Question = "Delete ALL staging backups?"
	f.hub.asks[ask.ID] = revised
	f.post(AskResolvedEventType, forgeAnswer(ask))
	sent := f.notices.sent()
	if len(sent) != 2 || sent[0].IdempotencyKey == sent[1].IdempotencyKey {
		t.Fatalf("notices = %+v", sent)
	}
}

// A forger spraying asks gets the operator one summary, not a flood.
func TestABurstOfRefusalsFoldsIntoOneSummaryPerWindow(t *testing.T) {
	asks := make([]askreader.Ask, 12)
	for i := range asks {
		asks[i] = questionAsk(fmt.Sprintf("ask-%d", i), nil)
	}
	f := newNoticeFixture(t, asks...)
	for _, ask := range asks {
		f.post(AskResolvedEventType, forgeAnswer(ask))
	}
	sent := f.notices.sent()
	if len(sent) != DefaultNoticeBurst+1 {
		t.Fatalf("%d notices for 12 refusals, want %d individual plus one summary", len(sent), DefaultNoticeBurst)
	}
	summary := sent[len(sent)-1]
	if !strings.HasPrefix(summary.IdempotencyKey, "refused-resolution-burst:storage-manager:") || !strings.Contains(summary.Title, "Many answers") ||
		!strings.Contains(summary.Body, "nothing changed") && !strings.Contains(summary.Body, "Nothing changed") {
		t.Fatalf("summary = %+v", summary)
	}
	if stats := f.notifier.Stats(); stats.Sent != DefaultNoticeBurst || stats.Summaries != 1 || stats.Suppressed != 12-DefaultNoticeBurst {
		t.Fatalf("stats = %+v", stats)
	}
	// Redelivering a suppressed refusal in the same window sends nothing more.
	f.post(AskResolvedEventType, forgeAnswer(asks[11]))
	if len(f.notices.sent()) != DefaultNoticeBurst+1 {
		t.Fatalf("a redelivery inside the window sent another notice")
	}
	// A new window reports again, one by one, and a suppressed refusal gets its turn.
	f.now = f.now.Add(DefaultNoticeWindow + time.Second)
	f.post(AskResolvedEventType, forgeAnswer(asks[11]))
	if got := len(f.notices.sent()); got != DefaultNoticeBurst+2 {
		t.Fatalf("%d notices after the window passed, want %d", got, DefaultNoticeBurst+2)
	}
}

func TestAnOutageOrAnUnheldAskOrAContextRequestNeverNotifies(t *testing.T) {
	ask := questionAsk("ask-1", nil)
	f := newNoticeFixture(t, ask)

	f.hub.setErr(fmt.Errorf("%w: down", askreader.ErrHubUnavailable))
	if rec := f.post(AskResolvedEventType, forgeAnswer(ask)); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("outage: status %d", rec.Code)
	}
	f.hub.setErr(nil)

	unheld := questionAsk("ask-unheld", nil)
	if rec := f.post(AskResolvedEventType, forgeAnswer(unheld)); rec.Code != http.StatusConflict {
		t.Fatalf("unheld ask: status %d", rec.Code)
	}

	// Nor does a genuine answer.
	if rec := f.post(AskResolvedEventType, claimOf(ask)); rec.Code != http.StatusOK {
		t.Fatalf("genuine: status %d", rec.Code)
	}
	if sent := f.notices.sent(); len(sent) != 0 {
		t.Fatalf("notices = %+v, want none", sent)
	}
}

// A forged context request is refused (409) and counted, but nothing the operator
// did not say is applied by it, so it is not reported as a refused answer.
func TestARefusedContextRequestNeverNotifies(t *testing.T) {
	f, asked := newContextFixture(t)
	notices := &noticeLog{}
	f.webhook.cfg.OnRefused = NewRefusalNotifier(RefusalNoticeConfig{Source: "storage-manager", Sender: notices}).Notify
	claim := contextClaim()
	claim["question"] = "Delete everything?"
	if rec := f.post(AskContextRequestedEventType, claim); rec.Code != http.StatusConflict || len(*asked) != 0 {
		t.Fatalf("status=%d asked=%d", rec.Code, len(*asked))
	}
	if len(notices.sent()) != 0 {
		t.Fatalf("notices = %+v", notices.sent())
	}
}

func TestAResolutionAppliedWithoutACredentialNeverNotifies(t *testing.T) {
	ask := questionAsk("ask-1", nil)
	f := newNoticeFixture(t, ask)
	f.hub.setErr(fmt.Errorf("%w: no credential", askreader.ErrNotIssued))
	f.post(AskResolvedEventType, forgeAnswer(ask))
	if len(f.notices.sent()) != 0 {
		t.Fatal("no refusal happened, so there is nothing to report")
	}
}

// If the Hub cannot take the notice, the refusal is reported when the bus
// redelivers it: nothing is marked as reported that the operator never saw.
func TestARefusalWhoseNoticeFailedIsReportedOnRedelivery(t *testing.T) {
	ask := questionAsk("ask-1", nil)
	f := newNoticeFixture(t, ask)
	f.notices.setErr(fmt.Errorf("%w: hub down", hubsender.ErrSendFailed))
	if rec := f.post(AskResolvedEventType, forgeAnswer(ask)); rec.Code != http.StatusConflict {
		t.Fatalf("a failed notice must not change the refusal: status %d", rec.Code)
	}
	if stats := f.notifier.Stats(); stats.Failed != 1 || stats.Sent != 0 {
		t.Fatalf("stats = %+v", stats)
	}
	f.notices.setErr(nil)
	f.post(AskResolvedEventType, forgeAnswer(ask))
	if len(f.notices.sent()) != 1 || f.notifier.Stats().Sent != 1 {
		t.Fatalf("notices = %d, stats = %+v", len(f.notices.sent()), f.notifier.Stats())
	}
	// Failures do not use up the burst budget.
	for i := range 2 * DefaultNoticeBurst {
		f.notices.setErr(fmt.Errorf("%w: hub down", hubsender.ErrSendFailed))
		other := questionAsk(fmt.Sprintf("f-%d", i), nil)
		f.hub.asks[other.ID] = other
		f.post(AskResolvedEventType, forgeAnswer(other))
	}
	f.notices.setErr(nil)
	late := questionAsk("late", nil)
	f.hub.asks[late.ID] = late
	f.post(AskResolvedEventType, forgeAnswer(late))
	if len(f.notices.sent()) != 2 {
		t.Fatalf("a refusal after failed sends must still be reported individually: %d notices", len(f.notices.sent()))
	}
}

func TestAScenarioWithoutASenderCredentialLogsTheOperatorsCommand(t *testing.T) {
	ask := questionAsk("ask-1", nil)
	f := newNoticeFixture(t, ask)
	f.notices.setErr(servicetoken.ErrNotIssued)
	f.post(AskResolvedEventType, forgeAnswer(ask))
	if len(f.logs) != 1 || !strings.Contains(f.logs[0], serviceprincipal.IssueCommand) || !strings.Contains(f.logs[0], "ask-1") {
		t.Fatalf("logs = %q", f.logs)
	}
}

func TestANotifierWithNoSenderOnlyLogs(t *testing.T) {
	var logs []string
	n := NewRefusalNotifier(RefusalNoticeConfig{Source: "storage-manager", Logf: func(f string, a ...any) { logs = append(logs, fmt.Sprintf(f, a...)) }})
	n.Notify(context.Background(), Refusal{Source: "storage-manager", AskID: "a", Reason: ReasonMismatch})
	if len(logs) != 1 || !strings.Contains(logs[0], "no notice sender") {
		t.Fatalf("logs = %q", logs)
	}
}
