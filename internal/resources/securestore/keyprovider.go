package securestore

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/vrooli/vrooli/internal/credentialpolicy"
	"github.com/vrooli/vrooli/internal/shell"
)

// A key-encryption provider wraps the data key that seals every entry. Several
// wraps sit side by side in one file and any one of them opens the store, which
// is what lets Vrooli pick the strongest option a host supports without per-
// machine operator configuration — and lets a host that gains a TPM later add a
// host-bound wrap without re-encrypting a single stored value.

const (
	// providerHostBound needs no human at boot. It is the reason a headless
	// server or a Pi reaches a working state after a reboot on its own.
	providerHostBound = "host-bound"
	// providerPassphrase is the floor: every host has an operator with a
	// memory, even when it has no TPM and no systemd.
	providerPassphrase = "passphrase"
	providerNativeWrap = "native-wrap"

	// keyStoreTPM2 means the wrap is protected by the host's TPM, so possession
	// of the disk alone does not open it.
	keyStoreTPM2 = "tpm2"
	// keyStoreHostKey means systemd-creds fell back to a root-owned key on the
	// same disk. That is materially weaker — on a Pi, possession of the SD card
	// is sufficient — and is reported rather than averaged into one number.
	keyStoreHostKey = "host-key"
	// keyStorePassphrase means the wrap opens only with what the operator
	// remembers.
	keyStorePassphrase = "operator-passphrase"
	keyStoreKeychain   = "keychain"
	keyStoreDPAPI      = "dpapi"

	pbkdf2Iterations = credentialpolicy.RecoveryPBKDF2Iterations
	pbkdf2SaltLen    = 32

	// systemdCredsName binds a host-bound wrap to this purpose. systemd-creds
	// refuses to decrypt a blob under a different name, so a Vrooli data key
	// cannot be opened by anything that asks for another credential.
	systemdCredsName = "vrooli.securestore.datakey"
)

var (
	// errKeyProviderUnavailable means this host cannot use the provider right
	// now — no systemd-creds binary, no TPM access, no passphrase supplied.
	errKeyProviderUnavailable = errors.New("credential key provider is unavailable on this host")
	// errWrongPassphrase is separated from corruption on purpose: the operator
	// action is to try again, not to conclude the store is destroyed.
	errWrongPassphrase = errors.New("credential store passphrase did not open the data key")
)

// keyProvider wraps and unwraps the data key. Availability is a probe, not a
// declaration: a systemd-creds binary that exists but cannot reach the TPM or
// the host secret is not an available provider, and claiming otherwise would
// produce a store that initializes and then never opens.
type keyProvider interface {
	// Name is the stable provider identifier recorded in the file.
	Name() string
	// Available reports the key store this provider would actually use, or an
	// error explaining why it cannot be used on this host.
	Available() (string, error)
	// Wrap seals the data key. It returns the complete wrappedKey record,
	// including the key store it used.
	Wrap(dataKey []byte) (wrappedKey, error)
	// Unwrap recovers the data key from this provider's wrap.
	Unwrap(wrap wrappedKey) ([]byte, error)
}

// newDataKey generates the single random key that seals every entry.
func newDataKey() ([]byte, error) {
	key := make([]byte, dataKeyLen)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate credential data key: %w", err)
	}
	return key, nil
}

// wrapAAD binds a wrap to its provider name, so a wrap cannot be relabelled as
// another provider's and fed to the wrong unwrap path.
func wrapAAD(provider string) []byte {
	aad := appendLengthPrefixed(nil, []byte(aeadContext+".wrap"))
	return appendLengthPrefixed(aad, []byte(provider))
}

