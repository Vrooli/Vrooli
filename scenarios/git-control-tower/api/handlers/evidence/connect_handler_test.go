package evidence

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	evidencepb "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/evidence"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

type fakeRuns struct {
	runs     []*runspb.RunInfo
	catalogs map[string]*runspb.ListRunArtifactsResponse
}

func (f fakeRuns) StartRun(_ context.Context, request *runspb.StartRunRequest) (*runspb.StartRunResponse, error) {
	return &runspb.StartRunResponse{RunId: "run-started", Target: request.GetTarget()}, nil
}

func (f fakeRuns) ListRuns(_ context.Context, _ string, limit int) ([]*runspb.RunInfo, error) {
	if limit > 0 && limit < len(f.runs) {
		return f.runs[:limit], nil
	}
	return f.runs, nil
}

func (f fakeRuns) GetRun(_ context.Context, _, runID string) (*runspb.GetRunResponse, error) {
	for _, run := range f.runs {
		if run.GetRunId() == runID {
			return &runspb.GetRunResponse{Run: run, TerminalSnapshotSchemaVersion: 1}, nil
		}
	}
	return &runspb.GetRunResponse{}, nil
}

func (f fakeRuns) ListRunArtifacts(_ context.Context, _, runID string, kinds []string) (*runspb.ListRunArtifactsResponse, error) {
	source := f.catalogs[runID]
	if len(kinds) == 0 {
		return proto.Clone(source).(*runspb.ListRunArtifactsResponse), nil
	}
	wanted := map[string]bool{}
	for _, kind := range kinds {
		wanted[kind] = true
	}
	result := proto.Clone(source).(*runspb.ListRunArtifactsResponse)
	result.Artifacts = nil
	for _, artifact := range source.GetArtifacts() {
		if wanted[artifact.GetKind()] {
			result.Artifacts = append(result.Artifacts, artifact)
		}
	}
	return result, nil
}

func unknownFixture() fakeRuns {
	run := &runspb.RunInfo{
		RunId: "run-future", Target: "demo", Status: "failed", StartedAt: "2026-07-10T12:00:00Z",
		Phases: []*runspb.PhaseInfo{{Name: "future-health", Status: "failed", DurationSeconds: 3.25}},
		DescriptorSnapshot: &runspb.RunDescriptorSnapshot{SchemaVersion: 1, Digest: "sha256:future", Phases: []*runspb.RunPhaseDescriptor{{
			Phase: "future-health", DisplayName: "Future Health", Provider: "future-provider", PhaseClass: "future-class",
			Dimensions: []string{"novel-dimension"}, EvidenceKinds: []string{"future.evidence"},
		}}},
	}
	return fakeRuns{
		runs: []*runspb.RunInfo{run},
		catalogs: map[string]*runspb.ListRunArtifactsResponse{
			"run-future": {
				SchemaVersion: 1, Digest: "sha256:catalog", Artifacts: []*runspb.ArtifactRef{{
					Id: "opaque-future", Kind: "future.evidence", Label: "Future artifact", ProducingPhase: "future-health",
					MediaType: "application/x-future", Metadata: map[string]string{"future-key": "future-value"},
				}},
			},
		},
	}
}

func TestGetRunPreservesUnknownDescriptorAndArtifactMetadata(t *testing.T) {
	server := NewServer(Deps{Runs: unknownFixture()})
	resp, err := server.GetRun(context.Background(), connect.NewRequest(&evidencepb.GetRunRequest{
		Scenario: "demo", RunId: "run-future", ArtifactKinds: []string{"future.evidence"},
	}))
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	descriptor := resp.Msg.GetRun().GetDescriptorSnapshot().GetPhases()[0]
	if descriptor.GetPhase() != "future-health" || descriptor.GetProvider() != "future-provider" || descriptor.GetPhaseClass() != "future-class" {
		t.Fatalf("descriptor was not preserved: %+v", descriptor)
	}
	artifact := resp.Msg.GetArtifacts()[0]
	if artifact.GetKind() != "future.evidence" || artifact.GetProducingPhase() != "future-health" || artifact.GetMetadata()["future-key"] != "future-value" {
		t.Fatalf("artifact was not preserved: %+v", artifact)
	}
}

func TestListRunsFiltersCapturedDescriptorMetadataAndPaginates(t *testing.T) {
	fixture := unknownFixture()
	fixture.runs = append(fixture.runs, &runspb.RunInfo{RunId: "run-other", Target: "demo", Status: "passed"})
	server := NewServer(Deps{Runs: fixture})
	resp, err := server.ListRuns(context.Background(), connect.NewRequest(&evidencepb.ListRunsRequest{
		Scenario: "demo", Provider: "future-provider", Dimension: "novel-dimension", Limit: 1,
	}))
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if resp.Msg.GetTotal() != 1 || len(resp.Msg.GetRuns()) != 1 || resp.Msg.GetRuns()[0].GetRunId() != "run-future" {
		t.Fatalf("unexpected response: %+v", resp.Msg)
	}
}

func TestListEvidenceSelectsKindsAcrossProducerPhases(t *testing.T) {
	fixture := unknownFixture()
	fixture.catalogs["run-future"].Artifacts = append(fixture.catalogs["run-future"].Artifacts,
		&runspb.ArtifactRef{Id: "video-a", Kind: "workflow.video", ProducingPhase: "future-health"},
		&runspb.ArtifactRef{Id: "image-a", Kind: "visual.screenshot", ProducingPhase: "another-future-phase"},
	)
	server := NewServer(Deps{Runs: fixture})
	resp, err := server.ListEvidence(context.Background(), connect.NewRequest(&evidencepb.ListEvidenceRequest{
		Scenario: "demo", Kinds: []string{"workflow.video", "visual.screenshot"}, Limit: 10,
	}))
	if err != nil {
		t.Fatalf("ListEvidence: %v", err)
	}
	if len(resp.Msg.GetItems()) != 2 {
		t.Fatalf("expected two kind matches, got %+v", resp.Msg.GetItems())
	}
	producers := map[string]bool{}
	for _, item := range resp.Msg.GetItems() {
		producers[item.GetArtifact().GetProducingPhase()] = true
	}
	if !producers["future-health"] || !producers["another-future-phase"] {
		t.Fatalf("producer metadata was filtered or rewritten: %+v", producers)
	}
}
