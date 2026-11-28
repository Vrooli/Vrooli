package courier

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"research-desk/internal/home"
)

// Every answer lands at the level that owns its item (docs/agent-system/LEVELS.md):
// the owning level's INBOX.md, which its next agent drains whether or not
// anything is running when the operator answers, and for a commission's item
// also the commission's OPERATOR.md.

const (
	goneItem  = "GONE-1"
	operators = "There will be no off-host backup. We only have the external drive."
	topic     = "local-model-performance"
)

// endCommission discards the rig's commission in LAUNCH.json, which ends it.
func (r *rig) endCommission(t *testing.T) {
	t.Helper()
	record := &home.LaunchRecord{Schema: home.LaunchSchema, CommissionID: commission, LaunchedBy: "desk", Discarded: &home.Discard{At: r.clock.Now(), Reason: "allowance ended"}}
	if _, err := r.courier.Writer.WriteLaunch(commission, record); err != nil {
		t.Fatal(err)
	}
}

// addCommission creates a live, desk-launched commission working on topic.
func (r *rig) addCommission(t *testing.T, id string) {
	t.Helper()
	r.write(t, filepath.Join(r.research, "cohorts", id, home.StateFile), []byte("# State\n"))
	r.write(t, filepath.Join(r.research, "topics", topic, "runs", id, "CHECKPOINT.md"), []byte("# Checkpoint\n"))
	if _, err := r.courier.Writer.WriteLaunch(id, &home.LaunchRecord{Schema: home.LaunchSchema, CommissionID: id, LaunchedBy: "desk"}); err != nil {
		t.Fatal(err)
	}
}

// answeredAfterTheEnd is the Hub's record of an ask for an item that is in no
// file, answered in the operator's own words.
func (r *rig) answeredAfterTheEnd() (string, *Ask) {
	key := AskPrefix + commission + "/" + goneItem
	ask := &Ask{ID: "ask-1", Key: key, State: "answered", Answer: "other", Note: operators, AnsweredAt: r.clock.Now().Add(-time.Hour)}
	r.hub.asks[key] = ask
	return key, ask
}

// landed is the answer entry for GONE-1 in the asking commission's OPERATOR.md.
func (r *rig) landed(t *testing.T) []home.Entry {
	t.Helper()
	var out []home.Entry
	for _, e := range r.operatorFile(t) {
		if e.Kind == home.EntryAnswer && e.Subject == goneItem {
			out = append(out, e)
		}
	}
	return out
}

func (r *rig) inbox(t *testing.T, folder string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(folder, home.InboxFile))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(data)
}

func (r *rig) topicDir(name string) string { return filepath.Join(r.research, "topics", name) }

// answerLine is the inbox entry the grammar names (topic-folder.md): heading,
// `<ID>: <key> — <label>`, the note, the packet, the ask.
func answerEntry(id, answer string, extra ...string) string {
	return strings.Join(append([]string{"· from operator · answer\n" + id + ": " + answer}, extra...), "\n")
}

