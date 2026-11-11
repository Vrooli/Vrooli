package chainforensics

import (
	"strings"
	"testing"
)

func TestNormalizeERC20TransferLogKeepsExactAtomicQuantityAndProvenance(t *testing.T) {
	source := Source{
		ID: "blockscout-transfer-1", Name: "Blockscout API",
		URL:       "https://eth.blockscout.com/api/v2/transactions/0x" + strings.Repeat("a", 64) + "/token-transfers",
		Retrieved: "2026-11-11T01:00:00Z", SHA256: strings.Repeat("b", 64),
	}
	log := EVMLog{
		Address:          "0x1234567890ABCDEF1234567890abcdef12345678",
		Topics:           []string{ERC20TransferTopic, "0x000000000000000000000000abcdefabcdefabcdefabcdefabcdefabcdefabcd", "0x0000000000000000000000001111111111111111111111111111111111111111"},
		Data:             "0x000000000000000000000000000000000000000000000000ffffffffffffffff",
		BlockNumber:      "0x1234",
		TransactionHash:  "0x" + strings.Repeat("A", 64),
		TransactionIndex: "0x1",
		LogIndex:         "0x2",
	}
	got, err := NormalizeERC20TransferLog(ChainEthereum, source, log, "2026-11-10T23:59:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if got.Chain != ChainEthereum || got.Block != 0x1234 || got.EventIndex != 2 || got.Kind != EvidenceObservation {
		t.Fatalf("unexpected EVM occurrence identity: %#v", got)
	}
	if got.Atomic != "18446744073709551615" || got.Scale != 0 {
		t.Fatalf("atomic amount was rounded or assigned unverified token decimals: %#v", got)
	}
	if got.From != "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd" || got.To != "0x1111111111111111111111111111111111111111" {
		t.Fatalf("indexed addresses decoded incorrectly: %#v", got)
	}
	if got.Asset != "erc20:0x1234567890abcdef1234567890abcdef12345678" || got.SourceID != source.ID {
		t.Fatalf("asset or source provenance lost: %#v", got)
	}
	caseFile := Case{ID: "erc20-test", Title: "ERC20 transfer", Sources: []Source{source}, Occurrences: []Occurrence{got}}
	if err := Validate(caseFile); err != nil {
		t.Fatalf("normalized event is not valid case evidence: %v", err)
	}
}

func TestNormalizeERC20TransferLogRejectsAmbiguousOrMalformedEvents(t *testing.T) {
	source := Source{ID: "capture", Name: "fixture", URL: "https://example.test/log", Retrieved: "2026-11-11T00:00:00Z", SHA256: strings.Repeat("a", 64)}
	valid := EVMLog{
		Address:     "0x1234567890abcdef1234567890abcdef12345678",
		Topics:      []string{ERC20TransferTopic, "0x000000000000000000000000abcdefabcdefabcdefabcdefabcdefabcdefabcd", "0x0000000000000000000000001111111111111111111111111111111111111111"},
		Data:        "0x" + strings.Repeat("0", 64),
		BlockNumber: "0x1", TransactionHash: "0x" + strings.Repeat("a", 64), LogIndex: "0x0",
	}
	tests := []struct {
		name  string
		chain Chain
		edit  func(*EVMLog)
	}{
		{name: "unsupported chain", chain: ChainSolana},
		{name: "wrong event", chain: ChainEthereum, edit: func(l *EVMLog) { l.Topics[0] = "0x" + strings.Repeat("0", 64) }},
		{name: "noncanonical address padding", chain: ChainEthereum, edit: func(l *EVMLog) { l.Topics[1] = "0x01" + strings.Repeat("0", 62) }},
		{name: "bad amount", chain: ChainEthereum, edit: func(l *EVMLog) { l.Data = "0x01" }},
		{name: "bad block", chain: ChainEthereum, edit: func(l *EVMLog) { l.BlockNumber = "0x0" }},
		{name: "bad log index", chain: ChainEthereum, edit: func(l *EVMLog) { l.LogIndex = "0x100000000" }},
		{name: "missing time", chain: ChainEthereum},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := valid
			log.Topics = append([]string(nil), valid.Topics...)
			if tt.edit != nil {
				tt.edit(&log)
			}
			timestamp := "2026-11-11T00:00:00Z"
			if tt.name == "missing time" {
				timestamp = ""
			}
			if _, err := NormalizeERC20TransferLog(tt.chain, source, log, timestamp); err == nil {
				t.Fatal("expected malformed event to be rejected")
			}
		})
	}
}

func TestNormalizeERC20TransferLogRequiresSourceCaptureProvenance(t *testing.T) {
	source := Source{ID: "capture", Name: "fixture", URL: "https://example.test/log", Retrieved: "2026-11-11T00:00:00Z", SHA256: strings.Repeat("a", 64)}
	log := EVMLog{
		Address:     "0x1234567890abcdef1234567890abcdef12345678",
		Topics:      []string{ERC20TransferTopic, "0x000000000000000000000000abcdefabcdefabcdefabcdefabcdefabcdefabcd", "0x0000000000000000000000001111111111111111111111111111111111111111"},
		Data:        "0x" + strings.Repeat("0", 64),
		BlockNumber: "0x1", TransactionHash: "0x" + strings.Repeat("a", 64), LogIndex: "0x0",
	}
	for _, tc := range []struct {
		name string
		edit func(*Source)
	}{
		{name: "missing response hash", edit: func(s *Source) { s.SHA256 = "" }},
		{name: "invalid source URL", edit: func(s *Source) { s.URL = "file:///tmp/log.json" }},
		{name: "invalid retrieval time", edit: func(s *Source) { s.Retrieved = "yesterday" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := source
			tc.edit(&bad)
			if _, err := NormalizeERC20TransferLog(ChainBase, bad, log, "2026-11-10T23:59:00Z"); err == nil {
				t.Fatal("expected incomplete provenance to be rejected")
			}
		})
	}
}
