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

	"data-backup-manager/internal/destinationreadiness"
)

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
	Binary string
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
	line, readErr := reader.ReadString('\n')
	if readErr != nil {
		_ = cmd.Process.Kill()
		waitErr := cmd.Wait()
		return nil, fmt.Errorf("host volume lease did not become ready: %s", leaseCommandError(line, readErr, waitErr, stderr.String()))
	}
	var ready struct {
		Status string `json:"status"`
		Device string `json:"device"`
	}
	if err := json.Unmarshal([]byte(line), &ready); err != nil || ready.Status != "leased" || ready.Device != identity.DevicePath {
		_ = cmd.Process.Kill()
		waitErr := cmd.Wait()
		return nil, fmt.Errorf("host volume lease returned an invalid readiness record: %s", leaseCommandError(line, err, waitErr, stderr.String()))
	}

	var once sync.Once
	return func() {
		once.Do(func() {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
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
