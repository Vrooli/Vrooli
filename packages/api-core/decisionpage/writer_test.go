package decisionpage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeRuns stands in for Agent Manager: each launch is a run the test ends
// by writing (or not writing) the page the run would have written.
type fakeRuns struct {
	launched []LaunchSpec
	status   map[string]RunRef
	refuse   error
	stopped  []string
	stopErr  error
}

func (f *fakeRuns) Launch(_ context.Context, spec LaunchSpec) (RunRef, error) {
	if f.refuse != nil {
		return RunRef{}, f.refuse
	}
	f.launched = append(f.launched, spec)
	id := fmt.Sprintf("run-%d", len(f.launched))
	ref := RunRef{ID: id, TaskID: "task", Status: "running", Model: "gpt-6-luna"}
	if f.status == nil {
		f.status = map[string]RunRef{}
	}
	f.status[id] = ref
	return ref, nil
}

func (f *fakeRuns) Run(_ context.Context, id string) (RunRef, error) { return f.status[id], nil }

func (f *fakeRuns) Stop(_ context.Context, id string) error {
	f.stopped = append(f.stopped, id)
	return f.stopErr
}

func (f *fakeRuns) end(id string) {
	r := f.status[id]
	r.Status, r.Ended, r.Succeeded = "complete", true, true
	f.status[id] = r
}

// fakeHub answers the packet check with findings per page content.
type fakeHub struct {
	findings func(html string) []Finding
	err      error
	calls    int
}

func (h *fakeHub) Check(_ context.Context, _ Request, html []byte) ([]Finding, error) {
	h.calls++
	if h.err != nil {
		return nil, h.err
	}
	if h.findings == nil {
		return nil, nil
	}
	return h.findings(string(html)), nil
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func request() Request {
	return Request{
		Sender: "Research Desk", SubjectKey: "research-desk/topic:standing-teams/ST-POLICY-BATCH",
		Question:    "Take the eight standing-team policy recommendations?",
		Options:     []Option{{Key: "accept", Label: "Take all eight"}, {Key: "hold", Label: "Keep the defaults"}},
		Recommended: "accept", RecommendationReason: "They remove most repeat asks.",
		Evidence: []string{"/research/topics/standing-teams/work/ST-POLICY-BATCH/BRIEF.md"},
	}
}

const goodPage = `<!doctype html><html><head><meta name="viewport" content="width=device-width, initial-scale=1"></head><body>
<button type="button" data-answer="accept">Choose this</button><button type="button" data-answer="hold">Choose this</button>
<script>document.querySelectorAll('[data-answer]').forEach(b => b.addEventListener('click', () => window.hub.prefill('answer', b.dataset.answer)));</script></body></html>`

func newWriter(t *testing.T) (*Writer, *fakeRuns, *fakeHub, *clock) {
	t.Helper()
	runs, hub := &fakeRuns{}, &fakeHub{}
	c := &clock{t: time.Date(2026, 11, 15, 22, 0, 0, 0, time.UTC)}
	return &Writer{Root: t.TempDir(), Launcher: runs, Checker: hub, SkillDir: "/skill", Now: c.now}, runs, hub, c
}

func writePage(t *testing.T, w *Writer, r Request, html string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(w.Dir(r), PageFile), []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}
}

