package capabilities

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/capabilities"
	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/capabilities/capabilities_v1connect"
)

type handlers struct {
	client capabilitiesconnect.CapabilityStatusServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: capabilitiesconnect.NewCapabilityStatusServiceClient(httpClient, baseURL)}
}

func (h *handlers) status(ctx cliapp.RunContext) error {
	resp, err := h.client.GetStatus(context.Background(), connect.NewRequest(&capabilitiesv1.GetStatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get capability status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no capability status")
	}
	rows := make([]string, 0, len(resp.Msg.Capabilities))
	for _, cap := range resp.Msg.Capabilities {
		if cap == nil {
			continue
		}
		blocker := cap.BlockingAxis
		if blocker == "" {
			blocker = cap.BlockingEvidence
		}
		rows = append(rows, fmt.Sprintf("%s: %s%s", cap.Id, cap.Status, func() string {
			if blocker == "" {
				return ""
			}
			return " — " + blocker
		}()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Provable capabilities: %d/%d.", resp.Msg.Provable, resp.Msg.Total)}, ResultsHeading: "Capability status", Results: rows})
}
