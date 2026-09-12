package devices

import (
	"device-sync-hub/internal/devices"

	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/devices"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// deviceToProto converts an internal devices.Device into the wire shape the
// devices proto declares. The conversion is mechanical and used only at the
// transport edge — the domain layer never imports proto.
func deviceToProto(d devices.Device) *devicesv1.Device {
	return &devicesv1.Device{
		Id:           d.ID,
		OwnerId:      d.OwnerID,
		Name:         d.Name,
		Kind:         d.Kind,
		Platform:     d.Platform,
		Capabilities: append([]string(nil), d.Capabilities...),
		TrustState:   trustToProto(d.TrustState),
		Online:       d.Online,
		LastSeenAt:   timestamppb.New(d.LastSeenAt.UTC()),
		CreatedAt:    timestamppb.New(d.CreatedAt.UTC()),
		UpdatedAt:    timestamppb.New(d.UpdatedAt.UTC()),
	}
}

// trustToProto maps the domain trust string to the proto enum.
func trustToProto(s devices.TrustState) devicesv1.TrustState {
	switch s {
	case devices.TrustPending:
		return devicesv1.TrustState_TRUST_STATE_PENDING
	case devices.TrustTrusted:
		return devicesv1.TrustState_TRUST_STATE_TRUSTED
	case devices.TrustRevoked:
		return devicesv1.TrustState_TRUST_STATE_REVOKED
	default:
		return devicesv1.TrustState_TRUST_STATE_UNSPECIFIED
	}
}

// pairingCodeToProto converts a freshly-issued code (raw value present).
func pairingCodeToProto(c devices.PairingCode) *devicesv1.PairingCode {
	return &devicesv1.PairingCode{
		Code:      c.Code,
		OwnerId:   c.OwnerID,
		ExpiresAt: timestamppb.New(c.ExpiresAt.UTC()),
		CreatedAt: timestamppb.New(c.CreatedAt.UTC()),
	}
}

// profileFromProto converts a wire DeviceProfile into the domain Profile.
// A nil proto profile yields the zero Profile (service fills the defaults).
func profileFromProto(p *devicesv1.DeviceProfile) devices.Profile {
	if p == nil {
		return devices.Profile{}
	}
	return devices.Profile{
		Name:         p.DeviceName,
		Kind:         p.Kind,
		Platform:     p.Platform,
		Capabilities: append([]string(nil), p.Capabilities...),
	}
}
