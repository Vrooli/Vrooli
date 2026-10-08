package lifecycle

import (
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
)

// A variant's lifecycle output must be filed under its own instance slug. When
// it was filed under the bare scenario slug, two things broke at once: the
// variant's log appeared to be missing (the reader and the failure block both
// resolve "scenario@variant"), and live's log silently gained entries for work
// the live instance never did.
func TestInstanceLogNameSeparatesVariantFromLive(t *testing.T) {
	live := scenario.Scenario{Slug: "web-console"}
	if got := instanceLogName(live); got != "web-console" {
		t.Errorf("live instance log name = %q, want %q", got, "web-console")
	}

	// An explicitly normalized live instance is still live.
	normalizedLive := scenario.Scenario{Slug: "web-console", Variant: "live"}
	if got := instanceLogName(normalizedLive); got != "web-console" {
		t.Errorf("normalized live instance log name = %q, want %q", got, "web-console")
	}

	variant := scenario.Scenario{Slug: "web-console", Variant: "presentation"}
	got := instanceLogName(variant)
	if got == instanceLogName(live) {
		t.Fatalf("variant and live share the log name %q; the variant would append to live's log", got)
	}
	if got != "web-console@presentation" {
		t.Errorf("variant instance log name = %q, want %q", got, "web-console@presentation")
	}
}
