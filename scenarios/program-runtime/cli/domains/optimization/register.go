package optimization

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	optimizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/optimization"
	optimizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/optimization/optimization_v1connect"
)

const GroupName = "optimization"

type handlers struct {
	client optimizationconnect.OptimizationServiceClient
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	h := &handlers{client: optimizationconnect.NewOptimizationServiceClient(httpClient, baseURL)}
	return cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"vrooli.program_runtime.v1.optimization.OptimizationService.Submit": cliapp.ProtoMutation(h.submit, h.submitReport),
		"vrooli.program_runtime.v1.optimization.OptimizationService.Wait":   cliapp.ProtoList(h.wait, h.waitReport),
	})
}

func (h *handlers) submit(ctx cliapp.OperationContext) (*optimizationv1.SubmitOptimizationResponse, error) {
	requestJSON := strings.TrimSpace(ctx.Flag("request-json"))
	var request map[string]any
	if requestJSON == "" || json.Unmarshal([]byte(requestJSON), &request) != nil || request == nil {
		return nil, fmt.Errorf("request-json must be one JSON object")
	}
	response, err := h.client.Submit(context.Background(), connect.NewRequest(&optimizationv1.SubmitOptimizationRequest{RequestJson: requestJSON}))
	if err != nil {
		return nil, cliapp.WrapAPIError("submit optimization", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) submitReport(_ cliapp.OperationContext, response *optimizationv1.SubmitOptimizationResponse) cliapp.MutationReport {
	stage, revision := recordSummary(response.GetRecordJson())
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Optimization run %s submitted at stage %s, revision %d.", response.GetRunId(), stage, revision)}}
}

func (h *handlers) wait(ctx cliapp.OperationContext) (*optimizationv1.WaitOptimizationResponse, error) {
	runID := strings.TrimSpace(ctx.Flag("run-id"))
	waitMS, err := strconv.ParseUint(strings.TrimSpace(ctx.Flag("wait-ms")), 10, 32)
	if err != nil || runID == "" || waitMS < 1 || waitMS > 60000 {
		return nil, fmt.Errorf("run-id is required and wait-ms must be between 1 and 60000")
	}
	response, err := h.client.Wait(context.Background(), connect.NewRequest(&optimizationv1.WaitOptimizationRequest{RunId: runID, WaitMs: uint32(waitMS)}))
	if err != nil {
		return nil, cliapp.WrapAPIError("wait for optimization", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) waitReport(ctx cliapp.OperationContext, response *optimizationv1.WaitOptimizationResponse) cliapp.ListReport {
	stage, revision := recordSummary(response.GetRecordJson())
	state := "pending"
	if response.GetTerminal() {
		state = "terminal"
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Optimization run %s after %d ms: %s at stage %s, revision %d.", ctx.Flag("run-id"), response.GetWaitedMs(), state, stage, revision)}}
}

func recordSummary(raw string) (stage string, revision uint64) {
	var record struct {
		Stage    string `json:"stage"`
		Revision uint64 `json:"revision"`
	}
	if json.Unmarshal([]byte(raw), &record) != nil {
		return "unknown", 0
	}
	if record.Stage == "" {
		record.Stage = "unknown"
	}
	return record.Stage, record.Revision
}
