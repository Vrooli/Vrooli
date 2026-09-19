package eventbus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadRuntimeStateUsesSharedHealthHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		w.Header().Set(RuntimeStateHeader, "connected_empty")
		w.Header().Set(RuntimeArmedHeader, "true")
		w.Header().Set(RuntimePolicyCountHeader, "0")
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	state, err := ReadRuntimeState(context.Background(), server.URL, server.Client())
	if err != nil || state.State != "connected_empty" || !state.Armed || state.PolicyCount != 0 {
		t.Fatalf("state=%+v err=%v", state, err)
	}
}

func TestRuntimeStateFromHeadersRejectsMissingContract(t *testing.T) {
	if _, err := RuntimeStateFromHeaders(http.Header{}); err == nil {
		t.Fatal("missing contract unexpectedly accepted")
	}
}
