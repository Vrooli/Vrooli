package securestore

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/credentialpolicy"
)

func TestPassphraseProviderRoundTripsTheDataKey(t *testing.T) {
	dataKey := testDataKey(t)
	provider := passphraseProvider{passphrase: "correct horse battery staple"}

	keyStore, err := provider.Available()
	if err != nil {
		t.Fatalf("Available: %v", err)
	}
	if keyStore != keyStorePassphrase {
		t.Fatalf("Available key store = %q, want %q", keyStore, keyStorePassphrase)
	}

	wrap, err := provider.Wrap(dataKey)
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}
	if wrap.Provider != providerPassphrase || wrap.KeyStore != keyStorePassphrase {
		t.Fatalf("wrap identity = %q/%q", wrap.Provider, wrap.KeyStore)
	}
	if strings.Contains(wrap.Ciphertext, base64.StdEncoding.EncodeToString(dataKey)) {
		t.Fatalf("the wrap contains the unwrapped data key")
	}

	opened, err := provider.Unwrap(wrap)
	if err != nil {
		t.Fatalf("Unwrap: %v", err)
	}
	if !bytes.Equal(opened, dataKey) {
		t.Fatalf("Unwrap returned a different data key")
	}
}

func TestHistoricalKeyWrapFixtureOpensThroughCompatibilityReader(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "credentialpolicy", "testdata", "historical-envelopes-v1.json")
	fixtureBytes, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read historical credential fixture: %v", err)
	}
	var fixture struct {
		Wrap struct {
			Provider   string `json:"provider"`
			Nonce      string `json:"nonce"`
			Ciphertext string `json:"ciphertext"`
			KEK        string `json:"kek"`
		} `json:"key_wrap_v1"`
	}
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		t.Fatalf("decode historical credential fixture: %v", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(fixture.Wrap.Nonce)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(fixture.Wrap.Ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	kek, err := base64.StdEncoding.DecodeString(fixture.Wrap.KEK)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := openDataKey(kek, base64.StdEncoding.EncodeToString(append(nonce, ciphertext...)), fixture.Wrap.Provider, errWrongPassphrase)
	if err != nil {
		t.Fatalf("historical key-wrap fixture failed to open: %v", err)
	}
	if !bytes.Equal(opened, bytes.Repeat([]byte{0x22}, dataKeyLen)) {
		t.Fatalf("historical key-wrap fixture returned unexpected data key")
	}
}

// TestPassphraseProviderMatchesTheRecoveryKDFPolicy keeps this package from
// drifting into a second KDF policy. internal/credentialauthority/recovery.go uses
// PBKDF2-SHA256 at 600,000 iterations; two different answers to "how hard is a
// passphrase to grind" is a policy nobody can reason about.
func TestPassphraseProviderMatchesTheRecoveryKDFPolicy(t *testing.T) {
	if pbkdf2Iterations != credentialpolicy.RecoveryPBKDF2Iterations {
		t.Fatalf("pbkdf2Iterations = %d, want %d from shared credential policy", pbkdf2Iterations, credentialpolicy.RecoveryPBKDF2Iterations)
	}
	wrap, err := passphraseProvider{passphrase: "pass"}.Wrap(testDataKey(t))
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}
	var params passphraseParams
	if err := json.Unmarshal(wrap.Params, &params); err != nil {
		t.Fatalf("decode wrap params: %v", err)
	}
	if params.KDF != "pbkdf2-sha256" || params.Iterations != pbkdf2Iterations {
		t.Fatalf("recorded KDF policy = %+v", params)
	}
	if salt, err := base64.StdEncoding.DecodeString(params.Salt); err != nil || len(salt) != pbkdf2SaltLen {
		t.Fatalf("recorded salt is %d bytes (err %v), want %d", len(salt), err, pbkdf2SaltLen)
	}
}

