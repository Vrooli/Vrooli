package certification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	validationv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation/validation_v1connect"
)

type certificationArtifactOwnerFake struct {
	catalog *runspb.ListRunArtifactsResponse
	meta    *runspb.ArtifactRef
	body    []byte
	listed  string
	runID   string
	gotID   string
	fail    error
}

func (f *certificationArtifactOwnerFake) ListRunArtifacts(_ context.Context, scenario, runID string) (*runspb.ListRunArtifactsResponse, error) {
	f.listed, f.runID = scenario, runID
	if f.fail != nil {
		return nil, f.fail
	}
	return f.catalog, nil
}

func (f *certificationArtifactOwnerFake) GetRunArtifact(_ context.Context, scenario, runID, id string) (*runspb.ArtifactRef, error) {
	if scenario != f.listed || runID != f.runID || id != f.meta.GetId() {
		return nil, fmt.Errorf("unexpected artifact metadata request %s/%s/%s", scenario, runID, id)
	}
	f.gotID = id
	metadata := *f.meta
	metadata.AccessCapability = runspb.ArtifactAccessCapability_ARTIFACT_ACCESS_CAPABILITY_STREAM
	metadata.AccessPath = testGenieArtifactAccessPath(scenario, runID, id)
	return &metadata, f.fail
}

func (f *certificationArtifactOwnerFake) DownloadRunArtifact(_ context.Context, scenario, runID, id string, size int64) ([]byte, error) {
	if scenario != f.listed || runID != f.runID || id != f.meta.GetId() || size != f.meta.GetSizeBytes() {
		return nil, errors.New("download was not scoped to the catalog artifact")
	}
	return append([]byte(nil), f.body...), f.fail
}

type assertionValidationReader struct {
	receipt *validationv1.ValidationReceipt
	reads   int
}

func (r *assertionValidationReader) GetValidation(_ context.Context, id string) (*validationv1.ValidationReceipt, error) {
	r.reads++
	if r.receipt == nil || r.receipt.GetReceiptId() != id {
		return nil, errors.New("receipt missing")
	}
	return r.receipt, nil
}

func TestVerifyTestGenieAssertionArtifactAuthenticatesReceiptRunCatalogAndBytes(t *testing.T) {
	result, receiptReader, artifactOwner, expectedEnvelope := certificationAssertionFixture(t)
	proof, err := VerifyTestGenieAssertionArtifact(context.Background(), receiptReader, artifactOwner, result)
	if err != nil {
		t.Fatal(err)
	}
	if proof.Owner != "test-genie" || proof.ValidationReceiptID != "receipt-7" || proof.Scenario != "scenario-to-cloud" || proof.RunID != "run-77" || proof.ArtifactID != "opaque-artifact" {
		t.Fatalf("proof owner identity = %+v", proof)
	}
	if proof.Envelope != expectedEnvelope {
		t.Fatalf("envelope = %+v, want %+v", proof.Envelope, expectedEnvelope)
	}
	digest := sha256.Sum256(artifactOwner.body)
	wantSHA := hex.EncodeToString(digest[:])
	if proof.SHA256 != wantSHA || proof.AssertionArtifactRef != "assertion-artifact:"+result.DriverRef+":sha256:"+wantSHA {
		t.Fatalf("artifact digest/ref = %s/%s", proof.SHA256, proof.AssertionArtifactRef)
	}
	if !strings.Contains(proof.ArtifactRef, "test-genie-artifact://scenario-to-cloud/run-77/opaque-artifact?sha256="+wantSHA) {
		t.Fatalf("Test Genie artifact ref does not preserve owner/run/artifact identity: %s", proof.ArtifactRef)
	}
	if artifactOwner.listed != "scenario-to-cloud" || artifactOwner.runID != "run-77" || artifactOwner.gotID != "opaque-artifact" {
		t.Fatalf("catalog/metadata not scoped to receipt run: %+v", artifactOwner)
	}
	if receiptReader.reads != 2 {
		t.Fatalf("receipt owner reads = %d, want verify plus stable reread", receiptReader.reads)
	}
}

