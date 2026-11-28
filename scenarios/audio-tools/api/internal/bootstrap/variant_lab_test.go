//go:build audio_lab

package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"audio-tools/internal/server"
	"audio-tools/internal/testutil/mocks"

	experimentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment/experiment_v1connect"
)

func TestLabCompositionRequiresSupervisorSelectedRuntime(t *testing.T) {
	requestStatus := func(runtime *labRuntimeDeps) int {
		t.Helper()
		srv := composeAPI(server.CompositionDeps{Logger: &mocks.FakeLogger{}}, runtime)
		req := httptest.NewRequest(http.MethodPost, experimentconnect.ExperimentServiceListQualificationEvidenceProcedure, strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		srv.Handler().ServeHTTP(response, req)
		return response.Code
	}

	if got := requestStatus(nil); got != http.StatusNotFound {
		t.Fatalf("lab evidence route without selected runtime returned HTTP %d, want %d", got, http.StatusNotFound)
	}
	if got := requestStatus(&labRuntimeDeps{}); got == http.StatusNotFound {
		t.Fatal("lab evidence route was not mounted when the supervisor selected the lab runtime")
	}
}
