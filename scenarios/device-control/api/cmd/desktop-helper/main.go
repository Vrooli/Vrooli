package main

import (
	"context"
	"device-control/internal/desktophelper"
	"github.com/vrooli/api-core/preflight"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// The lifecycle builds this named component; API-default self-rebuild would
	// select the wrong main package for the helper.
	if preflight.Run(preflight.Config{ScenarioName: "device-control", DisableStaleness: true}) {
		return
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	for ctx.Err() == nil {
		err := desktophelper.RunBound(ctx, os.Getenv("DEVICE_CONTROL_DESKTOP_HELPER_CONFIG"), os.Getenv("DEVICE_CONTROL_DESKTOP_OWNER_CONFIG"))
		if ctx.Err() != nil {
			return
		}
		// A locked or unavailable optional desktop must not tear down the API/UI.
		// Process liveness is not capture/input readiness; admission still requires
		// the bound socket, signed grant, and fresh native/session checks.
		log.Print("desktop helper unavailable; retrying bootstrap: ", err)
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
