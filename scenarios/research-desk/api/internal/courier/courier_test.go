package courier

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"

	"research-desk/internal/home"
	"research-desk/internal/testutil"
	"research-desk/internal/writer"
)

// The courier is tested at its Hub seam: a fake Hub over the testutil archive of real
// research-home excerpts. local-models-w1-20261111 is a real prepared packet; each test
// gives its STATE.md the contract's needs-you items.

const items = `
## Needs you

### AV4 — yt-dlp target choice
Keep or propose.
Recommended: keep — Keep yt-dlp uninstalled and continue AV5–AV7.
Options: keep — Keep it uninstalled; propose — Ask for a maintained target proposal.

### AV5 — Rights of the pose checkpoint
No recommendation line yet.
`

const commission = "local-models-w1-20261111"

// fakeHub stands in for Vrooli Events plus Notification Hub: a published
// ask or notice becomes a Hub record, read back by its key.
type fakeHub struct {
	asks      map[string]*Ask
	notices   map[string]*Notice
	raised    []AskSpec
	sent      []NoticeSpec
	readErr   error
	sendErr   error
	reads     int
	delivered bool
	clock     *schedule.Fake
}

func (h *fakeHub) Sendable(context.Context) error { return h.sendErr }

func (h *fakeHub) Ask(_ context.Context, spec AskSpec) error {
	h.raised = append(h.raised, spec)
	if keyOf(spec.Question) != spec.Key {
		return errors.New("the question does not lead with the key")
	}
	h.asks[spec.Key] = &Ask{ID: "ask-" + spec.Key, Key: spec.Key, State: "pending", CreatedAt: h.clock.Now(), Deadline: spec.Deadline}
	return nil
}

func (h *fakeHub) Notify(_ context.Context, spec NoticeSpec) error {
	h.sent = append(h.sent, spec)
	h.notices[spec.Key] = &Notice{ID: "n-" + spec.Key, Key: spec.Key, Delivered: h.delivered, State: "queued"}
	return nil
}

func (h *fakeHub) Asks(_ context.Context, prefix string) ([]Ask, error) {
	h.reads++
	if h.readErr != nil {
		return nil, h.readErr
	}
	var out []Ask
	for key, a := range h.asks {
		if strings.HasPrefix(key, prefix) {
			out = append(out, *a)
		}
	}
	return out, nil
}

func (h *fakeHub) Notices(context.Context) (map[string]Notice, error) {
	out := map[string]Notice{}
	for k, n := range h.notices {
		out[k] = *n
	}
	return out, nil
}

type rig struct {
	courier  *Courier
	hub      *fakeHub
	research string
	clock    *schedule.Fake
	state    string // the real STATE.md text
}

func newRig(t *testing.T, launchedBy string) *rig {
	t.Helper()
	h := testutil.NewHome(t)
	research := h.Research
	dir := filepath.Join(research, "cohorts", commission)
	state, _ := os.ReadFile(filepath.Join(dir, home.StateFile))
	clock := schedule.NewFake(time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC))
	reader := home.NewReader(home.Config{ResearchDir: research, CodexDir: h.Codex}, home.Deps{Clock: clock, Procs: home.FixedProcs{}})
	w := writer.New(research, h.State)
	t.Cleanup(func() { w.Close() })
	if launchedBy == "desk" {
		// The desk launches from templates/charter.md, which carries both
		// contract sentences; the real local-models charter predates them.
		f, err := os.OpenFile(filepath.Join(dir, home.CharterFile), os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		f.WriteString("\n## Steering\n\n" + home.SteeringSentence + "\n\n## Decisions\n\n" + home.DecisionsSentence + "\n")
		f.Close()
	}
	if launchedBy != "" {
		if _, err := w.WriteLaunch(commission, &home.LaunchRecord{Schema: home.LaunchSchema, CommissionID: commission, LaunchedBy: launchedBy}); err != nil {
			t.Fatal(err)
		}
	}
	hub := &fakeHub{asks: map[string]*Ask{}, notices: map[string]*Notice{}, clock: clock}
	r := &rig{
		courier:  &Courier{Reader: reader, Writer: w, Hub: hub, Clock: clock, Logger: log.New(io.Discard, "", 0)},
		hub:      hub,
		research: research,
		clock:    clock,
		state:    string(state),
	}
	r.setItems(t, items)
	return r
}

