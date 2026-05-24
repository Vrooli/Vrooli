package rewrite

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"connectrpc.com/connect"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph/graph_v1connect"
	rewritev1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/rewrite"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each
// RunCtx-func has typed access to the API client without re-resolving it.
type handlers struct {
	core   *cliapp.ScenarioApp
	client graphconnect.TypeScriptCodeGraphServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: graphconnect.NewTypeScriptCodeGraphServiceClient(httpClient, baseURL),
	}
}

// opsFileEntry models one entry in the JSON file consumed by
// `rewrite plan`. The file is an array of these. Example:
//
//	[
//	  {"kind": "file_move", "from_path": "src/a.ts", "to_path": "src/b.ts"},
//	  {"kind": "import_rewrite", "old_path": "./a", "new_path": "./b"}
//	]
//
// The CLI translates this list into []*rewritev1.Operation and ships it
// to the API; the API owns validation, normalization, and persistence.
type opsFileEntry struct {
	Kind          string                `json:"kind"`
	FileMove      *opsFileMoveBody      `json:"file_move,omitempty"`
	ImportRewrite *opsImportRewriteBody `json:"import_rewrite,omitempty"`
	// Convenience inline fields — accepted in addition to the nested
	// shapes above so the file format stays human-friendly.
	FromPath string `json:"from_path,omitempty"`
	ToPath   string `json:"to_path,omitempty"`
	OldPath  string `json:"old_path,omitempty"`
	NewPath  string `json:"new_path,omitempty"`
}

type opsFileMoveBody struct {
	FromPath string `json:"from_path"`
	ToPath   string `json:"to_path"`
}

type opsImportRewriteBody struct {
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
}

// plan reads an ops-file from disk, parses it into rewrite Operations,
// and calls TypeScriptCodeGraphService.RewritePlan. Renders a
// MutationReport surfacing the returned plan_id and the normalized
// operation list.
func (h *handlers) plan(ctx cliapp.RunContext) error {
	opsFile := ctx.Positional("ops-file")
	scenarioPath := ctx.Flag("scenario-path")

	ops, err := loadOperations(opsFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", opsFile, err)
	}

	resp, err := h.client.RewritePlan(context.Background(), connect.NewRequest(&graphv1.RewritePlanRequest{
		ScenarioPath: scenarioPath,
		Operations:   ops,
	}))
	if err != nil {
		return cliapp.WrapAPIError("rewrite plan", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no rewrite plan response")
	}

	normalized := resp.Msg.GetNormalizedOperations()
	changes := make([]string, 0, len(normalized))
	for _, op := range normalized {
		changes = append(changes, formatOperation(op))
	}

	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Plan %s ready (%d op(s)).", resp.Msg.GetPlanId(), len(normalized)),
		},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf(
				"`rewrite apply %s --scenario-path %s` — apply this plan",
				resp.Msg.GetPlanId(), scenarioPath,
			),
		},
	})
}

// apply executes a previously-planned rewrite. cli-core threads --dry-run
// through to an X-Dry-Run: true HTTP header on the Connect call; the API
// returns dry_run=true in the response when that header was present.
func (h *handlers) apply(ctx cliapp.RunContext) error {
	planID := ctx.Positional("plan-id")
	scenarioPath := ctx.Flag("scenario-path")

	resp, err := h.client.RewriteApply(context.Background(), connect.NewRequest(&graphv1.RewriteApplyRequest{
		ScenarioPath: scenarioPath,
		PlanId:       planID,
		Apply:        true,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("rewrite apply %q", planID), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no rewrite apply response")
	}

	results := resp.Msg.GetResults()
	changes := make([]string, 0, len(results))
	failed := 0
	for _, r := range results {
		if r.GetStatus() == rewritev1.OperationStatus_OPERATION_STATUS_FAILED {
			failed++
		}
		changes = append(changes, formatResult(r))
	}

	result := []string{
		fmt.Sprintf(
			"Applied plan %s: %d op(s), %d failed.",
			resp.Msg.GetPlanId(), len(results), failed,
		),
	}
	if resp.Msg.GetDryRun() {
		result = append([]string{"DRY RUN — no changes made to the filesystem."}, result...)
	}

	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  result,
		Changes: changes,
	})
}

func loadOperations(path string) ([]*rewritev1.Operation, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []opsFileEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("parse ops file: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("ops file %s contained no operations", path)
	}
	ops := make([]*rewritev1.Operation, 0, len(entries))
	for i, e := range entries {
		op, err := entryToOperation(e)
		if err != nil {
			return nil, fmt.Errorf("op %d: %w", i, err)
		}
		ops = append(ops, op)
	}
	return ops, nil
}

func entryToOperation(e opsFileEntry) (*rewritev1.Operation, error) {
	switch e.Kind {
	case "file_move":
		fm := e.FileMove
		if fm == nil {
			fm = &opsFileMoveBody{FromPath: e.FromPath, ToPath: e.ToPath}
		}
		return &rewritev1.Operation{Op: &rewritev1.Operation_FileMove{
			FileMove: &rewritev1.FileMove{FromPath: fm.FromPath, ToPath: fm.ToPath},
		}}, nil
	case "import_rewrite":
		ir := e.ImportRewrite
		if ir == nil {
			ir = &opsImportRewriteBody{OldPath: e.OldPath, NewPath: e.NewPath}
		}
		return &rewritev1.Operation{Op: &rewritev1.Operation_ImportRewrite{
			ImportRewrite: &rewritev1.ImportRewrite{OldPath: ir.OldPath, NewPath: ir.NewPath},
		}}, nil
	default:
		return nil, fmt.Errorf("unknown kind %q (expected file_move or import_rewrite)", e.Kind)
	}
}

func formatOperation(op *rewritev1.Operation) string {
	if op == nil {
		return "(nil)"
	}
	switch x := op.GetOp().(type) {
	case *rewritev1.Operation_FileMove:
		return fmt.Sprintf("file_move: %s -> %s", x.FileMove.GetFromPath(), x.FileMove.GetToPath())
	case *rewritev1.Operation_ImportRewrite:
		return fmt.Sprintf("import_rewrite: %s -> %s", x.ImportRewrite.GetOldPath(), x.ImportRewrite.GetNewPath())
	default:
		return "(unknown operation)"
	}
}

func formatResult(r *rewritev1.OperationResult) string {
	status := r.GetStatus().String()
	body := formatOperation(r.GetOperation())
	if msg := r.GetMessage(); msg != "" {
		return fmt.Sprintf("[%s] %s — %s", status, body, msg)
	}
	return fmt.Sprintf("[%s] %s", status, body)
}
