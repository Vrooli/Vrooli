// Package sessionpermission defines the permission contract shared by browser
// session adapters. It contains no consent or grant store: approval receipts
// must be verified by the existing authorization owner through ReceiptVerifier.
package sessionpermission

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/language"
)

type Effect string
type Approval string

const (
	EffectObserve      Effect = "observe"
	EffectAct          Effect = "act"
	EffectUnrestricted Effect = "unrestricted"

	ApprovalNone        Approval = "none"
	ApprovalSafetyPass  Approval = "safety_pass"
	ApprovalEveryAction Approval = "every_action"
)

type Backend string

// AudioMode is an Agent Manager-issued, session-scoped media capability.
// Empty denies microphone access. Host-device mode may alter PipeWire source
// routing and is kept distinct from Chromium fake-media capture.
type AudioMode string

const (
	BackendManaged      Backend = "managed"
	BackendInstalled    Backend = "installed"
	BackendUserAttached Backend = "user_attached"
	BackendRemote       Backend = "remote"

	AudioModeNone           AudioMode = ""
	AudioModeFakeMicrophone AudioMode = "fake_microphone"
	AudioModePipeWire       AudioMode = "pipewire_qualification"
)

var (
	ErrInvalidProfile = errors.New("invalid browser permission profile")
	ErrDenied         = errors.New("browser action denied by permission profile")
)

// Profile is the session-scoped, caller-visible browser permission contract.
// URL rules use web-extension match patterns (for example
// https://*.example.test/*). Empty allow lists deny all web origins.
type Profile struct {
	ID                   string              `json:"id,omitempty"`
	Effect               Effect              `json:"effect"`
	Approval             Approval            `json:"approval"`
	URLAllow             []string            `json:"url_allow,omitempty"`
	URLDeny              []string            `json:"url_deny,omitempty"`
	ProfileMode          string              `json:"profile_mode,omitempty"` // fresh, isolated, user_attached, remote
	ProfileID            string              `json:"profile_id,omitempty"`
	CredentialIDs        []string            `json:"credential_ids,omitempty"`
	ExtensionLoad        bool                `json:"extension_load,omitempty"`
	ExtensionAccess      bool                `json:"extension_access,omitempty"`
	ExtensionInstall     bool                `json:"extension_install,omitempty"`
	HistoryRead          bool                `json:"history_read,omitempty"`
	HistoryExport        bool                `json:"history_export,omitempty"`
	UserStateExport      bool                `json:"user_state_export,omitempty"`
	HistoryRetention     time.Duration       `json:"history_retention,omitempty"`
	View                 bool                `json:"view,omitempty"`
	Control              bool                `json:"control,omitempty"`
	Handoff              bool                `json:"handoff,omitempty"`
	MaxActions           int64               `json:"max_actions,omitempty"`
	MaxDuration          time.Duration       `json:"max_duration,omitempty"`
	MaxCredits           int64               `json:"max_credits,omitempty"`
	MaxBytes             int64               `json:"max_bytes,omitempty"`
	ExtensionPermissions map[string][]string `json:"extension_permissions,omitempty"`
	AudioMode            AudioMode           `json:"audio_mode,omitempty"`
	// PerformanceTrace remains the broader BAS artifact capability. AppMonitorMetrics
	// authorizes only a fixed scalar read. Both default false for legacy profiles.
	PerformanceTrace             bool                    `json:"performance_trace,omitempty"`
	AppMonitorMetrics            bool                    `json:"app_monitor_metrics,omitempty"`
	AppMonitorTrace              bool                    `json:"app_monitor_trace,omitempty"`
	AppMonitorTraceMaxDurationMS int64                   `json:"app_monitor_trace_max_duration_ms,omitempty"`
	AppMonitorTraceMaxBytes      int64                   `json:"app_monitor_trace_max_bytes,omitempty"`
	IdentityProfile              *BrowserIdentityProfile `json:"identity_profile,omitempty"`
}

