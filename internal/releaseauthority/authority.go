// Package releaseauthority owns the project-local release signing authority.
//
// The private key is held by the existing credential authority, which writes
// only to the operating system's secure credential facility. This package
// never creates a private-key file, accepts a key through argv, or returns key
// material to a caller. The public trust anchor remains repository-visible so
// desktop packaging and offline verification can remain deterministic.
package releaseauthority

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/binaryfetch"
	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

const (
	identityName = "vrooli/release-authority"
	privateField = "rsa-pkcs8-v1"
	publicPath   = "install/vrooli-release.pub"
)

// Status intentionally contains public operational state only.
type Status struct {
	Configured       bool   `json:"configured"`
	TrustAnchorMatch bool   `json:"trust_anchor_match"`
	KeyID            string `json:"key_id,omitempty"`
	Provider         string `json:"provider"`
}

// Authority generates, retains, and uses a release signing key without
// exposing its private material outside the native credential authority.
type Authority struct {
	credentials *credentialauthority.Authority
	identity    credentialauthority.Identity
}

func New(credentials *credentialauthority.Authority) (*Authority, error) {
	if credentials == nil {
		return nil, errors.New("release authority requires a native credential authority")
	}
	identity, err := credentialauthority.ParseIdentity(identityName)
	if err != nil {
		return nil, fmt.Errorf("parse release authority identity: %w", err)
	}
	return &Authority{credentials: credentials, identity: identity}, nil
}

// Initialize creates the authority only if none exists. Existing public trust
// anchors are protected: replacing one requires explicit operator intent.
func (a *Authority) Initialize(root string, replaceTrustAnchor bool) (Status, error) {
	key, configured, err := a.privateKey()
	if err != nil {
		return Status{}, err
	}
	if !configured {
		key, err = rsa.GenerateKey(rand.Reader, 3072)
		if err != nil {
			return Status{}, fmt.Errorf("generate release authority key: %w", err)
		}
		der, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			return Status{}, fmt.Errorf("marshal release authority key: %w", err)
		}
		privatePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
		if err := a.credentials.Put(a.identity, privateField, string(privatePEM)); err != nil {
			return Status{}, fmt.Errorf("store release authority key: %w", err)
		}
	}

	if err := writeTrustAnchors(root, publicPEM(key), replaceTrustAnchor); err != nil {
		return Status{}, err
	}
	return a.statusWithKey(root, key), nil
}

// Regenerate replaces the private key and trust anchor together. It is a
// destructive trust-root reset and is deliberately separate from Initialize.
func (a *Authority) Regenerate(root string) (Status, error) {
	key, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		return Status{}, fmt.Errorf("generate replacement release authority key: %w", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return Status{}, fmt.Errorf("marshal replacement release authority key: %w", err)
	}
	if err := a.credentials.Put(a.identity, privateField, string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))); err != nil {
		return Status{}, fmt.Errorf("store replacement release authority key: %w", err)
	}
	if err := writeTrustAnchors(root, publicPEM(key), true); err != nil {
		return Status{}, err
	}
	return a.statusWithKey(root, key), nil
}

func (a *Authority) Status(root string) (Status, error) {
	key, configured, err := a.privateKey()
	if err != nil {
		return Status{}, err
	}
	status := Status{Configured: configured, Provider: "native-secure-store"}
	if !configured {
		return status, nil
	}
	return a.statusWithKey(root, key), nil
}

// SignStage writes the production detached envelope after validating the
// staged manifest and its artifact hashes. It never signs an unchecked stage.
func (a *Authority) SignStage(root, stage string, overwrite bool) (resourcedeployment.ReleaseSignature, error) {
	key, configured, err := a.privateKey()
	if err != nil {
		return resourcedeployment.ReleaseSignature{}, err
	}
	if !configured {
		return resourcedeployment.ReleaseSignature{}, errors.New("release authority is not initialized")
	}
	if !a.statusWithKey(root, key).TrustAnchorMatch {
		return resourcedeployment.ReleaseSignature{}, errors.New("managed release authority does not match install/vrooli-release.pub; initialize with an explicit trust-anchor replacement before signing")
	}
	manifest, err := resourcedeployment.LoadReleaseManifest(stage)
	if err != nil {
		return resourcedeployment.ReleaseSignature{}, err
	}
	canonical, err := manifest.CanonicalBytes()
	if err != nil {
		return resourcedeployment.ReleaseSignature{}, err
	}
	for _, artifact := range manifest.Artifacts {
		path := filepath.Join(stage, artifact.Name)
		actual, err := stagedArtifactDigest(path)
		if err != nil {
			return resourcedeployment.ReleaseSignature{}, fmt.Errorf("hash staged artifact %s: %w", artifact.Name, err)
		}
		if actual != artifact.SHA256 {
			return resourcedeployment.ReleaseSignature{}, fmt.Errorf("staged artifact %s hash mismatch", artifact.Name)
		}
	}
	digest := sha256.Sum256(canonical)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return resourcedeployment.ReleaseSignature{}, fmt.Errorf("sign release manifest: %w", err)
	}
	envelope := resourcedeployment.ReleaseSignature{SchemaVersion: "v1", KeyID: keyID(key), Algorithm: "rsa-pkcs1v15-sha256", Signature: base64.StdEncoding.EncodeToString(signature)}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return resourcedeployment.ReleaseSignature{}, err
	}
	path := filepath.Join(stage, "release-manifest.sig.json")
	flags := os.O_WRONLY | os.O_CREATE
	if overwrite {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return resourcedeployment.ReleaseSignature{}, fmt.Errorf("create release signature: %w", err)
	}
	if _, err = file.Write(append(encoded, '\n')); err != nil {
		_ = file.Close()
		return resourcedeployment.ReleaseSignature{}, fmt.Errorf("write release signature: %w", err)
	}
	if err = file.Close(); err != nil {
		return resourcedeployment.ReleaseSignature{}, fmt.Errorf("close release signature: %w", err)
	}
	return envelope, nil
}

