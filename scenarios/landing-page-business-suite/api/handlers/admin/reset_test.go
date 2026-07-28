package admin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestResetDemoDataReportsResetFailuresWithoutLeakingTheCause(t *testing.T) {
	w := httptest.NewRecorder()
	ResetDemoData(ResetDependencies{Reset: func(context.Context) error { return errors.New("database credentials") }, Now: time.Now, LogError: func(string, map[string]any) {}}).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/admin/reset-demo-data", nil))
	if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "failed to reset demo data") || strings.Contains(w.Body.String(), "credentials") {
		t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
	}
}
