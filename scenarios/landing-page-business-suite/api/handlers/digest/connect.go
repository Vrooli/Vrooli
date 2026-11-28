package digesthttp

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

type Reader interface {
	GetBusinessDigest(context.Context, int32) (*lpbsv1.BusinessDigest, error)
}
type Handler struct{ reader Reader }

func New(reader Reader) *Handler { return &Handler{reader: reader} }
func (h *Handler) GetBusinessDigest(ctx context.Context, req *connect.Request[lpbsv1.GetBusinessDigestRequest]) (*connect.Response[lpbsv1.BusinessDigest], error) {
	days := req.Msg.GetWindowDays()
	if days == 0 {
		days = 30
	}
	if days != 7 && days != 30 && days != 90 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("window_days must be 7, 30, or 90"))
	}
	result, err := h.reader.GetBusinessDigest(ctx, days)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if result == nil {
		result = &lpbsv1.BusinessDigest{}
	}
	if result.ContractVersion == "" {
		result.ContractVersion = "business-digest.v1"
	}
	if result.ObservedAt == nil {
		result.ObservedAt = timestamppb.New(time.Now().UTC())
	}
	return connect.NewResponse(result), nil
}

var _ lpbsconnect.BusinessDigestServiceHandler = (*Handler)(nil)
