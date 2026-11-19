package agentmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// fakeAgentManagerConnect adapts the existing route fixture to generated RPCs.
// It keeps scenario tests focused on client semantics while asserting their
// requests still match the owner's protobuf JSON contract.
type fakeAgentManagerConnect struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	fake *fakeAgentManager
}

func (f fakeAgentManagerConnect) EnsureProfile(_ context.Context, req *connect.Request[apipb.EnsureProfileRequest]) (*connect.Response[apipb.EnsureProfileResponse], error) {
	resp := &apipb.EnsureProfileResponse{}
	if err := f.call(http.MethodPost, "/api/v1/profiles/ensure", "", "", req.Msg, resp); err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f fakeAgentManagerConnect) ReconcileScenarioProfiles(_ context.Context, req *connect.Request[apipb.ReconcileScenarioProfilesRequest]) (*connect.Response[apipb.ReconcileScenarioProfilesResponse], error) {
	resp := &apipb.ReconcileScenarioProfilesResponse{}
	if err := f.call(http.MethodPost, "/api/v1/profiles/reconcile-scenario", "", "", req.Msg, resp); err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f fakeAgentManagerConnect) CreateTask(_ context.Context, req *connect.Request[apipb.CreateTaskRequest]) (*connect.Response[apipb.CreateTaskResponse], error) {
	resp := &apipb.CreateTaskResponse{}
	if err := f.call(http.MethodPost, "/api/v1/tasks", "", req.Header().Get("Authorization"), req.Msg, resp); err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f fakeAgentManagerConnect) GetTask(_ context.Context, req *connect.Request[apipb.GetTaskRequest]) (*connect.Response[apipb.GetTaskResponse], error) {
	resp := &apipb.GetTaskResponse{}
	if err := f.call(http.MethodGet, "/api/v1/tasks/"+url.PathEscape(req.Msg.GetTaskId()), "", req.Header().Get("Authorization"), req.Msg, resp); err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f fakeAgentManagerConnect) CreateRun(_ context.Context, req *connect.Request[apipb.CreateRunRequest]) (*connect.Response[apipb.CreateRunResponse], error) {
	resp := &apipb.CreateRunResponse{}
	if err := f.call(http.MethodPost, "/api/v1/runs", "", req.Header().Get("Authorization"), req.Msg, resp); err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f fakeAgentManagerConnect) GetRun(_ context.Context, req *connect.Request[apipb.GetRunRequest]) (*connect.Response[apipb.GetRunResponse], error) {
	resp := &apipb.GetRunResponse{}
	if err := f.call(http.MethodGet, "/api/v1/runs/"+url.PathEscape(req.Msg.GetRunId()), "", req.Header().Get("Authorization"), req.Msg, resp); err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f fakeAgentManagerConnect) GetRunByTag(_ context.Context, req *connect.Request[apipb.GetRunByTagRequest]) (*connect.Response[apipb.GetRunByTagResponse], error) {
	resp := &apipb.GetRunByTagResponse{}
	if err := f.call(http.MethodGet, "/api/v1/runs/tag/"+url.PathEscape(req.Msg.GetTag()), "", req.Header().Get("Authorization"), req.Msg, resp); err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f fakeAgentManagerConnect) StopRun(_ context.Context, req *connect.Request[apipb.StopRunRequest]) (*connect.Response[apipb.StopRunResponse], error) {
	resp := &apipb.StopRunResponse{}
	if err := f.call(http.MethodPost, "/api/v1/runs/"+url.PathEscape(req.Msg.GetRunId())+"/stop", "", req.Header().Get("Authorization"), req.Msg, resp); err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f fakeAgentManagerConnect) StopRunByTag(_ context.Context, req *connect.Request[apipb.StopRunByTagRequest]) (*connect.Response[apipb.StopRunByTagResponse], error) {
	resp := &apipb.StopRunByTagResponse{}
	if err := f.call(http.MethodPost, "/api/v1/runs/tag/"+url.PathEscape(req.Msg.GetTag())+"/stop", "", req.Header().Get("Authorization"), req.Msg, resp); err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f fakeAgentManagerConnect) StopAllRuns(_ context.Context, req *connect.Request[apipb.StopAllRunsRequest]) (*connect.Response[apipb.StopAllRunsResponse], error) {
	resp := &apipb.StopAllRunsResponse{}
	if err := f.call(http.MethodPost, "/api/v1/runs/stop-all", "", req.Header().Get("Authorization"), req.Msg, resp); err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f fakeAgentManagerConnect) ListRuns(_ context.Context, req *connect.Request[apipb.ListRunsRequest]) (*connect.Response[apipb.ListRunsResponse], error) {
	query := url.Values{}
	if req.Msg.Status != nil {
		query.Set("status", strconv.Itoa(int(*req.Msg.Status)))
	}
	if req.Msg.TagPrefix != nil {
		query.Set("tag_prefix", req.Msg.GetTagPrefix())
	}
	resp := &apipb.ListRunsResponse{}
	if err := f.call(http.MethodGet, "/api/v1/runs", query.Encode(), req.Header().Get("Authorization"), req.Msg, resp); err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f fakeAgentManagerConnect) GetRunEvents(_ context.Context, req *connect.Request[apipb.GetRunEventsRequest]) (*connect.Response[apipb.GetRunEventsResponse], error) {
	query := url.Values{}
	if req.Msg.AfterSequence != nil {
		query.Set("after_sequence", strconv.FormatInt(req.Msg.GetAfterSequence(), 10))
	}
	resp := &apipb.GetRunEventsResponse{}
	path := "/api/v1/runs/" + url.PathEscape(req.Msg.GetRunId()) + "/events"
	if err := f.call(http.MethodGet, path, query.Encode(), req.Header().Get("Authorization"), req.Msg, resp); err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f fakeAgentManagerConnect) GetRunnerStatus(_ context.Context, req *connect.Request[apipb.GetRunnerStatusRequest]) (*connect.Response[apipb.GetRunnerStatusResponse], error) {
	resp := &apipb.GetRunnerStatusResponse{}
	if err := f.call(http.MethodGet, "/api/v1/runners", "", req.Header().Get("Authorization"), req.Msg, resp); err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (f fakeAgentManagerConnect) call(method, path, query, authorization string, request, response proto.Message) error {
	body, err := protojson.Marshal(request)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	req := fakeRequest{Method: method, Path: path, Query: query, Auth: authorization, Body: body}
	f.fake.mu.Lock()
	f.fake.requests = append(f.fake.requests, req)
	handler := f.fake.routes[method+" "+path]
	f.fake.mu.Unlock()
	if handler == nil {
		return connect.NewError(connect.CodeNotFound, errors.New("no route"))
	}
	writer := &fakeResponseWriter{header: http.Header{}}
	handler(writer, req)
	if writer.status == 0 {
		writer.status = http.StatusOK
	}
	if writer.status >= http.StatusBadRequest {
		var envelope struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(writer.body, &envelope)
		message := envelope.Error
		if message == "" {
			message = envelope.Message
		}
		if message == "" {
			message = fmt.Sprintf("status %d %s", writer.status, http.StatusText(writer.status))
			if strings.TrimSpace(string(writer.body)) != "" {
				message += ": " + strings.TrimSpace(string(writer.body))
			}
		}
		return connect.NewError(fakeConnectCode(writer.status), errors.New(message))
	}
	if err := protojson.Unmarshal(writer.body, response); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("unmarshal response: %w", err))
	}
	return nil
}

type fakeResponseWriter struct {
	header http.Header
	status int
	body   []byte
}

func (w *fakeResponseWriter) Header() http.Header    { return w.header }
func (w *fakeResponseWriter) WriteHeader(status int) { w.status = status }
func (w *fakeResponseWriter) Write(payload []byte) (int, error) {
	w.body = append(w.body, payload...)
	return len(payload), nil
}

func fakeConnectCode(status int) connect.Code {
	switch status {
	case http.StatusBadRequest:
		return connect.CodeInvalidArgument
	case http.StatusUnauthorized:
		return connect.CodeUnauthenticated
	case http.StatusForbidden:
		return connect.CodePermissionDenied
	case http.StatusNotFound:
		return connect.CodeNotFound
	case http.StatusConflict:
		return connect.CodeFailedPrecondition
	case http.StatusTooManyRequests:
		return connect.CodeResourceExhausted
	case http.StatusServiceUnavailable:
		return connect.CodeUnavailable
	default:
		return connect.CodeInternal
	}
}
