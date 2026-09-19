package shortcuts

import (
	"context"
	"os"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	shortcutsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shortcuts"
	shortcutsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shortcuts/shortcuts_v1connect"
)

type shortcutTestClient struct {
	shortcutsconnect.ShortcutsServiceClient
}

func (shortcutTestClient) GetEffective(context.Context, *connect.Request[shortcutsv1.GetEffectiveRequest]) (*connect.Response[shortcutsv1.GetEffectiveResponse], error) {
	return connect.NewResponse(&shortcutsv1.GetEffectiveResponse{Shortcuts: []*shortcutsv1.Shortcut{{Label: "build", Command: "make"}}}), nil
}

func (shortcutTestClient) ListProfiles(context.Context, *connect.Request[shortcutsv1.ListProfilesRequest]) (*connect.Response[shortcutsv1.ListProfilesResponse], error) {
	return connect.NewResponse(&shortcutsv1.ListProfilesResponse{Profiles: []*shortcutsv1.Profile{{Id: "profile-1", Name: "default", Scope: "global"}}}), nil
}

func (shortcutTestClient) UpsertProfile(context.Context, *connect.Request[shortcutsv1.UpsertProfileRequest]) (*connect.Response[shortcutsv1.UpsertProfileResponse], error) {
	return connect.NewResponse(&shortcutsv1.UpsertProfileResponse{Profile: &shortcutsv1.Profile{Id: "profile-1", Name: "default", Scope: "global"}}), nil
}

func (shortcutTestClient) DeleteProfile(context.Context, *connect.Request[shortcutsv1.DeleteProfileRequest]) (*connect.Response[shortcutsv1.DeleteProfileResponse], error) {
	return connect.NewResponse(&shortcutsv1.DeleteProfileResponse{}), nil
}

func TestHandlersRenderSuccessfulResponses(t *testing.T) {
	body := t.TempDir() + "/profile.json"
	if err := os.WriteFile(body, []byte(`{"id":"profile-1","scope":"global","name":"default","shortcuts":[{"label":"build","command":"make"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "body-file"}}, Positionals: []cliapp.Positional{{Name: "profile-id"}}}
	h := &handlers{client: shortcutTestClient{}}
	for _, call := range []func(cliapp.RunContext) error{h.effective, h.list, h.upsert, h.delete} {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: schema, Flags: map[string]string{"body-file": body}, Positionals: map[string]string{"profile-id": "profile-1"}, JSON: true})
		if err := call(ctx); err != nil {
			t.Fatal(err)
		}
	}
}
