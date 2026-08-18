package programs

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	programsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs/programs_v1connect"
)

const GroupName = "programs"

type handlers struct {
	client programsconnect.ProgramServiceClient
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	h := &handlers{client: programsconnect.NewProgramServiceClient(httpClient, baseURL)}
	return cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{"ProgramService.SubmitProgram": cliapp.ProtoMutation(h.submit, h.submitReport), "ProgramService.GetProgram": cliapp.ProtoList(h.get, h.programReport), "ProgramService.WaitForProgram": cliapp.ProtoList(h.waitCommand, h.waitReport), "ProgramService.ListPrograms": cliapp.ProtoList(h.list, h.listReport), "vrooli.program_runtime.v1.programs.ProgramService.MineFailures": cliapp.ProtoList(h.mine, h.failureReport), "vrooli.program_runtime.v1.programs.ProgramService.MineRefusals": cliapp.ProtoList(h.mineRefusals, h.refusalReport), "vrooli.program_runtime.v1.programs.ProgramService.MineUnresolvedBindings": cliapp.ProtoList(h.mineUnresolved, h.unresolvedReport)})
}

func (h *handlers) submit(ctx cliapp.OperationContext) (*programsv1.SubmitProgramResponse, error) {
	provenance, err := parseProvenance(ctx.Flag("provenance"))
	if err != nil {
		return nil, err
	}
	source, err := programSource(ctx.Flag("source"), ctx.Flag("source-file"))
	if err != nil {
		return nil, err
	}
	async := ctx.BoolFlag("async")
	waitTimeout, err := parseWaitTimeout(ctx.Flag("wait-timeout"))
	if err != nil {
		return nil, err
	}
	if waitTimeout > 0 {
		async = true
	}
	r, e := h.client.SubmitProgram(context.Background(), connect.NewRequest(&programsv1.SubmitProgramRequest{SessionId: ctx.Flag("session-id"), Source: source, Provenance: provenance, IncludeMaterialized: ctx.BoolFlag("include-materialized"), Explain: ctx.BoolFlag("explain"), Async: async}))
	if e != nil {
		return nil, cliapp.WrapAPIError("submit program", e, nil)
	}
	if waitTimeout > 0 && r.Msg.GetProgram() != nil {
		return h.wait(context.Background(), r.Msg.GetProgram().GetId(), waitTimeout)
	}
	return r.Msg, nil
}

// wait blocks once on the server-side wait RPC.
//
// This used to be a client-side loop calling GetProgram every 50ms — twenty
// requests a second for the whole life of the program — which contradicted the
// project rule that a caller blocks once and never polls, and which no
// non-CLI consumer could reuse. The runtime clamps the deadline, so a request
// longer than the ceiling returns the current non-terminal program and the
// caller resumes explicitly rather than spinning.
func (h *handlers) wait(parent context.Context, id string, timeout time.Duration) (*programsv1.SubmitProgramResponse, error) {
	// The transport deadline exceeds the requested wait so the server, not the
	// client, is the party that decides the wait is over.
	ctx, cancel := context.WithTimeout(parent, timeout+waitTransportMargin)
	defer cancel()
	result, err := h.client.WaitForProgram(ctx, connect.NewRequest(&programsv1.WaitForProgramRequest{
		Id:            id,
		TimeoutMillis: timeout.Milliseconds(),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("wait for program", err, nil)
	}
	return &programsv1.SubmitProgramResponse{Program: result.Msg.GetProgram()}, nil
}

// waitTransportMargin is the slack between the requested server-side wait and
// the client's transport deadline, so a wait that returns exactly at its
// deadline is never reported as a client-side timeout.
const waitTransportMargin = 15 * time.Second

func parseWaitTimeout(value string) (time.Duration, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	duration, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("wait-timeout must be a positive duration such as 30s")
	}
	return duration, nil
}

func programSource(source, sourceFile string) (string, error) {
	if strings.TrimSpace(source) != "" && strings.TrimSpace(sourceFile) != "" {
		return "", fmt.Errorf("source and source-file are mutually exclusive")
	}
	if strings.TrimSpace(sourceFile) == "" {
		return source, nil
	}
	if sourceFile == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read program source from stdin: %w", err)
		}
		return string(data), nil
	}
	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return "", fmt.Errorf("read program source file %q: %w", sourceFile, err)
	}
	return string(data), nil
}

