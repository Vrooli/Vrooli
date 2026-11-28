package inference

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func TestEmbedderUsesDistinctGatewayTasks(t *testing.T) {
	fake := &fakeInference{embedOut: []float64{0.1}}
	embedder := Embedder{Client: fake}
	if _, err := embedder.EmbedQuery(context.Background(), "query"); err != nil {
		t.Fatal(err)
	}
	if _, err := embedder.EmbedDocument(context.Background(), "document"); err != nil {
		t.Fatal(err)
	}
	if got, want := fake.embedTasks, []EmbeddingTask{EmbeddingQuery, EmbeddingDocument}; !equalTasks(got, want) {
		t.Fatalf("tasks = %v, want %v", got, want)
	}
}

func TestGatewayClientUsesTypedLocalPrivacyContracts(t *testing.T) {
	service := &gatewayInferenceStub{}
	path, handler := inferenceconnect.NewInferenceServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewGatewayClient(inferenceconnect.NewInferenceServiceClient(server.Client(), server.URL))

	vector, err := client.Embed(context.Background(), "signal", EmbeddingDocument)
	if err != nil {
		t.Fatal(err)
	}
	if len(vector) != 2 || vector[0] != 0.1 {
		t.Fatalf("embedding = %v", vector)
	}
	options := []ClassificationOption{{ID: "category-a", Description: "Research"}, {ID: "category-b", Description: "Personal"}}
	result, err := client.Classify(context.Background(), "captured signal", options)
	if err != nil {
		t.Fatal(err)
	}
	if result.CategoryID != "category-b" || result.Confidence != 0.8 || result.Model != "decision-model" {
		t.Fatalf("classification result = %+v", result)
	}

	if got := service.embedRequest.GetRole(); got != EmbeddingRole {
		t.Errorf("embedding role = %q, want %q", got, EmbeddingRole)
	}
	if got := service.embedRequest.GetTexts(); len(got) != 1 || got[0] != documentPrefix+"signal" {
		t.Errorf("embedding texts = %v", got)
	}
	assertContract := func(name string, profile sharedv1.Profile, class sharedv1.PrivacyClass, provenance string) {
		t.Helper()
		if profile != sharedv1.Profile_PROFILE_LOCAL_ONLY || class != sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL || provenance == "" {
			t.Errorf("%s privacy contract = profile:%v class:%v provenance:%q", name, profile, class, provenance)
		}
	}
	assertContract("embedding", service.embedRequest.GetProfile(), service.embedRequest.GetPrivacyClass(), service.embedRequest.GetPrivacyProvenance())
	if service.embedRequest.GetPrivacyProvenance() != EmbeddingProvenance {
		t.Errorf("embedding provenance = %q, want %q", service.embedRequest.GetPrivacyProvenance(), EmbeddingProvenance)
	}
	if got := service.decideRequest.GetRole(); got != DecisionRole {
		t.Errorf("decision role = %q, want %q", got, DecisionRole)
	}
	if got := service.decideRequest.GetState(); got != "captured signal" {
		t.Errorf("decision state = %q", got)
	}
	questions := service.decideRequest.GetQuestions()
	if len(questions) != 1 || len(questions[0].GetOptions()) != 2 || questions[0].GetOptions()[0].GetLabel() != "category-a" || questions[0].GetOptions()[1].GetLabel() != "category-b" {
		t.Errorf("decision questions = %+v", questions)
	}
	assertContract("decision", service.decideRequest.GetProfile(), service.decideRequest.GetPrivacyClass(), service.decideRequest.GetPrivacyProvenance())
	if service.decideRequest.GetPrivacyProvenance() != ClassificationProvenance {
		t.Errorf("decision provenance = %q, want %q", service.decideRequest.GetPrivacyProvenance(), ClassificationProvenance)
	}
}

func TestGatewayClientRefusesInvalidDecisionOptionSetsBeforeDispatch(t *testing.T) {
	service := &gatewayInferenceStub{}
	path, handler := inferenceconnect.NewInferenceServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewGatewayClient(inferenceconnect.NewInferenceServiceClient(server.Client(), server.URL))
	if _, err := client.Classify(context.Background(), "signal", []ClassificationOption{{ID: "only"}}); err == nil {
		t.Fatal("one option must not dispatch a model decision")
	}
	if service.decideRequest != nil {
		t.Fatal("invalid option set reached AI Gateway")
	}
}

type fakeInference struct {
	embedOut   []float64
	embedTasks []EmbeddingTask
}

func (f *fakeInference) Embed(_ context.Context, _ string, task EmbeddingTask) ([]float64, error) {
	f.embedTasks = append(f.embedTasks, task)
	return f.embedOut, nil
}

func (*fakeInference) Classify(context.Context, string, []ClassificationOption) (ClassificationResult, error) {
	return ClassificationResult{}, nil
}

type gatewayInferenceStub struct {
	inferenceconnect.UnimplementedInferenceServiceHandler
	embedRequest  *inferencev1.EmbedRequest
	decideRequest *inferencev1.DecideRequest
}

func (s *gatewayInferenceStub) Embed(_ context.Context, request *connect.Request[inferencev1.EmbedRequest]) (*connect.Response[inferencev1.EmbedResponse], error) {
	s.embedRequest = request.Msg
	return connect.NewResponse(&inferencev1.EmbedResponse{Vectors: []*inferencev1.EmbeddingVector{{Values: []float64{0.1, 0.2}}}, Dimension: 2}), nil
}

func (s *gatewayInferenceStub) Decide(_ context.Context, request *connect.Request[inferencev1.DecideRequest]) (*connect.Response[inferencev1.DecideResponse], error) {
	s.decideRequest = request.Msg
	confidence := 0.8
	return connect.NewResponse(&inferencev1.DecideResponse{Model: "decision-model", Answers: []*inferencev1.DecisionAnswer{{QuestionId: classificationQuestionID, Choice: "category-b", Confidence: &confidence}}}), nil
}

func equalTasks(got, want []EmbeddingTask) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
