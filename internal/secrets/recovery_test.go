package secrets

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecoveryRoundTripDoesNotExposeValueInBundleMetadata(t *testing.T) {
	source, err := NewAuthority(&authorityStore{})
	if err != nil {
		t.Fatal(err)
	}
	id, _ := ParseIdentity("vrooli/openrouter")
	const value = "recovery-fixture-value"
	if err := source.Put(id, "api-key", value); err != nil {
		t.Fatal(err)
	}
	bundle, err := source.ExportRecovery([]RecoveryEntry{{Identity: id, Field: "api-key"}}, "passphrase")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(bundle, []byte(value)) {
		t.Fatal("recovery bundle exposes plaintext value")
	}
	target, err := NewAuthority(&authorityStore{})
	if err != nil {
		t.Fatal(err)
	}
	if err := target.RestoreRecovery(bundle, "passphrase"); err != nil {
		t.Fatal(err)
	}
	got, err := target.read(id, "api-key")
	if err != nil || got != value {
		t.Fatalf("restore = %q, %v", got, err)
	}
	if err := target.RestoreRecovery(bundle, "wrong"); err == nil {
		t.Fatal("wrong passphrase restored bundle")
	}
}

func TestRecoveryReceiptCarriesVerificationMetadataWithoutValues(t *testing.T) {
	state := t.TempDir()
	id, _ := ParseIdentity("vrooli/fixture")
	entries := []RecoveryEntry{{Identity: id, Field: "api-key"}}
	metadata := RecoveryReceipt{ArtifactIdentity: "artifact-id", SourceGeneration: "generation-1", Checksum: "checksum", VerifiedAt: time.Now().UTC(), Verification: "decrypt-readback", SinkIdentity: "sink-id", ScheduleState: "ready", Remediation: ""}
	if err := WriteRecoveryReceiptWithMetadata(state, filepath.Join(state, "bundle.json"), entries, metadata, time.Now()); err != nil {
		t.Fatal(err)
	}
	receipt, found, err := ReadRecoveryReceipt(state)
	if err != nil || !found {
		t.Fatalf("read receipt = %+v, %v, found=%v", receipt, err, found)
	}
	if receipt.SourceGeneration != metadata.SourceGeneration || receipt.Checksum != metadata.Checksum || receipt.ScheduleState != "ready" || !receipt.Covers(id, "api-key") {
		t.Fatalf("receipt = %+v", receipt)
	}
	payload, err := os.ReadFile(filepath.Join(state, recoveryReceiptFile))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(payload, []byte("secret-answer")) {
		t.Fatal("receipt contains a secret value")
	}
}

func TestRecoveryEnvelopeCarriesKDFPolicyAndReadsVersionOne(t *testing.T) {
	source, err := NewAuthority(&authorityStore{})
	if err != nil {
		t.Fatal(err)
	}
	id, _ := ParseIdentity("vrooli/fixture")
	if err := source.Put(id, "api-key", "fixture-value"); err != nil {
		t.Fatal(err)
	}
	bundle, err := source.ExportRecovery([]RecoveryEntry{{Identity: id, Field: "api-key"}}, "passphrase")
	if err != nil {
		t.Fatal(err)
	}
	var envelope recoveryEnvelope
	if err := json.Unmarshal(bundle, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Version != 2 || envelope.KDF != "pbkdf2-sha256" || envelope.Iterations <= 0 {
		t.Fatalf("envelope = %+v, want versioned KDF policy", envelope)
	}
	envelope.Version = 1
	envelope.KDF = ""
	envelope.Iterations = 0
	legacyBundle, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := InspectRecovery(legacyBundle, "passphrase"); err != nil {
		t.Fatalf("version 1 recovery fixture failed after policy metadata was added: %v", err)
	}
}
