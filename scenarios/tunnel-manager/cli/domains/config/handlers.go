package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	configv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config"
	configconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config/config_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"
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

func (h *handlers) getCall(_ cliapp.OperationContext) (*configv1.GetConfigResponse, error) {
	resp, err := h.client.GetConfig(context.Background(), connect.NewRequest(&configv1.GetConfigRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get config", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Config == nil {
		return nil, fmt.Errorf("server returned no config")
	}
	return resp.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, msg *configv1.GetConfigResponse) cliapp.ListReport {
	results := []string{formatConfig(msg.Config)}
	if msg.Readiness != nil {
		results = append(results, formatReadiness(msg.Readiness))
	}
	return cliapp.ListReport{
		Summary:        []string{"Fetched tunnel configuration."},
		ResultsHeading: "Config",
		Results:        results,
		RetrievalHints: []string{
			"`config sync --dry-run` — preview ingress reconciliation",
			"`config mode --target remote|local` — switch management mode",
		},
	}
}

func (h *handlers) syncCall(ctx cliapp.OperationContext) (*configv1.SyncResponse, error) {
	dryRun := false
	if v := strings.TrimSpace(ctx.Flag("dry-run")); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("--dry-run must be true or false: %w", err)
		}
		dryRun = parsed
	}
	prune := false
	if v := strings.TrimSpace(ctx.Flag("prune")); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("--prune must be true or false: %w", err)
		}
		prune = parsed
	}
	resp, err := h.client.Sync(context.Background(), connect.NewRequest(&configv1.SyncRequest{DryRun: dryRun, Prune: prune}))
	if err != nil {
		return nil, cliapp.WrapAPIError("sync ingress", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no sync response")
	}
	return resp.Msg, nil
}

func (h *handlers) syncReport(ctx cliapp.OperationContext, msg *configv1.SyncResponse) cliapp.MutationReport {
	dryRun := false
	if v := strings.TrimSpace(ctx.Flag("dry-run")); v != "" {
		dryRun, _ = strconv.ParseBool(v)
	}
	changes := make([]string, 0, len(msg.Added)+len(msg.Pruned)+len(msg.DriftUnmanaged)+len(msg.Orphaned))
	for _, host := range msg.Added {
		changes = append(changes, "+ "+host)
	}
	for _, host := range msg.Pruned {
		changes = append(changes, "- "+host)
	}
	for _, host := range msg.Orphaned {
		changes = append(changes, "orphaned (prune candidate): "+host)
	}
	for _, host := range msg.DriftUnmanaged {
		changes = append(changes, "drift/unmanaged (left intact): "+host)
	}
	result := fmt.Sprintf("Synced ingress additively (%s mode).", strings.ToLower(msg.Mode.String()))
	if msg.NoChanges {
		result = fmt.Sprintf("Ingress already in sync (%s mode); no changes.", strings.ToLower(msg.Mode.String()))
	} else if msg.SetupRequired {
		result = "Dry-run: remote mode setup is incomplete; no live ingress diff was attempted."
	} else if dryRun {
		result = fmt.Sprintf("Dry-run: %d to add, %d to prune, %d unmanaged (not applied).", len(msg.Added), len(msg.Pruned), len(msg.DriftUnmanaged))
	}
	if msg.Message != "" {
		result = result + " " + msg.Message
	}
	if len(msg.MissingFields) > 0 {
		changes = append(changes, "missing: "+strings.Join(msg.MissingFields, ", "))
	}
	return cliapp.MutationReport{
		Result:  []string{result},
		Changes: changes,
	}
}

