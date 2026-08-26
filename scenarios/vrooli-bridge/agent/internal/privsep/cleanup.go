package privsep

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/cleanup"
	"github.com/vrooli/vrooli/packages/proto/privilegedops"
	"github.com/vrooli/vrooli/packages/proto/sealing"
)

// cleanupInputRunner is deliberately optional so the existing provisioning
// StepRunner seam remains source-compatible. Production osStepRunner supports
// it; tests can use a recorder that implements it when exercising secret flow.
type cleanupInputRunner interface {
	RunWithInput(context.Context, []string, string, []byte, func(string)) (int, error)
}

type cleanupEnvironmentRunner interface {
	RunWithEnvironment(context.Context, []string, string, []string, func(string)) (int, error)
}

type cleanupEnvironmentInputRunner interface {
	RunWithInputEnvironment(context.Context, []string, string, []byte, []string, func(string)) (int, error)
}

// Cleanup executes one named, fixed-argv operation. It never accepts argv from
// the caller. The command is the wire authorization envelope; the helper
// chooses every executable and argument itself.
func (h *Helper) Cleanup(ctx context.Context, cmd *channelv1.CleanupCommand, report func(*cleanupv1.CleanupEvent) error) error {
	if cmd == nil {
		return errors.New("cleanup command is required")
	}
	if _, ok := SupportedOperations[privilegedops.Name(cmd.GetOperation())]; !ok {
		return ErrUnsupportedOperation{Operation: privilegedops.Name(cmd.GetOperation())}
	}
	var seq uint64
	emit := func(ev *cleanupv1.CleanupEvent) error {
		seq++
		ev.OperationId = cmd.GetOpId()
		ev.Sequence = seq
		return report(ev)
	}
	emitStatus := func(status, reason string) error {
		return emit(&cleanupv1.CleanupEvent{Kind: cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_STATUS, Status: status, Reason: reason})
	}

	operation := privilegedops.Name(cmd.GetOperation())
	switch operation {
	case privilegedops.ResetBreakGlass:
		if !cmd.GetOperatorConfirmed() {
			return refusalEvent(emit, "operator_confirmed", "break-glass retirement requires explicit operator confirmation")
		}
		if _, err := h.runCLI(ctx, []string{"break-glass", "reset"}, nil); err != nil {
			return refusalEvent(emit, "reset", err.Error())
		}
		_ = emitStatus("completed", "managed target-bound break-glass material retired; no replacement was created")
		return emit(&cleanupv1.CleanupEvent{Kind: cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_EXIT, ExitCode: 0})
	case "provision_break_glass":
		// Provisioning is refusal-based idempotence: matching existing material
		// is a successful no-op, while ProvisionWrapped itself still refuses any
		// attempt to overwrite or silently adopt foreign material. Checking the
		// node-local status first lets onboarding report the existing-material path
		// as protected without weakening that lower-level refusal contract.
		ready, statusErr := h.breakGlassReady(ctx)
		if statusErr != nil {
			_ = emitStatus("failed", statusErr.Error())
			_ = emit(&cleanupv1.CleanupEvent{Kind: cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_EXIT, ExitCode: 1})
			return statusErr
		}
		if ready {
			_ = emitStatus("completed", "target-bound break-glass material already present; unchanged")
			return emit(&cleanupv1.CleanupEvent{Kind: cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_EXIT, ExitCode: 0})
		}
		if err := h.runSecretCLI(ctx, cmd, []string{"break-glass", "provision", "--account-id", cleanupAccountID(cmd), "--audience", privilegedops.BreakGlassAudience, "--target", cmd.GetTarget(), "--scopes", privilegedops.BreakGlassScope}); err != nil {
			_ = emitStatus("failed", err.Error())
			_ = emit(&cleanupv1.CleanupEvent{Kind: cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_EXIT, ExitCode: 1})
			return err
		}
		_ = emitStatus("completed", "target-bound break-glass material provisioned or already present")
		return emit(&cleanupv1.CleanupEvent{Kind: cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_EXIT, ExitCode: 0})
	case "issue_cleanup_capability":
		if err := emitStatus("applying", "issuing target-bound cleanup capability"); err != nil {
			return err
		}
		output, err := h.runSecretCLIOutput(ctx, cmd, []string{"break-glass", "issue", "--purpose", privilegedops.BreakGlassAudience, "--target", cmd.GetTarget(), "--scopes", privilegedops.BreakGlassScope, "--scope", cleanupScope(cmd.GetScope()), "--operator-id", cmd.GetOperatorId(), "--machine-id", cmd.GetMachineId(), "--node-id", cmd.GetNodeId(), "--plan-hash", cmd.GetPlanHash(), "--operation-id", cmd.GetPlanId(), "--ttl", "15m"})
		if err != nil {
			_ = emitStatus("failed", err.Error())
			_ = emit(&cleanupv1.CleanupEvent{Kind: cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_EXIT, ExitCode: 1})
			return err
		}
		// The CLI emits only the credential path and expiry. The signed token
		// remains on the node; it is never sent in a cleanup event.
		return finishJSONEvent(emit, emitStatus, cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_LOG, output, "completed")
	case "inventory_installation", "plan_uninstall":
		output, err := h.runCLI(ctx, []string{"uninstall", "--plan", "--plan-id", cmd.GetPlanId(), "--scope", cleanupScope(cmd.GetScope()), "--confirm-target", cmd.GetTarget(), "--json"}, nil)
		if err != nil {
			_ = emitStatus("failed", err.Error())
			_ = emit(&cleanupv1.CleanupEvent{Kind: cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_EXIT, ExitCode: 1})
			return err
		}
		return finishJSONEvent(emit, emitStatus, cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_PLAN, output, "planned")
	case "apply_frozen_plan":
		if !cmd.GetOperatorConfirmed() {
			return refusalEvent(emit, "operator_confirmed", "operator confirmation is required")
		}
		if err := emitStatus("applying", "starting frozen cleanup authorization"); err != nil {
			return err
		}
		token := append([]byte(nil), cmd.GetCapability()...)
		if len(token) == 0 {
			passphrase, openErr := h.openPassphrase(cmd)
			if openErr != nil {
				return refusalEvent(emit, "authorization", openErr.Error())
			}
			// Provisioning is intentionally idempotent for matching material and
			// refuses foreign/incomplete material. Keeping it immediately beside
			// issuance closes the pre-break-glass onboarding gap without creating a
			// second authorization path: the same sealed operator secret gates both
			// fixed-argv calls, and no secret is placed in the control plane event.
			ready, statusErr := h.breakGlassReady(ctx)
			if statusErr != nil {
				zeroBytes(passphrase)
				return refusalEvent(emit, "provision_status", statusErr.Error())
			}
			if !ready {
				if err := emitStatus("applying", "provisioning target-bound break-glass material"); err != nil {
					zeroBytes(passphrase)
					return err
				}
				provisionArgs := []string{"break-glass", "provision", "--account-id", cleanupAccountID(cmd), "--audience", privilegedops.BreakGlassAudience, "--target", cmd.GetTarget(), "--scopes", privilegedops.BreakGlassScope}
				provisionInput := append([]byte(nil), passphrase...)
				_, provisionErr := h.runCLI(ctx, provisionArgs, provisionInput)
				zeroBytes(provisionInput)
				if provisionErr != nil {
					zeroBytes(passphrase)
					return refusalEvent(emit, "provision", provisionErr.Error())
				}
			}
			if err := emitStatus("applying", "issuing target-bound cleanup capability"); err != nil {
				zeroBytes(passphrase)
				return err
			}
			issueArgs := []string{"break-glass", "issue", "--purpose", privilegedops.BreakGlassAudience, "--target", cmd.GetTarget(), "--scopes", privilegedops.BreakGlassScope, "--scope", cleanupScope(cmd.GetScope()), "--operator-id", cmd.GetOperatorId(), "--machine-id", cmd.GetMachineId(), "--node-id", cmd.GetNodeId(), "--plan-hash", cmd.GetPlanHash(), "--operation-id", cmd.GetPlanId(), "--ttl", "15m"}
			issueInput := append([]byte(nil), passphrase...)
			_, issueErr := h.runCLI(ctx, issueArgs, issueInput)
			zeroBytes(issueInput)
			zeroBytes(passphrase)
			if issueErr != nil {
				return refusalEvent(emit, "capability", issueErr.Error())
			}
			path, pathErr := h.breakGlassCredentialPath()
			if pathErr != nil {
				return refusalEvent(emit, "capability", pathErr.Error())
			}
			token, pathErr = os.ReadFile(path)
			if pathErr != nil {
				return refusalEvent(emit, "capability", pathErr.Error())
			}
		}
		if err := emitStatus("applying", "applying frozen cleanup plan"); err != nil {
			zeroBytes(token)
			return err
		}
		output, err := h.runCLI(ctx, []string{"uninstall", "--apply", cmd.GetPlanId(), "--scope", cleanupScope(cmd.GetScope()), "--confirm-target", cmd.GetTarget(), "--machine-id", cmd.GetMachineId(), "--node-id", cmd.GetNodeId(), "--operation-id", cmd.GetPlanId(), "--plan-hash", cmd.GetPlanHash(), "--operator-id", cmd.GetOperatorId(), "--break-glass-token-stdin", "--json"}, token)
		zeroBytes(token)
		if err != nil {
			_ = emitStatus("failed", err.Error())
			_ = emit(&cleanupv1.CleanupEvent{Kind: cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_EXIT, ExitCode: 1})
			return err
		}
		return finishJSONEvent(emit, emitStatus, cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_RECEIPT, output, "completed")
	case "verify_result":
		output, err := h.runCLI(ctx, []string{"uninstall", "--verify", cmd.GetPlanId(), "--scope", cleanupScope(cmd.GetScope()), "--confirm-target", cmd.GetTarget(), "--json"}, nil)
		if err != nil {
			return refusalEvent(emit, "verification", err.Error())
		}
		return finishJSONEvent(emit, emitStatus, cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_RECEIPT, output, "completed")
	case "rotate_break_glass":
		if !cmd.GetOperatorConfirmed() {
			return refusalEvent(emit, "operator_confirmed", "operator confirmation is required")
		}
		if err := h.runSecretCLI(ctx, cmd, []string{"break-glass", "rotate"}); err != nil {
			return refusalEvent(emit, "passphrase", err.Error())
		}
		_ = emitStatus("completed", "target-bound break-glass material rotated")
		return emit(&cleanupv1.CleanupEvent{Kind: cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_EXIT, ExitCode: 0})
	default:
		return ErrUnsupportedOperation{Operation: operation}
	}
}

