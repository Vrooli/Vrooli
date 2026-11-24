package aigateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	observer          EmbeddingObserver
}

// EmbeddingObservation contains only request metadata. It intentionally omits
// text, provenance strings and vectors so owners can measure query/document
// latency without retaining private content.
type EmbeddingObservation struct {
	Kind        string
	Role        string
	Model       string
	Duration    time.Duration
	VectorCount int
	Failed      bool
}

// EmbeddingObserver is an optional owner-provided hook for recording the
// client-visible duration of one typed gateway embedding operation.
type EmbeddingObserver func(EmbeddingObservation)

var _ aisearch.TaskEmbedder = (*TaskEmbeddingClient)(nil)

func NewTaskEmbedder(resolve URLResolver, client connect.HTTPClient, role, expectedModel string, expectedDimension int, queryPrefix, documentPrefix, privacySource string, observers ...EmbeddingObserver) *TaskEmbeddingClient {
	e := &TaskEmbeddingClient{
		resolve: resolve, client: client, role: strings.TrimSpace(role),
		expectedModel: strings.TrimSpace(expectedModel), expectedDimension: expectedDimension,
		queryPrefix: queryPrefix, documentPrefix: documentPrefix,
		privacySource: strings.TrimSpace(privacySource),
	}
	if len(observers) > 0 {
		e.observer = observers[0]
	}
	return e
}

func (e *TaskEmbeddingClient) Available(ctx context.Context) bool {
	if e == nil || e.resolve == nil {
		return false
	}
	url, err := e.resolve(ctx)
	return err == nil && strings.TrimSpace(url) != ""
}

func (e *TaskEmbeddingClient) Embed(ctx context.Context, text string) ([]float64, error) {
	return e.embed(ctx, "generic", text)
}

func (e *TaskEmbeddingClient) EmbedQuery(ctx context.Context, text string) ([]float64, error) {
	return e.embed(ctx, "query", e.queryPrefix+text)
}

func (e *TaskEmbeddingClient) EmbedDocument(ctx context.Context, text string) ([]float64, error) {
	return e.embed(ctx, "document", e.documentPrefix+text)
}

func (e *TaskEmbeddingClient) embed(ctx context.Context, kind, text string) ([]float64, error) {
	started := time.Now()
	var resultErr error
	vectorCount := 0
	defer func() {
		if e != nil && e.observer != nil {
			e.observer(EmbeddingObservation{Kind: kind, Role: e.role, Model: e.expectedModel, Duration: time.Since(started), VectorCount: vectorCount, Failed: resultErr != nil})
		}
	}()
	if e == nil || e.resolve == nil || e.role == "" || e.expectedModel == "" || e.expectedDimension <= 0 || e.privacySource == "" {
		resultErr = errors.New("AI Gateway embedding contract is incomplete")
		return nil, resultErr
	}
	baseURL, err := e.resolve(ctx)
	if err != nil {
		resultErr = fmt.Errorf("resolve AI Gateway: %w", err)
		return nil, resultErr
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
		resultErr = fmt.Errorf("AI Gateway embedding request: %w", err)
		return nil, resultErr
	}
	if gatewayErr := response.Msg.GetError(); gatewayErr != nil {
		resultErr = fmt.Errorf("AI Gateway embedding refused request (code %s)", gatewayErr.GetCode().String())
		return nil, resultErr
	}
	if response.Msg.GetModel() != e.expectedModel {
		resultErr = errors.New("AI Gateway embedding model identity differs from the pinned recipe")
		return nil, resultErr
	}
	if response.Msg.GetDimension() != int32(e.expectedDimension) || len(response.Msg.GetVectors()) != 1 {
		resultErr = errors.New("AI Gateway embedding dimensions or vector count differ from the pinned recipe")
		return nil, resultErr
	}
	values := response.Msg.GetVectors()[0].GetValues()
	if len(values) != e.expectedDimension {
		resultErr = errors.New("AI Gateway embedding vector length differs from the pinned recipe")
		return nil, resultErr
	}
	vectorCount = len(response.Msg.GetVectors())
	return append([]float64(nil), values...), nil
}
