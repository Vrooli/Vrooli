package deployments

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeployDesktopRequiresCommitIdentifier(t *testing.T) {
	o := NewOrchestrator(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deploy-desktop", strings.NewReader(`{"profile_id":"profile-test"}`))
	rec := httptest.NewRecorder()

	o.DeployDesktop(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
