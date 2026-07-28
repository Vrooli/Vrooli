// Package posttypes mounts the post-type registry Connect surface.
package posttypes

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"content-desk/internal/module"
	internalposttypes "content-desk/internal/posttypes"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	posttypesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/posttypes"
	posttypesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/posttypes/posttypes_v1connect"
)

type handler struct{ registry internalposttypes.Registry }

var _ posttypesconnect.PosttypesServiceHandler = handler{}

func (h handler) ListPostTypes(ctx context.Context, _ *connect.Request[posttypesv1.ListPostTypesRequest]) (*connect.Response[posttypesv1.ListPostTypesResponse], error) {
	types, err := h.registry.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &posttypesv1.ListPostTypesResponse{}
	for _, postType := range types {
		evaluation, err := h.registry.Evaluate(ctx, postType.ID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		status := internalposttypes.StatusV0
		if evaluation.Active {
			status = internalposttypes.StatusActive
		}
		response.PostTypes = append(response.PostTypes, postTypeMessage(postType.ID, status, postType.FailureModes))
	}
	return connect.NewResponse(response), nil
}

func (h handler) RegisterPostType(ctx context.Context, request *connect.Request[posttypesv1.RegisterPostTypeRequest]) (*connect.Response[posttypesv1.RegisterPostTypeResponse], error) {
	status := internalposttypes.StatusV0
	if request.Msg.Activate {
		status = internalposttypes.StatusActive
	}
	postType := internalposttypes.PostType{ID: request.Msg.Id, Status: status, PairedSkill: request.Msg.PairedSkill, SkillExists: request.Msg.SkillExists, DocV1: request.Msg.DocV1, ResponsibilitiesDeclared: request.Msg.ResponsibilitiesDeclared, FailureModes: request.Msg.FailureModes}
	if err := h.registry.Upsert(ctx, postType); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	evaluation, err := h.registry.Evaluate(ctx, postType.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if request.Msg.Activate && !evaluation.Active {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("post type %q fails activation criteria", postType.ID))
	}
	actualStatus := internalposttypes.StatusV0
	if evaluation.Active {
		actualStatus = internalposttypes.StatusActive
	}
	return connect.NewResponse(&posttypesv1.RegisterPostTypeResponse{PostType: postTypeMessage(postType.ID, actualStatus, postType.FailureModes)}), nil
}

func postTypeMessage(id, status string, failureModes []string) *posttypesv1.PostType {
	return &posttypesv1.PostType{Id: id, Status: status, FailureModes: failureModes}
}

func Module(db *database.RoutedDB) module.Module {
	path, h := posttypesconnect.NewPosttypesServiceHandler(handler{registry: internalposttypes.NewRegistry(db)})
	return module.Module{Name: "posttypes", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}
func Schema() string { return internalposttypes.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "posttypes_list", Path: posttypesconnect.PosttypesServiceListPostTypesProcedure, Method: "POST", Summary: "List post types", Category: "posttypes"},
	{ID: "posttypes_register", Path: posttypesconnect.PosttypesServiceRegisterPostTypeProcedure, Method: "POST", Summary: "Register or activate a post type", Category: "posttypes"},
}
