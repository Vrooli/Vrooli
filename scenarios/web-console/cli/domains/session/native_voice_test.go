package session

import (
	"context"
	"os"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
)

type nativeLaunchClient struct {
	sessionTestClient
	request *sessionsv1.CreateRequest
}

func (c *nativeLaunchClient) Create(_ context.Context, request *connect.Request[sessionsv1.CreateRequest]) (*connect.Response[sessionsv1.CreateResponse], error) {
	c.request = request.Msg
	return connect.NewResponse(&sessionsv1.CreateResponse{Session: &sessionsv1.Session{Id: "native-session"}}), nil
}
func TestCreatePreservesNativeLaunchDescriptor(t *testing.T) {
	path := t.TempDir() + "/create.json"
	if err := os.WriteFile(path, []byte(`{"agentType":"codex","launchMode":"codex_app_server","launchDescriptorJson":"{\"agent\":\"codex\",\"launchMode\":\"codex_app_server\"}"}`), 0600); err != nil {
		t.Fatal(err)
	}
	client := &nativeLaunchClient{}
	h := &handlers{client: client}
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "cols"}, {Name: "rows"}, {Name: "shell"}, {Name: "backend"}, {Name: "launch-command"}, {Name: "origin"}, {Name: "owner"}, {Name: "label"}, {Name: "target"}, {Name: "working-dir"}, {Name: "execute-launch-command"}, {Name: "body-file"}, {Name: "idempotency-key"}}}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: schema, Flags: map[string]string{"body-file": path}, JSON: true})
	if err := h.create(ctx); err != nil {
		t.Fatal(err)
	}
	if client.request == nil || client.request.LaunchMode != "codex_app_server" || client.request.AgentType != "codex" || client.request.LaunchDescriptorJson == "" {
		t.Fatalf("native descriptor lost: %+v", client.request)
	}
}