// setItems gives STATE.md the real file's text plus the given items.
func (r *rig) setItems(t *testing.T, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(r.research, "cohorts", commission, home.StateFile), []byte(r.state+text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (r *rig) pass(t *testing.T) Plan {
	t.Helper()
	plan, err := r.courier.Pass(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Every step says why as a code a client matches on, never as prose
	// alone (rulings P5).
	for _, s := range plan.Steps {
		if s.Reason == "" || s.Why == "" {
			t.Fatalf("a step without its reason code and sentence: %+v", s)
		}
	}
	return plan
}

func (r *rig) operatorFile(t *testing.T) []home.Entry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(r.research, "cohorts", commission, home.OperatorFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	entries, _ := home.ParseEntries(string(data))
	return entries
}

// [REQ:RD-P0-006]
func TestOldItemsAreNeverSentByDefault(t *testing.T) {
	r := newRig(t, "")
	plan := r.pass(t)
	if !plan.HubSkipped || r.hub.reads != 0 || len(r.hub.raised)+len(r.hub.sent) != 0 {
		t.Fatalf("with every switch at its default the Hub is not even read: %+v reads=%d", plan, r.hub.reads)
	}
	for _, s := range plan.Steps {
		if s.Item.CommissionID == commission && (s.On || !strings.Contains(s.Why, "did not launch")) {
			t.Errorf("%s: a hand-launched commission is off and says how to turn it on: %+v", s.Key, s)
		}
	}
}

// [REQ:RD-P0-006]
func TestDeskLaunchedCommissionAsksAndReadsTheAnswerBack(t *testing.T) {
	r := newRig(t, "desk")
	r.pass(t)
	if len(r.hub.raised) != 1 || r.hub.raised[0].Key != AskPrefix+commission+"/AV4" || r.hub.raised[0].Recommended != "keep" || len(r.hub.raised[0].Options) != 2 {
		t.Fatalf("one ask for AV4 keyed by commission and ID: %+v", r.hub.raised)
	}
	if len(r.hub.sent) != 1 || !strings.HasSuffix(r.hub.sent[0].Key, commission+"/AV5") {
		t.Fatalf("AV5 has no Recommended: line, so it goes as a notice: %+v", r.hub.sent)
	}
	if !r.hub.raised[0].Deadline.IsZero() {
		t.Fatal("an item's ask carries no deadline of its own, so an unchanged item is the same event")
	}

	// A second pass sends nothing new: the ask and the notice exist.
	r.pass(t)
	if len(r.hub.raised) != 1 || len(r.hub.sent) != 1 {
		t.Fatalf("an existing ask or notice is not sent again: %d asks, %d notices", len(r.hub.raised), len(r.hub.sent))
	}

	// The Hub delivers, then the operator answers "other" on the phone.
	a := r.hub.asks[AskPrefix+commission+"/AV4"]
	a.FirstDelivered = r.clock.Now()
	r.pass(t)
	a.State, a.Answer, a.Note = "answered", "other", "Ask me again after AV6"
	r.clock.Advance(time.Minute)
	r.pass(t)
	entries := r.operatorFile(t)
	if len(entries) != 2 || entries[0].Kind != home.EntryDelivered || entries[1].Kind != home.EntryAnswer || entries[1].Via != home.ViaHub || entries[1].Body != "other — Ask me again after AV6" {
		t.Fatalf("OPERATOR.md must record the delivery then the Hub answer: %+v", entries)
	}
	r.pass(t)
	if got := r.operatorFile(t); len(got) != 2 {
		t.Fatalf("an answer is recorded once: %+v", got)
	}
}

func TestAnAskTheDeskNoLongerNeedsStaysOpenAndSaysSo(t *testing.T) {
	r := newRig(t, "desk")
	r.hub.asks[AskPrefix+commission+"/AV1"] = &Ask{ID: "old", Key: AskPrefix + commission + "/AV1", State: "pending", Deadline: r.clock.Now().Add(time.Hour)}
	r.pass(t)
	if _, err := r.courier.Writer.AppendOperator(commission, home.Entry{At: r.clock.Now().Add(time.Minute), Kind: home.EntryAnswer, Subject: "AV4", Via: home.ViaDesk, Body: "keep — Keep it uninstalled"}); err != nil {
		t.Fatal(err)
	}
	r.hub.asks[AskPrefix+commission+"/AV4"].State, r.hub.asks[AskPrefix+commission+"/AV4"].Answer = "pending", ""
	plan := r.pass(t)
	var orphan, answered bool
	for _, s := range plan.Steps {
		switch {
		case s.Item == nil && s.Key == AskPrefix+commission+"/AV1":
			orphan = s.Action == ActNone && strings.Contains(s.Why, "stays open until") && strings.Contains(s.Why, "no way to withdraw")
		case s.Item != nil && s.Item.Item.ID == "AV4":
			answered = s.Action == ActNone && strings.Contains(s.Why, "stays open")
		}
	}
	if !orphan || !answered {
		t.Fatalf("an ask whose item left, and one answered in the desk, stay open and say so: %+v", plan.Steps)
	}
}

func TestAnswerInTheHubAfterADeskAnswerIsIgnored(t *testing.T) {
	r := newRig(t, "desk")
	r.pass(t)
	if _, err := r.courier.Writer.AppendOperator(commission, home.Entry{At: r.clock.Now().Add(time.Minute), Kind: home.EntryAnswer, Subject: "AV4", Via: home.ViaDesk, Body: "keep — Keep it uninstalled"}); err != nil {
		t.Fatal(err)
	}
	a := r.hub.asks[AskPrefix+commission+"/AV4"]
	a.State, a.Answer = "answered", "propose"
	r.clock.Advance(time.Minute)
	r.pass(t)
	var answers int
	for _, e := range r.operatorFile(t) {
		if e.Kind == home.EntryAnswer {
			answers++
		}
	}
	if answers != 1 {
		t.Fatalf("the desk answer stands; the later Hub answer is not recorded: %d answers", answers)
	}
}

func TestNoEventRouteSendsNothingAndSaysWhy(t *testing.T) {
	r := newRig(t, "desk")
	r.hub.sendErr = errors.New("vrooli-events is not reachable, so nothing can be sent: not running")
	plan := r.pass(t)
	if len(r.hub.raised)+len(r.hub.sent) != 0 || plan.HubRead == nil {
		t.Fatal("with no route to the Hub nothing is sent")
	}
	for _, s := range plan.Steps {
		if s.Item.CommissionID == commission && s.On && !strings.Contains(s.Why, "vrooli-events is not reachable") {
			t.Errorf("%s says why it was not sent: %q", s.Key, s.Why)
		}
	}
}

func TestHubRefusalSendsNothingAndSaysWhy(t *testing.T) {
	r := newRig(t, "desk")
	r.hub.readErr = errors.New("unauthenticated: verified identity required")
	plan := r.pass(t)
	if len(r.hub.raised)+len(r.hub.sent) != 0 || plan.HubRead == nil {
		t.Fatal("without a Hub read-back nothing is sent")
	}
	for _, s := range plan.Steps {
		if s.Item.CommissionID == commission && !strings.Contains(s.Why, "unauthenticated") {
			t.Errorf("%s says why it was not sent: %q", s.Key, s.Why)
		}
	}
}

func TestNoticeDeliveryIsRecordedOnce(t *testing.T) {
	r := newRig(t, "desk")
	r.hub.delivered = true
	r.pass(t)
	r.pass(t)
	r.pass(t)
	var delivered int
	for _, e := range r.operatorFile(t) {
		if e.Kind == home.EntryDelivered && e.Subject == "AV5" {
			delivered++
		}
	}
	if delivered != 1 || len(r.hub.sent) != 1 {
		t.Fatalf("a delivered notice is recorded once, from the Hub's read-back, and not sent again: %d entries, %d sends", delivered, len(r.hub.sent))
	}
}

func TestSwitches(t *testing.T) {
	r := newRig(t, "desk")
	if _, err := r.courier.Set(context.Background(), commission, false); err != nil {
		t.Fatal(err)
	}
	r.pass(t)
	if len(r.hub.raised)+len(r.hub.sent) != 0 {
		t.Fatal("the operator's off wins over the desk-launch default")
	}
	sw, _ := r.courier.Switches()
	if sw.Global.On || sw.Commissions[commission].On {
		t.Errorf("switches: %+v", sw)
	}
}

// F6: turning the courier on never sends an item that was already there.
func TestTurningTheCourierOnNeverSendsTheItemsAlreadyThere(t *testing.T) {
	r := newRig(t, "")
	ctx := context.Background()
	if _, err := r.courier.Set(ctx, commission, true); err != nil {
		t.Fatal(err)
	}
	if _, err := r.courier.Set(ctx, "", true); err != nil {
		t.Fatal(err)
	}
	r.pass(t)
	r.clock.Advance(SchedulerGrace)
	plan := r.pass(t)
	if len(r.hub.raised)+len(r.hub.sent) != 0 {
		t.Fatalf("items present when the switches turned on were sent: %+v %+v", r.hub.raised, r.hub.sent)
	}
	for _, s := range plan.Steps {
		if s.Item != nil && s.Delivery != home.DeliverNone && !strings.Contains(s.Why, "already there when the courier was turned on") {
			t.Errorf("%s does not say why it stays unsent: %q", s.Key, s.Why)
		}
	}

	r.setItems(t, items+"\n### AV6 — A question that came later\nRecommended: go — Go ahead.\n")
	r.pass(t)
	r.clock.Advance(SchedulerGrace)
	r.pass(t)
	if len(r.hub.sent) != 1 || !strings.HasSuffix(r.hub.sent[0].Key, commission+"/AV6") {
		t.Fatalf("only the item that appeared after the switch is sent: %+v", r.hub.sent)
	}
}

func TestAnOlderCharterIsLeftToItsSchedulerForTwoHours(t *testing.T) {
	r := newRig(t, "")
	r.setItems(t, "")
	if _, err := r.courier.Set(context.Background(), commission, true); err != nil {
		t.Fatal(err)
	}
	r.setItems(t, items)
	plan := r.pass(t)
	if len(r.hub.raised)+len(r.hub.sent) != 0 {
		t.Fatal("the real local-models charter has no Decisions sentence: its scheduler delivers first")
	}
	for _, s := range plan.Steps {
		if s.Item != nil && s.Item.CommissionID == commission && !strings.Contains(s.Why, "Delivered:") {
			t.Errorf("%s says what it waits for: %q", s.Key, s.Why)
		}
	}
	r.clock.Advance(SchedulerGrace - time.Minute)
	r.pass(t)
	if len(r.hub.sent) != 0 {
		t.Fatal("the wait counts from the first pass that saw the item")
	}
	r.clock.Advance(time.Minute)
	r.pass(t)
	if len(r.hub.raised) != 0 || len(r.hub.sent) != 2 {
		t.Fatalf("after the grace both go as notices: without the Steering line an answer would not be read: asks %d, notices %d", len(r.hub.raised), len(r.hub.sent))
	}
}

func TestDeliveryTestRaisesOneExpiringAsk(t *testing.T) {
	r := newRig(t, "")
	result, err := r.courier.Test(context.Background(), 0)
	if err != nil || len(r.hub.raised) != 1 || !strings.HasPrefix(result.Key, TestPrefix) || result.Ask.ID == "" {
		t.Fatalf("one test ask, found again by its key: %+v %v", result, err)
	}
	if got := r.hub.raised[0].Deadline; !got.Equal(r.clock.Now().Add(testLifetime)) || !result.Expires.Equal(got) {
		t.Fatalf("the test ask expires on its own: deadline %v, reported %v", got, result.Expires)
	}
	if r.operatorFile(t) != nil {
		t.Error("the test writes nothing into the research home")
	}
}

func TestKeysReadBackFromQuestions(t *testing.T) {
	for question, want := range map[string]string{
		"programme-three-20261111 · YTDLP-TARGET — yt-dlp target": AskPrefix + "programme-three-20261111/YTDLP-TARGET",
		testQuestion + "20261112T010000Z — no answer needed":      TestPrefix + "20261112T010000Z",
		"Can I extend this window?":                               "",
		"a b · c — d":                                             "",
	} {
		if got := keyOf(question); got != want {
			t.Errorf("keyOf(%q) = %q, want %q", question, got, want)
		}
	}
}

// E6, F15: the courier's health is its real state, and it says that an ask
// cannot be withdrawn.
func TestHealthIsTheLastPassAndSaysAnAskCannotBeWithdrawn(t *testing.T) {
	if ok, said := (Health{}).Say(); ok || !strings.Contains(said, "no pass yet") {
		t.Fatalf("before the first pass: %v %q", ok, said)
	}
	r := newRig(t, "")
	r.pass(t)
	if ok, said := r.courier.Health().Say(); !ok || !strings.Contains(said, "no switch on") || !strings.Contains(said, "last delivery never") {
		t.Fatalf("no switch on: %v %q", ok, said)
	}
	r = newRig(t, "desk")
	r.pass(t)
	h := r.courier.Health()
	if ok, said := h.Say(); !ok || h.LastDelivery.IsZero() || h.LastReadBack.IsZero() || !strings.Contains(said, Withdraw) {
		t.Fatalf("after a delivery: %+v %q", h, said)
	}
	r.hub.readErr = errors.New("unauthenticated: verified identity required")
	r.clock.Advance(time.Minute)
	r.pass(t)
	if ok, said := r.courier.Health().Say(); ok || !strings.Contains(said, "unauthenticated") || !strings.Contains(said, "answers last read back "+stamp(h.LastReadBack)) {
		t.Fatalf("a refused read-back: %v %q", ok, said)
	}
}

// F11: an ID the supervisor reuses for a new decision after the earlier one
// was answered is shown as reused, never taken as answered or left unsent.
func TestAReusedIDIsShownNotDelivered(t *testing.T) {
	r := newRig(t, "desk")
	r.pass(t)
	a := r.hub.asks[AskPrefix+commission+"/AV4"]
	a.State, a.Answer = "answered", "keep"
	r.clock.Advance(time.Minute)
	r.pass(t)

	// The supervisor acts on the answer and removes AV4, then later raises a
	// new decision under the same ID.
	r.setItems(t, items[:strings.Index(items, "### AV4")]+items[strings.Index(items, "### AV5"):])
	r.clock.Advance(time.Hour)
	r.pass(t)
	r.setItems(t, items[:strings.Index(items, "### AV4")]+"### AV4 — Licence for the new footage pack\nWhich licence?\nRecommended: cc-by — Use CC-BY only.\nOptions: cc-by — Use CC-BY only; any — Any open licence.\n\n"+items[strings.Index(items, "### AV5"):])
	r.clock.Advance(time.Hour)
	r.pass(t)
	plan := r.pass(t)
	var why string
	var reason home.DeliveryReason
	for _, s := range plan.Steps {
		if s.Item != nil && s.Item.Item.ID == "AV4" {
			why, reason = s.Why, s.Reason
			if s.Action != ActNone || !s.Reused {
				t.Fatalf("a reused ID is not acted on: %+v", s)
			}
		}
	}
	if reason != home.ReasonIDReused || !strings.HasPrefix(why, "ID reused, not delivered: AV4 already has an answer in OPERATOR.md") || strings.Contains(why, "research-desk ") {
		t.Fatalf("why = %q", why)
	}
	if len(r.hub.raised) != 1 {
		t.Fatalf("no ask for the new AV4 while its ID is reused: %d asks", len(r.hub.raised))
	}

	// An answer in the desk after the new item appeared settles it.
	if _, err := r.courier.Writer.AppendOperator(commission, home.Entry{At: r.clock.Now(), Kind: home.EntryAnswer, Subject: "AV4", Via: home.ViaDesk, Body: "cc-by — Use CC-BY only"}); err != nil {
		t.Fatal(err)
	}
	r.clock.Advance(time.Minute)
	for _, s := range r.pass(t).Steps {
		if s.Item != nil && s.Item.Item.ID == "AV4" && strings.HasPrefix(s.Why, "ID reused") {
			t.Fatalf("answered after it appeared: %q", s.Why)
		}
	}
}

// lock makes a file exist but refuse reads until the returned func runs.
func lock(t *testing.T, path string) func() {
	t.Helper()
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	unlock := func() { os.Chmod(path, 0o644) }
	t.Cleanup(unlock)
	if _, err := os.ReadFile(path); err == nil {
		t.Skip("this user reads a file with no permissions (root); cannot test an unreadable file")
	}
	return unlock
}

// [REQ:RD-P0-006]
// E1: while the courier cannot read a commission's answers it sends,
// records and forgets nothing for it, and says why; once readable, a Hub
// answer given meanwhile is recorded once.
func TestNothingIsDeliveredWhileAnswersCannotBeRead(t *testing.T) {
	r := newRig(t, "desk")
	r.pass(t)
	if len(r.hub.raised) != 1 {
		t.Fatalf("AV4 is asked first: %+v", r.hub.raised)
	}
	seenBefore, _ := os.ReadFile(filepath.Join(r.courier.Writer.StateDir(), SeenFile))
	operator := filepath.Join(r.research, "cohorts", commission, home.OperatorFile)
	if err := os.WriteFile(operator, []byte("# Operator — "+commission+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	unlock := lock(t, operator)
	a := r.hub.asks[AskPrefix+commission+"/AV4"]
	a.State, a.Answer = "answered", "keep"
	r.setItems(t, items+"\n### AV9 — a new item while answers cannot be read\nRecommended: go — Go.\n")
	r.clock.Advance(time.Minute)
	plan := r.pass(t)
	if len(r.hub.raised) != 1 || len(r.hub.sent) != 1 {
		t.Fatalf("nothing new is sent: %d asks, %d notices", len(r.hub.raised), len(r.hub.sent))
	}
	for _, s := range plan.Steps {
		if s.Item != nil && s.Item.CommissionID == commission && (s.Action != ActNone || !strings.Contains(s.Why, "cannot read "+operator)) {
			t.Errorf("%s: %s %q", s.Key, s.Action, s.Why)
		}
	}
	if ok, said := r.courier.Health().Say(); ok || !strings.Contains(said, "cannot read "+operator) {
		t.Errorf("health says it: %v %q", ok, said)
	}
	if _, err := r.courier.Set(context.Background(), commission, true); !errors.Is(err, home.ErrRefused) {
		t.Errorf("turning the switch on needs the items readable: %v", err)
	}
	if after, _ := os.ReadFile(filepath.Join(r.courier.Writer.StateDir(), SeenFile)); string(after) != string(seenBefore) {
		t.Errorf("first sights are kept, not forgotten:\nbefore %s\nafter  %s", seenBefore, after)
	}

	unlock()
	r.clock.Advance(time.Minute)
	r.pass(t)
	r.pass(t)
	if entries := r.operatorFile(t); len(entries) != 1 || entries[0].Kind != home.EntryAnswer || entries[0].Subject != "AV4" {
		t.Fatalf("the Hub answer is recorded once when OPERATOR.md is readable again: %+v", entries)
	}
}

// E1: an unreadable STATE.md hides its items; they are not taken as gone,
// so the two-hour wait for an older charter's scheduler runs from when the
// item first appeared, not from when STATE.md could be read again.
func TestAnUnreadableStateIsNotTakenAsNoItems(t *testing.T) {
	r := newRig(t, "")
	if _, err := r.courier.Set(context.Background(), commission, true); err != nil {
		t.Fatal(err)
	}
	r.setItems(t, items+"\n### AV9 — a new item\nRecommended: go — Go.\nOptions: go — Go; wait — Wait.\n")
	r.pass(t)
	unlock := lock(t, filepath.Join(r.research, "cohorts", commission, home.StateFile))
	r.clock.Advance(time.Hour)
	if plan := r.pass(t); len(plan.Unreadable) != 1 {
		t.Fatalf("the pass names the unreadable STATE.md: %+v", plan.Unreadable)
	}
	unlock()
	r.clock.Advance(SchedulerGrace)
	r.pass(t)
	// The older charter does not read OPERATOR.md, so AV9 goes as a notice.
	if len(r.hub.raised) != 0 || len(r.hub.sent) != 1 || r.hub.sent[0].Key != NoticePrefix+commission+"/AV9" {
		t.Fatalf("AV9 is sent once its scheduler's two hours from first sight are over; AV4 and AV5 were there when the switch was turned on: %+v %+v", r.hub.raised, r.hub.sent)
	}
}
