package readiness

import (
	"strings"
	"testing"
	"time"
)

func TestMarketingObservationIsAdvisoryAndNamesMissingAssets(t *testing.T) {
	signal, err := (MarketingObservation{Scenario: "web-console", ObservedAt: time.Now().UTC()}).Signal()
	if err != nil {
		t.Fatal(err)
	}
	if signal.ItemID != "marketing-assets-available" || signal.Status != SignalFailed || !strings.Contains(signal.Detail, "no launch assets") {
		t.Fatalf("unexpected marketing signal: %+v", signal)
	}
	verdict, err := Aggregate("web-console", "abc", Checklist{Version: ChecklistVersion, Items: []Item{{ID: "marketing-assets-available", Title: "Launch assets", Category: "mechanical", CleanRequirement: Advisory, GlobalImpact: AdvisoryImpact, AcceptanceCriteria: "report"}}}, []Signal{signal}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if !verdict.Approved {
		t.Fatal("advisory marketing signal blocked readiness")
	}
}
