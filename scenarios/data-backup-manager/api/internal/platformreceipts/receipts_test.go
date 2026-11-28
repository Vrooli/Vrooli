package platformreceipts

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestFileSinkWritesRedactedNativeReceipt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtime", "receipts.jsonl")
	sink := &FileSink{Path: path}
	if err := sink.Record(context.Background(), Input{
		Capability: "scratch_restore", Outcome: "passed", Operation: "restore-operation",
		TargetID: "private-target", DestinationID: "private-destination", Proof: "checksum-secret-like",
		ObservedAt: time.Date(2026, 11, 13, 20, 1, 2, 0, time.UTC),
	}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var receipt Receipt
	if err := json.Unmarshal(data[:len(data)-1], &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Platform == "" || receipt.Architecture == "" || receipt.Host == "" || receipt.EvidenceKind != "native-runtime" {
		t.Fatalf("receipt missing native execution identity: %+v", receipt)
	}
	if !regexp.MustCompile(`^host-[a-f0-9]{16}$`).MatchString(receipt.Host) {
		t.Fatalf("host identity is not privacy hashed: %q", receipt.Host)
	}
	for _, secret := range []string{"private-target", "private-destination", "checksum-secret-like"} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("receipt contains raw reference/proof %q", secret)
		}
	}
	if !strings.Contains(receipt.Evidence, "proof_sha256=") || !strings.Contains(receipt.Evidence, "operation_sha256=") {
		t.Fatalf("receipt lacks hashed evidence bindings: %+v", receipt)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("receipt file permissions are too broad: %o", info.Mode().Perm())
	}
}

func TestFileSinkRejectsUnsupportedOrIncompleteRows(t *testing.T) {
	sink := &FileSink{Path: filepath.Join(t.TempDir(), "rows.jsonl")}
	for _, input := range []Input{
		{Capability: "volume_mount", Outcome: "passed", Operation: "x", TargetID: "t", DestinationID: "d"},
		{Capability: "backup", Outcome: "unknown", Operation: "x", TargetID: "t", DestinationID: "d"},
		{Capability: "backup", Outcome: "passed", Operation: "x", TargetID: "", DestinationID: "d"},
	} {
		if err := sink.Record(context.Background(), input); err == nil {
			t.Fatalf("Record(%+v) succeeded", input)
		}
	}
}
