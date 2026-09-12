package registration

import (
	"testing"

	aisearch "github.com/vrooli/ai-go/search"
)

func TestTokenStoreTracksTokensPerLeafAndMatchesAnyOwnedLeaf(t *testing.T) {
	store := NewTokenStore()
	store.Set("code-facts.code", "code-token")
	store.Set("code-facts.contracts", "contract-token")
	if got := store.Get("code-facts.code"); got != "code-token" {
		t.Fatalf("code token = %q", got)
	}
	if !store.Matches("code-token") || !store.Matches("contract-token") || store.Matches("") || store.Matches("wrong") {
		t.Fatal("token matching did not preserve the per-leaf ownership set")
	}
}

func TestEvalSuitePreservesScopedDirectCases(t *testing.T) {
	provider := aisearch.ProviderConfig{ProviderID: "code-facts.code", Tests: aisearch.TestSuite{
		Name: "direct", Cases: []aisearch.TestCase{{ID: "scoped", Query: "where", Scope: "scenario:code-facts", Status: "reviewed", ExpectIDs: []string{"hit"}, ExpectWithinTopK: 5}},
	}}
	suite := evalSuite(provider)
	if suite.GetSuiteId() != "code-facts.code.primary" || suite.GetProviderId() != provider.ProviderID {
		t.Fatalf("suite identity = %+v", suite)
	}
	if len(suite.GetCases()) != 1 || suite.GetCases()[0].GetScope() != "scenario:code-facts" || suite.GetCases()[0].GetExpectIds()[0] != "hit" {
		t.Fatalf("suite case = %+v", suite.GetCases())
	}
}
