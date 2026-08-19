package volumeremediation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/vrooli/vrooli/internal/privilegebroker"
)

// BrokerElevated routes privileged filesystem actions through the
// setup-installed privilege broker. It is the only elevation path in this
// package: `vrooli setup` installs the broker, the broker validates every
// request against its immutable registry, and nothing here spawns sudo.
type BrokerElevated struct {
	client *privilegebroker.Client
	// newRequestID is a seam so tests get deterministic correlation ids.
	newRequestID func() string
}

var _ Elevated = (*BrokerElevated)(nil)

// NewBrokerElevated constructs the production elevation client.
func NewBrokerElevated() *BrokerElevated {
	return &BrokerElevated{client: privilegebroker.NewClient()}
}

// Available reports whether the broker socket is reachable.
func (b *BrokerElevated) Available() bool {
	return b != nil && b.client.Available()
}

// CheckFilesystem asks the broker for a non-writing consistency check.
func (b *BrokerElevated) CheckFilesystem(ctx context.Context, device Device) (string, string, error) {
	return b.do(ctx, privilegebroker.ActionVolumeFilesystemCheck, device)
}

// RepairFilesystem asks the broker to repair the filesystem.
func (b *BrokerElevated) RepairFilesystem(ctx context.Context, device Device) (string, string, error) {
	return b.do(ctx, privilegebroker.ActionVolumeFilesystemRepair, device)
}

func (b *BrokerElevated) do(ctx context.Context, action string, device Device) (string, string, error) {
	if b == nil || b.client == nil {
		return StatusUnsupported, "", ErrUnsupported{Reason: "privilege broker client is not configured", OperatorCommand: "sudo vrooli setup"}
	}
	result, err := b.client.Do(ctx, privilegebroker.Request{
		Version:   privilegebroker.ProtocolVersion,
		RequestID: b.requestID(),
		Action:    action,
		Volume: &privilegebroker.VolumeSubject{
			Device:     device.Path,
			Filesystem: device.Filesystem,
			UUID:       device.UUID,
			Serial:     device.Serial,
		},
	})
	if err != nil {
		return StatusFailed, "", err
	}
	return translateBrokerResult(result)
}

// translateBrokerResult maps the broker's vocabulary onto this package's.
// A broker refusal stays a refusal here: the two layers gate independently and
// either one saying no is final.
func translateBrokerResult(result privilegebroker.Result) (string, string, error) {
	detail := result.Evidence.Detail
	switch result.Status {
	case "verified":
		return StatusVerified, detail, nil
	case "changed":
		return StatusChanged, detail, nil
	case "already_present":
		return StatusAlreadySatisfied, detail, nil
	case "unavailable":
		return StatusUnsupported, detail, ErrUnsupported{
			Reason:          "the privilege broker reported the filesystem tool is unavailable: " + result.Code,
			OperatorCommand: "install the filesystem tools for this volume, then retry",
		}
	default:
		switch result.Code {
		case "volume_mounted":
			return StatusRefused, detail, ErrRefused{Reason: "the privilege broker refused: the volume is still mounted"}
		case "system_volume_refused":
			return StatusRefused, detail, ErrRefused{Reason: "the privilege broker refused: this is a system volume"}
		case "device_identity_mismatch":
			return StatusRefused, detail, ErrRefused{Reason: "the privilege broker could not confirm the device identity"}
		case "action_not_allowed", "volume_subject_required", "invalid_device", "filesystem_not_allowed", "device_identity_required":
			return StatusRefused, detail, ErrRefused{Reason: "the privilege broker rejected the request: " + result.Code}
		}
		return StatusFailed, detail, fmt.Errorf("privilege broker %s: %s", result.Status, result.Code)
	}
}

func (b *BrokerElevated) requestID() string {
	if b != nil && b.newRequestID != nil {
		return b.newRequestID()
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "volume-remediation"
	}
	return "volume-remediation-" + hex.EncodeToString(buf)
}
