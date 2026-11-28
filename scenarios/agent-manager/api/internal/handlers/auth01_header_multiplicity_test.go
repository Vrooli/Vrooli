package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/api-core/owneridentity"
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

// P-18: proof absence is refused only while the AUTH-01 switch is on. With it
// off an unattended caller proceeds to ordinary request validation.
func TestAuth01AbsentProofFollowsEnforcementSwitch(t *testing.T) {
	h := &Handler{}
	t.Setenv(owneridentity.CreateRunCallerEnforceEnv, "true")
	r := httptest.NewRequest(http.MethodPost, "/runs", nil)
	r.Body = io.NopCloser(auth01UnreadBody{t})
	w := httptest.NewRecorder()
	h.CreateRun(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("absent proof admitted while enforced: %d", w.Code)
	}
	t.Setenv(owneridentity.CreateRunCallerEnforceEnv, "")
	r = httptest.NewRequest(http.MethodPost, "/runs", strings.NewReader("not-json"))
	w = httptest.NewRecorder()
	h.CreateRun(w, r)
	if w.Code == http.StatusUnauthorized {
		t.Fatal("unattended caller refused while enforcement is off")
	}
}
