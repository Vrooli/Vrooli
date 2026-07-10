package workflowreplay

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	wrpb "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/workflowreplay"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

type fakeRuns struct {
	runs      []*runspb.RunInfo
	artifacts map[string][]*runspb.ArtifactRef
}

func (f fakeRuns) ListRuns(_ context.Context, _ string, _ int) ([]*runspb.RunInfo, error) {
	return f.runs, nil
}
func (f fakeRuns) GetRun(_ context.Context, _, runID string) (*runspb.RunInfo, error) {
	for _, run := range f.runs {
		if run.GetRunId() == runID {
			return run, nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, nil)
}
func (f fakeRuns) ListRunArtifacts(_ context.Context, _, runID string, kinds []string) ([]*runspb.ArtifactRef, error) {
	if len(kinds) != 1 || kinds[0] != workflowVideoKind {
		panic("workflow lens did not query by artifact kind")
	}
	return f.artifacts[runID], nil
}

func TestListRecentRunsSelectsWorkflowEvidenceNotPhase(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	srv := NewServer(Deps{Runs: fakeRuns{
		runs: []*runspb.RunInfo{
			{RunId: "future", Status: "passed", Phases: []*runspb.PhaseInfo{{Name: "future-provider", Status: "passed"}}},
			{RunId: "legacy-phase-only", Status: "passed", Phases: []*runspb.PhaseInfo{{Name: "playbooks", Status: "passed"}}},
		},
		artifacts: map[string][]*runspb.ArtifactRef{
			"future": {{Id: "artifact_0123456789abcdef0123456789abcdef", Kind: workflowVideoKind, Label: "login", ProducingPhase: "future-provider"}},
		},
	}})
	resp, err := srv.ListRecentRuns(context.Background(), connect.NewRequest(&wrpb.ListRecentRunsRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.GetRuns()) != 1 || resp.Msg.GetRuns()[0].GetRunId() != "future" {
		t.Fatalf("artifact-driven runs = %+v", resp.Msg.GetRuns())
	}
}

func TestGetRunDetailReturnsOpaqueWorkflowArtifacts(t *testing.T) {
	artifact := &runspb.ArtifactRef{Id: "artifact_0123456789abcdef0123456789abcdef", Kind: workflowVideoKind, Label: "login"}
	srv := NewServer(Deps{Runs: fakeRuns{
		runs:      []*runspb.RunInfo{{RunId: "r1", Status: "passed"}},
		artifacts: map[string][]*runspb.ArtifactRef{"r1": {artifact}},
	}})
	resp, err := srv.GetRunDetail(context.Background(), connect.NewRequest(&wrpb.GetRunDetailRequest{Scenario: "demo", RunId: "r1"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.GetArtifacts()) != 1 || resp.Msg.GetArtifacts()[0].GetId() != artifact.GetId() {
		t.Fatalf("artifacts = %+v", resp.Msg.GetArtifacts())
	}
}

func TestListRecentRunsRequiresScenario(t *testing.T) {
	_, err := NewServer(Deps{Runs: fakeRuns{}}).ListRecentRuns(context.Background(), connect.NewRequest(&wrpb.ListRecentRunsRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}