// BrowserIdentityProfile is a bounded subset of BAS's existing fresh-context
// controls. Managed contexts support the full set; installed contexts support
// viewport only. It excludes user-agent overrides, storage,
// proxy, geolocation, headers and browser-attach identity changes.
type BrowserIdentityProfile struct {
	Preset            string  `json:"preset,omitempty"`
	ViewportWidth     int     `json:"viewport_width,omitempty"`
	ViewportHeight    int     `json:"viewport_height,omitempty"`
	DeviceScaleFactor float64 `json:"device_scale_factor,omitempty"`
	Locale            string  `json:"locale,omitempty"`
	TimezoneID        string  `json:"timezone_id,omitempty"`
	ColorScheme       string  `json:"color_scheme,omitempty"`
}

// ExtensionPermissionEvidence describes where the permission lists came from.
// Effective permissions are runtime grants, while declared permissions come
// from an artifact manifest. These sources must never be conflated.
type ExtensionPermissionEvidence struct {
	Status           string    `json:"status"`              // "effective" or "unavailable"
	Source           string    `json:"source"`              // e.g. chrome.management.getAll
	Scope            string    `json:"scope"`               // "grant_set_only" or "unavailable"
	PerTabSiteAccess string    `json:"per_tab_site_access"` // always "not_measured"
	ObservedAt       time.Time `json:"observed_at,omitempty"`
}

const (
	ExtensionPermissionEvidenceEffective      = "effective"
	ExtensionPermissionEvidenceUnavailable    = "unavailable"
	ExtensionPermissionScopeGrantSetOnly      = "grant_set_only"
	ExtensionPermissionScopeUnavailable       = "unavailable"
	ExtensionPerTabSiteAccessNotMeasured      = "not_measured"
	ExtensionPermissionSourceChromeManagement = "chrome.management.getAll"
	ExtensionPermissionSourceChromeGetAll     = "chrome.permissions.getAll.v1"
	ExtensionPermissionSourceChromiumCDP      = "chromium.Extensions.getExtensions"
)

// ExtensionInventoryEntry is safe-to-display metadata reported by the browser
// backend; effective and declared permission sets remain separate.
type ExtensionInventoryEntry struct {
	ID                      string                      `json:"id"`
	Name                    string                      `json:"name"`
	Version                 string                      `json:"version,omitempty"`
	Enabled                 bool                        `json:"enabled"`
	Permissions             []string                    `json:"effective_permissions,omitempty"`
	HostPermissions         []string                    `json:"effective_host_permissions,omitempty"`
	PermissionEvidence      ExtensionPermissionEvidence `json:"permission_evidence"`
	DeclaredPermissions     []string                    `json:"declared_permissions,omitempty"`
	DeclaredHostPermissions []string                    `json:"declared_host_permissions,omitempty"`
	ArtifactSHA256          string                      `json:"artifact_sha256,omitempty"`
	ManifestSHA256          string                      `json:"manifest_sha256,omitempty"`
	ArtifactVerified        bool                        `json:"artifact_verified,omitempty"`
}

// ActionRequest carries the immutable broker identity and one intended action.
// TargetURL is required for navigation; CurrentURL must be the selected tab's
// current URL for all other operations.
type ActionRequest struct {
	SessionID string
	LeaseID   string
	OwnerID   string
	// CallerID is injected from authenticated request/run context, never from
	// agent-supplied JSON. OwnerID is the immutable broker session owner.
	CallerID            string
	Kind                string
	TargetURL           string
	CurrentURL          string
	ApprovalReceipt     string
	CredentialID        string
	ExtensionID         string
	ExtensionPermission string
	TabID               string
	ConsentReceipt      string
}

// OpenRequest is evaluated before the browser creates a session or lease.
// An admission decision must be bound to the caller grant, requested backend,
// profile and initial URL; it cannot be represented as an ActionRequest.
type OpenRequest struct {
	OwnerID        string
	CallerID       string
	Backend        Backend
	ProfileID      string
	InitialURL     string
	GrantReceipt   string
	TabID          string
	ConsentReceipt string
}