func TestAnAnswerAfterTheCommissionEndedLandsInItsTopicsInboxOnce(t *testing.T) {
	r := newRig(t, "desk")
	r.endCommission(t)
	key, ask := r.answeredAfterTheEnd()
	ask.AttachmentNotes = []AttachmentNote{{Attachment: "redesign", Path: "regions/settings-nav.json", Note: "Keep the settings link reachable by thumb."}}

	plan := r.pass(t)
	if s := r.step(plan, key); s.Action != ActLandAnswer || !s.Done || len(s.Inboxes) != 1 || s.Inboxes[0].Topic != topic {
		t.Fatalf("the answer lands in the topic's inbox: %+v", s)
	}
	if msg := problem(plan, nil); msg != "" {
		t.Fatalf("a landed answer is not a problem: %s", msg)
	}
	if got := r.landed(t); len(got) != 1 || !strings.HasPrefix(got[0].Body, "other — "+operators) || !strings.Contains(got[0].Body, `"path":"regions/settings-nav.json"`) || !strings.Contains(got[0].Body, `"note":"Keep the settings link reachable by thumb."`) {
		t.Fatalf("the ended commission's answer record holds the answer: %+v", got)
	}
	inbox := r.inbox(t, r.topicDir(topic))
	if want := answerEntry(goneItem, "other", "Note: "+operators, "Source: hub ask ask-1", `Attachment notes (exact path and text, JSON): [{"attachment":"redesign","path":"regions/settings-nav.json","note":"Keep the settings link reachable by thumb."}]`); !strings.HasPrefix(inbox, "# Inbox — "+topic) || !strings.Contains(inbox, want) {
		t.Fatalf("the inbox entry is in the grammar:\n%s\nwant\n%s", inbox, want)
	}

	// The topic's next agent drains the inbox into its history; nothing comes back.
	r.write(t, filepath.Join(r.topicDir(topic), "history", home.InboxFile), []byte(inbox))
	if err := os.Remove(filepath.Join(r.topicDir(topic), home.InboxFile)); err != nil {
		t.Fatal(err)
	}
	plan = r.pass(t)
	if s := r.step(plan, key); s.Key != "" || r.inbox(t, r.topicDir(topic)) != "" || len(r.landed(t)) != 1 {
		t.Fatalf("a landed answer is never written again: %+v", s)
	}
}

func TestExactPacketRegionNotesAreRejectedBeforeTheyCouldBeTruncated(t *testing.T) {
	r := newRig(t, "desk")
	ask := &Ask{ID: "ask-large", Answer: "keep", AttachmentNotes: []AttachmentNote{{
		Attachment: "redesign", Path: "regions/nav-main.json", Note: strings.Repeat("x", 4100),
	}}}
	step := &Step{Key: AskPrefix + commission + "/" + goneItem, Ask: ask}
	err := r.courier.recordAnswer(step, r.clock.Now())
	if err == nil || !strings.Contains(err.Error(), "exact Hub attachment notes exceed") {
		t.Fatalf("oversized exact notes must fail visibly instead of truncating: %v", err)
	}
	if got := r.operatorFile(t); len(got) != 0 {
		t.Fatalf("no partial commission answer is recorded: %+v", got)
	}
}

