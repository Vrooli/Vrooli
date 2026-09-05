package onboard

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/cleanup"
	cleanupconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/cleanup/cleanup_v1connect"
	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines"
	machinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines/machines_v1connect"
	onboardv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard"
	onboardconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard/onboard_v1connect"
	"github.com/vrooli/vrooli/packages/proto/privilegedops"

	"github.com/vrooli/cli-core/cliapp"
	"vrooli-bridge/cli/internal/operatorauth"
	"vrooli-bridge/cli/internal/session"
)

// defaultRevision is the presentation/value for "bring the node to the control
// plane's current commit". The server accepts empty or "@cp" identically
// (phase 6); the CLI sends "@cp" explicitly so the intent is visible on the wire
// and in `onboard start --help`.
const defaultRevision = "@cp"

// handlers bundles the Connect client and the (injectable) SSH-password source.
type handlers struct {
	core          *cliapp.ScenarioApp
	client        onboardconnect.OnboardServiceClient
	machines      machineCreator
	cleanup       cleanupconnect.CleanupServiceClient
	password      passwordSource
	authorization io.Reader
}

type machineCreator interface {
	CreateMachine(context.Context, *connect.Request[machinesv1.CreateMachineRequest]) (*connect.Response[machinesv1.CreateMachineResponse], error)
}

