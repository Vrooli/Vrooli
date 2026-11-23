package companion

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// The flag set, the signal handling and the exit contract were written three
// times, once per resource CLI, with three slightly different sets of signals
// handled. They are written once here, so a resource CLI declares what it
// observes and nothing else.

// CommandOptions are the pieces a resource CLI supplies to get a complete
// capacity-sync command.
type CommandOptions struct {
	// Config is the companion declaration. Its Exec and Log default to the
	// production shell and stderr when left nil. The resource's declared claim
	// (kind, tier, idle policy) is applied over it by Run.
	Config Config
	// Claim reads the resource's declared claim. nil reads resource.json
	// through DeclaredClaim.
	Claim ClaimSource
	// Stderr receives flag-parse errors and ledger warnings. nil means
	// os.Stderr.
	Stderr io.Writer
}

// DefaultExec shells the on-PATH vrooli binary. A failing command's stderr is
// part of the returned error: the verbs report why they refused there, and a
// bare "exit status 1" cannot tell a lost generation race from a real failure.
func DefaultExec(ctx context.Context, name string, args ...string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, name, args...).Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if detail := strings.TrimSpace(string(exitErr.Stderr)); detail != "" {
			if len(detail) > maxExecErrorDetail {
				detail = detail[:maxExecErrorDetail]
			}
			err = fmt.Errorf("%w: %s", err, detail)
		}
	}
	return out, err
}

const maxExecErrorDetail = 400

// Run parses the shared flags and drives the loop until signalled.
//
// The flags are the same three every companion had: --interval, --once, and
// nothing else. --once exists so a test or a cron entry can reconcile without
// owning a process.
func Run(options CommandOptions, args []string) error {
	stderr := options.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	cfg := options.Config
	if cfg.Exec == nil {
		cfg.Exec = DefaultExec
	}
	if cfg.Log == nil {
		cfg.Log = stderr
	}
	set := flag.NewFlagSet("capacity-sync", flag.ContinueOnError)
	set.SetOutput(stderr)
	interval := set.Duration("interval", cfg.interval(), "poll interval")
	once := set.Bool("once", false, "run one reconciliation and exit")
	parentPID := set.Int("parent-pid", cfg.ParentPID, "PID of the resource process that owns this companion")
	if err := set.Parse(args); err != nil {
		return err
	}
	cfg.Interval = *interval
	cfg.ParentPID = *parentPID
	if cfg.ParentPID <= 1 {
		return fmt.Errorf("capacity companion: --parent-pid is required and must be greater than 1")
	}

	// A companion that cannot read its manifest would create claims without the
	// idle policy the resource declared, which is the failure this exists to
	// prevent; refusing to start is visible, while a silent default is not.
	claim := options.Claim
	if claim == nil {
		claim = DeclaredClaim(os.Getenv)
	}
	spec, declared, err := claim(cfg.Resource)
	if err != nil {
		return fmt.Errorf("capacity companion: read declared claim for %s: %w", cfg.Resource, err)
	}
	if !declared {
		return fmt.Errorf("capacity companion: %s declares no acceleration.claim in resource.json", cfg.Resource)
	}
	cfg = cfg.WithClaimSpec(spec)

	runner, err := New(cfg)
	if err != nil {
		return err
	}
	if *once {
		runner.SyncOnce(context.Background())
		return nil
	}

	// A companion is a reporter, not a supervisor: it stops when asked and
	// never blocks the resource's own shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return runner.Run(ctx)
}

// PollInterval resolves a declared interval from an environment override,
// falling back to the given default. It exists so every companion reads its
// override the same way.
func PollInterval(getenv func(string) string, key string, fallback time.Duration) time.Duration {
	if getenv == nil {
		return fallback
	}
	if parsed, err := time.ParseDuration(getenv(key)); err == nil && parsed > 0 {
		return parsed
	}
	return fallback
}
