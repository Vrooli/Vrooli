package scenariocli

import "testing"

func TestParseTestArgsAcceptsCatalogPhaseSelectors(t *testing.T) {
	for _, selector := range []string{"branding", "ui-health", "storage", "playbooks", "proto"} {
		got, err := ParseTestArgs(false, false, []string{"brand-manager", selector})
		if err != nil {
			t.Fatalf("ParseTestArgs(%q) returned error: %v", selector, err)
		}
		if got.Selector != selector || len(got.Opts.Args) == 0 || got.Opts.Args[0] != selector {
			t.Fatalf("selector %q parsed as selector=%q args=%v", selector, got.Selector, got.Opts.Args)
		}
	}
}

func TestParseTestArgsMapsLegacyIntegrationAliasesToPlaybooks(t *testing.T) {
	for _, selector := range []string{"integration", "e2e"} {
		got, err := ParseTestArgs(false, false, []string{"brand-manager", selector})
		if err != nil {
			t.Fatalf("ParseTestArgs(%q) returned error: %v", selector, err)
		}
		if got.Selector != "playbooks" || len(got.Opts.Args) == 0 || got.Opts.Args[0] != "playbooks" {
			t.Fatalf("selector %q parsed as selector=%q args=%v, want playbooks", selector, got.Selector, got.Opts.Args)
		}
	}
}
