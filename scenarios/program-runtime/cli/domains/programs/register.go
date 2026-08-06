package programs

import (
	"context"
	"fmt"
	"strings"

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
	return cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{"ProgramService.SubmitProgram": cliapp.ProtoMutation(h.submit, h.submitReport), "ProgramService.GetProgram": cliapp.ProtoList(h.get, h.programReport), "ProgramService.ListPrograms": cliapp.ProtoList(h.list, h.listReport), "ProgramService.MineFailures": cliapp.ProtoList(h.mine, h.failureReport)})
}

func (h *handlers) submit(ctx cliapp.OperationContext) (*programsv1.SubmitProgramResponse, error) {
	provenance, err := parseProvenance(ctx.Flag("provenance"))
	if err != nil {
		return nil, err
	}
	r, e := h.client.SubmitProgram(context.Background(), connect.NewRequest(&programsv1.SubmitProgramRequest{SessionId: ctx.Flag("session-id"), Source: ctx.Flag("source"), Provenance: provenance, IncludeMaterialized: ctx.BoolFlag("include-materialized")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("submit program", e, nil)
	}
	return r.Msg, nil
}

func (h *handlers) get(ctx cliapp.OperationContext) (*programsv1.GetProgramResponse, error) {
	r, e := h.client.GetProgram(context.Background(), connect.NewRequest(&programsv1.GetProgramRequest{Id: ctx.Positional("id")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("get program", e, nil)
	}
	return r.Msg, nil
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

func (*handlers) submitReport(cliapp.OperationContext, *programsv1.SubmitProgramResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Program submitted."}}
}

func (*handlers) programReport(cliapp.OperationContext, *programsv1.GetProgramResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Program operation completed."}}
}

func (*handlers) listReport(_ cliapp.OperationContext, r *programsv1.ListProgramsResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d program(s).", len(r.Programs))}}
}

func (*handlers) failureReport(_ cliapp.OperationContext, r *programsv1.MineFailuresResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d recurring failure shape(s).", len(r.Shapes))}}
}

func parseProvenance(value string) (programsv1.Provenance, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "agent":
		return programsv1.Provenance_PROVENANCE_AGENT, nil
	case "operator":
		return programsv1.Provenance_PROVENANCE_OPERATOR, nil
	default:
		return programsv1.Provenance_PROVENANCE_UNSPECIFIED, fmt.Errorf("provenance must be agent or operator")
	}
}
