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
		Mode:                  modeToProto(c.Mode),
		TunnelId:              c.TunnelID,
		AccountId:             c.AccountID,
		CredRef:               c.CredRef,
		PromEndpoint:          c.PromEndpoint,
		PublicExposureEnabled: c.PublicExposureEnabled,
	}
}

// accessStatusToProto converts the internal /public Access-bypass read model
// into its wire shape.
func accessStatusToProto(s internalconfig.AccessStatus) *configv1.AccessStatus {
	hosts := make([]*configv1.AccessHostState, 0, len(s.Hosts))
	for _, h := range s.Hosts {
		hosts = append(hosts, &configv1.AccessHostState{
			Host:            h.Host,
			Override:        string(h.Override),
			EffectiveBypass: h.EffectiveBypass,
			Managed:         h.Managed,
			AppId:           h.AppID,
		})
	}
	return &configv1.AccessStatus{
		Enabled:    s.Enabled,
		Configured: s.Configured,
		Hosts:      hosts,
		ToCreate:   s.ToCreate,
		ToRemove:   s.ToRemove,
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

// verificationToProto converts a live CredentialVerification into the wire
// shape. Secret values never appear in CredentialCheck, so the mapping is a
// straight field copy.
func verificationToProto(v internalconfig.CredentialVerification) *configv1.VerifyCredentialsResponse {
	checks := make([]*configv1.CredentialCheck, 0, len(v.Checks))
	for _, c := range v.Checks {
		checks = append(checks, &configv1.CredentialCheck{
			Name:        c.Name,
			State:       checkStateToProto(c.State),
			Detail:      c.Detail,
			Remediation: c.Remediation,
		})
	}
	return &configv1.VerifyCredentialsResponse{Checks: checks, Ready: v.Ready}
}

func checkStateToProto(s internalconfig.CheckState) configv1.CheckState {
	switch s {
	case internalconfig.CheckOK:
		return configv1.CheckState_CHECK_STATE_OK
	case internalconfig.CheckMissing:
		return configv1.CheckState_CHECK_STATE_MISSING
	case internalconfig.CheckInvalid:
		return configv1.CheckState_CHECK_STATE_INVALID
	case internalconfig.CheckInsufficientScope:
		return configv1.CheckState_CHECK_STATE_INSUFFICIENT_SCOPE
	default:
		return configv1.CheckState_CHECK_STATE_UNSPECIFIED
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
		Mode:           modeToProto(r.Mode),
		Added:          r.Added,
		Removed:        r.Removed,
		NoChanges:      r.NoChanges,
		SetupRequired:  r.SetupRequired,
		MissingFields:  r.MissingFields,
		Message:        r.Message,
		DriftUnmanaged: r.DriftUnmanaged,
		Orphaned:       r.Orphaned,
		Pruned:         r.Pruned,
	}
}

// driftReportToProto converts a DriftReport into the GetDriftResponse wire
// shape.
func driftReportToProto(rep internalconfig.DriftReport) *configv1.GetDriftResponse {
	entries := make([]*configv1.IngressEntry, 0, len(rep.Entries))
	for _, e := range rep.Entries {
		entries = append(entries, ingressEntryToProto(e))
	}
	return &configv1.GetDriftResponse{
		Mode:    modeToProto(rep.Mode),
		Entries: entries,
		Counts: &configv1.DriftCounts{
			Managed:    int32(rep.Counts[internalconfig.StateManaged]),
			Missing:    int32(rep.Counts[internalconfig.StateMissing]),
			ExternalOk: int32(rep.Counts[internalconfig.StateExternalOK]),
			Orphaned:   int32(rep.Counts[internalconfig.StateOrphaned]),
			Ignored:    int32(rep.Counts[internalconfig.StateIgnored]),
			Unmanaged:  int32(rep.Counts[internalconfig.StateUnmanaged]),
		},
	}
}

func ingressEntryToProto(e internalconfig.IngressEntry) *configv1.IngressEntry {
	return &configv1.IngressEntry{
		Hostname:      e.Hostname,
		ServiceTarget: e.ServiceTarget,
		State:         ownershipStateToProto(e.State),
		Source:        ingressSourceToProto(e.Source),
		Scenario:      e.Scenario,
		LeaseId:       e.LeaseID,
		Note:          e.Note,
	}
}

func ownershipStateToProto(s internalconfig.OwnershipState) configv1.OwnershipState {
	switch s {
	case internalconfig.StateManaged:
		return configv1.OwnershipState_OWNERSHIP_STATE_MANAGED
	case internalconfig.StateMissing:
		return configv1.OwnershipState_OWNERSHIP_STATE_MISSING
	case internalconfig.StateExternalOK:
		return configv1.OwnershipState_OWNERSHIP_STATE_EXTERNAL_OK
	case internalconfig.StateOrphaned:
		return configv1.OwnershipState_OWNERSHIP_STATE_ORPHANED
	case internalconfig.StateIgnored:
		return configv1.OwnershipState_OWNERSHIP_STATE_IGNORED
	case internalconfig.StateUnmanaged:
		return configv1.OwnershipState_OWNERSHIP_STATE_UNMANAGED
	default:
		return configv1.OwnershipState_OWNERSHIP_STATE_UNSPECIFIED
	}
}

func ingressSourceToProto(s internalconfig.RouteSource) configv1.IngressSource {
	switch s {
	case internalconfig.SourceScenario:
		return configv1.IngressSource_INGRESS_SOURCE_SCENARIO
	case internalconfig.SourceExternal:
		return configv1.IngressSource_INGRESS_SOURCE_EXTERNAL
	default:
		return configv1.IngressSource_INGRESS_SOURCE_UNSPECIFIED
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
