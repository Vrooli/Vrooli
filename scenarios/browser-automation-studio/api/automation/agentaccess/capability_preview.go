package agentaccess

import (
	"context"
	"errors"
	"fmt"

	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

// CapabilityProfileSummary is the effective, run-scoped permission profile
// resolved by BAS from the current Agent Manager grant. It contains opaque
// credential references only, never credential values.
type CapabilityProfileSummary struct {
	ProfileID            string                                    `json:"profile_id"`
	ProfileMode          string                                    `json:"profile_mode"`
	Effect               sessionpermission.Effect                  `json:"effect"`
	Approval             sessionpermission.Approval                `json:"approval"`
	URLAllow             []string                                  `json:"url_allow,omitempty"`
	URLDeny              []string                                  `json:"url_deny,omitempty"`
	View                 bool                                      `json:"view"`
	Control              bool                                      `json:"control"`
	Handoff              bool                                      `json:"handoff"`
	CredentialRefs       []string                                  `json:"credential_refs,omitempty"`
	ExtensionAccess      bool                                      `json:"extension_access"`
	ExtensionInstall     bool                                      `json:"extension_install"`
	ExtensionLoad        bool                                      `json:"extension_load"`
	ExtensionPermissions map[string][]string                       `json:"extension_permissions,omitempty"`
	HistoryRead          bool                                      `json:"history_read"`
	HistoryExport        bool                                      `json:"history_export"`
	HistoryRetentionMS   int64                                     `json:"history_retention_ms,omitempty"`
	MaxActions           int64                                     `json:"max_actions,omitempty"`
	MaxDurationMS        int64                                     `json:"max_duration_ms,omitempty"`
	MaxCredits           int64                                     `json:"max_credits,omitempty"`
	MaxBytes             int64                                     `json:"max_bytes,omitempty"`
	AudioMode            sessionpermission.AudioMode               `json:"audio_mode,omitempty"`
	IdentityProfile      *sessionpermission.BrowserIdentityProfile `json:"identity_profile,omitempty"`
}

// ModeCapabilityPreview pairs the existing runtime qualification row with
// the active grant's mode compatibility and the backend's identity limits.
type ModeCapabilityPreview struct {
	sessionpermission.CapabilityNegotiation
	ProfileCompatible     bool                                      `json:"profile_compatible"`
	ProfileReason         string                                    `json:"profile_reason,omitempty"`
	PermissionDenied      []sessionpermission.Capability            `json:"permission_denied,omitempty"`
	UnsupportedOperations []string                                  `json:"unsupported_operations,omitempty"`
	OpenRequestAllowed    bool                                      `json:"open_request_allowed"`
	NextStep              string                                    `json:"next_step,omitempty"`
	IdentityBehavior      string                                    `json:"identity_behavior"`
	ConfiguredIdentity    *sessionpermission.BrowserIdentityProfile `json:"configured_identity,omitempty"`
}

// CapabilityProfilePreview is a pre-open view over existing Agent Manager,
// BAS Broker and runtime qualification owners. It creates no browser session.
type CapabilityProfilePreview struct {
	PermissionProfile CapabilityProfileSummary       `json:"permission_profile"`
	GrantedBackend    sessionpermission.Backend      `json:"granted_backend"`
	Requested         []sessionpermission.Capability `json:"requested,omitempty"`
	Required          []sessionpermission.Capability `json:"required"`
	Modes             []ModeCapabilityPreview        `json:"modes"`
}

// CapabilityPreview resolves the current trusted grant and returns its exact
// effective profile alongside all four backend negotiation rows. The active
// grant admits one profile mode; other backends remain visible but cannot be
// opened under that grant.
func (b *Broker) CapabilityPreview(ctx context.Context, requested []sessionpermission.Capability) (CapabilityProfilePreview, error) {
	b.mu.RLock()
	openContext, profiles := b.openContext, b.profiles
	b.mu.RUnlock()
	if openContext == nil || profiles == nil {
		return CapabilityProfilePreview{}, errors.New("trusted Agent Manager profile context is required for capability preview")
	}
	trusted, err := openContext.ResolveBrowserOpen(ctx)
	if err != nil {
		return CapabilityProfilePreview{}, fmt.Errorf("resolve trusted Agent Manager browser grant: %w", err)
	}
	if trusted.CallerID == "" || trusted.GrantReceipt == "" || trusted.IdentityToken == "" || trusted.PermissionRef == "" {
		return CapabilityProfilePreview{}, errors.New("trusted Agent Manager caller, grant, identity token and profile reference are required")
	}
	profile, policy, err := profiles.Resolve(ctx, trusted.PermissionRef)
	if err != nil {
		return CapabilityProfilePreview{}, fmt.Errorf("resolve effective Agent Manager permission profile: %w", err)
	}
	if policy == nil {
		return CapabilityProfilePreview{}, ErrAuthorizerRequired
	}
	if profile.ID == "" {
		profile.ID = trusted.PermissionRef
	}
	if _, err := sessionpermission.New(profile, nil); err != nil {
		return CapabilityProfilePreview{}, fmt.Errorf("resolved browser permission profile is invalid: %w", err)
	}
	grantedBackend, ok := backendForProfileMode(profile.ProfileMode)
	if !ok {
		return CapabilityProfilePreview{}, fmt.Errorf("unsupported Agent Manager browser profile mode %q", profile.ProfileMode)
	}
	requested = uniqueCapabilities(requested)
	required := uniqueCapabilities(append(append([]sessionpermission.Capability(nil), requested...), requiredCapabilities(profile, string(grantedBackend))...))
	negotiated, err := b.Capabilities(ctx, required)
	if err != nil {
		return CapabilityProfilePreview{}, err
	}
	modes := make([]ModeCapabilityPreview, 0, len(negotiated))
	for _, row := range negotiated {
		expectedMode, _ := profileModeForBackend(string(row.Backend))
		compatible := row.Backend == grantedBackend && profile.ProfileMode == expectedMode
		mode := ModeCapabilityPreview{CapabilityNegotiation: row, ProfileCompatible: compatible, IdentityBehavior: identityBehavior(row.Backend)}
		mode.UnsupportedOperations = unsupportedProfileOperations(profile, row.Backend)
		for _, capability := range requested {
			if !profileAllowsCapability(profile, row.Backend, capability) {
				mode.PermissionDenied = append(mode.PermissionDenied, capability)
			}
		}
		mode.OpenRequestAllowed = compatible && row.Accepted && len(mode.PermissionDenied) == 0 && len(mode.UnsupportedOperations) == 0 && row.Backend != sessionpermission.BackendRemote
		if !compatible {
			mode.ProfileReason = fmt.Sprintf("active Agent Manager grant is %q; backend %q requires profile mode %q", profile.ProfileMode, row.Backend, expectedMode)
		}
		if compatible && profile.IdentityProfile != nil && (row.Backend == sessionpermission.BackendManaged || row.Backend == sessionpermission.BackendInstalled) {
			identity := *profile.IdentityProfile
			mode.ConfiguredIdentity = &identity
		}
		if row.Backend == sessionpermission.BackendRemote {
			mode.OpenRequestAllowed = false
			mode.ProfileReason = "remote browser backend adapter is unavailable; BAS will not fall back to a local session"
		}
		if row.Backend == sessionpermission.BackendUserAttached && compatible {
			mode.NextStep = "a person must select the tab and complete the existing popup consent flow before attachment"
		}
		modes = append(modes, mode)
	}
	return CapabilityProfilePreview{
		PermissionProfile: summarizePermissionProfile(profile), GrantedBackend: grantedBackend,
		Requested: requested, Required: required, Modes: modes,
	}, nil
}

func unsupportedProfileOperations(profile sessionpermission.Profile, backend sessionpermission.Backend) []string {
	var unsupported []string
	if backend == sessionpermission.BackendRemote {
		return []string{"all session operations: no remote BAS session adapter is configured"}
	}
	if backend == sessionpermission.BackendUserAttached {
		if profile.Effect == sessionpermission.EffectUnrestricted {
			unsupported = append(unsupported, "unrestricted or raw-protocol access")
		}
		if profile.HistoryRead || profile.HistoryExport || profile.UserStateExport || len(profile.CredentialIDs) > 0 {
			unsupported = append(unsupported, "history, user-state export, and credential use")
		}
		if profile.ExtensionLoad || profile.ExtensionInstall {
			unsupported = append(unsupported, "extension install and load")
		}
	}
	if backend != sessionpermission.BackendManaged {
		if profile.AppMonitorMetrics || profile.AppMonitorTrace {
			unsupported = append(unsupported, "App Monitor capture")
		}
		if profile.AudioMode != sessionpermission.AudioModeNone {
			unsupported = append(unsupported, "audio capture")
		}
	}
	if backend == sessionpermission.BackendInstalled && (profile.ExtensionLoad || profile.ExtensionInstall) {
		unsupported = append(unsupported, "extension install and load")
	}
	return unsupported
}

func profileAllowsCapability(profile sessionpermission.Profile, backend sessionpermission.Backend, capability sessionpermission.Capability) bool {
	switch capability {
	case sessionpermission.CapabilityObserve, sessionpermission.CapabilityAccessibilityTree:
		return profile.View
	case sessionpermission.CapabilityControl:
		return profile.Control && profile.Effect != sessionpermission.EffectObserve
	case sessionpermission.CapabilityApprovalReceipt:
		return profile.Approval != sessionpermission.ApprovalNone
	case sessionpermission.CapabilityCredentialUse:
		return len(profile.CredentialIDs) > 0
	case sessionpermission.CapabilityCreditBudget:
		return profile.MaxCredits > 0
	case sessionpermission.CapabilityByteBudget:
		return profile.MaxBytes > 0
	case sessionpermission.CapabilityHistoryRead:
		return profile.HistoryRead
	case sessionpermission.CapabilityHistoryExport:
		return profile.HistoryExport
	case sessionpermission.CapabilityHistoryRetention:
		return profile.HistoryRetention > 0
	case sessionpermission.CapabilityExtensionInventory, sessionpermission.CapabilityExtensionHostPermissions:
		return profile.ExtensionAccess
	case sessionpermission.CapabilityExtensionInstall:
		return profile.ExtensionInstall && backend == sessionpermission.BackendManaged
	case sessionpermission.CapabilityExtensionLoad:
		return profile.ExtensionLoad && backend == sessionpermission.BackendManaged
	case sessionpermission.CapabilityIdentityCustomization:
		return profile.IdentityProfile != nil && (backend == sessionpermission.BackendManaged || backend == sessionpermission.BackendInstalled)
	case sessionpermission.CapabilityHandoff:
		return profile.Handoff
	case sessionpermission.CapabilityPreserveUserState, sessionpermission.CapabilityReconnect:
		return backend == sessionpermission.BackendUserAttached && profile.ProfileMode == "user_attached"
	case sessionpermission.CapabilityAudioCapture:
		return profile.AudioMode != sessionpermission.AudioModeNone && backend == sessionpermission.BackendManaged
	case sessionpermission.CapabilityAudioDeviceEvidence:
		return profile.AudioMode == sessionpermission.AudioModePipeWire && backend == sessionpermission.BackendManaged
	case sessionpermission.CapabilityAppMonitorMetrics:
		return profile.AppMonitorMetrics && backend == sessionpermission.BackendManaged
	case sessionpermission.CapabilityAppMonitorTrace:
		return profile.AppMonitorTrace && backend == sessionpermission.BackendManaged
	default:
		return false
	}
}

func backendForProfileMode(mode string) (sessionpermission.Backend, bool) {
	switch mode {
	case "fresh":
		return sessionpermission.BackendManaged, true
	case "isolated":
		return sessionpermission.BackendInstalled, true
	case "user_attached":
		return sessionpermission.BackendUserAttached, true
	case "remote":
		return sessionpermission.BackendRemote, true
	default:
		return "", false
	}
}

func uniqueCapabilities(values []sessionpermission.Capability) []sessionpermission.Capability {
	seen := make(map[sessionpermission.Capability]bool, len(values))
	result := make([]sessionpermission.Capability, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func summarizePermissionProfile(profile sessionpermission.Profile) CapabilityProfileSummary {
	identity := profile.IdentityProfile
	if identity != nil {
		copy := *identity
		identity = &copy
	}
	extensions := make(map[string][]string, len(profile.ExtensionPermissions))
	for id, permissions := range profile.ExtensionPermissions {
		extensions[id] = append([]string(nil), permissions...)
	}
	return CapabilityProfileSummary{
		ProfileID: profile.ProfileID, ProfileMode: profile.ProfileMode, Effect: profile.Effect, Approval: profile.Approval,
		URLAllow: append([]string(nil), profile.URLAllow...), URLDeny: append([]string(nil), profile.URLDeny...),
		View: profile.View, Control: profile.Control, Handoff: profile.Handoff,
		CredentialRefs:  append([]string(nil), profile.CredentialIDs...),
		ExtensionAccess: profile.ExtensionAccess, ExtensionInstall: profile.ExtensionInstall, ExtensionLoad: profile.ExtensionLoad,
		ExtensionPermissions: extensions,
		HistoryRead:          profile.HistoryRead, HistoryExport: profile.HistoryExport,
		HistoryRetentionMS: profile.HistoryRetention.Milliseconds(), MaxActions: profile.MaxActions,
		MaxDurationMS: profile.MaxDuration.Milliseconds(), MaxCredits: profile.MaxCredits, MaxBytes: profile.MaxBytes,
		AudioMode: profile.AudioMode, IdentityProfile: identity,
	}
}

func identityBehavior(backend sessionpermission.Backend) string {
	switch backend {
	case sessionpermission.BackendManaged:
		return "managed fresh-context identity values are taken from the active Agent Manager profile"
	case sessionpermission.BackendInstalled:
		return "installed binary keeps native defaults; only a persisted viewport profile is configurable"
	case sessionpermission.BackendUserAttached:
		return "preserve the selected real tab identity and state; fresh-session customization is not applied"
	default:
		return "remote identity behavior is unsupported until an existing remote session owner is configured"
	}
}