func (r OpenRequest) Validate(profile Profile) error {
	if strings.TrimSpace(r.OwnerID) == "" || strings.TrimSpace(r.CallerID) == "" || strings.TrimSpace(r.ProfileID) == "" {
		return errors.New("browser open requires trusted owner, caller, and profile ids")
	}
	if r.Backend != BackendManaged && r.Backend != BackendInstalled && r.Backend != BackendUserAttached && r.Backend != BackendRemote {
		return fmt.Errorf("unsupported browser backend %q", r.Backend)
	}
	if r.InitialURL != "" && !profile.AllowsURL(r.InitialURL) {
		return errors.New("browser open target is outside the allowed URL scope")
	}
	if profile.ProfileMode == "user_attached" || r.Backend == BackendUserAttached {
		if profile.ProfileMode != "user_attached" || r.Backend != BackendUserAttached || strings.TrimSpace(r.TabID) == "" || strings.TrimSpace(r.ConsentReceipt) == "" {
			return errors.New("user-attached open requires an explicitly consented selected tab")
		}
	}
	return nil
}

// OpenAuthorizer delegates admission to Agent Manager / the existing consent
// authority before a browser process or attached-tab lease is created.
type OpenAuthorizer interface {
	AuthorizeOpen(context.Context, OpenRequest) error
}

// CallerResolver derives the active principal from trusted request/run context
// on every operation. This permits explicit handoff without accepting caller
// identities from agent JSON.
type CallerResolver interface {
	BrowserCallerID(context.Context) (string, error)
}

// ReceiptVerifier delegates approval validity and scope to the existing
// authorization authority. Implementations must bind receipt to session,
// lease, owner, operation and URL; this package never stores approvals.
type ReceiptVerifier interface {
	VerifyBrowserApproval(context.Context, string, ActionRequest) error
}

// ConsentVerifier delegates one-tab attachment consent to its existing owner.
type ConsentVerifier interface {
	VerifyBrowserTabConsent(context.Context, string, ActionRequest) error
}

// SelectedTabResultVerifier rechecks consent for data returned by an attached
// tab after the browser operation completes. It does not admit another action
// or charge the profile's action budget.
type SelectedTabResultVerifier interface {
	VerifySelectedTabResult(context.Context, ActionRequest) error
}

// CreditBudgetVerifier keeps browser metering with LPBS/Agent Manager. If a
// profile reserves a positive credit budget, the authority must re-check it
// against session usage before every operation.
type CreditBudgetVerifier interface {
	AuthorizeBrowserCredits(context.Context, ActionRequest, int64) error
}

// CredentialUseVerifier delegates exact-origin secret handling to credential
// authority. It must never return secret bytes through this browser policy.
type CredentialUseVerifier interface {
	AuthorizeBrowserCredential(context.Context, ActionRequest, string) error
}

// ByteBudgetVerifier rechecks owner-measured transfer usage before each operation.
// Transcript bytes supplied by a caller are not an enforcement source.
type ByteBudgetVerifier interface {
	AuthorizeBrowserBytes(context.Context, ActionRequest, int64) error
}

// Authorizer is the injection point used by broker-backed session facades.
type Authorizer interface {
	Authorize(context.Context, ActionRequest) error
}

// ProfileBoundAuthorizer prevents the driver from receiving a more permissive
// static profile than the broker enforces.
type ProfileBoundAuthorizer interface {
	Authorizer
	ProfileSnapshot() Profile
}

type Policy struct {
	profile            Profile
	verifier           ReceiptVerifier
	consentVerifier    ConsentVerifier
	creditVerifier     CreditBudgetVerifier
	byteBudgetVerifier ByteBudgetVerifier
	credentialVerifier CredentialUseVerifier
	mu                 sync.Mutex
	started            time.Time
	actions            int64
}

func New(profile Profile, verifier ReceiptVerifier) (*Policy, error) {
	return NewWithVerifiers(profile, verifier, nil)
}

func NewWithVerifiers(profile Profile, verifier ReceiptVerifier, consentVerifier ConsentVerifier) (*Policy, error) {
	if err := profile.Validate(); err != nil {
		return nil, err
	}
	return &Policy{profile: cloneProfile(profile), verifier: verifier, consentVerifier: consentVerifier, started: time.Now()}, nil
}