func (h *Helper) breakGlassReady(ctx context.Context) (bool, error) {
	output, err := h.runCLI(ctx, []string{"break-glass", "status", "--json"}, nil)
	if err != nil {
		return false, err
	}
	var status struct {
		Complete bool `json:"complete"`
	}
	if err := json.Unmarshal(output, &status); err != nil {
		return false, fmt.Errorf("parse break-glass status: %w", err)
	}
	return status.Complete, nil
}

func (h *Helper) breakGlassCredentialPath() (string, error) {
	dir := strings.TrimSpace(os.Getenv("VROOLI_BREAK_GLASS_DIR"))
	if dir == "" {
		home := h.resolvedClientHome()
		if home == "" {
			return "", errors.New("resolve break-glass home")
		}
		dir = filepath.Join(home, ".vrooli", "identity", "break-glass")
	}
	return filepath.Join(dir, "credential"), nil
}

func finishJSONEvent(emit func(*cleanupv1.CleanupEvent) error, status func(string, string) error, kind cleanupv1.CleanupEventKind, output []byte, final string) error {
	if len(output) == 0 {
		return refusalEvent(emit, "response", "helper returned an empty response")
	}
	if !json.Valid(output) {
		return refusalEvent(emit, "response", "helper returned non-JSON output")
	}
	if err := emit(&cleanupv1.CleanupEvent{Kind: kind, PlanJson: jsonForKind(kind, output), ReceiptJson: receiptForKind(kind, output)}); err != nil {
		return err
	}
	if err := status(final, ""); err != nil {
		return err
	}
	return emit(&cleanupv1.CleanupEvent{Kind: cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_EXIT, ExitCode: 0})
}

