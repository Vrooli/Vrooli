package runtimehandlers

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/agentscope"
	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/clifreshness"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/rebaselineprogress"
	"github.com/vrooli/vrooli/internal/rootauthority"
	"github.com/vrooli/vrooli/internal/rootmetrics"
)

const rootUpdateUsage = "Usage: vrooli runtime update <status|observations|apply CANDIDATE|retry CANDIDATE|enroll|disable|reconcile|resume|rebaseline CANDIDATE [--hub|--force [--reason=TEXT]]> [--json]\nUpdates retain and qualify the serving image, activate under bounded native ownership, and restore independently on failure. Enrollment controls automatic installer/self-rebuild handoff; apply/retry are explicit one-shot requests.\nrebaseline crosses a persistent-state contract change without automatic rollback. Use `make rebaseline` in an operator terminal, --hub for a verified human decision in Notification Hub, or --force within an operator-approved `root-state-transition` grant; a forced run refuses unless it can snapshot state and write a receipt."

func rootUpdateCommand(args []string) bool {
	return len(args) >= 2 && args[0] == "runtime" && args[1] == "update"
}

// rootUpdateHelpRequested reports whether a root update invocation asks for
// help anywhere after `runtime update`. A help flag must never reach the
// effect path, where apply/retry would read it as the candidate path and
// submit a real root update request.
func rootUpdateHelpRequested(args []string) bool {
	if len(args) == 2 || args[2] == "help" {
		return true
	}
	for _, arg := range args[2:] {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

// Status/help remain available before the provisional candidate's ordinary
// command fence. They resolve the compiled state authority without init writes.
func RunRootUpdateReadCommand(args []string, stdout, stderr io.Writer) (bool, int) {
	if !rootUpdateCommand(args) {
		return false, 0
	}
	if rootUpdateHelpRequested(args) {
		_, _ = fmt.Fprintln(stdout, rootUpdateUsage)
		return true, 0
	}
	if args[2] != "status" && args[2] != "observations" {
		return false, 0
	}
	if buildinfo.RuntimeHomeSnapshotBase64 != "" {
		if err := buildinfo.PinCompiledRuntimeHome(); err != nil {
			_, _ = fmt.Fprintln(stderr, "Root update authority is invalid.")
			return true, 1
		}
	}
	if len(args) > 4 || len(args) == 4 && args[3] != "--json" {
		_, _ = fmt.Fprintln(stderr, rootUpdateUsage)
		return true, 2
	}
	home, err := config.HomeDir()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return true, 1
	}
	if args[2] == "observations" {
		state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
		if err == nil {
			var summary rootmetrics.Summary
			summary, err = rootmetrics.Summarize(filepath.Join(state, "root-rebaseline", "attempts"))
			if err == nil {
				err = json.NewEncoder(stdout).Encode(summary)
			}
		}
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return true, 1
		}
		return true, 0
	}
	enrollment, enrolled, record, exists, err := readRootUpdateStatus(home)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return true, 1
	}
	result := struct {
		Enrollment *coreactivation.Enrollment `json:"enrollment"`
		Operation  *coreactivation.Record     `json:"operation"`
	}{}
	if enrolled {
		result.Enrollment = &enrollment
	}
	if exists {
		result.Operation = &record
	}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		return true, 1
	}
	return true, 0
}

