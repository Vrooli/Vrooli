package settings

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"

	settingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/settings"
	settingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/settings/settings_v1connect"

	"github.com/vrooli/cli-core/cliapp"

	"web-console/cli/internal/support"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client settingsconnect.SettingsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: settingsconnect.NewSettingsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	resp, err := h.client.GetSessionDefaults(context.Background(), connect.NewRequest(&settingsv1.GetSessionDefaultsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get session defaults", err, nil)
	}

	report := cliapp.ListReport{
		Summary:        []string{"Session defaults"},
		ResultsHeading: "Values",
		Results:        defaultsRows(resp.Msg.GetDefaults()),
		RetrievalHints: []string{fmt.Sprintf("%s settings session-defaults-set --body-file defaults.json", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderListReport(ctx.Stdout(), report)
}

func (h *handlers) set(ctx cliapp.RunContext) error {
	payload, err := support.ReadJSONFile(ctx.Flag("body-file"), true)
	if err != nil {
		return err
	}

	// Map the legacy JSON body shape (default_backend + default_policy)
	// onto the proto request. Unknown fields are ignored; the wire-level
	// validation is server-side.
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}
	var legacy struct {
		DefaultBackend *string `json:"default_backend,omitempty"`
		DefaultPolicy  *struct {
			Mode     string `json:"mode"`
			Duration string `json:"duration,omitempty"`
		} `json:"default_policy,omitempty"`
	}
	if err := json.Unmarshal(body, &legacy); err != nil {
		return fmt.Errorf("decode body: %w", err)
	}

	req := &settingsv1.UpdateSessionDefaultsRequest{}
	if legacy.DefaultBackend != nil {
		v := *legacy.DefaultBackend
		req.DefaultBackend = &v
	}
	if legacy.DefaultPolicy != nil {
		req.DefaultPolicy = &settingsv1.ExpirationPolicy{
			Mode:     legacy.DefaultPolicy.Mode,
			Duration: legacy.DefaultPolicy.Duration,
		}
	}

	if _, err := h.client.UpdateSessionDefaults(context.Background(), connect.NewRequest(req)); err != nil {
		return cliapp.WrapAPIError("update session defaults", err, nil)
	}

	report := cliapp.MutationReport{
		Result:      []string{"Updated session defaults"},
		NextCommand: []string{fmt.Sprintf("%s settings session-defaults-get", support.CLIName)},
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), report)
	}
	return cliapp.RenderMutationReport(ctx.Stdout(), report)
}

func defaultsRows(d *settingsv1.SessionDefaults) []string {
	if d == nil {
		return nil
	}
	rows := []string{
		fmt.Sprintf("default_backend: %s", d.GetDefaultBackend()),
	}
	if p := d.GetDefaultPolicy(); p != nil {
		rows = append(rows,
			fmt.Sprintf("default_policy.mode: %s", p.GetMode()),
			fmt.Sprintf("default_policy.duration: %s", p.GetDuration()),
		)
	}
	return rows
}
