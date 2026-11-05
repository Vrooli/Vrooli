package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-repository/v1/source"
)

type recordingSourceReader struct {
	item        *sourcev1.Distribution
	unavailable bool
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

func TestSourceDistributionListProjectsExactIdentity(t *testing.T) {
	fake := &recordingSourceReader{item: testDistribution()}
	srv := &Server{sourceDistributions: fake}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/source-distributions", nil)
	rr := httptest.NewRecorder()
	srv.sourceDistributionList(rr, req)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "sha256:closure") || !strings.Contains(rr.Body.String(), "decision-1") || !reflect.DeepEqual(fake.calls, []string{"list"}) {
		t.Fatalf("unexpected list response: code=%d body=%s calls=%v", rr.Code, rr.Body.String(), fake.calls)
	}
}

func TestSourceDistributionDetailKeepsApprovalPublicationAndDriftSeparate(t *testing.T) {
	fake := &recordingSourceReader{item: testDistribution()}
	srv := &Server{sourceDistributions: fake}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/source-distributions/distribution-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "distribution-1"})
	rr := httptest.NewRecorder()
	srv.sourceDistributionDetail(rr, req)
	body := rr.Body.String()
	for _, want := range []string{"awaiting_human", "decision-1", "destination_edits_detected", "private material"} {
		if !strings.Contains(body, want) {
			t.Fatalf("detail missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "TOKEN=") {
		t.Fatal("secret value leaked")
	}
}

func TestSourceDistributionUnavailableIsTypedAndNonMutating(t *testing.T) {
	fake := &recordingSourceReader{unavailable: true}
	srv := &Server{sourceDistributions: fake}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/source-distributions", nil)
	rr := httptest.NewRecorder()
	srv.sourceDistributionList(rr, req)
	if rr.Code != http.StatusServiceUnavailable || !strings.Contains(rr.Body.String(), `"available":false`) || !strings.Contains(rr.Body.String(), "owner offline") || !reflect.DeepEqual(fake.calls, []string{"list"}) {
		t.Fatalf("unexpected unavailable response: code=%d body=%s calls=%v", rr.Code, rr.Body.String(), fake.calls)
	}
}
