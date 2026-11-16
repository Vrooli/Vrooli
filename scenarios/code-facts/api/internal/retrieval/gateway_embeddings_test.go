package retrieval

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	aisearch "github.com/vrooli/ai-go/search"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type fakeEmbeddingClient struct {
	request  *inferencev1.EmbedRequest
	response *inferencev1.EmbedResponse
	calls    int
}

func (f *fakeEmbeddingClient) Embed(_ context.Context, req *connect.Request[inferencev1.EmbedRequest]) (*connect.Response[inferencev1.EmbedResponse], error) {
	f.request = req.Msg
	f.calls++
	return connect.NewResponse(f.response), nil
}

func TestGatewayEmbedderPreservesBatchBoundary(t *testing.T) {
	client := &fakeEmbeddingClient{response: &inferencev1.EmbedResponse{
		Model: "embed-policy-model", Dimension: 2,
		Vectors: []*inferencev1.EmbeddingVector{{Values: []float64{1, 2}}, {Values: []float64{3, 4}}},
	}}
	embedder := &GatewayEmbedder{Client: client, ExpectedModel: "embed-policy-model", Dimensions: 2, TaskPrefix: "passage: "}
	vectors, err := embedder.Embed(context.Background(), []string{"first", "second"})
	if err != nil {
		t.Fatal(err)
	}
	if client.calls != 1 || len(vectors) != 2 || len(client.request.GetTexts()) != 2 {
		t.Fatalf("batch contract was split or lost: calls=%d request=%v vectors=%v", client.calls, client.request.GetTexts(), vectors)
	}
	if client.request.GetTexts()[0] != "passage: first" || client.request.GetTexts()[1] != "passage: second" {
		t.Fatalf("batch prefixes mismatch: %v", client.request.GetTexts())
	}
}

func TestGatewayEmbedderUsesLocalConfidentialRoleContract(t *testing.T) {
	client := &fakeEmbeddingClient{response: &inferencev1.EmbedResponse{
		Model: "embed-policy-model", Dimension: 3,
		Vectors: []*inferencev1.EmbeddingVector{{Values: []float64{0.25, -0.5, 1}}},
	}}
	embedder := &GatewayEmbedder{Client: client, ExpectedModel: "embed-policy-model", Dimensions: 3, TaskPrefix: "search_document: "}
	vectors, err := embedder.Embed(context.Background(), []string{"repository source excerpt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 1 || len(vectors[0]) != 3 || vectors[0][1] != -0.5 {
		t.Fatalf("unexpected vectors: %#v", vectors)
	}
	req := client.request
	if req.GetRole() != EmbeddingRole {
		t.Fatalf("embedding task contract mismatch: %+v", req)
	}
	if req.GetProfile() != sharedv1.Profile_PROFILE_LOCAL_ONLY || req.GetPrivacyClass() != sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL || req.GetPrivacyProvenance() != codeFactsProvenance {
		t.Fatalf("embedding privacy contract mismatch: %+v", req)
	}
	if len(client.request.GetTexts()) != 1 || client.request.GetTexts()[0] != "search_document: repository source excerpt" {
		t.Fatalf("document task prefix missing: %q", client.request.GetTexts())
	}
}

func TestGatewayEmbedderRejectsModelDrift(t *testing.T) {
	client := &fakeEmbeddingClient{response: &inferencev1.EmbedResponse{
		Model: "different-model", Dimension: 2,
		Vectors: []*inferencev1.EmbeddingVector{{Values: []float64{1, 2}}},
	}}
	if _, err := (&GatewayEmbedder{Client: client, ExpectedModel: "embed-policy-model", Dimensions: 2}).Embed(context.Background(), []string{"private source"}); err == nil {
		t.Fatal("model identity drift must be rejected")
	}
}

func TestGatewayEmbedderRejectsGatewayRefusal(t *testing.T) {
	client := &fakeEmbeddingClient{response: &inferencev1.EmbedResponse{
		Error: &inferencev1.InferenceError{Code: inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST},
	}}
	if _, err := (&GatewayEmbedder{Client: client, Dimensions: 2}).Embed(context.Background(), []string{"private source"}); err == nil {
		t.Fatal("Gateway refusal must remain fail-closed")
	}
}

type untouchedVectorStore struct{ queries int }

func (*untouchedVectorStore) Upsert(context.Context, []VectorRecord) error { return nil }
func (*untouchedVectorStore) Delete(context.Context, []string) error       { return nil }
func (s *untouchedVectorStore) Query(context.Context, []float32, Query) ([]Candidate, error) {
	s.queries++
	return nil, nil
}

func TestSemanticQueryCannotOverrideEmbeddingPolicy(t *testing.T) {
	store := &untouchedVectorStore{}
	searcher := GatewaySemanticSearcher{
		Store: store,
		ResolvePolicy: func(context.Context) (aisearch.EmbeddingPolicy, error) {
			return aisearch.EmbeddingPolicy{Role: EmbeddingRole, Model: "policy-model", Dimensions: 3}, nil
		},
	}
	_, err := searcher.SearchSemantic(context.Background(), Query{
		Text: "private repository query", EmbeddingModel: "caller-selected-model", EmbeddingDimensions: 3,
	})
	if err == nil {
		t.Fatal("caller-supplied embedding model must not override resolved policy")
	}
	if store.queries != 0 {
		t.Fatal("vector store must not receive a vector from an unapproved model")
	}
}
