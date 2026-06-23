package completeness

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestLiveScore_AgainstRealService exercises the full delegated-measurement path
// against a RUNNING scenario-completeness-scoring instance: api-core discovery
// resolution → Connect GetScore → scoreFromProto projection. It proves EM reads
// the rung, build verdict, and operational-targets completion from the authority
// on real data (the measurement the controller now gates on instead of the
// deleted MetricsSnapshot collectors).
//
// Gated on EM_LIVE_COMPLETENESS so the default `go test ./...` (and the
// test-genie unit phase, which has no guaranteed live service) skips it.
//
//	EM_LIVE_COMPLETENESS=1 go test ./pkg/completeness/ -run TestLiveScore -v
func TestLiveScore_AgainstRealService(t *testing.T) {
	if os.Getenv("EM_LIVE_COMPLETENESS") == "" {
		t.Skip("set EM_LIVE_COMPLETENESS=1 to run against a live scenario-completeness-scoring")
	}

	c := NewClient(0)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for _, scenario := range []string{"ecosystem-manager", "scenario-completeness-scoring"} {
		got, err := c.Score(ctx, scenario)
		if err != nil {
			t.Fatalf("live Score(%q) failed: %v", scenario, err)
		}
		t.Logf("LIVE %s → rung=%q satisfied_through=%q ladder_clean=%v build=%v composite=%d OT=%d/%d (%.0f%%) ot_known=%v ot_has=%v calc=%s",
			scenario, got.WorkingRung, got.SatisfiedThrough, got.LadderClean, got.BuildPassing,
			got.Composite, got.OTPassing, got.OTTotal, got.OTPercentage, got.OTKnown, got.OTHasTargets,
			got.CalculatedAt.Format(time.RFC3339))

		// The payload must be usable for gating: the rung is either resolved
		// (working or clean) and the composite is a real 0–100 reading.
		if got.WorkingRung == "" && !got.LadderClean && got.SatisfiedThrough == "" {
			t.Errorf("%s: no rung signal at all (working/satisfied/clean all empty) — payload not usable for gating", scenario)
		}
		if got.Composite < 0 || got.Composite > 100 {
			t.Errorf("%s: composite %d out of range", scenario, got.Composite)
		}
		// OT counts must be internally consistent.
		if got.OTPassing > got.OTTotal {
			t.Errorf("%s: OT passing %d > total %d", scenario, got.OTPassing, got.OTTotal)
		}
		if (got.OTTotal > 0) != got.OTHasTargets {
			t.Errorf("%s: OTHasTargets=%v disagrees with OTTotal=%d", scenario, got.OTHasTargets, got.OTTotal)
		}
		// D1 freshness: a live read carries a calculation timestamp.
		if got.CalculatedAt.IsZero() {
			t.Errorf("%s: CalculatedAt is zero — payload not stamped", scenario)
		}
	}
}

// TestLiveScore_DegradesLoudly proves the D2 contract against a real client: when
// the authority is unreachable, Score returns an error (no zero-value fallback),
// so the controller can halt with measurement_unavailable rather than silently
// continuing on stale data.
func TestLiveScore_DegradesLoudly(t *testing.T) {
	if os.Getenv("EM_LIVE_COMPLETENESS") == "" {
		t.Skip("set EM_LIVE_COMPLETENESS=1 to run the live degrade check")
	}
	// A resolver that points at a closed port — the real transport must surface
	// the failure rather than swallow it.
	c := &Client{
		httpClient: &http.Client{Timeout: 2 * time.Second},
		resolve:    func(context.Context) (string, error) { return "http://127.0.0.1:1", nil },
	}
	if _, err := c.Score(context.Background(), "ecosystem-manager"); err == nil {
		t.Fatal("expected a loud error from an unreachable authority, got nil (would be a silent fallback)")
	}
}
