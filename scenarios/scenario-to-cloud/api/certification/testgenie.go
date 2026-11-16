package certification

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	validationv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation/validation_v1connect"
)

// ValidationReceiptReader reads a durable receipt from Test Genie's owner API.
// Implementations must not synthesize receipts from local run state.
type ValidationReceiptReader interface {
	GetValidation(context.Context, string) (*validationv1.ValidationReceipt, error)
}

// ConnectValidationReceiptReader is the production Test Genie receipt adapter.
// The API URL and HTTP client are supplied by the scenario's existing service
// configuration so credentials and transport policy stay with the caller.
type ConnectValidationReceiptReader struct {
	Client validationv1connect.ValidationServiceClient
}

// DiscoveredValidationReceiptReader resolves Test Genie through the shared
// service registry on every read, so address changes and service restarts do
// not leave certification pinned to a stale endpoint.
type DiscoveredValidationReceiptReader struct {
	HTTPClient *http.Client
	resolveURL func(context.Context) (string, error)
}

func NewDiscoveredValidationReceiptReader(timeout time.Duration) *DiscoveredValidationReceiptReader {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &DiscoveredValidationReceiptReader{
		HTTPClient: &http.Client{Timeout: timeout},
		resolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "test-genie")
		},
	}
}

func (r *DiscoveredValidationReceiptReader) GetValidation(ctx context.Context, receiptID string) (*validationv1.ValidationReceipt, error) {
	if r == nil || r.resolveURL == nil {
		return nil, fmt.Errorf("Test Genie service discovery is unavailable")
	}
	apiURL, err := r.resolveURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve Test Genie owner: %w", err)
	}
	reader, err := NewConnectValidationReceiptReader(apiURL, r.HTTPClient)
	if err != nil {
		return nil, err
	}
	return reader.GetValidation(ctx, receiptID)
}

