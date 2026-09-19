package capabilities

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/capabilities"
	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/capabilities/capabilities_v1connect"

	"github.com/vrooli/cli-core/cliapp"

	"web-console/cli/internal/support"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client capabilitiesconnect.CapabilitiesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: capabilitiesconnect.NewCapabilitiesServiceClient(httpClient, baseURL),
	}
}

// Register exposes `web-console capabilities` — a flat command covering
// both the full snapshot and the cheap `--liveness` probe. Built from the
// embedded manifest; DefaultSubcommand preserves the flat invocation.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"CapabilitiesService.Get": h.run,
	}
	group, err := cliapp.LoadFromManifest(manifest, "capabilities", bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("capabilities: load from manifest: %w", err)
	}
	group.DefaultSubcommand = "capabilities"
	return group, nil
}

func (h *handlers) run(rc cliapp.RunContext) error {
	liveness := rc.BoolFlag("liveness")

	ctx := context.Background()

	var (
		caps      []*capabilitiesv1.CapabilityState
		timestamp string
		extras    []string
	)
	if liveness {
		resp, err := h.client.Liveness(ctx, connect.NewRequest(&capabilitiesv1.LivenessRequest{}))
		if err != nil {
			return cliapp.WrapAPIError("capabilities liveness", err, nil)
		}
		caps = resp.Msg.GetCapabilities()
		timestamp = resp.Msg.GetTimestamp()
	} else {
		resp, err := h.client.Get(ctx, connect.NewRequest(&capabilitiesv1.GetRequest{}))
		if err != nil {
			return cliapp.WrapAPIError("capabilities get", err, nil)
		}
		caps = resp.Msg.GetCapabilities()
		timestamp = resp.Msg.GetTimestamp()
		if def := resp.Msg.GetDefaultBackend(); def != "" {
			extras = append(extras, fmt.Sprintf("default_backend: %s", def))
		}
		for _, b := range resp.Msg.GetSessionBackends() {
			extras = append(extras, fmt.Sprintf("backend %s: available=%t survives_restart=%t", b.GetId(), b.GetAvailable(), b.GetSurvivesRestart()))
		}
	}

	heading := "Capabilities"
	if liveness {
		heading = "Capabilities (liveness)"
	}

	summary := []string{heading}
	if timestamp != "" {
		summary = append(summary, fmt.Sprintf("Checked at: %s", timestamp))
	}

	report := cliapp.OperationalReport{
		Status: summary,
		Triage: []cliapp.TriageGroup{
			{Heading: "Capabilities", Items: capabilityRows(caps)},
		},
		NextSteps: []string{fmt.Sprintf("%s capabilities --liveness", support.CLIName)},
	}
	if len(extras) > 0 {
		report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: "Backends", Items: extras})
	}
	if rc.JSON() {
		return cliapp.PrintReportJSON(rc.Stdout(), report)
	}
	return cliapp.RenderOperationalReport(rc.Stdout(), report)
}

func capabilityRows(caps []*capabilitiesv1.CapabilityState) []string {
	rows := make([]string, 0, len(caps))
	for _, c := range caps {
		line := fmt.Sprintf("%s [%s] — %s", c.GetId(), c.GetStatus(), c.GetName())
		if msg := c.GetMessage(); msg != "" {
			line = line + " — " + msg
		}
		rows = append(rows, line)
	}
	return rows
}