func advance(t *testing.T, w *Writer, r Request) Result {
	t.Helper()
	res, err := w.Advance(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestAPageThatPassesTheHubCheckIsReadyAfterOneRun(t *testing.T) {
	w, runs, hub, _ := newWriter(t)
	r := request()
	if res := advance(t, w, r); res.State != StateWorking || len(runs.launched) != 1 {
		t.Fatalf("first step = %+v, launched %d; want one run working", res, len(runs.launched))
	}
	brief, err := os.ReadFile(filepath.Join(w.Dir(r), BriefFile))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Take the eight standing-team policy recommendations?", "`accept` (recommended)", r.Evidence[0], "pages check " + filepath.Join(w.Dir(r), PageFile) + " --cover-file " + filepath.Join(w.Dir(r), "request.json"), "prompt-manager skill read notification-hub", "invent nothing", "Missing from the evidence"} {
		if !strings.Contains(string(brief), want) {
			t.Errorf("brief lacks %q", want)
		}
	}
	if res := advance(t, w, r); res.State != StateWorking || hub.calls != 0 {
		t.Fatalf("while the run works = %+v (checks %d); want working, unchecked", res, hub.calls)
	}
	writePage(t, w, r, goodPage)
	runs.end("run-1")
	res := advance(t, w, r)
	if res.State != StateReady || res.HTML != goodPage || res.SHA256 == "" || res.Attempts[0].Model != "gpt-6-luna" {
		t.Fatalf("after the run = %+v; want ready with the page and the run's model", res)
	}
	if again := advance(t, w, r); again.State != StateReady || len(runs.launched) != 1 {
		t.Fatalf("a ready job must not start another run: %+v, launched %d", again, len(runs.launched))
	}
}

func TestTheSameFactsAreTheSameJobAndChangedFactsAreANewOne(t *testing.T) {
	w, runs, _, _ := newWriter(t)
	r := request()
	advance(t, w, r)
	advance(t, w, r)
	if len(runs.launched) != 1 {
		t.Fatalf("repeating a request launched %d runs; want 1", len(runs.launched))
	}
	changed := r
	changed.Question = "Take six of the eight policies now?"
	if w.Dir(changed) == w.Dir(r) {
		t.Fatal("changed facts share the old job folder")
	}
	if !strings.HasPrefix(w.Dir(changed), filepath.Dir(w.Dir(r))) {
		t.Fatal("one subject's jobs should sit together")
	}
}

func TestAFailedCheckGetsOneMoreRunWithTheFindingsThenFails(t *testing.T) {
	w, runs, hub, _ := newWriter(t)
	r := request()
	hub.findings = func(string) []Finding {
		return []Finding{
			{Code: "explainer_restates", Message: "the page is mostly the question and options again", Severity: "advice"},
			{Code: "missing_big_picture", Message: "there is no big picture", Severity: "advice"},
		}
	}
	advance(t, w, r)
	writePage(t, w, r, goodPage)
	runs.end("run-1")
	res := advance(t, w, r)
	if res.State != StateWorking || len(runs.launched) != 2 {
		t.Fatalf("after a failed check = %+v, launched %d; want a second run", res, len(runs.launched))
	}
	brief, _ := os.ReadFile(filepath.Join(w.Dir(r), BriefFile))
	if !strings.Contains(string(brief), "explainer_restates") || strings.Contains(string(brief), "missing_big_picture") || !strings.Contains(string(brief), "attempt-1/page.html") {
		t.Fatalf("the retry brief must carry the page findings only and point at the first page:\n%s", brief)
	}
	if _, err := os.Stat(filepath.Join(w.Dir(r), "attempt-1", PageFile)); err != nil {
		t.Fatal("the first page was not kept for the retry")
	}
	if len(res.Advice) != 1 || res.Advice[0].Code != "missing_big_picture" {
		t.Fatalf("cover findings should come back as advice: %+v", res.Advice)
	}
	writePage(t, w, r, goodPage)
	runs.end("run-2")
	res = advance(t, w, r)
	if res.State != StateFailed || res.Failure.Class != FailLint || !res.Terminal() || len(runs.launched) != 2 {
		t.Fatalf("second failed check = %+v; want terminal lint_failed and no third run", res)
	}
}

func TestARunThatWritesNoPageFailsAfterItsRetry(t *testing.T) {
	w, runs, _, _ := newWriter(t)
	r := request()
	advance(t, w, r)
	runs.end("run-1")
	if res := advance(t, w, r); res.State != StateWorking || len(runs.launched) != 2 {
		t.Fatalf("no page = %+v; want the retry", res)
	}
	runs.end("run-2")
	res := advance(t, w, r)
	if res.State != StateFailed || res.Failure.Class != FailRun || !strings.Contains(res.Failure.Detail, "no page.html") {
		t.Fatalf("= %+v; want run_failed naming the missing page", res)
	}
}

func TestAPageMissingAnAnswerButtonIsNotCheckedAtTheHub(t *testing.T) {
	w, runs, hub, _ := newWriter(t)
	r := request()
	advance(t, w, r)
	writePage(t, w, r, strings.ReplaceAll(goodPage, `data-answer="hold"`, `data-answer="later"`))
	runs.end("run-1")
	res := advance(t, w, r)
	if hub.calls != 0 || res.Attempts[0].Problem == "" || !strings.Contains(res.Attempts[0].Problem, "hold") {
		t.Fatalf("= %+v (hub calls %d); want a local problem naming hold", res, hub.calls)
	}
}

func TestARunPastItsTimeIsStoppedAndRetried(t *testing.T) {
	w, runs, _, c := newWriter(t)
	r := request()
	advance(t, w, r)
	c.advance(DefaultAttemptTimeout + time.Minute)
	res := advance(t, w, r)
	if len(runs.stopped) != 1 || len(runs.launched) != 2 || !strings.Contains(res.Attempts[0].Problem, "did not finish") {
		t.Fatalf("= %+v, stopped %v; want the late run stopped and retried", res, runs.stopped)
	}
}

// A standing grant admits run creation only, so Agent Manager may refuse the
// stop; the attempt says so instead of claiming the run was stopped.
func TestARunThatCouldNotBeStoppedIsReportedAsPossiblyStillRunning(t *testing.T) {
	w, runs, _, c := newWriter(t)
	runs.stopErr = fmt.Errorf("%w: operation_not_admitted", ErrRefused)
	r := request()
	advance(t, w, r)
	c.advance(DefaultAttemptTimeout + time.Minute)
	res := advance(t, w, r)
	if p := res.Attempts[0].Problem; !strings.Contains(p, "could not be stopped") || strings.Contains(p, "was stopped") || len(runs.launched) != 2 {
		t.Fatalf("= %+v; want the refused stop reported and the retry started", res)
	}
}

// No authority to start a run says nothing about the page: it is retried
// after the cool-down, so a grant issued later takes effect.
func TestARefusedLaunchIsRetriedAfterACoolDown(t *testing.T) {
	w, runs, _, c := newWriter(t)
	r := request()
	runs.refuse = fmt.Errorf("%w: verified operator required", ErrRefused)
	res := advance(t, w, r)
	if res.State != StateFailed || res.Failure.Class != FailRefused || res.Terminal() {
		t.Fatalf("= %+v; want a non-terminal launch_refused", res)
	}
	runs.refuse = nil
	c.advance(DefaultRetryAfter)
	if again := advance(t, w, r); again.State != StateWorking || len(runs.launched) != 1 {
		t.Fatalf("after the cool-down = %+v; want a run", again)
	}
}

// The production launcher never creates a run without the writer's service
// credential, even where Agent Manager would admit a caller with none.
func TestTheLauncherRefusesWithoutAServiceCredential(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer writer" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		switch r.URL.Path {
		case "/api/v1/tasks":
			fmt.Fprint(w, `{"task":{"id":"task-1"}}`)
		case "/api/v1/runs":
			fmt.Fprint(w, `{"run":{"id":"run-1","task_id":"task-1","status":"RUN_STATUS_RUNNING","requested_model":"gpt-6-luna"}}`)
		}
	}))
	defer srv.Close()
	base := func(context.Context) (string, error) { return srv.URL, nil }
	spec := LaunchSpec{Title: "t", Prompt: "p", IdempotencyKey: "k"}
	if _, err := (&AgentManager{BaseURL: base}).Launch(context.Background(), spec); !errors.Is(err, ErrRefused) || calls != 0 {
		t.Fatalf("no credential: err %v after %d calls; want a refusal before any call", err, calls)
	}
	pending := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprint(w, `{"status":"approval_pending","subject":"research-desk-agent-manager"}`)
	}))
	defer pending.Close()
	auth := func(context.Context) (string, error) { return "Bearer writer", nil }
	pendingBase := func(context.Context) (string, error) { return pending.URL, nil }
	if _, err := (&AgentManager{BaseURL: pendingBase, Authorization: auth}).Launch(context.Background(), spec); !errors.Is(err, ErrRefused) {
		t.Fatalf("approval pending = %v; want a refusal the writer retries", err)
	}
	run, err := (&AgentManager{BaseURL: base, Authorization: auth}).Launch(context.Background(), spec)
	if err != nil || run.ID != "run-1" || run.Model != "gpt-6-luna" {
		t.Fatalf("with the credential = %+v, %v", run, err)
	}
}

