package launch

import "testing"

// A manager answers the stance in its own words, not always with the bare
// word. This is the screen of steward-channel-research-supervision-20261123,
// whose "Ready. I read the charter..." was rejected by the old exact-line
// match and left an idle session with no goal.
func TestRepliedReadyAcceptsASentenceAfterTheWord(t *testing.T) {
	screen := Screen{Text: "› Research commission: Program Runtime effectiveness and adoption.\n" +
		"\n" +
		"• Ready. I read the charter; `STATE.md` is not present yet. I’ll wait for your\n" +
		"  goal before starting the commission.\n" +
		"\n" +
		"› Ask Codex to do anything\n" +
		"[wc-268e2c0:codex*                       \"Decide on commission \" 14:12 23-Nov-26]\n"}
	if !repliedReady(screen) {
		t.Fatal("a Codex reply that begins \"Ready.\" was not recognized")
	}
}

func TestRepliedReadyDoesNotMatchTheStancesOwnInstruction(t *testing.T) {
	screen := Screen{Text: "The goal follows this message. Reply \"ready\" and wait for it.\n" +
		"\n" +
		"› Ask Codex to do anything\n"}
	if repliedReady(screen) {
		t.Fatal("the stance's instruction to reply ready was mistaken for the reply")
	}
}

func TestRepliedReadyStillAcceptsTheBareWord(t *testing.T) {
	for _, text := range []string{"ready\n", "• Ready.\n", "• READY!\n", "- \"ready\"\n"} {
		if !repliedReady(Screen{Text: text}) {
			t.Fatalf("%q was not recognized as a ready reply", text)
		}
	}
}
