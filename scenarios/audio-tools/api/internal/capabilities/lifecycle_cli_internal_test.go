package capabilities

import (
	"testing"

	"audio-tools/internal/providercatalog"
)

// TestAllowedSlugsCoverEveryLocalProvider pins the defence-in-depth allowlist
// to the provider catalog. Without this, a newly active local provider appears
// in ListLocalProviders but every lifecycle action is rejected with
// "disallowed resource slug".
func TestAllowedSlugsCoverEveryLocalProvider(t *testing.T) {
	for _, id := range providercatalog.LocalProviderIDs() {
		slug, ok := providercatalog.ResourceSlugForProviderID(id)
		if !ok {
			t.Fatalf("local provider %q has no resource slug", id)
		}
		if _, allowed := allowedSlugs[slug]; !allowed {
			t.Errorf("resource slug %q (provider %q) is not allowed by CLIController", slug, id)
		}
	}
}

func TestAllowedSlugsRejectsUnknown(t *testing.T) {
	if _, ok := allowedSlugs["definitely-not-a-resource"]; ok {
		t.Fatal("unknown slug unexpectedly allowed")
	}
}
