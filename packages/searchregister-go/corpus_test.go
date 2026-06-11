package searchregister

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	aisearch "github.com/vrooli/ai-go/search"
)

// corpus_test.go enforces corpusRoundTripsLossless: a file-authored corpus must
// survive SuiteToProto → SuiteFromProto unchanged. That round-trip is the
// mechanism behind the corpusStoreMirrorsFile invariant — if it ever drifts, a
// boot self-registration or a WriteCorpus write-back would silently mutate the
// scenario's golden corpus.

// TestSuiteRoundTripLossless is the property test: over many pseudo-random corpora
// (fixed seed = deterministic), the converter is a perfect inverse.
func TestSuiteRoundTripLossless(t *testing.T) {
	rng := rand.New(rand.NewSource(0xC0FFEE))
	const providerID = "demo.commands"
	for i := 0; i < 200; i++ {
		ts := randomSuite(rng, providerID, i)
		got := SuiteFromProto(SuiteToProto(providerID, ts))
		if !reflect.DeepEqual(ts, got) {
			t.Fatalf("round-trip #%d not lossless:\n in = %#v\nout = %#v", i, ts, got)
		}
	}
}

// TestSuiteRoundTripExplicitDefaultCollapses documents the one intentional
// normalization: an explicit suite_id equal to the default "<pid>.primary" is the
// same file as the implicit default, so it collapses to the empty override.
func TestSuiteRoundTripExplicitDefaultCollapses(t *testing.T) {
	const providerID = "demo.commands"
	in := aisearch.TestSuite{
		SuiteID: providerID + ".primary", // explicit, but redundant with the default
		Cases:   []aisearch.TestCase{{ID: "c1", Query: "q"}},
	}
	got := SuiteFromProto(SuiteToProto(providerID, in))
	if got.SuiteID != "" {
		t.Fatalf("explicit default suite_id must collapse to empty, got %q", got.SuiteID)
	}
	// A genuinely custom suite id is preserved verbatim.
	in.SuiteID = providerID + ".secondary"
	got = SuiteFromProto(SuiteToProto(providerID, in))
	if got.SuiteID != providerID+".secondary" {
		t.Fatalf("custom suite_id must survive, got %q", got.SuiteID)
	}
}

// TestSuiteToProtoStoreAssignedFields confirms the store-assigned fields are set
// (provider_id, state, suite_id default) on the way out and dropped on the way
// back — they are not part of the file SSOT.
func TestSuiteToProtoStoreAssignedFields(t *testing.T) {
	const providerID = "demo.commands"
	p := SuiteToProto(providerID, aisearch.TestSuite{Cases: []aisearch.TestCase{{ID: "c1", Query: "q"}}})
	if p.GetSuiteId() != providerID+".primary" {
		t.Errorf("default suite id = %q", p.GetSuiteId())
	}
	if p.GetProviderId() != providerID {
		t.Errorf("provider id = %q", p.GetProviderId())
	}
	if p.GetState() != "active" {
		t.Errorf("state = %q, want active", p.GetState())
	}
}

func TestSuiteFromProtoNil(t *testing.T) {
	if got := SuiteFromProto(nil); !reflect.DeepEqual(got, aisearch.TestSuite{}) {
		t.Fatalf("nil suite must map to the zero TestSuite, got %#v", got)
	}
}

// randomSuite builds a varied but well-formed corpus. SuiteID is either empty or a
// clearly-custom id (never the bare default — that collapse is covered separately).
func randomSuite(rng *rand.Rand, providerID string, n int) aisearch.TestSuite {
	ts := aisearch.TestSuite{
		Name:        pick(rng, "", "primary corpus", "golden"),
		Description: pick(rng, "", "rank-centric"),
	}
	if rng.Intn(3) == 0 {
		ts.SuiteID = fmt.Sprintf("%s.alt-%d", providerID, n)
	}
	cases := rng.Intn(6)
	for i := 0; i < cases; i++ {
		c := aisearch.TestCase{
			ID:     fmt.Sprintf("c%d", i),
			Query:  pick(rng, "restart a scenario", "view logs", "asdf qwer"),
			Status: pick(rng, "", aisearch.CaseStatusReviewed, aisearch.CaseStatusCandidate),
			Note:   pick(rng, "", "calibration note"),
		}
		if rng.Intn(4) == 0 { // a negative
			c.Tags = []string{"gibberish"}
			c.ExpectNoStrongHit = true
			c.ExpectMaxScore = 0.3
		} else { // a positive
			c.ExpectIDs = randStrings(rng, "restart", "logs", "start")
			c.ExpectWithinTopK = 1 + rng.Intn(5)
			if rng.Intn(5) == 0 {
				c.ExpectMinScore = 0.5
			}
			if rng.Intn(3) == 0 {
				c.Tags = randStrings(rng, "strong", "weak-real", "generated")
			}
		}
		ts.Cases = append(ts.Cases, c)
	}
	return ts
}

func pick(rng *rand.Rand, opts ...string) string { return opts[rng.Intn(len(opts))] }

// randStrings returns a non-nil 1..len(opts) subset (order-preserving), so the
// nil/non-nil distinction the converter preserves is exercised on both sides.
func randStrings(rng *rand.Rand, opts ...string) []string {
	k := 1 + rng.Intn(len(opts))
	out := make([]string, 0, k)
	for _, o := range opts[:k] {
		out = append(out, o)
	}
	return out
}
