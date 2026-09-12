package discovery

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	programsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs/programs_v1connect"
)

func discoveryEvalPasses(status string, floorMet bool) bool {
	return status == "met" && floorMet
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := programsconnect.NewProgramServiceClient(httpClient, baseURL)
	return cliapp.LoadFromManifestPrimitives(manifest, "discovery", map[string]cliapp.PrimitiveHandler{
		"vrooli.program_runtime.v1.programs.ProgramService.RunDiscoveryEval": cliapp.ProtoList(func(ctx cliapp.OperationContext) (*programsv1.RunDiscoveryEvalResponse, error) {
			r, err := client.RunDiscoveryEval(context.Background(), connect.NewRequest(&programsv1.RunDiscoveryEvalRequest{Suite: ctx.Flag("suite"), Mode: ctx.Flag("mode"), NoGate: ctx.BoolFlag("no-gate")}))
			if err != nil {
				return nil, cliapp.WrapAPIError("run discovery eval", err, nil)
			}
			if !ctx.BoolFlag("no-gate") && !discoveryEvalPasses(r.Msg.GetStatus(), r.Msg.GetFloorMet()) {
				return r.Msg, fmt.Errorf("discovery eval %s: met=%d floor=%d", r.Msg.GetStatus(), r.Msg.GetMet(), r.Msg.GetFloor())
			}
			return r.Msg, nil
		}, func(_ cliapp.OperationContext, r *programsv1.RunDiscoveryEvalResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Discovery eval: status=%s met=%d missed=%d wrong-selection=%d null-verdict=%d floor=%d.", r.GetStatus(), r.GetMet(), r.GetMissed(), r.GetWrongSelection(), r.GetNullVerdict(), r.GetFloor())}, ResultsHeading: "Cases", ResultCount: int(r.GetCases()), ListShaped: true}
		}),
	})
}
