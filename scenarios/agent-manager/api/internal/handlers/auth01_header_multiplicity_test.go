package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuth01RepeatedProofFieldsRejectBeforeBodyOrService(t *testing.T) {
	h := &Handler{}
	for _, headers := range []http.Header{{"Authorization": []string{"Bearer fixture"}, "X-Agent-Identity-Token": []string{""}}, {"Authorization": []string{""}, "X-Agent-Identity-Token": []string{"fixture"}}, {"Authorization": []string{"Bearer fixture", "Bearer invalid"}}, {"Authorization": []string{"Bearer fixture", ""}}, {"X-Agent-Identity-Token": []string{"fixture", ""}}, {"Authorization": []string{"Bearer fixture"}, "X-Agent-Identity-Token": []string{"fixture", "invalid"}}} {
		r := httptest.NewRequest(http.MethodPost, "/runs", nil)
		r.Header = headers
		r.Body = io.NopCloser(auth01UnreadBody{t})
		w := httptest.NewRecorder()
		h.CreateRun(w, r)
		if w.Code != http.StatusUnauthorized && w.Code != http.StatusBadRequest {
			t.Fatalf("ambiguous proof accepted%d", w.Code)
		}
	}
}

type auth01UnreadBody struct{ t *testing.T }

func (b auth01UnreadBody) Read([]byte) (int, error) {
	b.t.Fatal("invalid offered proof caused body read")
	return 0, io.EOF
}
