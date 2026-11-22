package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type demandLeaseReceiptVerification struct {
	Verified  bool      `json:"verified"`
	ReceiptID string    `json:"receipt_id"`
	Owner     string    `json:"owner"`
	RunID     string    `json:"run_id"`
	Target    string    `json:"target"`
	LeaseID   string    `json:"lease_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Audio Tools presents only the opaque Test Genie receipt ID. Test Genie
// resolves run and control-plane lease identity from its own live receipt map.
func (s *Server) handleVerifyDemandLeaseReceipt(w http.ResponseWriter, r *http.Request) {
	if s.runsService == nil {
		http.Error(w, "Test Genie runs service unavailable", http.StatusServiceUnavailable)
		return
	}
	if r.URL.Query().Get("target") != "audio-tools" {
		http.Error(w, "target must be audio-tools", http.StatusBadRequest)
		return
	}
	receiptID := strings.TrimSpace(r.Header.Get("X-Test-Genie-Lease-Receipt"))
	if receiptID == "" {
		http.Error(w, "X-Test-Genie-Lease-Receipt is required", http.StatusUnauthorized)
		return
	}
	receipt, err := s.runsService.VerifyDemandLeaseReceipt(receiptID, "audio-tools")
	if err != nil {
		http.Error(w, "Test Genie lease receipt rejected", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(demandLeaseReceiptVerification{
		Verified: true, ReceiptID: receipt.ReceiptID, Owner: receipt.Owner,
		RunID: receipt.RunID, Target: receipt.Target, LeaseID: receipt.LeaseID,
		ExpiresAt: receipt.ExpiresAt,
	})
}
