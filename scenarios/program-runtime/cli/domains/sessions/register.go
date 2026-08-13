package sessions

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/sessions/sessions_v1connect"
)

const GroupName = "sessions"

type handlers struct {
	client sessionsconnect.SessionServiceClient
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	h := &handlers{client: sessionsconnect.NewSessionServiceClient(httpClient, baseURL)}
	return cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"SessionService.CreateSession":                                   cliapp.ProtoMutation(h.create, h.createReport),
		"SessionService.GetSession":                                      cliapp.ProtoList(h.get, h.getReport),
		"vrooli.program_runtime.v1.sessions.SessionService.ListSessions": cliapp.ProtoList(h.list, h.listReport),
		"SessionService.DeleteSession":                                   cliapp.ProtoMutation(h.delete, h.deleteReport),
		"SessionService.GrantSession":                                    cliapp.ProtoMutation(h.grant, h.grantReport),
	})
}

func (h *handlers) create(ctx cliapp.OperationContext) (*sessionsv1.CreateSessionResponse, error) {
	inferenceCeiling, err := parseCeiling(ctx.Flag("inference-ceiling-micros"))
	if err != nil {
		return nil, err
	}
	delegationCeiling, err := parseCeiling(ctx.Flag("delegation-ceiling-micros"))
	if err != nil {
		return nil, err
	}
	wallBudget, err := parseBudget(ctx.Flag("wall-budget"))
	if err != nil {
		return nil, err
	}
	cpuBudget, err := parseBudget(ctx.Flag("cpu-budget"))
	if err != nil {
		return nil, err
	}
	r, e := h.client.CreateSession(context.Background(), connect.NewRequest(&sessionsv1.CreateSessionRequest{Name: ctx.Flag("name"), SandboxWorkspace: ctx.Flag("sandbox-workspace"), Grants: cliutil.ParseCSV(ctx.Flag("grants")), InferenceCeilingMicros: inferenceCeiling, DelegationCeilingMicros: delegationCeiling, WallBudgetMillis: wallBudget, CpuBudgetMillis: cpuBudget}))
	if e != nil {
		return nil, cliapp.WrapAPIError("create session", e, nil)
	}
	return r.Msg, nil
}

func parseBudget(value string) (int64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	duration, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("execution budget must be a positive duration such as 30s")
	}
	return duration.Milliseconds(), nil
}

func parseCeiling(value string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("spend ceiling must be a non-negative integer in micros")
	}
	return parsed, nil
}

func (h *handlers) get(ctx cliapp.OperationContext) (*sessionsv1.GetSessionResponse, error) {
	r, e := h.client.GetSession(context.Background(), connect.NewRequest(&sessionsv1.GetSessionRequest{Id: ctx.Positional("id")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("get session", e, nil)
	}
	return r.Msg, nil
}

func (h *handlers) list(_ cliapp.OperationContext) (*sessionsv1.ListSessionsResponse, error) {
	r, e := h.client.ListSessions(context.Background(), connect.NewRequest(&sessionsv1.ListSessionsRequest{}))
	if e != nil {
		return nil, cliapp.WrapAPIError("list sessions", e, nil)
	}
	return r.Msg, nil
}

func (h *handlers) delete(ctx cliapp.OperationContext) (*sessionsv1.DeleteSessionResponse, error) {
	r, e := h.client.DeleteSession(context.Background(), connect.NewRequest(&sessionsv1.DeleteSessionRequest{Id: ctx.Positional("id"), Reason: ctx.Flag("reason")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("delete session", e, nil)
	}
	return r.Msg, nil
}

func (h *handlers) grant(ctx cliapp.OperationContext) (*sessionsv1.GrantSessionResponse, error) {
	r, e := h.client.GrantSession(context.Background(), connect.NewRequest(&sessionsv1.GrantSessionRequest{Id: ctx.Positional("id"), Grants: cliutil.ParseCSV(ctx.Flag("grants"))}))
	if e != nil {
		return nil, cliapp.WrapAPIError("grant session", e, nil)
	}
	return r.Msg, nil
}

func (*handlers) createReport(cliapp.OperationContext, *sessionsv1.CreateSessionResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Session created."}}
}

func (*handlers) getReport(cliapp.OperationContext, *sessionsv1.GetSessionResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Session read."}}
}

func (*handlers) deleteReport(cliapp.OperationContext, *sessionsv1.DeleteSessionResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Session reclaimed."}}
}

func (*handlers) grantReport(cliapp.OperationContext, *sessionsv1.GrantSessionResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Session grant added."}}
}

func (h *handlers) listReport(_ cliapp.OperationContext, r *sessionsv1.ListSessionsResponse) cliapp.ListReport {
	items := make([]string, 0, len(r.Sessions))
	for _, s := range r.Sessions {
		items = append(items, fmt.Sprintf("%s [%s]", s.Id, s.State))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d session(s).", len(items))}, ResultsHeading: "Sessions", Results: items}
}
