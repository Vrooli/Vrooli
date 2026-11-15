package readiness

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	readinesssvc "github.com/vrooli/browser-automation-studio/services/readiness"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/readiness"
	readinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/readiness/readinessconnect"
)

type fakeProfileReader struct {
	profile readinesssvc.Profile
	err     error
}

func (f fakeProfileReader) GetReadinessProfile(context.Context, string, string) (readinesssvc.Profile, error) {
	return f.profile, f.err
}

func newReadinessTestClient(t *testing.T, reader readinesssvc.ProfileReader) readinessconnect.ReadinessServiceClient {
	t.Helper()
	mount := Module(Deps{Reader: reader, Logger: logrus.New()})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return readinessconnect.NewReadinessServiceClient(server.Client(), server.URL)
}

func TestGetReadinessProfilePreservesCompiledProfile(t *testing.T) {
	reader := fakeProfileReader{profile: readinesssvc.Profile{
		Scenario: "image-tools", ProfileJSON: `{"pages":[{"routes":["/editor"]}]}`, ProfileVersion: "profile-v4",
	}}
	client := newReadinessTestClient(t, reader)
	resp, err := client.GetReadinessProfile(context.Background(), connect.NewRequest(&readinessv1.GetReadinessProfileRequest{Scenario: "image-tools"}))
	require.NoError(t, err)
	require.Equal(t, "image-tools", resp.Msg.GetScenario())
	require.Equal(t, `{"pages":[{"routes":["/editor"]}]}`, resp.Msg.GetProfileJson())
	require.Equal(t, "profile-v4", resp.Msg.GetProfileVersion())
}

func TestGetReadinessProfileRejectsMissingScenarioAndSurfacesReaderOutage(t *testing.T) {
	t.Run("scenario required", func(t *testing.T) {
		client := newReadinessTestClient(t, fakeProfileReader{})
		_, err := client.GetReadinessProfile(context.Background(), connect.NewRequest(&readinessv1.GetReadinessProfileRequest{}))
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
	t.Run("reader outage", func(t *testing.T) {
		client := newReadinessTestClient(t, fakeProfileReader{err: errors.New("experience-manager unavailable")})
		_, err := client.GetReadinessProfile(context.Background(), connect.NewRequest(&readinessv1.GetReadinessProfileRequest{Scenario: "demo"}))
		require.Error(t, err)
		require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	})
}
