package resourcedeployment

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActivateReleasePromotesOnlyDeclaredArtifacts(t *testing.T) {
	source := t.TempDir()
	install := t.TempDir()
	writeRelease(t, source, map[string]string{"app.bin": "app", "runtime.bin": "runtime"}, false)
	if err := os.WriteFile(filepath.Join(source, "unrelated.txt"), []byte("must not ship"), 0o644); err != nil {
		t.Fatal(err)
	}

	receipt, err := ActivateRelease(context.Background(), ActivationRequest{
		SourceDir: source, InstallDir: install, OperationID: "op-1", GenerationID: "gen-1",
		TrustMode: ArtifactTrustDevelopmentLocal,
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != ActivationStateActivated || receipt.PreviousGeneration != "" {
		t.Fatalf("receipt = %+v", receipt)
	}
	current := filepath.Join(install, activationCurrentEntry)
	target, err := os.Readlink(current)
	if err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(install, target)
	for _, name := range []string{"app.bin", "runtime.bin", "release-manifest.json"} {
		if _, err := os.Stat(filepath.Join(active, name)); err != nil {
			t.Fatalf("declared artifact %q missing: %v", name, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(active, "unrelated.txt")); !os.IsNotExist(err) {
		t.Fatalf("unrelated content shipped, err=%v", err)
	}
}

func TestActivateReleasePreservesPreviousGenerationOnCorruption(t *testing.T) {
	source := t.TempDir()
	install := t.TempDir()
	writeRelease(t, source, map[string]string{"app.bin": "good"}, false)
	_, err := ActivateRelease(context.Background(), ActivationRequest{
		SourceDir: source, InstallDir: install, OperationID: "op-good", GenerationID: "gen-good",
		TrustMode: ArtifactTrustDevelopmentLocal,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Keep the signed/declarative digest and alter only the source bytes.
	if err := os.WriteFile(filepath.Join(source, "app.bin"), []byte("altered"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = ActivateRelease(context.Background(), ActivationRequest{
		SourceDir: source, InstallDir: install, OperationID: "op-corrupt", GenerationID: "gen-corrupt",
		TrustMode: ArtifactTrustDevelopmentLocal,
	})
	if err == nil || !strings.Contains(err.Error(), "changed during staging") {
		t.Fatalf("corruption error = %v", err)
	}
	active, err := activeGeneration(install)
	if err != nil {
		t.Fatal(err)
	}
	if active != "gen-good" {
		t.Fatalf("active generation changed after corruption: %q", active)
	}
}

func TestActivateReleaseRejectsInvalidSignerAndKeepsPreviousGeneration(t *testing.T) {
	goodSource := t.TempDir()
	install := t.TempDir()
	writeRelease(t, goodSource, map[string]string{"app.bin": "good"}, false)
	if _, err := ActivateRelease(context.Background(), ActivationRequest{
		SourceDir: goodSource, InstallDir: install, OperationID: "op-good", GenerationID: "gen-good",
		TrustMode: ArtifactTrustDevelopmentLocal,
	}); err != nil {
		t.Fatal(err)
	}

	badSource := t.TempDir()
	writeRelease(t, badSource, map[string]string{"app.bin": "new"}, true)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPath := filepath.Join(badSource, "pub")
	if err := os.WriteFile(pubPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pub}), 0o644); err != nil {
		t.Fatal(err)
	}
	// The signature is intentionally made with a different key.
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := LoadReleaseManifest(badSource)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := manifest.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	signature, err := rsa.SignPKCS1v15(rand.Reader, other, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := json.Marshal(ReleaseSignature{SchemaVersion: "v1", KeyID: "wrong", Algorithm: "rsa-pkcs1v15-sha256", Signature: base64.StdEncoding.EncodeToString(signature)})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badSource, "release-manifest.sig.json"), envelope, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = ActivateRelease(context.Background(), ActivationRequest{
		SourceDir: badSource, InstallDir: install, OperationID: "op-bad-signature", GenerationID: "gen-bad-signature",
		TrustMode: ArtifactTrustProduction, PublicKeyPath: pubPath,
	})
	if err == nil || !strings.Contains(err.Error(), "signature invalid") {
		t.Fatalf("invalid signer error = %v", err)
	}
	active, err := activeGeneration(install)
	if err != nil {
		t.Fatal(err)
	}
	if active != "gen-good" {
		t.Fatalf("active generation changed after signer failure: %q", active)
	}
}

func TestActivateReleaseResumesAndRevalidatesInterruptedStage(t *testing.T) {
	source := t.TempDir()
	install := t.TempDir()
	writeRelease(t, source, map[string]string{"a.bin": "a", "b.bin": "b"}, false)
	// Make the second source artifact fail after the first artifact has been
	// copied, leaving a verified resumable stage behind.
	manifest, err := LoadReleaseManifest(source)
	if err != nil {
		t.Fatal(err)
	}
	for i := range manifest.Artifacts {
		if manifest.Artifacts[i].Name == "b.bin" {
			manifest.Artifacts[i].SHA256 = strings.Repeat("0", 64)
		}
	}
	canonical, err := manifest.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "release-manifest.json"), canonical, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = ActivateRelease(context.Background(), ActivationRequest{
		SourceDir: source, InstallDir: install, OperationID: "op-resume", GenerationID: "gen-resume",
		TrustMode: ArtifactTrustDevelopmentLocal,
	})
	if err == nil {
		t.Fatal("interrupted source was accepted")
	}
	if _, err := os.Stat(filepath.Join(install, activationStagingDir, "op-resume", "a.bin")); err != nil {
		t.Fatalf("verified first artifact was not retained: %v", err)
	}

	// Restore the expected manifest. The existing a.bin is reused only after a
	// fresh hash check; b.bin is copied from scratch.
	writeRelease(t, source, map[string]string{"a.bin": "a", "b.bin": "b"}, false)
	receipt, err := ActivateRelease(context.Background(), ActivationRequest{
		SourceDir: source, InstallDir: install, OperationID: "op-resume", GenerationID: "gen-resume",
		TrustMode: ArtifactTrustDevelopmentLocal,
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ReusedBytes == 0 || receipt.RevalidatedFiles == 0 || receipt.BytesTransferred == 0 {
		t.Fatalf("resume did not distinguish reused and transferred content: %+v", receipt)
	}
	active, err := activeGeneration(install)
	if err != nil {
		t.Fatal(err)
	}
	if active != "gen-resume" {
		t.Fatalf("resume did not activate generation: %q", active)
	}
}

func TestActivateReleaseRejectsManifestPathAndSourceSymlink(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "outside.bin")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(source, "app.bin")); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("outside"))
	manifest := ReleaseManifest{SchemaVersion: "v1", Artifacts: []ReleaseArtifact{{Name: "app.bin", SHA256: hex.EncodeToString(sum[:]), Role: "runtime", UpstreamProvenance: "fixture"}}}
	canonical, err := manifest.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "release-manifest.json"), canonical, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = ActivateRelease(context.Background(), ActivationRequest{
		SourceDir: source, InstallDir: t.TempDir(), OperationID: "op-symlink", GenerationID: "gen-symlink",
		TrustMode: ArtifactTrustDevelopmentLocal,
	})
	if err == nil || !strings.Contains(err.Error(), "symlinks and directories are not allowed") {
		t.Fatalf("source symlink error = %v", err)
	}

	badManifestRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(badManifestRoot, "release-manifest.json"), []byte(`{"schema_version":"v1","artifacts":[{"name":"../outside.bin","sha256":"`+strings.Repeat("0", 64)+`","role":"runtime","upstream_provenance":"fixture"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = ActivateRelease(context.Background(), ActivationRequest{
		SourceDir: badManifestRoot, InstallDir: t.TempDir(), OperationID: "op-path", GenerationID: "gen-path",
		TrustMode: ArtifactTrustDevelopmentLocal,
	})
	if err == nil || !strings.Contains(err.Error(), "invalid release artifact") {
		t.Fatalf("manifest path error = %v", err)
	}
}

func writeRelease(t *testing.T, root string, artifacts map[string]string, withSignature bool) {
	t.Helper()
	entries := make([]ReleaseArtifact, 0, len(artifacts))
	for name, content := range artifacts {
		data := []byte(content)
		sum := sha256.Sum256(data)
		if err := os.WriteFile(filepath.Join(root, name), data, 0o755); err != nil {
			t.Fatal(err)
		}
		entries = append(entries, ReleaseArtifact{Name: name, SHA256: hex.EncodeToString(sum[:]), Role: "runtime", UpstreamProvenance: "fixture"})
	}
	manifest := ReleaseManifest{SchemaVersion: "v1", Artifacts: entries}
	canonical, err := manifest.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "release-manifest.json"), canonical, 0o644); err != nil {
		t.Fatal(err)
	}
	if withSignature {
		// The caller replaces this envelope when it needs to exercise a signer
		// mismatch. Keeping a structurally valid placeholder makes source setup
		// itself deterministic.
		if err := os.WriteFile(filepath.Join(root, "release-manifest.sig.json"), []byte(`{"schema_version":"v1","key_id":"fixture","algorithm":"rsa-pkcs1v15-sha256","signature":"AA=="}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
