package aisearch

import (
	"reflect"
	"testing"

	searchregister "github.com/vrooli/searchregister-go"
)

// corpus_roundtrip_test.go grounds the corpusRoundTripsLossless invariant on the
// REAL committed corpus: the cli-health.commands provider's tests block in
// scenarios/cli-health/.vrooli/search.json must survive the file → store → file
// converter (the same path boot self-registration and the WriteCorpus write-back
// use) byte-for-byte. If it ever drifts, a boot re-registration or a corpus
// write-back would silently mutate the golden corpus.
func TestCommandCorpusRoundTripsLossless(t *testing.T) {
	provider := searchProvider(t)
	suite := provider.Tests

	got := searchregister.SuiteFromProto(searchregister.SuiteToProto(provider.ProviderID, suite))
	if !reflect.DeepEqual(suite, got) {
		t.Fatalf("real cli-health corpus does not round-trip losslessly:\n in = %#v\nout = %#v", suite, got)
	}

	// The store-shape carries the default suite id + the enclosing provider id.
	proto := searchregister.SuiteToProto(provider.ProviderID, suite)
	if proto.GetSuiteId() != "cli-health.commands.primary" {
		t.Errorf("converted suite id = %q, want cli-health.commands.primary", proto.GetSuiteId())
	}
	if len(proto.GetCases()) != len(suite.Cases) || len(suite.Cases) == 0 {
		t.Errorf("converted case count = %d, want %d (non-zero)", len(proto.GetCases()), len(suite.Cases))
	}
}
