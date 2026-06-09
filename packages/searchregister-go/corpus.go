package searchregister

import (
	"strings"

	aisearch "github.com/vrooli/aisearch-go"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

// corpus.go is the lossless, bidirectional bridge between a scenario's
// search.json evaluation corpus (aisearch.TestSuite — the file SSOT shape) and
// search-hub's eval store/wire shape (evalv1.EvalSuite). It is the corpus sibling
// of descriptor.go's tuning mapping, and it lives here for the same reason: only
// this bridge package is allowed to know BOTH the search.json shape and the
// search-hub proto, so neither side imports the other.
//
// The two shapes are a structural 1:1 after the corpus unification (one rank-
// centric case list; negatives are cases; no full-path labels / gate-policy
// fields). The ONLY fields that do not round-trip are the ones search-hub's store
// assigns — provider_id (the enclosing provider already owns it), state, and the
// created_at / updated_at timestamps. Everything a scenario authors in the file
// survives a file → store → file round-trip byte-for-byte; that property is the
// enforcement mechanism for the corpusStoreMirrorsFile invariant.

// SuiteToProto maps a parsed search.json TestSuite to the EvalSuite that
// EvalService.RegisterSuite accepts. providerID comes from the enclosing
// ProviderConfig (the file nests tests under a provider). The suite id defaults
// to "<providerID>.primary" unless the file sets an explicit override; state is
// always "active"; created_at/updated_at are left for the store to assign.
func SuiteToProto(providerID string, ts aisearch.TestSuite) *evalv1.EvalSuite {
	cases := make([]*evalv1.EvalCase, 0, len(ts.Cases))
	for _, c := range ts.Cases {
		cases = append(cases, &evalv1.EvalCase{
			CaseId:            c.ID,
			Query:             c.Query,
			Tags:              cloneStrings(c.Tags),
			ExpectIds:         cloneStrings(c.ExpectIDs),
			ExpectWithinTopK:  int32(c.ExpectWithinTopK),
			ExpectMinScore:    c.ExpectMinScore,
			ExpectMaxScore:    c.ExpectMaxScore,
			ExpectNoStrongHit: c.ExpectNoStrongHit,
			Note:              c.Note,
		})
	}
	return &evalv1.EvalSuite{
		SuiteId:     ts.ResolvedSuiteID(providerID),
		ProviderId:  providerID,
		Name:        ts.Name,
		Description: ts.Description,
		Cases:       cases,
		State:       "active",
	}
}

// INVARIANT: corpusRoundTripsLossless
//
//	SuiteFromProto(SuiteToProto(pid, ts)) == ts for every file-authored corpus ts.
//	SuiteFromProto is the inverse of SuiteToProto: it maps an EvalSuite (e.g. one
//	carried by a control.WriteCorpus round-trip, or read back from the store) into
//	the aisearch.TestSuite a scenario persists to search.json. The store-assigned
//	fields (provider_id, state, timestamps) are dropped — they are not part of the
//	file SSOT. A suite id equal to the default "<provider_id>.primary" collapses
//	back to the empty override, so the implicit and explicit default are the same
//	file (the writer never emits a redundant suite_id).
func SuiteFromProto(s *evalv1.EvalSuite) aisearch.TestSuite {
	if s == nil {
		return aisearch.TestSuite{}
	}
	var cases []aisearch.TestCase
	if len(s.GetCases()) > 0 {
		cases = make([]aisearch.TestCase, 0, len(s.GetCases()))
	}
	for _, c := range s.GetCases() {
		cases = append(cases, aisearch.TestCase{
			ID:                c.GetCaseId(),
			Query:             c.GetQuery(),
			Tags:              cloneStrings(c.GetTags()),
			ExpectIDs:         cloneStrings(c.GetExpectIds()),
			ExpectWithinTopK:  int(c.GetExpectWithinTopK()),
			ExpectMinScore:    c.GetExpectMinScore(),
			ExpectMaxScore:    c.GetExpectMaxScore(),
			ExpectNoStrongHit: c.GetExpectNoStrongHit(),
			Note:              c.GetNote(),
		})
	}
	ts := aisearch.TestSuite{
		Name:        s.GetName(),
		Description: s.GetDescription(),
		Cases:       cases,
	}
	// Collapse the default suite id to the empty override so the round-trip is
	// lossless (an implicit "<pid>.primary" never materializes as an explicit one).
	if id := strings.TrimSpace(s.GetSuiteId()); id != "" && id != s.GetProviderId()+".primary" {
		ts.SuiteID = id
	}
	return ts
}

// cloneStrings copies a string slice while preserving the nil/non-nil
// distinction, so a round-trip through the converter is reflect.DeepEqual-stable.
func cloneStrings(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
