package registry

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	"search-hub/internal/control"

	internalregistry "search-hub/internal/registry"
)

// Deps wires the seams the Connect registry handler needs.
type Deps struct {
	Store    internalregistry.Store
	RepoRoot string
	Logger   *log.Logger
	Control  *control.Client
	Probe    EndpointProber
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
		ExecuteEmbeddingMigration(context.Context, *connect.Request[registryv1.ExecuteEmbeddingMigrationRequest]) (*connect.Response[registryv1.ExecuteEmbeddingMigrationResponse], error)
		ListMaturityTargets(context.Context, *connect.Request[registryv1.ListMaturityTargetsRequest]) (*connect.Response[registryv1.ListMaturityTargetsResponse], error)
		DeregisterProvider(context.Context, *connect.Request[registryv1.DeregisterProviderRequest]) (*connect.Response[registryv1.DeregisterProviderResponse], error)
	}
	var _ registryServiceHandler = (*connectHandler)(nil)
	return nil
}()

func (h *connectHandler) RegisterProvider(ctx context.Context, req *connect.Request[registryv1.RegisterProviderRequest]) (*connect.Response[registryv1.RegisterProviderResponse], error) {
	desc := req.Msg.GetDescriptor_()
	var prior []*registryv1.ProviderDescriptor
	if existing, listErr := h.deps.Store.List(ctx, internalregistry.ListFilter{State: int32(registryv1.ProviderState_PROVIDER_STATE_ACTIVE)}); listErr == nil {
		prior = existing
	}
	if h.deps.Probe != nil {
		if err := h.deps.Probe.Probe(ctx, desc); err != nil {
			h.deps.Logger.Printf("registry.RegisterProvider(%q): endpoint probe failed: %v", desc.GetProviderId(), err)
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
	}
	created, token, err := h.deps.Store.Upsert(ctx, desc, req.Msg.GetControlToken())
	if err != nil {
		connectErr := toConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("registry.RegisterProvider(%q): %v", desc.GetProviderId(), err)
		}
		return nil, connectErr
	}
	for _, warning := range internalregistry.SimilarDescriptorWarnings(desc, prior) {
		h.deps.Logger.Printf("registry.RegisterProvider(%q): descriptor is similar to %q (Jaccard %.2f); automatic routing may need more discriminative description text", desc.GetProviderId(), warning.ExistingProviderID, warning.Similarity)
	}
	return connect.NewResponse(&registryv1.RegisterProviderResponse{
		Descriptor_:  desc,
		Created:      created,
		ControlToken: token,
	}), nil
}

func (h *connectHandler) ExecuteEmbeddingMigration(ctx context.Context, req *connect.Request[registryv1.ExecuteEmbeddingMigrationRequest]) (*connect.Response[registryv1.ExecuteEmbeddingMigrationResponse], error) {
	if h.deps.Control == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("provider control client is not configured"))
	}
	r := req.Msg
	desc, err := h.deps.Store.Get(ctx, r.GetProviderId())
	if err != nil {
		return nil, toConnectError(err)
	}
	token, err := h.deps.Store.Token(ctx, r.GetProviderId())
	if err != nil {
		return nil, toConnectError(err)
	}
	if r.GetAction() == "status" {
		status, statusErr := h.deps.Control.ReindexStatus(ctx, desc, token, r.GetJobId())
		if statusErr != nil {
			return nil, connect.NewError(connect.CodeInternal, statusErr)
		}
		return connect.NewResponse(&registryv1.ExecuteEmbeddingMigrationResponse{
			JobId: status.GetJobId(), State: status.GetState(), Processed: status.GetProcessed(), Total: status.GetTotal(), Error: status.GetError(),
		}), nil
	}
	response, err := h.deps.Control.ReindexRequest(ctx, desc, token, &controlv1.ReindexRequest{
		Action: r.GetAction(), ShadowCollection: r.GetShadowCollection(), RollbackCollection: r.GetRollbackCollection(),
		EmbeddingModel: r.GetEmbeddingModel(), EmbeddingRole: r.GetEmbeddingRole(), EmbeddingDimensions: r.GetEmbeddingDimensions(),
		EmbeddingPolicySchemaVersion: r.GetEmbeddingPolicySchemaVersion(), Scope: r.GetScope(), DryRun: r.GetDryRun(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&registryv1.ExecuteEmbeddingMigrationResponse{
		JobId: response.GetJobId(), PlannedUpserts: response.GetPlannedUpserts(), PlannedDeletes: response.GetPlannedDeletes(),
		State: "accepted",
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

func (h *connectHandler) ListMaturityTargets(ctx context.Context, _ *connect.Request[registryv1.ListMaturityTargetsRequest]) (*connect.Response[registryv1.ListMaturityTargetsResponse], error) {
	targets, err := internalregistry.DiscoverMaturityTargets(h.deps.RepoRoot)
	if err != nil {
		h.deps.Logger.Printf("registry.ListMaturityTargets: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("maturity target discovery failed"))
	}
	providers, listErr := h.deps.Store.List(ctx, internalregistry.ListFilter{})
	if listErr != nil {
		// Descriptor/capability discovery remains useful when the registry is
		// temporarily unavailable; the scan's primary source is the repository.
		h.deps.Logger.Printf("registry.ListMaturityTargets: registered provider union unavailable: %v", listErr)
	} else {
		groups := make([]string, 0, len(providers))
		for _, provider := range providers {
			if provider != nil {
				groups = append(groups, provider.GetProviderGroup())
			}
		}
		targets = internalregistry.MergeRegisteredMaturityTargets(targets, groups, h.deps.RepoRoot)
	}
	out := make([]*registryv1.MaturityTarget, 0, len(targets))
	for _, target := range targets {
		out = append(out, &registryv1.MaturityTarget{
			Scenario:            target.Scenario,
			Path:                target.Path,
			ApplicabilityReason: target.ApplicabilityReason,
		})
	}
	return connect.NewResponse(&registryv1.ListMaturityTargetsResponse{Targets: out}), nil
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
	var tokenMismatch internalregistry.ErrTokenMismatch
	if errors.As(err, &tokenMismatch) {
		return connect.NewError(connect.CodePermissionDenied, tokenMismatch)
	}
	return connect.NewError(connect.CodeInternal, errors.New("internal error"))
}
