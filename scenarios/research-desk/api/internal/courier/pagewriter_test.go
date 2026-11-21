package courier

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/decisionpage"

	"research-desk/internal/home"
)

// A decision whose raiser left its page out waits for the shared
// decision-page writer and goes with the page it makes; the operator answers
// in hours, so the minutes a page takes cost nothing. Until then, and when the
// writer gives up or is down, the item stays packet_not_ready: no page
// restating the item reaches the operator. A writer that gave up is recorded
// and reported as a page miss.

const writtenPage = `<!doctype html><html><body><h1>Keeping it uninstalled costs nothing</h1>
<button type="button" data-answer="keep">Choose this</button><button type="button" data-answer="propose">Choose this</button></body></html>`

// fakeWriter is the decision-page writer: jobs by subject key, and what a
// pass's Advance makes of one.
type fakeWriter struct {
	jobs     map[string]decisionpage.Result
	advanced []decisionpage.Request
	// next is the job after an Advance; nil leaves a started run working.
	next func(decisionpage.Request) decisionpage.Result
}

func (w *fakeWriter) Status(r decisionpage.Request) (decisionpage.Result, error) {
	return w.jobs[r.SubjectKey+"@"+r.Digest()], nil
}

func (w *fakeWriter) Advance(_ context.Context, r decisionpage.Request) (decisionpage.Result, error) {
	w.advanced = append(w.advanced, r)
	res := decisionpage.Result{State: decisionpage.StateWorking, Dir: "/jobs/av4", Attempts: []decisionpage.Attempt{{N: 1, RunID: "run-1", Model: "gpt-6-luna", StartedAt: time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)}}}
	if w.next != nil {
		res = w.next(r)
	}
	w.jobs[r.SubjectKey+"@"+r.Digest()] = res
	return res, nil
}

// set makes the job for the last request the writer was advanced with.
func (w *fakeWriter) set(t *testing.T, res decisionpage.Result) {
	t.Helper()
	if len(w.advanced) == 0 {
		t.Fatal("the writer was never asked for a page")
	}
	r := w.advanced[len(w.advanced)-1]
	w.jobs[r.SubjectKey+"@"+r.Digest()] = res
}

// withoutPage gives the rig a writer and takes AV4's page away.
func (r *rig) withoutPage(t *testing.T) *fakeWriter {
	t.Helper()
	w := &fakeWriter{jobs: map[string]decisionpage.Result{}}
	r.courier.PageWriter = w
	if err := os.Remove(filepath.Join(r.av4Folder(), "page.html")); err != nil {
		t.Fatal(err)
	}
	return w
}

func ready(page string) decisionpage.Result {
	sum := sha256.Sum256([]byte(page))
	return decisionpage.Result{State: decisionpage.StateReady, Dir: "/jobs/av4", Page: "/jobs/av4/page.html", HTML: page, SHA256: hex.EncodeToString(sum[:])}
}

func TestADecisionWithoutItsPageWaitsForTheWriterThenGoesWithItsPage(t *testing.T) {
	r := newRig(t, "desk")
	w := r.withoutPage(t)
	s := r.step(r.pass(t), av4)
	if len(r.hub.raised) != 0 || s.Action != ActWritePage || s.Reason != home.ReasonPacketNotReady || !s.Done || !strings.Contains(s.Why, "waiting for its page from the decision-page writer") {
		t.Fatalf("the ask waits while the writer starts: %+v, raised %d", s, len(r.hub.raised))
	}
	if len(w.advanced) != 1 {
		t.Fatalf("one pass asks the writer once: %d", len(w.advanced))
	}
	req := w.advanced[0]
	brief := filepath.Join(r.av4Folder(), home.BriefFile)
	if req.SubjectKey != av4 || req.Recommended != "keep" || len(req.Options) != 2 || req.Options[0].Effect != "settles it this way." || len(req.Evidence) == 0 || req.Evidence[0] != brief || req.IfNoAnswer != noAnswer {
		t.Fatalf("the writer gets the item, what each answer executes, and its brief first: %+v", req)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("the desk's request is one the writer accepts: %v", err)
	}

	s = r.step(r.pass(t), av4)
	if len(r.hub.raised) != 0 || s.Action != ActWritePage || !strings.Contains(s.Why, "attempt 1 (agent run run-1, gpt-6-luna)") {
		t.Fatalf("while the run works the ask still waits, and says on what: %+v", s)
	}
	advanced := len(w.advanced)
	if _, _, err := r.courier.Plan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(w.advanced) != advanced {
		t.Fatal("the dry run only reads the writer's job")
	}

	w.set(t, ready(writtenPage))
	s = r.step(r.pass(t), av4)
	if len(r.hub.raised) != 1 || r.hub.raised[0].Explainer != writtenPage {
		t.Fatalf("once ready, the ask goes with the written page: %+v", r.hub.raised)
	}
	if !strings.Contains(r.hub.raised[0].TechnicalDetails, "Page: made by the decision-page writer, /jobs/av4/page.html (sha256 "+ready(writtenPage).SHA256[:16]+")") {
		t.Fatalf("the technical details name the written page and its hash:\n%s", r.hub.raised[0].TechnicalDetails)
	}
	if !strings.Contains(s.Why, "sent with the page the decision-page writer made") {
		t.Fatalf("the plan row says whose page went: %q", s.Why)
	}
	r.pass(t)
	if len(r.hub.raised) != 1 {
		t.Fatalf("an unchanged item with its written page is not sent again: %d asks", len(r.hub.raised))
	}
}

