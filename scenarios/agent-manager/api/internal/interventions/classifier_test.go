package interventions

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
)

type decodedState struct {
	PrecedingAgentText string `json:"preceding_agent_text"`
	OperatorMessage    string `json:"operator_message"`
}

// fakeGateway answers DecideBatch like ai-gateway: one result per item, in
// order. answer decides each item from its decoded state.
type fakeGateway struct {
	inferenceconnect.UnimplementedInferenceServiceHandler
	answer   func(decodedState) *inferencev1.DecideResponse
	err      error
	mu       sync.Mutex
	requests []*inferencev1.DecideBatchRequest
	inFlight atomic.Int32
	peak     atomic.Int32
}

func (f *fakeGateway) DecideBatch(_ context.Context, req *connect.Request[inferencev1.DecideBatchRequest]) (*connect.Response[inferencev1.DecideBatchResponse], error) {
	now := f.inFlight.Add(1)
	defer f.inFlight.Add(-1)
	for peak := f.peak.Load(); now > peak && !f.peak.CompareAndSwap(peak, now); peak = f.peak.Load() {
	}
	time.Sleep(5 * time.Millisecond)
	f.mu.Lock()
	f.requests = append(f.requests, req.Msg)
	f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	response := &inferencev1.DecideBatchResponse{}
	for _, item := range req.Msg.GetItems() {
		var state decodedState
		_ = json.Unmarshal([]byte(item.GetStateJson()), &state)
		response.Results = append(response.Results, f.answer(state))
	}
	return connect.NewResponse(response), nil
}

// decisionModel answers like Laya: a full distribution over the categories.
func decisionModel(category Category, p float64) *inferencev1.DecideResponse {
	answer := &inferencev1.DecisionAnswer{QuestionId: questionID, Type: inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE, Choice: string(category)}
	rest := (1 - p) / float64(len(taxonomy)-1)
	for _, c := range Categories() {
		probability := rest
		if c == category {
			probability = p
		}
		answer.Probabilities = append(answer.Probabilities, &inferencev1.DecisionProbability{Option: string(c), Probability: probability})
	}
	return &inferencev1.DecideResponse{
		Answers: []*inferencev1.DecisionAnswer{answer}, Method: inferencev1.DecisionMethod_DECISION_METHOD_DECISION_MODEL,
		Provider: "ollaya", Model: "laya:en", ResourceRole: "decide.local",
	}
}

// llmFallback answers like classify.fast: a choice without a distribution,
// optionally after a decision model could not fit the state.
func llmFallback(category Category, overflowed bool) *inferencev1.DecideResponse {
	response := &inferencev1.DecideResponse{
		Answers: []*inferencev1.DecisionAnswer{{QuestionId: questionID, Type: inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE, Choice: string(category)}},
		Method:  inferencev1.DecisionMethod_DECISION_METHOD_LLM_ENUM, Provider: "ollama", Model: "qwen3.5:4b", ResourceRole: "classify.fast",
	}
	if overflowed {
		response.Candidates = []*inferencev1.DecisionCandidate{{Provider: "ollaya", ResourceRole: "decide.local", Status: inferencev1.DecisionCandidateStatus_DECISION_CANDIDATE_STATUS_FAILED, Reason: "context_overflow"}}
	}
	return response
}

func messages(n int, text string) []Message {
	out := make([]Message, n)
	for i := range out {
		out[i] = Message{EventID: strconv.Itoa(i), OperatorText: text, PrecedingAgentText: "I fixed the wrapping."}
	}
	return out
}

