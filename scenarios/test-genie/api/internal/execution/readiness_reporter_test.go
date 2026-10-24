package execution

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/readiness"
	"test-genie/internal/orchestrator"
)

type recordingReadinessClient struct {
	request  *readinessv1.ReportEvidenceRequest
	requests []*readinessv1.ReportEvidenceRequest
}

type recordingReleaseComparator struct {
	target, predecessor, current string
	result                       ReleaseComparison
}

func (c *recordingReleaseComparator) CompareRelease(_ context.Context, target, predecessor, current string) (ReleaseComparison, error) {
	c.target, c.predecessor, c.current = target, predecessor, current
	return c.result, nil
}

func (c *recordingReadinessClient) ReportEvidence(_ context.Context, req *connect.Request[readinessv1.ReportEvidenceRequest]) (*connect.Response[readinessv1.ReportEvidenceResponse], error) {
	c.request = req.Msg
	c.requests = append(c.requests, req.Msg)
	return connect.NewResponse(&readinessv1.ReportEvidenceResponse{Accepted: true}), nil
}

func TestDeploymentManagerReadinessReporterReportsPerformanceOnlyWhenMeasured(t *testing.T) {
	client := &recordingReadinessClient{}
	reporter := &DeploymentManagerReadinessReporter{client: client, now: time.Now}
	request := orchestrator.SuiteExecutionRequest{ScenarioName: "demo", ReleaseIdentity: &orchestrator.ReleaseIdentity{
		ProfileID: "profile-1", CandidateCommit: "commit-1", ArtifactDigest: "sha256:artifact", Targets: []string{"linux"}, Channel: "stable", PolicyVersion: 2,
	}}
	result := &orchestrator.SuiteExecutionResult{RunID: "run-performance", Verdict: orchestrator.SuiteVerdictPass, CompletedAt: time.Unix(200, 0).UTC(), Phases: []orchestrator.PhaseExecutionResult{{Name: "performance", Status: "passed"}}}
	if err := reporter.Report(context.Background(), request, result); err != nil {
		t.Fatalf("Report() error = %v", err)
	}
	if len(client.requests) != 2 || client.requests[1].GetCriterionId() != performanceCriterion || client.requests[1].GetProducerBinding() != performanceBinding || client.requests[1].GetStatus() != "passed" {
		t.Fatalf("reported evidence = %#v", client.requests)
	}
}

func TestDeploymentManagerReadinessReporterReportsOnlyAttributedTerminalSuite(t *testing.T) {
	client := &recordingReadinessClient{}
	reporter := &DeploymentManagerReadinessReporter{client: client, now: func() time.Time { return time.Unix(100, 0).UTC() }}
	completed := time.Unix(200, 0).UTC()
	err := reporter.Report(context.Background(), orchestrator.SuiteExecutionRequest{
		ScenarioName: "demo",
		ReleaseIdentity: &orchestrator.ReleaseIdentity{
			ProfileID: "profile-1", CandidateCommit: "commit-1", ArtifactDigest: "sha256:artifact",
			Targets: []string{"linux-x64"}, Channel: "stable", PolicyVersion: 2,
			CandidateID: "candidate-1", DestinationRevisionID: "destination-1", AuthorizationEpoch: 4,
		},
	}, &orchestrator.SuiteExecutionResult{RunID: "run-1", Verdict: orchestrator.SuiteVerdictPass, CompletedAt: completed})
	if err != nil {
		t.Fatalf("Report() error = %v", err)
	}
	if client.request == nil {
		t.Fatal("ReportEvidence was not called")
	}
	if client.request.GetScenario() != "demo" || client.request.GetCriterionId() != suiteStateCriterion || client.request.GetProducerBinding() != suiteStateBinding {
		t.Fatalf("request routing = scenario %q criterion %q binding %q", client.request.GetScenario(), client.request.GetCriterionId(), client.request.GetProducerBinding())
	}
	if client.request.GetStatus() != "passed" || client.request.GetEvidenceReference() != "test-genie:run:run-1" || client.request.GetArtifactDigest() != "sha256:artifact" {
		t.Fatalf("request attribution = %#v", client.request)
	}
	if got := client.request.GetObservedAt().AsTime(); !got.Equal(completed) {
		t.Fatalf("observed_at = %s, want %s", got, completed)
	}
}