func RunRootUpdateEffectCommand(parent context.Context, args []string, stdout, stderr io.Writer, resolveSourceRoot func() (string, error)) (bool, int) {
	if !rootUpdateCommand(args) {
		return false, 0
	}
	if rootUpdateHelpRequested(args) {
		_, _ = fmt.Fprintln(stdout, rootUpdateUsage)
		return true, 0
	}
	if len(args) > 2 && args[len(args)-1] == "--json" {
		args = args[:len(args)-1]
	}
	if len(args) < 3 {
		_, _ = fmt.Fprintln(stderr, rootUpdateUsage)
		return true, 2
	}
	if args[2] == "rebaseline" {
		return true, runRootRebaseline(parent, args, os.Stdin, stdout, stderr, resolveSourceRoot)
	}
	if args[2] == "resume" {
		return true, runRootResume(parent, args[3:], stdout, stderr, resolveSourceRoot)
	}
	if args[2] == "retire-snapshot" {
		if len(args) != 4 || !filepath.IsAbs(args[3]) {
			_, _ = fmt.Fprintln(stderr, "usage: vrooli runtime update retire-snapshot ABSOLUTE_SNAPSHOT")
			return true, 2
		}
		ctx, cancel := context.WithTimeout(parent, 30*time.Second)
		defer cancel()
		client, err := newRootDecisionClient(ctx)
		if err == nil {
			err = retireSnapshotDecision(ctx, client, args[3])
		}
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return true, 1
		}
		return true, 0
	}
	mode := map[string]string{"apply": "--submit-core-update", "retry": "--retry-core-update", "enroll": "--enroll-core-updates", "disable": "--disable-core-updates", "reconcile": "--reconcile-core-updates"}[args[2]]
	if mode == "" || (args[2] == "apply" || args[2] == "retry") && len(args) != 4 || args[2] != "apply" && args[2] != "retry" && len(args) != 3 {
		_, _ = fmt.Fprintln(stderr, rootUpdateUsage)
		return true, 2
	}
	// The installed watchdog refuses these modes from an agent session too;
	// refusing here first keeps the reason in this process's own words.
	if (args[2] == "enroll" || args[2] == "disable" || args[2] == "retry") && refuseAgentSession(args[2], stderr) {
		return true, 1
	}
	if buildinfo.RuntimeHomeSnapshotBase64 != "" {
		if err := buildinfo.PinCompiledRuntimeHome(); err != nil {
			_, _ = fmt.Fprintln(stderr, "Root update authority is invalid.")
			return true, 1
		}
	}
	home, err := config.HomeDir()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return true, 1
	}
	worker, err := coreactivation.InstalledWatchdogPath(home)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return true, 1
	}
	workerArgs := []string{mode}
	if args[2] == "apply" || args[2] == "retry" {
		candidate, err := filepath.Abs(args[3])
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return true, 1
		}
		root, err := resolveSourceRoot()
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return true, 1
		}
		workerArgs = append(workerArgs, candidate, root)
	}
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runRootUpdateWorker(ctx, worker, workerArgs, stdout, stderr); err != nil {
		_, _ = fmt.Fprintln(stderr, "Root update did not complete; inspect `vrooli runtime update status --json` for the retained request.")
		return true, 1
	}
	return true, 0
}

