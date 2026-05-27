package workflowreplay

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	wrpb "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/workflowreplay"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

type fakeRuns struct {
	runs   []*runspb.RunInfo
	videos []*runspb.RunVideo
}

func (f fakeRuns) ListRuns(_ context.Context, _ string, _ int) ([]*runspb.RunInfo, error) {
	return f.runs, nil
}

func (f fakeRuns) GetRun(_ context.Context, _, runID string) (*runspb.RunInfo, error) {
	for _, r := range f.runs {
		if r.RunId == runID {
			return r, nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, nil)
}

func (f fakeRuns) ListRunVideos(_ context.Context, _, _ string) ([]*runspb.RunVideo, error) {
	return f.videos, nil
}

func playbooksRun(id string) *runspb.RunInfo {
	return &runspb.RunInfo{
		RunId:  id,
		Status: "passed",
		Phases: []*runspb.PhaseInfo{{Name: "playbooks", Status: "passed", DurationSeconds: 12.5}},
	}
}

func TestListRecentRunsFiltersToPlaybooks(t *testing.T) {
	srv := NewServer(Deps{Runs: fakeRuns{runs: []*runspb.RunInfo{
		playbooksRun("r1"),
		{RunId: "r2", Status: "passed", Phases: []*runspb.PhaseInfo{{Name: "unit", Status: "passed"}}},
		playbooksRun("r3"),
	}}})

	resp, err := srv.ListRecentRuns(context.Background(), connect.NewRequest(&wrpb.ListRecentRunsRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatal(err)
	}
	got := resp.Msg.GetRuns()
	if len(got) != 2 {
		t.Fatalf("expected 2 playbooks runs, got %d", len(got))
	}
	if got[0].RunId != "r1" || got[1].RunId != "r3" {
		t.Fatalf("unexpected runs: %+v", got)
	}
	if got[0].PlaybooksStatus != "passed" || got[0].PlaybooksDurationSeconds != 12.5 {
		t.Fatalf("playbooks phase not mapped: %+v", got[0])
	}
}

func TestListRecentRunsRequiresScenario(t *testing.T) {
	srv := NewServer(Deps{Runs: fakeRuns{}})
	_, err := srv.ListRecentRuns(context.Background(), connect.NewRequest(&wrpb.ListRecentRunsRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestGetRunDetailReturnsVideos(t *testing.T) {
	srv := NewServer(Deps{Runs: fakeRuns{
		runs:   []*runspb.RunInfo{playbooksRun("r1")},
		videos: []*runspb.RunVideo{{Workflow: "login", RelPath: "automation/login/video/a.webm", SizeBytes: 99}},
	}})
	resp, err := srv.GetRunDetail(context.Background(), connect.NewRequest(&wrpb.GetRunDetailRequest{Scenario: "demo", RunId: "r1"}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetRun().GetRunId() != "r1" {
		t.Fatalf("unexpected run: %+v", resp.Msg.GetRun())
	}
	vids := resp.Msg.GetVideos()
	if len(vids) != 1 || vids[0].RelPath != "automation/login/video/a.webm" {
		t.Fatalf("unexpected videos: %+v", vids)
	}
}