func TestVerifyTestGenieAssertionArtifactFailsClosedForWrongCellOrUnavailableCatalog(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*validationv1.ValidationReceipt, *runspb.ListRunArtifactsResponse, *runspb.ArtifactRef, []byte, DriverResult)
		wantErr string
	}{
		{
			name: "artifact envelope wrong ground",
			mutate: func(_ *validationv1.ValidationReceipt, _ *runspb.ListRunArtifactsResponse, _ *runspb.ArtifactRef, body []byte, result DriverResult) {
				var envelope TestGenieAssertionArtifact
				if err := json.Unmarshal(body, &envelope); err != nil {
					t.Fatal(err)
				}
				envelope.Cell.TargetGround = TargetGround(strings.Repeat("x", len(envelope.Cell.TargetGround)))
				changed, _ := json.Marshal(envelope)
				copy(body, changed)
			},
			wantErr: "bytes do not match",
		},
		{
			name: "legacy discovered catalog",
			mutate: func(_ *validationv1.ValidationReceipt, catalog *runspb.ListRunArtifactsResponse, _ *runspb.ArtifactRef, _ []byte, _ DriverResult) {
				catalog.LegacyDiscovered = true
			},
			wantErr: "degraded or legacy",
		},
		{
			name: "missing artifact",
			mutate: func(_ *validationv1.ValidationReceipt, catalog *runspb.ListRunArtifactsResponse, _ *runspb.ArtifactRef, _ []byte, _ DriverResult) {
				catalog.Artifacts = nil
			},
			wantErr: "has 0 declared assertion artifacts",
		},
		{
			name: "duplicate artifact",
			mutate: func(_ *validationv1.ValidationReceipt, catalog *runspb.ListRunArtifactsResponse, _ *runspb.ArtifactRef, _ []byte, _ DriverResult) {
				catalog.Artifacts = append(catalog.Artifacts, catalog.Artifacts[0])
			},
			wantErr: "has 2 declared assertion artifacts",
		},
		{
			name: "receipt cannot name another run",
			mutate: func(receipt *validationv1.ValidationReceipt, _ *runspb.ListRunArtifactsResponse, _ *runspb.ArtifactRef, _ []byte, _ DriverResult) {
				receipt.Evidence[0].EvidenceId = "different-run"
			},
			wantErr: "malformed run evidence",
		},
		{
			name: "missing ground metadata",
			mutate: func(_ *validationv1.ValidationReceipt, catalog *runspb.ListRunArtifactsResponse, _ *runspb.ArtifactRef, _ []byte, _ DriverResult) {
				catalog.Artifacts[0].Metadata["target_ground"] = ""
			},
			wantErr: "has 0 declared assertion artifacts",
		},
		{
			name: "artifact declared by another lane owner",
			mutate: func(_ *validationv1.ValidationReceipt, catalog *runspb.ListRunArtifactsResponse, _ *runspb.ArtifactRef, _ []byte, _ DriverResult) {
				catalog.Artifacts[0].Metadata["assertion_owner"] = "different-owner"
			},
			wantErr: "has 0 declared assertion artifacts",
		},
		{
			name: "missing underlying owner receipt",
			mutate: func(_ *validationv1.ValidationReceipt, catalog *runspb.ListRunArtifactsResponse, _ *runspb.ArtifactRef, _ []byte, _ DriverResult) {
				catalog.Artifacts[0].Metadata["owner_receipt_ref"] = ""
			},
			wantErr: "has 0 declared assertion artifacts",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, reader, owner, _ := certificationAssertionFixture(t)
			tt.mutate(reader.receipt, owner.catalog, owner.meta, owner.body, result)
			_, err := VerifyTestGenieAssertionArtifact(context.Background(), reader, owner, result)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestVerifyTestGenieAssertionArtifactRejectsBytesOutsideMetadataSize(t *testing.T) {
	result, reader, owner, _ := certificationAssertionFixture(t)
	owner.body = append(owner.body, ' ')
	if _, err := VerifyTestGenieAssertionArtifact(context.Background(), reader, owner, result); err == nil || !strings.Contains(err.Error(), "does not match catalog size") {
		t.Fatalf("unexpected artifact byte count accepted: %v", err)
	}
}

func TestDiscoveredTestGenieArtifactOwnerUsesTypedCatalogMetadataAndRunScopedTransfer(t *testing.T) {
	result, receiptReader, fakeOwner, _ := certificationAssertionFixture(t)
	transferPath := testGenieArtifactAccessPath("scenario-to-cloud", "run-77", "opaque-artifact")
	mux := http.NewServeMux()
	validationHandler := &validationOwnerHandler{receipt: receiptReader.receipt}
	mux.Handle(validationv1connect.ValidationServiceGetValidationProcedure, connect.NewUnaryHandler(validationv1connect.ValidationServiceGetValidationProcedure, validationHandler.GetValidation))
	owner := &runsOwnerHandler{catalog: fakeOwner.catalog, artifact: fakeOwner.meta}
	mux.Handle(runsconnect.RunsServiceListRunArtifactsProcedure, connect.NewUnaryHandler(runsconnect.RunsServiceListRunArtifactsProcedure, owner.ListRunArtifacts))
	mux.Handle(runsconnect.RunsServiceGetRunArtifactProcedure, connect.NewUnaryHandler(runsconnect.RunsServiceGetRunArtifactProcedure, owner.GetRunArtifact))
	mux.HandleFunc(transferPath, func(w http.ResponseWriter, req *http.Request) {
		owner.transferCalls++
		if req.Method != http.MethodGet {
			t.Errorf("artifact transfer method = %s", req.Method)
		}
		_, _ = w.Write(fakeOwner.body)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	reader, err := NewConnectValidationReceiptReader(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	ownerReader := &DiscoveredTestGenieArtifactOwner{
		HTTPClient: server.Client(),
		resolveURL: func(context.Context) (string, error) { return server.URL, nil },
	}
	if _, err := VerifyTestGenieAssertionArtifact(context.Background(), reader, ownerReader, result); err != nil {
		t.Fatal(err)
	}
	if owner.listCalls != 1 || owner.getCalls != 2 || owner.transferCalls != 1 {
		t.Fatalf("Test Genie owner API calls: catalog=%d metadata=%d transfer=%d", owner.listCalls, owner.getCalls, owner.transferCalls)
	}
}

type validationOwnerHandler struct {
	validationv1connect.UnimplementedValidationServiceHandler
	receipt *validationv1.ValidationReceipt
}

func (h *validationOwnerHandler) GetValidation(_ context.Context, req *connect.Request[validationv1.GetValidationRequest]) (*connect.Response[validationv1.GetValidationResponse], error) {
	if req.Msg.GetReceiptId() != h.receipt.GetReceiptId() {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("receipt missing"))
	}
	return connect.NewResponse(&validationv1.GetValidationResponse{Receipt: h.receipt}), nil
}

type runsOwnerHandler struct {
	runsconnect.UnimplementedRunsServiceHandler
	catalog       *runspb.ListRunArtifactsResponse
	artifact      *runspb.ArtifactRef
	listCalls     int
	getCalls      int
	transferCalls int
}

func (h *runsOwnerHandler) ListRunArtifacts(_ context.Context, req *connect.Request[runspb.ListRunArtifactsRequest]) (*connect.Response[runspb.ListRunArtifactsResponse], error) {
	h.listCalls++
	if req.Msg.GetTarget() != "scenario-to-cloud" || req.Msg.GetRunId() != "run-77" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unexpected run scope"))
	}
	return connect.NewResponse(h.catalog), nil
}

func (h *runsOwnerHandler) GetRunArtifact(_ context.Context, req *connect.Request[runspb.GetRunArtifactRequest]) (*connect.Response[runspb.GetRunArtifactResponse], error) {
	h.getCalls++
	if req.Msg.GetTarget() != "scenario-to-cloud" || req.Msg.GetRunId() != "run-77" || req.Msg.GetArtifactId() != "opaque-artifact" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unexpected artifact scope"))
	}
	metadata := *h.artifact
	metadata.AccessCapability = runspb.ArtifactAccessCapability_ARTIFACT_ACCESS_CAPABILITY_STREAM
	metadata.AccessPath = testGenieArtifactAccessPath("scenario-to-cloud", "run-77", "opaque-artifact")
	return connect.NewResponse(&runspb.GetRunArtifactResponse{Artifact: &metadata}), nil
}

func certificationAssertionFixture(t *testing.T) (DriverResult, *assertionValidationReader, *certificationArtifactOwnerFake, TestGenieAssertionArtifact) {
	t.Helper()
	matrix := mustMatrix(t)
	return certificationAssertionFixtureForCell(t, matrix.RequiredCells()[0])
}

func certificationAssertionFixtureForCell(t *testing.T, cell CellID) (DriverResult, *assertionValidationReader, *certificationArtifactOwnerFake, TestGenieAssertionArtifact) {
	t.Helper()
	matrix := mustMatrix(t)
	caseRow, ok := matrix.Case(cell.CaseID)
	if !ok {
		t.Fatalf("matrix case missing: %s", cell.CaseID)
	}
	driverRef, err := driverRefForMatrix(matrix, cell)
	if err != nil {
		t.Fatal(err)
	}
	assertionOwner, _ := laneEvidenceOwner(cell.Lane)
	ownerReceiptRef := "test-genie:receipt-7"
	if cell.Lane == LaneRelay {
		ownerReceiptRef = "bridge-run://run-77/assertion.json"
	}
	if cell.Lane == LaneQEMU || cell.Lane == LaneRealVPS {
		ownerReceiptRef = "target-receipt://op-77/apply?fence=1"
	}
	if cell.Lane == LaneBASUI {
		ownerReceiptRef = "bas-execution://00000000-0000-4000-8000-000000000077/00000000-0000-4000-8000-000000000078"
	}
	if cell.Lane == LaneIndependentReview {
		ownerReceiptRef = "reviewer-attestation://review-77"
	}
	if cell.Lane == LaneSoak {
		ownerReceiptRef = "soak-run://run-77"
	}
	result := DriverResult{
		Cell: cell, DriverRef: driverRef, Verdict: VerdictPassed,
		Candidate:            Candidate{ReleaseDigest: "sha256:" + strings.Repeat("a", 64), ConfigurationDigest: "sha256:" + strings.Repeat("b", 64), ClosureDigest: "sha256:" + strings.Repeat("c", 64)},
		Target:               Target{Ground: cell.TargetGround, MachineID: "machine-7", Generation: 3, Architecture: "amd64", OS: "linux"},
		ValidationReceiptRef: "test-genie:receipt-7", Assertion: caseRow.Assertion,
	}
	envelope := TestGenieAssertionArtifact{
		SchemaVersion: 1, ValidationReceiptID: "receipt-7", RunID: "run-77", Scenario: "scenario-to-cloud",
		Cell: cell, DriverRef: driverRef, AssertionOwner: assertionOwner, OwnerReceiptRef: ownerReceiptRef, Verdict: VerdictPassed, Assertion: caseRow.Assertion,
		Candidate: result.Candidate, Target: result.Target,
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	artifact := &runspb.ArtifactRef{
		Id: "opaque-artifact", Kind: certificationAssertionArtifactKind, SizeBytes: int64(len(body)),
		Provenance: runspb.ArtifactProvenance_ARTIFACT_PROVENANCE_CATALOG,
		Metadata: map[string]string{
			"validation_receipt_id": "receipt-7", "case_id": cell.CaseID, "lane": string(cell.Lane),
			"target_ground": string(cell.TargetGround), "driver_ref": driverRef, "assertion_owner": assertionOwner, "owner_receipt_ref": ownerReceiptRef,
		},
	}
	catalog := &runspb.ListRunArtifactsResponse{
		SchemaVersion: 1, Digest: "ac:sha256:" + strings.Repeat("d", 64), Artifacts: []*runspb.ArtifactRef{artifact},
	}
	reader := &assertionValidationReader{receipt: &validationv1.ValidationReceipt{
		ReceiptId: "receipt-7", State: validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED,
		AdmittedIdentity: &validationv1.SourceIdentity{Identity: "source-v1"},
		ObservedIdentity: &validationv1.SourceIdentity{Identity: "source-v1"},
		Evidence: []*validationv1.EvidenceReference{{
			EvidenceId: "run-77", Kind: "test-genie-run", Owner: "test-genie", SubjectId: "scenario-to-cloud",
			Uri: "test-genie://runs/scenario-to-cloud/run-77",
		}},
	}}
	owner := &certificationArtifactOwnerFake{catalog: catalog, meta: artifact, body: body}
	return result, reader, owner, envelope
}

var _ validationv1connect.ValidationServiceHandler = (*validationOwnerHandler)(nil)
var _ runsconnect.RunsServiceHandler = (*runsOwnerHandler)(nil)
