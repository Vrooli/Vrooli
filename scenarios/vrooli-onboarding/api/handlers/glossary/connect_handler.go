package glossary

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	glossaryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/glossary"
	glossaryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/glossary/glossaryv1connect"
	internalglossary "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/glossary"
)

type connectHandler struct{}

func NewConnectHandler() glossaryconnect.GlossaryServiceHandler { return &connectHandler{} }

func (*connectHandler) SearchGlossary(ctx context.Context, req *connect.Request[glossaryv1.SearchGlossaryRequest]) (*connect.Response[glossaryv1.SearchGlossaryResponse], error) {
	value, err := internalglossary.Search(ctx, req.Msg.GetQuery())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("search glossary: %w", err))
	}
	return connect.NewResponse(value), nil
}

func (*connectHandler) SearchConfiguration(ctx context.Context, req *connect.Request[glossaryv1.SearchConfigurationRequest]) (*connect.Response[glossaryv1.SearchConfigurationResponse], error) {
	value, err := internalglossary.SearchConfiguration(ctx, req.Msg.GetQuery(), req.Msg.GetTarget())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("search configuration: %w", err))
	}
	return connect.NewResponse(value), nil
}
