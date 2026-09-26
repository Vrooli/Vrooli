package measureindex

import (
	"context"
	"testing"

	measures "github.com/vrooli/measures-go"
)

func backlogCompleted() measures.MeasureDeclaration {
	return measures.MeasureDeclaration{
		Name:      "backlog.completed",
		Scenario:  "swarm-manager",
		Domain:    "backlog",
		Intent:    "How many backlog items were completed in a time window.",
		Questions: []string{"how many backlog items did we complete this week", "backlog items closed last month"},
		Effect:    measures.EffectRead,
	}
}

func recordsCreated() measures.MeasureDeclaration {
	return measures.MeasureDeclaration{
		Name:      "records.created",
		Scenario:  "swarm-manager",
		Domain:    "records",
		Intent:    "How many work records were created in a time window.",
		Questions: []string{"how many records did we write this week", "records created last month"},
		Effect:    measures.EffectRead,
	}
}

func TestLexicalMatcher_MatchesSalientMeasure(t *testing.T) {
	m := NewLexicalMatcher([]measures.MeasureDeclaration{backlogCompleted(), recordsCreated()})

	got, err := m.Match(context.Background(), "how many backlog items did we complete this week", 3)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected at least one match")
	}
	if got[0].Decl.Name != "backlog.completed" {
		t.Fatalf("expected backlog.completed first, got %q (score %.3f)", got[0].Decl.Name, got[0].Score)
	}
	if got[0].Score <= 0 || got[0].Score > 1 {
		t.Fatalf("score must be in (0,1], got %.3f", got[0].Score)
	}
}

func TestLexicalMatcher_RanksRecordsOverBacklog(t *testing.T) {
	m := NewLexicalMatcher([]measures.MeasureDeclaration{backlogCompleted(), recordsCreated()})

	got, err := m.Match(context.Background(), "how many records did we write last month", 2)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(got) == 0 || got[0].Decl.Name != "records.created" {
		t.Fatalf("expected records.created first, got %+v", got)
	}
}

func TestLexicalMatcher_NoSalientOverlapReturnsEmpty(t *testing.T) {
	m := NewLexicalMatcher([]measures.MeasureDeclaration{backlogCompleted()})

	got, err := m.Match(context.Background(), "qwerty zxcvb florbnax", 3)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no match for gibberish, got %+v", got)
	}
}

func TestLexicalMatcher_EmptyCorpus(t *testing.T) {
	m := NewLexicalMatcher(nil)
	if m.Len() != 0 {
		t.Fatalf("expected empty corpus, Len=%d", m.Len())
	}
	got, err := m.Match(context.Background(), "how many backlog items this week", 3)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("empty corpus must return no matches, got %+v", got)
	}
}

func TestLexicalMatcher_LimitClampsResults(t *testing.T) {
	m := NewLexicalMatcher([]measures.MeasureDeclaration{backlogCompleted(), recordsCreated()})
	got, err := m.Match(context.Background(), "how many items did we complete or write this week", 1)
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("limit=1 must return at most one match, got %d", len(got))
	}
}
