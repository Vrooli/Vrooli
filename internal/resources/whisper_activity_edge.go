package resources

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/packages/capacity/activityedge"
)

func controlWhisperActivityEdge(ctx context.Context, action string) error {
	return controlResourceActivityEdge(ctx, "whisper", action)
}

func controlResourceActivityEdge(ctx context.Context, resourceName, action string) error {
	if !hasStableActivityEdge(resourceName) {
		return fmt.Errorf("resource %q has no stable activity edge", resourceName)
	}
	dir, err := resolveCompanionDir(resourceName)
	if err != nil {
		return fmt.Errorf("resolve %s companion owner directory: %w", resourceName, err)
	}
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return activityedge.ControlSocketRequest(callCtx, filepath.Join(dir, "activity-edge.control.sock"), action)
}

// removeStaleWhisperControlSocket only removes a socket that is unreachable,
// belongs to the exact dead companion recorded in both owner pid files, and
// still has socket type. A live listener, symlink, regular file, or uncertain
// dial result remains untouched and blocks replacement.
func removeStaleWhisperControlSocket(path string, deadPID int) error {
	return removeStaleActivityEdgeControlSocket(path, deadPID)
}

func removeStaleActivityEdgeControlSocket(path string, deadPID int) error {
	if deadPID <= 1 || process.IsPIDRunning(deadPID) {
		return fmt.Errorf("activity-edge PID %d is not proven dead", deadPID)
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("control path exists with non-socket type %s", info.Mode().Type())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	conn, dialErr := (&net.Dialer{}).DialContext(ctx, "unix", path)
	if dialErr == nil {
		_ = conn.Close()
		return fmt.Errorf("control socket is still accepting connections")
	}
	var opErr *net.OpError
	if !errors.As(dialErr, &opErr) || !errors.Is(dialErr, syscall.ECONNREFUSED) {
		return fmt.Errorf("control socket liveness is uncertain: %w", dialErr)
	}
	if process.IsPIDRunning(deadPID) {
		return fmt.Errorf("activity-edge PID %d became live during socket recovery", deadPID)
	}
	current, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("recheck stale control socket before removal: %w", err)
	}
	if !os.SameFile(info, current) || current.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("control path changed during stale socket recovery")
	}
	return os.Remove(path)
}
