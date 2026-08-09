package onboard

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines"
	machinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines/machines_v1connect"
	onboardv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard"
	onboardconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard/onboard_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// defaultRevision is the presentation/value for "bring the node to the control
// plane's current commit". The server accepts empty or "@cp" identically
// (phase 6); the CLI sends "@cp" explicitly so the intent is visible on the wire
// and in `onboard start --help`.
const defaultRevision = "@cp"

// handlers bundles the Connect client and the (injectable) SSH-password source.
type handlers struct {
	core     *cliapp.ScenarioApp
	client   onboardconnect.OnboardServiceClient
	machines machineCreator
	password passwordSource
}

type machineCreator interface {
	CreateMachine(context.Context, *connect.Request[machinesv1.CreateMachineRequest]) (*connect.Response[machinesv1.CreateMachineResponse], error)
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:     core,
		client:   onboardconnect.NewOnboardServiceClient(httpClient, baseURL),
		machines: machinesconnect.NewMachineServiceClient(httpClient, baseURL),
		password: newPasswordSource(),
	}
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

	// Resolve the password BEFORE the RPC so a credential-intake failure never
	// half-starts. This never prompts unless --prompt-password was given.
	fromStdin := ctx.BoolFlag("password-stdin")
	setupPassphraseStdin := ctx.BoolFlag("setup-passphrase-stdin")
	if setupPassphraseStdin && !fromStdin {
		return fmt.Errorf("--setup-passphrase-stdin requires --password-stdin so both secrets can be read from one protected pipe")
	}
	var password string
	var credSource credentialSource
	var err error
	if setupPassphraseStdin {
		password, err = h.password.resolveLine()
		credSource = credentialFromStdin
	} else {
		password, credSource, err = h.password.resolve(user, host, fromStdin, ctx.BoolFlag("prompt-password"))
	}
	if err != nil {
		return err
	}
	setupPassphrase := ""
	if setupPassphraseStdin {
		setupPassphrase, err = h.password.resolveLine()
		if err != nil {
			return err
		}
		if strings.TrimSpace(setupPassphrase) == "" {
			return fmt.Errorf("setup credential-store passphrase from stdin must not be empty")
		}
	}
	machineID := ""
	machineResp, err := h.machines.CreateMachine(context.Background(), connect.NewRequest(&machinesv1.CreateMachineRequest{
		Locators: []*machinesv1.ConnectionLocator{{Kind: "hostname", Value: host, Ordinal: 0}},
	}))
	if err != nil {
		return cliapp.WrapAPIError("create Machine before onboarding", err, nil)
	}
	if machineResp == nil || machineResp.Msg == nil || machineResp.Msg.Machine == nil || strings.TrimSpace(machineResp.Msg.Machine.Id) == "" {
		return fmt.Errorf("server returned no machine for onboarding")
	}
	machineID = machineResp.Msg.Machine.Id

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
		Port:                 int32(parseInt(ctx.Flag("port"))),
		User:                 user,
		SshPassword:          password,
		SetupPassphrase:      setupPassphrase,
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
		SetupEnvironment:     strings.TrimSpace(ctx.Flag("setup-environment")),
		SetupResources:       strings.TrimSpace(ctx.Flag("setup-resources")),
		SetupScenarios:       strings.TrimSpace(ctx.Flag("setup-scenarios")),
		IncludeOptional:      ctx.BoolFlag("include-optional"),
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
