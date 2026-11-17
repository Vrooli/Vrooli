// Package inference is Signal Inbox's sole boundary for model inference.
package inference

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	aisearch "github.com/vrooli/ai-go/search"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const (
	EmbeddingRole            = "embedding.default"
	DecisionRole             = "decide.fast"
	EmbeddingProvenance      = "signal_inbox.user_captured_signal_content_and_search_query"
	ClassificationProvenance = "signal_inbox.user_captured_signal_content_and_category_definitions"
	queryPrefix              = "search_query: "
	documentPrefix           = "search_document: "
	classificationQuestionID = "category"
)

// EmbeddingTask preserves the asymmetric query/document instruction required
// by the gateway's embedding role.
type EmbeddingTask string

const (
	EmbeddingDocument EmbeddingTask = "document"
	EmbeddingQuery    EmbeddingTask = "query"
)

// Client is the scenario-owned inference seam. Callers never import provider
// clients or invoke resource commands directly.
type Client interface {
	Embed(context.Context, string, EmbeddingTask) ([]float64, error)
	Classify(context.Context, string, []ClassificationOption) (ClassificationResult, error)
}

type ClassificationOption struct {
	ID          string
	Description string
}

type ClassificationResult struct {
	CategoryID string
	Confidence float64
	Model      string
}

// GatewayClient routes every inference request through ai-gateway.
type GatewayClient struct {
	inference inferenceconnect.InferenceServiceClient
}

func NewGatewayClient(client inferenceconnect.InferenceServiceClient) *GatewayClient {
	return &GatewayClient{inference: client}
}

func (c *GatewayClient) Embed(ctx context.Context, text string, task EmbeddingTask) ([]float64, error) {
	if c == nil || c.inference == nil {
		return nil, errors.New("ai-gateway inference client is not configured")
	}
	response, err := c.inference.Embed(ctx, connect.NewRequest(&inferencev1.EmbedRequest{
		Texts: []string{embeddingInput(task, text)}, Role: EmbeddingRole,
		Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacyProvenance: EmbeddingProvenance,
	}))
	if err != nil {
		return nil, fmt.Errorf("execute ai-gateway embedding role %s: %w", EmbeddingRole, err)
	}
	if response == nil || response.Msg == nil {
		return nil, errors.New("ai-gateway returned no embedding response")
	}
	if response.Msg.GetError() != nil {
		return nil, fmt.Errorf("ai-gateway rejected embedding request: %s", response.Msg.GetError().GetMessage())
	}
	if len(response.Msg.GetVectors()) != 1 || len(response.Msg.GetVectors()[0].GetValues()) == 0 {
		return nil, errors.New("ai-gateway embedding response contained no vector")
	}
	return response.Msg.GetVectors()[0].GetValues(), nil
}

func (c *GatewayClient) Classify(ctx context.Context, state string, options []ClassificationOption) (ClassificationResult, error) {
	if c == nil || c.inference == nil {
		return ClassificationResult{}, errors.New("ai-gateway inference client is not configured")
	}
	if len(options) < 2 || len(options) > 255 {
		return ClassificationResult{}, fmt.Errorf("classification requires 2 to 255 category options, got %d", len(options))
	}
	decisionOptions := make([]*inferencev1.DecisionOption, 0, len(options))
	for _, option := range options {
		id := strings.TrimSpace(option.ID)
		if id == "" {
			return ClassificationResult{}, errors.New("classification category ID is required")
		}
		decisionOptions = append(decisionOptions, &inferencev1.DecisionOption{Label: id, Description: strings.TrimSpace(option.Description)})
	}
	response, err := c.inference.Decide(ctx, connect.NewRequest(&inferencev1.DecideRequest{
		Role: DecisionRole, State: state,
		Questions: []*inferencev1.DecisionQuestion{{Id: classificationQuestionID, Type: inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE,
			Instructions: "Choose the single active category that best matches the signal content.", Options: decisionOptions}},
		Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacyProvenance: ClassificationProvenance,
	}))
	if err != nil {
		return ClassificationResult{}, fmt.Errorf("execute ai-gateway decision role %s: %w", DecisionRole, err)
	}
	if response == nil || response.Msg == nil {
		return ClassificationResult{}, errors.New("ai-gateway returned no decision response")
	}
	if response.Msg.GetError() != nil {
		return ClassificationResult{}, fmt.Errorf("ai-gateway rejected decision request: %s", response.Msg.GetError().GetMessage())
	}
	if len(response.Msg.GetAnswers()) != 1 || response.Msg.GetAnswers()[0].GetQuestionId() != classificationQuestionID {
		return ClassificationResult{}, errors.New("ai-gateway returned an incomplete category decision")
	}
	answer := response.Msg.GetAnswers()[0]
	categoryID := strings.TrimSpace(answer.GetChoice())
	for _, option := range options {
		if categoryID == strings.TrimSpace(option.ID) {
			confidence := 0.0
			if answer.Confidence != nil {
				confidence = answer.GetConfidence()
			}
			return ClassificationResult{CategoryID: categoryID, Confidence: confidence, Model: response.Msg.GetModel()}, nil
		}
	}
	return ClassificationResult{}, errors.New("ai-gateway returned a category outside the declared option set")
}

func embeddingInput(task EmbeddingTask, text string) string {
	if task == EmbeddingQuery {
		return queryPrefix + text
	}
	return documentPrefix + text
}

// Embedder adapts Client to ai-go's task-aware embedding seam.
type Embedder struct{ Client Client }

func (e Embedder) Embed(ctx context.Context, text string) ([]float64, error) {
	return e.EmbedDocument(ctx, text)
}

func (e Embedder) EmbedQuery(ctx context.Context, text string) ([]float64, error) {
	return e.Client.Embed(ctx, text, EmbeddingQuery)
}

func (e Embedder) EmbedDocument(ctx context.Context, text string) ([]float64, error) {
	return e.Client.Embed(ctx, text, EmbeddingDocument)
}

func (e Embedder) Available(ctx context.Context) bool {
	_, err := e.Embed(ctx, "ping")
	return err == nil
}

var (
	_ aisearch.Embedder     = Embedder{}
	_ aisearch.TaskEmbedder = Embedder{}
)