func (p *Policy) SetCreditBudgetVerifier(verifier CreditBudgetVerifier) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.creditVerifier = verifier
	p.mu.Unlock()
}

func (p *Policy) SetByteBudgetVerifier(verifier ByteBudgetVerifier) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.byteBudgetVerifier = verifier
	p.mu.Unlock()
}

func (p *Policy) SetCredentialUseVerifier(verifier CredentialUseVerifier) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.credentialVerifier = verifier
	p.mu.Unlock()
}

// ProfileSnapshot returns an independent profile value for session metadata;
// callers cannot mutate the active policy through a shared map or slice.
func (p *Policy) ProfileSnapshot() Profile {
	if p == nil {
		return Profile{}
	}
	return cloneProfile(p.profile)
}

// VerifySelectedTabResult binds returned observation data to the same selected
// tab consent and URL scope used for the pre-read admission. A page may navigate
// while a read is in flight; this check prevents returning data from a newly
// observed origin that the existing consent owner did not authorize.
func (p *Policy) VerifySelectedTabResult(ctx context.Context, req ActionRequest) error {
	if p == nil || p.profile.ProfileMode != "user_attached" || req.Kind != "observe" ||
		req.SessionID == "" || req.LeaseID == "" || req.OwnerID == "" || req.CallerID == "" ||
		req.TabID == "" || req.ConsentReceipt == "" || req.CurrentURL == "" ||
		!p.profile.AllowsURL(req.CurrentURL) {
		return ErrDenied
	}
	p.mu.Lock()
	verifier := p.consentVerifier
	p.mu.Unlock()
	if verifier == nil {
		return fmt.Errorf("%w: selected-tab consent verifier unavailable", ErrDenied)
	}
	if err := verifier.VerifyBrowserTabConsent(ctx, req.ConsentReceipt, req); err != nil {
		return fmt.Errorf("%w: selected-tab result consent rejected: %v", ErrDenied, err)
	}
	return nil
}

func cloneProfile(profile Profile) Profile {
	profile.URLAllow = append([]string(nil), profile.URLAllow...)
	profile.URLDeny = append([]string(nil), profile.URLDeny...)
	profile.CredentialIDs = append([]string(nil), profile.CredentialIDs...)
	if profile.IdentityProfile != nil {
		identity := *profile.IdentityProfile
		profile.IdentityProfile = &identity
	}
	if profile.ExtensionPermissions != nil {
		copied := make(map[string][]string, len(profile.ExtensionPermissions))
		for id, perms := range profile.ExtensionPermissions {
			copied[id] = append([]string(nil), perms...)
		}
		profile.ExtensionPermissions = copied
	}
	return profile
}