func TestAnOutageIsRetriedAfterACoolDown(t *testing.T) {
	w, runs, _, c := newWriter(t)
	r := request()
	runs.refuse = fmt.Errorf("%w: connection refused", ErrUnavailable)
	res := advance(t, w, r)
	if res.State != StateFailed || res.Failure.Class != FailUnavailable || res.Terminal() {
		t.Fatalf("= %+v; want a non-terminal unavailable failure", res)
	}
	runs.refuse = nil
	if again := advance(t, w, r); again.State != StateFailed || len(runs.launched) != 0 {
		t.Fatalf("retried before the cool-down: %+v", again)
	}
	c.advance(DefaultRetryAfter)
	if again := advance(t, w, r); again.State != StateWorking || len(runs.launched) != 1 {
		t.Fatalf("after the cool-down = %+v; want a run", again)
	}
}

func TestTheWriterRunsAtMostMaxActiveJobsAtOnce(t *testing.T) {
	w, runs, _, _ := newWriter(t)
	w.MaxActive = 1
	a, b := request(), request()
	b.SubjectKey = "research-desk/supervisor/OTHER"
	advance(t, w, a)
	if res := advance(t, w, b); res.State != StateQueued || len(runs.launched) != 1 {
		t.Fatalf("second job = %+v; want queued behind the first", res)
	}
	writePage(t, w, a, goodPage)
	runs.end("run-1")
	advance(t, w, a)
	if res := advance(t, w, b); res.State != StateWorking || len(runs.launched) != 2 {
		t.Fatalf("after the first finished = %+v; want the second started", res)
	}
}

