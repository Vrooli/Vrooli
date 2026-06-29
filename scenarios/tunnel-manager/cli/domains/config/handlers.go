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
	prune := false
	if v := strings.TrimSpace(ctx.Flag("prune")); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("--prune must be true or false: %w", err)
		}
		prune = parsed
	}
	resp, err := h.client.Sync(context.Background(), connect.NewRequest(&configv1.SyncRequest{DryRun: dryRun, Prune: prune}))
	if err != nil {
		return cliapp.WrapAPIError("sync ingress", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no sync response")
	}
	changes := make([]string, 0, len(resp.Msg.Added)+len(resp.Msg.Pruned)+len(resp.Msg.DriftUnmanaged)+len(resp.Msg.Orphaned))
	for _, host := range resp.Msg.Added {
		changes = append(changes, "+ "+host)
	}
	for _, host := range resp.Msg.Pruned {
		changes = append(changes, "- "+host)
	}
	for _, host := range resp.Msg.Orphaned {
		changes = append(changes, "orphaned (prune candidate): "+host)
	}
	for _, host := range resp.Msg.DriftUnmanaged {
		changes = append(changes, "drift/unmanaged (left intact): "+host)
	}
	result := fmt.Sprintf("Synced ingress additively (%s mode).", strings.ToLower(resp.Msg.Mode.String()))
	if resp.Msg.NoChanges {
		result = fmt.Sprintf("Ingress already in sync (%s mode); no changes.", strings.ToLower(resp.Msg.Mode.String()))
	} else if resp.Msg.SetupRequired {
		result = "Dry-run: remote mode setup is incomplete; no live ingress diff was attempted."
	} else if dryRun {
		result = fmt.Sprintf("Dry-run: %d to add, %d to prune, %d unmanaged (not applied).", len(resp.Msg.Added), len(resp.Msg.Pruned), len(resp.Msg.DriftUnmanaged))
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

func (h *handlers) credentialsStatus(ctx cliapp.RunContext) error {
	if verify, _ := strconv.ParseBool(strings.TrimSpace(ctx.Flag("verify"))); verify {
		return h.credentialsVerify(ctx)
	}
	resp, err := h.client.GetCredentialStatus(context.Background(), connect.NewRequest(&configv1.GetCredentialStatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get credential status", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Status == nil {
		return fmt.Errorf("server returned no credential status")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{credentialStatusSummary(resp.Msg.Status)},
		ResultsHeading: "Credential fields",
		Results:        formatCredentialFields(resp.Msg.Status.Fields),
		RetrievalHints: []string{
			"`config credentials-status --verify` — run LIVE Cloudflare scope checks (token, account, tunnel, DNS:Edit)",
			"`config credentials-set --account-id <id> --tunnel-id <id> --api-token <token>` — save missing credentials",
			"`config sync --dry-run` — preview ingress reconciliation after credentials are ready",
		},
	})
}

// credentialsVerify runs the live VerifyCredentials probe and renders the
// per-check verdict. Distinct from the presence-only default path so the
// readiness fast-path never makes a network call.
func (h *handlers) credentialsVerify(ctx cliapp.RunContext) error {
	resp, err := h.client.VerifyCredentials(context.Background(), connect.NewRequest(&configv1.VerifyCredentialsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("verify credentials", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no verification response")
	}
	summary := "Live Cloudflare credential checks: all OK — DNS automation is unblocked."
	if !resp.Msg.Ready {
		summary = "Live Cloudflare credential checks found issues (see remediation below)."
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Credential checks",
		Results:        formatCredentialChecks(resp.Msg.Checks),
		RetrievalHints: []string{
			"`config credentials-set --api-token <token>` — store a re-issued token with the missing scope",
		},
	})
}

func (h *handlers) credentialsSet(ctx cliapp.RunContext) error {
	req := &configv1.SetCloudflareCredentialsRequest{
		AccountId: strings.TrimSpace(ctx.Flag("account-id")),
		TunnelId:  strings.TrimSpace(ctx.Flag("tunnel-id")),
		ApiToken:  strings.TrimSpace(ctx.Flag("api-token")),
	}
	if req.AccountId == "" && req.TunnelId == "" && req.ApiToken == "" {
		return fmt.Errorf("at least one credential flag is required")
	}
	resp, err := h.client.SetCloudflareCredentials(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("set Cloudflare credentials", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Status == nil {
		return fmt.Errorf("server returned no credential status")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{credentialStatusSummary(resp.Msg.Status)},
		Changes: append([]string{
			"Saved provided credential fields; token value will not be shown again.",
		}, formatCredentialFields(resp.Msg.Status.Fields)...),
	})
}

func (h *handlers) credentialsClear(ctx cliapp.RunContext) error {
	field := strings.TrimSpace(ctx.Flag("field"))
	fields := []string{"all"}
	if field != "" {
		fields = []string{field}
	}
	resp, err := h.client.ClearCloudflareCredentials(context.Background(), connect.NewRequest(&configv1.ClearCloudflareCredentialsRequest{
		Fields: fields,
	}))
	if err != nil {
		return cliapp.WrapAPIError("clear Cloudflare credentials", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Status == nil {
		return fmt.Errorf("server returned no credential status")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{credentialStatusSummary(resp.Msg.Status)},
		Changes: formatCredentialFields(resp.Msg.Status.Fields),
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

// publicExposure flips the global /public Access-bypass switch. Exactly one
// of --on/--off must be supplied; the resulting global state is rendered.
func (h *handlers) publicExposure(ctx cliapp.RunContext) error {
	on := ctx.BoolFlag("on")
	off := ctx.BoolFlag("off")
	switch {
	case on && off:
		return fmt.Errorf("--on and --off are mutually exclusive")
	case !on && !off:
		return fmt.Errorf("one of --on or --off is required")
	}
	enabled := on
	resp, err := h.client.SetPublicExposure(context.Background(), connect.NewRequest(&configv1.SetPublicExposureRequest{Enabled: enabled}))
	if err != nil {
		return cliapp.WrapAPIError("set public exposure", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Config == nil {
		return fmt.Errorf("server returned no config")
	}
	state := "disabled"
	if resp.Msg.Config.PublicExposureEnabled {
		state = "enabled"
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Global /public Access-bypass exposure is now %s.", state)},
		Changes: []string{formatConfig(resp.Msg.Config)},
		NextCommand: []string{
			"`access status --dry-run` — preview the Bypass apps that would be created/removed",
			"`access status` — show effective per-host bypass state",
		},
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
	return fmt.Sprintf("mode=%s tunnel_id=%s account_id=%s cred_ref=%s prom=%s public_exposure=%t",
		strings.ToLower(c.Mode.String()), c.TunnelId, c.AccountId, c.CredRef, c.PromEndpoint, c.PublicExposureEnabled)
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

func credentialStatusSummary(status *configv1.CredentialStatus) string {
	if status == nil {
		return "Cloudflare credential status unavailable."
	}
	if status.Ready {
		return fmt.Sprintf("Cloudflare credentials are ready (source=%s).", status.Source)
	}
	if len(status.MissingFields) == 0 {
		return fmt.Sprintf("Cloudflare credentials are incomplete (source=%s).", status.Source)
	}
	return fmt.Sprintf("Cloudflare credentials are incomplete; missing %s.", strings.Join(status.MissingFields, ", "))
}

// formatCredentialChecks renders each live verification result as a one-line
// "name: STATE — detail (remediation)" string. The state is upper-cased so an
// OK run reads at a glance and any non-OK verdict stands out.
func formatCredentialChecks(checks []*configv1.CredentialCheck) []string {
	if len(checks) == 0 {
		return []string{"checks=(none)"}
	}
	out := make([]string, 0, len(checks))
	for _, c := range checks {
		if c == nil {
			continue
		}
		line := fmt.Sprintf("%s: %s", c.Name, checkStateLabel(c.State))
		if c.Detail != "" {
			line += " — " + c.Detail
		}
		if c.State != configv1.CheckState_CHECK_STATE_OK && c.Remediation != "" {
			line += " → " + c.Remediation
		}
		out = append(out, line)
	}
	return out
}

func checkStateLabel(s configv1.CheckState) string {
	switch s {
	case configv1.CheckState_CHECK_STATE_OK:
		return "OK"
	case configv1.CheckState_CHECK_STATE_MISSING:
		return "MISSING"
	case configv1.CheckState_CHECK_STATE_INVALID:
		return "INVALID"
	case configv1.CheckState_CHECK_STATE_INSUFFICIENT_SCOPE:
		return "INSUFFICIENT_SCOPE"
	default:
		return "UNSPECIFIED"
	}
}

func formatCredentialFields(fields []*configv1.CredentialFieldStatus) []string {
	if len(fields) == 0 {
		return []string{"credential_fields=(none)"}
	}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if field == nil {
			continue
		}
		state := "missing"
		if field.Present {
			state = "present"
		}
		writable := "read-only"
		if field.Writable {
			writable = "writable"
		}
		parts := []string{field.Name, state, "source=" + field.Source, writable}
		if field.Ref != "" {
			parts = append(parts, "ref="+field.Ref)
		}
		out = append(out, strings.Join(parts, " "))
	}
	return out
}
