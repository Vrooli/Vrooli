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
