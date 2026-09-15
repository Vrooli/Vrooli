package testing

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"
	testingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/testing"
	testingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/testing/testing_v1connect"

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
	testRequest := domain.TestRequest{Role: req.Msg.GetRole(), Variables: req.Msg.GetVariables()}
	if req.Msg.MaxTokens != nil {
		value := int(req.Msg.GetMaxTokens())
		testRequest.MaxTokens = &value
	}
	if req.Msg.Temperature != nil {
		value := req.Msg.GetTemperature()
		testRequest.Temperature = &value
	}
	result, err := h.legacy.RunTest(ctx, req.Msg.GetSkillId(), testRequest)
	if err != nil {
		return nil, testingError(err)
	}
	return connect.NewResponse(&testingv1.SkillTestResponse{TestId: result.TestID, Role: result.Role, Response: result.Response, ResponseTime: result.ResponseTime, TokenCount: int32(result.TokenCount), TestedAt: result.TestedAt.Format(time.RFC3339Nano)}), nil
}

func (h *connectHandler) ListSkillTestHistory(ctx context.Context, req *connect.Request[testingv1.ListSkillTestHistoryRequest]) (*connect.Response[testingv1.ListSkillTestHistoryResponse], error) {
	limit := int(req.Msg.GetLimit())
	results, err := h.legacy.History(ctx, req.Msg.GetSkillId(), limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &testingv1.ListSkillTestHistoryResponse{}
	for _, result := range results {
		item := &testingv1.SkillTestResult{Id: result.ID, SkillId: result.SkillID, Role: result.Role, InputVariables: result.InputVars, Response: result.Response, ResponseTime: result.ResponseTime, TestedAt: result.TestedAt.Format(time.RFC3339Nano)}
		if result.TokenCount != nil {
			value := int32(*result.TokenCount)
			item.TokenCount = &value
		}
		if result.Rating != nil {
			value := int32(*result.Rating)
			item.Rating = &value
		}
		item.Notes = result.Notes
		out.Results = append(out.Results, item)
	}
	return connect.NewResponse(out), nil
}

func testingError(err error) error {
	code := connect.CodeInternal
	switch {
	case errors.Is(err, domain.ErrTestingUnavailable):
		code = connect.CodeUnavailable
	case errors.Is(err, domain.ErrSkillNotFound):
		code = connect.CodeNotFound
	}
	return connect.NewError(code, err)
}
