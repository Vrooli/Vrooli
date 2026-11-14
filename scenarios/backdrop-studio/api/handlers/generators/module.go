package generators

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/module"
	"backdrop-studio/internal/vector/authoring"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	generatorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/generators"
	generatorsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/generators/generators_v1connect"
)

// Module wires the authoring surface.
//
// The ai-gateway client is constructed here and resolves lazily, so a host with
// no gateway is a named missing capability at the moment someone authors rather
// than a startup failure for everyone who never does.
func Module(db *sql.DB) module.Module {
	h := &handler{store: catalog.NewStore(db), client: authoring.NewGatewayClient()}
	return module.Module{Name: "generators", Mount: func(r *mux.Router) {
		path, svc := generatorsconnect.NewGeneratorsServiceHandler(h)
		r.PathPrefix(path).Handler(svc)
	}, Endpoints: Endpoints}
}

type handler struct {
	store  *catalog.Store
	client authoring.Client
}

func (h *handler) AuthorGenerator(ctx context.Context, req *connect.Request[generatorsv1.AuthorGeneratorRequest]) (*connect.Response[generatorsv1.AuthorGeneratorResponse], error) {
	generator, report, err := authoring.Author(ctx, h.client, authoring.Request{
		ID:    req.Msg.GetId(),
		Brief: req.Msg.GetBrief(),
	})
	if err != nil {
		// Reaching the model failed, or its answer was unreadable. Neither is a
		// bad request from the caller and neither is fixed by retrying the same
		// call, so it is a precondition rather than an argument error.
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	// A refused generator is returned with its verdict rather than turned into
	// an error. The refusals are the actionable part — an operator reads them
	// and re-asks with a sharper brief — and an error body would carry the
	// message while dropping the generator that produced it.
	stored := false
	if report.Passed && req.Msg.GetStore() {
		if err := h.store.PutAuthoredGenerator(ctx, generator); err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		stored = true
	}
	return connect.NewResponse(&generatorsv1.AuthorGeneratorResponse{
		Generator: generatorProto(generator),
		Stored:    stored,
	}), nil
}

func (h *handler) ListGenerators(ctx context.Context, _ *connect.Request[generatorsv1.ListGeneratorsRequest]) (*connect.Response[generatorsv1.ListGeneratorsResponse], error) {
	stored, err := h.store.ListAuthoredGenerators(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &generatorsv1.ListGeneratorsResponse{}
	for _, g := range stored {
		resp.Generators = append(resp.Generators, generatorProto(g))
	}
	return connect.NewResponse(resp), nil
}

func (h *handler) DeleteGenerator(ctx context.Context, req *connect.Request[generatorsv1.DeleteGeneratorRequest]) (*connect.Response[generatorsv1.DeleteGeneratorResponse], error) {
	if _, err := h.store.AuthoredGenerator(ctx, req.Msg.GetId()); err != nil {
		var unknown *catalog.UnknownGeneratorError
		if errors.As(err, &unknown) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := h.store.DeleteAuthoredGenerator(ctx, req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&generatorsv1.DeleteGeneratorResponse{Deleted: true}), nil
}

func generatorProto(g authoring.Generator) *generatorsv1.Generator {
	out := &generatorsv1.Generator{
		Id: g.ID, Name: g.Name, Template: g.Template,
		Inks:   append([]string(nil), g.Inks...),
		Prompt: g.Prompt, ModelId: g.ModelID,
		Validation: &generatorsv1.ValidationReport{
			Passed:   g.Validation.Passed,
			Refusals: append([]string(nil), g.Validation.Refusals...),
		},
	}
	for _, p := range g.Params {
		out.Params = append(out.Params, &generatorsv1.ParamSpec{
			Name: p.Name, Min: p.Min, Max: p.Max, Default: p.Default, Description: p.Description,
		})
	}
	for _, c := range g.Validation.Checks {
		out.Validation.Checks = append(out.Validation.Checks, &generatorsv1.Check{
			Name: c.Name, Passed: c.Passed, Detail: c.Detail,
		})
	}
	return out
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "generators_author", Path: generatorsconnect.GeneratorsServiceAuthorGeneratorProcedure, Method: http.MethodPost, Summary: "Author a vector generator through a model", Category: "generators"},
	{ID: "generators_list", Path: generatorsconnect.GeneratorsServiceListGeneratorsProcedure, Method: http.MethodPost, Summary: "List stored authored generators", Category: "generators"},
	{ID: "generators_delete", Path: generatorsconnect.GeneratorsServiceDeleteGeneratorProcedure, Method: http.MethodPost, Summary: "Delete an authored generator", Category: "generators"},
}
