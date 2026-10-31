package workload

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	platform "github.com/vrooli/platform-go"
)

func runCommand(ctx context.Context, dir string, args []string, output string) error {
	stdout, err := os.OpenFile(filepath.Join(output, "stdout.txt"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer stdout.Close()
	stderr, err := os.OpenFile(filepath.Join(output, "stderr.txt"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer stderr.Close()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	invocation, err := json.Marshal(map[string]any{"executable": cmd.Path, "args": cmd.Args, "directory": cmd.Dir})
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(output, "command.json"), invocation, 0600); err != nil {
		return err
	}
	cmd.Stdout = &boundedWriter{out: stdout, remaining: 1 << 20, cancel: cancel}
	cmd.Stderr = &boundedWriter{out: stderr, remaining: 1 << 20, cancel: cancel}
	cmd.WaitDelay = 3 * time.Second
	if err := platform.ConfigureCommand(cmd, platform.ProcessOptions{Detached: true}); err != nil {
		return err
	}
	cmd.Cancel = func() error { return platform.GracefulStopProcess(cmd.Process) }
	if err := cmd.Start(); err != nil {
		return err
	}
	// Kill only this command's owned tree, using the existing platform owner.
	defer func() { _ = platform.SignalProcessGroup(cmd.Process.Pid, true) }()
	release, err := platform.AssignProcessContainment(cmd.Process)
	if err != nil {
		_ = platform.SignalProcessGroup(cmd.Process.Pid, true)
		return errors.Join(err, cmd.Wait())
	}
	defer release()
	return cmd.Wait()
}

type boundedWriter struct {
	out       io.Writer
	remaining int
	cancel    context.CancelFunc
}

func (w *boundedWriter) Write(p []byte) (int, error) {
	if len(p) > w.remaining {
		n, _ := w.out.Write(p[:w.remaining])
		w.remaining = 0
		w.cancel()
		return n, errors.New("workload command exceeded retained output limit")
	}
	w.remaining -= len(p)
	return w.out.Write(p)
}
