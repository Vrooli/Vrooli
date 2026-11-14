package agentsessions

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuth01SessionStartRejectsBeforeServiceEffects(t *testing.T) {
	h := &Handler{}
	for _, headers := range []http.Header{{}, {"Authorization": []string{"Bearer fixture", ""}}, {"Authorization": []string{"Bearer fixture"}, "X-Agent-Identity-Token": []string{"fixture-run"}}} {
		r := httptest.NewRequest(http.MethodPost, "/start", strings.NewReader(`{}`))
		r.Header = headers
		w := httptest.NewRecorder()
		h.Start(w, r)
		if w.Code != 401 {
			t.Fatalf("invalid ingress%d", w.Code)
		}
	}
}
