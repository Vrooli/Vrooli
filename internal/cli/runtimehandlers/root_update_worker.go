package runtimehandlers

import (
	"context"
	"io"
	"os/exec"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// runRootUpdateWorker launches the already-resolved watchdog worker for an
// ordinary apply/retry request. It does not inspect prepared-root recovery
// records; the bounded call shape is guarded by the buildinfo AST test.
func runRootUpdateWorker(parent context.Context, worker string, args []string, stdout, stderr io.Writer) error {
	ctx, cancel := context.WithTimeout(parent, 8*time.Minute)
	defer cancel()
	name, wrapped := hostreqkit.InvokingUserCommand(worker, args...)
	cmd := exec.CommandContext(ctx, name, wrapped...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.WaitDelay = time.Second
	return cmd.Run()
}
