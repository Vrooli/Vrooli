package agentmanager

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

const ownedRunID = "9b5a0610-88ec-4e4a-9964-626a8b67bdf5"
const ownedTaskID = "a8c764d5-2222-4c28-9e2a-852f5ba2a880"

type portalAgentManagerFixture struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	profileCalls, taskCalls, createCalls, stopCalls, readCalls, tagCalls int
	createErr, stopErr                                                   error
	stopRun                                                              *domain.Run
	getRun                                                               *domain.Run
	tagRun                                                               *domain.Run
	gotTask                                                              *domain.Task
	gotCreate                                                            *api.CreateRunRequest
	createAuth                                                           string
	taskAuth                                                             string
	createTaskErr                                                        error
}

func (f *portalAgentManagerFixture) ReconcileScenarioProfiles(_ context.Context, req *connect.Request[api.ReconcileScenarioProfilesRequest]) (*connect.Response[api.ReconcileScenarioProfilesResponse], error) {
	f.profileCalls++
	if req.Msg.GetScenario() != "portal" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("wrong scenario"))
	}
	return connect.NewResponse(&api.ReconcileScenarioProfilesResponse{Results: []*api.ProfileReconcileResult{{ProfileKey: "portal/agent-chat", ProfileId: "profile-1"}}}), nil
}

func (f *portalAgentManagerFixture) CreateTask(_ context.Context, req *connect.Request[api.CreateTaskRequest]) (*connect.Response[api.CreateTaskResponse], error) {
	f.taskCalls++
	f.gotTask = req.Msg.GetTask()
	f.taskAuth = req.Header().Get("Authorization")
	if f.createTaskErr != nil {
		return nil, f.createTaskErr
	}
	return connect.NewResponse(&api.CreateTaskResponse{Task: &domain.Task{Id: ownedTaskID}}), nil
}

func (f *portalAgentManagerFixture) CreateRun(_ context.Context, req *connect.Request[api.CreateRunRequest]) (*connect.Response[api.CreateRunResponse], error) {
	f.createCalls++
	f.gotCreate = req.Msg
	f.createAuth = req.Header().Get("Authorization")
	if f.createErr != nil {
		return nil, f.createErr
	}
	return connect.NewResponse(&api.CreateRunResponse{Run: &domain.Run{Id: ownedRunID, TaskId: req.Msg.GetTaskId(), Tag: req.Msg.GetTag()}}), nil
}

func (f *portalAgentManagerFixture) GetRunByTag(_ context.Context, req *connect.Request[api.GetRunByTagRequest]) (*connect.Response[api.GetRunByTagResponse], error) {
	f.tagCalls++
	if req.Msg.GetTag() == "" || f.tagRun == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("missing"))
	}
	return connect.NewResponse(&api.GetRunByTagResponse{Run: f.tagRun}), nil
}

func (f *portalAgentManagerFixture) StopRun(_ context.Context, req *connect.Request[api.StopRunRequest]) (*connect.Response[api.StopRunResponse], error) {
	f.stopCalls++
	if f.stopErr != nil {
		return nil, f.stopErr
	}
	if f.stopRun == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("missing stop run"))
	}
	return connect.NewResponse(&api.StopRunResponse{Run: f.stopRun}), nil
}

func (f *portalAgentManagerFixture) GetRun(_ context.Context, req *connect.Request[api.GetRunRequest]) (*connect.Response[api.GetRunResponse], error) {
	f.readCalls++
	if req.Msg.GetRunId() == "" || f.getRun == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("missing"))
	}
	return connect.NewResponse(&api.GetRunResponse{Run: f.getRun}), nil
}

func servePortalAgentManager(t *testing.T, fixture *portalAgentManagerFixture) *ConnectClient {
	t.Helper()
	path, handler := apiconnect.NewAgentManagerServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return NewConnectClient(server.URL, server.Client())
}

func TestStartUsesGeneratedConnectAndTypedContextAttachments(t *testing.T) {
	fixture := &portalAgentManagerFixture{}
	client := servePortalAgentManager(t, fixture)
	session, err := client.Start(auth01Caller(context.Background(), nil, false), StartInput{
		AdmissionID: ownedRunID, Prompt: " inspect this", ProjectRoot: "/workspace", AttachmentIDs: []string{"image-1"}, ParentRunID: "123e4567-e89b-12d3-a456-426614174002",
	})
	require.NoError(t, err)
	require.Equal(t, Session{TaskID: ownedTaskID, RunID: ownedRunID}, session)
	require.Equal(t, 1, fixture.profileCalls)
	require.Equal(t, 1, fixture.taskCalls)
	require.Equal(t, 1, fixture.createCalls)
	require.Equal(t, "inspect this", fixture.gotTask.GetDescription())
	require.Equal(t, "/workspace", fixture.gotTask.GetProjectRoot())
	require.Len(t, fixture.gotTask.GetContextAttachments(), 1)
	require.Equal(t, "image", fixture.gotTask.GetContextAttachments()[0].GetType())
	require.Equal(t, "image-1", fixture.gotTask.GetContextAttachments()[0].GetAttachmentId())
	require.Empty(t, fixture.taskAuth)
	require.Equal(t, "123e4567-e89b-12d3-a456-426614174002", fixture.gotCreate.GetParentRunId())
	require.Equal(t, "portal/agent-chat", fixture.gotCreate.GetProfileRef().GetProfileKey())
	require.Equal(t, "portal-admission-"+ownedRunID, fixture.gotCreate.GetTag())
	require.Equal(t, "portal-admission-"+ownedRunID, fixture.gotCreate.GetIdempotencyKey())
	require.Equal(t, "Bearer fixture-owner", fixture.createAuth)
}

