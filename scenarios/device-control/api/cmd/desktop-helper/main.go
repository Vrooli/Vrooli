package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"device-control/internal/desktophelper"
	"device-control/internal/environment"
	"github.com/vrooli/api-core/preflight"
)

func main() {
	// The lifecycle builds this named component; API-default self-rebuild would
	// select the wrong main package for the helper.
	if preflight.Run(preflight.Config{ScenarioName: "device-control", DisableStaleness: true}) {
		return
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	processEnv := environment.OSReader{}
	session := desktophelper.SessionEnvironment{SessionType: environment.Value(processEnv, "XDG_SESSION_TYPE"), WaylandDisplay: environment.Value(processEnv, "WAYLAND_DISPLAY")}
	run(ctx, environment.Value(processEnv, "DEVICE_CONTROL_DESKTOP_HELPER_CONFIG"), environment.Value(processEnv, "DEVICE_CONTROL_DESKTOP_OWNER_CONFIG"), session)
}

func run(ctx context.Context, helperConfig, ownerConfig string, session desktophelper.SessionEnvironment) {
	for ctx.Err() == nil {
		err := desktophelper.RunBound(ctx, helperConfig, ownerConfig, session)
		if ctx.Err() != nil {
			return
		}
		// A locked or unavailable optional desktop must not tear down the API/UI.
		// Process liveness is not capture/input readiness; admission still requires
		// the bound socket, signed grant, and fresh native/session checks.
		slog.Error("desktop helper unavailable; retrying bootstrap", "error", err)
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
