package trailers

import (
	"context"
	"testing"
)

func TestParsePreservesSupportedUnknownAndMultilineTrailers(t *testing.T) {
	msg := Parse("Draft summary\n\nVrooli-Plan: plan-1\nVrooli-Unknown: keep\n continuation\nReviewed-by: human")
	if msg.Subject != "Draft summary" || len(msg.Entries) != 2 {
		t.Fatalf("message = %#v", msg)
	}
	if msg.Entries[0].Key != "Vrooli-Plan" || !msg.Entries[0].Supported {
		t.Fatalf("plan = %#v", msg.Entries[0])
	}
	if msg.Entries[1].Value != "keep\ncontinuation" || msg.Entries[1].Supported {
		t.Fatalf("unknown = %#v", msg.Entries[1])
	}
	if msg.Body == "" {
		t.Fatal("non-trailer body was lost")
	}
}

func TestParseDoesNotTreatMalformedSeparatorAsReference(t *testing.T) {
	msg := Parse("Subject\n\nVrooli-Plan without separator")
	if len(msg.Entries) != 0 {
		t.Fatalf("entries = %#v", msg.Entries)
	}
}

type resolverStub struct{ status ResolutionStatus }

func (r resolverStub) Resolve(context.Context, string, string) (string, ResolutionStatus, string, error) {
	return "plan-42", r.status, "owner result", nil
}

func TestResolveNeverPromotesMetadataToAuthorship(t *testing.T) {
	message := Parse("subject\n\nVrooli-Plan: plan-42\nVrooli-Unknown: legacy")
	refs, err := Resolve(context.Background(), message, resolverStub{status: Resolved})
	if err != nil || len(refs) != 2 {
		t.Fatalf("refs=%#v err=%v", refs, err)
	}
	if refs[0].Status != Resolved || refs[0].Confidence != "owner-resolved-reference" {
		t.Fatalf("resolved ref=%#v", refs[0])
	}
	if refs[1].Status != Legacy || refs[1].Confidence != "asserted-metadata" {
		t.Fatalf("legacy ref=%#v", refs[1])
	}
}
