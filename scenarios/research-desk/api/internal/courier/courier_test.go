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

	"github.com/vrooli/api-core/operatordecision"
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
	asks map[string]*Ask
	// older holds asks the Hub kept besides the one in asks for a key: an ask
	// raised before asks carried a subject key stays open next to its replacement.
	older []*Ask
	// unkeyed is the ids of asks raised without a subject key: a request
	// does not update them, it opens a new decision.
	unkeyed map[string]bool
	// withdrawn is the reason given for each ask the courier closed, by ask id.
	withdrawn map[string]string
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
	if keyOf(spec.Question, spec.TechnicalDetails) != spec.Key {
		return errors.New("the technical details do not name the key")
	}
	ref := h.RequestRef(spec)
	if open := h.asks[spec.Key]; open != nil && open.Open() && !h.unkeyed[open.ID] {
		// One open decision per subject: the Hub updates it in place.
		open.RequesterRef = ref
		open.Question, open.BigPicture, open.TechnicalDetails, open.Recommended, open.Options = spec.Question, spec.BigPicture, spec.TechnicalDetails, spec.Recommended, spec.Options
		return nil
	} else if open != nil && open.Open() {
		h.older = append(h.older, open)
	}
	h.asks[spec.Key] = &Ask{
		ID: "ask-" + spec.Key + "-" + ref, Key: spec.Key, State: "pending", RequesterRef: ref, CreatedAt: h.clock.Now(), Deadline: spec.Deadline,
		Question: spec.Question, BigPicture: spec.BigPicture, TechnicalDetails: spec.TechnicalDetails, Recommended: spec.Recommended, Options: spec.Options,
	}
	return nil
}

func (h *fakeHub) RequestRef(spec AskSpec) string {
	ref := "ref:" + spec.Key + "|" + spec.Question + "|" + spec.BigPicture + "|" + spec.TechnicalDetails + "|" + spec.Recommended
	for _, o := range spec.Options {
		ref += "|" + o.Key + "=" + o.Label
	}
	return ref
}

func (h *fakeHub) Withdraw(_ context.Context, ask Ask, reason string, _ time.Time) error {
	if ask.RequesterRef == "" {
		return errors.New("no request id")
	}
	for _, a := range append([]*Ask{}, append(h.older, mapValues(h.asks)...)...) {
		if a.ID == ask.ID && a.Open() {
			a.State = "withdrawn"
			h.withdrawn[a.ID] = reason
		}
	}
	return nil
}

func mapValues(m map[string]*Ask) []*Ask {
	var out []*Ask
	for _, a := range m {
		out = append(out, a)
	}
	return out
}

