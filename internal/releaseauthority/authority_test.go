package releaseauthority

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/resources/securestore"
	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

type memoryStore struct{ values map[string]string }

func (s *memoryStore) Put(service, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"/"+key] = value
	return nil
}

func (s *memoryStore) Get(service, key string) (string, error) {
	value, ok := s.values[service+"/"+key]
	if !ok {
		return "", os.ErrNotExist
	}
	return value, nil
}

func (s *memoryStore) Delete(service, key string) error {
	delete(s.values, service+"/"+key)
	return nil
}

var _ securestore.Store = (*memoryStore)(nil)

func newTestAuthority(t *testing.T) (*Authority, string) {
	t.Helper()
	credentials, err := credentialauthority.NewAuthority(&memoryStore{})
	if err != nil {
		t.Fatal(err)
	}
	authority, err := New(credentials)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for path, contents := range map[string]string{
		"install/install.sh":                     "-----BEGIN PUBLIC KEY-----\nold\n-----END PUBLIC KEY-----\n",
		"install/install.ps1":                    "$releasePublicModulus = 'old'\n",
		"packages/cli-core/install/Platform.ps1": "$modulus = 'old'\n",
	} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return authority, root
}

func TestInitializeRetainsPrivateKeyAndProtectsExistingAnchor(t *testing.T) {
	authority, root := newTestAuthority(t)
	first, err := authority.Initialize(root, false)
	if err != nil || !first.Configured || !first.TrustAnchorMatch || first.KeyID == "" {
		t.Fatalf("first initialization = %#v, %v", first, err)
	}
	anchor, err := os.ReadFile(filepath.Join(root, publicPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"install/install.sh", "install/install.ps1", "packages/cli-core/install/Platform.ps1"} {
		contents, readErr := os.ReadFile(filepath.Join(root, path))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if string(contents) == "" || strings.Contains(string(contents), "old") || (path == "install/install.sh" && !strings.Contains(string(contents), string(anchor))) {
			t.Fatalf("trust anchor was not synchronized into %s", path)
		}
	}
	second, err := authority.Initialize(root, false)
	if err != nil || second.KeyID != first.KeyID || !second.TrustAnchorMatch {
		t.Fatalf("second initialization = %#v, %v", second, err)
	}
	if err := os.WriteFile(filepath.Join(root, publicPath), []byte("different"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := authority.Initialize(root, false); err == nil {
		t.Fatal("Initialize replaced a mismatched trust anchor without explicit authorization")
	}
	if _, err := authority.Initialize(root, true); err != nil {
		t.Fatalf("Initialize with explicit replacement: %v", err)
	}
}

func TestSignStageProducesProductionVerifiableEnvelope(t *testing.T) {
	authority, root := newTestAuthority(t)
	if _, err := authority.Initialize(root, false); err != nil {
		t.Fatal(err)
	}
	stage := t.TempDir()
	artifact := []byte("desktop evidence")
	if err := os.WriteFile(filepath.Join(stage, "evidence.bin"), artifact, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(artifact)
	manifest := resourcedeployment.ReleaseManifest{SchemaVersion: "v1", Artifacts: []resourcedeployment.ReleaseArtifact{{Name: "evidence.bin", SHA256: hex.EncodeToString(sum[:]), Role: "desktop-recording", UpstreamProvenance: "test"}}}
	canonical, err := manifest.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "release-manifest.json"), append(canonical, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := authority.SignStage(root, stage, false); err != nil {
		t.Fatalf("SignStage: %v", err)
	}
	if _, _, err := resourcedeployment.VerifyReleaseDirectory(stage, resourcedeployment.ArtifactTrustProduction, filepath.Join(root, publicPath)); err != nil {
		t.Fatalf("VerifyReleaseDirectory: %v", err)
	}
	if _, err := authority.SignStage(root, stage, false); err == nil {
		t.Fatal("SignStage overwrote an existing envelope")
	}
}

func TestAddEvidenceUpdatesManifestAndInvalidatesPriorSignature(t *testing.T) {
	authority, root := newTestAuthority(t)
	if _, err := authority.Initialize(root, false); err != nil {
		t.Fatal(err)
	}
	stage := t.TempDir()
	initial := []byte("initial evidence")
	if err := os.WriteFile(filepath.Join(stage, "initial.bin"), initial, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(initial)
	manifest := resourcedeployment.ReleaseManifest{SchemaVersion: "v1", Artifacts: []resourcedeployment.ReleaseArtifact{{Name: "initial.bin", SHA256: hex.EncodeToString(sum[:]), Role: "smoke-report", UpstreamProvenance: "test"}}}
	canonical, err := manifest.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "release-manifest.json"), append(canonical, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := authority.SignStage(root, stage, false); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "timeline.json")
	if err := os.WriteFile(source, []byte("{\"timeline\":true}"), 0o644); err != nil {
		t.Fatal(err)
	}
	artifact, err := authority.AddEvidence(stage, source, "console-timeline.json", "bas-run", "linux", "amd64", "BAS run test")
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Role != "bas-run" || artifact.Name != "console-timeline.json" {
		t.Fatalf("unexpected evidence artifact: %#v", artifact)
	}
	if _, err := os.Stat(filepath.Join(stage, "release-manifest.sig.json")); !os.IsNotExist(err) {
		t.Fatalf("stale signature should be removed, stat error = %v", err)
	}
	updated, err := resourcedeployment.LoadReleaseManifest(stage)
	if err != nil || len(updated.Artifacts) != 2 {
		t.Fatalf("updated manifest = %#v, %v", updated, err)
	}
	if _, err := authority.SignStage(root, stage, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := resourcedeployment.VerifyReleaseDirectory(stage, resourcedeployment.ArtifactTrustProduction, filepath.Join(root, publicPath)); err != nil {
		t.Fatal(err)
	}
}

func TestRegenerateReplacesTheTrustRoot(t *testing.T) {
	authority, root := newTestAuthority(t)
	first, err := authority.Initialize(root, false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := authority.Regenerate(root)
	if err != nil || second.KeyID == first.KeyID || !second.TrustAnchorMatch {
		t.Fatalf("Regenerate = %#v, %v", second, err)
	}
}
