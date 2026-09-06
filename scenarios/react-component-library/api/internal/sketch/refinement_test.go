package sketch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func refinementFixture(t *testing.T) (*Store, Candidate, Document) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"sketch":{},"claims":[{"id":"keep"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	store := NewStore(root)
	base, err := store.Read("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	doc := Document{Regions: []Region{{ID: "a"}, {ID: "b"}}}
	c, err := store.SaveCandidate("demo", "home", "design", base.ContentHash, doc)
	if err != nil {
		t.Fatal(err)
	}
	return store, c, doc
}
func TestRefinementBudgetRetryAndExplicitContinuation(t *testing.T) {
	store, parent, doc := refinementFixture(t)
	request := RefinementRequest{Document: doc, RequestedRegions: []string{"a"}, Reason: "Improve task clarity"}
	unchanged, err := store.RefineCandidate("demo", "design", parent.Hash, request)
	if err != nil || unchanged.Status != "unchanged" || unchanged.Round != 0 {
		t.Fatal("no-op consumed budget", unchanged, err)
	}
	for round := 1; round <= 3; round++ {
		request.Document.Regions[0].Note = string(rune('a' + round))
		got, err := store.RefineCandidate("demo", "design", parent.Hash, request)
		if err != nil || got.Status != "refined" || got.Round != round || got.Budget != 3 {
			t.Fatal(got, err)
		}
		retry, err := store.RefineCandidate("demo", "design", parent.Hash, request)
		if err != nil || retry.Candidate.Hash != got.Candidate.Hash {
			t.Fatal("retry duplicated refinement", err)
		}
		if got.Candidate.ParentHash != parent.Hash {
			t.Fatal("parent history lost")
		}
		parent = got.Candidate
	}
	request.Document.Regions[0].Note = "next"
	exhausted, err := store.RefineCandidate("demo", "design", parent.Hash, request)
	if err != nil || exhausted.Status != "budget_exhausted" || exhausted.Candidate.Hash != parent.Hash {
		t.Fatal("budget did not preserve parent", exhausted, err)
	}
	request.AdditionalRounds = 2
	continued, err := store.RefineCandidate("demo", "design", parent.Hash, request)
	if err != nil || continued.Round != 4 || continued.Budget != 5 {
		t.Fatal("continuation lost history", continued, err)
	}
	mapped, err := store.DeriveCandidate("demo", "design", continued.Candidate.Hash, request.Document)
	if err != nil || mapped.Refinement == nil || mapped.Refinement.Round != 4 || mapped.Refinement.Budget != 5 {
		t.Fatal("ordinary branch reset budget", err)
	}
}
func TestRefinementScopeExplanationCannotWaiveLocks(t *testing.T) {
	store, parent, doc := refinementFixture(t)
	request := RefinementRequest{Document: doc, RequestedRegions: []string{"a"}, Reason: "Improve task clarity"}
	request.Document.Regions[1].Note = "Broader change"
	if _, err := store.RefineCandidate("demo", "design", parent.Hash, request); err == nil {
		t.Fatal("unrequested region changed without explanation")
	}
	request.BroaderChangeReason = "Shared task terminology requires updating b"
	got, err := store.RefineCandidate("demo", "design", parent.Hash, request)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Candidate.Refinement.ChangedRegions) != 1 || got.Candidate.Refinement.ChangedRegions[0] != "b" {
		t.Fatal("actual scope not recorded")
	}
	snapshot, _ := parent.Snapshot()
	snapshot.Document.Regions[0].Locked = true
	locked, err := store.DeriveCandidate("demo", "design", parent.Hash, snapshot.Document)
	if err != nil {
		t.Fatal(err)
	}
	request.Document = snapshot.Document
	request.Document.Regions[0].Note = "Change locked region"
	if _, err := store.RefineCandidate("demo", "design", locked.Hash, request); err == nil {
		t.Fatal("broader explanation waived lock")
	}
}
func TestRefinementEnvelopeSurvivesReadWithoutChangingLegacyIdentity(t *testing.T) {
	store, parent, doc := refinementFixture(t)
	raw, _ := json.Marshal(parent)
	if string(raw) == "" {
		t.Fatal("empty candidate")
	}
	doc.Regions[0].Note = "Refined"
	result, err := store.RefineCandidate("demo", "design", parent.Hash, RefinementRequest{Document: doc, RequestedRegions: []string{"a"}, Reason: "Reviewed hierarchy finding"})
	if err != nil {
		t.Fatal(err)
	}
	read, err := store.ReadCandidate("demo", "design", result.Candidate.Hash)
	if err != nil || read.Refinement.Reason != "Reviewed hierarchy finding" {
		t.Fatal("refinement metadata lost", err)
	}
	original, err := store.ReadCandidate("demo", "design", parent.Hash)
	if err != nil || original.Hash != parent.Hash || original.Refinement != nil {
		t.Fatal("legacy candidate changed", err)
	}
}

func TestRefinementRejectsInvalidScopeAndPrematureContinuation(t *testing.T) {
	for _, name := range []string{"unknown", "duplicate", "missing reason", "premature continuation", "unbounded continuation"} {
		t.Run(name, func(t *testing.T) {
			store, parent, doc := refinementFixture(t)
			r := RefinementRequest{Document: doc, RequestedRegions: []string{"a"}, Reason: "Clarify recovery"}
			switch name {
			case "unknown":
				r.RequestedRegions = []string{"missing"}
			case "duplicate":
				r.RequestedRegions = []string{"a", "a"}
			case "missing reason":
				r.Reason = ""
			case "premature continuation":
				r.AdditionalRounds = 1
			case "unbounded continuation":
				r.AdditionalRounds = 11
			}
			if _, err := store.RefineCandidate("demo", "design", parent.Hash, r); err == nil {
				t.Fatal("invalid request accepted")
			}
		})
	}
}
