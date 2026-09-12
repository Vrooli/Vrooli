package testing

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"connectrpc.com/connect"
	testingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/testing"
	testingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/testing/testing_v1connect"

	"prompt-manager/handlers/transportbridge"
	domain "prompt-manager/internal/testing"
)

type connectHandler struct {
	testingconnect.UnimplementedTestingServiceHandler
	legacy *domain.Handlers
}

func NewConnectMount(legacy *domain.Handlers) (string, http.Handler) {
	return testingconnect.NewTestingServiceHandler(&connectHandler{legacy: legacy})
}

func (h *connectHandler) RunSkillTest(ctx context.Context, req *connect.Request[testingv1.RunSkillTestRequest]) (*connect.Response[testingv1.SkillTestResponse], error) {
	body := map[string]any{"role": req.Msg.GetRole(), "variables": req.Msg.GetVariables()}
	if req.Msg.MaxTokens != nil {
		body["maxTokens"] = req.Msg.GetMaxTokens()
	}
	if req.Msg.Temperature != nil {
		body["temperature"] = req.Msg.GetTemperature()
	}
	vars := map[string]string{"id": req.Msg.GetSkillId()}
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.Test, http.MethodPost, "/skills/"+url.PathEscape(req.Msg.GetSkillId())+"/test", body, vars, &testingv1.SkillTestResponse{})
}

func (h *connectHandler) ListSkillTestHistory(ctx context.Context, req *connect.Request[testingv1.ListSkillTestHistoryRequest]) (*connect.Response[testingv1.ListSkillTestHistoryResponse], error) {
	target := "/skills/" + url.PathEscape(req.Msg.GetSkillId()) + "/test-history"
	if req.Msg.GetLimit() > 0 {
		target += "?limit=" + strconv.FormatInt(int64(req.Msg.GetLimit()), 10)
	}
	return transportbridge.InvokeWrappedJSON(ctx, req.Header(), h.legacy.GetHistory, http.MethodGet, target, nil, map[string]string{"id": req.Msg.GetSkillId()}, "results", &testingv1.ListSkillTestHistoryResponse{})
}