// TestPassphraseProviderSeparatesAWrongPassphraseFromCorruption is the operator
// distinction: "try again" and "this store is damaged" are different situations
// and must not produce the same message.
func TestPassphraseProviderSeparatesAWrongPassphraseFromCorruption(t *testing.T) {
	wrap, err := passphraseProvider{passphrase: "the right one"}.Wrap(testDataKey(t))
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}

	_, err = passphraseProvider{passphrase: "the wrong one"}.Unwrap(wrap)
	if !errors.Is(err, errWrongPassphrase) {
		t.Fatalf("Unwrap with a wrong passphrase = %v, want errWrongPassphrase", err)
	}
	if errors.Is(err, errSealedCorrupt) {
		t.Fatalf("a wrong passphrase was reported as a damaged store: %v", err)
	}

	damaged := wrap
	damaged.Params = json.RawMessage(`{"kdf":"pbkdf2-sha256","iterations":600000,"salt":"not base64!!"}`)
	if _, err := (passphraseProvider{passphrase: "the right one"}).Unwrap(damaged); !errors.Is(err, errSealedCorrupt) {
		t.Fatalf("Unwrap with a damaged salt = %v, want errSealedCorrupt", err)
	}
}

func TestPassphraseProviderIsUnavailableWithoutAPassphrase(t *testing.T) {
	provider := passphraseProvider{passphrase: "   "}
	if _, err := provider.Available(); !errors.Is(err, errKeyProviderUnavailable) {
		t.Fatalf("Available without a passphrase = %v, want errKeyProviderUnavailable", err)
	}
	if _, err := provider.Wrap(testDataKey(t)); !errors.Is(err, errKeyProviderUnavailable) {
		t.Fatalf("Wrap without a passphrase = %v, want errKeyProviderUnavailable", err)
	}
}

// fakeSystemdCreds stands in for the binary so the tests can produce host
// conditions that cannot be arranged on demand: a machine with no
// systemd-creds, one whose TPM is unreachable, one where nothing works.
type fakeSystemdCreds struct {
	// working is the set of --with-key modes this fake host supports.
	working map[string]bool
	// calls records every argv, so a test can assert which key modes were
	// attempted and which were never touched.
	calls [][]string
	// blobs maps a returned ciphertext back to its plaintext.
	blobs map[string][]byte
}

func newFakeSystemdCreds(modes ...string) *fakeSystemdCreds {
	fake := &fakeSystemdCreds{working: map[string]bool{}, blobs: map[string][]byte{}}
	for _, mode := range modes {
		fake.working[mode] = true
	}
	return fake
}

func (fake *fakeSystemdCreds) run(args []string, stdin []byte) ([]byte, error) {
	fake.calls = append(fake.calls, args)
	switch args[0] {
	case "encrypt":
		mode := ""
		for _, arg := range args {
			if strings.HasPrefix(arg, "--with-key=") {
				mode = strings.TrimPrefix(arg, "--with-key=")
			}
		}
		if !fake.working[mode] {
			return nil, fmt.Errorf("%w: systemd-creds encrypt: mode %q unavailable", errKeyProviderUnavailable, mode)
		}
		blob := base64.StdEncoding.EncodeToString(stdin) + "." + mode
		fake.blobs[blob] = append([]byte(nil), stdin...)
		return []byte(blob + "\n"), nil
	case "decrypt":
		blob := strings.TrimSpace(string(stdin))
		plaintext, ok := fake.blobs[blob]
		if !ok {
			return nil, fmt.Errorf("%w: systemd-creds decrypt: unknown blob", errKeyProviderUnavailable)
		}
		return plaintext, nil
	default:
		return nil, fmt.Errorf("unexpected systemd-creds verb %q", args[0])
	}
}

func (fake *fakeSystemdCreds) modesTried() []string {
	modes := []string{}
	for _, call := range fake.calls {
		for _, arg := range call {
			if strings.HasPrefix(arg, "--with-key=") {
				modes = append(modes, strings.TrimPrefix(arg, "--with-key="))
			}
		}
	}
	return modes
}

func TestHostBoundProviderPrefersTheTPMAndReportsIt(t *testing.T) {
	fake := newFakeSystemdCreds("tpm2", "host")
	provider := hostBoundProvider{run: fake.run}

	keyStore, err := provider.Available()
	if err != nil {
		t.Fatalf("Available: %v", err)
	}
	if keyStore != keyStoreTPM2 {
		t.Fatalf("Available key store = %q, want %q on a host whose TPM works", keyStore, keyStoreTPM2)
	}

	dataKey := testDataKey(t)
	wrap, err := provider.Wrap(dataKey)
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}
	if wrap.KeyStore != keyStoreTPM2 {
		t.Fatalf("wrap key store = %q, want %q", wrap.KeyStore, keyStoreTPM2)
	}
	opened, err := provider.Unwrap(wrap)
	if err != nil {
		t.Fatalf("Unwrap: %v", err)
	}
	if !bytes.Equal(opened, dataKey) {
		t.Fatalf("Unwrap returned a different data key")
	}
}

