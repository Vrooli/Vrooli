package assistance

import (
	"errors"
	"testing"
)

func TestProposalValidationRejectsModelAuthorityFields(t *testing.T) {
	review := Validate(Proposal{SchemaVersion: 1, Changes: []Change{{Field: "allergens", Value: "none"}}})
	if review.Valid || len(review.Errors) != 1 {
		t.Fatalf("review=%#v", review)
	}
}

func TestProposalApplyRequiresReviewAndCurrentDraftRevision(t *testing.T) {
	proposal := Proposal{SchemaVersion: 1, BaseDraftRevision: 4, Changes: []Change{{Field: "name", Value: "Soup"}}}
	if _, err := Apply(proposal, 3); !errors.Is(err, ErrStaleDraft) {
		t.Fatalf("stale proposal err=%v", err)
	}
	changes, err := Apply(proposal, 4)
	if err != nil || changes["name"] != "Soup" {
		t.Fatalf("changes=%v err=%v", changes, err)
	}
}

func TestMalformedProposalNeverProducesPartialChanges(t *testing.T) {
	proposal := Proposal{SchemaVersion: 1, BaseDraftRevision: 1, Changes: []Change{{Field: "name", Value: "Soup"}, {Field: "targets", Value: "80g"}}}
	if changes, err := Apply(proposal, 1); err == nil || changes != nil {
		t.Fatalf("malformed proposal applied changes=%v err=%v", changes, err)
	}
}
