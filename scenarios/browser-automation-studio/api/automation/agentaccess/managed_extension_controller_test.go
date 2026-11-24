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
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

func TestManagedExtensionControllerStagesOnlyDigestAndManifestMatchedArtifact(t *testing.T) {
	artifact, archive := makeManagedExtensionArtifact(t)
	var downloads int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloads++
		if r.URL.Path != "/api/v1/extension/download/build-1" {
			t.Errorf("download path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(archive)
	}))
	defer server.Close()
	root := t.TempDir()
	t.Cleanup(func() { makeExtensionTreeWritable(root) })
	controller, err := NewManagedExtensionController(session.NewManagerWithClient(nil), server.URL, WithManagedExtensionStagingRoot(root))
	if err != nil {
		t.Fatal(err)
	}
	binding := testExtensionBinding("owner-1", "lease-1")
	path, err := controller.stageArtifact(context.Background(), binding, artifact)
	if err != nil {
		t.Fatal(err)
	}
	if downloads != 1 {
		t.Fatalf("artifact downloads = %d, want 1", downloads)
	}
	manifest, err := os.ReadFile(filepath.Join(path, "manifest.json"))
	if err != nil || len(manifest) == 0 {
		t.Fatalf("staged manifest read err=%v bytes=%d", err, len(manifest))
	}
	if filepath.Base(filepath.Dir(path)) != artifact.Digest {
		t.Fatalf("staged artifact is not under its digest directory: %s", path)
	}
	if mode, err := os.Stat(path); err != nil || mode.Mode().Perm() != 0o500 {
		t.Fatalf("staged directory mode=%v err=%v, want 0500", mode, err)
	}

	secondBinding := testExtensionBinding("owner-2", "lease-2")
	secondBinding.SessionID = "session-2"
	otherPath, err := controller.stageArtifact(context.Background(), secondBinding, artifact)
	if err != nil {
		t.Fatal(err)
	}
	if path == otherPath {
		t.Fatal("staged artifact directory was shared across different BAS session owners")
	}
	if downloads != 2 {
		t.Fatalf("per-binding staged artifact downloads = %d, want 2", downloads)
	}
	if err := controller.FinalizeClosedSession(context.Background(), binding.SessionID); err != nil {
		t.Fatalf("finalize first closed session: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("closed session staging path still exists: %s err=%v", path, err)
	}
	if _, err := os.Stat(otherPath); err != nil {
		t.Fatalf("cleanup crossed into another session's staging path: %s err=%v", otherPath, err)
	}
	if err := controller.FinalizeClosedSession(context.Background(), secondBinding.SessionID); err != nil {
		t.Fatalf("finalize second closed session: %v", err)
	}
	if _, err := os.Stat(otherPath); !os.IsNotExist(err) {
		t.Fatalf("second closed session staging path still exists: %s err=%v", otherPath, err)
	}
}

func TestBrokerCloseRemovesSessionBoundExtensionStagingAfterDriverClose(t *testing.T) {
	artifact, archive := makeManagedExtensionArtifact(t)
	artifactOwner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/extension/download/"+artifact.ArtifactRef {
			http.Error(w, "unexpected artifact request", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(archive)
	}))
	defer artifactOwner.Close()

	h := newPermissionParityHarness(t, string(sessionpermission.BackendManaged))
	defer h.server.Close()
	stagingRoot := t.TempDir()
	t.Cleanup(func() { makeExtensionTreeWritable(stagingRoot) })
	controller, err := NewManagedExtensionController(
		h.broker.manager, artifactOwner.URL,
		WithManagedExtensionStagingRoot(filepath.Join(stagingRoot, "staged-extensions")),
	)
	if err != nil {
		t.Fatal(err)
	}
	h.broker.extensions = controller
	h.broker.mu.RLock()
	entry := h.broker.entries[h.info.SessionID]
	h.broker.mu.RUnlock()
	binding, err := h.broker.extensionSessionBinding(entry)
	if err != nil {
		t.Fatal(err)
	}
	path, err := controller.stageArtifact(h.ctx, binding, artifact)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.broker.Close(h.ctx, h.info.SessionID); err != nil {
		t.Fatalf("close managed browser session: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("session-owned extension staging survived browser close: %s err=%v", path, err)
	}
}

func makeExtensionTreeWritable(root string) {
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		mode := os.FileMode(0o600)
		if entry.IsDir() {
			mode = 0o700
		}
		_ = os.Chmod(path, mode)
		return nil
	})
}

