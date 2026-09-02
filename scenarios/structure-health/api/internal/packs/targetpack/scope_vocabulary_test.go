package targetpack

import (
	"path/filepath"
	"runtime"
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

func TestScopeVocabularyRuleWalksProductionRepository(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../.."))
	findings := projectScopeVocabularyRules(root)
	if len(findings) != 0 {
		t.Fatalf("repository scope vocabulary findings = %v", findings)
	}
}
