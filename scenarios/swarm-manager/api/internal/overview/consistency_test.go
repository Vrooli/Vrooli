package overview

import (
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
)

func mkItem(kind backlog.BacklogKind, name string, deps ...string) backlog.BacklogItem {
	return backlog.BacklogItem{Kind: kind, Name: name, Status: backlog.StatusBacklog, DependsOn: deps}
}

func mkInit(name string, items []string, dependsOn []string) initiatives.InitiativeWithRollup {
	return initiatives.InitiativeWithRollup{
		Initiative: initiatives.Initiative{
			Name:      name,
			Title:     name,
			Status:    "active",
			Items:     items,
			DependsOn: dependsOn,
		},
	}
}

func TestComputeInitiativeEdgeSuggestions_MissingExplicit(t *testing.T) {
	items := []backlog.BacklogItem{
		mkItem("execute", "a-item", "execute/b-item"),
		mkItem("execute", "b-item"),
	}
	inits := []initiatives.InitiativeWithRollup{
		mkInit("alpha", []string{"execute/a-item"}, nil),
		mkInit("beta", []string{"execute/b-item"}, nil),
	}
	sug := computeInitiativeEdgeSuggestions(items, inits)
	if len(sug) != 1 {
		t.Fatalf("expected 1 suggestion, got %d: %+v", len(sug), sug)
	}
	if sug[0].From != "alpha" || sug[0].To != "beta" || sug[0].Direction != "missing_explicit" {
		t.Errorf("unexpected suggestion: %+v", sug[0])
	}
	if len(sug[0].InferredFromItems) != 1 || sug[0].InferredFromItems[0] != "execute/a-item" {
		t.Errorf("evidence mismatch: %v", sug[0].InferredFromItems)
	}
}

func TestComputeInitiativeEdgeSuggestions_NoDriftWhenExplicit(t *testing.T) {
	items := []backlog.BacklogItem{
		mkItem("execute", "a-item", "execute/b-item"),
		mkItem("execute", "b-item"),
	}
	inits := []initiatives.InitiativeWithRollup{
		mkInit("alpha", []string{"execute/a-item"}, []string{"beta"}),
		mkInit("beta", []string{"execute/b-item"}, nil),
	}
	if sug := computeInitiativeEdgeSuggestions(items, inits); len(sug) != 0 {
		t.Errorf("expected no suggestions when edge is explicit, got %+v", sug)
	}
}

func TestComputeInitiativeEdgeSuggestions_PossiblyStale(t *testing.T) {
	// alpha depends on beta explicitly, but no child of alpha depends on beta's items.
	items := []backlog.BacklogItem{
		mkItem("execute", "a-item"),
		mkItem("execute", "b-item"),
	}
	inits := []initiatives.InitiativeWithRollup{
		mkInit("alpha", []string{"execute/a-item"}, []string{"beta"}),
		mkInit("beta", []string{"execute/b-item"}, nil),
	}
	sug := computeInitiativeEdgeSuggestions(items, inits)
	if len(sug) != 1 {
		t.Fatalf("expected 1 suggestion, got %d: %+v", len(sug), sug)
	}
	if sug[0].Direction != "possibly_stale" {
		t.Errorf("expected possibly_stale, got %q", sug[0].Direction)
	}
}

func TestComputeInitiativeEdgeSuggestions_SameInitiativeIgnored(t *testing.T) {
	// Intra-initiative dependency must NOT produce a suggestion.
	items := []backlog.BacklogItem{
		mkItem("execute", "a", "execute/b"),
		mkItem("execute", "b"),
	}
	inits := []initiatives.InitiativeWithRollup{
		mkInit("solo", []string{"execute/a", "execute/b"}, nil),
	}
	if sug := computeInitiativeEdgeSuggestions(items, inits); len(sug) != 0 {
		t.Errorf("intra-initiative deps should not produce suggestions: %+v", sug)
	}
}

func TestComputeInitiativeEdgeSuggestions_UnownedItemIgnored(t *testing.T) {
	items := []backlog.BacklogItem{
		mkItem("execute", "a", "execute/orphan"),
	}
	inits := []initiatives.InitiativeWithRollup{
		mkInit("alpha", []string{"execute/a"}, nil),
	}
	if sug := computeInitiativeEdgeSuggestions(items, inits); len(sug) != 0 {
		t.Errorf("orphan deps should be ignored: %+v", sug)
	}
}

func TestComputeInitiativeEdgeSuggestions_StableSortedOutput(t *testing.T) {
	items := []backlog.BacklogItem{
		mkItem("execute", "a1", "execute/c1"),
		mkItem("execute", "b1", "execute/c1"),
		mkItem("execute", "c1"),
	}
	inits := []initiatives.InitiativeWithRollup{
		mkInit("alpha", []string{"execute/a1"}, nil),
		mkInit("beta", []string{"execute/b1"}, nil),
		mkInit("gamma", []string{"execute/c1"}, nil),
	}
	sug := computeInitiativeEdgeSuggestions(items, inits)
	if len(sug) != 2 {
		t.Fatalf("expected 2 suggestions, got %d", len(sug))
	}
	// Expect alpha->gamma before beta->gamma.
	if sug[0].From != "alpha" || sug[1].From != "beta" {
		t.Errorf("expected alphabetical order, got %+v", sug)
	}
}
