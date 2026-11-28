package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	gctsourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/source_distributions"
	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-repository/v1/source"
)

type recordingSourceReader struct {
	item        *sourcev1.Distribution
	unavailable bool
	partial     bool
	calls       []string
}

func (f *recordingSourceReader) List(context.Context, string, string, int) ([]*sourcev1.Distribution, string, string, error) {
	f.calls = append(f.calls, "list")
	if f.unavailable {
		return nil, "scenario-to-repository", "unavailable", errors.New("owner offline")
	}
	return []*sourcev1.Distribution{f.item}, "scenario-to-repository", "current", nil
}
func (f *recordingSourceReader) Get(context.Context, string) (*sourcev1.Distribution, error) {
	f.calls = append(f.calls, "get")
	if f.unavailable {
		return nil, errors.New("owner offline")
	}
	return f.item, nil
}
func (f *recordingSourceReader) Contents(context.Context, string) (*sourcev1.DistributionContentsResponse, error) {
	f.calls = append(f.calls, "contents")
	if f.partial {
		return nil, errors.New("contents replica unavailable")
	}
	return &sourcev1.DistributionContentsResponse{Contents: []*sourcev1.DistributionContent{{Path: "api/main.go", Category: "application", Digest: "sha256:file"}}, Exclusions: []*sourcev1.DistributionExclusion{{Path: ".env", Category: "private material", SafeReason: "privacy policy"}}, RuntimeRequirements: []string{"postgres"}, SourceOfTruth: "scenario-to-repository", Freshness: "current"}, nil
}
func (f *recordingSourceReader) Handoff(context.Context, string) (*sourcev1.PublicationHandoffResponse, error) {
	f.calls = append(f.calls, "handoff")
	return &sourcev1.PublicationHandoffResponse{Status: "awaiting_human", ApprovalReference: "decision-1", ReadbackOracle: "exact manifest read-back"}, nil
}
func (f *recordingSourceReader) Drift(context.Context, string) (*sourcev1.DistributionDriftResponse, error) {
	f.calls = append(f.calls, "drift")
	return &sourcev1.DistributionDriftResponse{State: "destination_edits_detected", DestinationChanged: true, Actions: []string{"review independent edits"}}, nil
}

func testDistribution() *sourcev1.Distribution {
	return &sourcev1.Distribution{DistributionId: "distribution-1", Scenario: "demo", SourceDigest: "sha256:source", ClosureDigest: "sha256:closure", RecipeDigest: "sha256:recipe", PolicyDigest: "sha256:policy", ArtifactId: "artifact-1", ArtifactDigest: "sha256:artifact", VerificationStatus: "passed", VerificationReceipt: "receipt-1", DeploymentManagerDecision: "decision-1", PublicationStatus: "awaiting_human", DestinationReference: "operator/repo", SourceOfTruth: "scenario-to-repository", Freshness: "current", DriftState: "up_to_date"}
}

func TestSourceDistributionConnectListProjectsExactIdentity(t *testing.T) {
	fake := &recordingSourceReader{item: testDistribution()}
	srv := &Server{sourceDistributions: fake}
	resp, err := (sourceDistributionConnectServer{server: srv}).ListDistributions(context.Background(), connect.NewRequest(&gctsourcev1.ListDistributionsRequest{Scenario: "demo", RepositoryContext: "repo-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Msg.GetAvailable() || len(resp.Msg.GetDistributions()) != 1 || resp.Msg.GetDistributions()[0].GetClosureDigest() != "sha256:closure" || resp.Msg.GetDistributions()[0].GetDeploymentManagerDecision() != "decision-1" {
		t.Fatalf("unexpected typed list response: %+v", resp.Msg)
	}
	if len(fake.calls) != 1 || fake.calls[0] != "list" {
		t.Fatalf("unexpected owner calls: %v", fake.calls)
	}
}

func TestSourceDistributionConnectDetailKeepsApprovalPublicationAndDriftSeparate(t *testing.T) {
	fake := &recordingSourceReader{item: testDistribution()}
	srv := &Server{sourceDistributions: fake}
	resp, err := (sourceDistributionConnectServer{server: srv}).GetDistributionDetails(context.Background(), connect.NewRequest(&gctsourcev1.GetDistributionDetailsRequest{DistributionId: "distribution-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Msg.GetAvailable() || resp.Msg.GetDistribution().GetPublicationStatus() != "awaiting_human" || resp.Msg.GetDistribution().GetDeploymentManagerDecision() != "decision-1" || resp.Msg.GetDrift().GetState() != "destination_edits_detected" || resp.Msg.GetExclusions()[0].GetCategory() != "private material" {
		t.Fatalf("typed detail lost independent evidence: %+v", resp.Msg)
	}
	if len(fake.calls) != 4 {
		t.Fatalf("expected owner read set, got %v", fake.calls)
	}
}

func TestSourceDistributionConnectUnavailableIsTypedAndNonMutating(t *testing.T) {
	fake := &recordingSourceReader{unavailable: true}
	srv := &Server{sourceDistributions: fake}
	_, err := (sourceDistributionConnectServer{server: srv}).ListDistributions(context.Background(), connect.NewRequest(&gctsourcev1.ListDistributionsRequest{}))
	if connect.CodeOf(err) != connect.CodeUnavailable || err == nil || !strings.Contains(err.Error(), "owner offline") || len(fake.calls) != 1 || fake.calls[0] != "list" {
		t.Fatalf("unexpected unavailable result: err=%v calls=%v", err, fake.calls)
	}
}

func TestSourceDistributionConnectPreservesPartialEvidenceReason(t *testing.T) {
	fake := &recordingSourceReader{item: testDistribution(), partial: true}
	resp, err := (sourceDistributionConnectServer{server: &Server{sourceDistributions: fake}}).GetDistributionDetails(context.Background(), connect.NewRequest(&gctsourcev1.GetDistributionDetailsRequest{DistributionId: "distribution-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Msg.GetAvailable() || resp.Msg.GetUnavailableReason() != "contents unavailable: contents replica unavailable" || len(resp.Msg.GetContents()) != 0 {
		t.Fatalf("partial read hid its missing evidence: %+v", resp.Msg)
	}
}
