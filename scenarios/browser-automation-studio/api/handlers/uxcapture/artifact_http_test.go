package uxcapture

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	bundle "github.com/vrooli/browser-automation-studio/services/uxcapture"
)

func makeArtifactBundle(t *testing.T, root string) (string, string, []byte) {
	return makeArtifactBundleWithPrivacy(t, root, bundle.Privacy{RedactionStatus: "complete"})
}

func makeArtifactBundleWithPrivacy(t *testing.T, root string, privacy bundle.Privacy) (string, string, []byte) {
	t.Helper()
	id := uuid.NewString()
	content := []byte("<html><body>captured</body></html>")
	manifest := bundle.Bundle{
		SchemaVersion: bundle.SchemaVersion,
		ID:            id,
		CapturedAt:    "2026-11-14T05:00:00Z",
		Source:        bundle.Source{Kind: bundle.SourceFile, Filename: "capture.html"},
		Session:       bundle.Session{ID: "session"},
		Viewport:      bundle.Viewport{Width: 1280, Height: 720},
		Lenses: map[bundle.LensName]bundle.Lens{
			bundle.LensPixels: {Status: bundle.LensPresent}, bundle.LensFrozenPage: {Status: bundle.LensPresent},
			bundle.LensOutline: {Status: bundle.LensAbsent, Reason: "fixture"}, bundle.LensStyle: {Status: bundle.LensAbsent, Reason: "fixture"},
			bundle.LensIdentity: {Status: bundle.LensAbsent, Reason: "fixture"}, bundle.LensNetwork: {Status: bundle.LensAbsent, Reason: "fixture"},
			bundle.LensStates: {Status: bundle.LensAbsent, Reason: "fixture"},
		},
		Privacy:   privacy,
		Artifacts: []bundle.Artifact{{Kind: "frozen-page", Path: "page/frozen.html", MediaType: "text/html", SizeBytes: int64(len(content)), SHA256: bundle.Digest(content)}},
	}
	if err := bundle.WriteBundleWithArtifacts(filepath.Join(root, "ux", id), manifest, []bundle.ArtifactPayload{{Path: "page/frozen.html", Content: content}}); err != nil {
		t.Fatal(err)
	}
	return id, filepath.Join(root, "ux", id, "page", "frozen.html"), content
}

func TestDeleteArtifactHTTPRemovesOnlyExternalEphemeralBundles(t *testing.T) {
	root := t.TempDir()
	id, _, _ := makeArtifactBundleWithPrivacy(t, root, bundle.Privacy{RedactionStatus: "complete", External: true, Ephemeral: true})
	router := chi.NewRouter()
	MountArtifactHTTP(router, root)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("DELETE", "/ux-capture/bundles/"+id, nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s, want 204", response.Code, response.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "ux", id)); !os.IsNotExist(err) {
		t.Fatalf("ephemeral bundle directory remains: %v", err)
	}
}

func TestDeleteArtifactHTTPRefusesRetainedScenarioBundles(t *testing.T) {
	root := t.TempDir()
	id, _, _ := makeArtifactBundle(t, root)
	router := chi.NewRouter()
	MountArtifactHTTP(router, root)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("DELETE", "/ux-capture/bundles/"+id, nil))
	if response.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s, want 409", response.Code, response.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "ux", id)); err != nil {
		t.Fatalf("retained scenario bundle was removed: %v", err)
	}
}

func TestArtifactRoutesShareBundleIDValidation(t *testing.T) {
	router := chi.NewRouter()
	MountArtifactHTTP(router, t.TempDir())
	for _, tc := range []struct {
		name   string
		method string
		path   string
	}{
		{name: "delete", method: http.MethodDelete, path: "/ux-capture/bundles/not-a-uuid"},
		{name: "artifact read", method: http.MethodGet, path: "/ux-capture/bundles/not-a-uuid/artifacts/page/frozen.html"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d body=%s, want 404", response.Code, response.Body.String())
			}
		})
	}
}

