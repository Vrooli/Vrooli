package signing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	pathutil "scenario-to-desktop-api/shared/path"
	"scenario-to-desktop-api/signing/types"
)

// generateLinuxKeyParams holds the parameters for a GPG key generation request.
type generateLinuxKeyParams struct {
	Name           string
	Email          string
	Passphrase     string
	PassphraseEnv  string
	KeyType        string
	Expiry         string
	Homedir        string
	Force          bool
	ExportPublic   bool
	Scenario       string
	LogicalID      string
	WorkingDirRoot string
}

type generateLinuxKeyResult struct {
	Fingerprint string
	Homedir     string
	PublicKey   string
	PublicPath  string
	LogicalID   string
}

// generateLinuxKey ensures a passphrase-protected GPG key exists and retains
// only its public identifier in the scenario. The private key and passphrase
// are stored in the native credential authority, so no private material
// persists in the repository. An explicit logical ID is the custody identity;
// when that identity already holds key material, the existing key is reused so
// one publisher key can sign every desktop app. A caller-supplied homedir is
// treated as an external keyring and custody of that keyring remains the
// caller's responsibility.
func (h *Handler) generateLinuxKey(ctx context.Context, params generateLinuxKeyParams) (*generateLinuxKeyResult, error) {
	if _, err := exec.LookPath("gpg"); err != nil {
		return nil, fmt.Errorf("gpg is not installed: %w", err)
	}

	name := strings.TrimSpace(params.Name)
	email := strings.TrimSpace(params.Email)
	if name == "" && email == "" {
		return nil, fmt.Errorf("name or email is required to generate a key")
	}
	if strings.TrimSpace(params.Homedir) != "" && strings.TrimSpace(params.LogicalID) != "" {
		return nil, fmt.Errorf("homedir (external keyring) and logical-id (managed custody) are mutually exclusive")
	}

	homedir, managed, cleanup, err := resolveManagedHomedir(params)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// A managed key is idempotent per identity: when the identity already holds
	// key material, reuse it so many scenarios can share one publisher key.
	if managed {
		reused, ok, reuseErr := h.reuseManagedKey(ctx, params, homedir)
		if reuseErr != nil {
			return nil, reuseErr
		}
		if ok {
			return reused, nil
		}
	}

	uid := formatUID(name, email)
	keyType := valueOrDefault(params.KeyType, "rsa4096")
	expiry := valueOrDefault(params.Expiry, "1y")

	passphrase := params.Passphrase
	if passphrase == "" {
		passphrase, err = generatePassphrase()
		if err != nil {
			return nil, err
		}
	}

	if err := runGPGGenerate(ctx, homedir, uid, keyType, expiry, passphrase); err != nil {
		return nil, err
	}

	fpr, err := latestFingerprint(homedir)
	if err != nil {
		return nil, err
	}

	logicalID := ""
	returnedHomedir := homedir
	if managed {
		logicalID, err = storeManagedKey(ctx, homedir, fpr, params, passphrase)
		if err != nil {
			return nil, err
		}
		// The keyring is ephemeral; nothing durable remains on disk.
		returnedHomedir = ""
	}

	pub, pubPath, err := optionalExportPublicKey(ctx, homedir, fpr, params)
	if err != nil {
		return nil, err
	}

	return &generateLinuxKeyResult{
		Fingerprint: fpr,
		Homedir:     returnedHomedir,
		PublicKey:   pub,
		PublicPath:  pubPath,
		LogicalID:   logicalID,
	}, nil
}

