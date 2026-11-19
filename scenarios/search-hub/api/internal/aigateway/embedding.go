package aigateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/ai-go/search"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// TaskEmbeddingClient implements Search Hub's task-prefixed embedding seam
// through AI Gateway's typed local-only Embed API.
type TaskEmbeddingClient struct {
	resolve           URLResolver
	client            connect.HTTPClient
	role              string
	expectedModel     string
	expectedDimension int
	queryPrefix       string
	documentPrefix    string
	privacySource     string
}

var _ aisearch.TaskEmbedder = (*TaskEmbeddingClient)(nil)

func NewTaskEmbedder(resolve URLResolver, client connect.HTTPClient, role, expectedModel string, expectedDimension int, queryPrefix, documentPrefix, privacySource string) *TaskEmbeddingClient {
	return &TaskEmbeddingClient{
		resolve: resolve, client: client, role: strings.TrimSpace(role),
		expectedModel: strings.TrimSpace(expectedModel), expectedDimension: expectedDimension,
		queryPrefix: queryPrefix, documentPrefix: documentPrefix,
		privacySource: strings.TrimSpace(privacySource),
	}
}

func (e *TaskEmbeddingClient) Available(ctx context.Context) bool {
	if e == nil || e.resolve == nil {
		return false
	}
	url, err := e.resolve(ctx)
	return err == nil && strings.TrimSpace(url) != ""
}

func (e *TaskEmbeddingClient) Embed(ctx context.Context, text string) ([]float64, error) {
	return e.embed(ctx, text)
}

func (e *TaskEmbeddingClient) EmbedQuery(ctx context.Context, text string) ([]float64, error) {
	return e.embed(ctx, e.queryPrefix+text)
}

func (e *TaskEmbeddingClient) EmbedDocument(ctx context.Context, text string) ([]float64, error) {
	return e.embed(ctx, e.documentPrefix+text)
}

func (e *TaskEmbeddingClient) embed(ctx context.Context, text string) ([]float64, error) {
	if e == nil || e.resolve == nil || e.role == "" || e.expectedModel == "" || e.expectedDimension <= 0 || e.privacySource == "" {
		return nil, errors.New("AI Gateway embedding contract is incomplete")
	}
	baseURL, err := e.resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve AI Gateway: %w", err)
	}
	httpClient := e.client
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	client := inferenceconnect.NewInferenceServiceClient(httpClient, strings.TrimRight(baseURL, "/"))
	response, err := client.Embed(ctx, connect.NewRequest(&inferencev1.EmbedRequest{
		Texts:             []string{text},
		Role:              e.role,
		Profile:           sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass:      sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacyProvenance: e.privacySource,
	}))
	if err != nil {
		return nil, fmt.Errorf("AI Gateway embedding request: %w", err)
	}
	if gatewayErr := response.Msg.GetError(); gatewayErr != nil {
		return nil, fmt.Errorf("AI Gateway embedding refused request (code %s)", gatewayErr.GetCode().String())
	}
	if response.Msg.GetModel() != e.expectedModel {
		return nil, fmt.Errorf("AI Gateway embedding model identity differs from the pinned recipe")
	}
	if response.Msg.GetDimension() != int32(e.expectedDimension) || len(response.Msg.GetVectors()) != 1 {
		return nil, errors.New("AI Gateway embedding dimensions or vector count differ from the pinned recipe")
	}
	values := response.Msg.GetVectors()[0].GetValues()
	if len(values) != e.expectedDimension {
		return nil, errors.New("AI Gateway embedding vector length differs from the pinned recipe")
	}
	return append([]float64(nil), values...), nil
}
