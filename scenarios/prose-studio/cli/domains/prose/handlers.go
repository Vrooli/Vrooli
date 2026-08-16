package prose

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/prose-studio/v1/prose"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prose-studio/v1/prose/prose_v1connect"
)

type handlers struct {
	client connectv1.ProseStudioServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: connectv1.NewProseStudioServiceClient(httpClient, baseURL)}
}

func (h *handlers) call(method string) func(cliapp.OperationContext) (*v1.JsonResponse, error) {
	return func(ctx cliapp.OperationContext) (*v1.JsonResponse, error) {
		payload := ctx.Flag("json")
		if payload == "" {
			payload = `{}`
		}
		var msg v1.JsonRequest
		msg.Json = payload
		var resp *connect.Response[v1.JsonResponse]
		var err error
		switch method {
		case "Registry":
			resp, err = h.client.Registry(context.Background(), connect.NewRequest(&msg))
		case "CreateStyle":
			resp, err = h.client.CreateStyle(context.Background(), connect.NewRequest(&msg))
		case "ResolveProfile":
			resp, err = h.client.ResolveProfile(context.Background(), connect.NewRequest(&msg))
		case "Generate":
			resp, err = h.client.Generate(context.Background(), connect.NewRequest(&msg))
		case "Reroll":
			resp, err = h.client.Reroll(context.Background(), connect.NewRequest(&msg))
		case "SessionAction":
			resp, err = h.client.SessionAction(context.Background(), connect.NewRequest(&msg))
		case "ReindexDeclarations":
			resp, err = h.client.ReindexDeclarations(context.Background(), connect.NewRequest(&msg))
		case "ValidateDeclarations":
			resp, err = h.client.ValidateDeclarations(context.Background(), connect.NewRequest(&msg))
		case "CreateDocument":
			resp, err = h.client.CreateDocument(context.Background(), connect.NewRequest(&msg))
		case "AssembleDocument":
			resp, err = h.client.AssembleDocument(context.Background(), connect.NewRequest(&msg))
		case "Conformance":
			resp, err = h.client.Conformance(context.Background(), connect.NewRequest(&msg))
		default:
			return nil, fmt.Errorf("unknown prose method %q", method)
		}
		if err != nil {
			return nil, cliapp.WrapAPIError("prose "+method, err, nil)
		}
		if resp == nil || resp.Msg == nil {
			return nil, fmt.Errorf("server returned no prose response")
		}
		return resp.Msg, nil
	}
}

func (h *handlers) report(_ cliapp.OperationContext, response *v1.JsonResponse) cliapp.ListReport {
	var pretty any
	if json.Unmarshal([]byte(response.GetJson()), &pretty) == nil {
		b, _ := json.MarshalIndent(pretty, "", "  ")
		return cliapp.ListReport{Summary: []string{"Prose Studio response"}, ResultsHeading: "Response", Results: []string{string(b)}}
	}
	return cliapp.ListReport{Summary: []string{"Prose Studio response"}, ResultsHeading: "Response", Results: []string{response.GetJson()}}
}
