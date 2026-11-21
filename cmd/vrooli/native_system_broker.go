package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	platform "github.com/vrooli/platform-go"
	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/buildinfo"
)

type nativeSystemBrokerEntrypoint struct {
	read      func(string) (platform.NativeSystemServiceBrokerOptions, error)
	pin       func() error
	guard     func(platform.NativeSystemServiceBrokerOptions) error
	supervise func(context.Context, platform.NativeSystemServiceBrokerOptions, func(context.Context) error) error
	run       func(context.Context, platform.NativeSystemServiceBrokerOptions) error
}

func runNativeSystemBrokerCommand(args []string, stderr io.Writer) (bool, int) {
	if len(args) == 0 || args[0] != platform.NativeSystemSupervisorVerb {
		return false, 0
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	effects := nativeSystemBrokerEntrypoint{read: platform.ReadNativeSystemServiceBrokerDescriptor, pin: buildinfo.PinCompiledRuntimeHome, guard: guardRootActivationForNativeBroker, supervise: platform.RunNativeSystemSupervisor, run: func(ctx context.Context, o platform.NativeSystemServiceBrokerOptions) error {
		app := runtimeapp.Service{Version: vrooliVersion, ResolveRootFn: func() (string, error) { return runtimeapp.ResolveNativeOperatorSource(o.OperatorHome) }}
		return app.RunSupervisor(ctx, o.OperatorHome, runtimeapp.SupervisorRunOptions{Takeover: true})
	}}
	return effects.dispatch(ctx, args, stderr)
}

func (e nativeSystemBrokerEntrypoint) dispatch(ctx context.Context, args []string, stderr io.Writer) (bool, int) {
	if len(args) == 0 || args[0] != platform.NativeSystemSupervisorVerb {
		return false, 0
	}
	if len(args) != 2 {
		fmt.Fprintln(stderr, "native system supervisor requires exactly one installed descriptor")
		return true, 1
	}
	o, err := e.read(args[1])
	if err == nil {
		err = e.pin()
	}
	if err == nil {
		err = e.guard(o)
	}
	if err == nil {
		err = e.supervise(ctx, o, func(ctx context.Context) error { return e.run(ctx, o) })
	}
	if err != nil {
		if errors.Is(err, context.Canceled) && ctx.Err() != nil {
			return true, 0
		}
		fmt.Fprintf(stderr, "native system supervisor stopped: %v\n", err)
		return true, 1
	}
	return true, 0
}
