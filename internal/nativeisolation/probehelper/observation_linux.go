//go:build linux

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/vrooli/vrooli/internal/nativeisolation/probeprotocol"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type fixedObservationEffects struct{}

func (fixedObservationEffects) markers(o probeprotocol.Observation) error {
	cwd, e := os.Getwd()
	if e != nil || cwd != o.Repository {
		return errors.New("fixed marker check refused")
	}
	for _, m := range []struct {
		path, hash string
		deny       bool
	}{{o.WorkspaceMarker(), o.WorkspaceSHA256, false}, {probeprotocol.DeniedMarker, o.DeniedSHA256, true}} {
		info, e := os.Lstat(m.path)
		if e != nil || !info.Mode().IsRegular() || info.Size() > 1024 {
			return errors.New("fixed marker check refused")
		}
		fd, e := syscall.Open(m.path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		if m.deny {
			if !errors.Is(e, syscall.EACCES) && !errors.Is(e, syscall.EPERM) {
				if e == nil {
					syscall.Close(fd)
				}
				return errors.New("marker read was not denied")
			}
			continue
		}
		if e != nil {
			return errors.New("workspace marker unreadable")
		}
		f := os.NewFile(uintptr(fd), m.path)
		raw, re := io.ReadAll(io.LimitReader(f, 1025))
		f.Close()
		hash := sha256.Sum256(raw)
		if re != nil || len(raw) > 1024 || hex.EncodeToString(hash[:]) != m.hash {
			return errors.New("workspace marker mismatch")
		}
	}
	return nil
}
func (fixedObservationEffects) child(_ context.Context, o probeprotocol.Observation) (int, func() error, error) {
	// Fixed pinned self executable only; no command, destination or input seam.
	cmd := exec.Command(probeprotocol.Helper, "--descendant", "--until", o.Deadline.UTC().Format(time.RFC3339Nano))
	cmd.Env = []string{"LANG=C", "LC_ALL=C"}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if e := cmd.Start(); e != nil {
		return 0, nil, e
	}
	cleanup := func() error {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case e := <-done:
			return e
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
			select {
			case e := <-done:
				return e
			case <-time.After(time.Second):
				return errors.New("descendant cleanup unconfirmed")
			}
		}
	}
	return cmd.Process.Pid, cleanup, nil
}
func (fixedObservationEffects) report(raw []byte) error {
	if len(raw) > probeprotocol.MaxReportBytes {
		return errors.New("report refused")
	}
	_, e := os.Stdout.Write(raw)
	return e
}
func (fixedObservationEffects) parentPID() int { return os.Getpid() }
func (fixedObservationEffects) wait(ctx context.Context, d time.Time) {
	timer := time.NewTimer(time.Until(d))
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func (fixedObservationEffects) now() time.Time { return time.Now().UTC() }
