package tags

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	tagsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/tags"
	tagsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/tags/tags_v1connect"

	domain "prompt-manager/internal/tags"
)

type connectHandler struct {
	tagsconnect.UnimplementedTagsServiceHandler
	repo domain.TagRepository
}

func NewConnectMount(repo domain.TagRepository) (string, http.Handler) {
	return tagsconnect.NewTagsServiceHandler(&connectHandler{repo: repo})
}

func (h *connectHandler) ListTags(ctx context.Context, req *connect.Request[tagsv1.ListTagsRequest]) (*connect.Response[tagsv1.ListTagsResponse], error) {
	items, err := domain.ListTags(ctx, h.repo)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &tagsv1.ListTagsResponse{}
	for _, item := range items {
		out.Tags = append(out.Tags, &tagsv1.Tag{Id: item.ID, Name: item.Name, Color: item.Color, Description: item.Description})
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateTag(ctx context.Context, req *connect.Request[tagsv1.CreateTagRequest]) (*connect.Response[tagsv1.CreateTagResponse], error) {
	item, err := domain.CreateTag(ctx, h.repo, domain.Tag{Name: req.Msg.GetName(), Color: req.Msg.Color, Description: req.Msg.Description})
	if err != nil {
		code := connect.CodeInternal
		if err == domain.ErrDuplicate {
			code = connect.CodeAlreadyExists
		}
		if err.Error() == "Name is required" {
			code = connect.CodeInvalidArgument
		}
		return nil, connect.NewError(code, err)
	}
	return connect.NewResponse(&tagsv1.CreateTagResponse{Tag: &tagsv1.Tag{Id: item.ID, Name: item.Name, Color: item.Color, Description: item.Description}}), nil
}