// O7 showcase (a), end to end at the courier's Hub seam: a commission raises a
// fresh decision, the desk sends it, the commission ends before the operator
// answers, and the answer reaches the topic, where the next commission on the
// topic reads it from the topic's inbox.
func TestAFreshDecisionWhoseCommissionEndsBeforeTheAnswerReachesTheNextCommission(t *testing.T) {
	r := newRig(t, "desk")
	r.pass(t) // the record of first sightings begins: AV4 and AV5 were there
	packet := filepath.Join(r.topicDir(topic), "work", "FRESH-1", "BRIEF.md")
	folder := r.ready(t, filepath.Dir(packet), "keep", "drop")
	r.clock.Advance(time.Minute)
	r.setItems(t, items+"\n### FRESH-1 — Keep the local model?\nWhy it is yours.\nRecommended: keep — Keep it\nOptions: keep — Keep it; drop — Drop it\n"+folder)
	r.pass(t)
	key := AskPrefix + commission + "/FRESH-1"
	ask := r.hub.asks[key]
	if ask == nil || !strings.Contains(ask.TechnicalDetails, "Packet: "+packet) || !strings.Contains(ask.TechnicalDetails, "Answer lands in: "+filepath.Join(r.topicDir(topic), home.InboxFile)) {
		t.Fatalf("the fresh decision is sent with its packet and where its answer lands: %+v", ask)
	}

	r.endCommission(t)
	r.clock.Advance(time.Hour)
	ask.State, ask.Answer, ask.AnswerLabel, ask.Note, ask.AnsweredAt = "answered", "drop", "Drop it", "It never helped.", r.clock.Now()
	plan := r.pass(t)
	if s := r.step(plan, key); s.Action != ActLandAnswer || !s.Done {
		t.Fatalf("the answer lands after the commission ended: %+v", s)
	}

	next := "local-models-w2-20261116"
	r.addCommission(t, next)
	snap, err := r.courier.Reader.Snapshot(context.Background(), home.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if c := snap.Commission(next); c == nil || c.State == home.StateEnded || len(c.AnswerTopics(nil)) != 1 || c.AnswerTopics(nil)[0] != topic {
		t.Fatalf("the next commission works on the topic: %+v", c)
	}
	got := snap.Level(home.TopicOwner(topic)).AnswerTo("FRESH-1")
	if got == nil || got.Drained || !strings.Contains(got.Body, "FRESH-1: drop — Drop it\nNote: It never helped.\nPacket: "+packet+"\nSource: hub ask "+ask.ID) {
		t.Fatalf("the next commission finds the answer and its packet in the topic's inbox: %+v", got)
	}
}

// A topic's own decision (DECISIONS.md) needs no commission at all: it is sent,
// and its answer lands in the topic's inbox, which marks it answered.
func TestATopicsDecisionIsSentAndAnsweredWithNoCommission(t *testing.T) {
	r := newRig(t, "desk")
	r.endCommission(t)
	r.pass(t)
	r.clock.Advance(time.Minute)
	packet := filepath.Join(r.topicDir(topic), "work", "LM-PICK", "BRIEF.md")
	r.ready(t, filepath.Dir(packet))
	r.write(t, packet, []byte("# LM-PICK\n\nChecked: 2026-11-12 the model still runs on the CPU, read in the benchmark log\n\n## On each answer\n\n- **`gpu` (recommended).** Move the model to the GPU host and rerun the benchmark.\n- **`cpu`.** Keep it on the CPU.\n- `other`: the operator's words are the instruction.\n"))
	r.write(t, filepath.Join(r.topicDir(topic), home.DecisionsFile), []byte("# Decisions\n\n## Needs you\n\n### LM-PICK — Where should the model run?\nTwo plain sentences.\nRecommended: gpu — On the GPU host\nOptions: gpu — On the GPU host; cpu — On the CPU\nPacket: work/LM-PICK/BRIEF.md\nExplainer: work/LM-PICK/page.html\n"))
	plan := r.pass(t)
	key := AskPrefix + home.TopicOwner(topic) + "/LM-PICK"
	if s := r.step(plan, key); s.Action != ActAsk || !s.Done || s.Item.Owner() != home.TopicOwner(topic) {
		t.Fatalf("the topic's decision is sent though no commission is live: %+v", s)
	}
	if got := r.hub.raised[len(r.hub.raised)-1].Explainer; got != page {
		t.Fatalf("the decision goes with the page beside its brief:\n%s", got)
	}

	ask := r.hub.asks[key]
	ask.State, ask.Answer, ask.AnswerLabel = "answered", "gpu", "On the GPU host"
	plan = r.pass(t)
	if s := r.step(plan, key); s.Action != ActRecordAnswer || !s.Done {
		t.Fatalf("the answer is recorded: %+v", s)
	}
	if inbox := r.inbox(t, r.topicDir(topic)); !strings.Contains(inbox, answerEntry("LM-PICK", "gpu — On the GPU host", "Packet: "+packet, "Source: hub ask "+ask.ID)) {
		t.Fatalf("the answer is in the topic's inbox:\n%s", inbox)
	}
	plan = r.pass(t)
	if s := r.step(plan, key); s.Action != ActNone || s.Delivery != home.DeliverNone || strings.Count(r.inbox(t, r.topicDir(topic)), "LM-PICK:") != 1 {
		t.Fatalf("the inbox entry marks it answered, once: %+v", s)
	}
}

// Moving a commission's item into its topic's DECISIONS.md, or the commission
// ending, never asks the operator twice: the open ask stays the item's.
func TestACopyOfAnItemInItsTopicNeverAsksTwice(t *testing.T) {
	r := newRig(t, "desk")
	r.pass(t)
	r.clock.Advance(time.Minute)
	brief := filepath.Join(r.topicDir(topic), "work", "MOVE-1", "BRIEF.md")
	item := "\n## Needs you\n\n### MOVE-1 — Move it?\nWhy.\nRecommended: yes — Yes\nOptions: yes — Yes; no — No\n" + r.ready(t, filepath.Dir(brief), "yes", "no")
	r.setItems(t, item)
	r.pass(t)
	move := AskPrefix + commission + "/MOVE-1"
	if r.hub.open(move) != 1 {
		t.Fatalf("one ask: %+v", r.hub.raised)
	}
	r.write(t, filepath.Join(r.topicDir(topic), home.DecisionsFile), []byte("# Decisions\n"+item))
	r.pass(t)
	r.endCommission(t)
	plan := r.pass(t)
	for _, spec := range r.hub.raised {
		if spec.Key != move && strings.HasSuffix(spec.Key, "/MOVE-1") {
			t.Fatalf("a second decision was raised for the item: %+v", spec)
		}
	}
	if a := r.hub.asks[move]; r.hub.open(move) != 1 || r.hub.withdrawn[a.ID] != "" {
		t.Fatalf("the copy and the commission's end leave the one ask open (updated in place at most): %v", r.hub.withdrawn)
	}
	s := r.step(plan, move)
	if s.Item == nil || s.Item.Owner() != home.TopicOwner(topic) {
		t.Fatalf("the topic owns the item now, through the same ask: %+v", s)
	}
	a := r.hub.asks[move]
	a.State, a.Answer, a.AnswerLabel = "answered", "yes", "Yes"
	r.pass(t)
	if inbox := r.inbox(t, r.topicDir(topic)); !strings.Contains(inbox, "MOVE-1: yes — Yes") {
		t.Fatalf("the answer lands in the topic's inbox:\n%s", inbox)
	}
}

// The supervisor's own decision (ATTENTION.md without `Commission:`) goes out
// unless the operator turned it off, and its answer lands in the research
// home's inbox.
func TestTheSupervisorsDecisionAnswerLandsInTheResearchHomesInbox(t *testing.T) {
	r := newRig(t, "desk")
	r.pass(t)
	r.clock.Advance(time.Minute)
	attention := filepath.Join(r.research, home.AttentionFile)
	old, _ := os.ReadFile(attention)
	r.ready(t, filepath.Join(r.research, "work", "SPLIT-TOPICS"), "split", "keep")
	r.write(t, attention, []byte(strings.Replace(string(old), "## Actionable now\n", "## Actionable now\n\n### SPLIT-TOPICS — Split the crowded topic in two?\nWhy.\nRecommended: split — Split it\nOptions: split — Split it; keep — Keep it\nPacket: work/SPLIT-TOPICS/BRIEF.md\nExplainer: work/SPLIT-TOPICS/page.html\n", 1)))
	plan := r.pass(t)
	key := AskPrefix + home.SupervisorOwner + "/SPLIT-TOPICS"
	if s := r.step(plan, key); s.Action != ActAsk || !s.Done {
		t.Fatalf("the supervisor's decision is sent: %+v", s)
	}
	if got := r.hub.raised[len(r.hub.raised)-1].Explainer; got != page {
		t.Fatalf("its page is the one beside its brief:\n%s", got)
	}
	a := r.hub.asks[key]
	a.State, a.Answer, a.AnswerLabel = "answered", "split", "Split it"
	r.pass(t)
	if inbox := r.inbox(t, r.research); !strings.Contains(inbox, "SPLIT-TOPICS: split — Split it") {
		t.Fatalf("the answer lands in the research home's inbox:\n%s", inbox)
	}
	if _, err := r.courier.Set(context.Background(), "", false); err != nil {
		t.Fatal(err)
	}
	sw, _ := r.courier.Switches()
	if on, _ := sw.Supervisor(); on {
		t.Fatal("the operator turns the manager's decisions off")
	}
}

func TestAPassThatFailsPartWayRepeatsOnlyWhatDidNotLand(t *testing.T) {
	r := newRig(t, "desk")
	r.endCommission(t)
	key, _ := r.answeredAfterTheEnd()
	// The commission folder takes no new file: the inbox entry lands, the
	// answer record cannot be created.
	dir := filepath.Join(r.research, "cohorts", commission)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o755)
	plan, err := r.courier.Pass(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s := r.step(plan, key); s.Err == nil || !strings.Contains(problem(plan, nil), "land-answer") {
		t.Fatalf("the failed record is reported: %+v", s)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	r.pass(t)
	if n := strings.Count(r.inbox(t, r.topicDir(topic)), "ask-1"); n != 1 || len(r.landed(t)) != 1 {
		t.Fatalf("the inbox holds the answer once (%d) and the record lands on the next pass", n)
	}
}

func TestAnAnswerGoesToTheTopicItsPacketIsIn(t *testing.T) {
	r := newRig(t, "desk")
	other := "niche-book-production"
	// The commission also worked on a second topic.
	r.write(t, filepath.Join(r.topicDir(other), "runs", commission, "CHECKPOINT.md"), []byte("# Checkpoint\n"))
	packet := filepath.Join(r.topicDir(other), "work", goneItem, "BRIEF.md")
	r.setItems(t, "\n## Needs you\n\n### "+goneItem+" — Off-host backup\nWhy.\nRecommended: keep — Keep\nOptions: keep — Keep; drop — Drop\n"+r.ready(t, filepath.Dir(packet), "keep", "drop"))
	r.endCommission(t)
	key, ask := r.answeredAfterTheEnd()
	ask.TechnicalDetails = deskKeyLine(key) + "\nPacket: " + packet

	plan := r.pass(t)
	if s := r.step(plan, key); len(s.Inboxes) != 1 || s.Inboxes[0].Topic != other {
		t.Fatalf("the answer goes to the topic holding the item's packet: %+v", s)
	}
	if inbox := r.inbox(t, r.topicDir(other)); !strings.Contains(inbox, "Packet: "+packet) {
		t.Fatalf("the inbox entry names the item's packet:\n%s", inbox)
	}
	if got := r.inbox(t, r.topicDir(topic)); got != "" {
		t.Fatalf("the other topic gets nothing: %q", got)
	}
}

func TestAnAnswerAlreadyInItsCommissionIsLeftAlone(t *testing.T) {
	r := newRig(t, "desk")
	r.endCommission(t)
	key, _ := r.answeredAfterTheEnd()
	// The answer was recorded while the item was live.
	if _, err := r.courier.Writer.AppendOperator(commission, home.Entry{At: r.clock.Now(), Kind: home.EntryAnswer, Subject: goneItem, Via: home.ViaHub, Body: "other — " + operators}); err != nil {
		t.Fatal(err)
	}
	plan := r.pass(t)
	if s := r.step(plan, key); s.Key != "" || r.inbox(t, r.topicDir(topic)) != "" {
		t.Fatalf("an answer already recorded needs nothing: %+v", s)
	}
}

func TestAnAnswerForAnItemThatLeftALiveCommissionLandsInItAndItsTopic(t *testing.T) {
	r := newRig(t, "desk")
	key, _ := r.answeredAfterTheEnd()

	plan := r.pass(t)
	if s := r.step(plan, key); s.Action != ActLandAnswer || len(s.Inboxes) != 1 || !s.Done {
		t.Fatalf("a live commission reads its own answer record; the topic keeps it too: %+v", s)
	}
	if got := r.landed(t); len(got) != 1 || !strings.Contains(r.inbox(t, r.topicDir(topic)), goneItem+": other") {
		t.Fatalf("OPERATOR.md and the topic's inbox hold the answer once: %+v", got)
	}
}

func TestNothingLandsWhileTheTopicsInboxCannotBeRead(t *testing.T) {
	r := newRig(t, "desk")
	r.endCommission(t)
	key, _ := r.answeredAfterTheEnd()
	inbox := filepath.Join(r.topicDir(topic), home.InboxFile)
	r.write(t, inbox, []byte("# Inbox\n"))
	unlock := lock(t, inbox)
	defer unlock()

	plan := r.pass(t)
	if s := r.step(plan, key); s.Action != ActNone || !strings.Contains(s.Why, "cannot be read") {
		t.Fatalf("nothing is landed while the inbox cannot be read: %+v", s)
	}
	if len(r.landed(t)) != 0 {
		t.Fatal("nothing written")
	}
	if msg := problem(plan, nil); msg == "" {
		t.Fatal("the pass reports why it waits")
	}
}
