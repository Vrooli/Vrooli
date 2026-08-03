package preview

import (
	"strings"
	"testing"
)

func TestValidateAdoptedContextReportsUndefinedToken(t *testing.T) {
	missing := ValidateAdoptedContext("bg-app-primary text-app-foreground", ConsumerTokenSet{Name: "wc", Tokens: map[string]struct{}{"wc-accent": {}}})
	if len(missing) != 2 || missing[0].Token != "app-foreground" || missing[1].Token != "app-primary" {
		t.Fatalf("missing = %#v", missing)
	}
	if err := AdoptedContextError(ConsumerTokenSet{Name: "wc"}, missing); err == nil || !strings.Contains(err.Error(), "undefined") {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateAdoptedContextPassesTranslatedTokens(t *testing.T) {
	missing := ValidateAdoptedContext("bg-wc-accent text-wc-text-primary", ConsumerTokenSet{Tokens: map[string]struct{}{"wc-accent": {}, "wc-text-primary": {}}})
	if len(missing) != 0 {
		t.Fatalf("missing = %#v", missing)
	}
}
