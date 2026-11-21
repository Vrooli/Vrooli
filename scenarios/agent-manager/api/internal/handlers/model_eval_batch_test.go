package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/model_eval"
	"agent-manager/internal/orchestration"
	"github.com/gorilla/mux"
)

type modelEvalApprovalAnswer struct{ answer string }

func (s modelEvalApprovalAnswer) ReadDecision(context.Context, string) (model_eval.Approval, error) {
	return model_eval.Approval{DecisionID: model_eval.AdmissionDecisionID, Answer: s.answer, Scope: model_eval.AdmissionScope, RecordedAt: time.Now().UTC()}, nil
}

type modelEvalNoReadSource struct{ selected, loaded int }

func (s *modelEvalNoReadSource) Select(context.Context, int) ([]model_eval.SourceRef, error) {
	s.selected++
	return nil, nil
}
func (s *modelEvalNoReadSource) VerifyAndLoad(context.Context, model_eval.SourceRef) (model_eval.VerifiedCase, error) {
	s.loaded++
	return model_eval.VerifiedCase{}, nil
}

func TestModelEvalBatchEndpointRefusesDeferWithoutReadingOrLeakingPrompt(t *testing.T) {
	source := &modelEvalNoReadSource{}
	h := New(orchestration.EmptyHandlerServices(), WithModelEvalRunner(model_eval.Runner{Approval: modelEvalApprovalAnswer{answer: "defer"}, Selector: source, Source: source}))
	router := muxRouter()
	h.RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/model-eval/batch", strings.NewReader(`{"decision_id":"`+model_eval.AdmissionDecisionID+`","max_cases":20}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusPreconditionFailed {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if source.selected != 0 || source.loaded != 0 {
		t.Fatalf("deferred operation selected/loaded source: %+v", source)
	}
	for _, forbidden := range []string{"prompt", "output", "LM2_PROMPT_SENTINEL", "cases"} {
		if strings.Contains(response.Body.String(), forbidden) {
			t.Fatalf("refusal leaked %q: %s", forbidden, response.Body.String())
		}
	}
}

func TestModelEvalBatchEndpointRejectsPromptBodyFields(t *testing.T) {
	source := &modelEvalNoReadSource{}
	h := New(orchestration.EmptyHandlerServices(), WithModelEvalRunner(model_eval.Runner{Approval: modelEvalApprovalAnswer{answer: "approve"}, Selector: source, Source: source}))
	router := muxRouter()
	h.RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/model-eval/batch", strings.NewReader(`{"decision_id":"`+model_eval.AdmissionDecisionID+`","prompt":"LM2_PROMPT_SENTINEL"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if source.selected != 0 || source.loaded != 0 {
		t.Fatalf("invalid request selected/loaded source: %+v", source)
	}
	if strings.Contains(response.Body.String(), "LM2_PROMPT_SENTINEL") {
		t.Fatalf("invalid request response leaked prompt: %s", response.Body.String())
	}
}

func muxRouter() *mux.Router { return mux.NewRouter() }
