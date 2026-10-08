package administration

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	domainadmin "landing-page-business-suite-api/internal/administration"
)

type (
	ReaderTokenDependencies struct{ Service *domainadmin.ReaderTokens }
	ReaderTokenHandler      struct{ service *domainadmin.ReaderTokens }
)

func NewReaderTokenHandler(d ReaderTokenDependencies) *ReaderTokenHandler {
	return &ReaderTokenHandler{service: d.Service}
}

func (h *ReaderTokenHandler) IssueReaderToken(ctx context.Context, r *connect.Request[lpbsv1.IssueReaderTokenRequest]) (*connect.Response[lpbsv1.IssuedReaderToken], error) {
	if h.service == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("reader token service unavailable"))
	}
	t, err := h.service.Issue(ctx, r.Msg.GetLabel(), r.Msg.GetScope())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&lpbsv1.IssuedReaderToken{Id: t.ID, Token: t.Token, Prefix: t.Prefix, Scope: t.Scope, CreatedAt: timestamppb.New(t.CreatedAt)}), nil
}

func (h *ReaderTokenHandler) ListReaderTokens(ctx context.Context, _ *connect.Request[lpbsv1.ListReaderTokensRequest]) (*connect.Response[lpbsv1.ListReaderTokensResponse], error) {
	ts, err := h.service.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*lpbsv1.ReaderToken, 0, len(ts))
	for _, t := range ts {
		p := &lpbsv1.ReaderToken{Id: t.ID, Label: t.Label, Scope: t.Scope, Prefix: t.Prefix, Source: t.Source, CreatedAt: timestamppb.New(t.CreatedAt)}
		if t.LastUsedAt != nil {
			p.LastUsedAt = timestamppb.New(*t.LastUsedAt)
		}
		if t.RevokedAt != nil {
			p.RevokedAt = timestamppb.New(*t.RevokedAt)
		}
		out = append(out, p)
	}
	return connect.NewResponse(&lpbsv1.ListReaderTokensResponse{Tokens: out}), nil
}

func (h *ReaderTokenHandler) RevokeReaderToken(ctx context.Context, r *connect.Request[lpbsv1.RevokeReaderTokenRequest]) (*connect.Response[lpbsv1.RevokeReaderTokenResponse], error) {
	if err := h.service.Revoke(ctx, r.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&lpbsv1.RevokeReaderTokenResponse{Revoked: true}), nil
}

var _ lpbsconnect.ReaderTokenServiceHandler = (*ReaderTokenHandler)(nil)
