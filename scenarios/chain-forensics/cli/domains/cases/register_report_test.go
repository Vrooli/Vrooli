package cases

import (
	"encoding/json"
	"strings"
	"testing"

	p "github.com/vrooli/vrooli/packages/proto/gen/go/chain-forensics/v1/cases"
)

func TestCaptureSolanaAddressPageReportShowsPumpDecodeCounts(t *testing.T) {
	report := (&client{}).captureSolanaAddressPageReport(nil, &p.CaptureSolanaAddressPageResponse{
		CapturedTransactionCount:       3,
		CapturedOccurrenceCount:        7,
		CapturedPumpInstructionCount:   1,
		UnresolvedPumpInstructionCount: 2,
		NextBeforeSignature:            "next-cursor",
	})
	joined := strings.Join(report.Result, " ")
	for _, expected := range []string{"3 finalized Solana transaction(s)", "7 token-balance observation(s)", "1 decoded Pump instruction(s)", "2 Pump instruction(s) remain unresolved", "case ", "Next page cursor: next-cursor"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("report %q does not include %q", joined, expected)
		}
	}
}

func TestReviseCaseReportShowsPumpEvidenceCounts(t *testing.T) {
	report := (&client{}).reviseReport(nil, &p.ReviseCaseResponse{
		RevisionId: "revision-id",
		Case: &p.EvidenceCase{
			Id:                         "case-id",
			PumpInstructions:           []*p.PumpInstruction{{Id: "decoded"}},
			UnresolvedPumpInstructions: []*p.UnresolvedPumpInstruction{{Id: "unresolved-1"}, {Id: "unresolved-2"}},
		},
	})
	joined := strings.Join(report.Changes, " ")
	for _, expected := range []string{"1 decoded Pump instruction(s)", "2 unresolved Pump instruction(s)", "revision receipts are retained"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("report %q does not include %q", joined, expected)
		}
	}
}

func TestCaptureSolanaSignatureOnlyPageReportLimitsItsClaim(t *testing.T) {
	report := (&client{}).captureSolanaAddressPageReport(nil, &p.CaptureSolanaAddressPageResponse{CapturedSignatureCount: 1000, NextBeforeSignature: "next-signature", SignaturesOnly: true})
	joined := strings.Join(append(report.Result, report.Changes...), " ")
	for _, expected := range []string{"1000 finalized signature(s)", "without fetching transactions", "Next page cursor: next-signature", "do not establish transaction-level transfers"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("report %q does not include %q", joined, expected)
		}
	}
}

func TestCaptureSolanaTransactionReportDisclosesSelectionLineage(t *testing.T) {
	report := (&client{}).captureSolanaTransactionReport(
		nil,
		&p.CaptureSolanaTransactionResponse{Signature: "sample-signature", CapturedOccurrenceCount: 2, CapturedPumpInstructionCount: 1, UnresolvedPumpInstructionCount: 1},
	)
	joined := strings.Join(append(report.Result, report.Changes...), " ")
	for _, expected := range []string{"sample-signature", "2 token-balance observation(s)", "1 decoded Pump instruction(s)", "1 unresolved instruction(s)", "selector-page lineage was not supplied"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("report %q does not include %q", joined, expected)
		}
	}
	selected := (&client{}).captureSolanaTransactionReport(nil, &p.CaptureSolanaTransactionResponse{Signature: "sample-signature", SelectionSourceId: "history-source-1"})
	if !strings.Contains(strings.Join(selected.Changes, " "), "verified against retained finalized signature-history source history-source-1") {
		t.Fatalf("selected report %v does not disclose verified history source", selected.Changes)
	}
}

func TestNormalizeEnumsPreservesPumpInstructionKind(t *testing.T) {
	input := map[string]any{
		"occurrences":       []any{map[string]any{"kind": "observation", "chain": "solana"}},
		"pump_instructions": []any{map[string]any{"kind": "create_v2"}},
	}
	normalizeEnums(input)
	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, expected := range []string{`"kind":"EVIDENCE_KIND_OBSERVATION"`, `"chain":"CHAIN_SOLANA"`, `"kind":"create_v2"`} {
		if !strings.Contains(text, expected) {
			t.Fatalf("normalized JSON %s is missing %s", text, expected)
		}
	}
}