// A revised brief is a new page: the open ask keeps its page until the new
// one is ready, never falling back to the summary page in between.
func TestARevisedBriefWaitsForANewPageWithoutReplacingTheOpenAsk(t *testing.T) {
	r := newRig(t, "desk")
	w := r.withoutPage(t)
	r.pass(t)
	w.set(t, ready(writtenPage))
	r.pass(t)
	if len(r.hub.raised) != 1 {
		t.Fatalf("sent once with the written page: %d", len(r.hub.raised))
	}
	r.write(t, filepath.Join(r.av4Folder(), home.BriefFile), append(readyBrief("keep", "propose"), []byte("\n## Commands\n\nNone.\n")...))
	s := r.step(r.pass(t), av4)
	if s.Action != ActWritePage || len(r.hub.raised) != 1 || len(w.advanced) != 2 {
		t.Fatalf("the revised brief asks for a new page while the open ask stays: %+v, raised %d, advanced %d", s, len(r.hub.raised), len(w.advanced))
	}
	revised := strings.Replace(writtenPage, "costs nothing", "still costs nothing", 1)
	w.set(t, ready(revised))
	r.pass(t)
	if len(r.hub.raised) != 2 || r.hub.raised[1].Explainer != revised || r.hub.open(av4) != 1 {
		t.Fatalf("the new page replaces the open ask's: %+v", r.hub.raised)
	}
}

func TestAWriterThatGaveUpHoldsTheDecisionAndRecordsAMiss(t *testing.T) {
	r := newRig(t, "desk")
	w := r.withoutPage(t)
	r.pass(t)
	w.set(t, decisionpage.Result{State: decisionpage.StateFailed, Dir: "/jobs/av4", Failure: &decisionpage.Failure{Class: decisionpage.FailLint, Detail: "the second page still failed the Hub's check"}})
	s := r.step(r.pass(t), av4)
	if len(r.hub.raised) != 0 || s.Action != ActNone || s.Reason != home.ReasonPacketNotReady {
		t.Fatalf("no page restating the item reaches the operator: %+v, raised %+v", s, r.hub.raised)
	}
	if !strings.Contains(s.Why, "the decision-page writer could not make its page") || !strings.Contains(s.Why, "lint_failed") || !strings.Contains(s.Why, "work/AV4/page.html") {
		t.Fatalf("the plan row says the page was missed, why, and how to unblock it: %q", s.Why)
	}
	misses := map[string]PageMiss{}
	if _, err := r.courier.Writer.ReadState(PageMissesFile, &misses); err != nil {
		t.Fatal(err)
	}
	miss, ok := misses[commission+"/AV4"]
	if !ok || miss.Class != decisionpage.FailLint || miss.Ask != av4 || miss.Job != "/jobs/av4" {
		t.Fatalf("the miss is recorded: %+v", misses)
	}
	if ok, said := r.courier.Health().Say(); !ok || !strings.Contains(said, "1 decision(s) held unsent because the decision-page writer could not make their page") {
		t.Fatalf("the courier's health reports the miss: %v %q", ok, said)
	}
	first := misses[commission+"/AV4"].At
	r.clock.Advance(time.Hour)
	r.pass(t)
	if _, err := r.courier.Writer.ReadState(PageMissesFile, &misses); err != nil || !misses[commission+"/AV4"].At.Equal(first) {
		t.Fatalf("a miss already recorded keeps its first time: %+v %v", misses, err)
	}
	r.ready(t, r.av4Folder(), "keep", "propose")
	r.pass(t)
	if len(r.hub.raised) != 1 || r.hub.raised[0].Explainer != page {
		t.Fatalf("the raiser's own page always wins: %+v", r.hub.raised)
	}
	if _, said := r.courier.Health().Say(); strings.Contains(said, "held unsent") {
		t.Fatalf("a decision sent with its page is no longer reported as held: %q", said)
	}
}

func TestAWriterOutageKeepsTheAskWaiting(t *testing.T) {
	r := newRig(t, "desk")
	w := r.withoutPage(t)
	w.next = func(decisionpage.Request) decisionpage.Result {
		return decisionpage.Result{State: decisionpage.StateFailed, Dir: "/jobs/av4", Failure: &decisionpage.Failure{Class: decisionpage.FailUnavailable, Detail: "starting the writer run: agent-manager unreachable"}}
	}
	r.pass(t)
	s := r.step(r.pass(t), av4)
	if len(r.hub.raised) != 0 || s.Action != ActWritePage || s.Reason != home.ReasonPacketNotReady || !strings.Contains(s.Why, "agent-manager unreachable") {
		t.Fatalf("an outage is not a miss: the ask waits and says why: %+v", s)
	}
}