// resolveManagedHomedir selects the GPG home for key generation. An empty
// homedir yields an ephemeral directory whose material is imported into the
// credential authority and then removed. A supplied homedir is preserved.
func resolveManagedHomedir(params generateLinuxKeyParams) (homedir string, managed bool, cleanup func(), err error) {
	cleanup = func() {}
	if strings.TrimSpace(params.Homedir) != "" {
		abs, absErr := filepath.Abs(params.Homedir)
		if absErr != nil {
			return "", false, cleanup, fmt.Errorf("resolve homedir: %w", absErr)
		}
		if mkErr := os.MkdirAll(abs, 0o700); mkErr != nil {
			return "", false, cleanup, fmt.Errorf("create homedir: %w", mkErr)
		}
		_ = os.Chmod(abs, 0o700)
		return abs, false, cleanup, nil
	}
	dir, tempErr := os.MkdirTemp("", "vrooli-signing-gnupg-")
	if tempErr != nil {
		return "", false, cleanup, fmt.Errorf("create signing homedir: %w", tempErr)
	}
	_ = os.Chmod(dir, 0o700)
	return dir, true, func() { _ = os.RemoveAll(dir) }, nil
}

// storeManagedKey retains the private key and passphrase in the credential
// authority and returns the logical identity that now holds them.
func storeManagedKey(ctx context.Context, homedir, fingerprint string, params generateLinuxKeyParams, passphrase string) (string, error) {
	identity, err := resolveManagedIdentity(params.Scenario, params.LogicalID)
	if err != nil {
		return "", err
	}
	authority, err := openCredentialAuthority()
	if err != nil {
		return "", fmt.Errorf("open credential authority: %w", err)
	}
	if err := authority.Availability(); err != nil {
		return "", fmt.Errorf("credential authority unavailable: %w", err)
	}
	if !params.Force && authority.Status(identity, types.DefaultPrivateKeyField).Configured {
		return "", fmt.Errorf("managed signing key %s already holds key material; reuse it from another scenario or pass force to rotate", identity)
	}

	secret, err := exportSecretKey(ctx, homedir, fingerprint, passphrase)
	if err != nil {
		return "", err
	}
	if err := authority.Put(identity, types.DefaultPrivateKeyField, secret); err != nil {
		return "", fmt.Errorf("store signing private key: %w", err)
	}
	if err := authority.Put(identity, types.DefaultPassphraseField, passphrase); err != nil {
		return "", fmt.Errorf("store signing passphrase: %w", err)
	}
	return string(identity), nil
}

// reuseManagedKey materializes key material already custodied under the target
// identity so the caller can report its fingerprint and publish its public
// half. It returns ok=false when there is nothing to reuse. It never creates a
// new key and never returns private material to the caller.
func (h *Handler) reuseManagedKey(ctx context.Context, params generateLinuxKeyParams, homedir string) (*generateLinuxKeyResult, bool, error) {
	identity, err := resolveManagedIdentity(params.Scenario, params.LogicalID)
	if err != nil {
		return nil, false, err
	}
	authority, err := openCredentialAuthority()
	if err != nil {
		return nil, false, fmt.Errorf("open credential authority: %w", err)
	}
	if err := authority.Availability(); err != nil {
		return nil, false, fmt.Errorf("credential authority unavailable: %w", err)
	}
	if params.Force || !authority.Status(identity, types.DefaultPrivateKeyField).Configured {
		return nil, false, nil
	}
	if !authority.Status(identity, types.DefaultPassphraseField).Configured {
		return nil, false, fmt.Errorf("managed signing key %s has no passphrase configured", identity)
	}

	secret, err := authority.Resolve(identity, types.DefaultPrivateKeyField)
	if err != nil {
		return nil, false, fmt.Errorf("resolve managed signing key: %w", err)
	}
	if err := importSecretKey(ctx, homedir, secret); err != nil {
		return nil, false, err
	}
	fingerprint, err := latestFingerprint(homedir)
	if err != nil {
		return nil, false, fmt.Errorf("read reused signing key: %w", err)
	}
	pub, pubPath, err := optionalExportPublicKey(ctx, homedir, fingerprint, params)
	if err != nil {
		return nil, false, err
	}
	return &generateLinuxKeyResult{
		Fingerprint: fingerprint,
		PublicKey:   pub,
		PublicPath:  pubPath,
		LogicalID:   string(identity),
	}, true, nil
}

