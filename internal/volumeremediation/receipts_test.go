package volumeremediation

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReceiptRecordsSuccessfulNativeCheckAndExports(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipts.jsonl")
	request := Request{Action: ActionCheck, Device: elementsDevice()}
	result := Result{Action: ActionCheck, Device: elementsDevice(), Status: StatusVerified, Backend: "udisks2", Command: []string{"udisksctl", "check", "--block-device", "/dev/sda1"}}
	if err := RecordReceiptAt(path, request, result, nil, true); err != nil {
		t.Fatal(err)
	}
	rows, err := ReadReceiptsAt(path)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	row := rows[0]
	if row.Capability != "destination_check" || row.Outcome != "passed" || row.EvidenceKind != "native-runtime" || row.Platform == "" || row.Architecture == "" || row.Host == "" || row.ObservedAt == "" {
		t.Fatalf("incomplete receipt: %+v", row)
	}
}

func TestReceiptCapturesRefusalAndExecutionFailureAsFailed(t *testing.T) {
	for _, tc := range []struct {
		name string
		res  Result
		err  error
	}{
		{name: "refusal", res: Result{Status: StatusRefused}, err: ErrRefused{Reason: "unsafe"}},
		{name: "execution", res: Result{Status: StatusFailed}, err: errors.New("command failed")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "receipts.jsonl")
			req := Request{Action: ActionRepair, Device: elementsDevice(), AcknowledgeDataLoss: true}
			tc.res.Action, tc.res.Device = req.Action, req.Device
			if err := RecordReceiptAt(path, req, tc.res, tc.err, true); err != nil {
				t.Fatal(err)
			}
			rows, err := ReadReceiptsAt(path)
			if err != nil || len(rows) != 1 || rows[0].Outcome != "failed" || rows[0].Capability != "filesystem_repair" {
				t.Fatalf("rows=%+v err=%v", rows, err)
			}
		})
	}
}

func TestInspectReceiptDoesNotClaimMutationAndDryRunIsFixture(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipts.jsonl")
	req := Request{Action: ActionInspect, Device: elementsDevice()}
	res := Result{Action: ActionInspect, Device: elementsDevice(), Status: StatusVerified}
	if err := RecordReceiptAt(path, req, res, nil, true); err != nil {
		t.Fatal(err)
	}
	// A dry run never qualifies as native execution, even when the caller asks.
	req.Action, req.DryRun = ActionRepair, true
	res.Action, res.DryRun = req.Action, true
	if err := RecordReceiptAt(path, req, res, nil, true); err != nil {
		t.Fatal(err)
	}
	rows, err := ReadReceiptsAt(path)
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	if rows[0].Capability != "volume_identity" || rows[0].Outcome != "passed" || rows[1].EvidenceKind == "native-runtime" {
		t.Fatalf("unexpected inspect/dry-run receipts: %+v", rows)
	}
}

func TestReceiptSanitizesDeviceIdentityAndCommands(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipts.jsonl")
	req := Request{Action: ActionMountReadWrite, Device: elementsDevice()}
	res := Result{Action: req.Action, Device: req.Device, Status: StatusChanged, Command: []string{"mount", "/dev/sda1", "/media/alice/Elements"}}
	if err := RecordReceiptAt(path, req, res, nil, true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"WD-WX52A946D6VL", "E26A883E6A881189", "/dev/sda1", "/media/alice/Elements", "alice"} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("receipt leaked %q: %s", secret, data)
		}
	}
	if !strings.Contains(string(data), "identity_sha256=") || !strings.Contains(string(data), "result_sha256=") {
		var receipt NativeReceipt
		if err := json.Unmarshal(data, &receipt); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(receipt.Evidence, "identity_sha256=") || !strings.Contains(receipt.Evidence, "result_sha256=") || strings.Contains(receipt.Command, "/dev/sda1") || strings.Contains(receipt.Command, "/media/alice/Elements") {
			t.Fatalf("receipt omitted binding or sanitization: %+v", receipt)
		}
	}
}
