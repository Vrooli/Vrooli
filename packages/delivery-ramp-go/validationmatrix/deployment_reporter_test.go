package validationmatrix

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type deploymentReporterRoundTripper func(*http.Request) (*http.Response, error)

func (f deploymentReporterRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDeploymentReporterServiceAuthTransportAddsConfiguredToken(t *testing.T) {
	t.Setenv("DEPLOYMENT_MANAGER_SERVICE_TOKEN", "ramp-token")
	var authorization string
	transport := deploymentManagerServiceAuthTransport{base: deploymentReporterRoundTripper(func(req *http.Request) (*http.Response, error) {
		authorization = req.Header.Get("Authorization")
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: req}, nil
	})}
	request := &http.Request{Method: http.MethodPost, URL: mustDeploymentReporterURL(t, "https://deployment-manager.test"), Header: make(http.Header)}
	if _, err := transport.RoundTrip(request); err != nil {
		t.Fatal(err)
	}
	if authorization != "Bearer ramp-token" {
		t.Fatalf("authorization=%q, want bearer token", authorization)
	}
}

func TestDeploymentReporterServiceAuthTransportDoesNotInventToken(t *testing.T) {
	t.Setenv("DEPLOYMENT_MANAGER_SERVICE_TOKEN", "")
	t.Setenv("DEPLOYMENT_MANAGER_SERVICE_TOKEN_FILE", "")
	var authorization string
	transport := deploymentManagerServiceAuthTransport{base: deploymentReporterRoundTripper(func(req *http.Request) (*http.Response, error) {
		authorization = req.Header.Get("Authorization")
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: req}, nil
	})}
	request := &http.Request{Method: http.MethodPost, URL: mustDeploymentReporterURL(t, "https://deployment-manager.test"), Header: make(http.Header)}
	if _, err := transport.RoundTrip(request); err != nil {
		t.Fatal(err)
	}
	if authorization != "" {
		t.Fatalf("authorization=%q, want empty", authorization)
	}
}

func mustDeploymentReporterURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	value, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

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
	WithDeploymentIdentity("scenario-to-android", "scenario-to-android", "android", "android")(reporter)
	gate := &domainv1.ReleaseGate{MatrixId: "matrix-1", Passed: true, Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, RequiredCellCount: 1, PassingCellCount: 1}
	verdict := ReleaseVerdict{RunID: "run-1", MatrixID: "matrix-1", ScenarioName: "demo", ArtifactDigest: "sha256:artifact", ArtifactRef: "android-debug:artifact", ProfileID: "profile-specific", GitCommit: "commit-specific", Targets: []TargetSelection{{Kind: TargetBridge, Descriptor: &domainv1.ValidationTargetDescriptor{TargetId: "android-physical-1", DisplayName: "Moto", Available: true}}}, Gate: gate, Evidence: validEvidence()}
	if err := reporter.ReportValidationGate(context.Background(), verdict); err != nil {
		t.Fatal(err)
	}
	if client.request.GetProfileId() != "profile-specific" || client.request.GetGitCommitHash() != "commit-specific" {
		t.Fatalf("release identity lost: %+v", client.request)
	}
	if client.request.GetVerdict().GetDisposition() != commonv1.Disposition_DISPOSITION_PASSED || client.request.GetVerdict().GetEvidenceClass() != "release-grade" || client.request.GetVerdict().GetRunId() != "run-1" || len(client.request.GetVerdict().GetRefs()) != 5 {
		t.Fatalf("release verdict lost evidence: %+v", client.request.GetVerdict())
	}
	if client.request.GetVerdict().GetTarget().GetDeviceKind() != commonv1.DeviceKind_DEVICE_KIND_PHYSICAL || client.request.GetVerdict().GetTarget().GetOs() != "android" {
		t.Fatalf("Android target identity was misclassified: %+v", client.request.GetVerdict().GetTarget())
	}
	for _, ref := range client.request.GetVerdict().GetRefs() {
		if ref.GetCreatedAt() == nil || !ref.GetCreatedAt().IsValid() {
			t.Fatalf("release evidence reference lacks a valid creation time: %+v", ref)
		}
	}
	var detail map[string]any
	if err := json.Unmarshal([]byte(client.request.GetVerdict().GetDetail()), &detail); err != nil {
		t.Fatal(err)
	}
	if detail["artifact_digest"] != "sha256:artifact" || detail["artifact_ref"] != "android-debug:artifact" || detail["matrix_id"] != "matrix-1" || detail["target_id"] != "android-physical-1" {
		t.Fatalf("release detail lost immutable identity: %v", detail)
	}
}
