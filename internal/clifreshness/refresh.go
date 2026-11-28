package clifreshness

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/setup"
)

const (
	// EnvDisable turns local refresh off for one process tree.
	EnvDisable = "VROOLI_LOCAL_REFRESH"
	// EnvDepth marks a process that already refreshed once. It is the loop
	// guard: a refresh that could not change the command surface must not try
	// again.
	EnvDepth = "VROOLI_LOCAL_REFRESH_DEPTH"
)

// LocalCLIPath is the repository's own CLI build. It is deliberately separate
// from the managed image in the runtime home, which root activation owns.
func LocalCLIPath(root string) string {
	return filepath.Join(root, ".vrooli", "build", "vrooli")
}

// Seams keep the decision testable without compiling a CLI or replacing a
// process during a test.
var (
	surfaceDigestFn  = CheckoutSurfaceDigest
	embeddedDigestFn = EmbeddedSurfaceDigest
	buildLocalCLI    = setup.BuildLocalCLI
	activationActive = activationInProgress
	runRefreshedCLI  = reexec
	executablePath   = os.Executable
)

// MaybeReexec refreshes the checkout's CLI and replaces this process when its
// command surface is older than the checkout. It reports the exit code and
// whether it re-executed; when it returns false the caller runs normally.
func MaybeReexec(ctx context.Context, root string, args []string, stderr io.Writer) (int, bool) {
	if disabled() || serviceContext() || strings.TrimSpace(os.Getenv(EnvDepth)) != "" {
		return 0, false
	}
	if skipForCommand(args) || strings.TrimSpace(root) == "" {
		return 0, false
	}
	if activationActive() {
		return 0, false
	}
	wanted, err := surfaceDigestFn(root)
	if err != nil {
		// Not a checkout, or the checkout's manifest is unreadable.
		return 0, false
	}
	local := LocalCLIPath(root)
	running, _ := executablePath()
	if publishedSurface(local) == wanted {
		// A build for exactly this surface is already published. The only
		// question left is whether this process is that build.
		if running == local {
			return 0, false
		}
		return runRefreshedCLI(local, args, stderr)
	}
	if running != local && embeddedDigestFn != nil {
		if embedded, embeddedErr := embeddedDigestFn(); embeddedErr == nil && embedded == wanted {
			// This build already serves the checkout and no local build is
			// needed.
			return 0, false
		}
	}
	if err := os.MkdirAll(filepath.Dir(local), 0o755); err != nil {
		return 0, false
	}
	_, _ = fmt.Fprintln(stderr, "vrooli: this build's command list is older than cli/manifest.json; refreshing the local CLI from source (one time).")
	if err := buildUnderLock(ctx, root, local, wanted); err != nil {
		_, _ = fmt.Fprintf(stderr, "vrooli: local refresh skipped (%v); continuing with this build.\n", firstLine(err))
		return 0, false
	}
	return runRefreshedCLI(local, args, stderr)
}

// publishedSurface is the command surface of the last local build, or "" when
// there is none. It lets every invocation after the first take a single small
// file read instead of rebuilding.
func publishedSurface(local string) string {
	data, err := os.ReadFile(local + ".surface")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// buildUnderLock serializes refreshes across concurrent agents. A sibling that
// already published a fresh build makes this cheap: the surface is rechecked
// inside the lock.
func buildUnderLock(ctx context.Context, root, local, wanted string) error {
	lockCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	release, err := platform.AcquireFileLockContext(lockCtx, local+".lock")
	if err != nil {
		return fmt.Errorf("another refresh holds the build lock: %w", err)
	}
	defer release()
	// A sibling may have published this exact surface while we waited.
	if publishedSurface(local) == wanted {
		return nil
	}
	if err := buildLocalCLI(ctx, root, local); err != nil {
		return err
	}
	return writePublishedSurface(local, wanted)
}

func writePublishedSurface(local, wanted string) error {
	temporary := fmt.Sprintf("%s.surface.tmp.%d", local, os.Getpid())
	if err := os.WriteFile(temporary, []byte(wanted+"\n"), 0o644); err != nil {
		return err
	}
	defer func() { _ = os.Remove(temporary) }()
	return platform.AtomicReplace(temporary, local+".surface")
}

func reexec(local string, args []string, stderr io.Writer) (int, bool) {
	command := exec.Command(local, args...)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	command.Env = append(os.Environ(), EnvDepth+"=1")
	err := command.Run()
	if err == nil {
		return 0, true
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), true
	}
	_, _ = fmt.Fprintf(stderr, "vrooli: could not run the refreshed CLI (%v); continuing with this build.\n", firstLine(err))
	return 0, false
}

func disabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvDisable))) {
	case "0", "false", "off", "no":
		return true
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv(buildinfo.ArtifactModeEnvVar)), "1") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv(buildinfo.ArtifactModeEnvVar)), "true")
}

// serviceContext reports whether this process was started by a service manager
// rather than by a person or an agent.
//
// A service must never re-exec a different image. The supervisor's identity is
// its unit's MainPID and the image that PID runs; the lease it renews must name
// the same build the transition installed. A child process running a
// development build satisfies neither, and the transition's own verification
// then refuses the image it just installed. systemd and launchd both leave
// unambiguous markers, and the supervisor marks its own environment.
func serviceContext() bool {
	for _, key := range []string{
		"INVOCATION_ID",  // systemd unit
		"NOTIFY_SOCKET",  // systemd readiness protocol
		"JOURNAL_STREAM", // systemd log forwarding
		"VROOLI_RUNTIME_SUPERVISOR",
		"LAUNCH_JOB_NAME",  // launchd
		"XPC_SERVICE_NAME", // launchd
	} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}
	return false
}

// activationInProgress reports whether a root activation owns the serving
// image. Replacing what this process runs during the window between activation
// and verification would make the running executable unlike either the prior or
// the candidate image, which the entrypoint guard must refuse.
func activationInProgress() bool {
	home, err := config.HomeDir()
	if err != nil {
		return true
	}
	store, err := coreactivation.NewStore(home)
	if err != nil {
		return true
	}
	record, exists, err := store.Current()
	if err != nil {
		// An initialized store whose record cannot be read is the owner's
		// problem to recover; a refresh must not add a second moving part.
		return true
	}
	return exists && !record.Terminal()
}

// longLivedCommands are commands that either are services themselves or own a
// process tree for the length of a session. Replacing the executable under them
// changes the process's identity while it is being observed.
var longLivedCommands = map[string]bool{
	"runtime supervisor": true,
	"runtime watchdog":   true,
	"runtime update":     true,
	"agent launch":       true,
	"agent exec":         true,
	"agent review":       true,
	"watchdog":           true,
}

// skipForCommand avoids a refresh for invocations that either do not run a
// command, rebuild everything themselves, or must keep one process identity.
func skipForCommand(args []string) bool {
	rebuildsItself := map[string]bool{"build": true, "install": true, "develop": true, "setup": true}
	words := make([]string, 0, 2)
	for _, arg := range args {
		switch arg {
		case "--help", "-h", "help":
			return true
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if strings.HasPrefix(arg, "__") {
			// Internal entry points are reached by a service or a supervisor.
			return true
		}
		words = append(words, arg)
		if len(words) == 2 {
			break
		}
	}
	command := ""
	if len(words) > 0 {
		command = words[0]
	}
	switch command {
	case "", "version", "completion":
		return true
	}
	if rebuildsItself[command] || longLivedCommands[command] {
		return true
	}
	return len(words) > 1 && longLivedCommands[command+" "+words[1]]
}

func firstLine(err error) string {
	line, _, _ := strings.Cut(err.Error(), "\n")
	return strings.TrimSpace(line)
}
