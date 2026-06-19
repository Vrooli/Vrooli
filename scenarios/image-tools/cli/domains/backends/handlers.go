package backends

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
	modelsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models/models_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client modelsconnect.ModelsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		client: modelsconnect.NewModelsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) doctor(ctx cliapp.RunContext) error {
	resp, err := h.client.DoctorBackends(context.Background(), connect.NewRequest(&modelsv1.DoctorBackendsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("doctor backends", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no backend doctor response")
	}
	results := make([]string, 0, len(resp.Msg.Backends))
	for _, b := range resp.Msg.Backends {
		results = append(results, formatBackend(b))
	}
	if len(results) == 0 {
		results = append(results, "no registered backends")
	}
	status := "Backend doctor passed."
	if !resp.Msg.Ok {
		status = fmt.Sprintf("Backend doctor found %d backend provisioning issue(s).", countUnavailableLocal(resp.Msg.Backends))
	}
	if err := cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{status},
		ResultsHeading: "Backends",
		Results:        results,
		RetrievalHints: []string{
			"`models select <operation>` — preview model and backend selection for an operation",
			"Provision missing local backends through Scenario Dependency Analyzer; do not run raw package managers.",
		},
	}); err != nil {
		return err
	}
	if !resp.Msg.Ok {
		return fmt.Errorf("backend doctor failed")
	}
	return nil
}

func formatBackend(b *modelsv1.BackendStatus) string {
	availability := "missing"
	if b.Available {
		availability = "ready"
	}
	tier := "local"
	if b.Cloud {
		tier = "byok-cloud"
	} else if !b.Standalone {
		tier = "local-comfyui"
	}
	gpu := "cpu-only"
	if b.GpuCapable {
		gpu = "gpu-capable"
	}
	return fmt.Sprintf("%s [%s, %s, %s] ops=%s — %s; provision: %s",
		b.Name,
		availability,
		tier,
		gpu,
		strings.Join(b.Operations, ","),
		b.Detail,
		b.Provision,
	)
}

func countUnavailableLocal(backends []*modelsv1.BackendStatus) int {
	count := 0
	for _, b := range backends {
		if !b.Available && !b.Cloud {
			count++
		}
	}
	return count
}