func (p Profile) Validate() error {
	if p.Effect != EffectObserve && p.Effect != EffectAct && p.Effect != EffectUnrestricted {
		return fmt.Errorf("%w: effect must be observe, act, or unrestricted", ErrInvalidProfile)
	}
	if p.Approval != ApprovalNone && p.Approval != ApprovalSafetyPass && p.Approval != ApprovalEveryAction {
		return fmt.Errorf("%w: approval must be none, safety_pass, or every_action", ErrInvalidProfile)
	}
	if p.ProfileMode != "" && p.ProfileMode != "fresh" && p.ProfileMode != "isolated" && p.ProfileMode != "user_attached" && p.ProfileMode != "remote" {
		return fmt.Errorf("%w: unsupported profile mode %q", ErrInvalidProfile, p.ProfileMode)
	}
	if p.PerformanceTrace && p.ProfileMode != "fresh" {
		return fmt.Errorf("%w: performance trace requires a fresh managed browser profile", ErrInvalidProfile)
	}
	if p.AppMonitorMetrics && p.ProfileMode != "fresh" {
		return fmt.Errorf("%w: App Monitor metrics require a fresh managed browser profile", ErrInvalidProfile)
	}
	if p.AppMonitorTrace {
		if p.ProfileMode != "fresh" || p.AppMonitorTraceMaxDurationMS <= 0 || p.AppMonitorTraceMaxDurationMS > 60_000 || p.AppMonitorTraceMaxBytes <= 0 || p.AppMonitorTraceMaxBytes > 16<<20 {
			return fmt.Errorf("%w: App Monitor trace requires a fresh managed profile and bounded positive duration/byte ceilings", ErrInvalidProfile)
		}
	} else if p.AppMonitorTraceMaxDurationMS != 0 || p.AppMonitorTraceMaxBytes != 0 {
		return fmt.Errorf("%w: App Monitor trace budgets require the explicit trace permission", ErrInvalidProfile)
	}
	if err := validateBrowserIdentityProfile(p.ProfileMode, p.IdentityProfile); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidProfile, err)
	}
	switch p.AudioMode {
	case AudioModeNone:
	case AudioModeFakeMicrophone, AudioModePipeWire:
		if p.ProfileMode != "fresh" || p.Effect == EffectObserve || !p.View || !p.Control || len(p.URLAllow) == 0 {
			return fmt.Errorf("%w: audio capture requires a managed view+act profile with an explicit URL scope", ErrInvalidProfile)
		}
	default:
		return fmt.Errorf("%w: unsupported audio mode %q", ErrInvalidProfile, p.AudioMode)
	}
	if p.MaxActions < 0 || p.MaxDuration < 0 || p.MaxCredits < 0 || p.MaxBytes < 0 || p.HistoryRetention < 0 {
		return fmt.Errorf("%w: budgets and retention cannot be negative", ErrInvalidProfile)
	}
	for _, pattern := range append(append([]string{}, p.URLAllow...), p.URLDeny...) {
		if err := validatePattern(pattern); err != nil {
			return fmt.Errorf("%w: %s", ErrInvalidProfile, err)
		}
	}
	for extension, perms := range p.ExtensionPermissions {
		if strings.TrimSpace(extension) == "" {
			return fmt.Errorf("%w: extension permission entry has empty extension id", ErrInvalidProfile)
		}
		for _, perm := range perms {
			if strings.TrimSpace(perm) == "" {
				return fmt.Errorf("%w: extension %s has empty permission", ErrInvalidProfile, extension)
			}
		}
	}
	return nil
}

func validateBrowserIdentityProfile(mode string, profile *BrowserIdentityProfile) error {
	if profile == nil {
		return nil
	}
	if mode == "isolated" {
		if profile.Preset != "" || profile.DeviceScaleFactor != 0 || profile.Locale != "" || profile.TimezoneID != "" || profile.ColorScheme != "" {
			return errors.New("installed-binary identity customization supports viewport only; preset, device scale, locale, timezone and color scheme are unsupported")
		}
		return validateIdentityViewport(profile)
	}
	if mode != "fresh" {
		return errors.New("fresh-session identity customization is unsupported for user-attached and remote browser modes")
	}
	switch profile.Preset {
	case "", "stealth", "balanced", "fast", "none":
	default:
		return fmt.Errorf("unsupported managed identity preset %q", profile.Preset)
	}
	if err := validateIdentityViewport(profile); err != nil {
		return err
	}
	if profile.DeviceScaleFactor < 0 || profile.DeviceScaleFactor > 5 {
		return errors.New("identity device scale factor must be between 0 and 5")
	}
	if profile.Locale != "" {
		if len(profile.Locale) > 64 || strings.TrimSpace(profile.Locale) != profile.Locale || !validIdentityLocale(profile.Locale) {
			return errors.New("identity locale must be a bounded language tag")
		}
	}
	if profile.TimezoneID != "" {
		if len(profile.TimezoneID) > 64 || strings.TrimSpace(profile.TimezoneID) != profile.TimezoneID {
			return errors.New("identity timezone must be a bounded IANA timezone")
		}
		if profile.TimezoneID == "Local" {
			return errors.New("identity timezone cannot use host-dependent Local; choose a supported IANA identifier or UTC alias")
		}
		if _, err := time.LoadLocation(profile.TimezoneID); err != nil {
			return errors.New("identity timezone must be a supported IANA timezone")
		}
	}
	switch profile.ColorScheme {
	case "", "light", "dark", "no-preference":
	default:
		return errors.New("identity color scheme must be light, dark, or no-preference")
	}
	return nil
}

