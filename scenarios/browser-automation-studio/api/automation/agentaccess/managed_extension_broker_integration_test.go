package agentaccess

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

func TestBrokerManagedExtensionLifecycleUsesExactSessionAndVerifiedArtifact(t *testing.T) {
	artifact, archiveBytes := makeSignedExtensionArchive(t)
	var downloads atomic.Int64
	artifactOwner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/extension/download/"+artifact.ArtifactRef {
			http.Error(w, "unexpected artifact request", http.StatusNotFound)
			return
		}
		downloads.Add(1)
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(archiveBytes)
	}))
	defer artifactOwner.Close()

	h := newExtensionArtifactHarness(t, artifact, nil)
	controller, err := NewManagedExtensionController(
		h.broker.manager,
		artifactOwner.URL,
		WithManagedExtensionStagingRoot(filepath.Join(t.TempDir(), "extensions")),
	)
	if err != nil {
		t.Fatal(err)
	}
	h.broker.extensions = controller
	ctx := withBridgeContextCaller(context.Background(), h.callerID)

	installed, err := h.broker.InstallExtension(ctx, h.entry.sessionID, "", artifact.ArtifactRef, artifact.Digest)
	if err != nil {
		t.Fatal(err)
	}
	binding, err := h.broker.extensionSessionBinding(h.entry)
	if err != nil {
		t.Fatal(err)
	}
	if installed.Operation != "install" || installed.State != "installed" || !extensionReceiptMatchesArtifact(installed, h.entry.sessionID, artifact) || !extensionReceiptMatchesBinding(installed, binding) {
		t.Fatalf("install receipt was not bound to the current owner and verified artifact: %#v", installed)
	}

	reloaded, err := h.broker.ReloadExtension(ctx, h.entry.sessionID, artifact.ExtensionID)
	if err != nil || reloaded.Operation != "reload" || reloaded.State != "reloaded" {
		t.Fatalf("reload receipt=%#v err=%v", reloaded, err)
	}
	validated, err := h.broker.ValidateExtension(ctx, h.entry.sessionID, artifact.ExtensionID)
	if err != nil || !validated.Valid || validated.Receipt.Operation != "validate" || validated.Receipt.State != "valid" {
		t.Fatalf("validation receipt=%#v err=%v", validated, err)
	}
	inventory, err := h.broker.SelectedTabExtensions(ctx, h.entry.sessionID)
	if err != nil || len(inventory.Extensions) != 1 {
		t.Fatalf("inventory=%#v err=%v", inventory, err)
	}
	if inventory.Binding != binding || inventory.SessionID != h.entry.sessionID {
		t.Fatalf("inventory receipt was not bound to the current owner and profile: %#v", inventory)
	}
	item := inventory.Extensions[0]
	if !item.ArtifactVerified || item.ArtifactSHA256 != artifact.Digest || item.ManifestSHA256 != artifact.ManifestSHA256 || !containsString(item.DeclaredPermissions, "storage") || !containsString(item.DeclaredHostPermissions, "https://example.test/*") {
		t.Fatalf("inventory did not expose verified declared permissions: %#v", item)
	}
	if item.PermissionEvidence.Status != sessionpermission.ExtensionPermissionEvidenceEffective || item.PermissionEvidence.Source != sessionpermission.ExtensionPermissionSourceChromeGetAll || item.PermissionEvidence.ObservedAt.IsZero() || !item.ArtifactVerified || item.ArtifactSHA256 != artifact.Digest || item.ManifestSHA256 != artifact.ManifestSHA256 || len(item.Permissions) != 1 || item.Permissions[0] != "storage" {
		t.Fatalf("managed inventory did not bind the runtime permission probe to the exact artifact: %#v", item)
	}

	unloaded, err := h.broker.UnloadExtension(ctx, h.entry.sessionID, artifact.ExtensionID)
	if err != nil || unloaded.Operation != "unload" || unloaded.State != "unloaded" {
		t.Fatalf("unload receipt=%#v err=%v", unloaded, err)
	}
	loaded, err := h.broker.LoadExtension(ctx, h.entry.sessionID, artifact.ExtensionID)
	if err != nil || loaded.Operation != "load" || loaded.State != "loaded" {
		t.Fatalf("load receipt=%#v err=%v", loaded, err)
	}
	if got := downloads.Load(); got != 2 {
		t.Fatalf("artifact owner downloads = %d, want one verified fetch for install and one after unload cleanup", got)
	}
	if _, err := h.broker.UnloadExtension(ctx, h.entry.sessionID, artifact.ExtensionID); err != nil {
		t.Fatalf("final unload did not release staged artifact: %v", err)
	}
}

func makeSignedExtensionArchive(t *testing.T) (ExtensionArtifact, []byte) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	key := base64.StdEncoding.EncodeToString(der)
	manifest := struct {
		ManifestVersion         int      `json:"manifest_version"`
		Name                    string   `json:"name"`
		Version                 string   `json:"version"`
		Key                     string   `json:"key"`
		Permissions             []string `json:"permissions"`
		OptionalPermissions     []string `json:"optional_permissions"`
		HostPermissions         []string `json:"host_permissions"`
		OptionalHostPermissions []string `json:"optional_host_permissions"`
	}{
		ManifestVersion: 3, Name: "Broker integration extension", Version: "1.0.0", Key: key,
		Permissions: []string{"storage"}, HostPermissions: []string{"https://example.test/*"},
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestHash := sha256.Sum256(manifestBytes)
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	entry, err := writer.Create("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(manifestBytes); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"__vrooli_bas/permission-report.html": "same-extension report page",
		"__vrooli_bas/permission-report.js":   "chrome.permissions.getAll()",
	} {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	archiveHash := sha256.Sum256(archive.Bytes())
	extensionID, err := extensionIDFromKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return ExtensionArtifact{
		ArtifactRef: "signed-build", ExtensionID: extensionID,
		Digest: hex.EncodeToString(archiveHash[:]), ManifestSHA256: hex.EncodeToString(manifestHash[:]),
		Version: "1.0.0", Permissions: []string{"storage"}, HostPermissions: []string{"https://example.test/*"},
		PermissionProbe: extensionPermissionProbeChromeGetAllV1,
	}, archive.Bytes()
}
