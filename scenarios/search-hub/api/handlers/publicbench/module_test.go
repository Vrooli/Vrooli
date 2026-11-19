package publicbench

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestStagedSciFactProviderSearchesFullCorpus(t *testing.T) {
	dir := t.TempDir()
	body := "{\"doc_id\":4001,\"title\":\"CRISPR gene therapy\",\"abstract\":[\"CRISPR gene therapy changed treatment outcomes in this study.\"]}\n" +
		"{\"doc_id\":4002,\"title\":\"Ocean temperature\",\"abstract\":[\"Marine temperatures vary by season in a study.\"]}\n"
	if err := os.WriteFile(filepath.Join(dir, "corpus.jsonl"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	idx, err := loadIndex(dir)
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(dir).Mount(router)
	reqBody := fmt.Sprintf(`{"query":"CRISPR gene therapy study","limit":10,"expected_corpus_sha256":"%s"}`, idx.CorpusSHA256)
	req := httptest.NewRequest(http.MethodPost, endpointPath, strings.NewReader(reqBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"id":"4001"`) || !strings.Contains(rec.Body.String(), `"id":"4002"`) {
		t.Fatalf("provider did not return matches from the full staged corpus: %s", rec.Body.String())
	}
	if strings.Index(rec.Body.String(), `"id":"4001"`) > strings.Index(rec.Body.String(), `"id":"4002"`) {
		t.Fatalf("gold document not ranked first: %s", rec.Body.String())
	}
}

func TestStagedSciFactProviderRejectsMismatchedSourceDigest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "corpus.jsonl"), []byte("{\"doc_id\":1,\"title\":\"Title\",\"abstract\":[\"abstract text\"]}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(dir).Mount(router)
	req := httptest.NewRequest(http.MethodPost, endpointPath, strings.NewReader(`{"query":"abstract","limit":5,"expected_corpus_sha256":"stale"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestStagedSciFactProviderIsUnavailableWithoutData(t *testing.T) {
	router := mux.NewRouter()
	Module("").Mount(router)
	req := httptest.NewRequest(http.MethodPost, endpointPath, strings.NewReader(`{"query":"claim"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
