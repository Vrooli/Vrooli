package credentialauthority

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/credentialpolicy"
)

const (
	recoveryParameterA      = 2
	recoveryParameterB      = 3
	recoveryParameterC      = 32
	recoveryKDFPBKDF2SHA256 = "pbkdf2-sha256"
)

// RecoveryEntry identifies a value to include in an encrypted recovery
// bundle. Callers supply descriptors; the authority never enumerates secrets.
type RecoveryEntry struct {
	Identity Identity `json:"identity"`
	Field    string   `json:"field"`
}

type recoveryPayload struct {
	Version int             `json:"version"`
	Values  []recoveryValue `json:"values"`
}
type recoveryValue struct {
	Identity Identity `json:"identity"`
	Field    string   `json:"field"`
	Value    string   `json:"value"`
}
type recoveryEnvelope struct {
	Version    int    `json:"version"`
	Purpose    string `json:"purpose,omitempty"`
	KDF        string `json:"kdf,omitempty"`
	Iterations int    `json:"iterations,omitempty"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// ExportRecovery creates an opaque encrypted recovery bundle for explicitly
// selected credentials. The passphrase exists only in the caller's memory.
func (a *Authority) ExportRecovery(entries []RecoveryEntry, passphrase string) ([]byte, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("at least one recovery entry is required")
	}
	if strings.TrimSpace(passphrase) == "" {
		return nil, fmt.Errorf("recovery passphrase is required")
	}
	// Recovery stays fail-closed while the rest of the seam degrades. Writing a
	// bundle from a store we cannot read would hand the operator an artifact
	// that looks like a backup and is not one.
	if err := a.Availability(); err != nil {
		return nil, fmt.Errorf("export recovery bundle: %w", err)
	}
	payload := recoveryPayload{Version: 1, Values: make([]recoveryValue, 0, len(entries))}
	for _, entry := range entries {
		identity, err := ParseIdentity(string(entry.Identity))
		if err != nil {
			return nil, err
		}
		field := strings.TrimSpace(entry.Field)
		if field == "" {
			return nil, fmt.Errorf("recovery field is required")
		}
		value, err := a.read(identity, field)
		if err != nil {
			return nil, fmt.Errorf("recover %s/%s: %w", identity, field, err)
		}
		payload.Values = append(payload.Values, recoveryValue{Identity: identity, Field: field, Value: value})
	}
	plain, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return encryptRecovery(plain, passphrase)
}

// RestoreRecovery decrypts and writes a bundle only through this authority.
func (a *Authority) RestoreRecovery(bundle []byte, passphrase string) error {
	if strings.TrimSpace(passphrase) == "" {
		return fmt.Errorf("recovery passphrase is required")
	}
	if err := a.Availability(); err != nil {
		return fmt.Errorf("restore recovery bundle: %w", err)
	}
	plain, err := decryptRecovery(bundle, passphrase)
	if err != nil {
		return err
	}
	var payload recoveryPayload
	if err := json.Unmarshal(plain, &payload); err != nil {
		return fmt.Errorf("decode recovery bundle: %w", err)
	}
	if payload.Version != 1 || len(payload.Values) == 0 {
		return fmt.Errorf("unsupported or empty recovery bundle")
	}
	for _, value := range payload.Values {
		if err := a.Put(value.Identity, value.Field, value.Value); err != nil {
			return fmt.Errorf("restore %s/%s: %w", value.Identity, value.Field, err)
		}
	}
	return nil
}

// read is the recovery-side value read. Unlike the injection path it has no
// degraded mode: a bundle that silently omitted a value the operator believed
// was captured would be worse than no bundle at all.
func (a *Authority) read(identity Identity, field string) (string, error) {
	if a == nil || a.store == nil {
		return "", fmt.Errorf("%w: no credential store on this host", ErrProviderAbsent)
	}
	value, err := a.get(identity, field)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) == "" {
		return "", ErrUnconfigured
	}
	return value, nil
}

func encryptRecovery(plain []byte, passphrase string) ([]byte, error) {
	salt := make([]byte, recoveryParameterC)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	key, err := recoveryKey(passphrase, salt, credentialpolicy.RecoveryPBKDF2Iterations)
	if err != nil {
		return nil, err
	}
	sealed, err := credentialpolicy.Seal(key, plain, "recovery-bundle", recoveryParameterB)
	if err != nil {
		return nil, err
	}
	envelope := recoveryEnvelope{Version: sealed.Version, Purpose: sealed.Purpose, KDF: "pbkdf2-sha256", Iterations: credentialpolicy.RecoveryPBKDF2Iterations, Salt: base64.StdEncoding.EncodeToString(salt), Nonce: base64.StdEncoding.EncodeToString(sealed.Nonce), Ciphertext: base64.StdEncoding.EncodeToString(sealed.Ciphertext)}
	return json.Marshal(envelope)
}

func decryptRecovery(bundle []byte, passphrase string) ([]byte, error) {
	var envelope recoveryEnvelope
	if err := json.Unmarshal(bundle, &envelope); err != nil {
		return nil, fmt.Errorf("decode recovery envelope: %w", err)
	}
	iterations := credentialpolicy.RecoveryPBKDF2Iterations
	switch envelope.Version {
	case 1:
		// Version 1 had no KDF metadata and is fixed to the original policy.
	case recoveryParameterA:
		if envelope.KDF != recoveryKDFPBKDF2SHA256 || envelope.Iterations <= 0 {
			return nil, fmt.Errorf("unsupported recovery KDF policy")
		}
		iterations = envelope.Iterations
	case recoveryParameterB:
		if envelope.KDF != recoveryKDFPBKDF2SHA256 || envelope.Iterations <= 0 || envelope.Purpose != "recovery-bundle" {
			return nil, fmt.Errorf("unsupported recovery envelope policy")
		}
		iterations = envelope.Iterations
	default:
		return nil, fmt.Errorf("unsupported recovery bundle version")
	}
	salt, err := base64.StdEncoding.DecodeString(envelope.Salt)
	if err != nil {
		return nil, fmt.Errorf("decode recovery salt: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decode recovery nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode recovery ciphertext: %w", err)
	}
	key, err := recoveryKey(passphrase, salt, iterations)
	if err != nil {
		return nil, err
	}
	if envelope.Version == 1 || envelope.Version == 2 {
		// Historical recovery bundles were AES-GCM envelopes without
		// authenticated framing. Keep their reader forever; only the writer
		// moves to the framed format.
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}
		plain, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return nil, fmt.Errorf("decrypt recovery bundle: authorization failed")
		}
		return plain, nil
	}
	plain, err := credentialpolicy.Open(key, credentialpolicy.Envelope{
		Version: envelope.Version, Purpose: envelope.Purpose, Nonce: nonce, Ciphertext: ciphertext,
	})
	if err != nil {
		return nil, fmt.Errorf("decrypt recovery bundle: authorization failed")
	}
	return plain, nil
}

func recoveryKey(passphrase string, salt []byte, iterations int) ([]byte, error) {
	return pbkdf2.Key(sha256.New, passphrase, salt, iterations, recoveryParameterC)
}

// RecoveryManifest describes what a bundle contains, without exposing any
// value it holds.
type RecoveryManifest struct {
	Version int             `json:"version"`
	Entries []RecoveryEntry `json:"entries"`
}

// InspectRecovery proves a bundle opens with a passphrase and reports what it
// would restore.
//
// An operator who cannot check a bundle has not made a backup, they have made
// an artifact they hope is a backup — and the difference only surfaces on the
// day the original is gone. Verifying it needs no credential store and touches
// nothing: the bundle and the passphrase are the whole input, so this is safe
// to run on a host that is not the one that wrote it.
//
// It deliberately returns identities and fields but never values. Confirming
// that a bundle is intact must not require printing the secrets it protects.
func InspectRecovery(bundle []byte, passphrase string) (RecoveryManifest, error) {
	if strings.TrimSpace(passphrase) == "" {
		return RecoveryManifest{}, fmt.Errorf("recovery passphrase is required")
	}
	plain, err := decryptRecovery(bundle, passphrase)
	if err != nil {
		return RecoveryManifest{}, err
	}
	var payload recoveryPayload
	if err := json.Unmarshal(plain, &payload); err != nil {
		return RecoveryManifest{}, fmt.Errorf("decode recovery bundle: %w", err)
	}
	if payload.Version != 1 || len(payload.Values) == 0 {
		return RecoveryManifest{}, fmt.Errorf("unsupported or empty recovery bundle")
	}
	manifest := RecoveryManifest{Version: payload.Version, Entries: make([]RecoveryEntry, 0, len(payload.Values))}
	for _, value := range payload.Values {
		// A stored entry with no value would restore nothing, so a bundle
		// carrying one is not intact even though it decrypted cleanly.
		if strings.TrimSpace(value.Value) == "" {
			return RecoveryManifest{}, fmt.Errorf("recovery bundle holds an empty value for %s/%s", value.Identity, value.Field)
		}
		manifest.Entries = append(manifest.Entries, RecoveryEntry{Identity: value.Identity, Field: value.Field})
	}
	return manifest, nil
}

// recoveryReceiptFile records the last successful export. It lives beside other
// durable runtime state and holds no secret: a path, a time, and the identities
// a bundle covers, all of which are already public in manifests.
const recoveryReceiptFile = "recovery-receipt.json"

// RecoveryReceipt is evidence that a bundle was made, so an operator can be
// told when one has not been.
//
// It exists because the absence of a backup is silent by nature. Nothing about
// a healthy host looks different from one whose credentials have never been
// exported, so the gap is invisible right up to the moment it is permanent.
type RecoveryReceipt struct {
	Path             string    `json:"path"`
	ArtifactIdentity string    `json:"artifact_identity,omitempty"`
	SourceGeneration string    `json:"source_generation,omitempty"`
	Checksum         string    `json:"checksum,omitempty"`
	VerifiedAt       time.Time `json:"verified_at,omitempty"`
	Verification     string    `json:"verification,omitempty"`
	SinkIdentity     string    `json:"sink_identity,omitempty"`
	ScheduleState    string    `json:"schedule_state,omitempty"`
	Remediation      string    `json:"remediation,omitempty"`
	ExportedAt       time.Time `json:"exported_at"`
	// Entries names what the bundle covers, so a later reader can tell a
	// current bundle from one taken before half these credentials existed.
	Entries []RecoveryEntry `json:"entries"`
}

// Covers reports whether the receipt already includes an identity and field.
func (r RecoveryReceipt) Covers(identity Identity, field string) bool {
	for _, entry := range r.Entries {
		if entry.Identity == identity && entry.Field == field {
			return true
		}
	}
	return false
}

// WriteRecoveryReceipt records a successful export. A failure to record is
// deliberately not a failure to export: the bundle on disk is the thing that
// matters, and refusing to acknowledge a good backup because a note could not
// be written would be the wrong trade.
func WriteRecoveryReceipt(stateDir, bundlePath string, entries []RecoveryEntry, now time.Time) error {
	return WriteRecoveryReceiptWithMetadata(stateDir, bundlePath, entries, RecoveryReceipt{}, now)
}

// WriteRecoveryReceiptWithMetadata records a verified bundle and its
// metadata-only evidence. Callers invoke it only after read-back verification.
func WriteRecoveryReceiptWithMetadata(stateDir, bundlePath string, entries []RecoveryEntry, metadata RecoveryReceipt, now time.Time) error {
	receipt := metadata
	receipt.Path = bundlePath
	receipt.ExportedAt = now.UTC()
	receipt.Entries = append([]RecoveryEntry(nil), entries...)
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(stateDir, tuning.PermPrivateDir); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(stateDir, recoveryReceiptFile), encoded, tuning.PermSecret)
}

// ReadRecoveryReceipt returns the last recorded export. found is false when no
// bundle has ever been exported on this host, which is a normal answer for a
// fresh install and the exact condition worth reporting.
func ReadRecoveryReceipt(stateDir string) (RecoveryReceipt, bool, error) {
	data, err := os.ReadFile(filepath.Join(stateDir, recoveryReceiptFile))
	if err != nil {
		if os.IsNotExist(err) {
			return RecoveryReceipt{}, false, nil
		}
		return RecoveryReceipt{}, false, err
	}
	var receipt RecoveryReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return RecoveryReceipt{}, false, fmt.Errorf("read recovery receipt: %w", err)
	}
	return receipt, true, nil
}