func TestClassifyBatchesAtMost32WithBoundedConcurrencyInOrder(t *testing.T) {
	gateway := &fakeGateway{answer: func(s decodedState) *inferencev1.DecideResponse {
		if strings.Contains(s.OperatorMessage, "still") {
			return decisionModel(QualityMiss, 0.7)
		}
		return decisionModel(ApprovalOrInfo, 0.9)
	}}
	input := messages(70, "approve")
	input[33].OperatorText, input[69].OperatorText = "it is still broken", "still broken"
	answers, err := (&Classifier{Client: gateway}).Classify(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	var sizes []int
	for _, request := range gateway.requests {
		sizes = append(sizes, len(request.GetItems()))
		if request.GetRole() != DefaultRole || len(request.GetQuestions()) != 1 || len(request.GetQuestions()[0].GetOptions()) != len(taxonomy) {
			t.Fatalf("request role/questions = %q/%v", request.GetRole(), request.GetQuestions())
		}
	}
	if len(sizes) != 3 || sizes[0]+sizes[1]+sizes[2] != 70 || max(sizes[0], sizes[1], sizes[2]) > MaxBatch {
		t.Fatalf("batch sizes = %v, want three batches of at most %d", sizes, MaxBatch)
	}
	if peak := gateway.peak.Load(); peak > DefaultConcurrency {
		t.Fatalf("peak concurrent calls = %d, want <= %d", peak, DefaultConcurrency)
	}
	for i, answer := range answers {
		want := ApprovalOrInfo
		if i == 33 || i == 69 {
			want = QualityMiss
		}
		if answer.Status != StatusClassified || answer.Category != want || answer.Model != "laya:en" || answer.Method != "DECISION_MODEL" {
			t.Fatalf("answer %d = %+v, want %s from laya", i, answer, want)
		}
	}
	if p := answers[33].Probability(); p == nil || *p != 0.7 {
		t.Fatalf("probability = %v, want 0.7", p)
	}
}

func TestStateIsBoundedOperatorHeadAgentTail(t *testing.T) {
	var got decodedState
	gateway := &fakeGateway{answer: func(s decodedState) *inferencev1.DecideResponse { got = s; return decisionModel(ScopeChange, 0.5) }}
	operator := "HEAD " + strings.Repeat("o", 3000)
	agent := strings.Repeat("a", 3000) + " TAIL"
	if _, err := (&Classifier{Client: gateway}).Classify(context.Background(), []Message{{EventID: "x", OperatorText: operator, PrecedingAgentText: agent}}); err != nil {
		t.Fatal(err)
	}
	if len([]rune(got.OperatorMessage)) != operatorChars || !strings.HasPrefix(got.OperatorMessage, "HEAD") {
		t.Fatalf("operator state = %d chars, prefix %q", len(got.OperatorMessage), got.OperatorMessage[:4])
	}
	if len([]rune(got.PrecedingAgentText)) != agentChars || !strings.HasSuffix(got.PrecedingAgentText, "TAIL") {
		t.Fatalf("agent state = %d chars", len(got.PrecedingAgentText))
	}
}

// A state only the fallback answered because the decision model could not fit
// it is asked again shrunk, so a calibrated answer replaces the bare choice.
func TestOverflowedStateIsRetriedShrunk(t *testing.T) {
	gateway := &fakeGateway{answer: func(s decodedState) *inferencev1.DecideResponse {
		if len(s.OperatorMessage) > int(operatorChars*shrinkScale) {
			return llmFallback(TooDense, true)
		}
		return decisionModel(QualityMiss, 0.8)
	}}
	answers, err := (&Classifier{Client: gateway}).Classify(context.Background(), []Message{{EventID: "x", OperatorText: strings.Repeat("broken ", 400)}})
	if err != nil {
		t.Fatal(err)
	}
	if len(gateway.requests) != 2 || answers[0].Category != QualityMiss || answers[0].Method != "DECISION_MODEL" || answers[0].Probability() == nil {
		t.Fatalf("requests = %d, answer = %+v; want a shrunk retry answered by the decision model", len(gateway.requests), answers[0])
	}
}

func TestFallbackAnswerHasNoDistribution(t *testing.T) {
	gateway := &fakeGateway{answer: func(decodedState) *inferencev1.DecideResponse { return llmFallback(EarlyStop, false) }}
	answers, err := (&Classifier{Client: gateway}).Classify(context.Background(), messages(1, "keep going"))
	if err != nil {
		t.Fatal(err)
	}
	if len(gateway.requests) != 1 || answers[0].Category != EarlyStop || answers[0].Method != "LLM_ENUM" || answers[0].Probability() != nil || len(answers[0].Probabilities) != 0 {
		t.Fatalf("answer = %+v, want a bare LLM choice and no retry", answers[0])
	}
}

// Unavailable is unclassified, with the reason, never a category.
func TestGatewayFailuresAreUnclassified(t *testing.T) {
	down := &fakeGateway{err: connect.NewError(connect.CodeUnavailable, errors.New("connection refused"))}
	answers, err := (&Classifier{Client: down}).Classify(context.Background(), messages(40, "keep going"))
	if !errors.Is(err, ErrGatewayUnavailable) {
		t.Fatalf("err = %v, want ErrGatewayUnavailable", err)
	}
	for _, answer := range answers {
		if answer.Status != StatusUnclassified || answer.Category != "" || !strings.Contains(answer.Reason, "connection refused") {
			t.Fatalf("answer = %+v, want unclassified with the reason", answer)
		}
	}
	noModel := &fakeGateway{answer: func(decodedState) *inferencev1.DecideResponse {
		return &inferencev1.DecideResponse{Error: &inferencev1.InferenceError{Code: inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE, Message: "no decision candidate answered"}}
	}}
	answers, err = (&Classifier{Client: noModel}).Classify(context.Background(), messages(2, "keep going"))
	if err != nil || answers[0].Status != StatusUnclassified || !strings.HasPrefix(answers[0].Reason, "unavailable: no decision candidate") {
		t.Fatalf("answer = %+v err = %v, want per-item unclassified", answers[0], err)
	}
	unresolved := &Classifier{ResolveURL: func(context.Context) (string, error) { return "", errors.New("ai-gateway is not running") }}
	answers, err = unresolved.Classify(context.Background(), messages(1, "x"))
	if !errors.Is(err, ErrGatewayUnavailable) || answers[0].Status != StatusUnclassified {
		t.Fatalf("unresolved gateway: answer = %+v err = %v", answers[0], err)
	}
}

// The production client path: service discovery URL, generated Connect
// client, a loopback fake ai-gateway.
func TestClassifierCallsGatewayOverConnect(t *testing.T) {
	gateway := &fakeGateway{answer: func(decodedState) *inferencev1.DecideResponse { return decisionModel(FalseInability, 0.6) }}
	mux := http.NewServeMux()
	mux.Handle(inferenceconnect.NewInferenceServiceHandler(gateway))
	server := httptest.NewServer(mux)
	defer server.Close()
	classifier := NewClassifier()
	classifier.ResolveURL = func(context.Context) (string, error) { return server.URL, nil }
	answers, err := classifier.Classify(context.Background(), messages(3, "why do you need me for this"))
	if err != nil {
		t.Fatal(err)
	}
	if answers[2].Category != FalseInability || answers[2].Provider != "ollaya" || answers[2].ResourceRole != "decide.local" {
		t.Fatalf("answer = %+v", answers[2])
	}
}
