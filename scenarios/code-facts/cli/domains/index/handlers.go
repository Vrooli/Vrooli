package index

import (
	"context"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
)

const (
	controlTokenEnv    = "CODE_FACTS_INDEX_CONTROL_TOKEN" // #nosec G101 -- environment variable name, not a credential.
	controlTokenHeader = "X-Code-Facts-Control-Token"     // #nosec G101 -- protocol header name, not a credential.
)

type handlers struct {
	client factsconnect.CodeFactsServiceClient
	getenv func(string) string
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: factsconnect.NewCodeFactsServiceClient(httpClient, baseURL), getenv: os.Getenv}
}

func (h *handlers) statusCall(cliapp.OperationContext) (*factsv1.IndexStatus, error) {
	response, err := h.client.GetIndexStatus(context.Background(), connect.NewRequest(&factsv1.GetIndexStatusRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get index status", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no index status")
	}
	return response.Msg, nil
}

func (h *handlers) reconcileCall(ctx cliapp.OperationContext) (*factsv1.IndexControlResponse, error) {
	response, err := h.client.ReconcileIndex(context.Background(), controlRequest(h.getenv, &factsv1.ReconcileIndexRequest{Generation: ctx.Flag("generation")}))
	return controlMessage("reconcile index", response, err)
}

func (h *handlers) reindexCall(ctx cliapp.OperationContext) (*factsv1.IndexControlResponse, error) {
	response, err := h.client.Reindex(context.Background(), controlRequest(h.getenv, &factsv1.ReindexRequest{Generation: ctx.Positional("generation"), Confirmed: ctx.BoolFlag("confirm")}))
	return controlMessage("reindex", response, err)
}

func (h *handlers) cancelCall(ctx cliapp.OperationContext) (*factsv1.IndexControlResponse, error) {
	response, err := h.client.CancelIndexJob(context.Background(), controlRequest(h.getenv, &factsv1.CancelIndexJobRequest{JobId: ctx.Positional("job-id")}))
	return controlMessage("cancel index job", response, err)
}

func (h *handlers) promoteCall(ctx cliapp.OperationContext) (*factsv1.IndexControlResponse, error) {
	response, err := h.client.PromoteIndexGeneration(context.Background(), controlRequest(h.getenv, &factsv1.PromoteIndexGenerationRequest{Generation: ctx.Positional("generation"), Confirmed: ctx.BoolFlag("confirm")}))
	return controlMessage("promote index generation", response, err)
}

func (h *handlers) rollbackCall(ctx cliapp.OperationContext) (*factsv1.IndexControlResponse, error) {
	response, err := h.client.RollbackIndexGeneration(context.Background(), controlRequest(h.getenv, &factsv1.RollbackIndexGenerationRequest{Generation: ctx.Positional("generation"), Confirmed: ctx.BoolFlag("confirm")}))
	return controlMessage("rollback index generation", response, err)
}

func (h *handlers) cleanupCall(ctx cliapp.OperationContext) (*factsv1.IndexControlResponse, error) {
	response, err := h.client.CleanupIndex(context.Background(), controlRequest(h.getenv, &factsv1.CleanupIndexRequest{DryRun: ctx.BoolFlag("dry-run"), Confirmed: ctx.BoolFlag("confirm")}))
	return controlMessage("cleanup index", response, err)
}

func controlRequest[T any](getenv func(string) string, message *T) *connect.Request[T] {
	request := connect.NewRequest(message)
	token := ""
	if getenv != nil {
		token = strings.TrimSpace(getenv(controlTokenEnv))
	}
	if token != "" {
		request.Header().Set(controlTokenHeader, token)
	}
	return request
}

func controlMessage(operation string, response *connect.Response[factsv1.IndexControlResponse], err error) (*factsv1.IndexControlResponse, error) {
	if err != nil {
		return nil, cliapp.WrapAPIError(operation, err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("server returned no index control response")
	}
	return response.Msg, nil
}

func (h *handlers) statusReport(_ cliapp.OperationContext, status *factsv1.IndexStatus) cliapp.ListReport {
	results := make([]string, 0, len(status.GetActiveJobs()))
	for _, job := range status.GetActiveJobs() {
		results = append(results, fmt.Sprintf("%s %s generation=%s processed=%d/%d", job.GetId(), job.GetState().String(), job.GetGeneration(), job.GetProcessed(), job.GetTotal()))
	}
	return cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Index %s; active generation %s; previous %s.", status.GetState(), status.GetActiveGeneration(), status.GetPreviousGeneration()),
			fmt.Sprintf("%d files, %d documents, %d cards, %d graph facts, %d bytes.", status.GetSourceFiles(), status.GetSearchDocuments(), status.GetSemanticCards(), status.GetGraphFacts(), status.GetStorageBytes()),
		},
		ResultsHeading: "Active jobs",
		Results:        results,
	}
}

func (h *handlers) mutationReport(_ cliapp.OperationContext, response *factsv1.IndexControlResponse) cliapp.MutationReport {
	result := []string{response.GetMessage()}
	if job := response.GetJob(); job != nil {
		result = append(result, fmt.Sprintf("job=%s state=%s generation=%s processed=%d/%d", job.GetId(), job.GetState().String(), job.GetGeneration(), job.GetProcessed(), job.GetTotal()))
	}
	return cliapp.MutationReport{Result: result}
}
