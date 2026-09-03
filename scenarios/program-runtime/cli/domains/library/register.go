package library

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library/library_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/shared"
)

const GroupName = "library"

type handlers struct {
	client libraryconnect.LibraryServiceClient
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	h := &handlers{client: libraryconnect.NewLibraryServiceClient(httpClient, baseURL)}
	return cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"LibraryService.ListLibrary":       cliapp.ProtoList(h.list, h.listReport),
		"search":                           cliapp.ProtoList(h.list, h.listReport),
		"LibraryService.GetLibrary":        cliapp.ProtoList(h.get, h.getReport),
		"LibraryService.PromoteLibrary":    cliapp.ProtoMutation(h.promote, h.promoteReport),
		"LibraryService.SetCurrentLibrary": cliapp.ProtoMutation(h.setCurrent, h.currentReport),
	})
}

func (h *handlers) list(ctx cliapp.OperationContext) (*libraryv1.ListLibraryResponse, error) {
	query := ""
	if args := ctx.Args(); len(args) > 0 {
		query = strings.TrimSpace(strings.ToLower(args[0]))
	}
	if query == "" {
		request := &libraryv1.ListLibraryRequest{Limit: 50}
		r, err := h.client.ListLibrary(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("list library", err, nil)
		}
		return r.Msg, nil
	}
	request := &libraryv1.ListLibraryRequest{Query: query, Limit: 50}
	r, err := h.client.ListLibrary(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("search library", err, nil)
	}
	type scored struct {
		program *sharedv1.LibraryProgram
		score   int
	}
	scoredPrograms := make([]scored, 0, len(r.Msg.GetPrograms()))
	kindFilter := strings.TrimSpace(strings.ToLower(ctx.Flag("kind")))
	if kindFilter == "" {
		kindFilter = "all"
	}
	for _, p := range r.Msg.GetPrograms() {
		if p == nil {
			continue
		}
		effectiveKind := p.GetKind()
		if effectiveKind == "" {
			effectiveKind = "callable"
		}
		if kindFilter != "all" && effectiveKind != kindFilter {
			continue
		}
		text := strings.ToLower(strings.Join([]string{p.GetName(), p.GetScenario(), p.GetPurpose(), p.GetDescription(), p.GetCoverage(), strings.Join(p.GetCalledBindingIds(), " ")}, " "))
		score := 0
		for _, term := range strings.Fields(query) {
			if strings.Contains(text, term) {
				score++
			}
		}
		if score > 0 {
			scoredPrograms = append(scoredPrograms, scored{program: p, score: score})
		}
	}
	sort.SliceStable(scoredPrograms, func(i, j int) bool {
		if scoredPrograms[i].score != scoredPrograms[j].score {
			return scoredPrograms[i].score > scoredPrograms[j].score
		}
		if scoredPrograms[i].program.GetTier() != scoredPrograms[j].program.GetTier() {
			return scoredPrograms[i].program.GetTier() == "promoted"
		}
		if scoredPrograms[i].program.GetName() == scoredPrograms[j].program.GetName() {
			return scoredPrograms[i].program.GetVersion() > scoredPrograms[j].program.GetVersion()
		}
		return scoredPrograms[i].program.GetName() < scoredPrograms[j].program.GetName()
	})
	programs := make([]*sharedv1.LibraryProgram, 0, len(scoredPrograms))
	for _, item := range scoredPrograms {
		programs = append(programs, item.program)
	}
	limit := 10
	if raw := strings.TrimSpace(ctx.Flag("limit")); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed > 0 {
			limit = parsed
		}
	}
	if len(programs) > limit {
		programs = programs[:limit]
	}
	return &libraryv1.ListLibraryResponse{Programs: programs}, nil
}