func TestArtifactReadValidatesBundleIDBeforeStorageAndPreservesBytesAndMIME(t *testing.T) {
	noStorage := chi.NewRouter()
	MountArtifactHTTP(noStorage, "")
	invalid := httptest.NewRecorder()
	noStorage.ServeHTTP(invalid, httptest.NewRequest(http.MethodGet, "/ux-capture/bundles/not-a-uuid/artifacts/page/frozen.html", nil))
	if invalid.Code != http.StatusNotFound {
		t.Fatalf("malformed bundle id status=%d body=%s, want 404 before storage readiness", invalid.Code, invalid.Body.String())
	}
	validIDWithoutStorage := httptest.NewRecorder()
	noStorage.ServeHTTP(validIDWithoutStorage, httptest.NewRequest(http.MethodGet, "/ux-capture/bundles/11111111-1111-4111-8111-111111111111/artifacts/page/frozen.html", nil))
	if validIDWithoutStorage.Code != http.StatusServiceUnavailable {
		t.Fatalf("valid id without storage status=%d body=%s, want 503", validIDWithoutStorage.Code, validIDWithoutStorage.Body.String())
	}

	root := t.TempDir()
	id, _, content := makeArtifactBundle(t, root)
	response := artifactRequest(t, root, id, "page/frozen.html")
	if response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), content) {
		t.Fatalf("valid artifact response=%d %q, want exact %q", response.Code, response.Body.Bytes(), content)
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("valid artifact MIME=%q, want declared text/html identity", got)
	}
}

func artifactRequest(t *testing.T, root, bundleID, path string) *httptest.ResponseRecorder {
	t.Helper()
	router := chi.NewRouter()
	MountArtifactHTTP(router, root)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/ux-capture/bundles/"+bundleID+"/artifacts/"+path, nil))
	return response
}

func TestMountArtifactHTTPReturnsDeclaredVerifiedBytes(t *testing.T) {
	root := t.TempDir()
	id, _, content := makeArtifactBundle(t, root)
	response := artifactRequest(t, root, id, "page/frozen.html")
	if response.Code != 200 || response.Body.String() != string(content) {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "text/html; charset=utf-8" || response.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.Contains(response.Header().Get("Cache-Control"), "no-store") || response.Header().Get("Content-Disposition") != "attachment" {
		t.Fatalf("unsafe or missing headers: %v", response.Header())
	}
}

func TestMountArtifactHTTPRejectsTampering(t *testing.T) {
	root := t.TempDir()
	id, path, _ := makeArtifactBundle(t, root)
	if err := os.WriteFile(path, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if response := artifactRequest(t, root, id, "page/frozen.html"); response.Code != 422 {
		t.Fatalf("status=%d body=%s, want integrity failure", response.Code, response.Body.String())
	}
}

func TestMountArtifactHTTPRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	id, path, _ := makeArtifactBundle(t, root)
	outside := filepath.Join(t.TempDir(), "outside.html")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if response := artifactRequest(t, root, id, "page/frozen.html"); response.Code != 404 {
		t.Fatalf("status=%d body=%s, want not found", response.Code, response.Body.String())
	}
}

func TestMountArtifactHTTPRejectsTraversalMalformedMissingAndUndeclared(t *testing.T) {
	root := t.TempDir()
	id, _, _ := makeArtifactBundle(t, root)
	cases := []struct {
		name string
		url  string
		want int
	}{
		{"malformed bundle id", "/ux-capture/bundles/not-an-id/artifacts/page/frozen.html", 404},
		{"undeclared artifact", "/ux-capture/bundles/" + id + "/artifacts/README.md", 404},
		{"missing bundle", "/ux-capture/bundles/" + uuid.NewString() + "/artifacts/page/frozen.html", 404},
		{"traversal", "/ux-capture/bundles/" + id + "/artifacts/%2e%2e/manifest.json", 404},
	}
	router := chi.NewRouter()
	MountArtifactHTTP(router, root)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest("GET", tc.url, nil))
			if response.Code != tc.want {
				t.Fatalf("status=%d body=%s, want %d", response.Code, response.Body.String(), tc.want)
			}
		})
	}
}

func TestSafeArtifactRequestPathRejectsUnsafeForms(t *testing.T) {
	for _, path := range []string{"", "/abs", "../secret", "a/../secret", "a//b", `a\\b`, "a\x00b"} {
		if safeArtifactRequestPath(path) {
			t.Errorf("accepted unsafe artifact path %q", path)
		}
	}
}
