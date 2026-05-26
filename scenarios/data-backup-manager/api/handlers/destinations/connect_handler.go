package destinations

import (
	"context"
	"log"

	"data-backup-manager/internal/destinations"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	destinationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations"
	destinationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations/destinations_v1connect"
)

// Ensure connectHandler satisfies the generated handler interface.
var _ destinationsconnect.DestinationsServiceHandler = (*connectHandler)(nil)

// Deps wires the seams the Connect destinations handler needs.
type Deps struct {
	Service destinations.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the destinations Connect-RPC handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) CreateDestination(ctx context.Context, req *connect.Request[destinationsv1.CreateDestinationRequest]) (*connect.Response[destinationsv1.CreateDestinationResponse], error) {
	d, err := h.deps.Service.CreateDestination(ctx, destinations.CreateInput{
		Name:      req.Msg.Name,
		Backend:   protoToBackend(req.Msg.BackendKind),
		Location:  req.Msg.Location,
		CapBytes:  req.Msg.CapBytes,
		CapPolicy: protoToCapPolicy(req.Msg.CapPolicy),
	})
	if err != nil {
		return nil, h.translate("CreateDestination", err)
	}
	return connect.NewResponse(&destinationsv1.CreateDestinationResponse{
		Destination: domainToProto(d),
	}), nil
}

func (h *connectHandler) GetDestination(ctx context.Context, req *connect.Request[destinationsv1.GetDestinationRequest]) (*connect.Response[destinationsv1.GetDestinationResponse], error) {
	d, err := h.deps.Service.GetDestination(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.translate("GetDestination", err)
	}
	return connect.NewResponse(&destinationsv1.GetDestinationResponse{
		Destination: domainToProto(d),
	}), nil
}

func (h *connectHandler) ListDestinations(ctx context.Context, req *connect.Request[destinationsv1.ListDestinationsRequest]) (*connect.Response[destinationsv1.ListDestinationsResponse], error) {
	list, err := h.deps.Service.ListDestinations(ctx, int(req.Msg.PageSize))
	if err != nil {
		return nil, h.translate("ListDestinations", err)
	}
	resp := &destinationsv1.ListDestinationsResponse{
		Destinations: make([]*destinationsv1.Destination, 0, len(list)),
	}
	for _, d := range list {
		resp.Destinations = append(resp.Destinations, domainToProto(d))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) UpdateDestination(ctx context.Context, req *connect.Request[destinationsv1.UpdateDestinationRequest]) (*connect.Response[destinationsv1.UpdateDestinationResponse], error) {
	d, err := h.deps.Service.UpdateDestination(ctx, destinations.UpdateInput{
		ID:        req.Msg.Id,
		CapBytes:  req.Msg.CapBytes,
		CapPolicy: protoToCapPolicy(req.Msg.CapPolicy),
	})
	if err != nil {
		return nil, h.translate("UpdateDestination", err)
	}
	return connect.NewResponse(&destinationsv1.UpdateDestinationResponse{
		Destination: domainToProto(d),
	}), nil
}

func (h *connectHandler) DeleteDestination(ctx context.Context, req *connect.Request[destinationsv1.DeleteDestinationRequest]) (*connect.Response[destinationsv1.DeleteDestinationResponse], error) {
	removed, err := h.deps.Service.DeleteDestination(ctx, req.Msg.Id, req.Msg.DeleteRepository)
	if err != nil {
		return nil, h.translate("DeleteDestination", err)
	}
	return connect.NewResponse(&destinationsv1.DeleteDestinationResponse{Removed: removed}), nil
}

func (h *connectHandler) GetDestinationUsage(ctx context.Context, req *connect.Request[destinationsv1.GetDestinationUsageRequest]) (*connect.Response[destinationsv1.GetDestinationUsageResponse], error) {
	report, err := h.deps.Service.GetDestinationUsage(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.translate("GetDestinationUsage", err)
	}
	return connect.NewResponse(&destinationsv1.GetDestinationUsageResponse{
		UsageBytes: report.UsageBytes,
		CapBytes:   report.CapBytes,
		UsageState: usageStateToProto(report.UsageState),
		CapPolicy:  capPolicyToProto(report.CapPolicy),
	}), nil
}

// translate maps a domain error to a Connect error, logging only internal ones.
func (h *connectHandler) translate(op string, err error) error {
	connectErr := destinations.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("destinations.%s: %v", op, err)
	}
	return connectErr
}

