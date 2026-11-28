package corpusgen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation/validation_v1connect"
	"google.golang.org/protobuf/proto"
)

type testGenieRepositoryEvidenceFixture struct {
	validationconnect.UnimplementedValidationServiceHandler
	evidence *validationv1.GetRepositoryTestEvidenceResponse
	received string
}

func (f *testGenieRepositoryEvidenceFixture) GetRepositoryTestEvidence(_ context.Context, req *connect.Request[validationv1.GetRepositoryTestEvidenceRequest]) (*connect.Response[validationv1.GetRepositoryTestEvidenceResponse], error) {
	f.received = req.Msg.GetReceiptId()
	response := proto.Clone(f.evidence).(*validationv1.GetRepositoryTestEvidenceResponse)
	return connect.NewResponse(response), nil
}

func TestTestGenieObjectiveSuiteVerifierBindsDurablePassedReceiptToExactTreeAndSuite(t *testing.T) {
	treeSHA := strings.Repeat("a", 64)
	suiteSHA := strings.Repeat("b", 64)
	receiptSHA := strings.Repeat("c", 64)
	evidence := &validationv1.GetRepositoryTestEvidenceResponse{
		ReceiptId: "tg-receipt-1", IntentDigest: "sha256:" + strings.Repeat("d", 64), ReceiptDigest: "sha256:" + receiptSHA,
		AdmittedSourceTree: treeSHA, ObservedSourceTree: treeSHA, SuiteId: "go-test-all-v1", SuiteDigest: suiteSHA,
		Outcome: validationv1.RepositoryTestOutcome_REPOSITORY_TEST_OUTCOME_PASSED, WorkspaceSandboxOperationId: "sandbox-test-1",
		ResultDigest: strings.Repeat("e", 64), CleanupDigest: strings.Repeat("f", 64), SourceIntactBefore: true, SourceIntactAfter: true,
		CleanupComplete: true, ReceiptState: validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED,
	}
	fixture := &testGenieRepositoryEvidenceFixture{evidence: evidence}
	path, handler := validationconnect.NewValidationServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	reader := DiscoveredTestGenieRepositoryTestEvidenceReader{HTTPClient: server.Client(), ResolveURL: func(context.Context, string) (string, error) { return server.URL, nil }}
	verifier := TestGenieObjectiveSuiteVerifier{Reader: reader}
	repository := CodeTaskRepositoryBase{RepositoryID: "fixture", Revision: strings.Repeat("1", 40), TreeSHA256: treeSHA}
	suite := CodeTaskObjectiveSuite{SuiteID: evidence.GetSuiteId(), Revision: strings.Repeat("1", 40), SHA256: suiteSHA, ValidationReceiptID: evidence.GetReceiptId(), ValidationReceiptSHA256: receiptSHA}
	if err := verifier.VerifyCodeTaskObjectiveSuite(context.Background(), repository, suite); err != nil {
		t.Fatalf("valid durable Test Genie receipt rejected: %v", err)
	}
	if fixture.received != suite.ValidationReceiptID {
		t.Fatalf("Test Genie owner request receipt=%q, want %q", fixture.received, suite.ValidationReceiptID)
	}

	for _, tc := range []struct {
		name   string
		mutate func(*validationv1.GetRepositoryTestEvidenceResponse, *CodeTaskRepositoryBase, *CodeTaskObjectiveSuite)
	}{
		{name: "different receipt digest", mutate: func(_ *validationv1.GetRepositoryTestEvidenceResponse, _ *CodeTaskRepositoryBase, suite *CodeTaskObjectiveSuite) {
			suite.ValidationReceiptSHA256 = strings.Repeat("0", 64)
		}},
		{name: "different tree", mutate: func(_ *validationv1.GetRepositoryTestEvidenceResponse, repository *CodeTaskRepositoryBase, _ *CodeTaskObjectiveSuite) {
			repository.TreeSHA256 = strings.Repeat("0", 64)
		}},
		{name: "different suite", mutate: func(_ *validationv1.GetRepositoryTestEvidenceResponse, _ *CodeTaskRepositoryBase, suite *CodeTaskObjectiveSuite) {
			suite.SHA256 = strings.Repeat("0", 64)
		}},
		{name: "failed suite", mutate: func(e *validationv1.GetRepositoryTestEvidenceResponse, _ *CodeTaskRepositoryBase, _ *CodeTaskObjectiveSuite) {
			e.Outcome = validationv1.RepositoryTestOutcome_REPOSITORY_TEST_OUTCOME_FAILED
		}},
		{name: "unclean result", mutate: func(e *validationv1.GetRepositoryTestEvidenceResponse, _ *CodeTaskRepositoryBase, _ *CodeTaskObjectiveSuite) {
			e.CleanupComplete = false
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changedEvidence := proto.Clone(evidence).(*validationv1.GetRepositoryTestEvidenceResponse)
			changedRepo := repository
			changedSuite := suite
			tc.mutate(changedEvidence, &changedRepo, &changedSuite)
			changedFixture := &testGenieRepositoryEvidenceFixture{evidence: changedEvidence}
			changedPath, changedHandler := validationconnect.NewValidationServiceHandler(changedFixture)
			changedMux := http.NewServeMux()
			changedMux.Handle(changedPath, changedHandler)
			changedServer := httptest.NewServer(changedMux)
			defer changedServer.Close()
			changedReader := DiscoveredTestGenieRepositoryTestEvidenceReader{HTTPClient: changedServer.Client(), ResolveURL: func(context.Context, string) (string, error) { return changedServer.URL, nil }}
			if err := (TestGenieObjectiveSuiteVerifier{Reader: changedReader}).VerifyCodeTaskObjectiveSuite(context.Background(), changedRepo, changedSuite); err == nil {
				t.Fatal("mismatched or failed Test Genie objective receipt was accepted")
			}
		})
	}
}