func validateIdentityViewport(profile *BrowserIdentityProfile) error {
	if profile.ViewportWidth < 0 || profile.ViewportWidth > 7680 || profile.ViewportHeight < 0 || profile.ViewportHeight > 4320 || (profile.ViewportWidth == 0) != (profile.ViewportHeight == 0) {
		return errors.New("identity viewport must set both dimensions within 7680x4320")
	}
	return nil
}

func validIdentityLocale(value string) bool {
	if value == "" || strings.TrimSpace(value) != value || strings.Contains(value, "_") {
		return false
	}
	for _, r := range value {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	_, err := language.Parse(value)
	return err == nil
}

func (p *Policy) Authorize(ctx context.Context, req ActionRequest) error {
	if p == nil || p.profile.Validate() != nil || req.SessionID == "" || req.LeaseID == "" || req.OwnerID == "" || req.CallerID == "" {
		return ErrDenied
	}
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if kind == "" {
		return ErrDenied
	}
	// Closing is always available to the already-resolved owning session so a
	// restrictive profile or revocation cannot strand a browser process.
	if kind == "close" {
		return nil
	}
	if kind == "reset" && p.profile.ProfileMode == "user_attached" {
		return fmt.Errorf("%w: resetting an attached user browser is denied", ErrDenied)
	}
	if strings.HasPrefix(kind, "audio_") && p.profile.AudioMode == AudioModeNone {
		return fmt.Errorf("%w: browser audio operation requires an Agent Manager audio capability", ErrDenied)
	}
	if strings.HasPrefix(kind, "audio_playback_") && p.profile.AudioMode != AudioModePipeWire {
		return fmt.Errorf("%w: host audio playback controls require the PipeWire qualification capability", ErrDenied)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	if p.profile.MaxActions > 0 && p.actions >= p.profile.MaxActions {
		return fmt.Errorf("%w: action budget exhausted", ErrDenied)
	}
	if p.profile.MaxDuration > 0 && now.Sub(p.started) >= p.profile.MaxDuration {
		return fmt.Errorf("%w: session time budget exhausted", ErrDenied)
	}
	if p.profile.Effect == EffectObserve && isMutation(kind) {
		return fmt.Errorf("%w: observe profile cannot %s", ErrDenied, kind)
	}
	if isMutation(kind) && !p.profile.Control {
		return fmt.Errorf("%w: browser control is disabled", ErrDenied)
	}
	if strings.Contains(kind, "cdp") || kind == "raw_protocol" {
		if p.profile.Effect != EffectUnrestricted {
			return fmt.Errorf("%w: raw CDP requires unrestricted effect", ErrDenied)
		}
	}
	switch kind {
	case "app_monitor_metrics":
		if !p.profile.AppMonitorMetrics || p.profile.ProfileMode != "fresh" {
			return fmt.Errorf("%w: bounded App Monitor metrics are disabled", ErrDenied)
		}
	case "app_monitor_trace":
		if !p.profile.AppMonitorTrace || p.profile.ProfileMode != "fresh" {
			return fmt.Errorf("%w: bounded App Monitor trace is disabled", ErrDenied)
		}
	case "view", "observe", "screenshot", "assert", "extract", "audio_observe":
		if !p.profile.View {
			return fmt.Errorf("%w: viewing is disabled", ErrDenied)
		}
	case "control", "input":
		if !p.profile.Control {
			return fmt.Errorf("%w: browser control is disabled", ErrDenied)
		}
	case "handoff":
		if !p.profile.Handoff {
			return fmt.Errorf("%w: handoff is disabled", ErrDenied)
		}
	case "history_read":
		if !p.profile.HistoryRead {
			return fmt.Errorf("%w: history read is disabled", ErrDenied)
		}
	case "recording_stop":
		if !p.profile.HistoryRead {
			return fmt.Errorf("%w: recording stop receipt requires history read", ErrDenied)
		}
	case "history_export":
		if !p.profile.HistoryExport {
			return fmt.Errorf("%w: history export is disabled", ErrDenied)
		}
	case "user_state_export":
		if !p.profile.UserStateExport {
			return fmt.Errorf("%w: browser user-state export is disabled", ErrDenied)
		}
	case "extension_install":
		if !p.profile.ExtensionInstall {
			return fmt.Errorf("%w: extension installation is disabled", ErrDenied)
		}
	case "extension_artifact_read":
		if !p.profile.ExtensionInstall {
			return fmt.Errorf("%w: extension installation is disabled", ErrDenied)
		}
	case "extension_inventory":
		if !p.profile.ExtensionAccess {
			return fmt.Errorf("%w: extension inventory is disabled", ErrDenied)
		}
	case "extension_load":
		if !p.profile.ExtensionLoad {
			return fmt.Errorf("%w: extension loading is disabled", ErrDenied)
		}
	case "extension_unload":
		if !p.profile.ExtensionLoad {
			return fmt.Errorf("%w: extension loading is disabled", ErrDenied)
		}
	}
	urlToCheck := req.CurrentURL
	if kind == "navigate" {
		urlToCheck = req.TargetURL
	}
	if urlToCheck == "" || !p.profile.AllowsURL(urlToCheck) {
		return fmt.Errorf("%w: URL outside allowed scope", ErrDenied)
	}
	if kind == "service_worker_unregister" && (req.TargetURL == "" || !p.profile.AllowsURL(req.TargetURL)) {
		return fmt.Errorf("%w: service worker scope URL outside allowed scope", ErrDenied)
	}
	if p.profile.ProfileMode == "user_attached" {
		if req.TabID == "" || req.ConsentReceipt == "" || p.consentVerifier == nil {
			return fmt.Errorf("%w: selected-tab consent is required", ErrDenied)
		}
		// Verify on every operation so the existing consent owner can enforce
		// expiry, revocation and the exact tab/URL/caller scope after pairing.
		if err := p.consentVerifier.VerifyBrowserTabConsent(ctx, req.ConsentReceipt, req); err != nil {
			return fmt.Errorf("%w: selected-tab consent rejected: %v", ErrDenied, err)
		}
	}
	if kind == "credential_use" && (req.CredentialID == "" || !contains(p.profile.CredentialIDs, req.CredentialID)) {
		return fmt.Errorf("%w: credential is not granted by this profile", ErrDenied)
	}
	if kind == "input" && len(p.profile.CredentialIDs) > 0 {
		return fmt.Errorf("%w: typed input is disabled for credential-scoped profiles; use the credential authority reference flow", ErrDenied)
	}
	if kind == "credential_use" {
		if p.credentialVerifier == nil {
			return fmt.Errorf("%w: credential authority unavailable", ErrDenied)
		}
		if err := p.credentialVerifier.AuthorizeBrowserCredential(ctx, req, req.CredentialID); err != nil {
			return fmt.Errorf("%w: exact-origin credential use denied: %v", ErrDenied, err)
		}
	}
	if kind != "extension_inventory" && kind != "extension_artifact_read" && strings.HasPrefix(kind, "extension_") && strings.TrimSpace(req.ExtensionID) == "" {
		return fmt.Errorf("%w: extension operations require an explicitly granted extension id", ErrDenied)
	}
	if req.ExtensionID != "" {
		if !p.profile.ExtensionAccess {
			return fmt.Errorf("%w: extension access is disabled", ErrDenied)
		}
		if _, allowed := p.profile.ExtensionPermissions[req.ExtensionID]; !allowed {
			return fmt.Errorf("%w: extension id is not granted", ErrDenied)
		}
		if req.ExtensionPermission != "" && !contains(p.profile.ExtensionPermissions[req.ExtensionID], req.ExtensionPermission) {
			return fmt.Errorf("%w: extension permission is not granted", ErrDenied)
		}
	}
	if p.profile.MaxCredits > 0 {
		if p.creditVerifier == nil {
			return fmt.Errorf("%w: credit budget authority unavailable", ErrDenied)
		}
		if err := p.creditVerifier.AuthorizeBrowserCredits(ctx, req, p.profile.MaxCredits); err != nil {
			return fmt.Errorf("%w: credit budget denied: %v", ErrDenied, err)
		}
	}
	if p.profile.MaxBytes > 0 {
		if p.byteBudgetVerifier == nil {
			return fmt.Errorf("%w: byte budget authority unavailable", ErrDenied)
		}
		if err := p.byteBudgetVerifier.AuthorizeBrowserBytes(ctx, req, p.profile.MaxBytes); err != nil {
			return fmt.Errorf("%w: byte budget denied: %v", ErrDenied, err)
		}
	}
	needsApproval := p.profile.Approval == ApprovalEveryAction || (p.profile.Approval == ApprovalSafetyPass && isHighImpact(kind))
	if needsApproval {
		if p.verifier == nil || strings.TrimSpace(req.ApprovalReceipt) == "" {
			return fmt.Errorf("%w: approval receipt required", ErrDenied)
		}
		if err := p.verifier.VerifyBrowserApproval(ctx, req.ApprovalReceipt, req); err != nil {
			return fmt.Errorf("%w: approval receipt rejected: %v", ErrDenied, err)
		}
	}
	p.actions++
	return nil
}

func contains(values []string, candidate string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == candidate {
			return true
		}
	}
	return false
}

func (p Profile) AllowsURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return false
	}
	for _, pattern := range p.URLDeny {
		if matches(pattern, u) {
			return false
		}
	}
	for _, pattern := range p.URLAllow {
		if matches(pattern, u) {
			return true
		}
	}
	return false
}