func TestManagedExtensionControllerRejectsRedirectsAndDigestMismatch(t *testing.T) {
	artifact, archive := makeManagedExtensionArtifact(t)
	var redirected int
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { redirected++ }))
	defer target.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/unexpected", http.StatusFound)
	}))
	defer redirector.Close()
	controller, err := NewManagedExtensionController(session.NewManagerWithClient(nil), redirector.URL, WithManagedExtensionStagingRoot(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := controller.downloadArtifact(context.Background(), artifact); err == nil {
		t.Fatal("artifact download followed or accepted an HTTP redirect")
	}
	if redirected != 0 {
		t.Fatalf("redirect target requests = %d, want 0", redirected)
	}

	badDigestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(archive) }))
	defer badDigestServer.Close()
	artifact.Digest = strings.Repeat("0", 64)
	badController, err := NewManagedExtensionController(session.NewManagerWithClient(nil), badDigestServer.URL, WithManagedExtensionStagingRoot(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := badController.downloadArtifact(context.Background(), artifact); err == nil || !strings.Contains(err.Error(), "differ from the verified") {
		t.Fatalf("digest mismatch error = %v", err)
	}
}

func TestVerifyExtensionArchiveRejectsPathTraversal(t *testing.T) {
	artifact, _ := makeManagedExtensionArtifact(t)
	archive := makeExtensionZip(t, []zipEntry{
		{name: "manifest.json", body: `{"manifest_version":3}`},
		{name: "../escape.txt", body: "outside"},
	})
	if err := extractExtensionArchive(archive, t.TempDir()); err == nil {
		t.Fatal("path traversal in ZIP entries was accepted")
	}
	if err := verifyExtensionArchive(archive, artifact); err == nil {
		t.Fatal("unverified ZIP digest must not be accepted as a verified artifact")
	}
}

func TestVerifyExtensionArchiveRequiresArtifactBoundPermissionProbeAssets(t *testing.T) {
	artifact, completeProbe := makeManagedExtensionArtifact(t)
	archive, err := zip.NewReader(bytes.NewReader(completeProbe), int64(len(completeProbe)))
	if err != nil {
		t.Fatal(err)
	}
	var manifest []byte
	for _, entry := range archive.File {
		if entry.Name != "manifest.json" {
			continue
		}
		reader, openErr := entry.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		manifest, err = io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil || closeErr != nil {
			t.Fatalf("read manifest err=%v close=%v", err, closeErr)
		}
	}
	if len(manifest) == 0 {
		t.Fatal("fixture archive has no manifest")
	}
	missingProbe := makeExtensionZip(t, []zipEntry{{name: "manifest.json", body: string(manifest)}})
	if err := verifyExtensionArchive(missingProbe, artifact); err == nil || !strings.Contains(err.Error(), "permission probe") {
		t.Fatalf("artifact with declared permission probe but missing assets was accepted: %v", err)
	}

	if err := verifyExtensionArchive(completeProbe, artifact); err != nil {
		t.Fatalf("artifact-bound permission probe contract was rejected: %v", err)
	}
	unbound := artifact
	unbound.PermissionProbe = ""
	if err := verifyExtensionArchive(completeProbe, unbound); err == nil || !strings.Contains(err.Error(), "not declared") {
		t.Fatalf("undeclared permission probe assets were accepted: %v", err)
	}
}

type zipEntry struct{ name, body string }

func makeManagedExtensionArtifact(t *testing.T) (ExtensionArtifact, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	encodedKey := base64.StdEncoding.EncodeToString(der)
	extensionID, err := extensionIDFromKey(encodedKey)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := json.Marshal(map[string]any{
		"manifest_version":     3,
		"name":                 "Test extension",
		"version":              "1.0.0",
		"key":                  encodedKey,
		"permissions":          []string{"storage"},
		"optional_permissions": []string{"tabs"},
		"host_permissions":     []string{"https://example.test/*"},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifestHash := sha256.Sum256(manifest)
	archive := makeExtensionZip(t, []zipEntry{
		{name: "manifest.json", body: string(manifest)},
		{name: "background.js", body: "chrome.runtime.onInstalled.addListener(() => {});"},
		{name: "__vrooli_bas/permission-report.html", body: "same-extension report page"},
		{name: "__vrooli_bas/permission-report.js", body: "chrome.permissions.getAll()"},
	})
	digest := sha256.Sum256(archive)
	return ExtensionArtifact{
		ArtifactRef: "build-1", ExtensionID: extensionID,
		Digest: hex.EncodeToString(digest[:]), ManifestSHA256: hex.EncodeToString(manifestHash[:]),
		Version: "1.0.0", Permissions: []string{"storage", "tabs"}, HostPermissions: []string{"https://example.test/*"},
		PermissionProbe: extensionPermissionProbeChromeGetAllV1,
	}, archive
}

func makeExtensionZip(t *testing.T, entries []zipEntry) []byte {
	t.Helper()
	var output strings.Builder
	writer := zip.NewWriter(stringWriter{&output})
	for _, entry := range entries {
		file, err := writer.Create(entry.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(file, entry.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return []byte(output.String())
}

type stringWriter struct{ builder *strings.Builder }

func (w stringWriter) Write(data []byte) (int, error) { return w.builder.Write(data) }

func testExtensionBinding(owner, lease string) ExtensionSessionBinding {
	return ExtensionSessionBinding{
		SessionID: "session-1", Backend: "managed", ProfileID: "profile-1",
		PermissionRef: "permission-1", OwnerID: owner, LeaseID: lease, CallerID: "caller-1",
	}
}
