package settings

import (
	"context"
	"os"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	settingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/settings"
	settingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/settings/settings_v1connect"
)

type settingsTestClient struct {
	settingsconnect.SettingsServiceClient
}

func (settingsTestClient) GetSessionDefaults(context.Context, *connect.Request[settingsv1.GetSessionDefaultsRequest]) (*connect.Response[settingsv1.GetSessionDefaultsResponse], error) {
	return connect.NewResponse(&settingsv1.GetSessionDefaultsResponse{Defaults: &settingsv1.SessionDefaults{DefaultBackend: "standard"}}), nil
}
func (settingsTestClient) UpdateSessionDefaults(context.Context, *connect.Request[settingsv1.UpdateSessionDefaultsRequest]) (*connect.Response[settingsv1.UpdateSessionDefaultsResponse], error) {
	return connect.NewResponse(&settingsv1.UpdateSessionDefaultsResponse{}), nil
}

func TestHandlersRenderSuccessfulResponses(t *testing.T) {
	body := t.TempDir() + "/defaults.json"
	if err := os.WriteFile(body, []byte(`{"default_backend":"persistent","default_policy":{"mode":"days","duration":"7d"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	h := &handlers{client: settingsTestClient{}}
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "body-file"}}}
	if err := h.get(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: schema, JSON: true})); err != nil {
		t.Fatal(err)
	}
	if err := h.set(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: schema, Flags: map[string]string{"body-file": body}, JSON: true})); err != nil {
		t.Fatal(err)
	}
}
