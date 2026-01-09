package signing

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	keyType := params.KeyType
	if keyType == "" {
		keyType = "rsa4096"
	}
	expiry := params.Expiry
	if expiry == "" {
		expiry = "1y"
	}

	// Resolve homedir under the scenario to avoid polluting the user's default keyring.
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
		return nil, fmt.Errorf("resolve homedir: %w", err)
	}
	if err := os.MkdirAll(absHomedir, 0o700); err != nil {
		return nil, fmt.Errorf("create homedir: %w", err)
	}
	_ = os.Chmod(absHomedir, 0o700)

	// If not forcing and keys already exist, bail out to avoid overwriting.
	if !params.Force {
		hasKeys, err := homedirHasSecretKeys(absHomedir)
		if err != nil {
			return nil, err
		}
		if hasKeys {
			return nil, fmt.Errorf("a GPG key already exists in %s (use force to overwrite)", absHomedir)
		}
	}

	uid := name
	if uid != "" && email != "" {
		uid = fmt.Sprintf("%s <%s>", name, email)
	} else if uid == "" {
		uid = email
	}

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
	genCmd.Stdin = strings.NewReader(params.Passphrase)
	if out, err := genCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("gpg key generation failed: %v: %s", err, string(out))
	}

	fpr, err := latestFingerprint(absHomedir)
	if err != nil {
		return nil, err
	}

	var pub string
	var pubPath string
	if params.ExportPublic {
		pub, err = exportPublicKey(ctx, absHomedir, fpr)
		if err != nil {
			return nil, err
		}
		pubPath, _ = writePublicKey(params.Scenario, pub)
	}

	return &generateLinuxKeyResult{
		Fingerprint: fpr,
		Homedir:     absHomedir,
		PublicKey:   pub,
		PublicPath:  pubPath,
	}, nil
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
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root
	}
	wd, err := os.Getwd()
	if err != nil {
		return filepath.Clean(filepath.Join("..", "..", ".."))
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", ".."))
}
