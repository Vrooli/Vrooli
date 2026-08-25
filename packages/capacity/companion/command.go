package companion

import (
	"context"
	"flag"
	"io"
	"os"
	"os/exec"
	"os/signal"
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
	// production shell and stderr when left nil.
	Config Config
	// Stderr receives flag-parse errors and ledger warnings. nil means
	// os.Stderr.
	Stderr io.Writer
}

// DefaultExec shells the on-PATH vrooli binary.
func DefaultExec(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

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
	if err := set.Parse(args); err != nil {
		return err
	}
	cfg.Interval = *interval

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
