package validation

import (
	"testing"
	"time"
)

func TestAdvisoryEpochStableAndChangesWithin24Hours(t *testing.T) {
	now := time.Date(2026, 8, 28, 17, 42, 0, 0, time.FixedZone("EDT", -4*60*60))
	for _, target := range []string{"security-health", "prompt-manager", "test-genie", "scenario-42"} {
		first := advisoryEpoch(target, now)
		if first == advisoryEpoch(target, now.Add(24*time.Hour)) {
			t.Fatalf("epoch for %q did not advance within 24 hours", target)
		}
		if first != advisoryEpoch(target, now.Add(5*time.Minute)) {
			t.Fatalf("epoch for %q changed within its hour", target)
		}
	}
}

func TestAdvisoryEpochUsesStableTargetBuckets(t *testing.T) {
	now := time.Date(2026, 8, 28, 17, 0, 0, 0, time.UTC)
	seen := map[string]string{}
	for i := 0; i < 200; i++ {
		target := "scenario-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		seen[target] = advisoryEpoch(target, now)
	}
	unique := map[string]struct{}{}
	for _, epoch := range seen {
		unique[epoch] = struct{}{}
	}
	if len(unique) < 2 {
		t.Fatal("all target identities received the same advisory bucket")
	}
	if advisoryEpoch("security-health", now) != advisoryEpoch("security-health", now) {
		t.Fatal("advisory epoch is not stable")
	}
}
