package registry

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"

	internalregistry "search-hub/internal/registry"
)

// Deps wires the seams the Connect registry handler needs.
type Deps struct {
	Store  internalregistry.Store
	Logger *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the RegistryService handler. Logger defaults to
// log.Default() when nil.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// Compile-time guarantee the handler satisfies the generated interface. This
// catches an RPC added to registry.proto that the handler hasn't implemented.
var _ = func() any {
	type registryServiceHandler interface {
		RegisterProvider(context.Context, *connect.Request[registryv1.RegisterProviderRequest]) (*connect.Response[registryv1.RegisterProviderResponse], error)
		ListProviders(context.Context, *connect.Request[registryv1.ListProvidersRequest]) (*connect.Response[registryv1.ListProvidersResponse], error)
		DeregisterProvider(context.Context, *connect.Request[registryv1.DeregisterProviderRequest]) (*connect.Response[registryv1.DeregisterProviderResponse], error)
	}
	var _ registryServiceHandler = (*connectHandler)(nil)
	return nil
}()

func (h *connectHandler) RegisterProvider(ctx context.Context, req *connect.Request[registryv1.RegisterProviderRequest]) (*connect.Response[registryv1.RegisterProviderResponse], error) {
	desc := req.Msg.GetDescriptor_()
	created, err := h.deps.Store.Upsert(ctx, desc)
	if err != nil {
		connectErr := toConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("registry.RegisterProvider(%q): %v", desc.GetProviderId(), err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&registryv1.RegisterProviderResponse{
		Descriptor_: desc,
		Created:     created,
	}), nil
}

func (h *connectHandler) ListProviders(ctx context.Context, req *connect.Request[registryv1.ListProvidersRequest]) (*connect.Response[registryv1.ListProvidersResponse], error) {
	providers, err := h.deps.Store.List(ctx, internalregistry.ListFilter{
		Bucket: int32(req.Msg.GetBucket()),
		Type:   req.Msg.GetType(),
		State:  int32(req.Msg.GetState()),
	})
	if err != nil {
		h.deps.Logger.Printf("registry.ListProviders: %v", err)
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&registryv1.ListProvidersResponse{Providers: providers}), nil
}

func (h *connectHandler) DeregisterProvider(ctx context.Context, req *connect.Request[registryv1.DeregisterProviderRequest]) (*connect.Response[registryv1.DeregisterProviderResponse], error) {
	removed, err := h.deps.Store.Delete(ctx, req.Msg.GetProviderId())
	if err != nil {
		h.deps.Logger.Printf("registry.DeregisterProvider(%q): %v", req.Msg.GetProviderId(), err)
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&registryv1.DeregisterProviderResponse{Removed: removed}), nil
}

// toConnectError translates registry domain sentinels into Connect codes at the
// transport edge (the domain layer never imports connect). Validation failures
// become InvalidArgument with the "<field>: <reason>" message; a missing
// provider becomes NotFound; everything else is an opaque Internal.
func toConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid internalregistry.ErrInvalidDescriptor
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var notFound internalregistry.ErrProviderNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	return connect.NewError(connect.CodeInternal, errors.New("internal error"))
}