func TestHostBoundProviderExplicitlyDisablesPCRBinding(t *testing.T) {
	fake := newFakeSystemdCreds("tpm2")
	provider := hostBoundProvider{run: fake.run}
	if _, err := provider.Available(); err != nil {
		t.Fatal(err)
	}
	for _, call := range fake.calls {
		if len(call) == 0 || call[0] != "encrypt" {
			continue
		}
		found := false
		for _, arg := range call {
			if arg == "--tpm2-pcrs=" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("TPM encrypt argv = %v, want explicit empty --tpm2-pcrs=", call)
		}
		return
	}
	t.Fatal("no TPM encrypt call recorded")
}

// TestHostBoundProviderReportsTheWeakerHostKeyHonestly is the disclosure the
// plan requires. On hardware with no TPM, systemd-creds protects the wrap with a
// key on the same disk, so possession of the disk is enough. Reporting that as
// though it were TPM protection would tell an operator their Pi's SD card is
// safe when it is not.
func TestHostBoundProviderReportsTheWeakerHostKeyHonestly(t *testing.T) {
	fake := newFakeSystemdCreds("host")
	provider := hostBoundProvider{run: fake.run}

	keyStore, err := provider.Available()
	if err != nil {
		t.Fatalf("Available: %v", err)
	}
	if keyStore != keyStoreHostKey {
		t.Fatalf("Available key store = %q, want %q on a host with no usable TPM", keyStore, keyStoreHostKey)
	}
	wrap, err := provider.Wrap(testDataKey(t))
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}
	if wrap.KeyStore != keyStoreHostKey {
		t.Fatalf("wrap key store = %q, want %q", wrap.KeyStore, keyStoreHostKey)
	}
	if tried := fake.modesTried(); tried[0] != "tpm2" {
		t.Fatalf("the TPM was not tried first: %v", tried)
	}
}

// TestHostBoundProviderNeverUsesTheNullKey is a security assertion, not a
// coverage one. systemd-creds accepts --with-key=tpm2-absent and encrypts under
// a null key while announcing that it provides neither confidentiality nor
// authenticity. A data key written that way would pass every structural check in
// this package and protect nothing.
func TestHostBoundProviderNeverUsesTheNullKey(t *testing.T) {
	fake := newFakeSystemdCreds("tpm2-absent", "null")
	provider := hostBoundProvider{run: fake.run}

	if keyStore, err := provider.Available(); err == nil {
		t.Fatalf("Available accepted a host where only the null key works: %q", keyStore)
	}
	if _, err := provider.Wrap(testDataKey(t)); !errors.Is(err, errKeyProviderUnavailable) {
		t.Fatalf("Wrap on a null-key-only host = %v, want errKeyProviderUnavailable", err)
	}
	for _, mode := range fake.modesTried() {
		if strings.Contains(mode, "absent") || mode == "null" {
			t.Fatalf("the provider attempted the null key mode %q", mode)
		}
	}
	for _, mode := range hostBoundKeyModes {
		if strings.Contains(mode.flag, "absent") {
			t.Fatalf("hostBoundKeyModes offers the null key mode %q", mode.flag)
		}
	}
}

// TestHostBoundProviderIsUnavailableWithoutTheBinary is the detection the plan
// calls for: the provider must report false on a host with no systemd-creds
// rather than failing later, when a credential is actually needed.
func TestHostBoundProviderIsUnavailableWithoutTheBinary(t *testing.T) {
	provider := hostBoundProvider{run: func([]string, []byte) ([]byte, error) {
		return nil, fmt.Errorf("%w: systemd-creds is not installed", errKeyProviderUnavailable)
	}}
	keyStore, err := provider.Available()
	if !errors.Is(err, errKeyProviderUnavailable) {
		t.Fatalf("Available without the binary = %q, %v; want errKeyProviderUnavailable", keyStore, err)
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("the explanation does not name the missing binary: %v", err)
	}
}

