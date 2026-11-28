//go:build lpbs_rehearsal

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"landing-page-business-suite-api/internal/commerce"
)

func TestPS2RehearsalFileVerifierBindsDurableJobAndOffer(t *testing.T) {
	root := t.TempDir()
	verifier, err := newPS2RehearsalFileVerifier(root)
	if err != nil {
		t.Fatal(err)
	}
	input := json.RawMessage(`{"sector":"Apple FY2026 public filings"}`)
	artifact := ps2DeliveryArtifact{
		JobID: "job-1", OfferID: "company-market-scan", OfferVersion: 1,
		JobInput: input,
		Report:   "# Test Scan\n\n## Scope\nApple FY2026 public filings. This test report summarizes a company filing and identifies the bounds of the supplied evidence.\n\n## Source\nhttps://www.sec.gov/Archives/edgar/data/example/filing.htm\n\n## Not checked\nCompetitor filings, forward estimates, market share, and analyst interpretation.\n",
	}
	write := func(value ps2DeliveryArtifact) {
		t.Helper()
		raw, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if writeErr := os.WriteFile(filepath.Join(root, value.JobID+".json"), raw, 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	write(artifact)
	order := commerce.ServiceOrder{ID: "order-1", JobID: "job-1", OfferID: "company-market-scan", OfferVersion: 1, JobInput: input}
	first, err := verifier.Verify(context.Background(), order)
	if err != nil || !first.Passed || first.ReceiptID == "" {
		t.Fatalf("Verify() = %+v, %v; want bound valid artifact", first, err)
	}
	replay, err := verifier.Verify(context.Background(), order)
	if err != nil || replay != first {
		t.Fatalf("same artifact replay = %+v, %v; want same receipt %+v", replay, err, first)
	}
	order.VerificationID = first.ReceiptID
	order.State = commerce.ServiceOrderReconciled
	result, err := verifier.ReadResult(context.Background(), order)
	if err != nil || result.Format != "markdown" || string(result.Body) != artifact.Report {
		t.Fatalf("verified result = %+v, %v; want the order-bound Markdown artifact", result, err)
	}
	artifact.JobInput = json.RawMessage(`{"sector":"changed input"}`)
	write(artifact)
	changed, err := verifier.Verify(context.Background(), order)
	if err != nil || changed.Passed || changed.ReceiptID == first.ReceiptID {
		t.Fatalf("changed-input artifact = %+v, %v; want failed distinct verification receipt", changed, err)
	}
}

func TestPS2RehearsalFileVerifierRejectsSymlinks(t *testing.T) {
	root := t.TempDir()
	verifier, err := newPS2RehearsalFileVerifier(root)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "artifact.json")
	if err := os.WriteFile(outside, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "job-1.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.Verify(context.Background(), commerce.ServiceOrder{JobID: "job-1"}); err == nil {
		t.Fatal("symlink delivery artifact was accepted")
	}
}
