package ops

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
	opsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops/ops_v1connect"
)

type handlers struct {
	core *cliapp.ScenarioApp
}

func newHandlers(core *cliapp.ScenarioApp) *handlers { return &handlers{core: core} }

// list mirrors OpsService.ListOperations (the Connect discovery RPC).
func (h *handlers) list(ctx cliapp.RunContext) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	client := opsconnect.NewOpsServiceClient(httpClient, baseURL)
	resp, err := client.ListOperations(context.Background(), connect.NewRequest(&opsv1.ListOperationsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list operations", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Operations))
	for _, op := range resp.Msg.Operations {
		results = append(results, fmt.Sprintf("%-10s [%s] %s", op.Name, op.Category, op.Summary))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d deterministic operations.", len(resp.Msg.Operations))},
		ResultsHeading: "Operations",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("decodable: %v", resp.Msg.DecodableFormats),
			fmt.Sprintf("encodable: %v", resp.Msg.EncodableFormats),
		},
	})
}

func (h *handlers) importLink(ctx cliapp.RunContext) error {
	rawURL := ctx.Positional("url")
	if rawURL == "" {
		return fmt.Errorf("a public image URL is required")
	}
	result, err := importImageLink(h.core, rawURL)
	if err != nil {
		return err
	}
	return ctx.RenderMutation(cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Imported %s (%dx%d %s)", result.InputRef, result.Width, result.Height, result.Format)},
		Changes: []string{fmt.Sprintf("input_ref=%s", result.InputRef)},
	})
}

// run wraps the common flow: build params from the operation's generated flag
// bindings, call the REST edge, write the output.
func (h *handlers) run(spec runSpec) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		input := ctx.Positional("input")
		if input == "" {
			return fmt.Errorf("an input image path is required")
		}
		// Only the overlay operation declares an --overlay (image watermark) flag.
		overlay := ""
		if ctx.FlagDeclared("overlay") {
			overlay = ctx.Flag("overlay")
		}
		params, err := buildParams(spec, ctx)
		if err != nil {
			return err
		}
		// Derive the output encoding from the --out extension so any op honors
		// `--out result.webp`. convert/compress still take an explicit --format.
		res, err := runOp(h.core, spec.Op, input, overlay, extToFormat(ctx.Flag("out")), params)
		if err != nil {
			return err
		}
		return writeOutput(ctx, res, ctx.Flag("out"))
	}
}