// sealDataKey and openDataKey are the shared AES-256-GCM envelope both
// providers use once they have derived or obtained a key-encryption key. Only
// the way the KEK is obtained differs between them.
func sealDataKey(kek, dataKey []byte, provider string) (string, error) {
	envelope, err := credentialpolicy.Seal(kek, dataKey, "key-wrap/"+provider, sealedFormatVersion)
	if err != nil {
		return "", err
	}
	sealed := append(append([]byte(nil), envelope.Nonce...), envelope.Ciphertext...)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func openDataKey(kek []byte, ciphertext, provider string, mismatch error) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: wrapped data key is not valid base64", errSealedCorrupt)
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(raw) < gcm.NonceSize() {
		return nil, fmt.Errorf("%w: wrapped data key is too short to hold a nonce", errSealedCorrupt)
	}
	nonce, rawCiphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	dataKey, modernErr := credentialpolicy.Open(kek, credentialpolicy.Envelope{
		Version: sealedFormatVersion, Purpose: "key-wrap/" + provider, Nonce: nonce, Ciphertext: rawCiphertext,
	})
	if modernErr != nil {
		// Historical store files used wrapAAD directly. Keep this reader for
		// existing hosts; all new wraps use the common authenticated framing.
		dataKey, err = gcm.Open(nil, nonce, rawCiphertext, wrapAAD(provider))
		if err != nil {
			return nil, mismatch
		}
	}
	if len(dataKey) != dataKeyLen {
		return nil, fmt.Errorf("%w: unwrapped data key is %d bytes, want %d", errSealedCorrupt, len(dataKey), dataKeyLen)
	}
	return dataKey, nil
}

// passphraseParams is the public KDF material stored beside the wrap. It holds
// no key material, which is why it can sit in cleartext next to the ciphertext.
type passphraseParams struct {
	KDF        string `json:"kdf"`
	Iterations int    `json:"iterations"`
	Salt       string `json:"salt"`
	// Generation is a non-secret, monotonic identifier for passphrase
	// replacement. Older stores omitted it; those are treated as generation 1
	// and acquire an explicit counter on the next passphrase change.
	Generation uint64 `json:"generation,omitempty"`
}

// nativeWrapProvider keeps the encrypted-file data key inside the platform's
// native per-user secret facility. It is deliberately separate from the
// native credential value store: this provider lets a host retain one
// encrypted-file authority while still opening it unattended after reboot.
type nativeWrapProvider struct{}

func newNativeWrapProvider() nativeWrapProvider { return nativeWrapProvider{} }
func (nativeWrapProvider) Name() string         { return providerNativeWrap }

func (nativeWrapProvider) Available() (string, error) {
	keyStore, err := nativeWrapAvailable()
	if err != nil {
		return "", err
	}
	const canary = "vrooli-native-wrap-probe"
	sealed, err := nativeWrapProtect([]byte(canary))
	if err != nil {
		return "", err
	}
	opened, err := nativeWrapUnprotect(sealed)
	if err != nil {
		return "", err
	}
	if !bytes.Equal(opened, []byte(canary)) {
		return "", fmt.Errorf("%w: native wrap round-trip did not return its canary", errKeyProviderUnavailable)
	}
	return keyStore, nil
}

func (nativeWrapProvider) Wrap(dataKey []byte) (wrappedKey, error) {
	keyStore, err := nativeWrapAvailable()
	if err != nil {
		return wrappedKey{}, err
	}
	ciphertext, err := nativeWrapProtect(dataKey)
	if err != nil {
		return wrappedKey{}, err
	}
	return wrappedKey{Provider: providerNativeWrap, KeyStore: keyStore, Ciphertext: base64.StdEncoding.EncodeToString(ciphertext)}, nil
}

