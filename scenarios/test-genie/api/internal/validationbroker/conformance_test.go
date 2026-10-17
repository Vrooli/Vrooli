package validationbroker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestReceiptStateJSONConformanceCoversEveryState(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	var fixture struct {
		Receipts []json.RawMessage `json:"receipts"`
	}
	readJSONFixture(t, "receipt_states.json", &fixture)
	covered := map[validationv1.ReceiptState]bool{}
	for _, raw := range fixture.Receipts {
		var receipt validationv1.ValidationReceipt
		if err := protojson.Unmarshal(raw, &receipt); err != nil {
			t.Fatalf("decode receipt fixture: %v", err)
		}
		if receipt.GetSchemaVersion() != ReceiptSchemaVersion || receipt.GetReceiptId() == "" || receipt.GetLineageId() == "" || receipt.GetReasonCode() == validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_UNSPECIFIED {
			t.Fatalf("incomplete receipt fixture: %#v", &receipt)
		}
		covered[receipt.GetState()] = true
	}
	for number := int32(1); number <= int32(validationv1.ReceiptState_RECEIPT_STATE_SUPERSEDED); number++ {
		state := validationv1.ReceiptState(number)
		if !covered[state] {
			t.Errorf("receipt state %s has no conformance fixture", state)
		}
	}
}

func TestMalformedIntentJSONFixturesFailClosed(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	var fixture struct {
		Intents []json.RawMessage `json:"intents"`
	}
	readJSONFixture(t, "malformed_intents.json", &fixture)
	for index, raw := range fixture.Intents {
		var intent validationv1.ValidationIntent
		if err := protojson.Unmarshal(raw, &intent); err != nil {
			t.Fatalf("fixture %d is not valid protobuf JSON: %v", index, err)
		}
		if _, err := normalizeIntent(&intent); err == nil {
			t.Errorf("malformed intent fixture %d was accepted", index)
		}
	}
}

func TestInvalidTransitionJSONFixturesRemainRejected(t *testing.T) { // [REQ:TESTGENIE-VALIDATION-RECEIPT-P0]
	var fixture struct {
		Transitions []struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"transitions"`
	}
	readJSONFixture(t, "invalid_transitions.json", &fixture)
	for _, candidate := range fixture.Transitions {
		fromNumber, fromOK := validationv1.ReceiptState_value[candidate.From]
		toNumber, toOK := validationv1.ReceiptState_value[candidate.To]
		if !fromOK || !toOK {
			t.Fatalf("unknown fixture state: %#v", candidate)
		}
		if CanTransition(validationv1.ReceiptState(fromNumber), validationv1.ReceiptState(toNumber)) {
			t.Errorf("invalid fixture transition became legal: %s -> %s", candidate.From, candidate.To)
		}
	}
}

func readJSONFixture(t *testing.T, name string, target any) {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatal(err)
	}
}
