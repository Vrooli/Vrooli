package validationmatrix

import (
	"context"
	"encoding/json"
	"testing"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type fakeDeploymentClient struct {
	request *evidencev1.ReportTargetVerdictRequest
}

func (f *fakeDeploymentClient) ReportTargetVerdict(_ context.Context, request *connect.Request[evidencev1.ReportTargetVerdictRequest]) (*connect.Response[evidencev1.ReportTargetVerdictResponse], error) {
	f.request = request.Msg
	return connect.NewResponse(&evidencev1.ReportTargetVerdictResponse{Verdict: request.Msg.Verdict}), nil
}

func TestDeploymentReporterPreservesGateAndEvidenceProvenance(t *testing.T) {
	client := &fakeDeploymentClient{}
	reporter := NewDeploymentReporter(client, "profile-1", "commit-1")
	gate := &domainv1.ReleaseGate{MatrixId: "matrix-1", Passed: true, Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, RequiredCellCount: 1, PassingCellCount: 1}
	if err := reporter.ReportValidationGate(context.Background(), ReleaseVerdict{RunID: "run-1", MatrixID: "matrix-1", ScenarioName: "demo", ArtifactDigest: "sha256:artifact", Gate: gate, Evidence: validEvidence()}); err != nil {
		t.Fatal(err)
	}
	if client.request.GetProfileId() != "profile-1" || client.request.GetGitCommitHash() != "commit-1" {
		t.Fatalf("release identity lost: %+v", client.request)
	}
	if client.request.GetVerdict().GetDisposition() != commonv1.Disposition_DISPOSITION_PASSED || client.request.GetVerdict().GetRunId() != "run-1" || len(client.request.GetVerdict().GetRefs()) != 4 {
		t.Fatalf("release verdict lost evidence: %+v", client.request.GetVerdict())
	}
	var detail map[string]any
	if err := json.Unmarshal([]byte(client.request.GetVerdict().GetDetail()), &detail); err != nil {
		t.Fatal(err)
	}
	if detail["artifact_digest"] != "sha256:artifact" || detail["matrix_id"] != "matrix-1" {
		t.Fatalf("release detail lost immutable identity: %v", detail)
	}
}
