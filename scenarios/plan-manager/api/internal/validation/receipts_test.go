package validation

import (
	"reflect"
	"testing"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
)

func TestProjectReceiptPersistsTypedProviderChildIdentity(t *testing.T) {
	receipt := &validationv1.ValidationReceipt{
		ReceiptId: "receipt-1",
		Children:  []*validationv1.ChildOperation{{ChildId: "child-1"}},
	}
	projected := projectReceipt(ValidationOperation{}, receipt, "2026-09-06T00:00:00Z")
	if len(projected.Children) != 1 {
		t.Fatalf("children = %d, want 1", len(projected.Children))
	}
	check := projected.Children[0].Check
	if check.Kind != ValidationCheckCustom || check.SemanticKey != "test-genie-receipt-child:child-1" {
		t.Fatalf("check = %#v, want stable typed provider-child identity", check)
	}
}

func TestProjectReceiptDistinguishesDriftFromAssertionFailure(t *testing.T) {
	for _, tc := range []struct {
		reason validationv1.ValidationReasonCode
		want   Verdict
	}{
		{validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_IDENTITY_CHANGED, VerdictUnknown},
		{validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE, VerdictUnknown},
		{validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_FAILED, VerdictFail},
	} {
		r := &validationv1.ValidationReceipt{ReceiptId: "receipt", State: validationv1.ReceiptState_RECEIPT_STATE_FAILED, ReasonCode: tc.reason}
		got := projectReceipt(ValidationOperation{}, r, "2026-09-06T00:00:00Z")
		if got.Result.Verdict != tc.want {
			t.Fatalf("reason %v: got %v want %v", tc.reason, got.Result.Verdict, tc.want)
		}
		if r.State != validationv1.ReceiptState_RECEIPT_STATE_FAILED {
			t.Fatal("producer receipt was rewritten")
		}
	}
}

func TestProjectReceiptPreservesExistingChildIdentity(t *testing.T) {
	receipt := &validationv1.ValidationReceipt{
		ReceiptId: "receipt-1",
		Children:  []*validationv1.ChildOperation{{ChildId: "child-1"}},
	}
	existing := ValidationCheck{Kind: ValidationCheckScenarioDiff, SemanticKey: "scenario-diff:plan-manager:baseline"}
	projected := projectReceipt(ValidationOperation{Children: []ValidationChild{{ID: "child-1", Check: existing}}}, receipt, "2026-09-06T00:00:00Z")
	if !reflect.DeepEqual(projected.Children[0].Check, existing) {
		t.Fatalf("check = %#v, want existing identity %#v", projected.Children[0].Check, existing)
	}
}
