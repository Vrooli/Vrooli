package builds

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"

	buildsdomain "scenario-to-android/internal/builds"
)

func TestNativeProfileReachesBuildOwnerThroughHTTP(t *testing.T) {
	var received deliveryramp.BuildRequest
	handler := GenerateHandler(Surface{Generate: func(_ context.Context, request deliveryramp.BuildRequest) (buildsdomain.GeneratedProject, error) {
		received = request
		return buildsdomain.GeneratedProject{PackageName: "com.example.daily"}, nil
	}})
	response := httptest.NewRecorder()
	handler(response, httptest.NewRequest("POST", "/android/generate", strings.NewReader(`{"source_ref":"/source","project_kind":"native","native_profile":"daily"}`)))
	require.Equal(t, 200, response.Code)
	require.Equal(t, "daily", received.Parameters["native_profile"], "profile selection must survive the HTTP boundary instead of silently building the default preview")
	require.Equal(t, "native", received.Parameters["project_kind"])
}
