// [REQ:GCT-OT-P1-002]
package proposals

import (
	"context"
	"testing"
	"time"
)

func TestRetainedCandidatesIncludeClosedSourcesNewestFirstWithoutReopening(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	base := time.Unix(1700000000, 0)
	for i, p := range []Proposal{{ID: "older-open", RepositoryID: "repo", State: StateOpen}, {ID: "closed", RepositoryID: "repo", State: StateOpen}, {ID: "newest", RepositoryID: "repo", State: StateOpen}, {ID: "other", RepositoryID: "other", State: StateOpen}} {
		p.Revision = 1
		p.CreatedAt = base.Add(time.Duration(i) * time.Minute)
		p.UpdatedAt = p.CreatedAt
		if e := s.store.Insert(ctx, p); e != nil {
			t.Fatal(e)
		}
		if p.ID == "closed" || p.ID == "newest" {
			closedState := StateWithdrawn
			if p.ID == "newest" {
				closedState = StateCommitted
			}
			p.State = closedState
			if e := s.store.Update(ctx, p, p.Revision, StateOpen); e != nil {
				t.Fatal(e)
			}
		}
	}
	got, e := s.RetainedCandidates(ctx, "repo", 2)
	if e != nil || len(got) != 2 || got[0].ID != "newest" || got[1].ID != "closed" || got[1].State != StateWithdrawn {
		t.Fatalf("retained candidates %#v %v", got, e)
	}
	stored, e := s.store.Get(ctx, "closed")
	if e != nil || stored.State != StateWithdrawn {
		t.Fatal("reading retained context reopened proposal", stored, e)
	}
}
