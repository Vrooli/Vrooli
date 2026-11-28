package targetruntime

import (
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/demand"
)

func TestAudioToolsRunGetsOwnerIssuedDemandLeaseReceipt(t *testing.T) {
	lease := demand.Lease{
		LeaseID: "lease-owner-issued", Scenario: "audio-tools", ConsumerID: "test-genie:job:run-audio-soak-1",
		Kind: demand.KindJob, RequestID: "run-audio-soak-1", Status: "active", ExpiresAt: time.Now().Add(time.Minute),
	}
	receipt, err := issueDemandLeaseReceipt("run-audio-soak-1", "audio-tools", lease)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(receipt.ReceiptID, "tglease_") || receipt.Owner != "test-genie" || receipt.RunID != lease.RequestID || receipt.Target != lease.Scenario {
		t.Fatalf("receipt is not owner-issued and run-bound: %+v", receipt)
	}
	if err := VerifyDemandLeaseReceipt(receipt, receipt, lease, "run-audio-soak-1", "audio-tools"); err != nil {
		t.Fatalf("verify owner-issued receipt: %v", err)
	}

	for name, scope := range map[string]struct{ run, target string }{
		"other run":    {"run-other", "audio-tools"},
		"other target": {"run-audio-soak-1", "other"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := VerifyDemandLeaseReceipt(receipt, receipt, lease, scope.run, scope.target); err == nil {
				t.Fatal("mismatched run or target verified")
			}
		})
	}
	forged := receipt
	forged.ReceiptID = "tglease_client_supplied"
	if err := VerifyDemandLeaseReceipt(forged, receipt, lease, "run-audio-soak-1", "audio-tools"); err == nil {
		t.Fatal("client-supplied receipt identifier verified")
	}
	wrongLease := lease
	wrongLease.LeaseID = "lease-from-another-run"
	if err := VerifyDemandLeaseReceipt(receipt, receipt, wrongLease, "run-audio-soak-1", "audio-tools"); err == nil {
		t.Fatal("receipt verified against a different retained lease")
	}
}

func TestDemandLeaseReceiptRequiresOwnerLease(t *testing.T) {
	if _, err := (Lease{}).DemandLeaseReceipt(); err != ErrDemandLeaseUnavailable {
		t.Fatalf("receipt without owner hold: got %v", err)
	}
	lease := demand.Lease{
		LeaseID: "client-chosen", Scenario: "audio-tools", ConsumerID: "client", Kind: demand.KindJob,
		RequestID: "run-audio-soak-1", Status: "active", ExpiresAt: time.Now().Add(time.Minute),
	}
	if _, err := issueDemandLeaseReceipt("run-audio-soak-1", "audio-tools", lease); err == nil {
		t.Fatal("client-selected lease identity was issued as a Test Genie receipt")
	}
}
