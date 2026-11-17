package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	x402rail "treasury/internal/rail/x402"
)

type treasuryX402TestSigner struct{}

func (treasuryX402TestSigner) SignTypedData(context.Context, x402rail.Credential, map[string]any) (string, error) {
	return "fixture-signature", nil
}

// [REQ:TRS-P1-001] Normal startup never enables response-body persistence.
func TestTreasuryX402UsesContentFreeConstructorByDefault(t *testing.T) {
	for _, name := range []string{"TREASURY_X402_PAID_BODY_EVIDENCE_DIR", "TREASURY_X402_PAID_BODY_EVIDENCE_ROOT", "TREASURY_CF5_REHEARSAL", "TREASURY_CF5_NOTIFICATION_TEST_MODE", "X402_RUNTIME_MODE"} {
		t.Setenv(name, "")
	}
	adapter, err := newTreasuryX402Adapter(&http.Client{}, treasuryX402TestSigner{})
	if err != nil || adapter == nil {
		t.Fatalf("default Treasury x402 adapter = %v, err=%v", adapter, err)
	}
}

// [REQ:TRS-P1-001] Raw response capture is opt-in only for the isolated
// CF5 local-fork test runtime.
func TestTreasuryX402EvidenceRequiresIsolatedLocalForkTestMode(t *testing.T) {
	root := filepath.Join(t.TempDir(), "run")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TREASURY_X402_PAID_BODY_EVIDENCE_ROOT", root)
	t.Setenv("TREASURY_X402_PAID_BODY_EVIDENCE_DIR", filepath.Join(root, "agent-manager-cli"))
	t.Setenv("TREASURY_CF5_REHEARSAL", "1")
	t.Setenv("TREASURY_CF5_NOTIFICATION_TEST_MODE", "1")
	t.Setenv("X402_RUNTIME_MODE", "")
	if _, err := newTreasuryX402Adapter(&http.Client{}, treasuryX402TestSigner{}); err == nil {
		t.Fatal("paid-body evidence must refuse outside local-fork runtime mode")
	}
	t.Setenv("X402_RUNTIME_MODE", "local-fork")
	if _, err := newTreasuryX402Adapter(&http.Client{}, treasuryX402TestSigner{}); err != nil {
		t.Fatalf("isolated local-fork evidence adapter: %v", err)
	}
}

// [REQ:TRS-P1-001] A configured body capture cannot write outside its current
// run's evidence root.
func TestTreasuryX402EvidenceRejectsRunDirectoryEscape(t *testing.T) {
	root := filepath.Join(t.TempDir(), "run")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TREASURY_X402_PAID_BODY_EVIDENCE_ROOT", root)
	t.Setenv("TREASURY_X402_PAID_BODY_EVIDENCE_DIR", filepath.Join(filepath.Dir(root), "outside"))
	t.Setenv("TREASURY_CF5_REHEARSAL", "1")
	t.Setenv("TREASURY_CF5_NOTIFICATION_TEST_MODE", "1")
	t.Setenv("X402_RUNTIME_MODE", "local-fork")
	if _, err := newTreasuryX402Adapter(&http.Client{}, treasuryX402TestSigner{}); err == nil {
		t.Fatal("paid-body evidence must refuse a directory outside this run root")
	}
}
