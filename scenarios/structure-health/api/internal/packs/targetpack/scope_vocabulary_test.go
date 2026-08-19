package targetpack

import (
	"testing"

	"github.com/vrooli/api-core/scopecatalog"
)

func TestScopeVocabularyRuleRejectsUnknownEnforcedScope(t *testing.T) {
	catalog := scopecatalog.Catalog{Scopes: []scopecatalog.Scope{{Value: "known:read"}}}
	got := unknownConcreteScopes(catalog, `Resolve(scopes, "known:read"); Resolve(scopes, "invented:write")`)
	if len(got) != 1 || got[0] != "invented:write" {
		t.Fatalf("unknown scopes = %v, want [invented:write]", got)
	}
}
