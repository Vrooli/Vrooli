package vroolicli

import (
	"context"
	"fmt"
	"strings"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// VolumeAction is the CLI vocabulary of `vrooli host volume`. It is the same
// size as the control plane's remediation registry: there is no format,
// partition, or clear action to reach through this wrapper.
type VolumeAction string

const (
	// VolumeInspect observes a volume without changing anything.
	VolumeInspect VolumeAction = "inspect"
	// VolumeCheck verifies filesystem consistency without writing.
	VolumeCheck VolumeAction = "check"
	// VolumeRepair writes filesystem metadata corrections. Requires
	// AcknowledgeDataLoss.
	VolumeRepair VolumeAction = "repair"
	// VolumeUnmount detaches the volume so a check or repair can run.
	VolumeUnmount VolumeAction = "unmount"
	// VolumeMountReadWrite returns the volume to a writable mount.
	VolumeMountReadWrite VolumeAction = "mount-rw"
)

// VolumeRequest addresses one volume action. Device is required; the identity
// and filesystem fields are optional expectations — supplying them makes the
// control plane refuse if the host disagrees, and omitting them lets it fill in
// what the host reports.
type VolumeRequest struct {
	Action     VolumeAction
	Device     string
	Filesystem string
	UUID       string
	Serial     string
	// Mountpoint is the target for VolumeMountReadWrite.
	Mountpoint string
	// AcknowledgeDataLoss is required for VolumeRepair. The control plane
	// enforces this independently; setting it here only lets the request reach
	// that gate.
	AcknowledgeDataLoss bool
	DryRun              bool
}

// HostVolume runs one `vrooli host volume <action> --json` operation and
// returns the typed result.
//
// A refused or unsupported action is a *result*, not a transport error: the CLI
// exits non-zero for them precisely so a shell caller can branch, and this
// wrapper still returns the decoded body so a Go caller can read the reason and
// the operator command instead of losing them to an exit status. The error
// return is reserved for genuinely not getting an answer.
func (c *Client) HostVolume(ctx context.Context, req VolumeRequest) (*cliv1.VolumeRemediationResponse, error) {
	if strings.TrimSpace(req.Device) == "" {
		return nil, fmt.Errorf("host volume: device is required")
	}
	if strings.TrimSpace(string(req.Action)) == "" {
		return nil, fmt.Errorf("host volume: action is required")
	}
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	args := []string{"host", "volume", string(req.Action), "--device", req.Device, "--json"}
	// Ordered, not a map range: argv order must be deterministic so the command
	// recorded in an audit trail is reproducible.
	for _, opt := range []struct{ flag, value string }{
		{"--filesystem", req.Filesystem},
		{"--uuid", req.UUID},
		{"--serial", req.Serial},
		{"--mountpoint", req.Mountpoint},
	} {
		if strings.TrimSpace(opt.value) != "" {
			args = append(args, opt.flag, opt.value)
		}
	}
	if req.AcknowledgeDataLoss {
		args = append(args, "--acknowledge-data-loss")
	}
	if req.DryRun {
		args = append(args, "--dry-run")
	}

	out, runErr := c.runKeepingOutput(ctx, args...)
	if len(strings.TrimSpace(string(out))) == 0 {
		if runErr != nil {
			return nil, fmt.Errorf("host volume %s: %w", req.Action, runErr)
		}
		return nil, fmt.Errorf("host volume %s: no output", req.Action)
	}
	resp, err := decode(out, &cliv1.VolumeRemediationResponse{})
	if err != nil {
		if runErr != nil {
			return nil, fmt.Errorf("host volume %s: %w", req.Action, runErr)
		}
		return nil, fmt.Errorf("host volume %s: %w", req.Action, err)
	}
	return resp, nil
}

// runKeepingOutput executes a CLI invocation and preserves stdout alongside any
// error. The base run helper discards stdout on failure, which would throw away
// the typed body a refused action reports — and `host volume` exits non-zero for
// exactly those results. The caller decides which of the two to trust by
// whether the body decodes.
func (c *Client) runKeepingOutput(ctx context.Context, args ...string) ([]byte, error) {
	out, err := c.runner.Run(ctx, c.bin, args...)
	if err != nil {
		return out, fmt.Errorf("run %s: %w", formatCommand(c.bin, args), err)
	}
	return out, nil
}

// VolumeChanged reports whether a response represents an actual host change.
func VolumeChanged(resp *cliv1.VolumeRemediationResponse) bool {
	return resp != nil && resp.GetChanged()
}

// VolumeSatisfied reports whether the requested end state holds, whether or not
// this call is what produced it. A retried recovery step that finds its work
// already done is a success, not a failure.
func VolumeSatisfied(resp *cliv1.VolumeRemediationResponse) bool {
	if resp == nil {
		return false
	}
	switch resp.GetStatus() {
	case "verified", "changed", "already_satisfied":
		return true
	default:
		return false
	}
}