func (h *handlers) get(ctx cliapp.OperationContext) (*libraryv1.GetLibraryResponse, error) {
	version := int64(0)
	if value := ctx.Flag("version"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed <= 0 {
			return nil, err
		}
		version = parsed
	}
	r, err := h.client.GetLibrary(context.Background(), connect.NewRequest(&libraryv1.GetLibraryRequest{Name: ctx.Positional("name"), Version: version}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get library", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) promote(ctx cliapp.OperationContext) (*libraryv1.PromoteLibraryResponse, error) {
	r, err := h.client.PromoteLibrary(context.Background(), connect.NewRequest(&libraryv1.PromoteLibraryRequest{ProgramId: ctx.Flag("program-id"), Name: ctx.Flag("name"), Description: ctx.Flag("description"), PromotedBy: ctx.Flag("promoted-by"), Reason: ctx.Flag("reason"), Coverage: ctx.Flag("coverage")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("promote library", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) setCurrent(ctx cliapp.OperationContext) (*libraryv1.SetCurrentLibraryResponse, error) {
	version, err := strconv.ParseInt(ctx.Flag("version"), 10, 64)
	if err != nil || version <= 0 {
		return nil, err
	}
	r, err := h.client.SetCurrentLibrary(context.Background(), connect.NewRequest(&libraryv1.SetCurrentLibraryRequest{Name: ctx.Positional("name"), Version: version}))
	if err != nil {
		return nil, cliapp.WrapAPIError("set current library", err, nil)
	}
	return r.Msg, nil
}

func (*handlers) listReport(_ cliapp.OperationContext, r *libraryv1.ListLibraryResponse) cliapp.ListReport {
	programs := r.GetPrograms()
	results := make([]string, 0, len(programs))
	for _, p := range programs {
		if p == nil {
			continue
		}
		marker := ""
		if p.GetCurrent() {
			marker = " [current]"
		}
		kind := p.GetKind()
		if kind == "" {
			kind = "callable"
		}
		line := fmt.Sprintf("%s [%s]%s — %s", p.GetName(), kind, marker, p.GetDescription())
		if p.GetScenario() != "" {
			line += "\n  scenario: " + p.GetScenario()
		}
		if p.GetPurpose() != "" {
			line += "\n  purpose: " + p.GetPurpose()
		}
		if len(p.GetCalledBindingIds()) > 0 {
			line += "\n  bindings: " + strings.Join(p.GetCalledBindingIds(), ", ")
		}
		if kind == "contract" {
			line += "\n  run: program-runtime programs submit --source <program.py>"
		} else {
			line += "\n  call: lib." + p.GetName() + "()"
		}
		results = append(results, line)
	}
	return cliapp.ListReport{
		Summary:        []string{"Library programs: " + strconv.Itoa(len(programs))},
		ResultsHeading: "Programs",
		Results:        results,
		ListShaped:     true,
		ResultCount:    len(programs),
	}
}

func (*handlers) getReport(_ cliapp.OperationContext, r *libraryv1.GetLibraryResponse) cliapp.ListReport {
	program := r.GetProgram()
	if program == nil {
		return cliapp.ListReport{Summary: []string{"Library program read."}}
	}
	results := []string{
		fmt.Sprintf("Version: %d", program.GetVersion()),
		"Description: " + program.GetDescription(),
		"Origin: " + program.GetOrigin(),
		"Tier: " + program.GetTier(),
		"Validated at: " + program.GetValidatedAt(),
		"Declared inputs: " + strings.Join(program.GetDeclaredInputs(), ", "),
		"Declared outputs: " + strings.Join(program.GetDeclaredOutputs(), ", "),
		"Coverage: " + program.GetCoverage(),
		"Called bindings: " + strings.Join(program.GetCalledBindingIds(), ", "),
	}
	for _, drift := range r.GetDrift() {
		results = append(results, fmt.Sprintf("Drift %s: changed=%t status=%s generation=%s — %s", drift.GetBindingId(), drift.GetChanged(), drift.GetDriftStatus(), drift.GetGenerationMtime(), drift.GetReason()))
	}
	results = append(results, "Source:\n"+program.GetSource())
	return cliapp.ListReport{
		Summary:        []string{"Library program: " + program.GetName()},
		ResultsHeading: "Program",
		Results:        results,
		ListShaped:     true,
		ResultCount:    1,
	}
}

func (*handlers) promoteReport(_ cliapp.OperationContext, _ *libraryv1.PromoteLibraryResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Library program promoted."}}
}

func (*handlers) currentReport(_ cliapp.OperationContext, _ *libraryv1.SetCurrentLibraryResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Library version selected for new sessions."}}
}
