package capabilities

import (
	"context"
	"fmt"
	"os"

	"connectrpc.com/connect"

	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/capabilities"
	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/capabilities/capabilities_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"web-console/cli/internal/support"
)

// Register exposes `web-console capabilities` — a flat command covering
// both the full snapshot and the cheap `--liveness` probe. Calls
// Connect-RPC CapabilitiesService directly via the generated client.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Capabilities",
		Commands: []cliapp.Command{
			{
				Name:        "capabilities",
				Description: "Inspect runtime capabilities (use --liveness for a cheap probe)",
				NeedsAPI:    true,
				Run:         func(args []string) error { return run(core, args) },
			},
		},
	}
}

func newClient(core *cliapp.ScenarioApp) capabilitiesconnect.CapabilitiesServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return capabilitiesconnect.NewCapabilitiesServiceClient(httpClient, baseURL)
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("capabilities")
	liveness := fs.Bool("liveness", false, "Use the cheap liveness probe instead of the full snapshot")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	client := newClient(core)
	ctx := context.Background()

	var (
		caps      []*capabilitiesv1.CapabilityState
		timestamp string
		extras    []string
	)
	if *liveness {
		resp, err := client.Liveness(ctx, connect.NewRequest(&capabilitiesv1.LivenessRequest{}))
		if err != nil {
			return cliapp.WrapAPIError("capabilities liveness", err, nil)
		}
		caps = resp.Msg.GetCapabilities()
		timestamp = resp.Msg.GetTimestamp()
	} else {
		resp, err := client.Get(ctx, connect.NewRequest(&capabilitiesv1.GetRequest{}))
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
	if *liveness {
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
	if *jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, report)
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
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
