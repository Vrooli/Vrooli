package authoring

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	programsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs/programs_v1connect"
)

// evalTimeout bounds one full corpus run.
const evalTimeout = 30 * time.Minute

// Register exposes the versioned authoring corpus through the same typed
// service as program submission. The API deliberately returns unavailable
// when no code-authoring model route is configured; that state is meaningful
// evidence and must not be converted into a zero score by the CLI.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	client, base := cliapp.NewConnectHTTPClient(core)
	// One eval authors and executes every corpus case, so the wall time is a
	// multiple of a model round-trip rather than a single request. The shared
	// client deadline is sized for ordinary reads and cuts the run off before
	// it can produce a verdict, which reads as a transport failure rather than
	// as a measurement.
	client = &http.Client{Timeout: evalTimeout}
	h := programsconnect.NewProgramServiceClient(client, base)
	return cliapp.LoadFromManifestPrimitives(manifest, "authoring", map[string]cliapp.PrimitiveHandler{
		"vrooli.program_runtime.v1.programs.ProgramService.RunAuthoringEval": cliapp.ProtoList(func(ctx cliapp.OperationContext) (*programsv1.RunAuthoringEvalResponse, error) {
			r, err := h.RunAuthoringEval(context.Background(), connect.NewRequest(&programsv1.RunAuthoringEvalRequest{}))
			if err != nil {
				return nil, cliapp.WrapAPIError("run authoring eval", err, nil)
			}
			return r.Msg, nil
		}, func(_ cliapp.OperationContext, result *programsv1.RunAuthoringEvalResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Authoring eval: status=%s met=%d floor=%d.", result.GetStatus(), result.GetMet(), result.GetFloor())}}
		}),
	})
}
