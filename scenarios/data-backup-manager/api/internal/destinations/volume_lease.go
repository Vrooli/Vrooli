package destinations

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"data-backup-manager/internal/destinationreadiness"
)

// DefaultLeaseReadyTimeout bounds how long DBM waits for the control plane's
// readiness handshake before giving up on the lease. Without it, a hung
// `vrooli host volume lease` freezes an entire backup run indefinitely with a
// stopped heartbeat and no operator-visible failure. The bound applies only to
// the handshake; an acquired lease stays held for the whole operation.
const DefaultLeaseReadyTimeout = 30 * time.Second

// VolumeOperationLease holds the host owner's cooperative lock for a strong
// volume identity. DBM keeps the returned release function alive across its
// rebind and complete resource-kopia subprocess operation.
type VolumeOperationLease interface {
	Acquire(context.Context, destinationreadiness.DeviceIdentity) (func(), error)
}

// HostVolumeOperationLease uses the supported control-plane CLI hold command.
// It fails closed if that command cannot prove the requested identity and
// return its readiness handshake.
type HostVolumeOperationLease struct {
	Binary       string
	ReadyTimeout time.Duration
}

func (l HostVolumeOperationLease) readyTimeout() time.Duration {
	if l.ReadyTimeout > 0 {
		return l.ReadyTimeout
	}
	return DefaultLeaseReadyTimeout
}

var _ VolumeOperationLease = HostVolumeOperationLease{}

func NewHostVolumeOperationLease(binary string) HostVolumeOperationLease {
	return HostVolumeOperationLease{Binary: strings.TrimSpace(binary)}
}

func (l HostVolumeOperationLease) Acquire(ctx context.Context, identity destinationreadiness.DeviceIdentity) (func(), error) {
	if ctx == nil {
		return nil, errors.New("host volume lease requires a context")
	}
	if !identity.StrongIdentity() || strings.TrimSpace(identity.DevicePath) == "" {
		return nil, errors.New("host volume lease requires a device path and stable UUID or serial")
	}
	binary := strings.TrimSpace(l.Binary)
	if binary == "" {
		return nil, errors.New("host volume lease CLI binary is required")
	}
	args := []string{"host", "volume", "lease", "--device", identity.DevicePath, "--json"}
	if strings.TrimSpace(identity.UUID) != "" {
		args = append(args, "--uuid", identity.UUID)
	}
	if strings.TrimSpace(identity.Serial) != "" {
		args = append(args, "--serial", identity.Serial)
	}
	if strings.TrimSpace(identity.Filesystem) != "" {
		args = append(args, "--filesystem", identity.Filesystem)
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	configureLeaseProcess(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open host volume lease readiness stream: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start host volume lease: %w", err)
	}
	reader := bufio.NewReader(stdout)
	// Read the handshake in a goroutine so a hung control plane cannot block
	// forever. The command itself is NOT tied to this timer: once the handshake
	// arrives, the lease process must stay alive to keep the lease held, and it
	// is ended by the caller's release or by the parent context.
	type handshake struct {
		line string
		err  error
	}
	done := make(chan handshake, 1)
	go func() {
		line, err := reader.ReadString('\n')
		done <- handshake{line: line, err: err}
	}()
	var line string
	var readErr error
	select {
	case result := <-done:
		line, readErr = result.line, result.err
	case <-time.After(l.readyTimeout()):
		killLeaseProcess(cmd)
		return nil, fmt.Errorf("host volume lease did not become ready within %s; the control plane lease owner did not answer", l.readyTimeout())
	case <-ctx.Done():
		killLeaseProcess(cmd)
		return nil, fmt.Errorf("host volume lease canceled before ready: %w", ctx.Err())
	}
	if readErr != nil {
		killLeaseProcess(cmd)
		return nil, fmt.Errorf("host volume lease did not become ready: %s", leaseCommandError(line, readErr, nil, stderr.String()))
	}
	var ready struct {
		Status string `json:"status"`
		Device string `json:"device"`
	}
	if err := json.Unmarshal([]byte(line), &ready); err != nil || ready.Status != "leased" || ready.Device != identity.DevicePath {
		killLeaseProcess(cmd)
		return nil, fmt.Errorf("host volume lease returned an invalid readiness record: %s", leaseCommandError(line, err, nil, stderr.String()))
	}

	var once sync.Once
	return func() {
		once.Do(func() {
			killLeaseProcess(cmd)
		})
	}, nil
}

func leaseCommandError(line string, readErr, waitErr error, stderr string) string {
	message := strings.TrimSpace(stderr)
	if message == "" {
		message = strings.TrimSpace(line)
	}
	if message == "" {
		message = "command exited before its readiness handshake"
	}
	return fmt.Sprintf("%s (read=%v, wait=%v)", message, readErr, waitErr)
}