func (h *handlers) get(ctx cliapp.OperationContext) (*programsv1.GetProgramResponse, error) {
	r, e := h.client.GetProgram(context.Background(), connect.NewRequest(&programsv1.GetProgramRequest{Id: ctx.Positional("id")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("get program", e, nil)
	}
	return r.Msg, nil
}

// waitCommand is the standalone block-once primitive. It exists as its own
// command, not only as a flag on submit, so a caller that already has a program
// id — a resumed wait, a workflow step, an operator following up — has a
// governed way to block without reimplementing a poll loop.
func (h *handlers) waitCommand(ctx cliapp.OperationContext) (*programsv1.WaitForProgramResponse, error) {
	timeout, err := parseWaitTimeout(ctx.Flag("timeout"))
	if err != nil {
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(context.Background(), timeout+waitTransportMargin)
	defer cancel()
	r, e := h.client.WaitForProgram(callCtx, connect.NewRequest(&programsv1.WaitForProgramRequest{
		Id:            ctx.Positional("id"),
		TimeoutMillis: timeout.Milliseconds(),
	}))
	if e != nil {
		return nil, cliapp.WrapAPIError("wait for program", e, nil)
	}
	return r.Msg, nil
}

func (*handlers) waitReport(_ cliapp.OperationContext, r *programsv1.WaitForProgramResponse) cliapp.ListReport {
	if r.GetTerminal() {
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("Program %s is terminal: %s.", r.GetProgram().GetId(), r.GetProgram().GetStatus())}}
	}
	// Not-terminal is a stated outcome, not a failure: the wait returned at its
	// bound and the caller may wait again on the same id.
	return cliapp.ListReport{Summary: []string{fmt.Sprintf(
		"Program %s is still %s after %dms; wait again on the same id to resume.",
		r.GetProgram().GetId(), r.GetProgram().GetStatus(), r.GetWaitedMillis())}}
}

func (h *handlers) list(ctx cliapp.OperationContext) (*programsv1.ListProgramsResponse, error) {
	r, e := h.client.ListPrograms(context.Background(), connect.NewRequest(&programsv1.ListProgramsRequest{SessionId: ctx.Flag("session-id"), IncludeOperator: ctx.BoolFlag("include-operator")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("list programs", e, nil)
	}
	return r.Msg, nil
}

func (h *handlers) mine(ctx cliapp.OperationContext) (*programsv1.MineFailuresResponse, error) {
	r, e := h.client.MineFailures(context.Background(), connect.NewRequest(&programsv1.MineFailuresRequest{IncludeOperator: ctx.BoolFlag("include-operator")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("mine failures", e, nil)
	}
	return r.Msg, nil
}

func (h *handlers) mineRefusals(ctx cliapp.OperationContext) (*programsv1.MineRefusalsResponse, error) {
	r, e := h.client.MineRefusals(context.Background(), connect.NewRequest(&programsv1.MineRefusalsRequest{IncludeOperator: ctx.BoolFlag("include-operator")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("mine refusals", e, nil)
	}
	return r.Msg, nil
}

func (h *handlers) mineUnresolved(ctx cliapp.OperationContext) (*programsv1.MineUnresolvedBindingsResponse, error) {
	r, e := h.client.MineUnresolvedBindings(context.Background(), connect.NewRequest(&programsv1.MineUnresolvedBindingsRequest{}))
	if e != nil {
		return nil, cliapp.WrapAPIError("mine unresolved bindings", e, nil)
	}
	return r.Msg, nil
}

func (*handlers) submitReport(cliapp.OperationContext, *programsv1.SubmitProgramResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Program submitted."}}
}

func (*handlers) programReport(cliapp.OperationContext, *programsv1.GetProgramResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Program operation completed."}}
}

func (*handlers) listReport(_ cliapp.OperationContext, r *programsv1.ListProgramsResponse) cliapp.ListReport {
	results := make([]string, 0, len(r.Programs))
	for _, program := range r.GetPrograms() {
		if program == nil {
			continue
		}
		results = append(results, fmt.Sprintf("%s [%s] session=%s library=%s", program.GetId(), program.GetStatus().String(), program.GetSessionId(), program.GetLibraryVersion()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d program(s).", len(r.Programs))}, ResultsHeading: "Programs", Results: results}
}

func (*handlers) failureReport(_ cliapp.OperationContext, r *programsv1.MineFailuresResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d recurring failure shape(s).", len(r.Shapes))}}
}

func (*handlers) refusalReport(_ cliapp.OperationContext, r *programsv1.MineRefusalsResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d recurring refusal shape(s).", len(r.Shapes))}}
}

func (*handlers) unresolvedReport(_ cliapp.OperationContext, r *programsv1.MineUnresolvedBindingsResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d unresolved binding name(s).", len(r.Shapes))}}
}

func parseProvenance(value string) (programsv1.Provenance, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "agent":
		return programsv1.Provenance_PROVENANCE_AGENT, nil
	case "operator":
		return programsv1.Provenance_PROVENANCE_OPERATOR, nil
	case "test":
		return programsv1.Provenance_PROVENANCE_TEST, nil
	case "replay":
		return programsv1.Provenance_PROVENANCE_REPLAY, nil
	default:
		return programsv1.Provenance_PROVENANCE_UNSPECIFIED, fmt.Errorf("provenance must be agent, operator, test, or replay")
	}
}
