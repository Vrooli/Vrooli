package retrieval

import (
	"context"
	"io"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	retrievalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/retrieval"
	retrievalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/retrieval/retrieval_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
	"google.golang.org/protobuf/proto"
)

type retrievalAPI struct {
	retrievalconnect.UnimplementedRetrievalServiceHandler
	queryRequest    *retrievalv1.QueryRequest
	similarRequest  *retrievalv1.SimilarRequest
	queryResponse   *retrievalv1.QueryResponse
	similarResponse *retrievalv1.SimilarResponse
	queryErr        error
}

func (s *retrievalAPI) Query(_ context.Context, req *connect.Request[retrievalv1.QueryRequest]) (*connect.Response[retrievalv1.QueryResponse], error) {
	s.queryRequest = req.Msg
	if s.queryErr != nil {
		return nil, s.queryErr
	}
	return connect.NewResponse(s.queryResponse), nil
}

func (s *retrievalAPI) Similar(_ context.Context, req *connect.Request[retrievalv1.SimilarRequest]) (*connect.Response[retrievalv1.SimilarResponse], error) {
	s.similarRequest = req.Msg
	return connect.NewResponse(s.similarResponse), nil
}

func retrievalServer(t *testing.T, service *retrievalAPI) http.Handler {
	t.Helper()
	path, handler := retrievalconnect.NewRetrievalServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func retrievalCommand(t *testing.T, name string, core *cliapp.ScenarioApp) cliapp.Command {
	t.Helper()
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)
	require.Equal(t, GroupName, group.Name)
	return cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, name)
}

func TestRegister_ExposesQueryAndSimilar(t *testing.T) {
	core := cliapptest.NewTestApp(t, http.NotFoundHandler())
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)
	require.Equal(t, GroupName, group.Name)

	query := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "query")
	similar := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "similar")
	require.NotNil(t, query.RunCtx)
	require.NotNil(t, similar.RunCtx)
	_, err = cliapptest.NewTestRunContextFromArgs(query.Args, nil, core, nil, nil)
	require.ErrorContains(t, err, "missing required flag --text")
	_, err = cliapptest.NewTestRunContextFromArgs(similar.Args, nil, core, nil, nil)
	require.ErrorContains(t, err, "missing required flag --document-hash")
}

func TestQuery_SendsFiltersAndRendersPartialResults(t *testing.T) {
	service := &retrievalAPI{queryResponse: &retrievalv1.QueryResponse{
		Partial: true,
		Results: []*retrievalv1.QueryResult{{DocumentHash: "sha-1", Score: 0.87654, AnchorUri: "file:///notes/a"}},
	}}
	core := cliapptest.NewTestApp(t, retrievalServer(t, service))
	cmd := retrievalCommand(t, "query", core)
	stdout, err := cliapptest.RunCommand(t, cmd, core,
		"--text", "launch checklist", "--collection", "ops", "--privacy", "CONFIDENTIAL", "--federated", "true", "--limit", " 7 ")
	require.NoError(t, err)
	require.True(t, proto.Equal(&retrievalv1.QueryRequest{
		Text: "launch checklist", CollectionId: "ops", CallerMaxPrivacy: sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL, Federated: true, Limit: 7,
	}, service.queryRequest), "query request sent to Connect service")
	require.Contains(t, stdout, "partial index: 1 result(s)")
	require.Contains(t, stdout, "sha-1 0.8765 file:///notes/a")
}

func TestSimilar_UsesDefaultsAndRendersCompleteResults(t *testing.T) {
	service := &retrievalAPI{similarResponse: &retrievalv1.SimilarResponse{
		Results: []*retrievalv1.QueryResult{{DocumentHash: "sha-2", Score: 0.5, AnchorUri: "doc://b"}},
	}}
	core := cliapptest.NewTestApp(t, retrievalServer(t, service))
	cmd := retrievalCommand(t, "similar", core)
	stdout, err := cliapptest.RunCommand(t, cmd, core, "--document-hash", "sha-1")
	require.NoError(t, err)
	require.True(t, proto.Equal(&retrievalv1.SimilarRequest{
		DocumentHash: "sha-1", CallerMaxPrivacy: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL, Limit: 20,
	}, service.similarRequest), "similar request sent to Connect service")
	require.Contains(t, stdout, "complete index: 1 result(s)")
	require.Contains(t, stdout, "sha-2 0.5000 doc://b")
}

func TestRetrievalCommands_RejectInvalidLimitsBeforeRequest(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "query", args: []string{"--text", "x", "--limit", "-1"}},
		{name: "similar", args: []string{"--document-hash", "sha-1", "--limit", "2147483648"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &retrievalAPI{}
			core := cliapptest.NewTestApp(t, retrievalServer(t, service))
			stdout, err := cliapptest.RunCommand(t, retrievalCommand(t, tc.name, core), core, tc.args...)
			require.Empty(t, stdout)
			require.ErrorContains(t, err, "invalid limit")
			require.Nil(t, service.queryRequest)
			require.Nil(t, service.similarRequest)
		})
	}
}

func TestQuery_PropagatesConnectError(t *testing.T) {
	service := &retrievalAPI{queryErr: connect.NewError(connect.CodeUnavailable, io.ErrUnexpectedEOF)}
	core := cliapptest.NewTestApp(t, retrievalServer(t, service))
	_, err := cliapptest.RunCommand(t, retrievalCommand(t, "query", core), core, "--text", "x")
	require.ErrorContains(t, err, "unavailable")
	require.ErrorContains(t, err, "unexpected EOF")
	require.NotNil(t, service.queryRequest)
}
