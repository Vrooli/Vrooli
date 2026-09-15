package content

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCustomizeRejectsMalformedRequest(t *testing.T) {
	w := httptest.NewRecorder()
	Customize(func() time.Time { return time.Unix(0, 0) }).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/customize", strings.NewReader("{")))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
}