// TestHostBoundProviderRejectsATamperedRoundTrip covers the host where
// systemd-creds answers but does not return what it was given. Treating that as
// available would initialize a store nobody could ever open.
func TestHostBoundProviderRejectsATamperedRoundTrip(t *testing.T) {
	provider := hostBoundProvider{run: func(args []string, stdin []byte) ([]byte, error) {
		if args[0] == "encrypt" {
			return []byte("blob\n"), nil
		}
		return []byte("something else entirely"), nil
	}}
	if _, err := provider.Available(); !errors.Is(err, errKeyProviderUnavailable) {
		t.Fatalf("Available with a lying backend = %v, want errKeyProviderUnavailable", err)
	}
}

// TestWrapsAreProviderBound stops a wrap being relabelled and fed to the wrong
// unwrap path, which is the cheapest way to attack a file holding several wraps
// of one key.
func TestWrapsAreProviderBound(t *testing.T) {
	dataKey := testDataKey(t)
	fake := newFakeSystemdCreds("tpm2")
	hostBound := hostBoundProvider{run: fake.run}
	passphrase := passphraseProvider{passphrase: "operator passphrase"}

	hostWrap, err := hostBound.Wrap(dataKey)
	if err != nil {
		t.Fatalf("host-bound Wrap: %v", err)
	}
	passWrap, err := passphrase.Wrap(dataKey)
	if err != nil {
		t.Fatalf("passphrase Wrap: %v", err)
	}

	// Each wrap opens under its own provider and returns the same data key —
	// that is what makes a second wrap free of re-encryption.
	fromHost, err := hostBound.Unwrap(hostWrap)
	if err != nil {
		t.Fatalf("host-bound Unwrap: %v", err)
	}
	fromPass, err := passphrase.Unwrap(passWrap)
	if err != nil {
		t.Fatalf("passphrase Unwrap: %v", err)
	}
	if !bytes.Equal(fromHost, fromPass) || !bytes.Equal(fromHost, dataKey) {
		t.Fatalf("the two wraps do not open the same data key")
	}

	// Neither provider opens the other's wrap.
	if _, err := passphrase.Unwrap(hostWrap); err == nil {
		t.Fatalf("the passphrase provider opened a host-bound wrap")
	}
	if _, err := hostBound.Unwrap(passWrap); err == nil {
		t.Fatalf("the host-bound provider opened a passphrase wrap")
	}

	// A passphrase wrap relabelled as another provider must not open, because
	// the provider name is bound into the AEAD additional data.
	relabelled := passWrap
	relabelled.Provider = providerHostBound
	if _, err := (passphraseProvider{passphrase: "operator passphrase"}).Unwrap(relabelled); err == nil {
		t.Fatalf("a relabelled wrap opened anyway")
	}
}

// TestHostBoundProviderOnThisHost runs the real binary. It asserts the one
// property that must hold on every machine: the provider either names a key
// store it actually proved, or explains why it cannot be used. What it must
// never do is claim availability it has not demonstrated.
func TestHostBoundProviderOnThisHost(t *testing.T) {
	provider := newHostBoundProvider()
	keyStore, err := provider.Available()
	if err != nil {
		if !errors.Is(err, errKeyProviderUnavailable) {
			t.Fatalf("Available failed with an unclassified error: %v", err)
		}
		if strings.TrimSpace(conciseReason(err)) == "" {
			t.Fatalf("the provider reported unavailable with no reason")
		}
		if _, lookErr := exec.LookPath("systemd-creds"); lookErr != nil {
			t.Skipf("no systemd-creds on this runner — %v", err)
		}
		// A host that has the binary and still cannot use it is the common
		// unprivileged case: /dev/tpmrm0 belongs to the tss group and the
		// host secret is root-only. The passphrase wrap covers it.
		t.Logf("systemd-creds is installed but unusable as this user, so the passphrase wrap is the fallback: %v", err)
		return
	}
	if keyStore != keyStoreTPM2 && keyStore != keyStoreHostKey {
		t.Fatalf("Available named an unknown key store %q", keyStore)
	}
	t.Logf("this host's host-bound wrap uses %q", keyStore)

	dataKey := testDataKey(t)
	wrap, err := provider.Wrap(dataKey)
	if err != nil {
		t.Fatalf("Wrap on a host that reported available: %v", err)
	}
	opened, err := provider.Unwrap(wrap)
	if err != nil {
		t.Fatalf("Unwrap on a host that reported available: %v", err)
	}
	if !bytes.Equal(opened, dataKey) {
		t.Fatalf("the real systemd-creds round trip did not return the data key")
	}
}
