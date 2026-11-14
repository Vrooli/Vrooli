package runtimehandlers

import (
	"bufio"
	"context"
	"encoding/json"
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
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

const rootUpdateUsage = "Usage: vrooli runtime update <status|apply CANDIDATE|retry CANDIDATE|enroll|disable|reconcile|rebaseline CANDIDATE> [--json]\nUpdates retain and qualify the serving image, activate under bounded native ownership, and restore independently on failure. Enrollment controls automatic installer/self-rebuild handoff; apply/retry are explicit one-shot requests.\nrebaseline is the operator transition for a candidate that changes the persistent-state contract (no automatic rollback); it needs an interactive terminal. Prefer `make rebaseline`."

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

func rootUpdateStore(home string) (*coreactivation.Store, error) {
	if buildinfo.RuntimeHomeSnapshotBase64 == "" {
		return coreactivation.NewStore(home)
	}
	state, err := buildinfo.CompiledRuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return nil, err
	}
	return coreactivation.NewStoreAt(home, filepath.Join(state, "core-activation"))
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
	if args[2] != "status" {
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
	store, err := rootUpdateStore(home)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return true, 1
	}
	enrollment, enrolled, err := store.Enrollment()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return true, 1
	}
	record, exists, err := store.Current()
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
	ctx, cancel := context.WithTimeout(ctx, 8*time.Minute)
	defer cancel()
	name, wrapped := hostreqkit.InvokingUserCommand(worker, workerArgs...)
	cmd := exec.CommandContext(ctx, name, wrapped...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil {
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
	if len(args) != 4 {
		_, _ = fmt.Fprintln(stderr, rootUpdateUsage)
		return 2
	}
	if refuseAgentSession("rebaseline", stderr) {
		return 1
	}
	// The terminal prompt is the operator's confirmation, not an authority
	// boundary: a same-user process with a pseudo-terminal can answer it.
	if info, err := stdin.Stat(); err != nil || info.Mode()&os.ModeCharDevice == 0 {
		_, _ = fmt.Fprintln(stderr, "Root rebaseline needs an interactive terminal; run `make rebaseline` from your own shell.")
		return 1
	}
	home, err := config.HomeDir()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	candidate, err := filepath.Abs(args[3])
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	source, err := pinnedRebaselineSource(resolveSourceRoot, candidate)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	reader := bufio.NewReader(stdin)
	result, err := runtimeapp.RebaselineRoot(ctx, home, runtimeapp.RebaselineRequest{
		Candidate: candidate,
		Source:    source,
		Progress:  stdout,
		Confirm: func(plan runtimeapp.RebaselinePlan) (bool, error) {
			writeRebaselinePlan(stdout, plan)
			_, _ = fmt.Fprint(stdout, "Make this build the new baseline? [y/N] ")
			answer, err := reader.ReadString('\n')
			if err != nil && answer == "" {
				return false, nil
			}
			answer = strings.ToLower(strings.TrimSpace(answer))
			return answer == "y" || answer == "yes", nil
		},
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "Root control plane rebaselined: build %.12s, persistent-state contract %.12s.\n", result.Build, result.Contract)
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

// Caller observations are replaced in tests.
var (
	callerKindFn   = cliutil.DetectCallerKind
	callerCgroupFn = agentscope.SelfCgroup
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
	}
	_, _ = fmt.Fprintf(stderr, "%s is an operator decision and cannot run from a coding-agent session (%s). Ask the operator to run %s from their own terminal.\n", what, evidence, remedy)
	return true
}

func writeRebaselinePlan(out io.Writer, plan runtimeapp.RebaselinePlan) {
	_, _ = fmt.Fprintln(out, "Root control-plane rebaseline")
	_, _ = fmt.Fprintf(out, "  serving:   build %.12s  persistent-state contract %.12s\n", plan.ServingBuild, plan.ServingContract)
	_, _ = fmt.Fprintf(out, "  candidate: build %.12s  persistent-state contract %.12s\n", plan.CandidateBuild, plan.CandidateContract)
	_, _ = fmt.Fprintln(out, "  Stored-state changes:")
	for _, change := range plan.Changes {
		_, _ = fmt.Fprintf(out, "    - %s\n", change)
	}
	_, _ = fmt.Fprintln(out, "  The serving image cannot be relied on to read state the candidate writes, so this transition has no automatic rollback.")
	_, _ = fmt.Fprintf(out, "  The serving root CLI, watchdog and supervisor unit will be retained in %s.\n", plan.RetainedDir)
}

// pinnedRebaselineSource is the checkout the command runs from. The candidate
// must be inside it, and a VROOLI_SOURCE_ROOT/VROOLI_ROOT override must agree,
// so the watchdog, the supervisor unit and the candidate all come from one
// checkout.
func pinnedRebaselineSource(resolveSourceRoot func() (string, error), candidate string) (string, error) {
	cwdRoot, err := repoRootFromWorkingDirectory()
	if err != nil {
		return "", fmt.Errorf("run root rebaseline from the source checkout that built the candidate: %w", err)
	}
	cwdRoot = canonicalPath(cwdRoot)
	if resolved, err := resolveSourceRoot(); err == nil && canonicalPath(resolved) != cwdRoot {
		return "", fmt.Errorf("the source root override (VROOLI_SOURCE_ROOT or VROOLI_ROOT) names %s, but this command runs from %s; unset it or run from that checkout", resolved, cwdRoot)
	}
	if rel, err := filepath.Rel(cwdRoot, canonicalPath(candidate)); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("the candidate %s was not built in the checkout this command runs from (%s)", candidate, cwdRoot)
	}
	return cwdRoot, nil
}

var repoRootFromWorkingDirectory = repocontract.FindRepoRootFromCWD

func canonicalPath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(resolved)
	}
	return filepath.Clean(path)
}
