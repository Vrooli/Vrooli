package runs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestStoreExperienceCohortCallPostsOneValidatedRequest(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != cohortReceiptEndpoint {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"receipt":{"schemaVersion":1,"receiptId":"immutable-id","scenarios":[{}]},"artifact":{"id":"immutable-id","sha256":"abc"}}`))
	}))
	defer server.Close()
	previous := cohortReceiptHTTPClient
	cohortReceiptHTTPClient = server.Client()
	t.Cleanup(func() { cohortReceiptHTTPClient = previous })
	apiClient := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{Client: server.Client()}), func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{Override: server.URL} }, nil)
	response, err := storeExperienceCohortCall(apiClient, `{"cancelled":true,"scenarios":[{"scenario":"demo","runId":"r1","invokerStatus":"cancelled"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if got["cancelled"] != true {
		t.Fatalf("request body = %#v", got)
	}
	if !json.Valid(response) {
		t.Fatalf("response = %s", response)
	}
}
