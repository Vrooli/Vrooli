package validation_run

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	vrunv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_run"
	vrunconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_run/validation_run_v1connect"
	vrv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_record"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client vrunconnect.ValidationRunServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: vrunconnect.NewValidationRunServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) start(ctx cliapp.RunContext) error {
	skill := strings.TrimSpace(ctx.Flag("skill"))
	tool := strings.TrimSpace(ctx.Flag("tool"))
	if (skill == "") == (tool == "") {
		return fmt.Errorf("exactly one of --skill or --tool must be provided")
	}
	kind := vrv1.TupleKind_TUPLE_KIND_SKILL
	subject := skill
	if tool != "" {
		kind = vrv1.TupleKind_TUPLE_KIND_TOOL
		subject = tool
	}
	req := &vrunv1.StartRequest{
		TupleKind:  kind,
		SubjectId:  subject,
		GoldenSlug: ctx.Flag("golden"),
		Force:      ctx.BoolFlag("force"),
	}
	resp, err := h.client.Start(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("start validation %s/%s", subject, ctx.Flag("golden")), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Run == nil {
		return fmt.Errorf("server returned no run")
	}
	if !ctx.BoolFlag("wait") {
		return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
			Result:  []string{fmt.Sprintf("Queued run %s.", resp.Msg.Run.Id)},
			Changes: []string{formatRun(resp.Msg.Run)},
			NextCommand: []string{
				fmt.Sprintf("`validation get %s` — poll for terminal status", resp.Msg.Run.Id),
			},
		})
	}
	timeout := 300
	if raw := strings.TrimSpace(ctx.Flag("wait-timeout")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			timeout = n
		}
	}
	final, err := h.poll(resp.Msg.Run.Id, time.Duration(timeout)*time.Second)
	if err != nil {
		return err
	}
	return cliapp.RenderProtoMutation(ctx, final, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Run %s reached %s.", final.Id, statusLabel(final.Status))},
		Changes: []string{formatRun(final)},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.Get(context.Background(), connect.NewRequest(&vrunv1.GetRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get validation run %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Run == nil {
		return fmt.Errorf("server returned no run")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched run %s.", resp.Msg.Run.Id)},
		ResultsHeading: "Run",
		Results:        []string{formatRun(resp.Msg.Run)},
	})
}

func (h *handlers) listActive(ctx cliapp.RunContext) error {
	resp, err := h.client.ListActive(context.Background(), connect.NewRequest(&vrunv1.ListActiveRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list active validation runs", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no list-active response")
	}
	results := make([]string, 0, len(resp.Msg.Runs))
	for _, r := range resp.Msg.Runs {
		results = append(results, formatRun(r))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d active run(s).", len(resp.Msg.Runs))},
		ResultsHeading: "Active runs",
		Results:        results,
	})
}

func (h *handlers) poll(id string, timeout time.Duration) (*vrunv1.ValidationRun, error) {
	deadline := time.Now().Add(timeout)
	for {
		resp, err := h.client.Get(context.Background(), connect.NewRequest(&vrunv1.GetRequest{Id: id}))
		if err != nil {
			return nil, cliapp.WrapAPIError(fmt.Sprintf("poll validation run %q", id), err, nil)
		}
		if resp == nil || resp.Msg == nil || resp.Msg.Run == nil {
			return nil, fmt.Errorf("server returned no run during poll")
		}
		if resp.Msg.Run.Status == vrunv1.Status_STATUS_TERMINAL {
			return resp.Msg.Run, nil
		}
		if time.Now().After(deadline) {
			return resp.Msg.Run, fmt.Errorf("validation run %s did not reach terminal status within %s", id, timeout)
		}
		time.Sleep(2 * time.Second)
	}
}

func formatRun(r *vrunv1.ValidationRun) string {
	if r == nil {
		return "(nil)"
	}
	verdict := ""
	if r.Status == vrunv1.Status_STATUS_TERMINAL {
		verdict = " verdict=" + verdictLabel(r.TerminalVerdict)
	}
	created := ""
	if r.CreatedAt != nil {
		created = r.CreatedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — %s/%s/%s status=%s%s created=%s err=%q",
		r.Id, tupleKindLabel(r.TupleKind), r.SubjectId, r.GoldenSlug,
		statusLabel(r.Status), verdict, created, r.ErrorMessage,
	)
}

func tupleKindLabel(t vrv1.TupleKind) string {
	switch t {
	case vrv1.TupleKind_TUPLE_KIND_SKILL:
		return "skill"
	case vrv1.TupleKind_TUPLE_KIND_TOOL:
		return "tool"
	default:
		return "unspecified"
	}
}

func statusLabel(s vrunv1.Status) string {
	switch s {
	case vrunv1.Status_STATUS_QUEUED:
		return "queued"
	case vrunv1.Status_STATUS_RUNNING:
		return "running"
	case vrunv1.Status_STATUS_EVALUATING:
		return "evaluating"
	case vrunv1.Status_STATUS_TERMINAL:
		return "terminal"
	default:
		return "unspecified"
	}
}

func verdictLabel(v vrv1.Verdict) string {
	switch v {
	case vrv1.Verdict_VERDICT_PASS:
		return "pass"
	case vrv1.Verdict_VERDICT_UNEXPECTED_MUTATION:
		return "unexpected_mutation"
	case vrv1.Verdict_VERDICT_RUN_FAILURE:
		return "run_failure"
	case vrv1.Verdict_VERDICT_TOOL_FAILURE:
		return "tool_failure"
	default:
		return "unspecified"
	}
}
