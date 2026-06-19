package config

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	configv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config"
	configconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config/config_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client configconnect.ConfigServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: configconnect.NewConfigServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	resp, err := h.client.GetConfig(context.Background(), connect.NewRequest(&configv1.GetConfigRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get config", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Config == nil {
		return fmt.Errorf("server returned no config")
	}
	results := []string{formatConfig(resp.Msg.Config)}
	if resp.Msg.Readiness != nil {
		results = append(results, formatReadiness(resp.Msg.Readiness))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{"Fetched tunnel configuration."},
		ResultsHeading: "Config",
		Results:        results,
		RetrievalHints: []string{
			"`config sync --dry-run` — preview ingress reconciliation",
			"`config mode --target remote|local` — switch management mode",
		},
	})
}

func (h *handlers) sync(ctx cliapp.RunContext) error {
	dryRun := false
	if v := strings.TrimSpace(ctx.Flag("dry-run")); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("--dry-run must be true or false: %w", err)
		}
		dryRun = parsed
	}
	resp, err := h.client.Sync(context.Background(), connect.NewRequest(&configv1.SyncRequest{DryRun: dryRun}))
	if err != nil {
		return cliapp.WrapAPIError("sync ingress", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no sync response")
	}
	changes := make([]string, 0, len(resp.Msg.Added)+len(resp.Msg.Removed))
	for _, host := range resp.Msg.Added {
		changes = append(changes, "+ "+host)
	}
	for _, host := range resp.Msg.Removed {
		changes = append(changes, "- "+host)
	}
	result := fmt.Sprintf("Synced ingress (%s mode).", strings.ToLower(resp.Msg.Mode.String()))
	if resp.Msg.NoChanges {
		result = fmt.Sprintf("Ingress already in sync (%s mode); no changes.", strings.ToLower(resp.Msg.Mode.String()))
	} else if resp.Msg.SetupRequired {
		result = "Dry-run: remote mode setup is incomplete; no live ingress diff was attempted."
	} else if dryRun {
		result = fmt.Sprintf("Dry-run: %d added, %d removed (not applied).", len(resp.Msg.Added), len(resp.Msg.Removed))
	}
	if resp.Msg.Message != "" {
		result = result + " " + resp.Msg.Message
	}
	if len(resp.Msg.MissingFields) > 0 {
		changes = append(changes, "missing: "+strings.Join(resp.Msg.MissingFields, ", "))
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{result},
		Changes: changes,
	})
}

func (h *handlers) mode(ctx cliapp.RunContext) error {
	target, err := modeFlag(ctx.Flag("target"))
	if err != nil {
		return err
	}
	resp, err := h.client.SwitchMode(context.Background(), connect.NewRequest(&configv1.SwitchModeRequest{TargetMode: target}))
	if err != nil {
		return cliapp.WrapAPIError("switch mode", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no switch-mode response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Switched mode %s → %s.",
			strings.ToLower(resp.Msg.PreviousMode.String()), strings.ToLower(resp.Msg.CurrentMode.String()))},
	})
}

// modeFlag maps a --target value (remote|local) to the proto enum.
func modeFlag(v string) (configv1.Mode, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "remote":
		return configv1.Mode_MODE_REMOTE, nil
	case "local":
		return configv1.Mode_MODE_LOCAL, nil
	case "":
		return configv1.Mode_MODE_UNSPECIFIED, fmt.Errorf("--target is required (use remote or local)")
	default:
		return configv1.Mode_MODE_UNSPECIFIED, fmt.Errorf("unknown target %q (use remote or local)", v)
	}
}

func formatConfig(c *configv1.TunnelConfig) string {
	if c == nil {
		return "(nil)"
	}
	return fmt.Sprintf("mode=%s tunnel_id=%s account_id=%s cred_ref=%s prom=%s",
		strings.ToLower(c.Mode.String()), c.TunnelId, c.AccountId, c.CredRef, c.PromEndpoint)
}

func formatReadiness(r *configv1.ConfigReadiness) string {
	if r == nil {
		return "readiness=(nil)"
	}
	parts := []string{
		fmt.Sprintf("desired_mode=%s", strings.ToLower(r.DesiredMode.String())),
		fmt.Sprintf("remote_available=%t", r.RemoteAvailable),
		fmt.Sprintf("sync_ready=%t", r.SyncReady),
		fmt.Sprintf("credential_source=%s", r.CredentialSource),
		fmt.Sprintf("local_config_path=%s", r.LocalConfigPath),
	}
	if r.CredentialRef != "" {
		parts = append(parts, "credential_ref="+r.CredentialRef)
	}
	if len(r.MissingFields) > 0 {
		parts = append(parts, "missing="+strings.Join(r.MissingFields, ","))
	}
	if r.ModeReason != "" {
		parts = append(parts, "reason="+r.ModeReason)
	}
	return "readiness " + strings.Join(parts, " ")
}
