package config

import (
	internalconfig "tunnel-manager/internal/config"

	configv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config"
)

// domainConfigToProto converts an internal TunnelConfig into the wire
// shape the config proto declares. Lives in the handler package by intent
// — the conversion is mechanical and only used at the transport edge.
func domainConfigToProto(c internalconfig.TunnelConfig) *configv1.TunnelConfig {
	return &configv1.TunnelConfig{
		Mode:         modeToProto(c.Mode),
		TunnelId:     c.TunnelID,
		AccountId:    c.AccountID,
		CredRef:      c.CredRef,
		PromEndpoint: c.PromEndpoint,
	}
}

// syncResultToProto converts a SyncResult into the SyncResponse wire shape.
func syncResultToProto(r internalconfig.SyncResult) *configv1.SyncResponse {
	return &configv1.SyncResponse{
		Mode:      modeToProto(r.Mode),
		Added:     r.Added,
		Removed:   r.Removed,
		NoChanges: r.NoChanges,
	}
}

func modeToProto(m internalconfig.Mode) configv1.Mode {
	switch m {
	case internalconfig.ModeRemote:
		return configv1.Mode_MODE_REMOTE
	case internalconfig.ModeLocal:
		return configv1.Mode_MODE_LOCAL
	default:
		return configv1.Mode_MODE_UNSPECIFIED
	}
}

// modeFromProto maps a wire mode to the domain mode. MODE_UNSPECIFIED
// returns the empty Mode so the service applies its default/validation.
func modeFromProto(m configv1.Mode) internalconfig.Mode {
	switch m {
	case configv1.Mode_MODE_REMOTE:
		return internalconfig.ModeRemote
	case configv1.Mode_MODE_LOCAL:
		return internalconfig.ModeLocal
	default:
		return internalconfig.ModeUnspecified
	}
}