// runRootRebaseline runs the operator state transition in this process: the
// installed coordinator may predate the candidate's contract format, so it
// cannot be the one to cross it. The decision is taken at an interactive
// terminal and is never implied by flags.
func runRootRebaseline(parent context.Context, args []string, stdin *os.File, stdout, stderr io.Writer, resolveSourceRoot func() (string, error)) int {
	if len(args) == 5 && args[3] == "--capability" && args[4] == "--probe-hub-publication" {
		_, _ = fmt.Fprintln(stdout, "hub-source-publication-v1")
		return 0
	}
	hubDecision := len(args) == 5 && args[4] == "--hub"
	prepared := len(args) == 5 && args[4] == "--prepared"
	forced := false
	forceReason := ""
	if len(args) > 4 {
		for _, argument := range args[4:] {
			switch {
			case argument == "--force":
				forced = true
			case strings.HasPrefix(argument, "--reason="):
				forceReason = strings.TrimPrefix(argument, "--reason=")
			}
		}
	}
	if len(args) < 4 || (len(args) != 4 && !hubDecision && !prepared && !forced) {
		_, _ = fmt.Fprintln(stderr, rootUpdateUsage)
		return 2
	}
	home, err := config.HomeDir()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	var authority rootauthority.Decision
	if forced {
		host, hostErr := os.Hostname()
		if hostErr != nil {
			_, _ = fmt.Fprintln(stderr, fmt.Errorf("root transition needs the host identity: %w", hostErr))
			return 1
		}
		authority, err = verifyRootTransitionAuthorityFn(parent, rootauthority.Target(host, home))
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "Forced root transition authorized by grant %s (approved by %s, run %s).\n", authority.GrantID, authority.ApprovedBy, authority.RunID)
	}
	transitionCaller := func() error {
		if hubDecision || forced {
			return nil
		}
		if refuseAgentSession("rebaseline", stderr) {
			return fmt.Errorf("root transition requires an operator")
		}
		// A pseudo-terminal alone is not an authority boundary.
		if stdin == nil {
			return fmt.Errorf("root rebaseline needs an interactive terminal, a `--force` standing grant, or `make rebaseline-hub` for a decision in Notification Hub")
		}
		if info, err := stdin.Stat(); err != nil || info.Mode()&os.ModeCharDevice == 0 {
			return fmt.Errorf("root rebaseline needs an interactive terminal, a `--force` standing grant, or `make rebaseline-hub` and review the decision in Notification Hub")
		}
		return nil
	}
	if !prepared && !hubDecision && !forced {
		if err := transitionCaller(); err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
	}
	candidate, err := filepath.Abs(args[3])
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	source, recovery, err := retainedRebaselineSource(home, candidate)
	var verifySnapshotSource func() error
	if err == nil && !recovery && (prepared || hubDecision) {
		source, verifySnapshotSource, err = pinnedRuntimeSnapshotSource(home, candidate, resolveSourceRoot)
	}
	if err == nil && !recovery && verifySnapshotSource == nil {
		source, err = pinnedRebaselineSource(resolveSourceRoot, candidate)
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	var bundleSHA, expectedCandidateSHA string
	if hubDecision || prepared {
		bundle, err := readPreparedRootBundle(candidate)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		bundleSHA = bundle.manifestSHA
		expectedCandidateSHA = bundle.rootSHA
	}
	// A forced transition must show its compensating controls before it touches
	// the serving image: a consistent state snapshot, a retained prior image and
	// a receipt. If the snapshot cannot be taken the transition does not run.
	var controls *runtimeapp.ForceControls
	if forced {
		preparedControls, err := prepareForceControlsFn(ctx, home, runtimeapp.ForceRequest{
			Authority:    authority,
			Reason:       forceReason,
			CandidateSHA: expectedCandidateSHA,
		})
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		controls = &preparedControls
		_, _ = fmt.Fprintf(stdout, "Pre-transition state snapshot: %s (%d file(s), digest %.12s).\nReceipt: %s\n", preparedControls.Dir, len(preparedControls.Receipt.SnapshotFiles), preparedControls.Receipt.SnapshotSHA, preparedControls.ReceiptAt)
	}
	recoveryOwner, freshLegacyFinishing, err := newPreparedRootRecovery(home, candidate, bundleSHA, hubDecision, recovery)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	pendingPath := recoveryOwner.pendingPath
	reader := bufio.NewReader(stdin)
	var afterActivation func(context.Context, runtimeapp.RebaselineRecovery) error
	var beforeEffects func(context.Context, runtimeapp.RebaselineAdmission) error
	var prepareRecovery func(context.Context, runtimeapp.RebaselineRecovery) error
	if hubDecision || prepared {
		if hubDecision {
			beforeEffects = recoveryOwner.beforeEffects
		}
		prepareRecovery = recoveryOwner.prepare
		afterActivation = func(ctx context.Context, recovery runtimeapp.RebaselineRecovery) error {
			return recoveryOwner.afterActivation(ctx, recovery, source)
		}
	}
	if verifySnapshotSource != nil {
		previous := beforeEffects
		beforeEffects = func(ctx context.Context, admission runtimeapp.RebaselineAdmission) error {
			if err := verifySnapshotSource(); err != nil {
				return err
			}
			if previous != nil {
				return previous(ctx, admission)
			}
			return nil
		}
	}
	result, err := rebaselineRootFn(ctx, home, runtimeapp.RebaselineRequest{
		Candidate:                     candidate,
		Source:                        source,
		RequireTransitionCaller:       transitionCaller,
		Progress:                      stdout,
		AfterActivation:               afterActivation,
		BeforeEffects:                 beforeEffects,
		PrepareRecovery:               prepareRecovery,
		HubDecision:                   hubDecision,
		RequireFreshFinishingDecision: freshLegacyFinishing,
		ExpectedCandidateSHA:          expectedCandidateSHA,
		BundleSHA:                     bundleSHA,
		AllowContinuationDecision:     hubDecision,
		WatchdogSnapshot:              recoveryOwner.watchdogSnapshot(),
		Confirm: func(plan runtimeapp.RebaselinePlan) (bool, error) {
			writeRebaselinePlan(stdout, plan)
			if hubDecision {
				client, err := rootDecisionClientFactory(ctx)
				if err != nil {
					return false, err
				}
				return confirmRootViaHub(ctx, client, plan, bundleSHA, pendingPath, stdout)
			}
			// A verified standing grant is the consent. Asking again on a TTY
			// would make `--force` unusable from an agent and would imply the
			// verified grant is advisory. The receipt records the grant, the
			// snapshot digest and the images, so the decision stays reviewable.
			if forced {
				return true, nil
			}
			_, _ = fmt.Fprint(stdout, "Make this build the new baseline? [y/N] ")
			answer, err := reader.ReadString('\n')
			if err != nil && answer == "" {
				return false, nil
			}
			answer = strings.ToLower(strings.TrimSpace(answer))
			return answer == "y" || answer == "yes", nil
		},
	})
	if handled, code := installSameContractBundle(err, prepared || hubDecision, home, candidate, bundleSHA, stdout, stderr, func(bundle, bin, manifest string) error {
		return installPreparedSameContractBundleFn(bundle, bin, manifest, source, verifySnapshotSource)
	}); handled {
		return code
	}
	if err != nil {
		if controls != nil {
			if restoreErr := restoreForceControlsFn(context.Background(), home, *controls); restoreErr != nil {
				_, _ = fmt.Fprintf(stderr, "State restore from %s failed: %v\nRestore it by hand before serving again.\n", controls.Dir, restoreErr)
			} else {
				_, _ = fmt.Fprintf(stdout, "State restored from the pre-transition snapshot %s.\n", controls.Dir)
			}
		}
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "Root control plane rebaselined: build %.12s, persistent-state contract %.12s.\n", result.Build, result.Contract)
	if controls != nil {
		_, _ = fmt.Fprintf(stdout, "Forced transition recorded in %s; the snapshot is retained at %s for recovery.\n", controls.ReceiptAt, controls.Dir)
	}
	if result.Enrolled {
		_, _ = fmt.Fprintln(stdout, "Automatic root updates are enrolled; behaviour-only changes now activate through `make install`.")
	} else {
		_, _ = fmt.Fprintln(stdout, "Automatic root updates are not enrolled (`vrooli runtime update enroll` enables them).")
	}
	if result.RetainedDir != "" {
		_, _ = fmt.Fprintf(stdout, "Prior baseline retained in %s.\n", result.RetainedDir)
	}
	return 0
}

