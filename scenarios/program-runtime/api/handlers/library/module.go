package library

import (
	"context"
	"strings"
	"time"

	programbindings "program-runtime/internal/bindings"
	"program-runtime/internal/contracts"
	"program-runtime/internal/library"
	"program-runtime/internal/module"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library/library_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/shared"
)

type handler struct {
	libraryconnect.UnimplementedLibraryServiceHandler
	repo      *library.Repository
	bindings  *programbindings.Registry
	contracts *contracts.Index
}

func Module(repo *library.Repository, registry *programbindings.Registry, indexes ...*contracts.Index) module.Module {
	var index *contracts.Index
	if len(indexes) > 0 {
		index = indexes[0]
	}
	return module.Module{Name: "library", Mount: func(r *mux.Router) {
		path, h := libraryconnect.NewLibraryServiceHandler(&handler{repo: repo, bindings: registry, contracts: index})
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func (h *handler) ListLibrary(ctx context.Context, req *connect.Request[libraryv1.ListLibraryRequest]) (*connect.Response[libraryv1.ListLibraryResponse], error) {
	limit, offset := int(req.Msg.GetLimit()), int(req.Msg.GetOffset())
	if limit <= 0 {
		limit = 50
	}
	var programs []*sharedv1.LibraryProgram
	var err error
	if strings.TrimSpace(req.Msg.GetQuery()) != "" {
		programs, err = h.repo.ListCallable(ctx)
	} else {
		programs, err = h.repo.List(ctx, limit, offset)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if h.contracts != nil && strings.TrimSpace(req.Msg.GetQuery()) != "" {
		for _, contract := range h.contracts.List() {
			programs = append(programs, &sharedv1.LibraryProgram{Name: contract.ID, Id: contract.ID, Description: contract.Purpose, Scenario: contract.Scenario, Purpose: contract.Purpose, Kind: "contract", Rung: contract.Rung, OwnerSkill: contract.OwnerSkill, ValidationError: contract.ValidationError, Path: contract.SourcePath, CalledBindingIds: contract.BindingIDs})
		}
	}
	return connect.NewResponse(&libraryv1.ListLibraryResponse{Programs: programs}), nil
}

func (h *handler) GetLibrary(ctx context.Context, req *connect.Request[libraryv1.GetLibraryRequest]) (*connect.Response[libraryv1.GetLibraryResponse], error) {
	program, err := h.repo.Get(ctx, strings.TrimSpace(req.Msg.GetName()), req.Msg.GetVersion())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	response := &libraryv1.GetLibraryResponse{Program: program}
	if h.bindings != nil {
		for _, bindingID := range program.GetCalledBindingIds() {
			drift := &libraryv1.BindingDrift{BindingId: bindingID, ValidatedAt: program.GetValidatedAt()}
			conditions, conditionErr := h.bindings.Conditions(ctx, bindingID, "", 24*time.Hour)
			if conditionErr != nil {
				drift.DriftStatus = "unavailable"
				drift.Reason = conditionErr.Error()
				response.Drift = append(response.Drift, drift)
				continue
			}
			for _, condition := range conditions.GetConditions() {
				if condition.GetBindingId() != bindingID {
					continue
				}
				drift.DriftStatus = condition.GetFreshness().GetDriftStatus().String()
				drift.GenerationMtime = condition.GetFreshness().GetGenerationMtime()
				drift.Reason = condition.GetFreshness().GetDriftReason()
				generation, parseErr := time.Parse(time.RFC3339Nano, drift.GenerationMtime)
				validated, validatedErr := time.Parse(time.RFC3339Nano, program.GetValidatedAt())
				drift.Changed = parseErr == nil && validatedErr == nil && generation.After(validated)
				if drift.Changed {
					drift.Reason = "binding generation is newer than program validation"
				}
				break
			}
			response.Drift = append(response.Drift, drift)
		}
	}
	return connect.NewResponse(response), nil
}

func (h *handler) PromoteLibrary(ctx context.Context, req *connect.Request[libraryv1.PromoteLibraryRequest]) (*connect.Response[libraryv1.PromoteLibraryResponse], error) {
	program, err := h.repo.PromoteByID(ctx, req.Msg.GetProgramId(), req.Msg.GetName(), req.Msg.GetDescription(), req.Msg.GetPromotedBy(), req.Msg.GetReason(), req.Msg.GetCoverage(), req.Msg.GetDeclaredInputs(), req.Msg.GetDeclaredOutputs(), time.Now().UTC())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&libraryv1.PromoteLibraryResponse{Program: program}), nil
}

func (h *handler) SetCurrentLibrary(ctx context.Context, req *connect.Request[libraryv1.SetCurrentLibraryRequest]) (*connect.Response[libraryv1.SetCurrentLibraryResponse], error) {
	program, err := h.repo.SetCurrent(ctx, req.Msg.GetName(), req.Msg.GetVersion())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&libraryv1.SetCurrentLibraryResponse{Program: program}), nil
}
