package agentaccess

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScenarioToExtensionVerifierBindsOwnerMetadataToExactBuildAndDigest(t *testing.T) {
	digest, manifestDigest := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/extension/artifact/build-42/metadata" || r.URL.Query().Get("expected_digest") != digest {
			t.Fatalf("unexpected artifact owner request: %s", r.URL.String())
		}
		if r.URL.Query().Has("extension_id") {
			t.Fatal("caller-provided extension ID was sent to artifact authority")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(scenarioToExtensionArtifactResponse{
			ArtifactRef: "build-42", ArtifactSHA256: digest, ManifestSHA256: manifestDigest,
			ExtensionID: "abcdefghijklmnopabcdefghijklmnop", Version: "1.2.3",
			Permissions: []string{"storage"}, HostPermissions: []string{"https://example.test/*"},
			PermissionProbe: extensionPermissionProbeChromeGetAllV1,
		})
	}))
	defer server.Close()

	verifier, err := NewScenarioToExtensionArtifactVerifier(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := verifier.VerifyBrowserExtensionArtifact(context.Background(), "build-42", digest)
	if err != nil {
		t.Fatal(err)
	}
	if artifact.ArtifactRef != "build-42" || artifact.Digest != digest || artifact.ManifestSHA256 != manifestDigest || artifact.ExtensionID != "abcdefghijklmnopabcdefghijklmnop" || artifact.Version != "1.2.3" || artifact.PermissionProbe != extensionPermissionProbeChromeGetAllV1 {
		t.Fatalf("artifact owner response was not preserved: %#v", artifact)
	}
}

func TestScenarioToExtensionVerifierRejectsUnboundOrMalformedOwnerMetadata(t *testing.T) {
	digest := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	for name, payload := range map[string]scenarioToExtensionArtifactResponse{
		"wrong build":                   {ArtifactRef: "other", ArtifactSHA256: digest, ManifestSHA256: digest, ExtensionID: "abcdefghijklmnopabcdefghijklmnop", Version: "1.0"},
		"wrong digest":                  {ArtifactRef: "build", ArtifactSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", ManifestSHA256: digest, ExtensionID: "abcdefghijklmnopabcdefghijklmnop", Version: "1.0"},
		"malformed id":                  {ArtifactRef: "build", ArtifactSHA256: digest, ManifestSHA256: digest, ExtensionID: "caller-id", Version: "1.0"},
		"missing manifest digest":       {ArtifactRef: "build", ArtifactSHA256: digest, ExtensionID: "abcdefghijklmnopabcdefghijklmnop", Version: "1.0"},
		"duplicate permissions":         {ArtifactRef: "build", ArtifactSHA256: digest, ManifestSHA256: digest, ExtensionID: "abcdefghijklmnopabcdefghijklmnop", Version: "1.0", Permissions: []string{"storage", "storage"}},
		"unsupported permission source": {ArtifactRef: "build", ArtifactSHA256: digest, ManifestSHA256: digest, ExtensionID: "abcdefghijklmnopabcdefghijklmnop", Version: "1.0", PermissionProbe: "manifest-only"},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _ = json.NewEncoder(w).Encode(payload) }))
			defer server.Close()
			verifier, err := NewScenarioToExtensionArtifactVerifier(server.URL, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := verifier.VerifyBrowserExtensionArtifact(context.Background(), "build", digest); err == nil {
				t.Fatal("unbound metadata was accepted")
			}
		})
	}
}

func TestScenarioToExtensionVerifierRejectsRedirectAndInvalidOwnerURL(t *testing.T) {
	targetHit := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { targetHit = true; w.WriteHeader(http.StatusOK) }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer redirect.Close()
	verifier, err := NewScenarioToExtensionArtifactVerifier(redirect.URL, redirect.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.VerifyBrowserExtensionArtifact(context.Background(), "build", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"); err == nil {
		t.Fatal("artifact verifier followed or accepted a redirect")
	}
	if targetHit {
		t.Fatal("artifact verifier forwarded request to redirect target")
	}
	for _, invalid := range []string{"file:///tmp/artifacts", "https://user:pass@example.test", "https://example.test/path"} {
		if _, err := NewScenarioToExtensionArtifactVerifier(invalid, nil); err == nil {
			t.Fatalf("unsafe owner URL accepted: %q", invalid)
		}
	}
}