// open is how many asks for key are open at the Hub.
func (h *fakeHub) open(key string) int {
	var n int
	for _, a := range append(append([]*Ask{}, h.older...), mapValues(h.asks)...) {
		if a.Key == key && a.Open() {
			n++
		}
	}
	return n
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
	for _, a := range h.older {
		if strings.HasPrefix(a.Key, prefix) {
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
	w := writer.New(research, h.State, testutil.EntryChars)
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
	hub := &fakeHub{asks: map[string]*Ask{}, unkeyed: map[string]bool{}, withdrawn: map[string]string{}, notices: map[string]*Notice{}, clock: clock}
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
	// alone.
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
	if len(r.hub.raised)+len(r.hub.sent) != 0 {
		t.Fatalf("with every switch at its default nothing already there is sent: %+v", plan)
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

// answerInDesk records the operator's answer to AV4 in OPERATOR.md, as the desk does.
func (r *rig) answerInDesk(t *testing.T) {
	t.Helper()
	r.clock.Advance(time.Minute)
	if _, err := r.courier.Writer.AppendOperator(commission, home.Entry{At: r.clock.Now(), Kind: home.EntryAnswer, Subject: "AV4", Via: home.ViaDesk, Body: "keep — Keep it uninstalled"}); err != nil {
		t.Fatal(err)
	}
	r.clock.Advance(time.Minute)
}

func (r *rig) step(plan Plan, key string) Step {
	for _, s := range plan.Steps {
		if s.Key == key {
			return s
		}
	}
	return Step{}
}

// An ask for an item the operator answered in the desk is withdrawn, once,
// with a plain reason; it is never read as an answer.
func TestAnAskForAnItemAnsweredInTheDeskIsWithdrawn(t *testing.T) {
	r := newRig(t, "desk")
	key := AskPrefix + commission + "/AV4"
	r.pass(t)
	r.answerInDesk(t)
	plan := r.pass(t)
	if s := r.step(plan, key); s.Action != ActWithdraw || s.Reason != home.ReasonAnswered || !s.Done {
		t.Fatalf("the ask is withdrawn: %+v", s)
	}
	if a := r.hub.asks[key]; a.State != "withdrawn" || r.hub.withdrawn[a.ID] != withdrawAnswered || r.hub.open(key) != 0 {
		t.Fatalf("the Hub ask closes with the reason: %+v %v", a, r.hub.withdrawn)
	}
	if got := r.operatorFile(t); len(got) != 1 || got[0].Via != home.ViaDesk {
		t.Fatalf("a withdrawal is not an answer; only the desk's own answer is in OPERATOR.md: %+v", got)
	}
	if s := r.step(r.pass(t), key); s.Action != ActNone || strings.Contains(s.Why, "answer it in the desk") {
		t.Fatalf("a closed ask is left alone and the answered item is not asked for again: %+v", s)
	}
	if len(r.hub.withdrawn) != 1 {
		t.Fatalf("withdrawn once: %v", r.hub.withdrawn)
	}
}

// An open ask whose item left the files is withdrawn; one whose files cannot
// be read is kept, because the item may still be there.
func TestAnAskWhoseItemLeftIsWithdrawnButNotWhileFilesAreUnreadable(t *testing.T) {
	r := newRig(t, "desk")
	gone := AskPrefix + commission + "/AV1"
	r.hub.asks[gone] = &Ask{ID: "old", Key: gone, State: "pending", RequesterRef: "ref-old", Deadline: r.clock.Now().Add(time.Hour)}
	plan := r.pass(t)
	if s := r.step(plan, gone); s.Item != nil || s.Action != ActWithdraw || s.Reason != home.ReasonOrphaned || !s.Done {
		t.Fatalf("an ask with no item is withdrawn: %+v", s)
	}
	if r.hub.asks[gone].State != "withdrawn" || r.hub.withdrawn["old"] != withdrawOrphaned {
		t.Fatalf("the Hub ask closes with the reason: %+v %v", r.hub.asks[gone], r.hub.withdrawn)
	}

	// STATE.md cannot be read, so its items are not known to be gone.
	kept := AskPrefix + commission + "/AV4"
	unlock := lock(t, filepath.Join(r.research, "cohorts", commission, home.StateFile))
	defer unlock()
	plan = r.pass(t)
	if s := r.step(plan, kept); s.Action != ActNone || r.hub.asks[kept].State != "pending" || !strings.Contains(s.Why, "cannot be read") {
		t.Fatalf("an ask is kept while its item's files cannot be read: %+v", s)
	}
}

// An item whose text changed while its ask was open updates that ask: the Hub
// keeps one open decision for it. A closed ask is never raised again.
func TestAChangedItemUpdatesItsOpenAsk(t *testing.T) {
	r := newRig(t, "desk")
	key := AskPrefix + commission + "/AV4"
	r.pass(t)
	r.pass(t)
	if len(r.hub.raised) != 1 {
		t.Fatalf("an unchanged item is raised once: %d", len(r.hub.raised))
	}
	r.setItems(t, strings.Replace(items, "yt-dlp target choice", "yt-dlp target choice (revised)", 1))
	r.clock.Advance(time.Minute)
	plan := r.pass(t)
	if s := r.step(plan, key); s.Action != ActRefreshAsk || !s.Done {
		t.Fatalf("the changed item is raised again: %+v", s)
	}
	if len(r.hub.raised) != 2 || !strings.Contains(r.hub.raised[1].Question, "(revised)") || r.hub.open(key) != 1 || len(r.hub.older) != 0 {
		t.Fatalf("one open ask carries the new text: raised %+v open %d older %d", r.hub.raised, r.hub.open(key), len(r.hub.older))
	}
	r.pass(t)
	if len(r.hub.raised) != 2 {
		t.Fatalf("the updated ask is current, so it is not raised again: %d", len(r.hub.raised))
	}

	// The operator dismissed it: the desk does not raise it again.
	r.hub.asks[key].State = "dismissed"
	r.setItems(t, strings.Replace(items, "yt-dlp target choice", "yt-dlp target choice (revised twice)", 1))
	r.pass(t)
	if len(r.hub.raised) != 2 {
		t.Fatalf("a dismissed ask stays closed: %d", len(r.hub.raised))
	}
}

// An ask raised before asks carried a subject key and still saying what its
// item says is left alone, not raised again (that would page the operator twice
// for nothing); it is withdrawn by its request id when the item is answered.
// When its item changes it is replaced by a keyed ask, and the old one is
// withdrawn, so the operator is left with one open decision.
func TestAnAskRaisedBeforeSubjectKeysIsClosedWhenItsItemIsAnsweredOrReplacedWhenItChanges(t *testing.T) {
	r := newRig(t, "desk")
	key := AskPrefix + commission + "/AV4"
	r.pass(t)
	old := r.hub.asks[key]
	old.ID, old.RequesterRef = "legacy-ask", "event-id-before-subject-keys"
	r.hub.unkeyed[old.ID] = true
	r.clock.Advance(time.Minute)

	if s := r.step(r.pass(t), key); s.Action != ActNone || len(r.hub.raised) != 1 {
		t.Fatalf("an unchanged legacy ask is left alone: %+v, %d raised", s, len(r.hub.raised))
	}

	// The item changes: the keyed ask opens next to it, then the legacy one closes.
	r.setItems(t, strings.Replace(items, "yt-dlp target choice", "yt-dlp target choice (revised)", 1))
	if s := r.step(r.pass(t), key); s.Action != ActRefreshAsk {
		t.Fatalf("the changed item is raised again with its subject key: %+v", s)
	}
	if r.hub.open(key) != 2 {
		t.Fatalf("until the next pass the old and the new ask are both open: %d", r.hub.open(key))
	}
	var closed bool
	for _, s := range r.pass(t).Steps {
		closed = closed || (s.Key == key && s.Item == nil && s.Action == ActWithdraw && s.Done)
	}
	if !closed || old.State != "withdrawn" || r.hub.withdrawn[old.ID] != withdrawDuplicate {
		t.Fatalf("the old ask is withdrawn as replaced: %+v %v", old, r.hub.withdrawn)
	}
	if r.hub.open(key) != 1 || r.hub.asks[key].State != "pending" {
		t.Fatalf("one open ask is left, the new one: %d", r.hub.open(key))
	}
	raised := len(r.hub.raised)
	r.pass(t)
	if len(r.hub.raised) != raised || len(r.hub.withdrawn) != 1 {
		t.Fatalf("the next pass is quiet: %d raised, %v", len(r.hub.raised), r.hub.withdrawn)
	}
}

// A legacy ask is closed by its request id when its item is answered in the desk.
func TestALegacyAskIsWithdrawnWhenItsItemIsAnswered(t *testing.T) {
	r := newRig(t, "desk")
	key := AskPrefix + commission + "/AV4"
	r.pass(t)
	old := r.hub.asks[key]
	old.RequesterRef = "event-id-before-subject-keys"
	r.hub.unkeyed[old.ID] = true
	r.answerInDesk(t)
	r.pass(t)
	if old.State != "withdrawn" || r.hub.withdrawn[old.ID] != withdrawAnswered {
		t.Fatalf("the legacy ask closes: %+v %v", old, r.hub.withdrawn)
	}
}

// A Hub-side withdrawal is no answer: the item still waits, nothing is written,
// and the desk does not raise the ask again.
func TestAWithdrawnHubAskIsNeitherAnAnswerNorRaisedAgain(t *testing.T) {
	r := newRig(t, "desk")
	key := AskPrefix + commission + "/AV4"
	r.pass(t)
	r.hub.asks[key].State = "withdrawn"
	raised := len(r.hub.raised)
	plan := r.pass(t)
	s := r.step(plan, key)
	if s.Action != ActNone || !s.Waiting() || len(r.hub.raised) != raised || len(r.operatorFile(t)) != 0 {
		t.Fatalf("a withdrawn ask leaves the item waiting, to be answered in the desk: %+v", s)
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

// Turning the courier on never sends an item that was already there.
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

// A scheduler's `Delivered:` line stops the desk from sending an item, not
// from keeping its own open ask current: the line itself changes nothing the
// operator reads, and a changed item still updates the ask in place.
func TestAScheduledDeliveryStillKeepsTheOpenAskCurrent(t *testing.T) {
	r := newRig(t, "desk")
	key := AskPrefix + commission + "/AV4"
	r.pass(t)
	delivered := strings.Replace(items, "Keep or propose.\n", "Keep or propose.\nDelivered: hub 2026-11-12T09:00:00Z\n", 1)
	r.setItems(t, delivered)
	r.clock.Advance(time.Minute)
	if s := r.step(r.pass(t), key); s.Action != ActNone || len(r.hub.raised) != 1 {
		t.Fatalf("a Delivered: line alone does not change the ask: %+v raised %d", s, len(r.hub.raised))
	}
	r.setItems(t, strings.Replace(delivered, "yt-dlp target choice", "Keep the video download tool uninstalled?", 1))
	r.clock.Advance(time.Minute)
	if s := r.step(r.pass(t), key); s.Action != ActRefreshAsk || !s.Done {
		t.Fatalf("the changed item updates its ask though its scheduler delivered it: %+v", s)
	}
	if r.hub.open(key) != 1 || r.hub.asks[key].Question != "Keep the video download tool uninstalled?" {
		t.Fatalf("one open ask carries the new question: %+v", r.hub.asks[key])
	}
}

func TestAnOlderCharterIsLeftToItsSchedulerForTwoHours(t *testing.T) {
	r := newRig(t, "")
	r.setItems(t, "")
	if _, err := r.courier.Set(context.Background(), commission, true); err != nil {
		t.Fatal(err)
	}
	// The items appear after the switch was turned on.
	r.clock.Advance(time.Minute)
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

func TestKeysReadBackFromTheDeskKeyLine(t *testing.T) {
	for _, c := range []struct{ question, technical, want string }{
		{"Choose the off-host backup destination", "Desk key: " + AskPrefix + "free-disk/DB1-OFFHOST-SINK\nPacket: /x", AskPrefix + "free-disk/DB1-OFFHOST-SINK"},
		{"Anything", "Research commission: c\nDesk key: " + TestPrefix + "20261112T010000Z", TestPrefix + "20261112T010000Z"},
		{"Anything", "Desk key: someone-else/x", ""},
		// An ask raised before the desk wrote the line is found by its question.
		{"programme-three-20261111 · YTDLP-TARGET — yt-dlp target", "", AskPrefix + "programme-three-20261111/YTDLP-TARGET"},
		{testQuestion + "20261112T010000Z — no answer needed", "", TestPrefix + "20261112T010000Z"},
		{"Can I extend this window?", "", ""},
		{"a b · c — d", "", ""},
	} {
		if got := keyOf(c.question, c.technical); got != c.want {
			t.Errorf("keyOf(%q, %q) = %q, want %q", c.question, c.technical, got, c.want)
		}
	}
}

// The operator reads an ask cold: the question is the item's title and the
// big picture its prose; the desk's key, IDs and paths are technical details.
func TestAnAskSpeaksPlainlyAndKeepsIDsInTechnicalDetails(t *testing.T) {
	text := `
## Needs you

### DB1-SINK — Choose where off-site backups are kept
Backups now live only on this computer. Pick where an encrypted copy goes.
Recommended: s3 — Use the storage account you already have.
Reason: It is already approved and costs nothing new.
Options: s3 — Use the storage account you already have; host — Use another of your computers
Packet: /home/x/work/DB1-SINK/BRIEF.md
Details: plan F12; evidence runs/r1
Delivered: hub 2026-11-13T00:00:00Z
`
	items, _ := home.ParseNeedsYou(text, "Needs you", "/home/x/STATE.md")
	spec := askSpec(&home.FeedItem{Item: items[0], CommissionID: "steward-x"}, AskPrefix+"steward-x/DB1-SINK", Pages{})
	if spec.Question != "Choose where off-site backups are kept" || spec.Subject != spec.Question {
		t.Fatalf("the question is the title alone: %q / %q", spec.Question, spec.Subject)
	}
	if spec.BigPicture != "Backups now live only on this computer. Pick where an encrypted copy goes." {
		t.Fatalf("the big picture is the prose without field lines: %q", spec.BigPicture)
	}
	if spec.Reason != "It is already approved and costs nothing new." {
		t.Fatalf("a Reason: line is the recommendation reason: %q", spec.Reason)
	}
	if facts, err := askFacts(spec); err != nil || len(operatordecision.CoverProblems(facts)) > 0 {
		t.Fatalf("a cold reader cannot parse the cover: %v %v", err, operatordecision.CoverProblems(facts))
	}
	for _, want := range []string{"Desk key: " + AskPrefix + "steward-x/DB1-SINK", "Research commission: steward-x", "Needs-you item: DB1-SINK", "Item file: /home/x/STATE.md", "Packet: /home/x/work/DB1-SINK/BRIEF.md", "Details: plan F12; evidence runs/r1"} {
		if !strings.Contains(spec.TechnicalDetails, want) {
			t.Errorf("technical details lack %q:\n%s", want, spec.TechnicalDetails)
		}
	}
	if keyOf(spec.Question, spec.TechnicalDetails) != spec.Key {
		t.Fatal("the desk finds the ask again by its technical details")
	}
	items[0].Reason = ""
	if got := askSpec(&home.FeedItem{Item: items[0], CommissionID: "steward-x"}, AskPrefix+"steward-x/DB1-SINK", Pages{}).Reason; got != "Use the storage account you already have." {
		t.Fatalf("without a Reason: line the recommended label explains: %q", got)
	}
}

// The courier's health is its real state, and it says when an ask closes.
func TestHealthIsTheLastPassAndSaysWhenAnAskCloses(t *testing.T) {
	if ok, said := (Health{}).Say(); ok || !strings.Contains(said, "no pass yet") {
		t.Fatalf("before the first pass: %v %q", ok, said)
	}
	r := newRig(t, "")
	r.pass(t)
	if ok, said := r.courier.Health().Say(); !ok || !strings.Contains(said, "answers read back") || !strings.Contains(said, "last delivery never") {
		t.Fatalf("the Hub is read every pass, though nothing was sent: %v %q", ok, said)
	}
	r = newRig(t, "desk")
	r.pass(t)
	h := r.courier.Health()
	if ok, said := h.Say(); !ok || h.LastDelivery.IsZero() || h.LastReadBack.IsZero() || !strings.Contains(said, Withdraw) {
		t.Fatalf("after a delivery: %+v %q", h, said)
	}
	// A restarted courier remembers nothing it sent, but a delivery recorded
	// in OPERATOR.md is still a delivery: never "never".
	at := r.clock.Now().Add(-time.Hour).Truncate(time.Second)
	if _, err := r.courier.Writer.AppendOperator(commission, home.Entry{At: at, Kind: home.EntryDelivered, Subject: "DISK-HEADROOM", Via: "desk", Body: "hub notice n-1"}); err != nil {
		t.Fatal(err)
	}
	restarted := &Courier{Reader: r.courier.Reader, Writer: r.courier.Writer, Hub: r.hub, Clock: r.clock, Logger: r.courier.Logger}
	r.pass(t)
	if _, err := restarted.Pass(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ok, said := restarted.Health().Say(); !ok || strings.Contains(said, "last delivery never") || restarted.Health().LastDelivery.Before(at) {
		t.Fatalf("a recorded delivery survives a restart: %q", said)
	}
	r.hub.readErr = errors.New("unauthenticated: verified identity required")
	r.clock.Advance(time.Minute)
	r.pass(t)
	if ok, said := r.courier.Health().Say(); ok || !strings.Contains(said, "unauthenticated") || !strings.Contains(said, "answers last read back "+stamp(h.LastReadBack)) {
		t.Fatalf("a refused read-back: %v %q", ok, said)
	}
}

// An ID the supervisor reuses for a new decision after the earlier one
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
// While the courier cannot read a commission's answers it sends,
// records and forgets nothing for it, and says why; once readable, a Hub
// answer given meanwhile is recorded once.
func TestNothingIsDeliveredWhileAnswersCannotBeRead(t *testing.T) {
	r := newRig(t, "desk")
	r.pass(t)
	if len(r.hub.raised) != 1 {
		t.Fatalf("AV4 is asked first: %+v", r.hub.raised)
	}
	sightedBefore, _ := os.ReadFile(filepath.Join(r.courier.Writer.StateDir(), SightingsFile))
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
	if after, _ := os.ReadFile(filepath.Join(r.courier.Writer.StateDir(), SightingsFile)); string(after) != string(sightedBefore) {
		t.Errorf("first sightings are kept, not forgotten:\nbefore %s\nafter  %s", sightedBefore, after)
	}

	unlock()
	r.clock.Advance(time.Minute)
	r.pass(t)
	r.pass(t)
	if entries := r.operatorFile(t); len(entries) != 1 || entries[0].Kind != home.EntryAnswer || entries[0].Subject != "AV4" {
		t.Fatalf("the Hub answer is recorded once when OPERATOR.md is readable again: %+v", entries)
	}
}

// An unreadable STATE.md hides its items; they are not taken as gone,
// so the two-hour wait for an older charter's scheduler runs from when the
// item first appeared, not from when STATE.md could be read again.
func TestAnUnreadableStateIsNotTakenAsNoItems(t *testing.T) {
	r := newRig(t, "")
	if _, err := r.courier.Set(context.Background(), commission, true); err != nil {
		t.Fatal(err)
	}
	r.clock.Advance(time.Minute)
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

func TestAnItemWaitsUntilAnswered(t *testing.T) {
	answer := &home.Entry{Kind: home.EntryAnswer, Subject: "X"}
	cases := []struct {
		name string
		step Step
		want bool
	}{
		{"no answer, no ask", Step{Item: &home.FeedItem{}}, true},
		{"answered in OPERATOR.md", Step{Item: &home.FeedItem{Answer: answer}}, false},
		{"answered in the Hub", Step{Item: &home.FeedItem{}, Ask: &Ask{State: "answered"}}, false},
		{"ask pending", Step{Item: &home.FeedItem{}, Ask: &Ask{State: "pending"}}, true},
		{"ask expired", Step{Item: &home.FeedItem{}, Ask: &Ask{State: "expired"}}, true},
		{"the answer belongs to an earlier item", Step{Item: &home.FeedItem{Answer: answer}, Reused: true}, true},
		{"an orphaned ask", Step{Ask: &Ask{State: "pending"}}, false},
	}
	for _, tc := range cases {
		if got := tc.step.Waiting(); got != tc.want {
			t.Errorf("%s: waiting = %v", tc.name, got)
		}
	}
}
