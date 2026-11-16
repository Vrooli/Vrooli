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

func TestNormalizeEnumsPreservesPumpInstructionKind(t *testing.T) {
	input := map[string]any{
		"occurrences": []any{map[string]any{"kind": "observation", "chain": "solana"}},
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
