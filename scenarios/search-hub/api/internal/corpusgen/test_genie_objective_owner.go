package corpusgen

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation/validation_v1connect"
)

// TestGenieRepositoryTestEvidenceReader exposes the body-free durable receipt
// projection owned by Test Genie.
type TestGenieRepositoryTestEvidenceReader interface {
	GetRepositoryTestEvidence(context.Context, string) (*validationv1.GetRepositoryTestEvidenceResponse, error)
}

// TestGenieObjectiveSuiteVerifier accepts only a successful durable Test Genie
// receipt bound to the exact sealed repository tree and fixed suite.
type TestGenieObjectiveSuiteVerifier struct {
	Reader TestGenieRepositoryTestEvidenceReader
}

func (v TestGenieObjectiveSuiteVerifier) VerifyCodeTaskObjectiveSuite(ctx context.Context, repo CodeTaskRepositoryBase, suite CodeTaskObjectiveSuite) error {
	if v.Reader == nil {
		return fmt.Errorf("Test Genie repository-test evidence owner is unavailable")
	}
	if suite.ValidationReceiptID == "" || !validCodeTaskSHA256(suite.ValidationReceiptSHA256) {
		return fmt.Errorf("Test Genie validation receipt identity is incomplete")
	}
	evidence, err := v.Reader.GetRepositoryTestEvidence(ctx, suite.ValidationReceiptID)
	if err != nil {
		return fmt.Errorf("read Test Genie objective suite receipt: %w", err)
	}
	if evidence == nil || evidence.GetReceiptId() != suite.ValidationReceiptID ||
		canonicalReceiptSHA(evidence.GetReceiptDigest()) != suite.ValidationReceiptSHA256 ||
		!validPrefixedReceiptSHA(evidence.GetIntentDigest()) ||
		evidence.GetReceiptState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED ||
		evidence.GetOutcome() != validationv1.RepositoryTestOutcome_REPOSITORY_TEST_OUTCOME_PASSED ||
		evidence.GetAdmittedSourceTree() != repo.TreeSHA256 || evidence.GetObservedSourceTree() != repo.TreeSHA256 ||
		evidence.GetSuiteId() != suite.SuiteID || evidence.GetSuiteDigest() != suite.SHA256 ||
		evidence.GetWorkspaceSandboxOperationId() == "" ||
		!validCodeTaskSHA256(evidence.GetResultDigest()) || !validCodeTaskSHA256(evidence.GetCleanupDigest()) ||
		!evidence.GetSourceIntactBefore() || !evidence.GetSourceIntactAfter() || !evidence.GetCleanupComplete() {
		return fmt.Errorf("Test Genie receipt is not a successful, cleaned objective suite on the exact repository base")
	}
	return nil
}

// DiscoveredTestGenieRepositoryTestEvidenceReader reads only the narrow
// owner-produced evidence projection. It never retrieves receipt bodies or
// test output into Search Hub.
type DiscoveredTestGenieRepositoryTestEvidenceReader struct {
	HTTPClient *http.Client
	ResolveURL func(context.Context, string) (string, error)
}

func (r DiscoveredTestGenieRepositoryTestEvidenceReader) GetRepositoryTestEvidence(ctx context.Context, receiptID string) (*validationv1.GetRepositoryTestEvidenceResponse, error) {
	receiptID = strings.TrimSpace(receiptID)
	if receiptID == "" {
		return nil, fmt.Errorf("Test Genie receipt ID is required")
	}
	resolve := r.ResolveURL
	if resolve == nil {
		resolve = discovery.ResolveScenarioURLDefault
	}
	baseURL, err := resolve(ctx, "test-genie")
	if err != nil {
		return nil, fmt.Errorf("resolve Test Genie evidence owner: %w", err)
	}
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("Test Genie discovery returned an empty URL")
	}
	client := r.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	owner := validationconnect.NewValidationServiceClient(client, strings.TrimRight(baseURL, "/"))
	response, err := owner.GetRepositoryTestEvidence(ctx, connect.NewRequest(&validationv1.GetRepositoryTestEvidenceRequest{ReceiptId: receiptID}))
	if err != nil {
		return nil, fmt.Errorf("read Test Genie repository evidence %q: %w", receiptID, err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetReceiptId() != receiptID {
		return nil, fmt.Errorf("Test Genie returned a mismatched repository receipt identity")
	}
	return response.Msg, nil
}

func canonicalReceiptSHA(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "sha256:")
}

func validPrefixedReceiptSHA(value string) bool {
	return strings.HasPrefix(value, "sha256:") && validCodeTaskSHA256(strings.TrimPrefix(value, "sha256:"))
}
