package library

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library/library_v1connect"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/shared"
	"google.golang.org/protobuf/types/known/structpb"
)

const GroupName = "library"

type handlers struct {
	client libraryconnect.LibraryServiceClient
}

func parseInputPairs(raw string) (map[string]any, error) {
	inputs := map[string]any{}
	for _, item := range splitInputPairs(raw) {
		if strings.TrimSpace(item) == "" {
			continue
		}
		pair := strings.SplitN(item, "=", 2)
		if len(pair) != 2 || strings.TrimSpace(pair[0]) == "" {
			return nil, fmt.Errorf("input must be key=value")
		}
		var value any = pair[1]
		if json.Unmarshal([]byte(pair[1]), &value) != nil {
			value = pair[1]
		}
		inputs[strings.TrimSpace(pair[0])] = value
	}
	return inputs, nil
}

// splitInputPairs keeps commas inside JSON objects, arrays, and quoted
// strings in one value. Structured contract inputs must remain one CLI value.
func splitInputPairs(raw string) []string {
	var out []string
	start, depth := 0, 0
	var quote rune
	escaped := false
	for i, r := range raw {
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
			}
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case '{', '[':
			depth++
		case '}', ']':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				out = append(out, raw[start:i])
				start = i + 1
			}
		}
	}
	return append(out, raw[start:])
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	h := &handlers{client: libraryconnect.NewLibraryServiceClient(httpClient, baseURL)}
	return cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"LibraryService.ListLibrary":        cliapp.ProtoList(h.list, h.listReport),
		"search":                            cliapp.ProtoList(h.list, h.listReport),
		"LibraryService.GetLibrary":         cliapp.ProtoList(h.get, h.getReport),
		"LibraryService.PromoteLibrary":     cliapp.ProtoMutation(h.promote, h.promoteReport),
		"LibraryService.SetCurrentLibrary":  cliapp.ProtoMutation(h.setCurrent, h.currentReport),
		"LibraryService.RunDeclaredProgram": cliapp.ProtoMutationOutcome(h.run, h.runReport, h.runOutcome),
	})
}

var libraryRunStatusPattern = regexp.MustCompile(`'status'\s*:\s*'([^']+)'`)

func libraryRunStatus(stdout string) string {
	var envelope struct {
		Status string `json:"status"`
	}
	if json.Unmarshal([]byte(strings.TrimSpace(stdout)), &envelope) == nil && envelope.Status != "" {
		return envelope.Status
	}
	match := libraryRunStatusPattern.FindStringSubmatch(stdout)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func (h *handlers) run(ctx cliapp.OperationContext) (*libraryv1.RunDeclaredProgramResponse, error) {
	name := ctx.Positional("name")
	parts := strings.SplitN(name, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("name must be <scenario>.<program>")
	}
	inputs, err := parseInputPairs(ctx.Flag("input"))
	if err != nil {
		return nil, err
	}
	structured, err := structpb.NewStruct(inputs)
	if err != nil {
		return nil, fmt.Errorf("encode library run inputs: %w", err)
	}
	result, err := h.client.RunDeclaredProgram(context.Background(), connect.NewRequest(&libraryv1.RunDeclaredProgramRequest{
		Name: name, Inputs: structured, Provenance: programsv1.Provenance_PROVENANCE_OPERATOR,
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("run declared library program", err, nil)
	}
	return result.Msg, nil
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

func (*handlers) runReport(_ cliapp.OperationContext, r *libraryv1.RunDeclaredProgramResponse) cliapp.MutationReport {
	if r.GetProgram() == nil {
		return cliapp.MutationReport{Result: []string{"Library program produced no result."}}
	}
	return cliapp.MutationReport{Result: []string{r.GetProgram().GetStdout()}}
}

func (*handlers) runOutcome(_ cliapp.OperationContext, r *libraryv1.RunDeclaredProgramResponse) error {
	program := r.GetProgram()
	if program == nil {
		return fmt.Errorf("library run produced no program result")
	}
	if program.GetStatus() != programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED {
		return fmt.Errorf("library run completed with program status %s", program.GetStatus().String())
	}
	if !r.GetTerminal() {
		return fmt.Errorf("library run did not reach a terminal state after %d ms", r.GetWaitedMillis())
	}
	switch status := libraryRunStatus(program.GetStdout()); status {
	case "ok", "partial":
		return nil
	case "failed", "unavailable", "refused":
		return fmt.Errorf("library run envelope status %q", status)
	default:
		return fmt.Errorf("library run returned no recognized envelope status")
	}
}
