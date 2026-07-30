package resourcedeployment

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseTrustModes(t *testing.T) {
	root := t.TempDir()
	body := []byte("vault")
	sum := sha256.Sum256(body)
	if err := os.WriteFile(filepath.Join(root, "vault_linux_amd64"), body, 0755); err != nil {
		t.Fatal(err)
	}
	m := ReleaseManifest{SchemaVersion: "v1", Artifacts: []ReleaseArtifact{{Name: "vault_linux_amd64", SHA256: fmtHex(sum[:]), Role: "managed-service", UpstreamProvenance: "hashicorp"}}}
	canonical, err := m.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "release-manifest.json"), canonical, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err = VerifyReleaseDirectory(root, ArtifactTrustDevelopmentLocal, ""); err != nil {
		t.Fatalf("development-local should accept verified manifest: %v", err)
	}
	if _, _, err = VerifyReleaseDirectory(root, ArtifactTrustProduction, filepath.Join(root, "pub")); err == nil {
		t.Fatal("production accepted unsigned manifest")
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	pub, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	os.WriteFile(filepath.Join(root, "pub"), pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pub}), 0644)
	os.WriteFile(filepath.Join(root, "release-manifest.sig.json"), []byte(`{"schema_version":"v1","key_id":"fixture","algorithm":"rsa-pkcs1v15-sha256","signature":"`+base64.StdEncoding.EncodeToString(sig)+`"}`), 0644)
	if _, _, err = VerifyReleaseDirectory(root, ArtifactTrustProduction, filepath.Join(root, "pub")); err != nil {
		t.Fatalf("production signature rejected: %v", err)
	}
}
func fmtHex(b []byte) string {
	const h = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = h[v>>4]
		out[i*2+1] = h[v&15]
	}
	return string(out)
}
