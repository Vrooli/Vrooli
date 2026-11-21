package home

import (
	"os"
	"path/filepath"
	"testing"
)

func decision(id string) string {
	return "### " + id + " — Pick?\nWhy.\nRecommended: yes — Yes\nOptions: yes — Yes; no — No\nPacket: work/" + id + "/BRIEF.md\n"
}

// A level keeps its own decisions and reads its answers from its own inbox, so
// what is open there needs no commission (docs/agent-system/LEVELS.md).
func TestALevelsDecisionsAndAnswersLiveInItsFolder(t *testing.T) {
	f := newFixture(t)
	topic := "niche-book-production"
	f.write(t, "topics/"+topic+"/DECISIONS.md", "# Decisions\n\n## Needs you\n\n"+decision("OPEN-1")+decision("DONE-1"))
	f.write(t, "topics/"+topic+"/history/INBOX.md", "# Inbox history\n\n### 2026-11-15T09:00Z · from operator · answer\nDONE-1: no — No\nSource: hub ask a-1\nDrained 2026-11-15T10:00:00Z: built it\n")
	attention, _ := os.ReadFile(filepath.Join(f.research, AttentionFile))
	f.write(t, AttentionFile, string(attention)+"\n## Actionable now\n\n"+decision("OPEN-1")+decision("SUP-1"))
	f.write(t, "INBOX.md", "# Inbox\n\n### 2026-11-15T11:00:00Z · from operator · answer\nSUP-1: yes — Yes\nSource: hub ask a-2\n")

	feed := f.snapshot(t, Options{}).NeedsYou()
	got := map[string]FeedItem{}
	for _, it := range feed.Items {
		got[it.Key()] = it
	}
	open := got[TopicOwner(topic)+"/OPEN-1"]
	if open.Topic != topic || len(open.Sources) != 2 || open.Answer != nil || len(open.Inboxes) != 1 || open.Inboxes[0].Dir != filepath.Join(f.research, "topics", topic) {
		t.Fatalf("the topic owns its item, and the supervisor's copy is the same item: %+v", open)
	}
	if d, _, _ := open.Delivery(); d != DeliverAsk {
		t.Fatalf("a topic's decision is asked with no commission behind it: %s", d)
	}
	if want := filepath.Join(f.research, "topics", topic, "work", "OPEN-1", BriefFile); open.Item.PacketPath() != want {
		t.Fatalf("the packet resolves beside DECISIONS.md: %s", open.Item.PacketPath())
	}
	done := got[TopicOwner(topic)+"/DONE-1"]
	if done.Answer == nil || done.Answer.Subject != "DONE-1" || !done.Read.Value {
		t.Fatalf("a drained answer in the topic's history answers it, and was read: %+v", done)
	}
	sup := got[SupervisorOwner+"/SUP-1"]
	if sup.Answer == nil || sup.Read.Value || len(sup.Inboxes) != 1 || sup.Inboxes[0].Dir != f.research {
		t.Fatalf("the supervisor's item is answered in the research home's inbox, not drained yet: %+v", sup)
	}
	if ok, why := sup.Answerable(); !ok {
		t.Fatalf("the supervisor's item can be answered: %s", why)
	}
}

func TestAnAnswerEntryNamesItsItemByTheGrammarOrTheOlderDeskSentence(t *testing.T) {
	cases := map[string]bool{
		"CF3-SECOND-NODE-001: other\nNote: It is online.":                                        true,
		"The operator answered \"Reconnect\" (item CF3-SECOND-NODE-001 of steward-x, Hub ask 1)": true,
		"CF3-SECOND-NODE-0012: yes — Yes":                                                        false,
		"Source: hub ask CF3-SECOND-NODE-001-and-more":                                           false,
		"See CF3-SECOND-NODE-001 for what changed, no answer":                                    false,
	}
	for body, want := range cases {
		if got := (InboxAnswer{Entry: Entry{Body: body}}).Names("CF3-SECOND-NODE-001"); got != want {
			t.Errorf("%q names CF3-SECOND-NODE-001: %v, want %v", body, got, want)
		}
	}
}
