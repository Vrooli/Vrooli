package certification

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	validationv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation/validation_v1connect"
)

type validationReceiptReaderFunc func(context.Context, string) (*validationv1.ValidationReceipt, error)

func (f validationReceiptReaderFunc) GetValidation(ctx context.Context, id string) (*validationv1.ValidationReceipt, error) {
	return f(ctx, id)
}

func TestVerifyTestGenieReceiptRequiresOwnerLookupAndTerminalState(t *testing.T) {
	tests := []struct {
		name    string
		ref     string
		verdict Verdict
		reader  ValidationReceiptReader
		wantErr string
	}{
		{
			name: "missing reader", ref: "test-genie:r1", verdict: VerdictPassed,
			wantErr: "reader is required",
		},
		{
			name: "wrong owner ref", ref: "local:r1", verdict: VerdictPassed,
			reader: validationReceiptReaderFunc(func(context.Context, string) (*validationv1.ValidationReceipt, error) {
				t.Fatal("reader called for wrong owner")
				return nil, nil
			}),
			wantErr: "must use test-genie",
		},
		{
			name: "nonterminal", ref: "test-genie:r1", verdict: VerdictPassed,
			reader:  receiptReader("r1", validationv1.ReceiptState_RECEIPT_STATE_RUNNING),
			wantErr: "not terminal",
		},
		{
			name: "failed cannot support pass", ref: "test-genie:r1", verdict: VerdictPassed,
			reader:  receiptReader("r1", validationv1.ReceiptState_RECEIPT_STATE_FAILED),
			wantErr: "not succeeded",
		},
		{
			name: "source identity changed", ref: "test-genie:r1", verdict: VerdictPassed,
			reader: validationReceiptReaderFunc(func(context.Context, string) (*validationv1.ValidationReceipt, error) {
				return &validationv1.ValidationReceipt{
					ReceiptId: "r1", State: validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED,
					AdmittedIdentity: &validationv1.SourceIdentity{Identity: "before"},
					ObservedIdentity: &validationv1.SourceIdentity{Identity: "after"},
				}, nil
			}),
			wantErr: "stable admitted/observed source identity",
		},
		{
			name: "mismatched identity", ref: "test-genie:r1", verdict: VerdictFailed,
			reader: validationReceiptReaderFunc(func(_ context.Context, requested string) (*validationv1.ValidationReceipt, error) {
				if requested != "r1" {
					return nil, errors.New("unexpected requested id")
				}
				return &validationv1.ValidationReceipt{ReceiptId: "different", State: validationv1.ReceiptState_RECEIPT_STATE_FAILED}, nil
			}),
			wantErr: "identity does not match",
		},
		{
			name: "succeeded", ref: "test-genie:r1", verdict: VerdictPassed,
			reader: receiptReader("r1", validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED),
		},
		{
			name: "failed result retains failed owner receipt", ref: "test-genie:r1", verdict: VerdictFailed,
			reader: receiptReader("r1", validationv1.ReceiptState_RECEIPT_STATE_FAILED),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyTestGenieReceipt(context.Background(), tt.reader, tt.ref, tt.verdict)
			if tt.wantErr == "" && err != nil {
				t.Fatal(err)
			}
			if tt.wantErr != "" && (err == nil || !containsText(err.Error(), tt.wantErr)) {
				t.Fatalf("error = %v, want text %q", err, tt.wantErr)
			}
		})
	}
}

func TestAuthenticatedCollectorLeavesMissingOwnerReceiptPending(t *testing.T) {
	cell := mustMatrix(t).RequiredCells()[0]
	result := syntheticDriverResults(t, []CellID{cell})[0]
	duplicate := result
	duplicate.ValidationReceiptRef = "test-genie:missing"
	reader := validationReceiptReaderFunc(func(_ context.Context, id string) (*validationv1.ValidationReceipt, error) {
		if id == "missing" {
			return nil, errors.New("receipt not found")
		}
		return &validationv1.ValidationReceipt{
			ReceiptId: id, State: validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED,
			AdmittedIdentity: &validationv1.SourceIdentity{Identity: "source-v1"},
			ObservedIdentity: &validationv1.SourceIdentity{Identity: "source-v1"},
		}, nil
	})
	report, err := CollectAuthenticatedDriverResults(context.Background(), t.TempDir(), []DriverResult{result, duplicate}, reader)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Recorded) != 0 || len(report.Pending) != 532 || len(report.Issues) != 1 {
		t.Fatalf("missing owner receipt was not fail-closed: recorded=%d pending=%d issues=%d", len(report.Recorded), len(report.Pending), len(report.Issues))
	}
	if report.Issues[0].Cell != cell || !containsText(report.Issues[0].Error, "receipt not found") {
		t.Fatalf("unexpected authentication issue: %+v", report.Issues[0])
	}
}

