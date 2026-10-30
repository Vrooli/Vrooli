package packaged

import (
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp, _ []byte) (cliapp.SubcommandGroup, error) {
	client, base := cliapp.NewConnectHTTPClientWithTimeout(core, 65*time.Minute)
	h := handlers{client: client, base: base}
	return cliapp.SubcommandGroup{Name: "packaged", Description: "Capture and compare packaged desktop launch measurements", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "capture", Description: "Capture packaged launch samples for one scenario", NeedsAPI: true, Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "plan", Bool: true, Description: "Validate and show planned work without launching or recording samples"}, {Name: "scenario", Required: true, Description: "Scenario slug"}, {Name: "artifact", Required: true, Description: "Built desktop artifact path"}, {Name: "ramp", Default: "desktop", Values: []string{"desktop"}}, {Name: "strategy", Default: "electron-sidecar", Values: []string{"electron-sidecar", "electron-optimized"}}, {Name: "shared-runtime-readiness", Bool: true, Description: "Enable parallel independent-service startup"}, {Name: "secure-custom-scheme-ui", Bool: true, Description: "Enable the secure custom-scheme UI path"}, {Name: "v8-snapshot-code-cache", Bool: true, Description: "Enable the V8 snapshot/code-cache profile toggle"}, {Name: "electron-fuses", Bool: true, Description: "Enable Electron fuse hardening"}, {Name: "runs", Default: "10", Description: "Retained samples; adds one discarded warmup. Each operation normally starts protocol and demo processes"}, {Name: "condition", Required: true, Values: []string{"cold", "warm"}}, {Name: "experiment", Description: "Experiment ID; generated when omitted"}, {Name: "journey", Description: "Reference journey ID"}}}, RunCtx: h.capture},
		{Name: "report", Description: "Summarize comparable packaged launch samples", NeedsAPI: true, Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "scenario"}, {Name: "ramp", Default: "desktop"}, {Name: "strategy", Description: "Filter to one strategy; omitted includes all strategies"}, {Name: "condition", Values: []string{"cold", "warm"}}, {Name: "experiment"}}}, RunCtx: h.report},
		{Name: "compare", Description: "Compare two packaged launch experiment IDs", NeedsAPI: true, Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "a", Required: true}, {Name: "b", Required: true}}}, RunCtx: h.compare},
	}}, nil
}
