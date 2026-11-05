//go:build linux

package target

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	component "github.com/vrooli/vrooli/packages/desktop-component"
	release "github.com/vrooli/vrooli/packages/resource-deployment"
	bundleruntime "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"scenario-to-desktop-api/bundle"
)

// TestJ11PublishUsageFixture exports the owner-declared Agent Manager slice
// through the real build and component packaging boundaries. The ephemeral
// signing key belongs to the isolated producer, never a live release channel.
func TestJ11PublishUsageFixture(t *testing.T) {
	root := os.Getenv("J11_USAGE_PRODUCER_ROOT")
	if root == "" {
		t.Skip("J11_USAGE_PRODUCER_ROOT selects the isolated usage release producer")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	owner, err := filepath.Abs("../../../agent-manager")
	if err != nil {
		t.Fatal(err)
	}
	if selected := os.Getenv("J11_USAGE_OWNER_ROOT"); selected != "" {
		owner = selected
	}
	declaration := filepath.Join(owner, ".vrooli", "desktop-slices", "usage.json")
	// The production API runs from api/, where its standard runtime source
	// resolver finds ../runtime. Match that owner execution context.
	t.Chdir("..")
	result, err := bundle.NewPackager().Package(owner, declaration, "electron", []string{"linux-amd64"}, filepath.Join(root, "build"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(result.BundleDir, "bundle.json"))
	if err != nil {
		t.Fatal(err)
	}
	var admitted manifest.Manifest
	if err = json.Unmarshal(raw, &admitted); err != nil {
		t.Fatal(err)
	}
	if err = admitted.Validate("linux", "amd64"); err != nil {
		t.Fatalf("runtime admission before signing: %v", err)
	}
	// A real native start proves paths and injected environment, which schema
	// validation alone cannot. Keep both the peer registry and HOME isolated.
	smokeHome := filepath.Join(root, "smoke-home")
	supervisor, err := bundleruntime.NewSupervisor(bundleruntime.Options{Manifest: &admitted, BundlePath: result.BundleDir, AppDataDir: filepath.Join(smokeHome, "app"), PeerHomeDir: smokeHome, EnvReader: j11IsolatedEnv{"HOME": smokeHome, "PATH": "/usr/bin:/bin"}})
	if err != nil {
		t.Fatal(err)
	}
	if err = supervisor.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = supervisor.Shutdown(ctx)
	})
	deadline := time.Now().Add(20 * time.Second)
	for {
		response, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", supervisor.PortMap()["usage"]["api"]))
		if err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("real usage owner did not start: %v; statuses=%+v", err, supervisor.ServiceStatuses())
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err = supervisor.Shutdown(t.Context()); err != nil {
		t.Fatal(err)
	}
	entry := component.Entry{SchemaVersion: 1, ID: "agent-manager-usage", Owner: "agent-manager", Slice: "usage", Version: "0.0.1", OS: "linux", Arch: "amd64", HostRange: ">=0.0.1 <1.0.0", StateClass: "stateless", License: "Vrooli source license", ArtifactURL: "http://127.0.0.1:1/api/v1/components?id=agent-manager-usage", Provides: []component.Feature{{ID: "usage.live", Service: "usage", Port: "api"}}}
	entry, err = bundle.PackageComponent(result.BundleDir, filepath.Join(root, "agent-manager-usage.tar.gz"), entry)
	if err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	public, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public})
	canonical, err := entry.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	signed := component.SignedEntry{Entry: entry, Signature: release.ReleaseSignature{SchemaVersion: "v1", KeyID: "j11-isolated-release", Algorithm: "rsa-pkcs1v15-sha256", Signature: base64.StdEncoding.EncodeToString(signature)}}
	if err := signed.Verify(string(publicPEM)); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.MarshalIndent(signed, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"entry.json": encoded, "release.pub": publicPEM, "release.key": pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})} {
		if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("real owner=%s version=%s bundle=%s archive_bytes=%d archive_sha256=%s tree_sha256=%s", entry.Owner, entry.Version, result.BundleDir, entry.Size, entry.SHA256, entry.TreeSHA256)
}
