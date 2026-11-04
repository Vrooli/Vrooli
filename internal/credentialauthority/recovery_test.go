package credentialauthority

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/credentialpolicy"
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

func TestRecoveryEnvelopeCarriesAuthenticatedPolicyAndReadsHistoricalVersionOne(t *testing.T) {
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
	if envelope.Version != 3 || envelope.Purpose != "recovery-bundle" || envelope.KDF != "pbkdf2-sha256" || envelope.Iterations <= 0 {
		t.Fatalf("envelope = %+v, want versioned KDF policy", envelope)
	}
	envelope.Version = 1
	envelope.Purpose = ""
	envelope.KDF = ""
	envelope.Iterations = 0
	// Re-seal the known plaintext in the historical no-AAD format. Merely
	// relabelling a current envelope would (correctly) fail authentication.
	plain, err := decryptRecovery(bundle, "passphrase")
	if err != nil {
		t.Fatal(err)
	}
	salt, err := base64.StdEncoding.DecodeString(envelope.Salt)
	if err != nil {
		t.Fatal(err)
	}
	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		t.Fatal(err)
	}
	key, err := recoveryKey("passphrase", salt, credentialpolicy.RecoveryPBKDF2Iterations)
	if err != nil {
		t.Fatal(err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	envelope.Ciphertext = base64.StdEncoding.EncodeToString(gcm.Seal(nil, nonce, plain, nil))
	legacyBundle, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := InspectRecovery(legacyBundle, "passphrase"); err != nil {
		t.Fatalf("version 1 recovery fixture failed after policy metadata was added: %v", err)
	}
}

func TestHistoricalRecoveryFixtureOpensThroughCompatibilityReader(t *testing.T) {
	fixturePath := filepath.Join("..", "credentialpolicy", "testdata", "historical-envelopes-v1.json")
	fixtureBytes, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read historical credential fixture: %v", err)
	}
	var fixture struct {
		Recovery struct {
			Version    int    `json:"version"`
			Salt       string `json:"salt"`
			Nonce      string `json:"nonce"`
			Ciphertext string `json:"ciphertext"`
			Passphrase string `json:"passphrase"`
		} `json:"recovery_v1"`
	}
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		t.Fatalf("decode historical credential fixture: %v", err)
	}
	bundle, err := json.Marshal(recoveryEnvelope{
		Version: fixture.Recovery.Version, Salt: fixture.Recovery.Salt,
		Nonce: fixture.Recovery.Nonce, Ciphertext: fixture.Recovery.Ciphertext,
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := InspectRecovery(bundle, fixture.Recovery.Passphrase)
	if err != nil {
		t.Fatalf("historical recovery fixture failed to open: %v", err)
	}
	if len(manifest.Entries) != 1 || string(manifest.Entries[0].Identity) != "vrooli/fixture" || manifest.Entries[0].Field != "api-key" {
		t.Fatalf("historical recovery manifest = %+v", manifest)
	}
}

func TestRecoveryRejectsMalformedEnvelopesWithoutPanicOrTargetChanges(t *testing.T) {
	fixtureBytes, err := os.ReadFile(filepath.Join("..", "credentialpolicy", "testdata", "historical-envelopes-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Recovery struct {
			recoveryEnvelope
			Passphrase string `json:"passphrase"`
		} `json:"recovery_v1"`
	}
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		t.Fatal(err)
	}
	marshal := func(envelope recoveryEnvelope) []byte {
		t.Helper()
		bundle, err := json.Marshal(envelope)
		if err != nil {
			t.Fatal(err)
		}
		return bundle
	}
	passphrase := fixture.Recovery.Passphrase
	legacy := fixture.Recovery.recoveryEnvelope
	plain, err := decryptRecovery(marshal(legacy), passphrase)
	if err != nil {
		t.Fatal(err)
	}
	currentBundle, err := encryptRecovery(plain, passphrase)
	if err != nil {
		t.Fatal(err)
	}
	var current recoveryEnvelope
	if err := json.Unmarshal(currentBundle, &current); err != nil {
		t.Fatal(err)
	}
	versionTwo := legacy
	versionTwo.Version = 2
	versionTwo.KDF = recoveryKDFPBKDF2SHA256
	versionTwo.Iterations = credentialpolicy.RecoveryPBKDF2Iterations
	for _, envelope := range []recoveryEnvelope{legacy, versionTwo, current} {
		t.Run(fmt.Sprintf("version_%d", envelope.Version), func(t *testing.T) {
			if _, err := InspectRecovery(marshal(envelope), passphrase); err != nil {
				t.Fatalf("valid recovery fixture rejected: %v", err)
			}
			reject := func(t *testing.T, bundle []byte, password string) {
				t.Helper()
				store := &authorityStore{}
				target, err := NewAuthority(store)
				if err != nil {
					t.Fatal(err)
				}
				id, _ := ParseIdentity("vrooli/fixture")
				if err := target.Put(id, "api-key", "existing-test-value"); err != nil {
					t.Fatal(err)
				}
				before := maps.Clone(store.values)
				check := func(name string, call func() error) {
					t.Helper()
					defer func() {
						if recover() != nil {
							t.Errorf("%s panicked instead of rejecting malformed input", name)
						}
					}()
					if err := call(); err == nil {
						t.Errorf("%s accepted malformed input", name)
					}
				}
				check("inspect", func() error {
					_, err := InspectRecovery(bundle, password)
					return err
				})
				check("restore", func() error { return target.RestoreRecovery(bundle, password) })
				if !maps.Equal(before, store.values) {
					t.Fatal("rejected recovery changed target credentials")
				}
			}
			for _, size := range []int{0, 1, 13} {
				t.Run(fmt.Sprintf("nonce_%d", size), func(t *testing.T) {
					malformed := envelope
					malformed.Nonce = base64.StdEncoding.EncodeToString(make([]byte, size))
					reject(t, marshal(malformed), passphrase)
				})
			}
			t.Run("wrong_passphrase", func(t *testing.T) { reject(t, marshal(envelope), "wrong-test-passphrase") })
			t.Run("truncated_ciphertext", func(t *testing.T) {
				malformed := envelope
				malformed.Ciphertext = base64.StdEncoding.EncodeToString([]byte{0})
				reject(t, marshal(malformed), passphrase)
			})
		})
	}
}
