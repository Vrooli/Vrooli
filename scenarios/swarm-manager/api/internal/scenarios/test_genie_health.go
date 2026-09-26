package scenarios

import (
	"context"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// TestGenieHealthClient is deliberately limited to canonical persisted
// findings evidence and Test Genie's read-only freshness authority. It cannot
// start runs or access provider internals.
type TestGenieHealthClient interface {
	GetRunFindings(context.Context, *connect.Request[runspb.GetRunFindingsRequest]) (*connect.Response[runspb.GetRunFindingsResponse], error)
	CheckFreshness(context.Context, *connect.Request[runspb.CheckFreshnessRequest]) (*connect.Response[runspb.CheckFreshnessResponse], error)
}

type TestGenieHealthSource struct {
	resolver   scenarioURLResolver
	httpClient connect.HTTPClient
	timeout    time.Duration
}

func NewTestGenieHealthSource(timeout time.Duration) *TestGenieHealthSource {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &TestGenieHealthSource{resolver: discovery.NewResolver(discovery.ResolverConfig{}), httpClient: &http.Client{Timeout: timeout}, timeout: timeout}
}

func (s *TestGenieHealthSource) Snapshot(ctx context.Context, scenario string) ScenarioHealthSnapshot {
	if s == nil || s.resolver == nil {
		return ScenarioHealthSnapshot{EvidenceState: HealthEvidenceUnavailable, Reason: "Test Genie integration is not configured."}
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	url, err := s.resolver.ResolveScenarioURLDefault(ctx, "test-genie")
	if err != nil {
		return ScenarioHealthSnapshot{EvidenceState: HealthEvidenceUnavailable, Reason: "Test Genie integration is unavailable."}
	}
	return ProjectTestGenieHealth(ctx, runsconnect.NewRunsServiceClient(s.httpClient, strings.TrimRight(url, "/")), scenario)
}

// ProjectTestGenieHealth maps provider presentation without interpreting its
// phase semantics. Errors become explicit unavailable evidence for callers.
func ProjectTestGenieHealth(ctx context.Context, client TestGenieHealthClient, scenario string) ScenarioHealthSnapshot {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return ScenarioHealthSnapshot{EvidenceState: HealthEvidenceNone, Reason: "A scenario is required to locate Test Genie evidence."}
	}
	if client == nil {
		return ScenarioHealthSnapshot{EvidenceState: HealthEvidenceUnavailable, Reason: "Test Genie integration is not configured."}
	}
	response, err := client.GetRunFindings(ctx, connect.NewRequest(&runspb.GetRunFindingsRequest{Target: scenario, RunId: "latest"}))
	if err != nil || response == nil || response.Msg == nil {
		return providerFailure(err, "retrieve canonical findings")
	}
	result := response.Msg
	if strings.TrimSpace(result.GetRunId()) == "" {
		return ScenarioHealthSnapshot{EvidenceState: HealthEvidenceNone, Reason: "Test Genie has no completed comparable evidence for this scenario."}
	}
	snapshot := ScenarioHealthSnapshot{EvidenceState: HealthEvidenceFresh, SourceRunID: result.GetRunId(), ObservedAt: result.GetCompletedAt(), Verdict: result.GetVerdict()}
	phaseNames := make([]string, 0, len(result.GetPhases()))
	for _, phase := range result.GetPhases() {
		presentation := phase.GetPhasePresentation()
		if presentation == nil {
			continue
		}
		snapshot.Phases = append(snapshot.Phases, projectPhase(phase.GetName(), phase.GetStatus(), presentation))
		phaseNames = append(phaseNames, phase.GetName())
	}
	if len(snapshot.Phases) == 0 {
		snapshot.EvidenceState = HealthEvidenceDegraded
		snapshot.Reason = "The latest Test Genie evidence has no canonical phase presentation."
		return snapshot
	}
	freshness, err := client.CheckFreshness(ctx, connect.NewRequest(&runspb.CheckFreshnessRequest{Target: scenario, Phases: phaseNames}))
	if err != nil || freshness == nil || freshness.Msg == nil {
		return providerFailure(err, "check evidence freshness")
	}
	snapshot.Freshness = "fresh"
	for _, phase := range freshness.Msg.GetPhases() {
		if phase.GetStatus() == "fresh" {
			continue
		}
		snapshot.EvidenceState = HealthEvidenceStale
		snapshot.Freshness = "stale"
		snapshot.Reason = "Test Genie reports one or more presented phases as " + phase.GetStatus() + " for the current scenario tree."
		break
	}
	return snapshot
}

func providerFailure(err error, operation string) ScenarioHealthSnapshot {
	if connect.CodeOf(err) == connect.CodeNotFound {
		return ScenarioHealthSnapshot{EvidenceState: HealthEvidenceNone, Reason: "Test Genie has no completed comparable evidence for this scenario."}
	}
	if code := connect.CodeOf(err); code == connect.CodeUnavailable || code == connect.CodeDeadlineExceeded {
		return ScenarioHealthSnapshot{EvidenceState: HealthEvidenceUnavailable, Reason: "Test Genie integration is unavailable while attempting to " + operation + "."}
	}
	return ScenarioHealthSnapshot{EvidenceState: HealthEvidenceDegraded, Reason: "Test Genie could not " + operation + "."}
}

func projectPhase(name, verdict string, presentation *commonv1.PhasePresentation) ScenarioHealthPhase {
	return ScenarioHealthPhase{
		Phase: name, Verdict: verdict, CurrentRung: presentation.GetCurrentLevel(), NextRung: presentation.GetNextLevel(),
		PriorityCapabilityID: presentation.GetFocusCapabilityId(), PriorityCapabilityLabel: presentation.GetFocusCapabilityLabel(),
		BlockingCodes:     append([]string(nil), presentation.GetBlockingFindingCodes()...),
		RemediationTopics: append([]string(nil), presentation.GetDocumentationTopics()...),
	}
}