// installSameContractBundle routes only a qualified same-contract prepared
// bundle through ordinary activation, which rechecks the live image and manifest.
func installSameContractBundle(rebaselineErr error, prepared bool, home, candidate, manifest string, stdout, stderr io.Writer, install func(string, string, string) error) (bool, int) {
	if !prepared || !errors.Is(rebaselineErr, runtimeapp.ErrRebaselineNotRequired) {
		return false, 0
	}
	binDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err == nil {
		_, _ = fmt.Fprintln(stdout, "Persistent-state contract is unchanged; activating the prepared bundle through the normal installer.")
		err = install(filepath.Dir(candidate), binDir, manifest)
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Normal activation of the prepared bundle failed: %v\n", err)
		return true, 1
	}
	_, _ = fmt.Fprintln(stdout, "Prepared bundle installed; no rebaseline was needed.")
	return true, 0
}

// Caller observations are replaced in tests.
var (
	callerKindFn                        = cliutil.DetectCallerKind
	callerCgroupFn                      = agentscope.SelfCgroup
	rootDecisionClientFactory           = newRootDecisionClient
	rebaselineRootFn                    = runtimeapp.RebaselineRoot
	installPreparedSameContractBundleFn = cliinstall.InstallPreparedBundleSameContractWithSourceChecked
	// verifyRootTransitionAuthorityFn resolves and verifies the operator's
	// standing grant for a forced root transition through the canonical
	// operation evaluator.
	verifyRootTransitionAuthorityFn = func(ctx context.Context, target string) (rootauthority.Decision, error) {
		return rootauthority.NewVerifier().Verify(ctx, target)
	}
	prepareForceControlsFn = runtimeapp.PrepareForceControls
	restoreForceControlsFn = runtimeapp.RestoreForceControls
)

// refuseAgentSession refuses an operator-only root update verb from a
// coding-agent session before anything else happens.
func refuseAgentSession(verb string, stderr io.Writer) bool {
	evidence := agentscope.SessionEvidence(callerKindFn(), callerCgroupFn())
	if evidence == "" {
		return false
	}
	what := map[string]string{
		"rebaseline": "Root rebaseline",
		"enroll":     "Changing automatic root update enrollment",
		"disable":    "Changing automatic root update enrollment",
		"retry":      "Overriding a failed root release's suppression",
	}[verb]
	remedy := "`vrooli runtime update " + verb + "`"
	if verb == "rebaseline" {
		remedy = "`make rebaseline`"
		_, _ = fmt.Fprintf(stderr, "%s is an operator decision and cannot run in terminal mode from a coding-agent session (%s). Ask the operator to run %s, approve a `root-state-transition` grant for `--force`, or use `make rebaseline-hub` for a decision in Notification Hub.\n", what, evidence, remedy)
		return true
	}
	_, _ = fmt.Fprintf(stderr, "%s is an operator decision and cannot run from a coding-agent session (%s). Ask the operator to run %s from their own terminal.\n", what, evidence, remedy)
	return true
}