func TestAuthenticatedCollectorReadsOneSharedTestGenieReceiptOnce(t *testing.T) {
	cells := mustMatrix(t).RequiredCells()[:2]
	results := syntheticDriverResults(t, cells)
	reads := 0
	reader := validationReceiptReaderFunc(func(_ context.Context, id string) (*validationv1.ValidationReceipt, error) {
		reads++
		return &validationv1.ValidationReceipt{
			ReceiptId: id, State: validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED,
			AdmittedIdentity: &validationv1.SourceIdentity{Identity: "source-v1"},
			ObservedIdentity: &validationv1.SourceIdentity{Identity: "source-v1"},
		}, nil
	})
	report, err := CollectAuthenticatedDriverResults(context.Background(), t.TempDir(), results, reader)
	if err != nil {
		t.Fatal(err)
	}
	if reads != 1 || len(report.Recorded) != 2 || len(report.Issues) != 0 {
		t.Fatalf("shared suite receipt reads=%d recorded=%d issues=%d", reads, len(report.Recorded), len(report.Issues))
	}
}

func TestConnectValidationReceiptReaderFetchesFromOwnerAPI(t *testing.T) {
	handler := &testGenieValidationHandler{}
	_, service := validationv1connect.NewValidationServiceHandler(handler)
	server := httptest.NewServer(service)
	defer server.Close()

	reader, err := NewConnectValidationReceiptReader(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := reader.GetValidation(context.Background(), "receipt-7")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.GetReceiptId() != "receipt-7" || receipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED {
		t.Fatalf("owner receipt = %s/%s", receipt.GetReceiptId(), receipt.GetState())
	}
	if handler.requested != "receipt-7" {
		t.Fatalf("owner API requested %q", handler.requested)
	}
}

func TestDiscoveredValidationReceiptReaderResolvesOwnerOnEveryRead(t *testing.T) {
	handler := &testGenieValidationHandler{}
	_, service := validationv1connect.NewValidationServiceHandler(handler)
	server := httptest.NewServer(service)
	defer server.Close()
	resolutions := 0
	reader := &DiscoveredValidationReceiptReader{
		HTTPClient: server.Client(),
		resolveURL: func(context.Context) (string, error) {
			resolutions++
			return server.URL, nil
		},
	}
	for _, id := range []string{"receipt-1", "receipt-2"} {
		if _, err := reader.GetValidation(context.Background(), id); err != nil {
			t.Fatal(err)
		}
	}
	if resolutions != 2 {
		t.Fatalf("service discovery calls = %d, want 2", resolutions)
	}
}

type testGenieValidationHandler struct {
	validationv1connect.UnimplementedValidationServiceHandler
	requested string
}

func (h *testGenieValidationHandler) GetValidation(_ context.Context, req *connect.Request[validationv1.GetValidationRequest]) (*connect.Response[validationv1.GetValidationResponse], error) {
	h.requested = req.Msg.GetReceiptId()
	return connect.NewResponse(&validationv1.GetValidationResponse{Receipt: &validationv1.ValidationReceipt{ReceiptId: h.requested, State: validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED}}), nil
}

func receiptReader(id string, state validationv1.ReceiptState) ValidationReceiptReader {
	return validationReceiptReaderFunc(func(_ context.Context, requested string) (*validationv1.ValidationReceipt, error) {
		if requested != id {
			return nil, errors.New("unexpected receipt id")
		}
		receipt := &validationv1.ValidationReceipt{ReceiptId: id, State: state}
		if state == validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED {
			receipt.AdmittedIdentity = &validationv1.SourceIdentity{Identity: "source-v1"}
			receipt.ObservedIdentity = &validationv1.SourceIdentity{Identity: "source-v1"}
		}
		return receipt, nil
	})
}

func containsText(value, fragment string) bool {
	for i := 0; i+len(fragment) <= len(value); i++ {
		if value[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