func jsonForKind(kind cleanupv1.CleanupEventKind, output []byte) []byte {
	if kind == cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_PLAN {
		return append([]byte(nil), output...)
	}
	return nil
}

func receiptForKind(kind cleanupv1.CleanupEventKind, output []byte) []byte {
	if kind == cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_RECEIPT {
		return append([]byte(nil), output...)
	}
	return nil
}

func refusalEvent(emit func(*cleanupv1.CleanupEvent) error, field, reason string) error {
	_ = emit(&cleanupv1.CleanupEvent{Kind: cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_STATUS, Status: "blocked", Reason: field + ": " + reason})
	_ = emit(&cleanupv1.CleanupEvent{Kind: cleanupv1.CleanupEventKind_CLEANUP_EVENT_KIND_EXIT, ExitCode: 2})
	return fmt.Errorf("cleanup blocked: %s: %s", field, reason)
}

func (h *Helper) runSecretCLI(ctx context.Context, cmd *channelv1.CleanupCommand, args []string) error {
	passphrase, err := h.openPassphrase(cmd)
	if err != nil {
		return err
	}
	defer zeroBytes(passphrase)
	_, err = h.runCLI(ctx, args, passphrase)
	return err
}

func (h *Helper) runSecretCLIOutput(ctx context.Context, cmd *channelv1.CleanupCommand, args []string) ([]byte, error) {
	passphrase, err := h.openPassphrase(cmd)
	if err != nil {
		return nil, err
	}
	defer zeroBytes(passphrase)
	return h.runCLI(ctx, args, passphrase)
}

