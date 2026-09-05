package session

import (
	"context"
	"os"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"
)

type sessionTestClient struct {
	sessionsconnect.SessionsServiceClient
}

func (sessionTestClient) List(context.Context, *connect.Request[sessionsv1.ListRequest]) (*connect.Response[sessionsv1.ListResponse], error) {
	return connect.NewResponse(&sessionsv1.ListResponse{}), nil
}
func (sessionTestClient) Get(context.Context, *connect.Request[sessionsv1.GetRequest]) (*connect.Response[sessionsv1.GetResponse], error) {
	return connect.NewResponse(&sessionsv1.GetResponse{}), nil
}
func (sessionTestClient) Delete(context.Context, *connect.Request[sessionsv1.DeleteRequest]) (*connect.Response[sessionsv1.DeleteResponse], error) {
	return connect.NewResponse(&sessionsv1.DeleteResponse{}), nil
}
func (sessionTestClient) GetArchiveRetention(context.Context, *connect.Request[sessionsv1.GetArchiveRetentionRequest]) (*connect.Response[sessionsv1.GetArchiveRetentionResponse], error) {
	return connect.NewResponse(&sessionsv1.GetArchiveRetentionResponse{}), nil
}
func (sessionTestClient) PruneArchive(context.Context, *connect.Request[sessionsv1.PruneArchiveRequest]) (*connect.Response[sessionsv1.PruneArchiveResponse], error) {
	return connect.NewResponse(&sessionsv1.PruneArchiveResponse{}), nil
}
func (sessionTestClient) GetPolicy(context.Context, *connect.Request[sessionsv1.GetPolicyRequest]) (*connect.Response[sessionsv1.GetPolicyResponse], error) {
	return connect.NewResponse(&sessionsv1.GetPolicyResponse{}), nil
}
func (sessionTestClient) UpdatePolicy(context.Context, *connect.Request[sessionsv1.UpdatePolicyRequest]) (*connect.Response[sessionsv1.UpdatePolicyResponse], error) {
	return connect.NewResponse(&sessionsv1.UpdatePolicyResponse{}), nil
}
func (sessionTestClient) ListRecoverable(context.Context, *connect.Request[sessionsv1.ListRecoverableRequest]) (*connect.Response[sessionsv1.ListRecoverableResponse], error) {
	return connect.NewResponse(&sessionsv1.ListRecoverableResponse{}), nil
}
func (sessionTestClient) Recover(context.Context, *connect.Request[sessionsv1.RecoverRequest]) (*connect.Response[sessionsv1.RecoverResponse], error) {
	return connect.NewResponse(&sessionsv1.RecoverResponse{}), nil
}
func (sessionTestClient) DismissRecoverable(context.Context, *connect.Request[sessionsv1.DismissRecoverableRequest]) (*connect.Response[sessionsv1.DismissRecoverableResponse], error) {
	return connect.NewResponse(&sessionsv1.DismissRecoverableResponse{}), nil
}

func TestHandlersRenderSuccessfulResponses(t *testing.T) {
	body := t.TempDir() + "/policy.json"
	if err := os.WriteFile(body, []byte(`{"mode":"days","duration":"7d"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	h := &handlers{client: sessionTestClient{}}
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "apply"}, {Name: "body-file"}}, Positionals: []cliapp.Positional{{Name: "session-id"}}}
	ctx := func() cliapp.RunContext {
		return cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: schema, Flags: map[string]string{"apply": "false", "body-file": body}, Positionals: map[string]string{"session-id": "s1"}, JSON: true})
	}
	for _, call := range []func(cliapp.RunContext) error{h.list, h.archiveRetention, h.archivePrune, h.get, h.delete, h.policyGet, h.policySet, h.listRecoverable, h.recover, h.dismiss} {
		if err := call(ctx()); err != nil {
			t.Fatal(err)
		}
	}
}
