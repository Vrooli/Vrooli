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

func readinessToProto(r internalconfig.ConfigReadiness) *configv1.ConfigReadiness {
	return &configv1.ConfigReadiness{
		DesiredMode:      modeToProto(r.DesiredMode),
		RemoteAvailable:  r.RemoteAvailable,
		MissingFields:    r.MissingFields,
		CredentialSource: r.CredentialSource,
		CredentialRef:    r.CredentialRef,
		LocalConfigPath:  r.LocalConfigPath,
		SyncReady:        r.SyncReady,
		ModeReason:       r.ModeReason,
		CredentialFields: credentialFieldsToProto(r.CredentialStatus.Fields),
	}
}

func credentialStatusToProto(status internalconfig.CredentialStatus) *configv1.CredentialStatus {
	return &configv1.CredentialStatus{
		Fields:        credentialFieldsToProto(status.Fields),
		MissingFields: status.MissingFields,
		Source:        status.Source,
		Ref:           status.Ref,
		Ready:         status.Ready,
	}
}

func credentialFieldsToProto(fields []internalconfig.CredentialFieldStatus) []*configv1.CredentialFieldStatus {
	out := make([]*configv1.CredentialFieldStatus, 0, len(fields))
	for _, field := range fields {
		out = append(out, &configv1.CredentialFieldStatus{
			Name:     field.Name,
			Present:  field.Present,
			Source:   field.Source,
			Ref:      field.Ref,
			Writable: field.Writable,
		})
	}
	return out
}

// syncResultToProto converts a SyncResult into the SyncResponse wire shape.
func syncResultToProto(r internalconfig.SyncResult) *configv1.SyncResponse {
	return &configv1.SyncResponse{
		Mode:          modeToProto(r.Mode),
		Added:         r.Added,
		Removed:       r.Removed,
		NoChanges:     r.NoChanges,
		SetupRequired: r.SetupRequired,
		MissingFields: r.MissingFields,
		Message:       r.Message,
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
