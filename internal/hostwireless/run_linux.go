//go:build linux

package hostwireless

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

func openBundleFile(root *os.Root, name string) (*os.File, error) {
	return root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
}

func runBuild(ctx context.Context, work, abi, phase string, aliases []string) error {
	logFile, err := os.Create(filepath.Join(work, phase+".log"))
	if err != nil {
		return err
	}
	defer logFile.Close()
	home := filepath.Join(work, "build-home")
	if err := os.MkdirAll(home, 0700); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/bash", append([]string{"--noprofile", "--norc", "build.sh", phase, abi}, aliases...)...)
	cmd.Dir = work
	cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LC_ALL=C", "HOME=" + home, "TMPDIR=" + work,
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null"}
	cmd.Stdout, cmd.Stderr = logFile, logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = 2 * time.Second
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	if err := cmd.Run(); err != nil {
		info, statErr := logFile.Stat()
		if statErr != nil {
			return fmt.Errorf("wireless %s build: %w", phase, err)
		}
		start := max(0, info.Size()-4096)
		reader, openErr := os.Open(logFile.Name())
		if openErr != nil {
			return fmt.Errorf("wireless %s build: %w", phase, err)
		}
		defer reader.Close()
		tail, _ := io.ReadAll(io.NewSectionReader(reader, start, 4096))
		return fmt.Errorf("wireless %s build: %w\n%s", phase, err, tail)
	}
	return nil
}
