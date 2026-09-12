package posttypes

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	posttypesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/posttypes"
	posttypesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/posttypes/posttypes_v1connect"
)

const GroupName = "posttypes"

type handlers struct {
	client posttypesconnect.PosttypesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: posttypesconnect.NewPosttypesServiceClient(httpClient, baseURL)}
}

func (h *handlers) listCall(_ cliapp.OperationContext) (*posttypesv1.ListPostTypesResponse, error) {
	response, err := h.client.ListPostTypes(context.Background(), connect.NewRequest(&posttypesv1.ListPostTypesRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list post types", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no post types response")
	}
	return response.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, message *posttypesv1.ListPostTypesResponse) cliapp.ListReport {
	results := make([]string, 0, len(message.PostTypes))
	for _, postType := range message.PostTypes {
		results = append(results, fmt.Sprintf("%s — %s", postType.Id, postType.Status))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d post type(s).", len(message.PostTypes))}, ResultsHeading: "Post types", Results: results}
}

func (h *handlers) registerCall(ctx cliapp.OperationContext) (*posttypesv1.RegisterPostTypeResponse, error) {
	modes := strings.Split(ctx.Flag("failure-modes"), ",")
	response, err := h.client.RegisterPostType(context.Background(), connect.NewRequest(&posttypesv1.RegisterPostTypeRequest{Id: ctx.Flag("id"), PairedSkill: ctx.Flag("paired-skill"), SkillExists: ctx.BoolFlag("skill-exists"), DocV1: ctx.BoolFlag("doc-v1"), ResponsibilitiesDeclared: ctx.BoolFlag("responsibilities-declared"), Activate: ctx.BoolFlag("activate"), FailureModes: modes}))
	if err != nil {
		return nil, cliapp.WrapAPIError("register post type", err, nil)
	}
	if response == nil || response.Msg == nil || response.Msg.PostType == nil {
		return nil, fmt.Errorf("server returned no post type")
	}
	return response.Msg, nil
}
func (h *handlers) registerReport(_ cliapp.OperationContext, message *posttypesv1.RegisterPostTypeResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Post type %s is %s.", message.PostType.Id, message.PostType.Status)}}
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{"PosttypesService.ListPostTypes": cliapp.ProtoList(h.listCall, h.listReport), "PosttypesService.RegisterPostType": cliapp.ProtoMutation(h.registerCall, h.registerReport)})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("posttypes: load from manifest: %w", err)
	}
	return group, nil
}