func writeRebaselinePlan(out io.Writer, plan runtimeapp.RebaselinePlan) {
	if plan.Continuation {
		_, _ = fmt.Fprintln(out, "Continue interrupted root control-plane rebaseline")
	} else {
		_, _ = fmt.Fprintln(out, "Root control-plane rebaseline")
	}
	_, _ = fmt.Fprintf(out, "  serving:   build %.12s  persistent-state contract %.12s\n", plan.ServingBuild, plan.ServingContract)
	_, _ = fmt.Fprintf(out, "  candidate: build %.12s  persistent-state contract %.12s\n", plan.CandidateBuild, plan.CandidateContract)
	_, _ = fmt.Fprintln(out, "  Stored-state changes:")
	for _, change := range plan.Changes {
		_, _ = fmt.Fprintf(out, "    - %s\n", change)
	}
	if plan.Continuation {
		_, _ = fmt.Fprintln(out, "  Automatic rollback remains unavailable while the interrupted transition is completed.")
	} else {
		_, _ = fmt.Fprintln(out, "  The serving image cannot be relied on to read state the candidate writes, so this transition has no automatic rollback.")
	}
	if plan.RetainedDir != "" {
		_, _ = fmt.Fprintf(out, "  The serving root CLI, watchdog and supervisor unit will be retained in %s.\n", plan.RetainedDir)
	} else {
		_, _ = fmt.Fprintln(out, "  This continues the interrupted transition; inspect the earlier retained baseline for recovery.")
	}
}

// runRootResume finishes a transition that was interrupted after its effects
// began. The progress record names the prepared bundle, so the operator does
// not reconstruct a path or pick a command: `make rebaseline` calls this first
// and starts a new transition only when nothing is pending.
//
// It is the same operator-gated path as `rebaseline --prepared`. A non-operator
// caller is refused before anything is read or changed.
func runRootResume(parent context.Context, args []string, stdout, stderr io.Writer, resolveSourceRoot func() (string, error)) int {
	statusOnly := len(args) == 1 && args[0] == "--status"
	if len(args) != 0 && !statusOnly {
		_, _ = fmt.Fprintln(stderr, rootUpdateUsage)
		return 2
	}
	home, err := config.HomeDir()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	progress, err := rebaselineprogress.Read(filepath.Join(state, "root-rebaseline", "progress.json"))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if progress == nil || progress.Complete {
		if statusOnly {
			// A content answer, not an exit code: `go run` flattens a child's
			// non-zero status, and the make target only needs to know whether
			// preparing a new transition is necessary.
			_, _ = fmt.Fprintln(stdout, "none")
			return 0
		}
		_, _ = fmt.Fprintln(stdout, "No interrupted root transition to resume.")
		return 0
	}
	if statusOnly {
		_, _ = fmt.Fprintln(stdout, "pending")
		return 0
	}
	if progress.Mode != "" && progress.Mode != "prepared" {
		_, _ = fmt.Fprintln(stderr, "The interrupted transition is a Hub decision; resume it with `make rebaseline-hub-resume`.")
		return 1
	}
	candidate := filepath.Join(state, "root-rebaseline", "prepared-bundles", progress.Bundle, "vrooli")
	if _, err := os.Stat(candidate); err != nil {
		_, _ = fmt.Fprintf(stderr, "The prepared bundle for the interrupted transition is gone (%v); run `make rebaseline` to prepare a new one.\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "Resuming the interrupted root transition from %s.\n", filepath.Dir(candidate))
	return execResumeCandidate(candidate, stdout, stderr)
}

// execResumeCandidate runs the transition under the prepared bundle's own
// binary, exactly as the qualified transition does. Running it in this process
// would work, but the bundle binary is the image the transition qualified and
// carries its build stamps; using it also keeps the operator gate and the
// confirmation prompt in the same process the operator expects.
//
// The bundle may predate the local-refresh guard, so refresh is disabled for
// its process tree. That is a property of this recovery command, not something
// the operator should have to remember.
func execResumeCandidate(candidate string, stdout, stderr io.Writer) int {
	command := exec.Command(candidate, "runtime", "update", "rebaseline", candidate, "--prepared")
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, stdout, stderr
	command.Env = append(os.Environ(), clifreshness.EnvDisable+"=0")
	if err := command.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode()
		}
		_, _ = fmt.Fprintf(stderr, "resume the prepared transition with %s: %v\n", candidate, err)
		return 1
	}
	return 0
}
