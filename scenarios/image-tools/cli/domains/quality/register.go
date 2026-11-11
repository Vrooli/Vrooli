package quality

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	qualityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality"
	qualityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality/quality_v1connect"
)

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	profiles := func(ctx cliapp.RunContext) error {
		hc, base := cliapp.NewConnectHTTPClient(core)
		resp, err := qualityconnect.NewQualityServiceClient(hc, base).ListQualityProfiles(context.Background(), connect.NewRequest(&qualityv1.ListQualityProfilesRequest{}))
		if err != nil {
			return err
		}
		lines := []string{}
		for _, p := range resp.Msg.Profiles {
			lines = append(lines, p.Id+": "+p.Description)
		}
		return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{resp.Msg.EvaluatorVersion}, Results: lines})
	}
	group, err := cliapp.LoadFromManifest(manifest, "quality", map[string]func(cliapp.RunContext) error{"QualityService.ListQualityProfiles": profiles})
	if err != nil {
		return group, fmt.Errorf("quality: %w", err)
	}
	group.Subcommands = append(group.Subcommands, cliapp.Command{Name: "evaluate", Description: "Score an output-set manifest with reference metrics, local judge and regression budgets", NeedsAPI: true, Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "manifest", Required: true, Description: "Evaluation manifest JSON; image paths are relative to this file"}}, Flags: []cliapp.Flag{{Name: "baseline", Description: "Prior immutable EvaluationReport to compare"}, {Name: "out", Description: "Write the complete report JSON here"}, {Name: "report-only", Description: "Return success even when quality budgets fail (report still records failure)", Bool: true}}}, RunCtx: func(ctx cliapp.RunContext) error { return evaluateCommand(core, ctx) }})
	group.Subcommands = append(group.Subcommands, cliapp.Command{Name: "caption-compare", Description: "Run the frozen C07 corpus on two installed CPU models and produce paired promotion reports; invoke only after the owner confirms fresh admission is safe (stops on first refused job)", NeedsAPI: true, Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "corpus", Required: true, Description: "Frozen C07 corpus JSON (normally api/internal/quality/testdata/caption-v1/corpus.json)"}}, Flags: []cliapp.Flag{{Name: "models", Description: "Exactly two installed model ids, ordered baseline,candidate"}, {Name: "out-dir", Description: "New or empty directory for immutable candidate manifests, run receipts and reports"}}}, RunCtx: func(ctx cliapp.RunContext) error { return captionCompareCommand(core, ctx) }})
	return group, nil
}
