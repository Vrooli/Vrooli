package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"image-tools/internal/jobs"
)

type warmRunner interface {
	Run(ctx context.Context, program string, args []string) error
	Close() error
}

type warmSidecarRunner struct {
	mu            sync.Mutex
	gate          chan struct{}
	cmd           *exec.Cmd
	exited        chan struct{}
	stdin         io.WriteCloser
	encoder       *json.Encoder
	decoder       *json.Decoder
	idleTimeout   time.Duration
	idleTimer     *time.Timer
	activity      uint64
	resident      *leaseGroup
	residentModel string
	residencyID   string
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
	reserved, _ := ctx.Value(warmReservationKey{}).(*warmSidecarRunner)
	if reserved != r {
		select {
		case r.gate <- struct{}{}:
			defer func() { <-r.gate }()
		case <-ctx.Done():
			return ctx.Err()
		}
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

	if err := r.ensureStarted(ctx, program); err != nil {
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
		if reserved == r {
			r.recordResidencyPeakRSSLocked(ctx)
			return nil
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

func (r *warmSidecarRunner) ensureStarted(ctx context.Context, program string) error {
	if r.cmd != nil && r.cmd.Process != nil {
		return nil
	}
	cmd := inferenceCommand(context.WithoutCancel(ctx), program, "-m", "image_tools_sidecar.worker")
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
	if r.resident != nil {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		releaseErr := r.resident.release(cleanup)
		cancel()
		if releaseErr != nil {
			return fmt.Errorf("release warm model residency: %w", releaseErr)
		}
		r.resident, r.residentModel, r.residencyID = nil, "", ""
	}
	return err
}

type warmReservationKey struct{}

// reserve obtains exclusive use of this warm worker without blocking a
// scheduler slot. Its resident lease belongs to the worker, while the returned
// job lease only owns the current request. Model switches reap before release.
func (r *warmSidecarRunner) reserve(ctx context.Context, model, jobID string, acquire func(context.Context) (*leaseGroup, error), resultMeta map[string]string) (jobs.Admission, error) {
	select {
	case r.gate <- struct{}{}:
	default:
		return jobs.Admission{}, fmt.Errorf("%w: warm worker is active", jobs.ErrCapacityBusy)
	}
	r.mu.Lock()
	if err := ctx.Err(); err != nil {
		r.mu.Unlock()
		<-r.gate
		return jobs.Admission{}, err
	}
	if r.idleTimer != nil {
		r.idleTimer.Stop()
		r.idleTimer = nil
	}
	r.activity++
	reused := r.resident != nil && r.residentModel == model && r.cmd != nil
	if (r.resident != nil && (r.residentModel != model || r.resident.ctx.Err() != nil)) || (r.resident == nil && r.cmd != nil) {
		if err := r.stopLocked(); err != nil {
			r.mu.Unlock()
			<-r.gate
			return jobs.Admission{}, err
		}
		reused = false
	}
	if r.resident == nil {
		residentCtx, cancelResident := context.WithCancelCause(context.WithoutCancel(ctx))
		stopAcquisition := context.AfterFunc(ctx, func() { cancelResident(context.Cause(ctx)) })
		resident, err := acquire(residentCtx)
		stopAcquisition()
		if ctx.Err() != nil {
			err = errors.Join(err, context.Cause(ctx))
		}
		if resident != nil {
			resident.stops = append(resident.stops, func() { cancelResident(context.Canceled) })
		}
		if err != nil {
			cancelResident(err)
			if resident != nil {
				cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
				if releaseErr := resident.release(cleanup); releaseErr != nil {
					err = fmt.Errorf("warm admission rollback: %v; %w", err, releaseErr)
				}
				cancel()
			}
			r.mu.Unlock()
			<-r.gate
			return jobs.Admission{}, err
		}
		r.resident, r.residentModel = resident, model
		r.residencyID = "image-tools-worker-residency:" + jobID
	}
	residencyID := r.residencyID
	residentCtx := r.resident.ctx
	r.mu.Unlock()
	if resultMeta != nil {
		resultMeta[workerResidencyIDMetadataKey] = residencyID
		resultMeta[workerResidencyModelMetadataKey] = model
	}
	jobCtx, cancel := context.WithCancelCause(ctx)
	jobCtx = context.WithValue(jobCtx, workerResidencyMetadataKey{}, resultMeta)
	stop := context.AfterFunc(residentCtx, func() { cancel(context.Cause(residentCtx)) })
	var once sync.Once
	var releaseErr error
	return jobs.Admission{
		Context: context.WithValue(jobCtx, warmReservationKey{}, r),
		Meta:    map[string]string{"warm_reused": fmt.Sprint(reused), "residency_model": model, "residency_idle_seconds": fmt.Sprint(r.idleTimeout.Seconds()), workerResidencyIDMetadataKey: residencyID, workerResidencyModelMetadataKey: model},
		Release: func(context.Context) error {
			once.Do(func() {
				r.mu.Lock()
				if jobCtx.Err() != nil || r.cmd == nil {
					releaseErr = r.stopLocked()
				} else {
					activity := r.activity
					r.idleTimer = time.AfterFunc(r.idleTimeout, func() {
						r.mu.Lock()
						defer r.mu.Unlock()
						if r.activity == activity {
							_ = r.stopLocked()
						}
					})
				}
				r.mu.Unlock()
				stop()
				cancel(context.Canceled)
				<-r.gate
			})
			return releaseErr
		},
	}, nil
}

const (
	workerResidencyIDMetadataKey      = "worker_residency_id"
	workerResidencyModelMetadataKey   = "worker_residency_model"
	workerResidencyPIDMetadataKey     = "worker_residency_pid"
	workerResidencyPeakRSSBytesKey    = "worker_residency_peak_rss_bytes"
	workerResidencyPeakRSSSourceKey   = "worker_residency_peak_rss_source"
	workerResidencyPeakRSSStatusKey   = "worker_residency_peak_rss_status"
	workerResidencyPeakRSSReasonKey   = "worker_residency_peak_rss_reason"
	workerResidencyPeakRSSLinuxSource = "linux_proc_status_vm_hwm_residency"
)

type workerResidencyMetadataKey struct{}

// recordResidencyPeakRSSLocked snapshots the warm process high-water mark
// after a successful request. Run holds r.mu while calling it. It deliberately
// reports the entire worker/model residency lifetime, not an isolated per-job
// peak.
func (r *warmSidecarRunner) recordResidencyPeakRSSLocked(ctx context.Context) {
	meta, _ := ctx.Value(workerResidencyMetadataKey{}).(map[string]string)
	if meta == nil {
		return
	}

	id := meta[workerResidencyIDMetadataKey]
	model := meta[workerResidencyModelMetadataKey]
	pid := 0
	if r.cmd != nil && r.cmd.Process != nil {
		pid = r.cmd.Process.Pid
	}
	meta[workerResidencyPIDMetadataKey] = strconv.Itoa(pid)
	setUnavailable := func(reason string) {
		delete(meta, workerResidencyPeakRSSBytesKey)
		delete(meta, workerResidencyPeakRSSSourceKey)
		meta[workerResidencyPeakRSSStatusKey] = "unavailable"
		meta[workerResidencyPeakRSSReasonKey] = reason
	}
	if id == "" || id != r.residencyID || model == "" || model != r.residentModel {
		setUnavailable("worker model residency identity changed before RSS capture")
		return
	}
	if pid <= 0 {
		setUnavailable("warm worker PID is unavailable")
		return
	}
	if runtime.GOOS != "linux" {
		setUnavailable("worker residency high-water RSS is unsupported on " + runtime.GOOS)
		return
	}
	bytes, err := workerResidencyPeakRSSBytes(pid)
	if err != nil {
		setUnavailable(err.Error())
		return
	}
	meta[workerResidencyPeakRSSBytesKey] = strconv.FormatInt(bytes, 10)
	meta[workerResidencyPeakRSSSourceKey] = workerResidencyPeakRSSLinuxSource
	meta[workerResidencyPeakRSSStatusKey] = "measured"
	delete(meta, workerResidencyPeakRSSReasonKey)
}

// workerResidencyPeakRSSBytes reads Linux VmHWM for the exact warm worker PID.
// VmHWM is a process-lifetime high-water mark; it cannot attribute memory to
// one inference and must only be interpreted with a worker residency identity.
func workerResidencyPeakRSSBytes(pid int) (int64, error) {
	if pid <= 0 {
		return 0, fmt.Errorf("warm worker PID is unavailable")
	}
	if runtime.GOOS != "linux" {
		return 0, fmt.Errorf("worker residency high-water RSS is unsupported on %s", runtime.GOOS)
	}
	status, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, fmt.Errorf("read Linux worker status for PID %d: %w", pid, err)
	}
	for _, line := range strings.Split(string(status), "\n") {
		if !strings.HasPrefix(line, "VmHWM:") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "VmHWM:"))
		if len(fields) != 2 || fields[1] != "kB" {
			return 0, fmt.Errorf("parse Linux worker VmHWM for PID %d: malformed value", pid)
		}
		kib, parseErr := strconv.ParseInt(fields[0], 10, 64)
		if parseErr != nil || kib <= 0 || kib > int64(^uint64(0)>>1)/1024 {
			return 0, fmt.Errorf("parse Linux worker VmHWM for PID %d: invalid high-water value", pid)
		}
		return kib * 1024, nil
	}
	return 0, fmt.Errorf("Linux worker status for PID %d has no VmHWM field", pid)
}
