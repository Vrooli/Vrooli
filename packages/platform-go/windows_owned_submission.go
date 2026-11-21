package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"unicode/utf8"
)

// Submission state is durable metadata, never a liveness/ownership verdict.
// Every persisted state prohibits replay. Only a claimed state can subsequently
// be checked against actual native root/image/Job custody for Running.
type windowsOwnedSubmissionState string

const (
	windowsSubmissionReserved windowsOwnedSubmissionState = "reserved"
	windowsSubmissionPrepared windowsOwnedSubmissionState = "prepared"
	windowsSubmissionClaimed  windowsOwnedSubmissionState = "claimed"
)

func refuseWindowsOwnedReplay(expected, prior windowsOwnedPayload) error {
	if !reflect.DeepEqual(expected, prior) {
		return ErrOperationIdentity
	}
	return ErrOperationExists
}
func windowsOwnedSubmissionMetadata(p windowsOwnedPayload, root windowsOwnedRootRecord, c windowsOwnedCustodyRecord) (windowsOwnedSubmissionState, error) {
	if e := validateWindowsPayload(p); e != nil {
		return "", e
	}
	if root.PID == 0 {
		if root != (windowsOwnedRootRecord{}) || c.Version != 0 {
			return "", ErrOperationIdentity
		}
		return windowsSubmissionReserved, nil
	}
	if root.PID < 0 || !nativeIncarnationShape("windows", root.PID, root.Incarnation) || !nativeStartMatches("windows", root.Start, root.Incarnation) || root.ImageID == "" || root.ImageHandle == 0 || root.JobHandle == 0 || root.TaskInstance == "" {
		return "", ErrOperationIdentity
	}
	if c.Version == 0 {
		if c != (windowsOwnedCustodyRecord{}) {
			return "", ErrOperationIdentity
		}
		return windowsSubmissionPrepared, nil
	}
	if !c.matches(p, root) {
		return "", ErrOperationIdentity
	}
	return windowsSubmissionClaimed, nil
}

// RegFlushKey explicitly requires KEY_QUERY_VALUE. This write boundary requests
// only QUERY_VALUE|SET_VALUE (READ_CONTROL is added by the ledger verifier).
const windowsCustodyRegistryWriteAccess uint32 = 0x0001 | 0x0002

type windowsCustodyCommitEffects struct {
	WritePending func(string) error
	Flush        func() error
	WriteReady   func(string) error
}

// Pending registry values are visible before RegFlushKey. A separate ready
// marker is published only after the exact pending bytes have reached disk.
// The marker itself need not be durable to prove the already completed barrier.
func commitWindowsOwnedCustody(raw string, e windowsCustodyCommitEffects) error {
	if e.WritePending == nil || e.Flush == nil || e.WriteReady == nil {
		return ErrOperationIdentity
	}
	if err := e.WritePending(raw); err != nil {
		return err
	}
	if err := e.Flush(); err != nil {
		return err
	}
	return e.WriteReady(windowsCustodyReadyMarker(raw))
}
func windowsCustodyReadyMarker(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
func windowsCustodyReadyMatches(raw, marker string) bool {
	return raw != "" && validNativeBrokerDigest(marker) && marker == windowsCustodyReadyMarker(raw)
}

// An empty protected key is a pre-publication reservation, not an admitted
// intent. It can be discarded only when no native Job namespace exists.
func windowsOwnedUnpublishedSlotRemovable(values, subkeys uint32, nativeAbsent bool) bool {
	return values == 0 && subkeys == 0 && nativeAbsent
}

func validateWindowsFrozenText(path, dir string, args, env []string) error {
	values := []string{path, dir}
	values = append(values, args...)
	values = append(values, env...)
	for _, value := range values {
		if !utf8.ValidString(value) {
			return ErrOperationIdentity
		}
	}
	return nil
}