// importSecretKey imports an ASCII-armored private key into a GPG home.
func importSecretKey(ctx context.Context, homedir, armored string) error {
	cmd := exec.CommandContext(ctx, "gpg", "--batch", "--homedir", homedir, "--import")
	cmd.Stdin = strings.NewReader(armored)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("import signing key: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// exportSecretKey exports an ASCII-armored private key for custody transfer.
// The key is passphrase-protected, so the export unlocks it with loopback
// pinentry; without this the agent blocks on a prompt and times out.
func exportSecretKey(ctx context.Context, homedir, fingerprint, passphrase string) (string, error) {
	cmd := exec.CommandContext(ctx, "gpg",
		"--batch",
		"--homedir", homedir,
		"--pinentry-mode", "loopback",
		"--passphrase-fd", "0",
		"--armor",
		"--export-secret-keys", fingerprint,
	)
	cmd.Stdin = strings.NewReader(passphrase)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("export secret key failed: %s", string(ee.Stderr))
		}
		return "", err
	}
	if strings.TrimSpace(string(out)) == "" {
		return "", fmt.Errorf("gpg returned an empty secret key export")
	}
	return string(out), nil
}

// generatePassphrase returns a fresh random passphrase for a generated key.
// The value is never returned to a caller or written to a scenario file.
func generatePassphrase() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate signing passphrase: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// formatUID builds the GPG UID string from name and email.
func formatUID(name, email string) string {
	if name != "" && email != "" {
		return fmt.Sprintf("%s <%s>", name, email)
	}
	if name != "" {
		return name
	}
	return email
}

// valueOrDefault returns val if non-empty, otherwise def.
func valueOrDefault(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

// runGPGGenerate executes the gpg quick-generate-key command.
func runGPGGenerate(ctx context.Context, absHomedir, uid, keyType, expiry, passphrase string) error {
	genArgs := []string{
		"--batch",
		"--homedir", absHomedir,
		"--pinentry-mode", "loopback",
		"--passphrase-fd", "0",
		"--quick-generate-key",
		uid,
		keyType,
		"sign",
		expiry,
	}
	genCmd := exec.CommandContext(ctx, "gpg", genArgs...)
	genCmd.Stdin = strings.NewReader(passphrase)
	if out, err := genCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gpg key generation failed: %v: %s", err, string(out))
	}
	return nil
}

// optionalExportPublicKey exports the public key if requested.
func optionalExportPublicKey(ctx context.Context, absHomedir, fpr string, params generateLinuxKeyParams) (string, string, error) {
	if !params.ExportPublic {
		return "", "", nil
	}
	pub, err := exportPublicKey(ctx, absHomedir, fpr)
	if err != nil {
		return "", "", err
	}
	pubPath, _ := writePublicKey(params.Scenario, pub)
	return pub, pubPath, nil
}

func latestFingerprint(homedir string) (string, error) {
	cmd := exec.Command("gpg", "--batch", "--homedir", homedir, "--list-secret-keys", "--with-colons")
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("list-secret-keys failed: %s", string(ee.Stderr))
		}
		return "", err
	}
	lines := strings.Split(string(out), "\n")
	var fpr string
	for _, line := range lines {
		if strings.HasPrefix(line, "fpr:") {
			parts := strings.Split(line, ":")
			if len(parts) > 9 {
				fpr = strings.TrimSpace(parts[9])
			}
		}
	}
	if fpr == "" {
		return "", fmt.Errorf("no fingerprint found after generation")
	}
	return fpr, nil
}

func exportPublicKey(ctx context.Context, homedir, fingerprint string) (string, error) {
	cmd := exec.CommandContext(ctx, "gpg",
		"--batch",
		"--homedir", homedir,
		"--armor",
		"--export", fingerprint,
	)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("export public key failed: %s", string(ee.Stderr))
		}
		return "", err
	}
	return string(out), nil
}

func writePublicKey(scenario, contents string) (string, error) {
	if contents == "" {
		return "", nil
	}
	base := filepath.Join(resolveVrooliRoot(), "scenarios", scenario, "signing")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(base, "public-key.asc")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func resolveVrooliRoot() string {
	return pathutil.DetectVrooliRoot()
}
