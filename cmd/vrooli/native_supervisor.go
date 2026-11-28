package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"syscall"

	"github.com/vrooli/repo-contract-go/cliinvoke"
	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
)

// Native startup accepts exactly the catalogued supervisor invocation. It
// precedes session/shim/source initialization and uses the authority embedded
// by the governed build, so a missing checkout cannot stop lease supervision.
// Unstamped development binaries retain their existing source-bound CLI path;
// their empty authority digest makes them ineligible for native qualification.
func runNativeSupervisorCommand(args []string, stderr io.Writer) (bool, int) {
	if !slices.Equal(args, cliinvoke.RuntimeSupervisorRun()) || os.Getenv(runtimesupervisor.ModeEnv) != runtimesupervisor.ModeOn || buildinfo.RuntimeHomeSnapshotBase64 == "" {
		return false, 0
	}
	if err := buildinfo.PinCompiledRuntimeHome(); err != nil {
		_, _ = fmt.Fprintln(stderr, "native supervisor bootstrap refused: compiled runtime-home authority is invalid")
		return true, 1
	}
	if err := guardRootActivationEntrypoint(true); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return true, 1
	}
	home, err := config.HomeDir()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "native supervisor bootstrap could not resolve the invoking user home")
		return true, 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	app := runtimeapp.Service{Version: vrooliVersion, ResolveRootFn: buildinfo.ResolveSourceRoot}
	if err := app.RunSupervisor(ctx, home, runtimeapp.SupervisorRunOptions{}); err != nil {
		if errors.Is(err, context.Canceled) && ctx.Err() != nil {
			return true, 0
		}
		_, _ = fmt.Fprintf(stderr, "native supervisor stopped: %v\n", err)
		return true, 1
	}
	return true, 0
}
