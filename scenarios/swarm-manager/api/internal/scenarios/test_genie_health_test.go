package scenarios

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

type fakeTestGenieHealthClient struct {
	findings   *runspb.GetRunFindingsResponse
	findingsE  error
	freshness  *runspb.CheckFreshnessResponse
	freshnessE error
}

func (f fakeTestGenieHealthClient) GetRunFindings(context.Context, *connect.Request[runspb.GetRunFindingsRequest]) (*connect.Response[runspb.GetRunFindingsResponse], error) {
	if f.findingsE != nil {
		return nil, f.findingsE
	}
	return connect.NewResponse(f.findings), nil
}

func (f fakeTestGenieHealthClient) CheckFreshness(context.Context, *connect.Request[runspb.CheckFreshnessRequest]) (*connect.Response[runspb.CheckFreshnessResponse], error) {
	if f.freshnessE != nil {
		return nil, f.freshnessE
	}
	return connect.NewResponse(f.freshness), nil
}

func canonicalFindings() *runspb.GetRunFindingsResponse {
	return &runspb.GetRunFindingsResponse{
		RunId: "run-42", CompletedAt: "2026-07-26T12:00:00Z", Verdict: "failed",
		Phases: []*runspb.RunFindingsPhase{{Name: "unit", Status: "failed", PhasePresentation: &commonv1.PhasePresentation{
			CurrentLevel: "L1", NextLevel: "L2", FocusCapabilityId: "coverage", FocusCapabilityLabel: "Coverage", BlockingFindingCodes: []string{"coverage_gap"},
		}}},
	}
}

func TestProjectTestGenieHealthFreshPreservesProviderPresentation(t *testing.T) {
	snapshot := ProjectTestGenieHealth(context.Background(), fakeTestGenieHealthClient{
		findings: canonicalFindings(), freshness: &runspb.CheckFreshnessResponse{Phases: []*runspb.PhaseFreshness{{Phase: "unit", Status: "fresh"}}},
	}, "demo")
	if snapshot.EvidenceState != HealthEvidenceFresh || snapshot.SourceRunID != "run-42" || snapshot.Freshness != "fresh" {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
	if len(snapshot.Phases) != 1 || snapshot.Phases[0].PriorityCapabilityID != "coverage" || snapshot.Phases[0].CurrentRung != "L1" {
		t.Fatalf("provider presentation was not preserved: %#v", snapshot.Phases)
	}
}

func TestProjectTestGenieHealthMarksStaleFromProviderFreshness(t *testing.T) {
	snapshot := ProjectTestGenieHealth(context.Background(), fakeTestGenieHealthClient{
		findings: canonicalFindings(), freshness: &runspb.CheckFreshnessResponse{Phases: []*runspb.PhaseFreshness{{Phase: "unit", Status: "stale"}}},
	}, "demo")
	if snapshot.EvidenceState != HealthEvidenceStale || snapshot.Freshness != "stale" || snapshot.Reason == "" {
		t.Fatalf("expected explicit stale evidence, got %#v", snapshot)
	}
}

func TestProjectTestGenieHealthDistinguishesUnavailableAndDegraded(t *testing.T) {
	unavailable := ProjectTestGenieHealth(context.Background(), fakeTestGenieHealthClient{findingsE: connect.NewError(connect.CodeUnavailable, errors.New("offline"))}, "demo")
	if unavailable.EvidenceState != HealthEvidenceUnavailable {
		t.Fatalf("expected unavailable, got %#v", unavailable)
	}
	degraded := ProjectTestGenieHealth(context.Background(), fakeTestGenieHealthClient{findings: canonicalFindings(), freshnessE: connect.NewError(connect.CodeInternal, errors.New("bad response"))}, "demo")
	if degraded.EvidenceState != HealthEvidenceDegraded {
		t.Fatalf("expected degraded, got %#v", degraded)
	}
}

func TestProjectTestGenieHealthTreatsMissingRunAsNoEvidence(t *testing.T) {
	snapshot := ProjectTestGenieHealth(context.Background(), fakeTestGenieHealthClient{findings: &runspb.GetRunFindingsResponse{}}, "demo")
	if snapshot.EvidenceState != HealthEvidenceNone {
		t.Fatalf("expected no evidence, got %#v", snapshot)
	}
}

func TestProjectTestGenieHealthTreatsProviderNotFoundAsNoEvidence(t *testing.T) {
	snapshot := ProjectTestGenieHealth(context.Background(), fakeTestGenieHealthClient{findingsE: connect.NewError(connect.CodeNotFound, errors.New("no run"))}, "demo")
	if snapshot.EvidenceState != HealthEvidenceNone {
		t.Fatalf("expected no evidence, got %#v", snapshot)
	}
}
