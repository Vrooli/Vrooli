package volumeremediation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

// NativeReceipt is a privacy-safe record of a volume owner operation. Device
// selectors and serials are never persisted; identity_sha256 binds the row to
// the observed volume without disclosing either value.
type NativeReceipt struct {
	ID           string `json:"id"`
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
	Host         string `json:"host"`
	ObservedAt   string `json:"observed_at"`
	Action       string `json:"action"`
	Capability   string `json:"capability"`
	Outcome      string `json:"outcome"`
	EvidenceKind string `json:"evidence_kind"`
	Command      string `json:"command"`
	Evidence     string `json:"evidence"`
}

var receiptWriteMu sync.Mutex

// ReceiptPath resolves the durable host-local receipt ledger. VROOLI_HOME is
// honored so managed installations keep receipts with their runtime state.
func ReceiptPath() (string, error) {
	if root := strings.TrimSpace(os.Getenv("VROOLI_HOME")); root != "" {
		return filepath.Join(root, "state", "volume-remediation", "receipts.jsonl"), nil
	}
	return config.VrooliPath(repocontract.HomeKeyState, "volume-remediation", "receipts.jsonl")
}

// RecordReceipt stores one operation result. nativeExecution must only be true
// for a live owner invocation on this host; dry runs and fixtures remain
// visibly unqualified.
func RecordReceipt(request Request, result Result, execErr error, nativeExecution bool) error {
	path, err := ReceiptPath()
	if err != nil {
		return err
	}
	return RecordReceiptAt(path, request, result, execErr, nativeExecution)
}

// RecordReceiptAt is the injectable store seam used by owner tests.
func RecordReceiptAt(path string, request Request, result Result, execErr error, nativeExecution bool) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("receipt path is required")
	}
	if request.DryRun || result.DryRun {
		nativeExecution = false
	}
	status := result.Status
	if status == "" {
		status = StatusFailed
	}
	outcome := "failed"
	if execErr == nil && (status == StatusVerified || status == StatusChanged || status == StatusAlreadySatisfied) {
		outcome = "passed"
	}
	capability := receiptCapability(request.Action)
	if capability == "" {
		return fmt.Errorf("unsupported receipt action %q", request.Action)
	}
	device := result.Device
	if device.Path == "" {
		device = request.Device
	}
	identity := strings.Join([]string{strings.ToLower(strings.TrimSpace(device.Filesystem)), strings.ToLower(strings.TrimSpace(device.UUID)), strings.TrimSpace(device.Serial), fmt.Sprint(device.TotalBytes)}, "\x00")
	identityDigest := digest(identity)
	command := safeCommand(result.Command)
	if command == "" {
		command = "native volume observer"
	}
	stateEvidence := strings.Join([]string{fmt.Sprint(result.State.Mounted), fmt.Sprint(result.State.ReadOnly), string(result.State.Dirty), strings.ToLower(strings.TrimSpace(result.State.Device.Filesystem))}, "\x00")
	evidenceDigest := digest(strings.Join([]string{identityDigest, string(request.Action), status, outcome, result.Backend, command, string(result.Consistent), stateEvidence}, "\x00"))
	kind := "fixture"
	if nativeExecution {
		kind = "native-runtime"
	}
	host, err := safeHostID()
	if err != nil {
		return err
	}
	observedAt := time.Now().UTC()
	r := NativeReceipt{
		ID:       "volume-" + observedAt.Format("20060102t150405000000000z") + "-" + evidenceDigest[:12],
		Platform: receiptPlatform(runtime.GOOS), Architecture: runtime.GOARCH, Host: host,
		ObservedAt: observedAt.Format(time.RFC3339Nano), Capability: capability,
		Action: string(request.Action), Outcome: outcome, EvidenceKind: kind, Command: command,
		Evidence: "identity_sha256=" + identityDigest + "; result_sha256=" + evidenceDigest + "; status=" + status,
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	receiptWriteMu.Lock()
	defer receiptWriteMu.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

func receiptPlatform(goos string) string {
	if goos == "darwin" {
		return "macos"
	}
	return goos
}

// ReadReceipts exports the owner ledger as typed rows for support-matrix
// generators and other control-plane consumers.
func ReadReceipts() ([]NativeReceipt, error) {
	path, err := ReceiptPath()
	if err != nil {
		return nil, err
	}
	return ReadReceiptsAt(path)
}

func ReadReceiptsAt(path string) ([]NativeReceipt, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []NativeReceipt{}, nil
	}
	if err != nil {
		return nil, err
	}
	var rows []NativeReceipt
	for lineNo, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var row NativeReceipt
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("decode volume receipt line %d: %w", lineNo+1, err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func receiptCapability(action Action) string {
	switch action {
	case ActionInspect:
		return "volume_identity"
	case ActionCheck:
		return "destination_check"
	case ActionRepair:
		return "filesystem_repair"
	case ActionUnmount, ActionMountReadWrite:
		return "volume_mount"
	default:
		return ""
	}
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func safeHostID() (string, error) {
	host, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return "host-" + digest(strings.TrimSpace(host))[:16], nil
}

// safeCommand retains the operation shape while replacing device paths and
// mount paths that could disclose user or machine details.
func safeCommand(argv []string) string {
	if len(argv) == 0 {
		return ""
	}
	parts := make([]string, 0, len(argv))
	for i, arg := range argv {
		if i == 0 {
			arg = filepath.Base(arg)
		} else if strings.HasPrefix(arg, "--") && strings.Contains(arg, "=") {
			key, value, _ := strings.Cut(arg, "=")
			if pathLike(value) {
				arg = key + "=<path>"
			}
		} else if pathLike(arg) {
			arg = "<path>"
		}
		parts = append(parts, arg)
	}
	return strings.Join(parts, " ")
}

func pathLike(value string) bool {
	return strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\\`) || strings.Contains(value, `:\`)
}
