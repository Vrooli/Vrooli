package targetruntime

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/demand"
)

var ErrDemandLeaseUnavailable = errors.New("Test Genie demand lease receipt unavailable")

var demandLeaseReceipts = struct {
	sync.RWMutex
	issued map[string]issuedDemandLeaseReceipt
}{issued: make(map[string]issuedDemandLeaseReceipt)}

type issuedDemandLeaseReceipt struct {
	receipt DemandLeaseReceipt
	lease   demand.Lease
}

// DemandLeaseReceipt is Test Genie's owner-issued, run-scoped snapshot of the
// demand lease granted by the control plane. ReceiptID is an opaque lookup
// handle. It is not a grant: consumers must present it to Test Genie, which
// verifies it against the live run's retained lease.
type DemandLeaseReceipt struct {
	ReceiptID  string    `json:"receipt_id"`
	Owner      string    `json:"owner"`
	RunID      string    `json:"run_id"`
	Target     string    `json:"target"`
	LeaseID    string    `json:"lease_id"`
	ConsumerID string    `json:"consumer_id"`
	RequestID  string    `json:"request_id"`
	Kind       string    `json:"kind"`
	Status     string    `json:"status"`
	IssuedAt   time.Time `json:"issued_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// VerifyDemandLeaseReceipt verifies an issued receipt against the exact
// owner-retained control-plane lease. Expected run and target are constraints,
// never sources of lease identity. Call only with the lease retained by the
// Test Genie run manager, not a client-provided lease record.
func VerifyDemandLeaseReceipt(receipt, issued DemandLeaseReceipt, retained demand.Lease, expectedRunID, expectedTarget string) error {
	expectedRunID = strings.TrimSpace(expectedRunID)
	expectedTarget = strings.TrimSpace(expectedTarget)
	if receipt.ReceiptID == "" || receipt.Owner != "test-genie" || expectedRunID == "" || expectedTarget == "" {
		return errors.New("receipt, Test Genie owner, run, and target are required")
	}
	if receipt.RunID != expectedRunID || receipt.Target != expectedTarget {
		return errors.New("Test Genie demand lease receipt scope mismatch")
	}
	if receipt != issued {
		return errors.New("Test Genie demand lease receipt was not issued for this retained run")
	}
	if retained.LeaseID == "" || receipt.LeaseID != retained.LeaseID ||
		receipt.ConsumerID != retained.ConsumerID || receipt.RequestID != retained.RequestID ||
		receipt.Kind != retained.Kind || receipt.Status != "active" ||
		retained.Status != "active" || receipt.ExpiresAt.IsZero() ||
		!receipt.ExpiresAt.Equal(retained.ExpiresAt) || !receipt.ExpiresAt.After(time.Now()) {
		return errors.New("Test Genie demand lease receipt does not match the retained active lease")
	}
	if receipt.RequestID != expectedRunID || receipt.ConsumerID != "test-genie:job:"+expectedRunID || receipt.Kind != demand.KindJob || retained.Scenario != expectedTarget {
		return errors.New("Test Genie demand lease receipt is not bound to this target run")
	}
	return nil
}

// VerifyIssuedDemandLeaseReceipt is the Test Genie service-side verifier.
// Clients present only the opaque receipt ID and expected target; run and
// control-plane lease identity always come from Test Genie's retained record.
func VerifyIssuedDemandLeaseReceipt(receiptID, expectedTarget string) (DemandLeaseReceipt, error) {
	receiptID, expectedTarget = strings.TrimSpace(receiptID), strings.TrimSpace(expectedTarget)
	demandLeaseReceipts.RLock()
	issued, ok := demandLeaseReceipts.issued[receiptID]
	demandLeaseReceipts.RUnlock()
	if !ok {
		return DemandLeaseReceipt{}, errors.New("Test Genie demand lease receipt not found")
	}
	if err := VerifyDemandLeaseReceipt(issued.receipt, issued.receipt, issued.lease, issued.receipt.RunID, expectedTarget); err != nil {
		return DemandLeaseReceipt{}, err
	}
	return issued.receipt, nil
}

func revokeDemandLeaseReceipt(receiptID string) {
	demandLeaseReceipts.Lock()
	delete(demandLeaseReceipts.issued, receiptID)
	demandLeaseReceipts.Unlock()
}

func issueDemandLeaseReceipt(runID, target string, lease demand.Lease) (DemandLeaseReceipt, error) {
	runID, target = strings.TrimSpace(runID), strings.TrimSpace(target)
	if runID == "" || target == "" || lease.LeaseID == "" || lease.Status != "active" || lease.ExpiresAt.IsZero() {
		return DemandLeaseReceipt{}, fmt.Errorf("cannot issue receipt for incomplete Test Genie demand lease")
	}
	if lease.RequestID != runID || lease.ConsumerID != "test-genie:job:"+runID || lease.Kind != demand.KindJob || lease.Scenario != target {
		return DemandLeaseReceipt{}, errors.New("control-plane lease is not bound to this Test Genie target run")
	}
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return DemandLeaseReceipt{}, fmt.Errorf("mint Test Genie lease receipt id: %w", err)
	}
	receipt := DemandLeaseReceipt{
		ReceiptID: "tglease_" + hex.EncodeToString(nonce[:]), Owner: "test-genie",
		RunID: runID, Target: target, LeaseID: lease.LeaseID, ConsumerID: lease.ConsumerID,
		RequestID: lease.RequestID, Kind: lease.Kind, Status: lease.Status,
		IssuedAt: time.Now().UTC(), ExpiresAt: lease.ExpiresAt.UTC(),
	}
	demandLeaseReceipts.Lock()
	demandLeaseReceipts.issued[receipt.ReceiptID] = issuedDemandLeaseReceipt{receipt: receipt, lease: lease}
	demandLeaseReceipts.Unlock()
	return receipt, nil
}