func NewConnectValidationReceiptReader(apiURL string, client *http.Client) (*ConnectValidationReceiptReader, error) {
	apiURL = strings.TrimRight(strings.TrimSpace(apiURL), "/")
	if apiURL == "" {
		return nil, fmt.Errorf("Test Genie API URL is required")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &ConnectValidationReceiptReader{Client: validationv1connect.NewValidationServiceClient(client, apiURL)}, nil
}

func (r *ConnectValidationReceiptReader) GetValidation(ctx context.Context, receiptID string) (*validationv1.ValidationReceipt, error) {
	if r == nil || r.Client == nil {
		return nil, fmt.Errorf("Test Genie validation receipt client is unavailable")
	}
	response, err := r.Client.GetValidation(ctx, connect.NewRequest(&validationv1.GetValidationRequest{ReceiptId: receiptID}))
	if err != nil {
		return nil, fmt.Errorf("read Test Genie validation receipt %q: %w", receiptID, err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetReceipt() == nil {
		return nil, fmt.Errorf("Test Genie returned no validation receipt for %q", receiptID)
	}
	return response.Msg.GetReceipt(), nil
}

// VerifyTestGenieReceipt confirms the ref resolves to a real, terminal owner
// receipt. Passing results require SUCCEEDED; non-passing outcomes may retain
// any terminal Test Genie state as failure evidence. This validates only the
// Test Genie provenance edge, not target-owner assertions or artifact bytes.
func VerifyTestGenieReceipt(ctx context.Context, reader ValidationReceiptReader, ref string, verdict Verdict) error {
	receiptID, err := validateTestGenieReceiptRequest(reader, ref)
	if err != nil {
		return err
	}
	receipt, err := reader.GetValidation(ctx, receiptID)
	if err != nil {
		return err
	}
	if err := validateTestGenieReceiptIdentity(receipt, receiptID); err != nil {
		return err
	}
	return validateTestGenieReceiptOutcome(receipt, receiptID, verdict)
}

func validateTestGenieReceiptRequest(reader ValidationReceiptReader, ref string) (string, error) {
	if reader == nil {
		return "", fmt.Errorf("Test Genie receipt reader is required")
	}
	if !strings.HasPrefix(ref, "test-genie:") {
		return "", fmt.Errorf("validation receipt ref must use test-genie:<receipt-id>")
	}
	receiptID := strings.TrimSpace(strings.TrimPrefix(ref, "test-genie:"))
	if receiptID == "" || strings.ContainsAny(receiptID, "/?#") {
		return "", fmt.Errorf("Test Genie receipt ref has an invalid receipt id")
	}
	return receiptID, nil
}

func validateTestGenieReceiptIdentity(receipt *validationv1.ValidationReceipt, receiptID string) error {
	if receipt == nil || receipt.GetReceiptId() != receiptID {
		return fmt.Errorf("Test Genie receipt identity does not match %q", receiptID)
	}
	if !isTerminalTestGenieReceiptState(receipt.GetState()) {
		return fmt.Errorf("Test Genie receipt %q is not terminal", receiptID)
	}
	return nil
}

func isTerminalTestGenieReceiptState(state validationv1.ReceiptState) bool {
	switch state {
	case validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED,
		validationv1.ReceiptState_RECEIPT_STATE_FAILED,
		validationv1.ReceiptState_RECEIPT_STATE_DEGRADED,
		validationv1.ReceiptState_RECEIPT_STATE_CANCELLED,
		validationv1.ReceiptState_RECEIPT_STATE_SUPERSEDED:
		return true
	default:
		return false
	}
}

func validateTestGenieReceiptOutcome(receipt *validationv1.ValidationReceipt, receiptID string, verdict Verdict) error {
	state := receipt.GetState()
	if err := validateTestGenieReceiptVerdict(state, receiptID, verdict); err != nil {
		return err
	}
	if verdict == VerdictPassed {
		return validateTestGenieAdmittedIdentity(receipt, receiptID)
	}
	return nil
}

func validateTestGenieReceiptVerdict(state validationv1.ReceiptState, receiptID string, verdict Verdict) error {
	if verdict == VerdictPassed && state != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED {
		return fmt.Errorf("Test Genie receipt %q is %s, not succeeded", receiptID, state)
	}
	return nil
}

func validateTestGenieAdmittedIdentity(receipt *validationv1.ValidationReceipt, receiptID string) error {
	admitted, observed := receipt.GetAdmittedIdentity(), receipt.GetObservedIdentity()
	if admitted == nil || observed == nil || strings.TrimSpace(admitted.GetIdentity()) == "" || admitted.GetIdentity() != observed.GetIdentity() {
		return fmt.Errorf("Test Genie receipt %q lacks one stable admitted/observed source identity", receiptID)
	}
	return nil
}

// CollectAuthenticatedDriverResults verifies every supplied Test Genie owner
// receipt before delegating to the fail-closed append-only collector. This
// proves only the Test Genie edge. Passed results still remain pending until
// target-owner receipts and assertion-artifact bytes are read and checked by
// their lane owners; this function never upgrades that missing proof.
func CollectAuthenticatedDriverResults(ctx context.Context, evidenceDir string, results []DriverResult, reader ValidationReceiptReader) (CollectionReport, error) {
	verified := make([]DriverResult, 0, len(results))
	var authIssues []CollectionIssue
	authFailed := make(map[CellID]bool)
	verificationCache := make(map[string]error)
	for _, result := range results {
		cacheKey := result.ValidationReceiptRef + "\x00" + string(result.Verdict)
		verifyErr, checked := verificationCache[cacheKey]
		if !checked {
			verifyErr = VerifyTestGenieReceipt(ctx, reader, result.ValidationReceiptRef, result.Verdict)
			verificationCache[cacheKey] = verifyErr
		}
		if verifyErr != nil {
			authIssues = append(authIssues, CollectionIssue{Cell: result.Cell, Error: verifyErr.Error()})
			authFailed[result.Cell] = true
			continue
		}
		verified = append(verified, result)
	}
	if len(authFailed) > 0 {
		eligible := verified[:0]
		for _, result := range verified {
			if !authFailed[result.Cell] {
				eligible = append(eligible, result)
			}
		}
		verified = eligible
	}
	report, err := CollectDriverResults(evidenceDir, verified)
	if err != nil {
		return CollectionReport{}, err
	}
	report.Issues = append(report.Issues, authIssues...)
	sort.Slice(report.Issues, func(i, j int) bool {
		if report.Issues[i].Cell != report.Issues[j].Cell {
			return report.Issues[i].Cell.String() < report.Issues[j].Cell.String()
		}
		return report.Issues[i].Error < report.Issues[j].Error
	})
	return report, nil
}
