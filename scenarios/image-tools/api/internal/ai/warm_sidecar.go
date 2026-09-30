package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

type warmRunner interface {
	Run(ctx context.Context, program string, args []string) error
	Close() error
}

type warmSidecarRunner struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	encoder *json.Encoder
	decoder *json.Decoder
}

func newWarmSidecarRunner() *warmSidecarRunner {
	return &warmSidecarRunner{}
}

type warmRequest struct {
	Argv []string `json:"argv"`
}

type warmResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Trace string `json:"trace,omitempty"`
}

func (r *warmSidecarRunner) Run(ctx context.Context, program string, args []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.ensureStarted(program); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() {
		if err := r.encoder.Encode(warmRequest{Argv: args}); err != nil {
			done <- err
			return
		}
		var resp warmResponse
		if err := r.decoder.Decode(&resp); err != nil {
			done <- err
			return
		}
		if !resp.OK {
			if resp.Trace != "" {
				done <- fmt.Errorf("%s\n%s", resp.Error, resp.Trace)
				return
			}
			done <- fmt.Errorf("%s", resp.Error)
			return
		}
		done <- nil
	}()

	select {
	case <-ctx.Done():
		_ = r.stopLocked() // best-effort teardown; we are already returning the cancel error
		return ctx.Err()
	case err := <-done:
		if err != nil {
			_ = r.stopLocked() // best-effort teardown; we are already returning the worker error
			return err
		}
		return nil
	}
}

func (r *warmSidecarRunner) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stopLocked()
}

func (r *warmSidecarRunner) ensureStarted(program string) error {
	if r.cmd != nil && r.cmd.Process != nil {
		return nil
	}
	cmd := exec.Command(program, "-m", "image_tools_sidecar.worker")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("warm sidecar stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("warm sidecar stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("warm sidecar stderr: %w", err)
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("warm sidecar start: %w", err)
	}
	r.cmd = cmd
	r.stdin = stdin
	r.encoder = json.NewEncoder(stdin)
	r.decoder = json.NewDecoder(stdout)
	go func() { _, _ = io.Copy(io.Discard, stderr) }()
	go func() { _ = cmd.Wait() }()
	return nil
}

func (r *warmSidecarRunner) stopLocked() error {
	var err error
	if r.stdin != nil {
		err = r.stdin.Close()
	}
	if r.cmd != nil && r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
	}
	r.cmd = nil
	r.stdin = nil
	r.encoder = nil
	r.decoder = nil
	return err
}
