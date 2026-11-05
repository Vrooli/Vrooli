package validation_record

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	vrv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_record"
	vrconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_record/validation_record_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client vrconnect.ValidationRecordServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: vrconnect.NewValidationRecordServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	pageSize := 0
	if raw := strings.TrimSpace(ctx.Flag("page-size")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			pageSize = n
		}
	}
	req := &vrv1.ListRecordsRequest{
		GoldenSlug: ctx.Flag("golden"),
		SubjectId:  ctx.Flag("subject"),
		TupleKind:  parseTupleKind(ctx.Flag("kind")),
		PageSize:   int32(pageSize),
		PageToken:  ctx.Flag("page-token"),
	}
	resp, err := h.client.ListRecords(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("list records", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no records response")
	}
	results := make([]string, 0, len(resp.Msg.Records))
	for _, r := range resp.Msg.Records {
		results = append(results, formatRecord(r))
	}
	hints := []string{}
	if resp.Msg.NextPageToken != "" {
		hints = append(hints, fmt.Sprintf("`record list --page-token %s` — next page", resp.Msg.NextPageToken))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d record(s).", len(resp.Msg.Records))},
		ResultsHeading: "Records",
		Results:        results,
		RetrievalHints: hints,
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetRecord(context.Background(), connect.NewRequest(&vrv1.GetRecordRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get record %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Record == nil {
		return fmt.Errorf("server returned no record")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched record %s.", resp.Msg.Record.Id)},
		ResultsHeading: "Record",
		Results:        []string{formatRecord(resp.Msg.Record)},
	})
}

func parseTupleKind(s string) vrv1.TupleKind {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "skill":
		return vrv1.TupleKind_TUPLE_KIND_SKILL
	case "tool":
		return vrv1.TupleKind_TUPLE_KIND_TOOL
	default:
		return vrv1.TupleKind_TUPLE_KIND_UNSPECIFIED
	}
}

func formatRecord(r *vrv1.ValidationRecord) string {
	if r == nil {
		return "(nil)"
	}
	ended := ""
	if r.EndedAt != nil {
		ended = r.EndedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — %s/%s/%s verdict=%s duration=%dms ended=%s",
		r.Id,
		tupleKindLabel(r.TupleKind), r.SubjectId, r.GoldenSlug,
		verdictLabel(r.Verdict),
		r.DurationMs, ended,
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
