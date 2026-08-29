package main

import (
	"crypto/ed25519"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/trustposture"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/vroolicli"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/floorengagement"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/privilegebroker"
	codingagentshims "github.com/vrooli/vrooli/internal/safeguards/coding-agent-shims"
	"github.com/vrooli/vrooli/internal/scenarioexec"
	projectsetup "github.com/vrooli/vrooli/internal/setup"
	"github.com/vrooli/vrooli/internal/shell"
)

const (
	workingDirectoryFallbackCapacity = 2
)

const cliVersion = "1.0.0"

// vrooliVersion is a variable so release builds can embed the git-tag version
// with -ldflags while local development builds retain the repository default.
var vrooliVersion = "2.0.0"

var (
	resolveSourceRootFn = buildinfo.ResolveSourceRoot
	checkStalenessFn    = buildinfo.CheckStaleness
	rebuildAndReexecFn  = buildinfo.RebuildAndReexec
	lookPathFn          = shell.LookPath
	newLoggerFn         = createCommandLogger
)

type globalOptions = rootcli.GlobalOptions

func main() {
	// This internal-only service entry point is reached by the root-owned
	// systemd unit installed by `sudo vrooli setup`. It is intentionally
	// handled before normal CLI initialization and accepts no general command.
	if len(os.Args) > 1 && os.Args[1] == "__privilege-broker" {
		os.Exit(privilegebroker.RunServiceCommand(os.Args[2:], os.Stdout, os.Stderr))
	}
	ensureUsableWorkingDirectory(os.Stderr)
	recoverUserToolPath()
	reassertCodingAgentShims()
	installEngagementResolver(os.Stderr)
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// recoverUserToolPath makes managed per-user tool installs visible to every
// root-control-plane command, including commands launched by a non-login SSH
// session or a service manager. Setup already performs the same recovery while
// bootstrapping; doing it at process entry also covers scenario/resource
// lifecycle host-requirement enforcement before a lifecycle step gets a chance
// to apply its own child environment.
func recoverUserToolPath() {
	home, err := config.HomeDir()
	if err != nil {
		return
	}
	_ = os.Setenv("PATH", hostreqkit.AugmentUserToolPath(home, os.Getenv("PATH"), os.Getenv("LOCALAPPDATA")))
}

// workingDirSeams isolates the process-global state ensureUsableWorkingDirectory
// touches so the behaviour is testable without moving the test binary's own CWD.
type workingDirSeams struct {
	getwd   func() (string, error)
	stat    func(string) (os.FileInfo, error)
	chdir   func(string) error
	homeDir func() (string, error)
	tempDir func() string
}

var defaultWorkingDirSeams = workingDirSeams{
	getwd:   os.Getwd,
	stat:    os.Stat,
	chdir:   os.Chdir,
	homeDir: config.HomeDir,
	tempDir: os.TempDir,
}

// ensureUsableWorkingDirectory relocates the process when it starts in a
// directory it cannot stat, which happens more often than it sounds: a shell
// that dropped privileges while sitting in /root, a directory deleted after the
// shell entered it, or an unmounted network path. The failure is worth
// intercepting here because of how it presents downstream — the Go toolchain and
// many other tools refuse to start at all without a resolvable CWD, so every
// version probe returns empty and the host-requirement report blames the tool
// ("version mismatch: found \"\"") for what is really a property of the caller's
// environment. Relative paths are already broken in this state, so moving to the
// home directory forfeits nothing and makes every child process well-defined.
func ensureUsableWorkingDirectory(stderr io.Writer) {
	ensureUsableWorkingDirectoryWith(defaultWorkingDirSeams, stderr)
}

func ensureUsableWorkingDirectoryWith(seams workingDirSeams, stderr io.Writer) {
	// Both probes are needed because the two ways a working directory goes bad
	// fail differently: an unreadable directory (a shell left in /root after a
	// privilege drop) fails stat, while a directory deleted underneath a live
	// shell keeps a stat-able inode and fails only when its path is resolved.
	// The Go toolchain rejects both, so the guard must catch both.
	if _, err := seams.getwd(); err != nil {
		reportUnusableWorkingDirectory(seams, stderr)
		return
	}
	if _, err := seams.stat("."); err != nil {
		reportUnusableWorkingDirectory(seams, stderr)
	}
}

func reportUnusableWorkingDirectory(seams workingDirSeams, stderr io.Writer) {
	candidates := make([]string, 0, workingDirectoryFallbackCapacity)
	if home, err := seams.homeDir(); err == nil && strings.TrimSpace(home) != "" {
		candidates = append(candidates, home)
	}
	if temp := strings.TrimSpace(seams.tempDir()); temp != "" {
		candidates = append(candidates, temp)
	}
	for _, candidate := range candidates {
		if err := seams.chdir(candidate); err != nil {
			continue
		}
		fmt.Fprintf(stderr, "warning: the current working directory is unusable (deleted or not readable); continuing from %s so tool probes and builds can run\n", candidate)
		return
	}
	// Never fatal: plenty of commands (status, help, version) do not touch the
	// filesystem relative to CWD and must still work. Commands that do need it
	// will fail on their own terms with their own diagnostics.
	fmt.Fprintln(stderr, "warning: the current working directory is unusable (deleted or not readable) and no fallback directory could be entered; tools that require a resolvable working directory will fail")
}

// installEngagementResolver wires the Baseline Modes engagement resolver into
// the lifecycle once, before any Runner is constructed. With it installed, a
// live restart during an open shadow engagement resolves its build/run CWD to
// the frozen restore-point copy rather than the working tree the agent is
// editing (see internal/lifecycle effectiveSourceDir). It is intentionally not
// called from run(), so tests exercising run() stay hermetic. A resolver-
// construction failure is non-fatal (it only happens if the cache root cannot be
// resolved, which would already break far more), but it is surfaced loudly: the
// floor would be unenforced.
func installEngagementResolver(stderr io.Writer) {
	resolver, err := floorengagement.New()
	if err != nil {
		fmt.Fprintf(stderr, "warning: baseline-modes engagement resolver unavailable; live isolation not enforced: %v\n", err)
		return
	}
	lifecycle.SetDefaultEngagementResolver(resolver)
}

func run(args []string, stdout, stderr io.Writer) int {
	return configuredRunner().Run(args, stdout, stderr)
}

func configuredRunner() *rootcli.Runner[*vroolicli.CommandContext] {
	return configuredApp().Runner()
}

func configuredApp() *vroolicli.App {
	return vroolicli.New(vroolicli.Config{
		VersionInfo: vroolicli.VersionInfo{
			CLIVersion:      cliVersion,
			PlatformVersion: vrooliVersion,
		},
		ResolveSourceRootFn: resolveSourceRootFn,
		HomeDirFn:           config.HomeDir,
		CheckStalenessFn:    checkStalenessFn,
		RebuildAndReexecFn:  rebuildAndReexecFn,
		LookPathFn:          lookPathFn,
		NewLoggerFn:         newLoggerFn,
		DebugLogFn:          debugLog,
		RunProjectBuildFn:   projectsetup.RunBuild,
		RunProjectSetupFn:   projectsetup.RunSetupWithOptions,
		RunProjectDevelopFn: projectsetup.RunDevelopWithOptions,
		NewUninstallerFn:    newUninstaller,
		RunScenarioSubprocess: func(spec scenarioexec.SubprocessSpec) error {
			return scenarioexec.RunSubprocess(spec)
		},
		ScenarioExecutableFn: os.Executable,
	})
}

// newUninstaller is the only process-entry construction point for the real
// file remover. The orchestration package receives only its one-method seam,
// which keeps package tests incapable of reaching host deletion accidentally.
func newUninstaller(root, home string) (cliinstall.Uninstaller, error) {
	paths, err := trustposture.ResolveKeyPaths()
	if err != nil {
		return nil, err
	}
	verify := func(token string, now time.Time) error {
		public, err := os.ReadFile(paths.Public)
		if err != nil {
			return fmt.Errorf("read pinned break-glass public key: %w", err)
		}
		target, err := os.Hostname()
		if err != nil || strings.TrimSpace(target) == "" {
			return fmt.Errorf("resolve local break-glass target: %w", err)
		}
		claims, err := trustposture.Verify(ed25519.PublicKey(public), token, cliinstall.UninstallBreakGlassAudience, target, now)
		if err != nil {
			return err
		}
		for _, scope := range claims.Scopes {
			if scope == "*" || scope == cliinstall.UninstallBreakGlassScope || scope == "vrooli:*" {
				return nil
			}
		}
		return fmt.Errorf("break-glass: scope %q is required", cliinstall.UninstallBreakGlassScope)
	}
	boundVerify := func(token string, request cliinstall.UninstallRequest, plan cliinstall.UninstallPlan, now time.Time) error {
		public, err := os.ReadFile(paths.Public)
		if err != nil {
			return fmt.Errorf("read pinned break-glass public key: %w", err)
		}
		target, err := os.Hostname()
		if err != nil || strings.TrimSpace(target) == "" {
			return fmt.Errorf("resolve local break-glass target: %w", err)
		}
		claims, err := trustposture.VerifyBound(ed25519.PublicKey(public), token, cliinstall.UninstallBreakGlassAudience, target, trustposture.BreakGlassBinding{
			OperatorID:  request.AuthorizingUser,
			MachineID:   request.MachineID,
			NodeID:      request.NodeID,
			Scope:       string(plan.Scope),
			PlanHash:    plan.PlanHash,
			OperationID: request.OperationID,
		}, now)
		if err != nil {
			return err
		}
		for _, scope := range claims.Scopes {
			if scope == "*" || scope == cliinstall.UninstallBreakGlassScope || scope == "vrooli:*" {
				return nil
			}
		}
		return fmt.Errorf("break-glass: scope %q is required", cliinstall.UninstallBreakGlassScope)
	}
	deferredServiceNames := strings.Split(os.Getenv("VROOLI_BRIDGE_DEFER_SERVICE_STOPS"), ",")
	return cliinstall.NewUninstallService(root, home, cliinstall.NewFileRemover(home, deferredServiceNames...), verify, cliinstall.WithBoundBreakGlassVerifier(boundVerify))
}

// reassertCodingAgentShims repairs the coding-agent alias set on process entry.
//
// The alias set is five links reconstructible from a binary already on disk, so
// re-asserting it is cheaper than assuming nothing ever removes it -- and
// something does: the shared install root is churned by many installers, and
// the shims have already been observed to vanish from it with no record of what
// took them. Repairing at the point of use rather than only at setup time is
// what turns that from an outage lasting until the next `vrooli setup
// --include-optional` into a gap lasting until the next `vrooli` command.
//
// It runs here, in the control-plane binary, and not in vrooli-agent-launcher.
// The launcher is only reached through a shim, so a launcher-side repair could
// never fix the case that matters: the shim being gone.
//
// Cost on a healthy host is one Stat for the launcher plus one Lstat per alias.
// Every failure is silent by design. Attribution is observability, and a
// process must not print a warning, slow down, or fail because an optional
// convenience could not be repaired.
func reassertCodingAgentShims() {
	defer func() {
		// Nothing in this path may take down a CLI invocation.
		_ = recover()
	}()
	_, _ = codingagentshims.EnsureInstalled()
}