func TestDeploymentManagerReadinessReporterReportsReleaseComparison(t *testing.T) {
	client := &recordingReadinessClient{}
	comparator := &recordingReleaseComparator{result: ReleaseComparison{
		Verdict: "clean", Behavior: "clean", Coverage: "measured", Compatibility: "compatible", Provenance: "strict",
	}}
	reporter := &DeploymentManagerReadinessReporter{client: client, now: time.Now, comparator: comparator}
	request := orchestrator.SuiteExecutionRequest{ScenarioName: "scenario:demo", ReleaseIdentity: &orchestrator.ReleaseIdentity{
		ProfileID: "profile-1", CandidateCommit: "commit-1", ArtifactDigest: "sha256:artifact", Targets: []string{"linux"}, Channel: "stable", PolicyVersion: 2, PredecessorRunID: "run-before",
	}}
	if err := reporter.Report(context.Background(), request, &orchestrator.SuiteExecutionResult{RunID: "run-current", Verdict: orchestrator.SuiteVerdictPass}); err != nil {
		t.Fatalf("Report() error = %v", err)
	}
	if comparator.target != "demo" || comparator.predecessor != "run-before" || comparator.current != "run-current" {
		t.Fatalf("comparison inputs = %#v", comparator)
	}
	if len(client.requests) != 2 || client.requests[1].GetCriterionId() != regressionCriterion || client.requests[1].GetProducerBinding() != regressionBinding || client.requests[1].GetStatus() != "passed" {
		t.Fatalf("reported comparison evidence = %#v", client.requests)
	}
}

func TestDeploymentManagerReadinessReporterReportsPerformanceAndComparison(t *testing.T) {
	client := &recordingReadinessClient{}
	comparator := &recordingReleaseComparator{result: ReleaseComparison{
		Verdict: "clean", Behavior: "clean", Coverage: "measured", Compatibility: "compatible", Provenance: "strict",
	}}
	reporter := &DeploymentManagerReadinessReporter{client: client, now: time.Now, comparator: comparator}
	request := orchestrator.SuiteExecutionRequest{ScenarioName: "demo", ReleaseIdentity: &orchestrator.ReleaseIdentity{
		ProfileID: "profile-1", CandidateCommit: "commit-1", ArtifactDigest: "sha256:artifact", Targets: []string{"linux"}, Channel: "stable", PolicyVersion: 2, PredecessorRunID: "run-before",
	}}
	result := &orchestrator.SuiteExecutionResult{RunID: "run-current", Verdict: orchestrator.SuiteVerdictPass, Phases: []orchestrator.PhaseExecutionResult{{Name: "performance", Status: "passed"}}}
	if err := reporter.Report(context.Background(), request, result); err != nil {
		t.Fatalf("Report() error = %v", err)
	}
	if len(client.requests) != 3 {
		t.Fatalf("reported evidence count = %d, want suite, performance, and comparison", len(client.requests))
	}
	if client.requests[1].GetCriterionId() != performanceCriterion || client.requests[2].GetCriterionId() != regressionCriterion {
		t.Fatalf("reported criteria = %q, %q, %q", client.requests[0].GetCriterionId(), client.requests[1].GetCriterionId(), client.requests[2].GetCriterionId())
	}
}

func TestDeploymentManagerReadinessReporterFailsComparisonClosed(t *testing.T) {
	client := &recordingReadinessClient{}
	reporter := &DeploymentManagerReadinessReporter{client: client, now: time.Now}
	err := reporter.Report(context.Background(), orchestrator.SuiteExecutionRequest{ScenarioName: "demo", ReleaseIdentity: &orchestrator.ReleaseIdentity{
		ProfileID: "profile-1", CandidateCommit: "commit-1", ArtifactDigest: "sha256:artifact", Targets: []string{"linux"}, Channel: "stable", PolicyVersion: 2, PredecessorRunID: "run-before",
	}}, &orchestrator.SuiteExecutionResult{RunID: "run-current", Verdict: orchestrator.SuiteVerdictPass})
	if err == nil || !strings.Contains(err.Error(), "predecessor comparison") {
		t.Fatalf("Report() error = %v, want closed comparison error", err)
	}
	if len(client.requests) != 1 || client.requests[0].GetCriterionId() != suiteStateCriterion {
		t.Fatalf("comparison should not be reported without a comparator: %#v", client.requests)
	}
}

func TestDeploymentManagerReadinessReporterRejectsIncompleteIdentity(t *testing.T) {
	client := &recordingReadinessClient{}
	reporter := &DeploymentManagerReadinessReporter{client: client, now: time.Now}
	err := reporter.Report(context.Background(), orchestrator.SuiteExecutionRequest{
		ScenarioName:    "demo",
		ReleaseIdentity: &orchestrator.ReleaseIdentity{ProfileID: "profile-1"},
	}, &orchestrator.SuiteExecutionResult{RunID: "run-1", Verdict: orchestrator.SuiteVerdictPass})
	if err == nil {
		t.Fatal("Report() accepted incomplete release identity")
	}
	if client.request != nil {
		t.Fatal("ReportEvidence was called for incomplete identity")
	}
}
