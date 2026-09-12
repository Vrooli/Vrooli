package designinference

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	"github.com/vrooli/api-core/discovery"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"net/http"
	"time"
)

type Gateway interface {
	Run(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error)
}
type Service struct {
	Validate   func(Request, *inferencev1.RunResponse) error
	Repository Repository
	Client     Gateway
	Resolve    func(context.Context) (Gateway, error)
}

func NewService(repo Repository) *Service {
	return &Service{Repository: repo, Resolve: func(ctx context.Context) (Gateway, error) {
		url, err := discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		if err != nil {
			return nil, err
		}
		return inferenceconnect.NewInferenceServiceClient(&http.Client{Timeout: 3 * time.Minute}, url), nil
	}}
}
func (s *Service) Run(ctx context.Context, key string, request Request) (Operation, error) {
	op, err := s.Repository.Create(ctx, key, request)
	if err != nil || op.State != "prepared" {
		return op, err
	}
	client := s.Client
	if client == nil {
		if s.Resolve == nil {
			return op, fmt.Errorf("inference gateway unavailable")
		}
		client, err = s.Resolve(ctx)
		if err != nil {
			return op, err
		}
	}
	claimed, err := s.Repository.Claim(ctx, op.ID)
	if err != nil {
		return op, err
	}
	if !claimed {
		return s.Repository.Get(ctx, op.ID)
	}
	response, callErr := client.Run(ctx, connect.NewRequest(&inferencev1.RunRequest{Source: request.Source, SchemaJson: request.Schema, Instruction: request.Instruction, Role: request.Role, MaxOutputTokens: request.MaxOutputTokens}))
	// Preserve a received response even when the caller stops observing it.
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if callErr != nil || response == nil || response.Msg == nil {
		return s.Repository.Get(saveCtx, op.ID)
	}
	raw, err := protojson.Marshal(response.Msg)
	if err != nil {
		return Operation{}, err
	}
	state, detail := "completed", ""
	if response.Msg.Error != nil || !response.Msg.Validated || response.Msg.ValueJson == "" {
		state, detail = "failed", "Gateway refused or did not validate the inference result."
	}
	if state == "completed" && s.Validate != nil {
		if err := s.Validate(request, response.Msg); err != nil {
			state, detail = "failed", fmt.Sprintf("Design proposal validation failed: %.3500s", err.Error())
		}
	}
	return s.Repository.Finish(saveCtx, op.ID, state, string(raw), detail)
}
