package opencode

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReplyPermissionPostsTheReply(t *testing.T) {
	// [REQ:P0-017i] OpenCode's permission reply: POST /permission/{requestID}/reply {"reply": ...}.
	var gotRequest, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequest = r.Method + " " + r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("true"))
	}))
	defer srv.Close()
	if err := NewHTTPClient(srv.URL).ReplyPermission(context.Background(), "per_1", "once"); err != nil {
		t.Fatal(err)
	}
	if gotRequest != "POST /permission/per_1/reply" || gotBody != `{"reply":"once"}` {
		t.Fatalf("request = %s %s, want POST /permission/per_1/reply {\"reply\":\"once\"}", gotRequest, gotBody)
	}
}

func TestReplyPermissionReportsARefusedReply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()
	if err := NewHTTPClient(srv.URL).ReplyPermission(context.Background(), "per_1", "once"); err == nil {
		t.Fatal("a refused reply returned no error")
	}
}