// domainToProto converts the internal Destination to its wire shape.
func domainToProto(d destinations.Destination) *destinationsv1.Destination {
	pd := &destinationsv1.Destination{
		Id:                  d.ID,
		Name:                d.Name,
		BackendKind:         backendToProto(d.BackendKind),
		Location:            d.Location,
		CapBytes:            d.CapBytes,
		CapPolicy:           capPolicyToProto(d.CapPolicy),
		EncryptionAlgorithm: d.EncryptionAlgorithm,
		SecretRef:           d.SecretRef,
	}
	if !d.CreatedAt.IsZero() {
		pd.CreatedAt = timestamppb.New(d.CreatedAt)
	}
	if !d.UpdatedAt.IsZero() {
		pd.UpdatedAt = timestamppb.New(d.UpdatedAt)
	}
	return pd
}

// protoToBackend / backendToProto translate the proto BackendKind enum to the
// domain vocabulary so domain code never imports the generated enum.
func protoToBackend(k destinationsv1.BackendKind) destinations.BackendKind {
	switch k {
	case destinationsv1.BackendKind_BACKEND_KIND_FILESYSTEM:
		return destinations.BackendFilesystem
	case destinationsv1.BackendKind_BACKEND_KIND_S3:
		return destinations.BackendS3
	default:
		return ""
	}
}

func backendToProto(k destinations.BackendKind) destinationsv1.BackendKind {
	switch k {
	case destinations.BackendFilesystem:
		return destinationsv1.BackendKind_BACKEND_KIND_FILESYSTEM
	case destinations.BackendS3:
		return destinationsv1.BackendKind_BACKEND_KIND_S3
	default:
		return destinationsv1.BackendKind_BACKEND_KIND_UNSPECIFIED
	}
}

// protoToCapPolicy / capPolicyToProto translate the proto CapPolicy enum.
func protoToCapPolicy(p destinationsv1.CapPolicy) destinations.CapPolicy {
	switch p {
	case destinationsv1.CapPolicy_CAP_POLICY_ALERT_BLOCK:
		return destinations.CapPolicyAlertBlock
	case destinationsv1.CapPolicy_CAP_POLICY_ALERT_ONLY:
		return destinations.CapPolicyAlertOnly
	default:
		return ""
	}
}

func capPolicyToProto(p destinations.CapPolicy) destinationsv1.CapPolicy {
	switch p {
	case destinations.CapPolicyAlertBlock:
		return destinationsv1.CapPolicy_CAP_POLICY_ALERT_BLOCK
	case destinations.CapPolicyAlertOnly:
		return destinationsv1.CapPolicy_CAP_POLICY_ALERT_ONLY
	default:
		return destinationsv1.CapPolicy_CAP_POLICY_UNSPECIFIED
	}
}

// usageStateToProto translates the domain UsageState to the proto enum.
func usageStateToProto(s destinations.UsageState) destinationsv1.UsageState {
	switch s {
	case destinations.UsageStateWithin:
		return destinationsv1.UsageState_USAGE_STATE_WITHIN
	case destinations.UsageStateNear:
		return destinationsv1.UsageState_USAGE_STATE_NEAR
	case destinations.UsageStateOver:
		return destinationsv1.UsageState_USAGE_STATE_OVER
	default:
		return destinationsv1.UsageState_USAGE_STATE_UNSPECIFIED
	}
}