func (h *Helper) openPassphrase(cmd *channelv1.CleanupCommand) ([]byte, error) {
	if strings.TrimSpace(h.sealingSeedPath) == "" {
		return nil, errors.New("node sealing seed is not configured")
	}
	seed, err := os.ReadFile(h.sealingSeedPath)
	if err != nil {
		return nil, fmt.Errorf("read node sealing seed: %w", err)
	}
	defer zeroBytes(seed)
	private, err := sealing.PrivateKeyFromRaw(seed)
	if err != nil {
		return nil, err
	}
	aad := sealing.Context(cmd.GetMachineId(), cmd.GetNodeId(), cmd.GetTarget(), cmd.GetScope(), cmd.GetPlanHash(), cmd.GetPlanId(), cmd.GetOperatorId())
	return sealing.Open(private, cmd.GetSealedPassphrase(), aad)
}

func (h *Helper) runCLI(ctx context.Context, args []string, input []byte) ([]byte, error) {
	// Cleanup commands are executed from a privileged helper precisely when the
	// source tree or the installed CLI may be removed. The normal Vrooli entry
	// point performs a source-fingerprint check and may rebuild/re-exec a stale
	// binary; that would both make a destructive operation unbounded and can
	// rebuild from the very tree this command is deleting. The helper already
	// carries the exact installed executable, so maintenance commands must opt
	// out of that developer convenience and remain fixed-argv operations.
	argv := append([]string{h.vrooliBin, "--no-stale-check"}, args...)
	var output strings.Builder
	onLog := func(chunk string) { output.WriteString(chunk) }
	var code int
	var err error
	env := []string{}
	// Keep the child process outside the frozen removal set. In particular, the
	// checkout itself is commonly the final recorded runtime entry; deleting a
	// process's current directory can strand it before it persists or reports
	// the receipt. VROOLI_ROOT preserves the repo-contract context explicitly.
	workDir := h.cleanupWorkDir
	if strings.TrimSpace(workDir) == "" {
		workDir = os.TempDir()
	}
	if strings.TrimSpace(h.workDir) != "" {
		// buildinfo.ResolveSourceRoot gives VROOLI_SOURCE_ROOT precedence over
		// VROOLI_ROOT. Services can inherit a stale source pointer from an
		// installed CLI, so pin both repo-contract inputs to the exact checkout
		// selected by Bridge; otherwise cleanup can resolve ~/.vrooli/bin instead
		// of the managed repository and fail before the frozen plan is touched.
		env = append(env, "VROOLI_ROOT="+h.workDir)
		env = append(env, "VROOLI_SOURCE_ROOT="+h.workDir)
	}
	if len(h.deferredServiceNames) > 0 {
		env = append(env, "VROOLI_BRIDGE_DEFER_SERVICE_STOPS="+strings.Join(h.deferredServiceNames, ","))
	}
	if home := h.resolvedClientHome(); home != "" {
		env = append(env, "HOME="+home)
		// Break-glass material is runner-owned state. Pin the child to the same
		// managed path as the resolved runner home instead of inheriting an
		// ambient override from a service manager or an older installation. This
		// keeps reset, status, provision, issue, and apply on one exact path.
		env = append(env, "VROOLI_BREAK_GLASS_DIR="+filepath.Join(home, ".vrooli", "identity", "break-glass"))
	}
	if len(input) > 0 {
		if runner, ok := h.step.(cleanupEnvironmentInputRunner); ok && len(env) > 0 {
			code, err = runner.RunWithInputEnvironment(ctx, argv, workDir, input, env, onLog)
			zeroBytes(input)
		} else {
			runner, ok := h.step.(cleanupInputRunner)
			if !ok {
				return nil, errors.New("privileged helper cannot accept secret input")
			}
			code, err = runner.RunWithInput(ctx, argv, workDir, input, onLog)
			zeroBytes(input)
		}
	} else {
		if runner, ok := h.step.(cleanupEnvironmentRunner); ok && len(env) > 0 {
			code, err = runner.RunWithEnvironment(ctx, argv, workDir, env, onLog)
		} else {
			code, err = h.step.Run(ctx, argv, workDir, onLog)
		}
	}
	if err != nil {
		return nil, err
	}
	if code != 0 {
		rootHint := strings.TrimSpace(h.workDir)
		if rootHint == "" {
			rootHint = "<unset>"
		}
		return nil, fmt.Errorf("vrooli cleanup command exited with code %d (helper work dir %s): %s", code, rootHint, strings.TrimSpace(output.String()))
	}
	return []byte(strings.TrimSpace(output.String())), nil
}

func cleanupAccountID(cmd *channelv1.CleanupCommand) string {
	if strings.TrimSpace(cmd.GetMachineId()) != "" {
		return cmd.GetMachineId()
	}
	return cmd.GetNodeId()
}

func cleanupScope(scope string) string {
	if strings.TrimSpace(scope) == "" {
		return "all"
	}
	return scope
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
	runtime.KeepAlive(b)
}
