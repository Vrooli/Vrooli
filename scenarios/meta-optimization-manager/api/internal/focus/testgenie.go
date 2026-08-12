package focus

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

const testGenieFocusReadDeadline = 3 * time.Second

type testGenieGapSource struct {
	resolver scenarioURLResolver
	http     connect.HTTPClient
	deadline time.Duration
}

// NewTestGenieGapSource constructs the empirical test-fleet signal source.
// It reads stored fleet evidence only; it never starts or waits for a suite.
func NewTestGenieGapSource() GapSource {
	return &testGenieGapSource{
		resolver: discovery.NewResolver(discovery.ResolverConfig{}),
		http:     &http.Client{Timeout: testGenieFocusReadDeadline},
		deadline: testGenieFocusReadDeadline,
	}
}

var _ GapSource = (*testGenieGapSource)(nil)

func (*testGenieGapSource) Axis() Axis { return AxisEmpirical }

func (s *testGenieGapSource) DerivedGaps(ctx context.Context) ([]Gap, error) {
	if s == nil || s.resolver == nil {
		return nil, fmt.Errorf("test-genie reader is not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, s.deadline)
	defer cancel()
	base, err := s.resolver.ResolveScenarioURLDefault(ctx, "test-genie")
	if err != nil {
		return nil, fmt.Errorf("resolve test-genie: %w", err)
	}
	client := runsconnect.NewRunsServiceClient(s.http, base)
	fleet, err := client.GetFleetHealth(ctx, connect.NewRequest(&runsv1.GetFleetHealthRequest{WindowDays: 30}))
	if err != nil {
		return nil, fmt.Errorf("read test-genie fleet health: %w", err)
	}
	health := fleet.Msg.GetFleetHealth()
	var out []Gap
	// Self-health is the per-leg condition source. It is deliberately best
	// effort here: fleet evidence remains useful when an older test-genie
	// binary has not yet exposed the ledger method.
	if self, selfErr := client.GetSelfHealth(ctx, connect.NewRequest(&runsv1.GetSelfHealthRequest{SkipConformance: true, WindowDays: 30})); selfErr == nil && self != nil && self.Msg != nil && self.Msg.GetSelfHealth() != nil {
		for _, phase := range self.Msg.GetSelfHealth().GetLedger().GetPhases() {
			if phase == nil || phase.GetFailureRate() <= 0 || phase.GetProvider() == "" {
				continue
			}
			notes := []string{fmt.Sprintf("phase=%s; failure_rate=%.1f%%; observations=%d", phase.GetPhase(), phase.GetFailureRate()*100, phase.GetTotalObservations())}
			if streak := phase.GetConsecutiveFailures(); streak > 0 {
				notes = append(notes, "consecutive_failures="+strconv.Itoa(int(streak)))
			}
			out = append(out, Gap{
				ID: "empirical/test-genie/condition/" + phase.GetPhase(), Axis: AxisEmpirical,
				Title: "test-genie validation leg is degraded", ProviderIDs: []string{phase.GetProvider()},
				EvidenceSource: "test-genie", EvidenceLocator: "test-genie://self-health/phase/" + phase.GetPhase(),
				Recurrence: int(phase.GetFailed()), Notes: notes,
			})
		}
	}
	if health == nil || health.GetTotalRuns() == 0 {
		return out, nil
	}
	failed := int(health.GetTotalRuns()) - countFleetPassed(health)
	if failed <= 0 {
		return nil, nil
	}
	failureRate := float64(failed) / float64(health.GetTotalRuns())
	notes := []string{
		fmt.Sprintf("window=%dd; runs=%d; failed=%d; failure_rate=%.1f%%", health.GetWindowDays(), health.GetTotalRuns(), failed, failureRate*100),
	}
	for _, bucket := range health.GetFailureClassifications() {
		notes = append(notes, "failure_classification="+bucket.GetClassification()+":"+strconv.Itoa(int(bucket.GetCount())))
	}
	out = append(out, Gap{
		ID:              "empirical/test-genie/fleet",
		Axis:            AxisEmpirical,
		Title:           "test-genie fleet has sustained failed validation evidence",
		Global:          true,
		EvidenceSource:  "test-genie",
		EvidenceLocator: "test-genie://fleet-health?window_days=" + strconv.Itoa(int(health.GetWindowDays())),
		Recurrence:      failed,
		Notes:           notes,
	})
	return out, nil
}

func countFleetPassed(health *runsv1.FleetHealth) int {
	passed := 0
	for _, scenario := range health.GetScenarios() {
		if scenario == nil {
			continue
		}
		passed += int(scenario.GetPassedRuns())
	}
	return passed
}