func (nativeWrapProvider) Unwrap(wrap wrappedKey) ([]byte, error) {
	if err := checkWrapProvider(wrap, providerNativeWrap); err != nil {
		return nil, err
	}
	if _, err := nativeWrapAvailable(); err != nil {
		return nil, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(wrap.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: native wrap is not valid base64", errSealedCorrupt)
	}
	dataKey, err := nativeWrapUnprotect(ciphertext)
	if err != nil {
		return nil, err
	}
	if len(dataKey) != dataKeyLen {
		return nil, fmt.Errorf("%w: native wrap returned %d bytes, want %d", errSealedCorrupt, len(dataKey), dataKeyLen)
	}
	return dataKey, nil
}

// passphraseProvider derives the key-encryption key from what the operator
// remembers. It is the only provider that works on every host, which is why it
// is the fallback rather than the exception.
//
// The passphrase is resolved at use rather than at construction. Default()
// builds the store long before the CLI has read a passphrase from stdin or an
// unlock has populated the session cache, so a value captured at construction
// would always be the empty one.
type passphraseProvider struct {
	passphrase string
	source     func() string
	generation uint64
}

func (passphraseProvider) Name() string { return providerPassphrase }

// secret is the single resolution point for the passphrase.
func (provider passphraseProvider) secret() string {
	if provider.passphrase != "" {
		return provider.passphrase
	}
	if provider.source != nil {
		return provider.source()
	}
	return ""
}

func (provider passphraseProvider) Available() (string, error) {
	if strings.TrimSpace(provider.secret()) == "" {
		return "", fmt.Errorf("%w: no passphrase was supplied", errKeyProviderUnavailable)
	}
	return keyStorePassphrase, nil
}

func (provider passphraseProvider) Wrap(dataKey []byte) (wrappedKey, error) {
	if _, err := provider.Available(); err != nil {
		return wrappedKey{}, err
	}
	salt := make([]byte, pbkdf2SaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return wrappedKey{}, fmt.Errorf("generate passphrase salt: %w", err)
	}
	kek, err := pbkdf2.Key(sha256.New, provider.secret(), salt, pbkdf2Iterations, dataKeyLen)
	if err != nil {
		return wrappedKey{}, err
	}
	ciphertext, err := sealDataKey(kek, dataKey, providerPassphrase)
	if err != nil {
		return wrappedKey{}, err
	}
	params, err := json.Marshal(passphraseParams{
		KDF:        "pbkdf2-sha256",
		Iterations: pbkdf2Iterations,
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Generation: normalizedPassphraseGeneration(provider.generation),
	})
	if err != nil {
		return wrappedKey{}, err
	}
	return wrappedKey{
		Provider:   providerPassphrase,
		KeyStore:   keyStorePassphrase,
		Params:     params,
		Ciphertext: ciphertext,
	}, nil
}

func normalizedPassphraseGeneration(generation uint64) uint64 {
	if generation == 0 {
		return 1
	}
	return generation
}

// checkWrapProvider refuses a wrap that belongs to another provider. The AEAD
// additional data already stops a relabelled wrap from decrypting under the
// wrong key, but that only fires after the expensive KDF and only if the
// ciphertext is even reachable. Checking the label first makes the recorded
// provider name authoritative rather than decorative.
func checkWrapProvider(wrap wrappedKey, name string) error {
	if wrap.Provider != name {
		return fmt.Errorf("%w: wrap is labelled %q and was handed to the %q provider",
			errSealedCorrupt, wrap.Provider, name)
	}
	return nil
}

func (provider passphraseProvider) Unwrap(wrap wrappedKey) ([]byte, error) {
	if err := checkWrapProvider(wrap, providerPassphrase); err != nil {
		return nil, err
	}
	if _, err := provider.Available(); err != nil {
		return nil, err
	}
	var params passphraseParams
	if err := json.Unmarshal(wrap.Params, &params); err != nil {
		return nil, fmt.Errorf("%w: passphrase wrap has unreadable parameters", errSealedCorrupt)
	}
	salt, err := base64.StdEncoding.DecodeString(params.Salt)
	if err != nil || len(salt) == 0 {
		return nil, fmt.Errorf("%w: passphrase wrap has no usable salt", errSealedCorrupt)
	}
	// The iteration count is read from the file rather than assumed, so a store
	// written under an older policy still opens. It is floored at the current
	// policy's value on write, never on read.
	if params.Iterations <= 0 {
		return nil, fmt.Errorf("%w: passphrase wrap declares no iteration count", errSealedCorrupt)
	}
	kek, err := pbkdf2.Key(sha256.New, provider.secret(), salt, params.Iterations, dataKeyLen)
	if err != nil {
		return nil, err
	}
	return openDataKey(kek, wrap.Ciphertext, providerPassphrase, errWrongPassphrase)
}

// hostBoundProvider wraps the data key through systemd-creds, which uses the
// TPM when one is reachable and a root-owned host key otherwise. It needs no
// human at boot, which is the entire reason a headless host can reach a working
// state on its own.
//
// Its strength is not uniform and must not be presented as though it were. With
// a TPM, disk theft does not open the wrap. Without one, systemd-creds falls
// back to a key on the same disk, so possession of the disk is enough. The
// active key store travels with the wrap so `doctor` can say which it is.
type hostBoundProvider struct {
	// run executes systemd-creds. It is a field so the provider can be tested
	// against a host that has no systemd-creds, and against one whose TPM is
	// unreachable, neither of which can be produced on demand on a real
	// machine.
	run func(args []string, stdin []byte) ([]byte, error)
}

func newHostBoundProvider() hostBoundProvider {
	return hostBoundProvider{run: runSystemdCreds}
}

func (hostBoundProvider) Name() string { return providerHostBound }

// runSystemdCreds invokes the binary with the payload on stdin and the result
// on stdout. Neither the data key nor the wrapped key ever enters argv, so
// neither appears in /proc, in a process listing, or in an audit log.
func runSystemdCreds(args []string, stdin []byte) ([]byte, error) {
	if _, err := exec.LookPath("systemd-creds"); err != nil {
		return nil, fmt.Errorf("%w: systemd-creds is not installed", errKeyProviderUnavailable)
	}
	name, argv := withDeviceGroup("systemd-creds", args)
	cmd := shell.NewCommand(name, argv...)
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return nil, fmt.Errorf("%w: systemd-creds %s: %s", errKeyProviderUnavailable, args[0], detail)
	}
	return stdout.Bytes(), nil
}