func (h *handlers) credentialsStatusCall(ctx cliapp.OperationContext) (proto.Message, error) {
	if verify, _ := strconv.ParseBool(strings.TrimSpace(ctx.Flag("verify"))); verify {
		resp, err := h.client.VerifyCredentials(context.Background(), connect.NewRequest(&configv1.VerifyCredentialsRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("verify credentials", err, nil)
		}
		if resp == nil || resp.Msg == nil {
			return nil, fmt.Errorf("server returned no verification response")
		}
		return resp.Msg, nil
	}
	resp, err := h.client.GetCredentialStatus(context.Background(), connect.NewRequest(&configv1.GetCredentialStatusRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get credential status", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Status == nil {
		return nil, fmt.Errorf("server returned no credential status")
	}
	return resp.Msg, nil
}

func (h *handlers) credentialsStatusReport(_ cliapp.OperationContext, message proto.Message) cliapp.ListReport {
	switch msg := message.(type) {
	case *configv1.GetCredentialStatusResponse:
		return cliapp.ListReport{
			Summary:        []string{credentialStatusSummary(msg.Status)},
			ResultsHeading: "Credential fields",
			Results:        formatCredentialFields(msg.Status.Fields),
			RetrievalHints: []string{
				"`config credentials-status --verify` — run LIVE Cloudflare scope checks (token, account, tunnel, DNS:Edit)",
				"`printf '%s' <token> | config credentials-set --account-id <id> --tunnel-id <id> --api-token-stdin` — save a missing API token without argv exposure",
				"`config sync --dry-run` — preview ingress reconciliation after credentials are ready",
			},
		}
	case *configv1.VerifyCredentialsResponse:
		summary := "Live Cloudflare credential checks: all OK — DNS automation is unblocked."
		if !msg.Ready {
			summary = "Live Cloudflare credential checks found issues (see remediation below)."
		}
		return cliapp.ListReport{
			Summary:        []string{summary},
			ResultsHeading: "Credential checks",
			Results:        formatCredentialChecks(msg.Checks),
			RetrievalHints: []string{"`printf '%s' <token> | config credentials-set --api-token-stdin` — store a re-issued token with the missing scope without argv exposure"},
		}
	default:
		return cliapp.ListReport{Summary: []string{"Credential status response unavailable."}}
	}
}

func (h *handlers) bootstrapCall(ctx cliapp.OperationContext) (*configv1.BootstrapCloudflareResponse, error) {
	if !ctx.BoolFlag("api-token-stdin") {
		return nil, fmt.Errorf("--api-token-stdin is required; provide the token on standard input so it never appears in argv")
	}
	apiToken, err := readSecretFromStdin("Cloudflare API token")
	if err != nil {
		return nil, err
	}
	request := &configv1.BootstrapCloudflareRequest{
		ApiToken:   apiToken,
		AccountId:  strings.TrimSpace(ctx.Flag("account-id")),
		TunnelId:   strings.TrimSpace(ctx.Flag("tunnel-id")),
		TunnelName: strings.TrimSpace(ctx.Flag("tunnel-name")),
	}
	request.DryRun = ctx.BoolFlag("dry-run")
	resp, err := h.client.BootstrapCloudflare(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("bootstrap Cloudflare tunnel", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no bootstrap response")
	}
	return resp.Msg, nil
}

func (h *handlers) bootstrapReport(ctx cliapp.OperationContext, msg *configv1.BootstrapCloudflareResponse) cliapp.MutationReport {
	result := fmt.Sprintf("Cloudflare tunnel %s resolved in account %s.", msg.TunnelId, msg.AccountId)
	if msg.Adopted {
		result = "Adopted existing Cloudflare tunnel. " + result
	} else if msg.Created {
		result = "Created Cloudflare tunnel. " + result
	}
	if ctx.BoolFlag("dry-run") {
		result = "Dry-run: " + result + " No credentials were written."
	} else if msg.Written {
		result += " Authority-backed credentials were written."
	}
	return cliapp.MutationReport{Result: []string{result}}
}

func readSecretFromStdin(label string) (string, error) {
	return readSecret(os.Stdin, label)
}

func readSecret(reader io.Reader, label string) (string, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, 4097))
	if err != nil {
		return "", fmt.Errorf("read %s from standard input: %w", label, err)
	}
	value := strings.TrimSpace(string(raw))
	if value == "" {
		return "", fmt.Errorf("%s was empty on standard input", label)
	}
	if len(raw) > 4096 {
		return "", fmt.Errorf("%s exceeds the maximum accepted length", label)
	}
	return value, nil
}

func (h *handlers) credentialsSetCall(ctx cliapp.OperationContext) (*configv1.SetCloudflareCredentialsResponse, error) {
	apiToken := ""
	if ctx.BoolFlag("api-token-stdin") {
		var err error
		apiToken, err = readSecretFromStdin("Cloudflare API token")
		if err != nil {
			return nil, err
		}
	}
	req := &configv1.SetCloudflareCredentialsRequest{
		AccountId: strings.TrimSpace(ctx.Flag("account-id")),
		TunnelId:  strings.TrimSpace(ctx.Flag("tunnel-id")),
		ApiToken:  apiToken,
	}
	if req.AccountId == "" && req.TunnelId == "" && req.ApiToken == "" {
		return nil, fmt.Errorf("at least one credential flag is required")
	}
	resp, err := h.client.SetCloudflareCredentials(context.Background(), connect.NewRequest(req))
	if err != nil {
		return nil, cliapp.WrapAPIError("set Cloudflare credentials", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Status == nil {
		return nil, fmt.Errorf("server returned no credential status")
	}
	return resp.Msg, nil
}

func (h *handlers) credentialsSetReport(_ cliapp.OperationContext, msg *configv1.SetCloudflareCredentialsResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result: []string{credentialStatusSummary(msg.Status)},
		Changes: append([]string{
			"Saved provided credential fields; token value will not be shown again.",
		}, formatCredentialFields(msg.Status.Fields)...),
	}
}

func (h *handlers) credentialsClearCall(ctx cliapp.OperationContext) (*configv1.ClearCloudflareCredentialsResponse, error) {
	field := strings.TrimSpace(ctx.Flag("field"))
	fields := []string{"all"}
	if field != "" {
		fields = []string{field}
	}
	resp, err := h.client.ClearCloudflareCredentials(context.Background(), connect.NewRequest(&configv1.ClearCloudflareCredentialsRequest{
		Fields: fields,
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("clear Cloudflare credentials", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Status == nil {
		return nil, fmt.Errorf("server returned no credential status")
	}
	return resp.Msg, nil
}

func (h *handlers) credentialsClearReport(_ cliapp.OperationContext, msg *configv1.ClearCloudflareCredentialsResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result:  []string{credentialStatusSummary(msg.Status)},
		Changes: formatCredentialFields(msg.Status.Fields),
	}
}

func (h *handlers) modeCall(ctx cliapp.OperationContext) (*configv1.SwitchModeResponse, error) {
	target, err := modeFlag(ctx.Flag("target"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.SwitchMode(context.Background(), connect.NewRequest(&configv1.SwitchModeRequest{TargetMode: target}))
	if err != nil {
		return nil, cliapp.WrapAPIError("switch mode", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no switch-mode response")
	}
	return resp.Msg, nil
}

func (h *handlers) modeReport(_ cliapp.OperationContext, msg *configv1.SwitchModeResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Switched mode %s → %s.",
		strings.ToLower(msg.PreviousMode.String()), strings.ToLower(msg.CurrentMode.String()))}}
}

// publicExposure flips the global /public Access-bypass switch. Exactly one
// of --on/--off must be supplied; the resulting global state is rendered.
func (h *handlers) publicExposureCall(ctx cliapp.OperationContext) (*configv1.SetPublicExposureResponse, error) {
	on := ctx.BoolFlag("on")
	off := ctx.BoolFlag("off")
	switch {
	case on && off:
		return nil, fmt.Errorf("--on and --off are mutually exclusive")
	case !on && !off:
		return nil, fmt.Errorf("one of --on or --off is required")
	}
	enabled := on
	resp, err := h.client.SetPublicExposure(context.Background(), connect.NewRequest(&configv1.SetPublicExposureRequest{Enabled: enabled}))
	if err != nil {
		return nil, cliapp.WrapAPIError("set public exposure", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Config == nil {
		return nil, fmt.Errorf("server returned no config")
	}
	return resp.Msg, nil
}

func (h *handlers) publicExposureReport(_ cliapp.OperationContext, msg *configv1.SetPublicExposureResponse) cliapp.MutationReport {
	state := "disabled"
	if msg.Config.PublicExposureEnabled {
		state = "enabled"
	}
	return cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Global /public Access-bypass exposure is now %s.", state)},
		Changes: []string{formatConfig(msg.Config)},
		NextCommand: []string{
			"`access status --dry-run` — preview the Bypass apps that would be created/removed",
			"`access status` — show effective per-host bypass state",
		},
	}
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