func stagedArtifactDigest(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return binaryfetch.TreeDigest(path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// AddEvidence copies a durable evidence file into a release stage and updates
// its canonical manifest. A prior signature is removed because the manifest
// bytes have changed; the caller must explicitly sign the completed stage.
func (a *Authority) AddEvidence(stage, source, name, role, osName, arch, provenance string) (resourcedeployment.ReleaseArtifact, error) {
	if !resourcedeployment.IsSafeArtifactName(name) || strings.TrimSpace(role) == "" || strings.TrimSpace(provenance) == "" {
		return resourcedeployment.ReleaseArtifact{}, errors.New("evidence requires a safe name, role, and provenance")
	}
	info, err := os.Stat(source)
	if err != nil {
		return resourcedeployment.ReleaseArtifact{}, fmt.Errorf("stat evidence source: %w", err)
	}
	if !info.Mode().IsRegular() {
		return resourcedeployment.ReleaseArtifact{}, errors.New("evidence source must be a regular file")
	}
	manifest, err := resourcedeployment.LoadReleaseManifest(stage)
	if err != nil {
		return resourcedeployment.ReleaseArtifact{}, err
	}
	for _, artifact := range manifest.Artifacts {
		if artifact.Name == name {
			return resourcedeployment.ReleaseArtifact{}, fmt.Errorf("release manifest already contains %q", name)
		}
	}
	in, err := os.Open(source)
	if err != nil {
		return resourcedeployment.ReleaseArtifact{}, fmt.Errorf("open evidence source: %w", err)
	}
	defer in.Close()
	destination := filepath.Join(stage, name)
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return resourcedeployment.ReleaseArtifact{}, fmt.Errorf("create staged evidence: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(destination)
		return resourcedeployment.ReleaseArtifact{}, fmt.Errorf("copy staged evidence: %w", err)
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(destination)
		return resourcedeployment.ReleaseArtifact{}, fmt.Errorf("close staged evidence: %w", err)
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		return resourcedeployment.ReleaseArtifact{}, err
	}
	sum := sha256.Sum256(data)
	artifact := resourcedeployment.ReleaseArtifact{Name: name, SHA256: hex.EncodeToString(sum[:]), Role: role, OS: osName, Arch: arch, UpstreamProvenance: provenance}
	manifest.Artifacts = append(manifest.Artifacts, artifact)
	sort.Slice(manifest.Artifacts, func(i, j int) bool { return manifest.Artifacts[i].Name < manifest.Artifacts[j].Name })
	canonical, err := manifest.CanonicalBytes()
	if err != nil {
		return resourcedeployment.ReleaseArtifact{}, err
	}
	if err := os.WriteFile(filepath.Join(stage, "release-manifest.json"), append(canonical, '\n'), 0o644); err != nil {
		return resourcedeployment.ReleaseArtifact{}, fmt.Errorf("write release manifest: %w", err)
	}
	if err := os.Remove(filepath.Join(stage, "release-manifest.sig.json")); err != nil && !os.IsNotExist(err) {
		return resourcedeployment.ReleaseArtifact{}, fmt.Errorf("remove stale release signature: %w", err)
	}
	return artifact, nil
}

func (a *Authority) privateKey() (*rsa.PrivateKey, bool, error) {
	values := map[string]string{}
	err := a.credentials.Inject(a.identity, privateField, privateField, values)
	if errors.Is(err, credentialauthority.ErrUnconfigured) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("load release authority key: %w", err)
	}
	block, _ := pem.Decode([]byte(values[privateField]))
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, false, errors.New("stored release authority key is not PKCS#8 PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, false, fmt.Errorf("parse stored release authority key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok || key.N.BitLen() != 3072 {
		return nil, false, errors.New("stored release authority key must be RSA-3072")
	}
	return key, true, nil
}

func (a *Authority) statusWithKey(root string, key *rsa.PrivateKey) Status {
	anchor, err := os.ReadFile(filepath.Join(root, publicPath))
	return Status{Configured: true, TrustAnchorMatch: err == nil && strings.TrimSpace(string(anchor)) == strings.TrimSpace(string(publicPEM(key))), KeyID: keyID(key), Provider: "native-secure-store"}
}

func publicPEM(key *rsa.PrivateKey) []byte {
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		panic(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
}

func keyID(key *rsa.PrivateKey) string {
	der, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	digest := sha256.Sum256(der)
	return "rsa3072-" + hex.EncodeToString(digest[:8])
}

func writeTrustAnchors(root string, public []byte, replace bool) error {
	path := filepath.Join(root, publicPath)
	current, err := os.ReadFile(path)
	if err == nil && strings.TrimSpace(string(current)) != strings.TrimSpace(string(public)) && !replace {
		return errors.New("release trust anchor already exists and does not match the managed key; rerun with explicit trust-anchor replacement")
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read release trust anchor: %w", err)
	}
	shellPath := filepath.Join(root, "install", "install.sh")
	shell, err := os.ReadFile(shellPath)
	if err != nil {
		return fmt.Errorf("read shell installer trust anchor: %w", err)
	}
	updatedShell, err := replacePublicPEMBlock(string(shell), string(public))
	if err != nil {
		return fmt.Errorf("update shell installer trust anchor: %w", err)
	}
	modulus := base64.StdEncoding.EncodeToString(publicKeyFromPEM(public).N.Bytes())
	powershellPaths := []string{
		filepath.Join(root, "install", "install.ps1"),
		filepath.Join(root, "packages", "cli-core", "install", "Platform.ps1"),
	}
	updates := map[string][]byte{path: public, shellPath: []byte(updatedShell)}
	for _, powershellPath := range powershellPaths {
		contents, readErr := os.ReadFile(powershellPath)
		if readErr != nil {
			return fmt.Errorf("read PowerShell installer trust anchor: %w", readErr)
		}
		updated, replaceErr := replacePowerShellModulus(string(contents), modulus)
		if replaceErr != nil {
			return fmt.Errorf("update PowerShell installer trust anchor: %w", replaceErr)
		}
		updates[powershellPath] = []byte(updated)
	}
	return writeAtomically(updates)
}

func publicKeyFromPEM(value []byte) *rsa.PublicKey {
	block, _ := pem.Decode(value)
	if block == nil {
		panic("release authority generated invalid public PEM")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		panic(err)
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok {
		panic("release authority generated non-RSA public key")
	}
	return key
}

func replacePublicPEMBlock(contents, replacement string) (string, error) {
	const begin = "-----BEGIN PUBLIC KEY-----"
	const end = "-----END PUBLIC KEY-----"
	start := strings.Index(contents, begin)
	if start < 0 {
		return "", errors.New("missing embedded release public key")
	}
	finish := strings.Index(contents[start:], end)
	if finish < 0 {
		return "", errors.New("unterminated embedded release public key")
	}
	finish += start + len(end)
	return contents[:start] + strings.TrimSpace(replacement) + contents[finish:], nil
}

func replacePowerShellModulus(contents, modulus string) (string, error) {
	const prefix = "$releasePublicModulus = '"
	start := strings.Index(contents, prefix)
	if start < 0 {
		const fallback = "$modulus = '"
		start = strings.Index(contents, fallback)
		if start < 0 {
			return "", errors.New("missing RSA public modulus assignment")
		}
		start += len(fallback)
	} else {
		start += len(prefix)
	}
	end := strings.Index(contents[start:], "'")
	if end < 0 {
		return "", errors.New("unterminated RSA public modulus assignment")
	}
	end += start
	return contents[:start] + modulus + contents[end:], nil
}

func writeAtomically(updates map[string][]byte) error {
	temporary := make(map[string]string, len(updates))
	for path, contents := range updates {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		file, err := os.CreateTemp(filepath.Dir(path), ".vrooli-release-anchor-*")
		if err != nil {
			return err
		}
		if _, err = file.Write(contents); err == nil {
			err = file.Chmod(0o644)
		}
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(file.Name())
			return err
		}
		temporary[path] = file.Name()
	}
	for path, temporaryPath := range temporary {
		if err := os.Rename(temporaryPath, path); err != nil {
			return fmt.Errorf("publish release trust anchor %s: %w", path, err)
		}
	}
	return nil
}
