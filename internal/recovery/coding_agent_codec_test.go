package recovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodingAgentLedgerRejectsAmbiguousUnsupportedAndUnboundedState(t *testing.T) {
	valid := `{"id":"fixture","scenario":"fixture","outcome":"running"}`
	var many []string
	for i := 0; i <= codingAgentMaxRecords; i++ {
		many = append(many, fmt.Sprintf(`{"id":"fixture-%d","scenario":"fixture","outcome":"running"}`, i))
	}
	for name, data := range map[string]string{
		"duplicate field":     `[{"id":"fixture","id":"replacement","scenario":"fixture","outcome":"running"}]`,
		"case alias":          `[{"id":"fixture","ID":"replacement","scenario":"fixture","outcome":"running"}]`,
		"future schema":       `[{"id":"fixture","scenario":"fixture","outcome":"running","future_native_identity":{}}]`,
		"duplicate receipt":   "[" + valid + "," + valid + "]",
		"unsupported outcome": `[{"id":"fixture","scenario":"fixture","outcome":"new-native-phase"}]`,
		"trailing data":       "[" + valid + "] []",
		"null receipt":        "[null]",
		"too many records":    "[" + strings.Join(many, ",") + "]",
		"oversized":           "[" + strings.Repeat(" ", codingAgentMaxLedgerBytes) + "]",
		"oversized field":     `[{"id":"fixture","scenario":"fixture","outcome":"running","reason":"` + strings.Repeat("x", 4097) + `"}]`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "coding-agent-records.json")
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := NewCodingAgentBroker(path, nil); err == nil {
				t.Fatal("unsafe ledger became recovery authority")
			} else if name == "too many records" && !strings.Contains(err.Error(), "record limit") {
				t.Fatalf("record limit was not exercised: %v", err)
			}
			current, err := os.ReadFile(path)
			if err != nil || string(current) != data {
				t.Fatal("refusal rewrote unresolved state")
			}
		})
	}
}

func TestCodingAgentLedgerRefusesLeafAlias(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "other-owner.json")
	path := filepath.Join(dir, "coding-agent-records.json")
	if err := os.WriteFile(target, []byte("[]"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if _, err := NewCodingAgentBroker(path, nil); err == nil {
		t.Fatal("aliased ledger became writable recovery authority")
	}
}

func TestCodingAgentLedgerDiagnosticsDoNotEchoMalformedValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coding-agent-records.json")
	const private = "fixture-value-must-not-enter-health-or-logs"
	data := `[{"id":"fixture","scenario":"fixture","outcome":"running","created_at":"` + private + `"}]`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewCodingAgentBroker(path, nil); err == nil || strings.Contains(err.Error(), private) {
		t.Fatalf("unsafe diagnostic: %v", err)
	}
}
