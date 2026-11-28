package codingjobs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/codingjobs"
	connectclient "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/codingjobs/codingjobs_v1connect"
)

type handlers struct {
	client connectclient.CodingJobServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: connectclient.NewCodingJobServiceClient(httpClient, baseURL)}
}

func (h *handlers) createCall(ctx cliapp.OperationContext) (*v1.CreateCodingJobResponse, error) {
	resp, err := h.client.CreateCodingJob(context.Background(), connect.NewRequest(&v1.CreateCodingJobRequest{ReplayKey: ctx.Flag("replay-key"), CaseId: ctx.Flag("case-id"), Project: ctx.Flag("project"), Title: ctx.Flag("title"), Task: ctx.Flag("task"), SourceSealRef: ctx.Flag("source-seal-ref")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create coding job", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) createReport(_ cliapp.OperationContext, m *v1.CreateCodingJobResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created coding job %s (%s).", m.Job.Id, m.Job.State)}, Changes: []string{formatJob(m.Job)}}
}

func (h *handlers) stageCall(ctx cliapp.OperationContext) (*v1.RecordCodingJobStageResponse, error) {
	revision, err := parseRevision(ctx.Flag("expected-revision"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.RecordCodingJobStage(context.Background(), connect.NewRequest(&v1.RecordCodingJobStageRequest{Id: ctx.Positional("id"), Stage: ctx.Flag("stage"), Actor: ctx.Flag("actor"), EvidenceRef: ctx.Flag("evidence-ref"), EvidenceSha256: ctx.Flag("evidence-sha256"), Note: ctx.Flag("note"), ExpectedRevision: revision}))
	if err != nil {
		return nil, cliapp.WrapAPIError("record coding job stage", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) stageReport(_ cliapp.OperationContext, m *v1.RecordCodingJobStageResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Recorded %d event(s), revision %d.", len(m.Job.Events), m.Job.Revision)}, Changes: []string{formatJob(m.Job)}}
}

func (h *handlers) evaluationCall(ctx cliapp.OperationContext) (*v1.RecordCodingJobEvaluationResponse, error) {
	revision, err := parseRevision(ctx.Flag("expected-revision"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.RecordCodingJobEvaluation(context.Background(), connect.NewRequest(&v1.RecordCodingJobEvaluationRequest{Id: ctx.Positional("id"), ReceiptRef: ctx.Flag("receipt-ref"), ReceiptSha256: ctx.Flag("receipt-sha256"), Score: ctx.Flag("score"), ExpectedRevision: revision}))
	if err != nil {
		return nil, cliapp.WrapAPIError("record coding job evaluation", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) evaluationReport(_ cliapp.OperationContext, m *v1.RecordCodingJobEvaluationResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Recorded immutable evaluation %s (%s).", m.Job.EvaluatorReceiptRef, m.Job.Score)}, Changes: []string{formatJob(m.Job)}}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*v1.GetCodingJobResponse, error) {
	resp, err := h.client.GetCodingJob(context.Background(), connect.NewRequest(&v1.GetCodingJobRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get coding job", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) getReport(_ cliapp.OperationContext, m *v1.GetCodingJobResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{formatJob(m.Job)}, ResultsHeading: "Coding job events", Results: formatEvents(m.Job.Events)}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*v1.ListCodingJobsResponse, error) {
	limit := uint32(50)
	if raw := strings.TrimSpace(ctx.Flag("limit")); raw != "" {
		n, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("limit must be a positive integer")
		}
		limit = uint32(n)
	}
	resp, err := h.client.ListCodingJobs(context.Background(), connect.NewRequest(&v1.ListCodingJobsRequest{Limit: limit}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list coding jobs", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) listReport(_ cliapp.OperationContext, m *v1.ListCodingJobsResponse) cliapp.ListReport {
	items := make([]string, 0, len(m.Jobs))
	for _, j := range m.Jobs {
		items = append(items, formatJob(j))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d coding job(s).", len(items))}, ResultsHeading: "Coding jobs", Results: items}
}

func parseRevision(raw string) (uint64, error) {
	n, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil || n == 0 {
		return 0, fmt.Errorf("expected-revision must be a positive integer")
	}
	return n, nil
}
func formatJob(j *v1.CodingJob) string {
	if j == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s [%s, revision=%d, case=%s, score=%s]", j.Id, j.Title, j.State, j.Revision, j.CaseId, j.Score)
}
func formatEvents(events []*v1.CodingJobEvent) []string {
	out := make([]string, 0, len(events))
	for _, e := range events {
		out = append(out, fmt.Sprintf("#%d %s by %s · %s", e.Sequence, e.Stage, e.Actor, e.EvidenceRef))
	}
	return out
}
