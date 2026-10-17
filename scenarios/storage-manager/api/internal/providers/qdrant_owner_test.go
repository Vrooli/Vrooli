package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPQdrantGenerationReaderRequiresTokenAndPreservesOwnerReport(t *testing.T) {
	var receivedToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("X-Search-Control-Token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"owner":"agent-manager.runs","namespace":"conversation-search","alias":"agent-manager_conversation-search","activeCollection":"active","generations":[{"collectionName":"active","state":"active","bytes":42}]}`))
	}))
	defer server.Close()
	reader := NewHTTPQdrantGenerationReader(func(context.Context, string) (string, error) { return server.URL, nil }, server.Client(), func() string { return "secret" })
	inspection, err := reader.InspectQdrantGenerations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if receivedToken != "secret" || inspection.Owner != "agent-manager.runs" || inspection.Generations[0].Bytes != 42 {
		t.Fatalf("token=%q inspection=%+v", receivedToken, inspection)
	}
	reader.Token = func() string { return "" }
	if _, err := reader.InspectQdrantGenerations(context.Background()); err == nil {
		t.Fatal("missing token unexpectedly succeeded")
	}
}
