package runtimehandlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/rootrebaselinedecision"
	"github.com/vrooli/vrooli/internal/rootdecision"
)

func TestRetireSnapshotDecisionRequiresExactAuthenticatedReceipt(t *testing.T) {
	dir := t.TempDir()
	sha := strings.Repeat("a", 64)
	manifest := []byte("prepared bundle manifest")
	if err := os.WriteFile(filepath.Join(dir, "SHA256SUMS"), manifest, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(manifest)
	terms := rootDecisionTerms{RequesterRef: "req", Deadline: "2020-01-01T00:00:00Z", CandidateSHA: sha, ServingSHA: sha, WatchdogSHA: sha, BundleSHA: hex.EncodeToString(sum[:]), CandidateContract: sha, ServingContract: sha, Changes: []string{"state"}}
	if err := rootdecision.Save(filepath.Join(dir, rootdecision.PendingFilename), terms); err != nil {
		t.Fatal(err)
	}
	digest, err := rootrebaselinedecision.Digest(terms)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		status   int
		response string
		wantErr  bool
	}{
		{"sealed", 200, `{"requester_ref":"req","canonical_terms_sha256":"` + digest + `","state":"retired"}`, false},
		{"consumed", 409, `{}`, true}, {"unavailable", 503, `{}`, true}, {"mismatch", 200, `{"requester_ref":"other","canonical_terms_sha256":"` + digest + `","state":"retired"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/internal/root-rebaseline/retire" || r.Header.Get("Authorization") != "Bearer secret" {
					t.Error("incorrect authenticated retirement request")
				}
				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["requester_ref"] != "req" || body["canonical_terms_sha256"] != digest {
					t.Errorf("bad identity: %v %v", body, err)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.response))
			}))
			defer server.Close()
			c := &rootDecisionClient{base: server.URL, http: server.Client(), token: func(context.Context) (string, error) { return "secret", nil }}
			if err := retireSnapshotDecision(context.Background(), c, dir); (err != nil) != tc.wantErr {
				t.Fatalf("error: %v", err)
			}
		})
	}
	if err := retireSnapshotDecision(context.Background(), nil, t.TempDir()); err == nil {
		t.Fatal("missing locator must fail closed")
	}
}