func isMutation(kind string) bool {
	switch kind {
	case "observe", "read", "view", "screenshot", "assert", "extract", "audio_observe", "app_monitor_metrics", "app_monitor_trace", "history_read", "user_state_export", "extension_inventory", "extension_artifact_read", "attach", "close":
		return false
	default:
		return true
	}
}

func isHighImpact(kind string) bool {
	switch kind {
	case "purchase", "submit", "delete", "credential_use", "extension_install", "download", "input", "service_worker_unregister", "recording_start", "recording_stop", "replay_preview", "close_page", "reset", "audio_playback_restart", "audio_playback_stop":
		return true
	default:
		return false
	}
}

func validatePattern(pattern string) error {
	pattern = strings.TrimSpace(pattern)
	parts := strings.SplitN(pattern, "://", 2)
	if len(parts) != 2 || (parts[0] != "http" && parts[0] != "https") {
		return fmt.Errorf("URL pattern %q must use http(s)", pattern)
	}
	host := strings.SplitN(parts[1], "/", 2)[0]
	if host == "" || strings.Contains(host, "@") || strings.Contains(host, "?") || strings.Contains(host, "#") {
		return fmt.Errorf("URL pattern %q has invalid host", pattern)
	}
	if strings.Contains(host, "*") && !strings.HasPrefix(host, "*.") {
		return fmt.Errorf("URL pattern %q only supports a leading *. host wildcard", pattern)
	}
	return nil
}

func matches(pattern string, candidate *url.URL) bool {
	parts := strings.SplitN(strings.TrimSpace(pattern), "://", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], candidate.Scheme) {
		return false
	}
	patternHost, pathPattern := parts[1], "/*"
	if slash := strings.IndexByte(patternHost, '/'); slash >= 0 {
		pathPattern = "/" + strings.TrimLeft(patternHost[slash+1:], "/")
		patternHost = patternHost[:slash]
	}
	host := strings.ToLower(candidate.Hostname())
	patternHost = strings.ToLower(patternHost)
	if strings.HasPrefix(patternHost, "*.") {
		suffix := strings.TrimPrefix(patternHost, "*")
		if !strings.HasSuffix(host, suffix) || host == strings.TrimPrefix(suffix, ".") {
			return false
		}
	} else if host != patternHost {
		return false
	}
	return pathPattern == "/*" || strings.HasPrefix(candidate.EscapedPath()+"/", strings.TrimSuffix(pathPattern, "*"))
}
