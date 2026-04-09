package signing

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	pathutil "scenario-to-desktop-api/shared/path"
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
	WorkingDirRoot string
}

type generateLinuxKeyResult struct {
	Fingerprint string
	Homedir     string
	PublicKey   string
	PublicPath  string
}

func (h *Handler) generateLinuxKey(ctx context.Context, params generateLinuxKeyParams) (*generateLinuxKeyResult, error) {
	if _, err := exec.LookPath("gpg"); err != nil {
		return nil, fmt.Errorf("gpg is not installed: %w", err)
	}

	name := strings.TrimSpace(params.Name)
	email := strings.TrimSpace(params.Email)
	if name == "" && email == "" {
		return nil, fmt.Errorf("name or email is required to generate a key")
	}

	absHomedir, err := prepareGPGHomedir(params)
	if err != nil {
		return nil, err
	}

	if err := checkExistingKeys(absHomedir, params.Force); err != nil {
		return nil, err
	}

	uid := formatUID(name, email)
	keyType := valueOrDefault(params.KeyType, "rsa4096")
	expiry := valueOrDefault(params.Expiry, "1y")

	if err := runGPGGenerate(ctx, absHomedir, uid, keyType, expiry, params.Passphrase); err != nil {
		return nil, err
	}

	fpr, err := latestFingerprint(absHomedir)
	if err != nil {
		return nil, err
	}

	pub, pubPath, err := optionalExportPublicKey(ctx, absHomedir, fpr, params)
	if err != nil {
		return nil, err
	}

	return &generateLinuxKeyResult{
		Fingerprint: fpr,
		Homedir:     absHomedir,
		PublicKey:   pub,
		PublicPath:  pubPath,
	}, nil
}

// prepareGPGHomedir resolves and creates the GPG homedir.
func prepareGPGHomedir(params generateLinuxKeyParams) (string, error) {
	homedir := params.Homedir
	if homedir == "" {
		base := params.WorkingDirRoot
		if base == "" {
			base = "."
		}
		homedir = filepath.Join(base, "scenarios", params.Scenario, "signing", "gnupg")
	}
	absHomedir, err := filepath.Abs(homedir)
	if err != nil {
		return "", fmt.Errorf("resolve homedir: %w", err)
	}
	if err := os.MkdirAll(absHomedir, 0o700); err != nil {
		return "", fmt.Errorf("create homedir: %w", err)
	}
	_ = os.Chmod(absHomedir, 0o700)
	return absHomedir, nil
}

// checkExistingKeys returns an error if keys already exist and force is false.
func checkExistingKeys(absHomedir string, force bool) error {
	if force {
		return nil
	}
	hasKeys, err := homedirHasSecretKeys(absHomedir)
	if err != nil {
		return err
	}
	if hasKeys {
		return fmt.Errorf("a GPG key already exists in %s (use force to overwrite)", absHomedir)
	}
	return nil
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

func homedirHasSecretKeys(homedir string) (bool, error) {
	listArgs := []string{
		"--batch",
		"--homedir", homedir,
		"--list-secret-keys",
		"--with-colons",
	}
	cmd := exec.Command("gpg", listArgs...)
	out, err := cmd.Output()
	if err != nil {
		// If no keys, gpg exits 0; other errors should bubble up.
		if ee, ok := err.(*exec.ExitError); ok {
			return false, fmt.Errorf("gpg list-secret-keys failed: %s", string(ee.Stderr))
		}
		return false, err
	}
	return bytes.Contains(out, []byte("fpr:")), nil
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
