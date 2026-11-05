package mentions

import "testing"

func validDelivery() Delivery {
	return Delivery{Provider: "github", EventID: "evt-1", ActorID: "alice", ThreadID: "thread-1", RepositoryID: "repo-1", HeadRevision: "head-1", Body: "@vrooli review", Authenticated: true, ActorCanRead: true, ReplyPermitted: true}
}

func TestParseRejectsForgedOrBotDelivery(t *testing.T) {
	d := validDelivery()
	d.Authenticated = false
	if _, err := Parse(d); err == nil {
		t.Fatal("forged delivery accepted")
	}
	d = validDelivery()
	d.BotActor = true
	if _, err := Parse(d); err == nil {
		t.Fatal("bot loop accepted")
	}
}

func TestParseAcceptsSmallCommandSetAndDedupKey(t *testing.T) {
	r, err := Parse(validDelivery())
	if err != nil {
		t.Fatal(err)
	}
	if r.Command != CommandReview || r.Key != "github:evt-1" {
		t.Fatalf("unexpected request: %+v", r)
	}
}

func TestParseGivesEditedEventItsOwnDeduplicationIdentity(t *testing.T) {
	first, err := Parse(validDelivery())
	if err != nil {
		t.Fatal(err)
	}
	edited := validDelivery()
	edited.EventVersion = "edit-2"
	edited.Body = "@vrooli review"
	second, err := Parse(edited)
	if err != nil {
		t.Fatal(err)
	}
	if first.Key == second.Key {
		t.Fatalf("edited event reused original key %q", first.Key)
	}
	if second.Key != "github:evt-1:edit-2" {
		t.Fatalf("unexpected edited event key %q", second.Key)
	}
}

func TestPublicTextRemovesPrivateReferences(t *testing.T) {
	got := PublicText("See plan-private and observed change", []string{"plan-private"})
	if got != "See [private reference omitted] and observed change" {
		t.Fatalf("unexpected redaction: %q", got)
	}
}