// withDeviceGroup re-enters a TPM command under a group this account holds and
// this process cannot use, and is a no-op whenever there is nothing to pick up.
//
// It sits at the one place that touches the TPM rather than at any single
// caller, because every caller has the same problem and none of them can solve
// it for itself: supplementary groups are attached by the kernel at login, so a
// process that was already running when `vrooli setup` granted the group can
// never see the grant, no matter which entry point it was reached through. The
// case that forces this is the ordinary one — vrooli-onboarding is a
// long-running scenario, started before setup ran, and it is where the operator
// types the passphrase. Without this, the wrap that removes the need for that
// passphrase could not be added by the very command the operator supplied it
// to, and the host would go on asking at every boot until someone happened to
// log out.
//
// No credential material passes through the shell: the data key and the wrapped
// key travel on standard input, and only the non-secret systemd-creds arguments
// are quoted into the command string.
func withDeviceGroup(name string, args []string) (string, []string) {
	group := PendingGroupGrant()
	if group == "" {
		return name, args
	}
	sg, err := exec.LookPath("sg")
	if err != nil {
		return name, args
	}
	quoted := make([]string, 0, len(args)+1)
	for _, part := range append([]string{name}, args...) {
		quoted = append(quoted, shellQuoteArgument(part))
	}
	return sg, []string{group, "-c", strings.Join(quoted, " ")}
}