func TestFindAdmissionUsesExactGeneratedGetRunByTag(t *testing.T) {
	tag := "portal-admission-" + ownedRunID
	fixture := &portalAgentManagerFixture{tagRun: &domain.Run{Id: ownedRunID, TaskId: ownedTaskID, Tag: tag}}
	client := servePortalAgentManager(t, fixture)
	session, err := client.FindAdmission(context.Background(), ownedRunID)
	require.NoError(t, err)
	require.Equal(t, Session{RunID: ownedRunID, TaskID: ownedTaskID}, session)
	require.Equal(t, 1, fixture.tagCalls)

	fixture.tagRun.Tag = tag + "-suffix"
	_, err = client.FindAdmission(context.Background(), ownedRunID)
	require.ErrorContains(t, err, "exact run identity")
}

func TestStartReturnsExistingAdmissionBeforeCreatingAnotherTaskOrRun(t *testing.T) {
	tag := "portal-admission-" + ownedRunID
	fixture := &portalAgentManagerFixture{tagRun: &domain.Run{Id: ownedRunID, TaskId: ownedTaskID, Tag: tag}}
	session, err := servePortalAgentManager(t, fixture).Start(auth01Caller(context.Background(), nil, false), StartInput{AdmissionID: ownedRunID, Prompt: "retry"})
	require.NoError(t, err)
	require.Equal(t, Session{RunID: ownedRunID, TaskID: ownedTaskID}, session)
	require.Equal(t, 1, fixture.tagCalls)
	require.Zero(t, fixture.profileCalls)
	require.Zero(t, fixture.taskCalls)
	require.Zero(t, fixture.createCalls)
}

func TestStopRequiresMatchingTerminalOwnerState(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    domain.RunStatus
		confirmed bool
	}{
		{"cancelled", domain.RunStatus_RUN_STATUS_CANCELLED, true},
		{"completed-before-stop", domain.RunStatus_RUN_STATUS_COMPLETE, true},
		{"still-running", domain.RunStatus_RUN_STATUS_RUNNING, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := &portalAgentManagerFixture{stopRun: &domain.Run{Id: ownedRunID, Status: tc.status}}
			state, err := servePortalAgentManager(t, fixture).Stop(context.Background(), ownedRunID)
			if tc.confirmed {
				require.NoError(t, err)
				require.True(t, state.Terminal)
				require.Equal(t, ownedRunID, state.RunID)
			} else {
				require.ErrorIs(t, err, ErrStopUnconfirmed)
			}
			require.Equal(t, 1, fixture.stopCalls)
		})
	}
}

func TestLostStopReplyReconcilesWithReadWithoutRepeatingStop(t *testing.T) {
	fixture := &portalAgentManagerFixture{
		stopErr: connect.NewError(connect.CodeUnavailable, errors.New("reply lost")),
		getRun:  &domain.Run{Id: ownedRunID, Status: domain.RunStatus_RUN_STATUS_CANCELLED},
	}
	state, err := servePortalAgentManager(t, fixture).Stop(context.Background(), ownedRunID)
	require.NoError(t, err)
	require.True(t, state.Terminal)
	require.Equal(t, 1, fixture.stopCalls)
	require.Equal(t, 1, fixture.readCalls)
}

func TestRunOperationsRefuseNonCanonicalIdentityBeforeNetwork(t *testing.T) {
	client := NewConnectClient("http://127.0.0.1:1", nil)
	for _, id := range []string{"", "../other/stop", "00000000-0000-0000-0000-000000000000"} {
		_, err := client.Run(context.Background(), id)
		require.ErrorContains(t, err, "canonical agent run UUID required")
	}
}

func TestConnectDeadlinePreservesUnavailableClassification(t *testing.T) {
	err := mapAgentManagerError(context.Background(), connect.NewError(connect.CodeDeadlineExceeded, context.DeadlineExceeded))
	require.ErrorIs(t, err, ErrUnavailable)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestUploadAttachmentKeepsMultipartRESTOnlyForBinaryUpload(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/attachments/upload", r.URL.Path)
		require.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")
		require.NoError(t, r.ParseMultipartForm(1024))
		file, header, err := r.FormFile("file")
		require.NoError(t, err)
		defer file.Close()
		require.Equal(t, "portal-context.png", header.Filename)
		body, err := io.ReadAll(file)
		require.NoError(t, err)
		require.Equal(t, []byte("rendered"), body)
		_, _ = io.WriteString(w, `{"id":"attachment-1"}`)
	}))
	defer server.Close()
	id, err := NewConnectClient(server.URL, server.Client()).UploadAttachment(context.Background(), []byte("rendered"), "portal-context.png", "image/png")
	require.NoError(t, err)
	require.Equal(t, "attachment-1", id)
	require.Equal(t, 1, calls)
}
