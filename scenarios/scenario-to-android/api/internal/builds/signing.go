package builds

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

// prepareSigningFiles materializes secrets only in a short-lived, permission
// restricted directory outside the generated project. The paths are passed to
// Gradle as a system property and removed as soon as the build ends.
func (b Builder) prepareSigningFiles(ctx context.Context, projectRoot, identity string) ([]string, error) {
	if b.Signing == nil {
		return nil, fmt.Errorf("Android signing unavailable: secrets-manager credential client is not configured; provision %s", identity)
	}
	keystore, err := b.Signing.Resolve(ctx, identity, SigningKeystoreField)
	if err != nil {
		return nil, fmt.Errorf("Android signing unavailable: resolve %s/%s: %w", identity, SigningKeystoreField, err)
	}
	password, err := b.Signing.Resolve(ctx, identity, SigningPasswordField)
	if err != nil {
		return nil, fmt.Errorf("Android signing unavailable: resolve %s/%s: %w", identity, SigningPasswordField, err)
	}
	alias, err := b.Signing.Resolve(ctx, identity, SigningAliasField)
	if err != nil {
		return nil, fmt.Errorf("Android signing unavailable: resolve %s/%s: %w", identity, SigningAliasField, err)
	}
	keystoreBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(keystore))
	if err != nil || len(keystoreBytes) == 0 {
		return nil, fmt.Errorf("Android signing unavailable: %s/%s is not valid base64 keystore data", identity, SigningKeystoreField)
	}
	secretDir, err := os.MkdirTemp("", "vrooli-android-signing-")
	if err != nil {
		return nil, fmt.Errorf("create temporary Android signing directory: %w", err)
	}
	if err := os.Chmod(secretDir, 0o700); err != nil {
		_ = os.RemoveAll(secretDir)
		return nil, fmt.Errorf("restrict temporary Android signing directory: %w", err)
	}
	keystorePath := filepath.Join(secretDir, "upload.keystore")
	propertiesPath := filepath.Join(secretDir, "signing.properties")
	if err := os.WriteFile(keystorePath, keystoreBytes, 0o600); err != nil {
		_ = os.RemoveAll(secretDir)
		return nil, fmt.Errorf("write temporary Android keystore: %w", err)
	}
	properties := fmt.Sprintf("storeFile=%s\nstorePassword=%s\nkeyAlias=%s\nkeyPassword=%s\n", escapeProperties(keystorePath), escapeProperties(password), escapeProperties(alias), escapeProperties(password))
	if err := os.WriteFile(propertiesPath, []byte(properties), 0o600); err != nil {
		_ = os.RemoveAll(secretDir)
		return nil, fmt.Errorf("write temporary Android signing properties: %w", err)
	}
	_ = projectRoot // keeps the call-site explicit: these files must not live under the generated project.
	return []string{secretDir, propertiesPath}, nil
}

func removeFiles(paths []string) {
	if len(paths) > 0 {
		_ = os.RemoveAll(paths[0])
	}
	for _, path := range paths[1:] {
		if strings.TrimSpace(path) == "" {
			continue
		}
		_ = os.Remove(path)
	}
}

func escapeProperties(value string) string {
	return strings.NewReplacer("\\", "\\\\", "\n", "\\n", "\r", "\\r", "=", "\\=", ":", "\\:").Replace(value)
}

// ProvisionSigningKey creates an upload key only when the three declared
// credential fields are all absent. It never returns key material. Partial
// configuration is rejected because silently replacing one field would make
// an existing Play upload identity unusable.
type SigningProvisioner interface {
	SigningStore
	Provision(context.Context, credentialclient.ProvisionRequest) (credentialclient.ProvisionResponse, error)
}

func ProvisionSigningKey(ctx context.Context, store SigningProvisioner, identity, keytoolBin string, run RunCommand) error {
	if store == nil {
		return fmt.Errorf("secrets-manager credential client is not configured")
	}
	identity = strings.TrimSpace(identity)
	if identity == "" {
		identity = DefaultSigningIdentity
	}
	values := make(map[string]string, 3)
	fields := []string{SigningKeystoreField, SigningPasswordField, SigningAliasField}
	configured := 0
	for _, field := range fields {
		value, err := store.Resolve(ctx, identity, field)
		if err == nil && strings.TrimSpace(value) != "" {
			values[field] = value
			configured++
		}
	}
	if configured == len(fields) {
		return nil
	}
	if configured != 0 {
		return fmt.Errorf("signing identity %s is partially configured; refuse to replace it", identity)
	}
	if keytoolBin == "" {
		keytoolBin = "keytool"
	}
	if filepath.Base(keytoolBin) == keytoolBin {
		if _, err := exec.LookPath(keytoolBin); err != nil {
			return fmt.Errorf("keytool is unavailable: %w", err)
		}
	}
	password, err := randomSecret(32)
	if err != nil {
		return fmt.Errorf("generate signing password: %w", err)
	}
	secretDir, err := os.MkdirTemp("", "vrooli-android-provision-")
	if err != nil {
		return fmt.Errorf("create signing provision directory: %w", err)
	}
	defer os.RemoveAll(secretDir)
	_ = os.Chmod(secretDir, 0o700)
	keystorePath := filepath.Join(secretDir, "upload.keystore")
	passwordPath := filepath.Join(secretDir, "upload.password")
	alias := "vrooli-upload"
	if err := os.WriteFile(passwordPath, []byte(password+"\n"), 0o600); err != nil {
		return fmt.Errorf("write temporary Android signing password: %w", err)
	}
	if run == nil {
		run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).CombinedOutput() // #nosec G702 -- explicit governed keytool binary; tests inject Run
		}
	}
	output, err := run(ctx, keytoolBin, "-genkeypair", "-alias", alias, "-keyalg", "RSA", "-keysize", "2048", "-validity", "10000", "-storetype", "PKCS12", "-keystore", keystorePath, "-storepass:file", passwordPath, "-keypass:file", passwordPath, "-dname", "CN=Vrooli Android Upload,O=Vrooli")
	if err != nil {
		return fmt.Errorf("generate Android upload key: %w: %s", err, strings.TrimSpace(string(output)))
	}
	data, err := os.ReadFile(keystorePath)
	if err != nil {
		return fmt.Errorf("read generated upload key: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	for field, value := range map[string]string{SigningKeystoreField: encoded, SigningPasswordField: password, SigningAliasField: alias} {
		if _, err := store.Provision(ctx, credentialclient.ProvisionRequest{Identity: identity, Field: field, Value: value}); err != nil {
			return fmt.Errorf("provision signing identity %s/%s: %w", identity, field, err)
		}
	}
	return nil
}

func randomSecret(size int) (string, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}