// shellQuoteArgument wraps a value so a shell reproduces it exactly, including
// one that contains a quote of its own.
func shellQuoteArgument(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// hostBoundKeyModes are tried strongest first. `tpm2-absent` is deliberately
// absent from this list: systemd-creds accepts it and encrypts under a null
// key, announcing that it provides neither confidentiality nor authenticity.
// Writing a data key that way would satisfy every structural check in this
// package while protecting nothing.
var hostBoundKeyModes = []struct {
	flag     string
	keyStore string
}{
	{flag: hostBoundTPM2Mode, keyStore: keyStoreTPM2},
	{flag: "host", keyStore: keyStoreHostKey},
}

// hostBoundTPM2Mode is the systemd-creds key mode that seals to the TPM.
const hostBoundTPM2Mode = "tpm2"

// tpm2PCRPolicy is the PCR set the host-bound wrap binds to, and it is
// deliberately empty. Passing the flag explicitly is the point: systemd-creds
// binds to PCR 7 when it is omitted.
//
// PCR 7 measures Secure Boot policy — the signature databases, their revocation
// list, and the certificate that signed the running bootloader. Distributions
// ship dbx revocation updates through fwupd as ordinary security updates, and
// any of them changes PCR 7. A wrap bound to it stops opening after a routine
// update, with no failed action to attribute it to: the operator simply finds
// themselves typing a passphrase at boot again, which is the exact outcome this
// provider exists to remove. A mechanism that silently reverts to the thing it
// replaced is worse than one that was never installed, because the operator
// stops checking.
//
// Binding to no PCRs makes the wrap mean "this TPM, on this machine". What is
// given up is resistance to someone with physical possession who boots another
// OS on the same board. What is kept is resistance to theft of the disk, the
// store file, or a backup containing it — and the store file is designed to
// travel in backups, so that is the threat this key actually faces. It is also
// the same guarantee the macOS Keychain and Windows DPAPI wraps give, which
// keeps one promise across the three platforms instead of three different ones.
//
// An operator who wants PCR binding has a stronger option than this flag: seal
// the whole disk with systemd-cryptenroll, where a broken policy fails at boot
// in front of a human rather than silently at first credential read.
const tpm2PCRPolicy = ""

func (provider hostBoundProvider) encrypt(mode string, plaintext []byte) ([]byte, error) {
	args := []string{
		"encrypt",
		"--name=" + systemdCredsName,
		"--with-key=" + mode,
	}
	if mode == hostBoundTPM2Mode {
		args = append(args, "--tpm2-pcrs="+tpm2PCRPolicy)
	}
	return provider.run(append(args, "-", "-"), plaintext)
}

func (provider hostBoundProvider) decrypt(ciphertext []byte) ([]byte, error) {
	return provider.run([]string{
		"decrypt",
		"--name=" + systemdCredsName,
		"-", "-",
	}, ciphertext)
}

// Available proves the provider works rather than assuming it from the presence
// of a binary. A full encrypt-then-decrypt cycle over a throwaway value is the
// only honest test: on this host the binary exists, /dev/tpm0 exists, and an
// unprivileged process still cannot use either the TPM resource manager or the
// root-owned host secret.
func (provider hostBoundProvider) Available() (string, error) {
	keyStore, _, err := provider.probe()
	return keyStore, err
}

// probe returns the strongest working key mode and the round-trip proof.
func (provider hostBoundProvider) probe() (string, string, error) {
	const canary = "vrooli-host-bound-probe"
	var reasons []string
	for _, mode := range hostBoundKeyModes {
		sealed, err := provider.encrypt(mode.flag, []byte(canary))
		if err != nil {
			reasons = append(reasons, mode.flag+": "+conciseReason(err))
			continue
		}
		opened, err := provider.decrypt(sealed)
		if err != nil {
			reasons = append(reasons, mode.flag+": "+conciseReason(err))
			continue
		}
		if subtle.ConstantTimeCompare(opened, []byte(canary)) != 1 {
			reasons = append(reasons, mode.flag+": round-trip did not return what it was given")
			continue
		}
		return mode.keyStore, mode.flag, nil
	}
	// The reasons say what failed; the fix says what to do about it. Without
	// the second half a "Permission denied" from the TPM reads as a host that
	// cannot do this, when it is usually a host one group membership away.
	detail := strings.Join(reasons, "; ")
	if fix := hostBoundFix(); fix != "" {
		detail += " — " + fix
	}
	return "", "", fmt.Errorf("%w: systemd-creds cannot protect a key on this host (%s)",
		errKeyProviderUnavailable, detail)
}

// conciseReason keeps a provider explanation to one line so a diagnosis stays
// readable. It never carries key material, because runSystemdCreds only ever
// reports stderr and systemd-creds writes results to stdout.
func conciseReason(err error) string {
	message := err.Error()
	if index := strings.Index(message, "\n"); index >= 0 {
		message = message[:index]
	}
	if prefix := errKeyProviderUnavailable.Error() + ": "; strings.HasPrefix(message, prefix) {
		message = strings.TrimPrefix(message, prefix)
	}
	return strings.TrimSpace(message)
}

func (provider hostBoundProvider) Wrap(dataKey []byte) (wrappedKey, error) {
	keyStore, mode, err := provider.probe()
	if err != nil {
		return wrappedKey{}, err
	}
	sealed, err := provider.encrypt(mode, dataKey)
	if err != nil {
		return wrappedKey{}, err
	}
	return wrappedKey{
		Provider: providerHostBound,
		KeyStore: keyStore,
		// systemd-creds emits its own base64 text. It is stored verbatim so the
		// blob Vrooli hands back on decrypt is byte-identical to what
		// systemd-creds produced.
		Ciphertext: strings.TrimSpace(string(sealed)),
	}, nil
}

func (provider hostBoundProvider) Unwrap(wrap wrappedKey) ([]byte, error) {
	if err := checkWrapProvider(wrap, providerHostBound); err != nil {
		return nil, err
	}
	dataKey, err := provider.decrypt([]byte(wrap.Ciphertext + "\n"))
	if err != nil {
		return nil, err
	}
	if len(dataKey) != dataKeyLen {
		return nil, fmt.Errorf("%w: host-bound wrap returned %d bytes, want %d", errSealedCorrupt, len(dataKey), dataKeyLen)
	}
	return dataKey, nil
}
