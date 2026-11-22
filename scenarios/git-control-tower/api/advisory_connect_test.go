package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	domainadvisory "git-control-tower/internal/advisory"
	"git-control-tower/internal/reviewjobs"
	"git-control-tower/internal/stagedmessages"
	testdb "github.com/vrooli/api-core/databasetest"
	advisoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

func TestAdvisoryConnectDraftValidatesAndReturnsDraft(t *testing.T) {
	subject := domainadvisory.ChangeSubject{
		RepositoryID: "repo", Kind: domainadvisory.SubjectCurrent,
		Scope:          domainadvisory.Scope{Paths: []string{"api/a.go"}, SelectionDigest: "sha256:selection"},
		SnapshotDigest: "sha256:snapshot",
	}
	digest, err := subject.Digest()
	if err != nil {
		t.Fatal(err)
	}
	server := advisoryConnectServer{}
	response, err := server.Draft(context.Background(), connect.NewRequest(&advisoryv1.DraftRequest{
		Kind:     "summary",
		Subject:  &advisoryv1.ChangeSubject{RepositoryId: "repo", Kind: string(domainadvisory.SubjectCurrent), Scope: &advisoryv1.SubjectScope{Paths: []string{"api/a.go"}, SelectionDigest: "sha256:selection"}, SnapshotDigest: "sha256:snapshot"},
		Evidence: &advisoryv1.EvidenceBundle{OperationId: "op-1", SubjectDigest: digest, Status: "partial", Claims: []*advisoryv1.Claim{{Text: "Adds recovery", EvidenceRefs: []string{"hunk:1"}}}, Unknowns: []string{"deployment timing"}, Coverage: &advisoryv1.Coverage{IncludedFiles: 1, OmittedFiles: 1, Omissions: []*advisoryv1.Omission{{Path: "asset.bin", Reason: "binary"}}}},
	}))
	if err != nil {
		t.Fatalf("Draft() error = %v", err)
	}
	got := response.Msg
	if got.Status != "partial" || got.Kind != "summary" || got.SubjectDigest != digest || got.Title != "Adds recovery" {
		t.Fatalf("unexpected draft response: %+v", got)
	}
	if len(got.EvidenceRefs) != 1 || got.EvidenceRefs[0] != "hunk:1" || len(got.Unknowns) != 1 || got.Coverage.GetOmittedFiles() != 1 || len(got.Coverage.GetOmissions()) != 1 {
		t.Fatalf("draft evidence was not preserved: %+v", got)
	}
}

func TestAdvisoryConnectGetStagedMessageMetricsUsesDurableOwner(t *testing.T) {
	store, err := reviewjobs.New(testdb.OpenSQLiteMemory(t))
	if err != nil {
		t.Fatal(err)
	}
	server := advisoryConnectServer{server: &Server{stagedMessages: stagedmessages.New(store, nil, nil, nil)}}
	response, err := server.GetStagedMessageMetrics(context.Background(), connect.NewRequest(&advisoryv1.GetStagedMessageMetricsRequest{}))
	if err != nil {
		t.Fatalf("GetStagedMessageMetrics returned error: %v", err)
	}
	if response.Msg.GetObservedAt() == "" || response.Msg.GetJobs() != 0 || response.Msg.FailureRate != nil || response.Msg.GetStuckThresholdSeconds() != int64((5*time.Minute).Seconds()) {
		t.Fatalf("unexpected empty durable metrics: %+v", response.Msg)
	}
}

func TestAdvisoryConnectDraftRejectsMissingAndInvalidRequests(t *testing.T) {
	server := advisoryConnectServer{}
	for name, req := range map[string]*connect.Request[advisoryv1.DraftRequest]{
		"nil request":     nil,
		"nil message":     connect.NewRequest((*advisoryv1.DraftRequest)(nil)),
		"invalid subject": connect.NewRequest(&advisoryv1.DraftRequest{Kind: "summary"}),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := server.Draft(context.Background(), req)
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("Draft() error = %v, want InvalidArgument", err)
			}
		})
	}
}

func TestAdvisoryProtoConversionsPreserveNestedEvidenceAndIgnoreNilEntries(t *testing.T) {
	protoSubject := &advisoryv1.ChangeSubject{
		RepositoryId: "repo", Kind: string(domainadvisory.SubjectRange), BaseRevision: "base", HeadRevision: "head", ParentRevision: "parent", SnapshotDigest: "snapshot",
		Scope: &advisoryv1.SubjectScope{Paths: []string{"a", "b"}, SelectionDigest: "selection"},
		Host:  &advisoryv1.HostSubject{Provider: "forge", InstanceId: "instance", RepositoryId: "host-repo", ChangeNumber: "42"},
	}
	subject := advisorySubjectFromProto(protoSubject)
	if subject.RepositoryID != "repo" || subject.Kind != domainadvisory.SubjectRange || subject.BaseRevision != "base" || subject.HeadRevision != "head" || subject.ParentRevision != "parent" || subject.Scope.SelectionDigest != "selection" || len(subject.Scope.Paths) != 2 || subject.Host == nil || subject.Host.ChangeNumber != "42" {
		t.Fatalf("subject conversion lost fields: %+v", subject)
	}
	protoEvidence := &advisoryv1.EvidenceBundle{
		OperationId: "op", SubjectDigest: "digest", Status: "complete", Unknowns: []string{"unknown"}, Versions: map[string]string{"scanner": "1"},
		Claims:     []*advisoryv1.Claim{nil, {Text: "claim", EvidenceRefs: []string{"ref"}}},
		Coverage:   &advisoryv1.Coverage{IncludedFiles: 2, OmittedFiles: 1, Omissions: []*advisoryv1.Omission{nil, {Path: "x", Reason: "binary"}}},
		Validation: []*advisoryv1.ValidationRef{nil, {ExecutionId: "exec", Availability: "available", Verdict: "pass"}},
	}
	evidence := advisoryEvidenceFromProto(protoEvidence)
	if evidence.OperationID != "op" || evidence.Versions["scanner"] != "1" || len(evidence.Claims) != 1 || evidence.Claims[0].EvidenceRefs[0] != "ref" || len(evidence.Coverage.Omissions) != 1 || len(evidence.Validation) != 1 || evidence.Validation[0].ExecutionID != "exec" {
		t.Fatalf("evidence conversion lost fields or retained nil entries: %+v", evidence)
	}
	protoEvidence.Unknowns[0] = "changed"
	protoEvidence.Claims[1].EvidenceRefs[0] = "changed"
	if evidence.Unknowns[0] != "unknown" || evidence.Claims[0].EvidenceRefs[0] != "ref" {
		t.Fatal("conversion shared mutable repeated fields with protobuf input")
	}
	coverage := advisoryCoverageToProto(domainadvisory.Coverage{IncludedFiles: 3, OmittedFiles: 1, Omissions: []domainadvisory.Omission{{Path: "secret", Reason: "filtered"}}})
	if coverage.GetIncludedFiles() != 3 || len(coverage.GetOmissions()) != 1 || !strings.EqualFold(coverage.GetOmissions()[0].GetPath(), "secret") {
		t.Fatalf("coverage conversion = %+v", coverage)
	}
	if advisorySubjectFromProto(nil).RepositoryID != "" || advisoryEvidenceFromProto(nil).OperationID != "" {
		t.Fatal("nil protobuf values should convert to zero domain values")
	}
}
