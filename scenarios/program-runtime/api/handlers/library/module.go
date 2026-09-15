package library

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	programbindings "program-runtime/internal/bindings"
	"program-runtime/internal/contracts"
	"program-runtime/internal/library"
	"program-runtime/internal/module"
	internalprograms "program-runtime/internal/programs"
	internalsessions "program-runtime/internal/sessions"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library/library_v1connect"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/shared"
	"google.golang.org/protobuf/proto"
)

type handler struct {
	libraryconnect.UnimplementedLibraryServiceHandler
	repo      *library.Repository
	bindings  *programbindings.Registry
	contracts *contracts.Index
	repoRoot  string
	sessions  *internalsessions.Manager
	programs  *internalprograms.Service
}

type RunDependencies struct {
	Repository *library.Repository
	RepoRoot   string
	Sessions   *internalsessions.Manager
	Programs   *internalprograms.Service
}

func Module(repo *library.Repository, registry *programbindings.Registry, indexes ...*contracts.Index) module.Module {
	var index *contracts.Index
	var runDeps RunDependencies
	if len(indexes) > 0 {
		index = indexes[0]
	}
	// Preserve the compact constructor used by tests while allowing production
	// to install the server-owned declared-program runner through ModuleWithRun.
	return module.Module{Name: "library", Mount: func(r *mux.Router) {
		path, h := libraryconnect.NewLibraryServiceHandler(&handler{repo: repo, bindings: registry, contracts: index, repoRoot: runDeps.RepoRoot, sessions: runDeps.Sessions, programs: runDeps.Programs})
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

// DeclaredRunner shares execution, content pinning and session reclamation with
// library callers; validation must not implement a second kernel runner.
func DeclaredRunner(registry *programbindings.Registry, index *contracts.Index, deps RunDependencies) libraryconnect.LibraryServiceHandler {
	return &handler{repo: deps.Repository, bindings: registry, contracts: index, repoRoot: deps.RepoRoot, sessions: deps.Sessions, programs: deps.Programs}
}

func ModuleWithRun(repo *library.Repository, registry *programbindings.Registry, index *contracts.Index, deps RunDependencies) module.Module {
	return module.Module{Name: "library", Mount: func(r *mux.Router) {
		path, h := libraryconnect.NewLibraryServiceHandler(&handler{repo: repo, bindings: registry, contracts: index, repoRoot: deps.RepoRoot, sessions: deps.Sessions, programs: deps.Programs})
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
	if h.contracts != nil {
		for _, contract := range h.contracts.List() {
			programs = append(programs, contractProgram(contract))
		}
	}
	return connect.NewResponse(&libraryv1.ListLibraryResponse{Programs: programs}), nil
}

func (h *handler) GetLibrary(ctx context.Context, req *connect.Request[libraryv1.GetLibraryRequest]) (*connect.Response[libraryv1.GetLibraryResponse], error) {
	if h.contracts != nil {
		if h.repoRoot != "" {
			if _, err := h.contracts.Refresh(h.repoRoot); err != nil {
				return nil, connect.NewError(connect.CodeUnavailable, err)
			}
		}
		parts := strings.SplitN(strings.TrimSpace(req.Msg.GetName()), ".", 2)
		if len(parts) == 2 {
			if contract, ok := h.contracts.Get(parts[0], parts[1]); ok {
				if requested := req.Msg.GetVersion(); requested != 0 {
					version, parseErr := strconv.ParseInt(contract.Version, 10, 64)
					if parseErr != nil || version != requested {
						return nil, connect.NewError(connect.CodeNotFound, library.ErrNotFound)
					}
				}
				if h.repo != nil && contract.ValidationError == "" {
					if err := h.repo.RetainDeclared(ctx, contract); err != nil {
						return nil, connect.NewError(connect.CodeInternal, err)
					}
				}
				return connect.NewResponse(&libraryv1.GetLibraryResponse{Program: contractProgram(contract)}), nil
			}
		}
	}
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

func contractProgram(contract contracts.Contract) *sharedv1.LibraryProgram {
	version, _ := strconv.ParseInt(contract.Version, 10, 64)
	liveFixtures := int32(0)
	for _, fixture := range contract.Fixtures {
		if len(fixture.Requires) > 0 {
			liveFixtures++
		}
	}
	optionalBindings := int32(0)
	for _, binding := range contract.Bindings {
		if binding.Optional {
			optionalBindings++
		}
	}
	return &sharedv1.LibraryProgram{
		Name:                             contract.ID,
		ContentDigest:                    contract.Digest,
		Id:                               contract.ID,
		Version:                          version,
		Source:                           contract.Source,
		Description:                      contract.Purpose,
		Scenario:                         contract.Scenario,
		Purpose:                          contract.Purpose,
		Kind:                             "contract",
		Rung:                             contract.Rung,
		OwnerSkill:                       contract.OwnerSkill,
		ValidationError:                  contract.ValidationError,
		Path:                             contract.SourcePath,
		CalledBindingIds:                 contract.BindingIDs,
		DeclaredInputs:                   contract.InputNames,
		Verbs:                            contract.Verbs,
		MemoryDeclared:                   contract.Memory != nil,
		FixtureCount:                     int32(len(contract.Fixtures)),
		LiveFixtureCount:                 liveFixtures,
		BindingCount:                     int32(len(contract.Bindings)),
		OptionalBindingCount:             optionalBindings,
		SourceMissing:                    contract.SourceMissing,
		OutputSchemaPresent:              contract.OutputSchemaPath != "",
		LearningVerbs:                    contract.Learning.Verbs,
		LearningUsesMemory:               contract.Learning.UsesMemory,
		LearningNoteKinds:                contract.Learning.NoteKinds,
		LearningFreeTextInputsWithoutKey: contract.Learning.FreeTextInputsWithoutKey,
	}
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

func (h *handler) RunDeclaredProgram(ctx context.Context, req *connect.Request[libraryv1.RunDeclaredProgramRequest]) (*connect.Response[libraryv1.RunDeclaredProgramResponse], error) {
	if h.contracts == nil || h.sessions == nil || h.programs == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("declared program runner is unavailable"))
	}
	if h.repoRoot != "" {
		if _, err := h.contracts.Refresh(h.repoRoot); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("refresh declared programs: %w", err))
		}
	}
	parts := strings.SplitN(strings.TrimSpace(req.Msg.GetName()), ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name must be <scenario>.<program>"))
	}
	if req.Msg.GetProvenance() == programsv1.Provenance_PROVENANCE_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("provenance is required"))
	}
	contract, ok := h.contracts.Get(parts[0], parts[1])
	if expected := req.Msg.GetExpectedDigest(); expected != "" && (!ok || expected != contract.Digest) {
		if h.repo == nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("pinned program artifact unavailable"))
		}
		archived, err := h.repo.GetDeclaredArtifact(ctx, req.Msg.GetName(), expected)
		if err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("pinned program artifact unavailable: %w", err))
		}
		contract, ok = archived, true
	}
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("declared program not found"))
	}
	if contract.ValidationError != "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("declared contract invalid: %s", contract.ValidationError))
	}
	if h.repo != nil {
		if err := h.repo.RetainDeclared(ctx, contract); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	provided := map[string]any{}
	if req.Msg.GetInputs() != nil {
		provided = req.Msg.GetInputs().AsMap()
	}
	resolved, err := contract.ResolveInputs(provided)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	encoded, err := json.Marshal(resolved)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("encode declared program inputs: %w", err))
	}
	source := "import json\ninputs = json.loads(" + strconv.Quote(string(encoded)) + ")\n# declared-program generated input preamble\n" + contract.Source
	if contract.LearningTask != nil {
		source = "import json\nprint(json.dumps(tasks.run(operation=" + strconv.Quote(contract.ID) + ", inputs=json.loads(" + strconv.Quote(string(encoded)) + "), expected_digest=" + strconv.Quote(contract.Digest) + ").head(1)[0]))"
	} else if len(contract.Learning.Verbs) > 0 && !containsLearningVerb(contract.Learning.Verbs, "task") {
		source = "learn.task(operation=" + strconv.Quote(contract.ID) + ")\n" + source
	}
	session, err := h.sessions.CreateWithExecutionBudgets(ctx, "declared-program:"+contract.ID, "", nil, 0, 0, contract.WallMS, 0)
	if err != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("create declared program session: %w", err))
	}
	caller := internalprograms.Caller{}
	if value := req.Msg.GetCaller(); value != nil {
		caller = internalprograms.Caller{RunID: value.GetRunId(), AgentProfile: value.GetAgentProfile(), SkillID: value.GetSkillId(), Harness: value.GetHarness()}
	}
	program, _, err := h.programs.SubmitDeclared(ctx, session.ID, source, req.Msg.GetProvenance(), contract.OutputBytes == 65536, false, internalprograms.Identity{ProgramName: contract.ID, ProgramDigest: contract.Digest}, caller, true)
	if err != nil {
		_, _ = h.sessions.Delete(context.Background(), session.ID, "declared program submission failed")
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("submit declared program: %w", err))
	}
	wait := time.Duration(contract.WallMS) * time.Millisecond
	if wait <= 0 {
		wait = 60 * time.Second
	}
	// Cleanup belongs to execution, never the observer's HTTP connection. A
	// disconnected wait must not kill a kernel that may be performing effects.
	if internalprograms.IsTerminal(program.GetStatus()) {
		_, _ = h.sessions.Delete(context.Background(), session.ID, "declared program complete")
	} else {
		go func(id string) {
			_, terminal, waitErr := h.programs.Wait(context.Background(), id, wait+30*time.Second)
			if waitErr == nil && terminal {
				_, _ = h.sessions.Delete(context.Background(), session.ID, "declared program complete")
			}
		}(program.GetId())
	}
	if req.Msg.GetAsync() {
		return declaredProgramResponse(program, internalprograms.IsTerminal(program.GetStatus()), 0), nil
	}
	waitStarted := time.Now()
	acceptedID := program.GetId()
	if !internalprograms.IsTerminal(program.GetStatus()) {
		program, ok, err = h.programs.Wait(ctx, program.GetId(), wait)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("wait interrupted; inspect program %s with programs get before retrying: %w", acceptedID, err))
		}
		if ok {
			_, _ = h.sessions.Delete(context.Background(), session.ID, "declared program complete")
		}
		return declaredProgramResponse(program, ok, time.Since(waitStarted)), nil
	}
	return declaredProgramResponse(program, true, time.Since(waitStarted)), nil
}

func containsLearningVerb(verbs []string, wanted string) bool {
	for _, verb := range verbs {
		if verb == wanted {
			return true
		}
	}
	return false
}

func declaredProgramResponse(program *programsv1.Program, terminal bool, waited time.Duration) *connect.Response[libraryv1.RunDeclaredProgramResponse] {
	summary := proto.Clone(program).(*programsv1.Program)
	// The executable source can be large and can contain caller-supplied values.
	// RunDeclaredProgram returns execution evidence, not the implementation body.
	summary.Source = ""
	return connect.NewResponse(&libraryv1.RunDeclaredProgramResponse{Program: summary, Terminal: terminal, WaitedMillis: waited.Milliseconds()})
}
