package journal

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/journal/journal_v1connect"
)

type handlers struct {
	client journalconnect.JournalServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	// Retry replay is intentionally server-owned and can span hundreds of
	// bounded inference calls. Do not let the generic CLI transport deadline
	// cancel it mid-batch; each individual gateway call still has its explicit
	// execution budget.
	http, base := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	return &handlers{client: journalconnect.NewJournalServiceClient(http, base)}
}

func (h *handlers) rememberCall(ctx cliapp.OperationContext) (*journalv1.RememberResponse, error) {
	request := &journalv1.RememberRequest{
		Observations: ctx.Positionals("observations"), Scope: ctx.Flag("scope"),
		Evidence: ctx.Flag("evidence"), RoutingHint: ctx.Flag("routing-hint"), RequestKey: ctx.Flag("request-key"),
	}
	resp, err := h.client.Remember(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("remember observations", err, nil)
	}
	if len(resp.Msg.GetEntries()) != len(request.Observations) {
		return nil, fmt.Errorf("server returned %d entries for %d observations", len(resp.Msg.GetEntries()), len(request.Observations))
	}
	return resp.Msg, nil
}

func (h *handlers) rememberReport(_ cliapp.OperationContext, msg *journalv1.RememberResponse) cliapp.MutationReport {
	entries := msg.GetEntries()
	changes := make([]string, 0, len(entries))
	for _, entry := range entries {
		changes = append(changes, entry.GetId()+"  "+entry.GetBody())
	}
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Remembered %d observation(s).", len(entries))}, Changes: changes}
}

func (h *handlers) noticesCall(ctx cliapp.OperationContext) (*journalv1.ListClassificationNoticesResponse, error) {
	after, err := strconv.ParseInt(ctx.Flag("after"), 10, 64)
	if err != nil || after < 0 {
		return nil, fmt.Errorf("after must be a nonnegative sequence")
	}
	limit, err := retryLimit(ctx.Flag("limit"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.ListClassificationNotices(context.Background(), connect.NewRequest(&journalv1.ListClassificationNoticesRequest{
		Scope: ctx.Flag("scope"), CategoryId: ctx.Positional("category-id"), AfterSequence: after, Limit: limit,
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list classification notices", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) noticesReport(_ cliapp.OperationContext, msg *journalv1.ListClassificationNoticesResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetNotices()))
	for _, notice := range msg.GetNotices() {
		results = append(results, fmt.Sprintf("%d  %s  %s", notice.GetSequence(), notice.GetScope(), notice.GetObservationId()))
	}
	return cliapp.ListReport{Results: results, Summary: []string{fmt.Sprintf("%d classification notice(s).", len(results))}}
}

func (h *handlers) noteCall(ctx cliapp.OperationContext) (*journalv1.AppendEntryResponse, error) {
	body, err := resolveNoteBody(ctx.Positional("body"), len(ctx.Args()) > 0, ctx.Flag("body-file"), os.Stdin)
	if err != nil {
		return nil, err
	}
	resp, err := h.client.AppendEntry(context.Background(), connect.NewRequest(&journalv1.AppendEntryRequest{Body: body, Scope: ctx.Flag("scope"), Kind: ctx.Flag("kind"), Trigger: ctx.Flag("trigger"), Approach: ctx.Flag("approach"), Evidence: ctx.Flag("evidence"), Outcome: ctx.Flag("outcome")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("append memory", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Entry == nil {
		return nil, fmt.Errorf("server returned no entry")
	}
	return resp.Msg, nil
}

func resolveNoteBody(positional string, positionalProvided bool, bodyFile string, stdin io.Reader) (string, error) {
	if positionalProvided && bodyFile != "" {
		return "", fmt.Errorf("provide the memory body as a positional argument or with --body-file, not both")
	}
	if bodyFile == "" {
		if !positionalProvided {
			return "", fmt.Errorf("memory body is required; work-record fields do not replace it; pass <body> or --body-file PATH (use - for stdin)")
		}
		return positional, nil
	}
	if bodyFile == "-" {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("read memory body from stdin: %w", err)
		}
		return string(data), nil
	}
	data, err := os.ReadFile(bodyFile)
	if err != nil {
		return "", fmt.Errorf("read memory body file %q: %w", bodyFile, err)
	}
	return string(data), nil
}

func (h *handlers) noteReport(_ cliapp.OperationContext, msg *journalv1.AppendEntryResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Remembered %s.", msg.Entry.Id)}, Changes: []string{fmt.Sprintf("[%s] %s", msg.Entry.FacetId, msg.Entry.Body)}, NextCommand: []string{"`recall <query>` — retrieve related memory"}}
}

func (h *handlers) retryCall(ctx cliapp.OperationContext) (*journalv1.ProcessClassificationRetriesResponse, error) {
	limit, err := retryLimit(ctx.Flag("limit"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.ProcessClassificationRetries(context.Background(), connect.NewRequest(&journalv1.ProcessClassificationRetriesRequest{Limit: limit, Scope: ctx.Flag("scope")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("process classification retries", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no retry result")
	}
	return resp.Msg, nil
}

func (h *handlers) retryReport(_ cliapp.OperationContext, msg *journalv1.ProcessClassificationRetriesResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Processed %d classification retry item(s).", msg.GetProcessed())}, Changes: []string{fmt.Sprintf("Deferred: %d; already resolved: %d.", msg.GetDeferred(), msg.GetAlreadyResolved())}, NextCommand: []string{"`journal retry-classifications --limit 500` — continue replaying backlog"}}
}

func (h *handlers) retryEmbeddingsCall(ctx cliapp.OperationContext) (*journalv1.ProcessEmbeddingRetriesResponse, error) {
	limit, err := retryLimit(ctx.Flag("limit"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.ProcessEmbeddingRetries(context.Background(), connect.NewRequest(&journalv1.ProcessEmbeddingRetriesRequest{Limit: limit, Scope: ctx.Flag("scope")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("process embedding retries", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no retry result")
	}
	return resp.Msg, nil
}

func (h *handlers) retryEmbeddingsReport(_ cliapp.OperationContext, msg *journalv1.ProcessEmbeddingRetriesResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Processed %d embedding retry item(s).", msg.GetProcessed())}, Changes: []string{fmt.Sprintf("Deferred: %d; already resolved: %d.", msg.GetDeferred(), msg.GetAlreadyResolved())}, NextCommand: []string{"`retry-embeddings --limit 100` — continue replaying backlog"}}
}

func retryLimit(raw string) (int32, error) {
	if raw == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || parsed < 0 || parsed > 500 {
		return 0, fmt.Errorf("limit must be an integer from 0 through 500")
	}
	return int32(parsed), nil
}
