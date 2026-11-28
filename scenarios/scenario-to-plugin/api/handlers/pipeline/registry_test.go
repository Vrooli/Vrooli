package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestOCIRegistryPushConfirmsArtifactAndReferrers(t *testing.T) {
	var mu sync.Mutex
	manifests := map[string][]byte{}
	referrers := []ociReferrerDescriptor{}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/blobs/uploads") {
			w.Header().Set("Location", server.URL+"/uploads/1")
			w.WriteHeader(http.StatusAccepted)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/uploads/") && r.Method == http.MethodPut {
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusCreated)
			return
		}
		if strings.Contains(r.URL.Path, "/manifests/") {
			key := strings.TrimPrefix(r.URL.Path, "/v2/vrooli/plugin-test/manifests/")
			mu.Lock()
			defer mu.Unlock()
			if r.Method == http.MethodPut {
				body, _ := io.ReadAll(r.Body)
				manifests[key] = body
				var parsed struct {
					ArtifactType string         `json:"artifactType"`
					Subject      map[string]any `json:"subject"`
				}
				if json.Unmarshal(body, &parsed) == nil && parsed.Subject != nil {
					referrers = append(referrers, ociReferrerDescriptor{Digest: key, ArtifactType: parsed.ArtifactType})
				}
				w.WriteHeader(http.StatusCreated)
				return
			}
			body, ok := manifests[key]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			_, _ = w.Write(body)
			return
		}
		if strings.Contains(r.URL.Path, "/referrers/") && r.Method == http.MethodGet {
			mu.Lock()
			defer mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"schemaVersion": 2, "manifests": referrers})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	registry := &ociRegistry{base: server.URL, repository: "vrooli/plugin-test", client: server.Client()}
	coordinate, err := registry.pushPackage(t.Context(), "0.1.0", []byte("plugin"), []byte("signature"), []byte("provenance"), []byte("sbom"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(coordinate, "/manifests/0.1.0") {
		t.Fatalf("coordinate = %q", coordinate)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(manifests) != 4 {
		t.Fatalf("manifest count = %d, want package plus three referrers", len(manifests))
	}
}

func TestOCIRegistryConfirmReferrersRequiresDistinctDigestBoundEvidence(t *testing.T) {
	const subject = "sha256:artifact"
	kinds := []string{
		"application/vnd.dev.cosign.signature",
		"application/vnd.in-toto+json",
		"application/vnd.cyclonedx+json",
	}
	manifests := map[string][]byte{}
	valid := make([]ociReferrerDescriptor, 0, len(kinds))
	for _, kind := range kinds {
		body, err := json.Marshal(map[string]any{
			"artifactType": kind,
			"subject":      map[string]any{"digest": subject},
		})
		if err != nil {
			t.Fatal(err)
		}
		digest := digestBytes(body)
		manifests[digest] = body
		valid = append(valid, ociReferrerDescriptor{Digest: digest, ArtifactType: kind})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		digest := strings.TrimPrefix(r.URL.Path, "/v2/repo/manifests/")
		body, ok := manifests[digest]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	defer server.Close()
	registry := &ociRegistry{base: server.URL, repository: "repo", client: server.Client()}

	tests := []struct {
		name      string
		referrers []ociReferrerDescriptor
		wantError bool
	}{
		{name: "all distinct and bound", referrers: valid},
		{name: "missing sbom", referrers: valid[:2], wantError: true},
		{name: "duplicate signature cannot replace provenance", referrers: []ociReferrerDescriptor{valid[0], valid[0], valid[2]}, wantError: true},
		{name: "digest only cannot replace typed evidence", referrers: []ociReferrerDescriptor{{Digest: valid[0].Digest}, valid[1], valid[2]}, wantError: true},
		{name: "unknown type ignored", referrers: []ociReferrerDescriptor{{Digest: valid[0].Digest, ArtifactType: "application/vnd.unknown"}, valid[1], valid[2]}, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := registry.confirmReferrers(context.Background(), subject, test.referrers)
			if (err != nil) != test.wantError {
				t.Fatalf("confirmReferrers error = %v, wantError %t", err, test.wantError)
			}
		})
	}

	wrongSubjectBody, _ := json.Marshal(map[string]any{
		"artifactType": kinds[2],
		"subject":      map[string]any{"digest": "sha256:other"},
	})
	wrongSubjectDigest := digestBytes(wrongSubjectBody)
	manifests[wrongSubjectDigest] = wrongSubjectBody
	wrongSubject := append([]ociReferrerDescriptor(nil), valid...)
	wrongSubject[2] = ociReferrerDescriptor{Digest: wrongSubjectDigest, ArtifactType: kinds[2]}
	if err := registry.confirmReferrers(context.Background(), subject, wrongSubject); err == nil || !strings.Contains(err.Error(), kinds[2]) {
		t.Fatalf("wrong-subject evidence error = %v, want missing %s", err, kinds[2])
	}

	missingManifest := append([]ociReferrerDescriptor(nil), valid...)
	missingManifest[2] = ociReferrerDescriptor{Digest: fmt.Sprintf("sha256:%064d", 0), ArtifactType: kinds[2]}
	if err := registry.confirmReferrers(context.Background(), subject, missingManifest); err == nil {
		t.Fatal("missing manifest was accepted")
	}
}
