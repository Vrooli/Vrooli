package readiness

import (
	"fmt"
	"strings"
	"time"
)

// MarketingObservation is the producer-neutral projection of Content Desk's
// launch-assets report. Deployment-manager consumes this as an advisory
// signal; it does not decide which channels marketing should own.
type MarketingObservation struct {
	Scenario      string
	SlotCount     int
	OpenSlotCount int
	DraftCount    int
	ObservedAt    time.Time
}

func (o MarketingObservation) Signal() (Signal, error) {
	if strings.TrimSpace(o.Scenario) == "" || o.ObservedAt.IsZero() {
		return Signal{}, fmt.Errorf("marketing observation requires scenario and observed_at")
	}
	status := SignalPassed
	detail := fmt.Sprintf("%d launch slot(s), %d open slot(s), %d approved/published draft(s)", o.SlotCount, o.OpenSlotCount, o.DraftCount)
	if o.SlotCount == 0 || o.DraftCount == 0 {
		status = SignalFailed
		detail = fmt.Sprintf("no launch assets reported for scenario %s (%s)", o.Scenario, detail)
	}
	return Signal{ItemID: "marketing-assets-available", Status: status, Source: "content-desk", ObservedAt: o.ObservedAt.UTC(), Detail: detail}, nil
}
