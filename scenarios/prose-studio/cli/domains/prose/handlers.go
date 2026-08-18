package prose

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/prose-studio/v1/prose"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prose-studio/v1/prose/prose_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type handlers struct {
	client connectv1.ProseStudioServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: connectv1.NewProseStudioServiceClient(httpClient, baseURL)}
}

func requestFromFlag[T proto.Message](ctx cliapp.OperationContext, message T) (T, error) {
	payload := ctx.Flag("request-json")
	if payload == "" {
		payload = `{}`
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal([]byte(payload), message); err != nil {
		return message, fmt.Errorf("decode --json: %w", err)
	}
	return message, nil
}

func (h *handlers) registryCall(ctx cliapp.OperationContext) (*v1.RegistryResponse, error) {
	request, err := requestFromFlag(ctx, &v1.RegistryRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.Registry(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose Registry", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) createStyleCall(ctx cliapp.OperationContext) (*v1.CreateStyleResponse, error) {
	request, err := requestFromFlag(ctx, &v1.CreateStyleRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.CreateStyle(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose CreateStyle", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) resolveProfileCall(ctx cliapp.OperationContext) (*v1.ResolveProfileResponse, error) {
	request, err := requestFromFlag(ctx, &v1.ResolveProfileRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.ResolveProfile(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose ResolveProfile", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) generateCall(ctx cliapp.OperationContext) (*v1.GenerateResponse, error) {
	request, err := requestFromFlag(ctx, &v1.GenerateRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.Generate(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose Generate", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) rerollCall(ctx cliapp.OperationContext) (*v1.RerollResponse, error) {
	request, err := requestFromFlag(ctx, &v1.RerollRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.Reroll(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose Reroll", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) sessionActionCall(ctx cliapp.OperationContext) (*v1.SessionActionResponse, error) {
	request, err := requestFromFlag(ctx, &v1.SessionActionRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.SessionAction(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose SessionAction", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) reindexCall(ctx cliapp.OperationContext) (*v1.ReindexDeclarationsResponse, error) {
	request, err := requestFromFlag(ctx, &v1.ReindexDeclarationsRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.ReindexDeclarations(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose ReindexDeclarations", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) validateCall(ctx cliapp.OperationContext) (*v1.ValidateDeclarationsResponse, error) {
	request, err := requestFromFlag(ctx, &v1.ValidateDeclarationsRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.ValidateDeclarations(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose ValidateDeclarations", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) createDocumentCall(ctx cliapp.OperationContext) (*v1.CreateDocumentResponse, error) {
	request, err := requestFromFlag(ctx, &v1.CreateDocumentRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.CreateDocument(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose CreateDocument", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) assembleDocumentCall(ctx cliapp.OperationContext) (*v1.AssembleDocumentResponse, error) {
	request, err := requestFromFlag(ctx, &v1.AssembleDocumentRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.AssembleDocument(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose AssembleDocument", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) resumeDocumentCall(ctx cliapp.OperationContext) (*v1.ResumeDocumentResponse, error) {
	request, err := requestFromFlag(ctx, &v1.ResumeDocumentRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.ResumeDocument(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose ResumeDocument", err, nil)
	}
	return responseMessage(response)
}
func (h *handlers) conformanceCall(ctx cliapp.OperationContext) (*v1.ConformanceResponse, error) {
	request, err := requestFromFlag(ctx, &v1.ConformanceRequest{})
	if err != nil {
		return nil, err
	}
	response, err := h.client.Conformance(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("prose Conformance", err, nil)
	}
	return responseMessage(response)
}

func responseMessage[T any](response *connect.Response[T]) (*T, error) {
	var zero *T
	if response == nil || response.Msg == nil {
		return zero, fmt.Errorf("server returned no prose response")
	}
	return response.Msg, nil
}

func protoReport[T proto.Message](label string) func(cliapp.OperationContext, T) cliapp.ListReport {
	return func(_ cliapp.OperationContext, message T) cliapp.ListReport {
		raw, _ := (protojson.MarshalOptions{UseProtoNames: true, Multiline: true, Indent: "  "}).Marshal(message)
		var pretty any
		if json.Unmarshal(raw, &pretty) == nil {
			raw, _ = json.MarshalIndent(pretty, "", "  ")
		}
		return cliapp.ListReport{Summary: []string{label}, ResultsHeading: "Response", Results: []string{string(raw)}}
	}
}