func (h *handlers) preflightConnect(ctx cliapp.RunContext) error {
	host := strings.TrimSpace(ctx.Flag("host"))
	user := strings.TrimSpace(ctx.Flag("user"))
	port := parseInt(ctx.Flag("port"))
	if port == 0 {
		port = 22
	}
	// Keep progress on stderr so --json remains machine-readable on stdout.
	// The server-owned start + wait can take several minutes (especially for a
	// working-tree transfer), and silence makes an interactive operator unsure
	// whether the CLI is alive or ready for input. Explicitly state that no
	// input is expected until the post-enrollment protection prompt appears.
	fmt.Fprintf(ctx.Stderr(), "Bridge: checking trust and preparing onboarding for %s (no input is needed yet)\n", host)
	preflight, err := h.client.PreflightOnboarding(context.Background(), connect.NewRequest(&onboardv1.PreflightOnboardingRequest{
		Host: host, Port: int32(port), User: user, MachineId: strings.TrimSpace(ctx.Flag("machine-id")),
	}))
	if err != nil {
		return wrapOnboardAPIError("preflight onboarding connection", err)
	}
	if preflight == nil || preflight.Msg == nil {
		return fmt.Errorf("server returned no onboarding preflight")
	}
	decision := preflight.Msg.Decision
	if decision == onboardv1.ConnectDecision_CONNECT_DECISION_AMBIGUOUS || decision == onboardv1.ConnectDecision_CONNECT_DECISION_HOST_KEY_REVIEW {
		return fmt.Errorf("cannot connect %s: %s", connectDecisionLabel(decision), preflight.Msg.Message)
	}

	password := ""
	credSource := credentialNone
	setupPassphraseStdin := ctx.BoolFlag("setup-passphrase-stdin")
	if setupPassphraseStdin {
		return fmt.Errorf("--setup-passphrase-stdin is retired: the target setup queues credential-store protection for vrooli-onboarding")
	}
	fromStdin := ctx.BoolFlag("password-stdin")
	if preflight.Msg.PasswordRequired {
		var resolveErr error
		if setupPassphraseStdin {
			password, resolveErr = h.password.resolveLine()
			credSource = credentialFromStdin
		} else if fromStdin {
			var source credentialSource
			password, source, resolveErr = h.password.resolve(user, host, true, false)
			credSource = source
		} else if _, ok := h.password.lookupEnv(sshPasswordEnvVar); ok {
			password, credSource, resolveErr = h.password.resolve(user, host, false, false)
		} else {
			// Connect is intentionally different from start: the server has already
			// proven that a credential is required, so the interactive prompt is
			// automatic and cannot be accidentally omitted.
			password, credSource, resolveErr = h.password.resolve(user, host, false, true)
		}
		if resolveErr != nil {
			return resolveErr
		}
		if strings.TrimSpace(password) == "" {
			return fmt.Errorf("SSH password is required for %s; use the masked prompt, --password-stdin, or $%s", connectDecisionLabel(decision), sshPasswordEnvVar)
		}
	}

	revision := strings.TrimSpace(ctx.Flag("revision"))
	if revision == "" {
		revision = defaultRevision
	}
	sourceMode, err := resolveSourceMode(ctx.Flag("source"))
	if err != nil {
		return err
	}
	startResp, err := h.client.StartOnboarding(context.Background(), connect.NewRequest(&onboardv1.StartOnboardingRequest{
		MachineId: preflight.Msg.MachineId, Host: preflight.Msg.Host, Port: preflight.Msg.Port, User: preflight.Msg.User,
		SshPassword: password, NodeName: ctx.Flag("name"), TargetRevision: revision,
		RepoUrl: ctx.Flag("repo-url"), CheckoutDir: ctx.Flag("checkout-dir"), ControlPlaneUrl: ctx.Flag("control-plane-url"),
		ReachabilityMode: strings.TrimSpace(ctx.Flag("reachability-mode")), VerifyTimeoutSeconds: int32(parseInt(ctx.Flag("verify-timeout"))),
		SkipSetup: ctx.BoolFlag("skip-setup"), SkipPrereqs: ctx.BoolFlag("skip-prereqs"), ProvisionSudo: resolveProvisionSudo(ctx),
		SetupPreset:          strings.TrimSpace(ctx.Flag("preset")),
		ProvisionServiceUser: strings.TrimSpace(ctx.Flag("provision-service-user")), SourceMode: sourceMode,
	}))
	if err != nil {
		return wrapOnboardAPIError("start onboarding connection", err)
	}
	if startResp == nil || startResp.Msg == nil || startResp.Msg.OpId == "" {
		return fmt.Errorf("server returned no onboarding operation")
	}
	if !ctx.JSON() {
		fmt.Fprintf(ctx.Stdout(), "Connection preflight: %s — %s\n", connectDecisionLabel(decision), preflight.Msg.Message)
		if preflight.Msg.ClientKeyFingerprint != "" {
			fmt.Fprintf(ctx.Stdout(), "Machine: %s; client key: %s\n", preflight.Msg.MachineId, preflight.Msg.ClientKeyFingerprint)
		} else {
			fmt.Fprintf(ctx.Stdout(), "Machine: %s; credential: %s\n", preflight.Msg.MachineId, credentialReportLine(credSource))
		}
	}
	fmt.Fprintf(ctx.Stderr(), "Bridge: onboarding operation %s started; waiting for remote setup and ONLINE confirmation. This may take several minutes. Do not press Enter until prompted.\n", startResp.Msg.OpId)
	waitResp, err := h.client.WaitOnboarding(context.Background(), connect.NewRequest(&onboardv1.WaitOnboardingRequest{Id: startResp.Msg.OpId, TimeoutSeconds: 1800}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("wait for onboarding %q", startResp.Msg.OpId), err, nil)
	}
	if waitResp == nil || waitResp.Msg == nil {
		return fmt.Errorf("server returned no onboarding wait response")
	}
	getResp, err := h.client.GetOnboarding(context.Background(), connect.NewRequest(&onboardv1.GetOnboardingRequest{Id: startResp.Msg.OpId}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get onboarding %q", startResp.Msg.OpId), err, nil)
	}
	if getResp == nil || getResp.Msg == nil || getResp.Msg.Op == nil {
		return fmt.Errorf("server returned no onboarding op")
	}
	if getResp.Msg.Op.State == onboardv1.OnboardingState_ONBOARDING_STATE_SUCCEEDED && strings.TrimSpace(getResp.Msg.Op.NodeId) != "" {
		fmt.Fprintln(ctx.Stderr(), "Bridge: remote onboarding succeeded; the break-glass prompt is now ready.")
	}
	protectionDetail, protectionErr := h.protectAfterOnboarding(ctx, preflight.Msg.MachineId, getResp.Msg.Op)
	if protectionErr != nil {
		return protectionErr
	}
	if !ctx.JSON() && protectionDetail != "" {
		fmt.Fprintln(ctx.Stdout(), protectionDetail)
	}
	if protectionDetail != "" {
		refreshed, refreshErr := h.client.GetOnboarding(context.Background(), connect.NewRequest(&onboardv1.GetOnboardingRequest{Id: getResp.Msg.Op.Id}))
		if refreshErr != nil {
			return cliapp.WrapAPIError(fmt.Sprintf("get protected onboarding %q", getResp.Msg.Op.Id), refreshErr, nil)
		}
		if refreshed == nil || refreshed.Msg == nil || refreshed.Msg.Op == nil {
			return fmt.Errorf("server returned no protected onboarding op")
		}
		getResp = refreshed
	}
	return h.renderTerminal(ctx, getResp.Msg)
}

func connectDecisionLabel(decision onboardv1.ConnectDecision) string {
	switch decision {
	case onboardv1.ConnectDecision_CONNECT_DECISION_RECONNECT:
		return "reconnect"
	case onboardv1.ConnectDecision_CONNECT_DECISION_FIRST_TOUCH:
		return "first touch"
	case onboardv1.ConnectDecision_CONNECT_DECISION_RECOVERY_REQUIRED:
		return "recovery required"
	case onboardv1.ConnectDecision_CONNECT_DECISION_AMBIGUOUS:
		return "ambiguous identity"
	case onboardv1.ConnectDecision_CONNECT_DECISION_HOST_KEY_REVIEW:
		return "host-key review required"
	default:
		return "unknown connection state"
	}
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	// Onboarding RPCs include server-owned waits that may legitimately span
	// many minutes. Keep this domain attached long enough to reattach and
	// render the durable terminal result instead of timing out locally.
	httpClient, baseURL := session.NewConnectHTTPClientWithTimeout(core, 2*time.Hour)
	return &handlers{
		core:          core,
		client:        onboardconnect.NewOnboardServiceClient(httpClient, baseURL),
		machines:      machinesconnect.NewMachineServiceClient(httpClient, baseURL),
		cleanup:       cleanupconnect.NewCleanupServiceClient(httpClient, baseURL),
		password:      newPasswordSource(),
		authorization: os.Stdin,
	}
}

// protect is the named first-touch protection step. It deliberately runs
// after onboarding has produced a durable node identity: the passphrase is
// sealed locally to that node's pinned key, and the control plane receives only
// the opaque envelope. Re-running the command is safe because the node helper's
// provision operation is idempotent for matching material.
func (h *handlers) protect(ctx cliapp.RunContext) error {
	prepared, err := h.cleanup.PrepareCleanup(context.Background(), connect.NewRequest(&cleanupv1.PrepareCleanupRequest{
		MachineId: ctx.Flag("machine"), NodeId: ctx.Flag("node"), Target: ctx.Flag("target"), Scope: ctx.Flag("scope"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("prepare target protection", err, nil)
	}
	if prepared == nil || prepared.Msg == nil || prepared.Msg.Target == nil {
		return fmt.Errorf("server returned no cleanup protection target")
	}
	target := prepared.Msg.Target
	if strings.TrimSpace(target.OperationId) == "" || len(target.SealingPublicKey) == 0 {
		return fmt.Errorf("cleanup protection target did not publish an operation id and sealing key")
	}
	sealed, _, err := operatorauth.Read(h.authorization, operatorauth.Target{
		MachineID: target.MachineId, NodeID: target.NodeId, Target: target.Target, Scope: target.Scope,
		OperationID: target.OperationId, OperatorID: target.OperatorId, SealingPublicKey: target.SealingPublicKey,
	})
	if err != nil {
		return err
	}
	resp, err := h.cleanup.ProvisionBreakGlass(context.Background(), connect.NewRequest(&cleanupv1.ProvisionBreakGlassRequest{
		MachineId: target.MachineId, NodeId: target.NodeId, Target: target.Target, Scope: target.Scope,
		OperationId: target.OperationId, SealedPassphrase: sealed, OperatorId: target.OperatorId,
	}))
	if err != nil {
		return cliapp.WrapAPIError("establish target break-glass protection", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Operation == nil {
		return fmt.Errorf("server returned no protection operation")
	}
	op := resp.Msg.Operation
	changes := capabilityReport(target, true)
	changes = append(changes, fmt.Sprintf("target-bound break-glass: provisioning operation %s (%s)", op.Id, cleanupStatusLabel(op.Status)))
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Onboarding protection step dispatched for %s.", target.Target)},
		Changes:     changes,
		NextCommand: []string{fmt.Sprintf("`cleanup get %s` — confirm the target-bound break-glass capability is established", op.Id)},
	})
}

// completeProtection is the explicit recovery of the named onboarding step
// for callers that already have a durable onboarding operation. Its stdin
// contract is the same opaque authorization object as standalone `protect`;
// the canonical interactive `connect` path uses the local masked prompt.
func (h *handlers) completeProtection(ctx cliapp.RunContext) error {
	prepared, err := h.cleanup.PrepareCleanup(context.Background(), connect.NewRequest(&cleanupv1.PrepareCleanupRequest{
		MachineId: ctx.Flag("machine"), NodeId: ctx.Flag("node"), Target: ctx.Flag("target"), Scope: ctx.Flag("scope"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("prepare onboarding protection", err, nil)
	}
	if prepared == nil || prepared.Msg == nil || prepared.Msg.Target == nil {
		return fmt.Errorf("server returned no onboarding protection target")
	}
	target := prepared.Msg.Target
	sealed, _, err := operatorauth.Read(h.authorization, operatorauth.Target{
		MachineID: target.MachineId, NodeID: target.NodeId, Target: target.Target, Scope: target.Scope,
		OperationID: target.OperationId, OperatorID: target.OperatorId, SealingPublicKey: target.SealingPublicKey,
	})
	if err != nil {
		return err
	}
	resp, err := h.client.ProtectOnboarding(context.Background(), connect.NewRequest(&onboardv1.ProtectOnboardingRequest{
		OnboardingOpId: ctx.Flag("onboarding-op-id"), MachineId: target.MachineId, NodeId: target.NodeId,
		Target: target.Target, Scope: target.Scope, CleanupOperationId: target.OperationId, SealedPassphrase: sealed,
	}))
	if err != nil {
		return cliapp.WrapAPIError("complete onboarding protection", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no onboarding protection result")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{"Onboarding protection step completed."},
		Changes: capabilityReport(target, resp.Msg.ProtectionStatus == "completed"),
	})
}

// protectAfterOnboarding is the canonical first-touch protection step. It
// runs only after the node identity exists, so the passphrase can be sealed on
// the operator machine to the node's pinned key. A non-interactive invocation
// deliberately declines and records the missing capability instead of reading
// an environment variable or hanging for input.
func (h *handlers) protectAfterOnboarding(ctx cliapp.RunContext, machineID string, op *onboardv1.OnboardingOp) (string, error) {
	if op == nil || op.State != onboardv1.OnboardingState_ONBOARDING_STATE_SUCCEEDED || strings.TrimSpace(op.NodeId) == "" {
		return "", nil
	}
	prepared, err := h.cleanup.PrepareCleanup(context.Background(), connect.NewRequest(&cleanupv1.PrepareCleanupRequest{
		MachineId: machineID, NodeId: op.NodeId, Target: op.Host, Scope: "all",
	}))
	if err != nil {
		return "", cliapp.WrapAPIError("prepare onboarding protection", err, nil)
	}
	if prepared == nil || prepared.Msg == nil || prepared.Msg.Target == nil {
		return "", fmt.Errorf("server returned no onboarding protection target")
	}
	target := prepared.Msg.Target
	passphrase, err := h.password.resolveBreakGlass(target.Target)
	if err != nil {
		return "", err
	}
	request := &onboardv1.ProtectOnboardingRequest{
		OnboardingOpId: op.Id, MachineId: target.MachineId, NodeId: target.NodeId, Target: target.Target,
		Scope: target.Scope, CleanupOperationId: target.OperationId,
	}
	if strings.TrimSpace(passphrase) == "" {
		request.Declined = true
	} else {
		sealed, sealErr := operatorauth.SealPassphrase(passphrase, operatorauth.Target{
			MachineID: target.MachineId, NodeID: target.NodeId, Target: target.Target, Scope: target.Scope,
			OperationID: target.OperationId, OperatorID: target.OperatorId, SealingPublicKey: target.SealingPublicKey,
		})
		if sealErr != nil {
			return "", sealErr
		}
		request.SealedPassphrase = sealed
	}
	resp, err := h.client.ProtectOnboarding(context.Background(), connect.NewRequest(request))
	if err != nil {
		return "", cliapp.WrapAPIError("complete onboarding protection", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return "", fmt.Errorf("server returned no onboarding protection result")
	}
	lines := capabilityReport(target, !request.Declined && resp.Msg.ProtectionStatus == "completed")
	if request.Declined {
		lines = append(lines, "protection: declined — target-bound break-glass is missing")
	} else {
		lines = append(lines, "protection: "+resp.Msg.Detail)
	}
	return "Onboarding protection report:\n  " + strings.Join(lines, "\n  "), nil
}

func capabilityReport(target *cleanupv1.CleanupTarget, protected bool) []string {
	capabilities := make(map[string]bool, len(target.Capabilities))
	for _, value := range target.Capabilities {
		capabilities[strings.TrimSpace(value)] = true
	}
	scopes := make(map[string]bool, len(target.ApprovedScopes))
	for _, value := range target.ApprovedScopes {
		scopes[strings.TrimSpace(value)] = true
	}
	lines := make([]string, 0, len(privilegedops.OnboardingCapabilities()))
	for _, capability := range privilegedops.OnboardingCapabilities() {
		status := "not reported"
		switch capability.Name {
		case privilegedops.CapabilityAgentPresence:
			status = target.Transport + " (" + target.TransportReason + ")"
		case privilegedops.CapabilityRuntime:
			status = capabilityState(capabilities[capability.Name])
		case privilegedops.CapabilityProvisioning:
			status = capabilityState(capabilities[capability.Name])
		case privilegedops.CapabilitySSHManagement:
			status = approvedState(capabilities[capability.Name], scopes[capability.Name])
		case privilegedops.CapabilityCleanupPlanning:
			status = "available through the typed helper"
		case privilegedops.CapabilityCleanupApplication:
			status = "operator-confirmed only"
		case privilegedops.CapabilityTargetBoundBreakGlass:
			status = capabilityState(protected)
		}
		lines = append(lines, capability.Label+": "+status)
	}
	return lines
}

func capabilityState(available bool) string {
	if available {
		return "reported"
	}
	return "not reported"
}

func approvedState(capability, scope bool) string {
	switch {
	case capability && scope:
		return "approved"
	case capability:
		return "reported but not approved"
	default:
		return "not reported"
	}
}

func cleanupStatusLabel(status cleanupv1.CleanupStatus) string {
	return strings.TrimPrefix(status.String(), "CLEANUP_STATUS_")
}

// start kicks off a durable, server-owned onboarding op. The SSH password is
// resolved promptlessly — --password-stdin, opt-in --prompt-password, or
// $BRIDGE_SSH_PASSWORD, else empty (key-trusted host) — and carried once in
// the request body, never on argv. Honours the global --dry-run (which sets
// X-Dry-Run): the server validates and returns dry_run=true without creating an
// op or touching the host.
func (h *handlers) start(ctx cliapp.RunContext) error {
	host := strings.TrimSpace(ctx.Flag("host"))
	user := strings.TrimSpace(ctx.Flag("user"))
	machineID := strings.TrimSpace(ctx.Flag("machine-id"))
	port := parseInt(ctx.Flag("port"))
	if port == 0 {
		port = 22
	}

	// Start is the lower-level automation verb, but it must still use the same
	// server-owned identity resolver as connect. This prevents an omitted
	// machine-id from silently creating a replacement Machine and prevents an
	// empty password from being treated as proof of SSH trust.
	preflight, err := h.client.PreflightOnboarding(context.Background(), connect.NewRequest(&onboardv1.PreflightOnboardingRequest{
		Host: host, Port: int32(port), User: user, MachineId: machineID,
	}))
	if err != nil {
		return wrapOnboardAPIError("preflight onboarding", err)
	}
	if preflight == nil || preflight.Msg == nil {
		return fmt.Errorf("server returned no onboarding preflight")
	}
	decision := preflight.Msg.Decision
	if decision == onboardv1.ConnectDecision_CONNECT_DECISION_AMBIGUOUS || decision == onboardv1.ConnectDecision_CONNECT_DECISION_HOST_KEY_REVIEW {
		return fmt.Errorf("cannot start onboarding %s: %s", connectDecisionLabel(decision), preflight.Msg.Message)
	}
	host = preflight.Msg.Host
	user = preflight.Msg.User
	port = int(preflight.Msg.Port)
	machineID = preflight.Msg.MachineId

	// Resolve the password BEFORE the RPC so a credential-intake failure never
	// half-starts. An explicitly selected Machine reuses its Bridge-managed key
	// without prompting; an explicit stdin/prompt/env credential remains
	// available if that key is genuinely missing or revoked.
	fromStdin := ctx.BoolFlag("password-stdin")
	setupPassphraseStdin := ctx.BoolFlag("setup-passphrase-stdin")
	if setupPassphraseStdin {
		return fmt.Errorf("--setup-passphrase-stdin is retired: the target setup queues credential-store protection for vrooli-onboarding")
	}
	var password string
	var credSource credentialSource
	password, credSource, err = h.password.resolve(user, host, fromStdin, ctx.BoolFlag("prompt-password"))
	if err != nil {
		return err
	}
	if preflight.Msg.PasswordRequired && strings.TrimSpace(password) == "" {
		return fmt.Errorf("SSH password is required for %s; use --password-stdin, --prompt-password, or $%s", connectDecisionLabel(decision), sshPasswordEnvVar)
	}
	revision := strings.TrimSpace(ctx.Flag("revision"))
	if revision == "" {
		revision = defaultRevision
	}

	sourceMode, err := resolveSourceMode(ctx.Flag("source"))
	if err != nil {
		return err
	}

	resp, err := h.client.StartOnboarding(context.Background(), connect.NewRequest(&onboardv1.StartOnboardingRequest{
		MachineId:            machineID,
		Host:                 host,
		Port:                 int32(port),
		User:                 user,
		SshPassword:          password,
		NodeName:             ctx.Flag("name"),
		TargetRevision:       revision,
		RepoUrl:              ctx.Flag("repo-url"),
		CheckoutDir:          ctx.Flag("checkout-dir"),
		ControlPlaneUrl:      ctx.Flag("control-plane-url"),
		ReachabilityMode:     strings.TrimSpace(ctx.Flag("reachability-mode")),
		Capabilities:         splitCSV(ctx.Flag("capabilities")),
		VerifyTimeoutSeconds: int32(parseInt(ctx.Flag("verify-timeout"))),
		SkipSetup:            ctx.BoolFlag("skip-setup"),
		SkipPrereqs:          ctx.BoolFlag("skip-prereqs"),
		ProvisionSudo:        resolveProvisionSudo(ctx),
		SetupPreset:          strings.TrimSpace(ctx.Flag("preset")),
		ProvisionServiceUser: strings.TrimSpace(ctx.Flag("provision-service-user")),
		SourceMode:           sourceMode,
	}))
	if err != nil {
		// The server's revision-preflight and validation errors (unsafe ref,
		// unpushed commit, cannot-determine-CP-commit, bad SSH target) carry their
		// own Connect code + actionable message; WrapAPIError surfaces both.
		return cliapp.WrapAPIError("start onboarding", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no response")
	}

	target := formatTarget(resp.Msg.User, resp.Msg.Host, resp.Msg.Port)
	if resp.Msg.DryRun {
		return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
			Result:  []string{fmt.Sprintf("Dry run: %s would be onboarded.", target)},
			Changes: []string{"No op was created and the host was not touched."},
		})
	}
	changes := []string{
		fmt.Sprintf("op id: %s", resp.Msg.OpId),
		fmt.Sprintf("target revision: %s", revision),
		credentialReportLine(credSource),
	}
	if sourceMode == onboardv1.SourceMode_SOURCE_MODE_WORKING_TREE {
		changes = append(changes,
			"source: working-tree — shipping the control plane's local tree over SSH (uncommitted work included)",
			"provenance: the node records a dirty working-tree revision and is excluded from fleet rolls until re-onboarded pinned",
		)
	} else {
		changes = append(changes, "source: pinned — the node clones the pushed revision")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Onboarding started: op %s for %s.", resp.Msg.OpId, target)},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("`onboard watch %s` — follow the live step states through to ONLINE", resp.Msg.OpId),
		},
	})
}

func wrapOnboardAPIError(action string, err error) error {
	if connect.CodeOf(err) == connect.CodeUnauthenticated {
		return fmt.Errorf("%s: owner session unavailable; the local Bridge machine binding could not authenticate this CLI. Run `vrooli-bridge auth login --email <your-account-email>` once, then retry (the login uses the local binding before asking for a password)", action)
	}
	return cliapp.WrapAPIError(action, err, nil)
}

// credentialReportLine echoes the (non-secret) credential intake path in the
// start report, so the operator can tell which source won — or that none was
// provided and the run is riding on an already-key-trusted host.
func credentialReportLine(source credentialSource) string {
	if source == credentialNone {
		return "credential: none provided — assuming the host already trusts the bridge key " +
			"(supply one via --password-stdin, --prompt-password, $" + sshPasswordEnvVar + ", or the UI onboard form)"
	}
	return fmt.Sprintf("credential: SSH password from %s — sent once in the request body, used for first-touch, never stored", source)
}

// resolveSourceMode maps the --source flag to the proto SourceMode. Empty or
// "pinned" is the default (clone/fetch the pushed revision); "working-tree" ships
// the control plane's local tree over SSH. Any other value is a usage error.
func resolveSourceMode(raw string) (onboardv1.SourceMode, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "pinned":
		return onboardv1.SourceMode_SOURCE_MODE_PINNED_REVISION, nil
	case "working-tree", "working_tree", "worktree":
		return onboardv1.SourceMode_SOURCE_MODE_WORKING_TREE, nil
	default:
		return onboardv1.SourceMode_SOURCE_MODE_UNSPECIFIED, fmt.Errorf("invalid --source %q: expected pinned or working-tree", raw)
	}
}

// resolveProvisionSudo resolves the default-ON passwordless-sudo provisioning
// intent from the presence-based flags: on unless --no-provision-sudo is given,
// with an explicit --provision-sudo winning if both appear. The operator is
// handing over admin credentials, so the useful default is to leave the host with
// working non-interactive sudo for later privileged steps.
func resolveProvisionSudo(ctx cliapp.RunContext) bool {
	provision := true
	if ctx.BoolFlag("no-provision-sudo") {
		provision = false
	}
	if ctx.BoolFlag("provision-sudo") {
		provision = true
	}
	return provision
}

// status shows one op by id with its full persisted step-event history.
func (h *handlers) status(ctx cliapp.RunContext) error {
	id := ctx.Positional("op-id")
	resp, err := h.client.GetOnboarding(context.Background(), connect.NewRequest(&onboardv1.GetOnboardingRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get onboarding %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Op == nil {
		return fmt.Errorf("server returned no onboarding op")
	}
	return renderOpWithEvents(ctx, resp.Msg)
}

// list shows onboarding ops newest-first, optionally filtered by host.
func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListOnboardings(context.Background(), connect.NewRequest(&onboardv1.ListOnboardingsRequest{
		Host:  ctx.Flag("host"),
		Limit: int32(parseInt(ctx.Flag("limit"))),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list onboardings", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no onboardings response")
	}
	results := make([]string, 0, len(resp.Msg.Ops))
	for _, op := range resp.Msg.Ops {
		results = append(results, formatOp(op))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d onboarding op(s).", len(resp.Msg.Ops))},
		ResultsHeading: "Onboardings",
		Results:        results,
		RetrievalHints: []string{
			"`onboard status <op-id>` — show one op with its step history",
			"`onboard watch <op-id>` — follow a running op to completion",
		},
	})
}

func (h *handlers) removeFailed(ctx cliapp.RunContext) error {
	id := ctx.Positional("op-id")
	_, err := h.client.RemoveFailedOnboarding(context.Background(), connect.NewRequest(&onboardv1.RemoveFailedOnboardingRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("remove failed onboarding %q", id), err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, &onboardv1.RemoveFailedOnboardingResponse{}, cliapp.MutationReport{Result: []string{fmt.Sprintf("Removed failed onboarding history %s.", id)}})
}

// watch follows an op's live step states through to a terminal state. It drives
// the block-once WaitOnboarding (which blocks server-side for the wait window)
// interleaved with GetOnboarding to pull the step events that accrued — this is
// a re-attach loop, NOT a tight poll: the pacing is the server-side wait, and no
// client-side sleep is used. Exits non-zero on a FAILED/CANCELLED op so
// automation can branch on the process code.
func (h *handlers) watch(ctx cliapp.RunContext) error {
	id := ctx.Positional("op-id")
	timeout := parseInt64(ctx.Flag("timeout"))
	bg := context.Background()

	var lastSeq uint64
	for {
		getResp, err := h.client.GetOnboarding(bg, connect.NewRequest(&onboardv1.GetOnboardingRequest{Id: id}))
		if err != nil {
			return cliapp.WrapAPIError(fmt.Sprintf("get onboarding %q", id), err, nil)
		}
		if getResp == nil || getResp.Msg == nil || getResp.Msg.Op == nil {
			return fmt.Errorf("server returned no onboarding op")
		}
		op := getResp.Msg.Op

		// Human mode streams step events as they arrive; JSON mode emits a single
		// terminal document at the end (streaming JSON fragments is not a contract).
		if !ctx.JSON() {
			lastSeq = printNewEvents(ctx.Stdout(), getResp.Msg.Events, lastSeq)
		}

		if isTerminalState(op.State) {
			return h.renderTerminal(ctx, getResp.Msg)
		}

		waitResp, err := h.client.WaitOnboarding(bg, connect.NewRequest(&onboardv1.WaitOnboardingRequest{
			Id:             id,
			TimeoutSeconds: timeout,
		}))
		if err != nil {
			return cliapp.WrapAPIError(fmt.Sprintf("wait for onboarding %q", id), err, nil)
		}
		if waitResp == nil || waitResp.Msg == nil {
			return fmt.Errorf("server returned no wait response")
		}
		// Whether the wait returned terminal or timed_out, the next iteration's
		// GetOnboarding pulls the fresh events and re-checks the terminal state.
	}
}

// renderTerminal reports a finished op. In human mode the step events were
// already streamed, so it prints only a terminal summary (plus failure guidance
// on FAILED); in JSON mode it emits the full op + event history once. It returns
// a non-nil error on FAILED/CANCELLED so the process exits non-zero.
func (h *handlers) renderTerminal(ctx cliapp.RunContext, msg *onboardv1.GetOnboardingResponse) error {
	op := msg.Op
	if ctx.JSON() {
		if err := renderOpWithEvents(ctx, msg); err != nil {
			return err
		}
	} else {
		out := ctx.Stdout()
		for _, line := range terminalSummaryLines(op) {
			fmt.Fprintln(out, line)
		}
	}
	switch op.State {
	case onboardv1.OnboardingState_ONBOARDING_STATE_FAILED:
		return fmt.Errorf("onboarding %s failed: %s", op.Id, failureGuidance(op))
	case onboardv1.OnboardingState_ONBOARDING_STATE_CANCELLED:
		return fmt.Errorf("onboarding %s was cancelled", op.Id)
	default:
		return nil
	}
}

// cancel requests cancellation of a non-terminal op.
func (h *handlers) cancel(ctx cliapp.RunContext) error {
	id := ctx.Positional("op-id")
	resp, err := h.client.CancelOnboarding(context.Background(), connect.NewRequest(&onboardv1.CancelOnboardingRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("cancel onboarding %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Op == nil {
		return fmt.Errorf("server returned no onboarding op")
	}
	op := resp.Msg.Op
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Cancellation requested for onboarding %s (now %s).", op.Id, stateLabel(op.State))},
		Changes:     []string{formatOp(op)},
		NextCommand: []string{fmt.Sprintf("`onboard status %s` — confirm the terminal state; re-run `onboard start` to converge (idempotent)", op.Id)},
	})
}

// renderOpWithEvents routes a GetOnboardingResponse (op + full step history) to
// JSON or human output. Shared by `status` and the JSON path of `watch`.
func renderOpWithEvents(ctx cliapp.RunContext, msg *onboardv1.GetOnboardingResponse) error {
	op := msg.Op
	results := make([]string, 0, len(msg.Events)+1)
	results = append(results, formatOp(op))
	for _, ev := range msg.Events {
		results = append(results, "  "+formatStepEvent(ev))
	}
	summary := fmt.Sprintf("Onboarding %s — %s (%d event(s)).", op.Id, stateLabel(op.State), len(msg.Events))
	report := cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Onboarding",
		Results:        results,
	}
	if op.State == onboardv1.OnboardingState_ONBOARDING_STATE_FAILED {
		report.RetrievalHints = append(diagnosticsBlock(op), failureGuidance(op))
	}
	return cliapp.RenderProtoList(ctx, msg, report)
}

// ---- formatting helpers ----

func formatTarget(user, host string, port int32) string {
	who := user
	if who == "" {
		who = "root"
	}
	if port == 0 {
		port = 22
	}
	return fmt.Sprintf("%s@%s:%d", who, host, port)
}

func formatOp(op *onboardv1.OnboardingOp) string {
	if op == nil {
		return "(nil)"
	}
	created := ""
	if op.CreatedAt != nil {
		created = op.CreatedAt.AsTime().Format(time.RFC3339)
	}
	node := op.NodeId
	if node == "" {
		node = "-"
	}
	extra := ""
	if op.FailureReason != "" {
		extra = fmt.Sprintf(" reason=%s", op.FailureReason)
	}
	if op.ControlPlaneUrl != "" {
		extra += fmt.Sprintf(" endpoint=%s mode=%s", op.ControlPlaneUrl, op.ReachabilityMode)
	}
	return fmt.Sprintf("%s — %s@%s:%d name=%q [state=%s node=%s exit=%d%s created=%s]",
		op.Id, formatUser(op.User), op.Host, op.Port, op.NodeName, stateLabel(op.State), node, op.ExitCode, extra, created)
}

func formatUser(user string) string {
	if user == "" {
		return "root"
	}
	return user
}

func formatStepEvent(ev *onboardv1.OnboardingStepEvent) string {
	if ev == nil {
		return "(nil event)"
	}
	detail := ""
	if ev.Detail != "" {
		detail = " — " + ev.Detail
	}
	return fmt.Sprintf("[%03d] %-22s %s%s", ev.Sequence, ev.StepId, stepStatusLabel(ev.Status), detail)
}

// printNewEvents streams step events with a sequence greater than lastSeq and
// returns the new high-water mark, so a re-attach never re-prints history.
func printNewEvents(w io.Writer, events []*onboardv1.OnboardingStepEvent, lastSeq uint64) uint64 {
	for _, ev := range events {
		if ev == nil || ev.Sequence <= lastSeq {
			continue
		}
		fmt.Fprintln(w, formatStepEvent(ev))
		lastSeq = ev.Sequence
	}
	return lastSeq
}

// terminalSummaryLines is the human closing block for a finished op: a one-line
// verdict plus, on failure, the taxonomy-specific guidance, and on success the
// next step.
func terminalSummaryLines(op *onboardv1.OnboardingOp) []string {
	if op == nil {
		return []string{"Onboarding finished (no op)."}
	}
	target := fmt.Sprintf("%s@%s", formatUser(op.User), op.Host)
	switch op.State {
	case onboardv1.OnboardingState_ONBOARDING_STATE_SUCCEEDED:
		return []string{
			fmt.Sprintf("Onboarding %s SUCCEEDED: %s is paired and ONLINE (node %s).", op.Id, target, op.NodeId),
			fmt.Sprintf("`nodes list` — see %s in the fleet.", op.NodeId),
		}
	case onboardv1.OnboardingState_ONBOARDING_STATE_FAILED:
		lines := []string{fmt.Sprintf("Onboarding %s FAILED (exit %d).", op.Id, op.ExitCode)}
		lines = append(lines, diagnosticsBlock(op)...)
		lines = append(lines, failureGuidance(op))
		return lines
	case onboardv1.OnboardingState_ONBOARDING_STATE_CANCELLED:
		return []string{
			fmt.Sprintf("Onboarding %s was CANCELLED. The host may be partially set up; re-run `onboard start` to converge (idempotent).", op.Id),
		}
	default:
		return []string{fmt.Sprintf("Onboarding %s ended in state %s.", op.Id, stateLabel(op.State))}
	}
}

// diagnosticsBlock renders the node-side failure output (op.FailureDetail) as a
// framed, indented block so the operator sees the concrete cause — the actual
// "output above" the bootstrap's failure message referred to — not just the
// taxonomy guidance. Empty (control-plane-side failures produce no node output)
// yields no lines.
func diagnosticsBlock(op *onboardv1.OnboardingOp) []string {
	if op == nil || op.FailureDetail == "" {
		return nil
	}
	body := strings.Split(strings.TrimRight(op.FailureDetail, "\n"), "\n")
	lines := make([]string, 0, len(body)+2)
	lines = append(lines, "── node output (tail) ──────────────────────────────")
	for _, l := range body {
		lines = append(lines, "  "+l)
	}
	lines = append(lines, "────────────────────────────────────────────────────")
	return lines
}

func isTerminalState(s onboardv1.OnboardingState) bool {
	switch s {
	case onboardv1.OnboardingState_ONBOARDING_STATE_SUCCEEDED,
		onboardv1.OnboardingState_ONBOARDING_STATE_FAILED,
		onboardv1.OnboardingState_ONBOARDING_STATE_CANCELLED:
		return true
	default:
		return false
	}
}

func stateLabel(s onboardv1.OnboardingState) string {
	switch s {
	case onboardv1.OnboardingState_ONBOARDING_STATE_PENDING:
		return "pending"
	case onboardv1.OnboardingState_ONBOARDING_STATE_SSH_SETUP:
		return "ssh-setup"
	case onboardv1.OnboardingState_ONBOARDING_STATE_PUSHING_SCRIPT:
		return "pushing-script"
	case onboardv1.OnboardingState_ONBOARDING_STATE_BOOTSTRAPPING:
		return "bootstrapping"
	case onboardv1.OnboardingState_ONBOARDING_STATE_VERIFYING:
		return "verifying"
	case onboardv1.OnboardingState_ONBOARDING_STATE_SUCCEEDED:
		return "succeeded"
	case onboardv1.OnboardingState_ONBOARDING_STATE_FAILED:
		return "failed"
	case onboardv1.OnboardingState_ONBOARDING_STATE_CANCELLED:
		return "cancelled"
	default:
		return "unspecified"
	}
}

func stepStatusLabel(s onboardv1.OnboardingStepStatus) string {
	switch s {
	case onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_STARTED:
		return "started"
	case onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_OK:
		return "ok"
	case onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_SKIPPED:
		return "skipped"
	case onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_FAILED:
		return "failed"
	default:
		return "unspecified"
	}
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseInt(raw string) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return v
}

func parseInt64(raw string) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return v
}