func TestStatusChangesNothing(t *testing.T) {
	w, runs, _, _ := newWriter(t)
	r := request()
	res, err := w.Status(r)
	if err != nil || res.State != StateQueued || len(runs.launched) != 0 {
		t.Fatalf("= %+v %v; want queued, nothing launched", res, err)
	}
	if _, err := os.Stat(w.Dir(r)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("Status created the job folder")
	}
}

func TestAnInvalidRequestIsRefusedWithoutARun(t *testing.T) {
	w, runs, _, _ := newWriter(t)
	r := request()
	r.Options = r.Options[:1]
	res := advance(t, w, r)
	if res.State != StateFailed || res.Failure.Class != FailInvalid || len(runs.launched) != 0 {
		t.Fatalf("= %+v; want invalid_request", res)
	}
}

func TestPageFindingsAreThePagesToFix(t *testing.T) {
	for code, want := range map[string]bool{
		"explainer_restates": true, "explainer_network": true, "command_hidden": true, "page_unreadable": true,
		"explainer_missing": false, "missing_big_picture": false, "packet_missing": false, "recurring_subject": false,
	} {
		if got := PageFinding(Finding{Code: code, Severity: "advice"}); got != want {
			t.Errorf("PageFinding(%s) = %v, want %v", code, got, want)
		}
	}
	if !PageFinding(Finding{Code: "something_new", Field: "attachments", Severity: "blocking"}) {
		t.Error("a blocking finding on the page must fail the page")
	}
	// The Hub blocks an ask with no recommendation reason; no page fixes that.
	if PageFinding(Finding{Code: "missing_recommendation_reason", Field: "recommendation_reason", Severity: "blocking"}) {
		t.Error("a blocking finding on the sender's cover is advice to the sender, not the page's to fix")
	}
}

// Agent Manager holds an idempotency key for good: a job made again for the
// same facts (its folder removed, or another root) must start a new run, not
// collide with the first job's accepted run.
func TestAJobMadeAgainForTheSameFactsLaunchesUnderANewKey(t *testing.T) {
	w, runs, _, c := newWriter(t)
	r := request()
	advance(t, w, r)
	if err := os.RemoveAll(w.Dir(r)); err != nil {
		t.Fatal(err)
	}
	c.advance(time.Minute)
	advance(t, w, r)
	if len(runs.launched) != 2 || runs.launched[0].IdempotencyKey == runs.launched[1].IdempotencyKey {
		t.Fatalf("the remade job reused the first job's key: %+v", runs.launched)
	}
}
