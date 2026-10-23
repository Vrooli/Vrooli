package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

type warmRunner interface {
	Run(ctx context.Context, program string, args []string) error
	Close() error
}

type warmSidecarRunner struct {
	mu          sync.Mutex
	gate        chan struct{}
	cmd         *exec.Cmd
	exited      chan struct{}
	stdin       io.WriteCloser
	encoder     *json.Encoder
	decoder     *json.Decoder
	idleTimeout time.Duration
	idleTimer   *time.Timer
	activity    uint64
}

func newWarmSidecarRunner() *warmSidecarRunner {
	return &warmSidecarRunner{gate: make(chan struct{}, 1), idleTimeout: 30 * time.Second}
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
	select {
	case r.gate <- struct{}{}:
		defer func() { <-r.gate }()
	case <-ctx.Done():
		return ctx.Err()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	r.activity++
	if r.idleTimer != nil {
		r.idleTimer.Stop()
		r.idleTimer = nil
	}

	if err := r.ensureStarted(program); err != nil {
		return err
	}
	done := make(chan error, 1)
	// Capture this process's streams. Cancellation may clear runner state while
	// the decoder finishes; it must never read the next process's stream.
	encoder, decoder := r.encoder, r.decoder
	go func() {
		if err := encoder.Encode(warmRequest{Argv: args}); err != nil {
			done <- err
			return
		}
		var resp warmResponse
		if err := decoder.Decode(&resp); err != nil {
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
		<-done
		return ctx.Err()
	case err := <-done:
		if err != nil {
			_ = r.stopLocked() // best-effort teardown; we are already returning the worker error
			return err
		}
		if admitted, _ := ctx.Value(operationMemoryContextKey{}).(bool); admitted {
			// This request owns an operation RAM lease. Reap under the same
			// mutex/gate, before another request can acquire this worker and
			// before the engine releases the lease. Unleased callers retain
			// the existing bounded idle-reuse behavior.
			return r.stopLocked()
		}
		activity := r.activity
		r.idleTimer = time.AfterFunc(r.idleTimeout, func() {
			r.mu.Lock()
			defer r.mu.Unlock()
			if r.activity == activity {
				_ = r.stopLocked()
			}
		})
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
	r.exited = make(chan struct{})
	r.stdin = stdin
	r.encoder = json.NewEncoder(stdin)
	r.decoder = json.NewDecoder(stdout)
	go func() { _, _ = io.Copy(io.Discard, stderr) }()
	exited := r.exited
	go func() { _ = cmd.Wait(); close(exited) }()
	return nil
}

func (r *warmSidecarRunner) stopLocked() error {
	r.activity++
	if r.idleTimer != nil {
		r.idleTimer.Stop()
		r.idleTimer = nil
	}
	var err error
	if r.stdin != nil {
		err = r.stdin.Close()
	}
	if r.cmd != nil && r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
		// Reap before returning: the caller may release capacity immediately.
		<-r.exited
	}
	r.cmd = nil
	r.stdin = nil
	r.encoder = nil
	r.decoder = nil
	r.exited = nil
	return err
}
