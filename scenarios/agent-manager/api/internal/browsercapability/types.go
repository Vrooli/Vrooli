// Package browsercapability owns Agent Manager's browser authorization contract.
// It authorizes run-scoped access; BAS owns opening and operating sessions.
package browsercapability

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/language"
)

var (
	ErrNoEligibleCandidate = errors.New("no eligible browser capability candidate")
	ErrRunNotAuthorized    = errors.New("browser capability is not authorized for this run")
)

type Mode string

const (
	ModeManaged   Mode = "managed"
	ModeInstalled Mode = "installed-binary"
	ModeAttached  Mode = "user-attached"
	ModeRemote    Mode = "remote"
)

// SessionProfileMode is the persisted BAS permission-profile mode resolved
// from the selected browser backend. It is stored explicitly in each run
// capability so BAS does not infer session isolation from a request at open.
type SessionProfileMode string

const (
	SessionProfileFresh        SessionProfileMode = "fresh"
	SessionProfileIsolated     SessionProfileMode = "isolated"
	SessionProfileUserAttached SessionProfileMode = "user_attached"
	SessionProfileRemote       SessionProfileMode = "remote"
)

// ProfileModeForMode is the canonical owner mapping between an AM browser
// backend choice and BAS's persisted session permission-profile mode.
func ProfileModeForMode(mode Mode) (SessionProfileMode, bool) {
	switch mode {
	case ModeManaged:
		return SessionProfileFresh, true
	case ModeInstalled:
		return SessionProfileIsolated, true
	case ModeAttached:
		return SessionProfileUserAttached, true
	case ModeRemote:
		return SessionProfileRemote, true
	default:
		return "", false
	}
}

const (
	MaxCandidates                      = 16
	MaxSessionMinutes            int64 = 24 * 60
	MaxActions                   int64 = 100_000
	MaxBytes                     int64 = 1 << 30
	MaxAppMonitorTraceBytes      int64 = 16 << 20
	MaxAppMonitorTraceDurationMS int64 = 60_000
	MaxInferenceTokens           int64 = 10_000_000
	MaxPermissionEntries               = 256
)

// ResourceRoleSession is the only browser resource role currently qualified
// by the BAS session broker. New roles require an owner-declared extension to
// this contract; request JSON cannot create one.
const ResourceRoleSession = "browser.session"

type PermissionProfile string

// AudioProfileMode is an explicit run-scoped authorization for browser media
// capture. Empty is deny. The host-device mode is reserved for a qualified
// deterministic audio test task and must never be inferred from an audio URL.
type AudioProfileMode string

const (
	AudioProfileNone           AudioProfileMode = ""
	AudioProfileFakeMicrophone AudioProfileMode = "fake_microphone"
	AudioProfilePipeWire       AudioProfileMode = "pipewire_qualification"
)

const (
	ProfileObserve      PermissionProfile = "observe"
	ProfileAct          PermissionProfile = "act"
	ProfileUnrestricted PermissionProfile = "unrestricted"
)

// Candidate ties a harness to the browser resource role it may request. The
// resource owns concrete browser acquisition and mode capability.
type Candidate struct {
	Harness      string `json:"harness"`
	ResourceRole string `json:"resourceRole"`
}

// Budget is a hard per-run ceiling. Zero means the capability is disabled for
// that dimension; it never means unlimited.
type Budget struct {
	SessionMinutes  int64 `json:"sessionMinutes"`
	Actions         int64 `json:"actions"`
	Bytes           int64 `json:"bytes"`
	InferenceTokens int64 `json:"inferenceTokens"`
}

// PermissionProfileSpec is persisted with the run's browser capability. URL
// scopes are origins, credential references are opaque owner IDs (never key
// material), and extension IDs are the exact approved inventory.
type PermissionProfileSpec struct {
	ProfileID      string             `json:"profileId,omitempty"`
	ProfileMode    SessionProfileMode `json:"profileMode,omitempty"`
	Level          PermissionProfile  `json:"level"`
	AllowedOrigins []string           `json:"allowedOrigins,omitempty"`
	CredentialRefs []string           `json:"credentialRefs,omitempty"`
	ExtensionIDs   []string           `json:"extensionIds,omitempty"`
	// ExtensionInstall is a separate run-scoped install grant. ExtensionIDs
	// remains the exact approved ID allowlist; inventory access never implies
	// permission to install an artifact.
	ExtensionInstall bool `json:"extensionInstall,omitempty"`
	// ExtensionLoad separately authorizes extension load/reload operations. It
	// is not implied by ExtensionInstall or ExtensionIDs.
	ExtensionLoad      bool     `json:"extensionLoad,omitempty"`
	AllowHistory       bool     `json:"allowHistory,omitempty"`
	RequireApprovalFor []string `json:"requireApprovalFor,omitempty"`
	// View and Control are explicit Web Console stream grants. False is the
	// default; BAS must not infer either from Level.
	View    bool `json:"view,omitempty"`
	Control bool `json:"control,omitempty"`
	// AudioMode is a run-scoped opt-in; unset denies microphone capture. BAS
	// maps only these named modes to its existing session-owned audio controls.
	AudioMode AudioProfileMode `json:"audioMode,omitempty"`
	// PerformanceTrace grants the managed session's bounded performance trace
	// artifact. It is default-deny and is never implied by browser visibility.
	PerformanceTrace bool `json:"performanceTrace,omitempty"`
	// AppMonitorMetrics grants only the fixed bounded workspace metric set used
	// by App Monitor's capture adapter. It does not authorize a general trace.
	AppMonitorMetrics bool `json:"appMonitorMetrics,omitempty"`
	// AppMonitorTrace separately grants a fixed-category sanitized trace for
	// one capture. The ceilings are mandatory when enabled and zero when off.
	AppMonitorTrace              bool  `json:"appMonitorTrace,omitempty"`
	AppMonitorTraceMaxDurationMS int64 `json:"appMonitorTraceMaxDurationMs,omitempty"`
	AppMonitorTraceMaxBytes      int64 `json:"appMonitorTraceMaxBytes,omitempty"`
	// IdentityProfile selects bounded BAS-managed context characteristics. It
	// never changes a user-attached tab and is not a bot-detection guarantee.
	IdentityProfile *IdentityProfile `json:"identity_profile,omitempty"`
}

// IdentityProfile contains only characteristics an existing isolated BAS
// context owner can apply without a user profile or credentials.
// It deliberately excludes custom/stale user-agent strings, proxy credentials,
// headers, geolocation and storage state.
type IdentityProfile struct {
	Preset            string  `json:"preset,omitempty"`
	ViewportWidth     int     `json:"viewport_width,omitempty"`
	ViewportHeight    int     `json:"viewport_height,omitempty"`
	DeviceScaleFactor float64 `json:"device_scale_factor,omitempty"`
	Locale            string  `json:"locale,omitempty"`
	TimezoneID        string  `json:"timezone_id,omitempty"`
	ColorScheme       string  `json:"color_scheme,omitempty"`
}

// Capability is the immutable run request resolved against policy candidates.
type Capability struct {
	Enabled bool `json:"enabled"`
	// GrantReceipt is minted by Agent Manager for the persisted run and is
	// copied to the executor environment as an opaque run-scoped reference.
	GrantReceipt string                `json:"grantReceipt,omitempty"`
	Candidates   []Candidate           `json:"candidates,omitempty"`
	Mode         Mode                  `json:"mode,omitempty"`
	Permissions  PermissionProfileSpec `json:"permissions,omitempty"`
	Budget       Budget                `json:"budget,omitempty"`
}

// Clone detaches the mutable slices when profiles are copied into a run.
func (c Capability) Clone() Capability {
	clone := c
	clone.Candidates = append([]Candidate(nil), c.Candidates...)
	clone.Permissions.AllowedOrigins = append([]string(nil), c.Permissions.AllowedOrigins...)
	clone.Permissions.CredentialRefs = append([]string(nil), c.Permissions.CredentialRefs...)
	clone.Permissions.ExtensionIDs = append([]string(nil), c.Permissions.ExtensionIDs...)
	clone.Permissions.RequireApprovalFor = append([]string(nil), c.Permissions.RequireApprovalFor...)
	if c.Permissions.IdentityProfile != nil {
		identity := *c.Permissions.IdentityProfile
		clone.Permissions.IdentityProfile = &identity
	}
	return clone
}

// TrustedCaller is created from Agent Manager's authenticated, persisted run
// context. Callers must not decode this from browser request JSON.
type TrustedCaller struct {
	RunID        string
	Harness      string
	ResourceRole string
	GrantID      string
}

// Grant is a run-scoped authorization record handed to BAS. It contains no
// provider key and does not itself open a session. SessionID is attached by the
// session owner after a successful open; ObservedActions are transcript facts.
type Grant struct {
	ID                string                `json:"id"`
	RunID             string                `json:"runId"`
	Harness           string                `json:"harness"`
	ResourceRole      string                `json:"resourceRole"`
	Mode              Mode                  `json:"mode"`
	PermissionProfile PermissionProfileSpec `json:"permissionProfile"`
	Budget            Budget                `json:"budget"`
	IssuedAt          time.Time             `json:"issuedAt"`
	ExpiresAt         time.Time             `json:"expiresAt"`
	RevokedAt         *time.Time            `json:"revokedAt,omitempty"`
	SessionID         string                `json:"sessionId,omitempty"`
	ObservedActions   []ObservedAction      `json:"observedActions,omitempty"`
}

type ObservedAction struct {
	At        time.Time `json:"at"`
	SessionID string    `json:"sessionId"`
	Mode      Mode      `json:"mode"`
	Action    string    `json:"action"`
	Outcome   string    `json:"outcome"`
}

// Request deliberately has no caller or capability fields. Both are resolved
// from authenticated request context and the persisted run by Agent Manager.
type Request struct{}

type CallerResolver interface {
	ResolveTrustedCaller(context.Context) (TrustedCaller, error)
}

type RunCapabilityReader interface {
	BrowserCapability(context.Context, string) (Capability, error)
}

type GrantIssuer interface {
	IssueBrowserGrant(context.Context, TrustedCaller, Capability, time.Time) (Grant, error)
}

type GrantReader interface {
	ReadBrowserGrant(context.Context, string) (Grant, error)
}

type GrantRecorder interface {
	RecordBrowserSession(context.Context, string, string, Mode, time.Time) error
	RecordObservedBrowserAction(context.Context, string, ObservedAction) error
}

// OpenRequest is a BAS admission request. GrantReceipt is opaque; caller/run
// identity is derived only from the trusted run token in context.
type OpenRequest struct {
	GrantReceipt string
	Mode         Mode
	ProfileID    string
	InitialURL   string
}

// ValidateOpenCapability verifies that the requested session is no broader
// than the run's stored capability. Agent Manager uses this before BAS opens;
// it does not perform the open.
func ValidateOpenCapability(capability Capability, req OpenRequest) error {
	if err := Validate(capability); err != nil {
		return err
	}
	if req.Mode != capability.Mode || strings.TrimSpace(req.ProfileID) != strings.TrimSpace(capability.Permissions.ProfileID) {
		return ErrRunNotAuthorized
	}
	return validateInitialURL(req.InitialURL, capability.Permissions.AllowedOrigins)
}

type Authorizer struct {
	Callers  CallerResolver
	Runs     RunCapabilityReader
	Issuer   GrantIssuer
	Grants   GrantReader
	Recorder GrantRecorder
	Now      func() time.Time
	TTL      time.Duration
}

// RecordSession binds the BAS-owned session ID to the already-authorized run
// grant after BAS successfully opens the session.
func (a *Authorizer) RecordSession(ctx context.Context, grantID, sessionID string, mode Mode) error {
	grant, err := a.trustedGrant(ctx, grantID)
	if err != nil {
		return err
	}
	if a.Recorder == nil || grant.Mode != mode || strings.TrimSpace(sessionID) == "" {
		return ErrRunNotAuthorized
	}
	return a.Recorder.RecordBrowserSession(ctx, grant.ID, strings.TrimSpace(sessionID), mode, a.now())
}

// RecordObservedAction keeps a run-linked, authority-neutral transcript of
// browser activity. BAS remains the authority for allowing or denying actions.
func (a *Authorizer) RecordObservedAction(ctx context.Context, grantID, sessionID string, action ObservedAction) error {
	grant, err := a.trustedGrant(ctx, grantID)
	if err != nil {
		return err
	}
	if a.Recorder == nil || grant.SessionID == "" || grant.SessionID != strings.TrimSpace(sessionID) || action.Mode != grant.Mode {
		return ErrRunNotAuthorized
	}
	action.SessionID = grant.SessionID
	if action.At.IsZero() {
		action.At = a.now()
	}
	return a.Recorder.RecordObservedBrowserAction(ctx, grant.ID, action)
}

// AuthorizeOpen validates the run grant before BAS creates a browser process
// or attached-tab lease. It does not open the session itself.
func (a *Authorizer) AuthorizeOpen(ctx context.Context, req OpenRequest) (Grant, error) {
	if a == nil || a.Callers == nil || a.Grants == nil {
		return Grant{}, errors.New("browser open authorization is unavailable")
	}
	caller, err := a.Callers.ResolveTrustedCaller(ctx)
	if err != nil {
		return Grant{}, fmt.Errorf("resolve trusted browser caller: %w", err)
	}
	if strings.TrimSpace(caller.RunID) == "" || strings.TrimSpace(req.GrantReceipt) == "" {
		return Grant{}, ErrRunNotAuthorized
	}
	grant, err := a.trustedGrant(ctx, req.GrantReceipt)
	if err != nil {
		return Grant{}, err
	}
	if req.Mode != grant.Mode || strings.TrimSpace(req.ProfileID) != strings.TrimSpace(grant.PermissionProfile.ProfileID) {
		return Grant{}, ErrRunNotAuthorized
	}
	if err := validateInitialURL(req.InitialURL, grant.PermissionProfile.AllowedOrigins); err != nil {
		return Grant{}, err
	}
	return grant, nil
}

func (a *Authorizer) trustedGrant(ctx context.Context, grantReceipt string) (Grant, error) {
	if a == nil || a.Callers == nil || a.Grants == nil {
		return Grant{}, errors.New("browser grant verification is unavailable")
	}
	caller, err := a.Callers.ResolveTrustedCaller(ctx)
	if err != nil {
		return Grant{}, fmt.Errorf("resolve trusted browser caller: %w", err)
	}
	if strings.TrimSpace(caller.RunID) == "" || strings.TrimSpace(grantReceipt) == "" {
		return Grant{}, ErrRunNotAuthorized
	}
	grant, err := a.Grants.ReadBrowserGrant(ctx, strings.TrimSpace(grantReceipt))
	if err != nil {
		return Grant{}, fmt.Errorf("read browser capability grant: %w", err)
	}
	if grant.ID != strings.TrimSpace(grantReceipt) || grant.RunID != caller.RunID || grant.RevokedAt != nil || !grant.ExpiresAt.After(a.now()) {
		return Grant{}, ErrRunNotAuthorized
	}
	return grant, nil
}

func (a *Authorizer) now() time.Time {
	if a != nil && a.Now != nil {
		return a.Now().UTC()
	}
	return time.Now().UTC()
}

func validateInitialURL(raw string, allowedOrigins []string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (!strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https")) {
		return errors.New("browser initial URL must be an absolute http(s) URL")
	}
	for _, origin := range allowedOrigins {
		allowed, err := url.Parse(origin)
		if err == nil && strings.EqualFold(parsed.Scheme, allowed.Scheme) && strings.EqualFold(parsed.Host, allowed.Host) {
			return nil
		}
	}
	return errors.New("browser initial URL is outside the granted origin scope")
}

// Authorize binds a trusted, persisted run to its declared browser capability.
// It refuses any request that widens the run policy; session creation stays in BAS.
func (a *Authorizer) Authorize(ctx context.Context, req Request) (Grant, error) {
	if a == nil || a.Callers == nil || a.Runs == nil || a.Issuer == nil {
		return Grant{}, errors.New("browser capability authorization is unavailable")
	}
	caller, err := a.Callers.ResolveTrustedCaller(ctx)
	if err != nil {
		return Grant{}, fmt.Errorf("resolve trusted browser caller: %w", err)
	}
	if strings.TrimSpace(caller.RunID) == "" {
		return Grant{}, ErrRunNotAuthorized
	}
	persisted, err := a.Runs.BrowserCapability(ctx, caller.RunID)
	if err != nil {
		return Grant{}, fmt.Errorf("load run browser capability: %w", err)
	}
	if !persisted.Enabled {
		return Grant{}, ErrRunNotAuthorized
	}
	if !candidateAllowed(persisted.Candidates, caller.Harness, caller.ResourceRole) {
		return Grant{}, ErrNoEligibleCandidate
	}
	if err := Validate(persisted); err != nil {
		return Grant{}, err
	}
	now := time.Now().UTC()
	if a.Now != nil {
		now = a.Now().UTC()
	}
	ttl := a.TTL
	if ttl <= 0 || ttl > 24*time.Hour {
		ttl = 1 * time.Hour
	}
	return a.Issuer.IssueBrowserGrant(ctx, caller, persisted, now.Add(ttl))
}

func candidateAllowed(candidates []Candidate, harness string, resourceRole string) bool {
	for _, candidate := range candidates {
		if candidate.Harness == harness && strings.TrimSpace(candidate.ResourceRole) == strings.TrimSpace(resourceRole) {
			return true
		}
	}
	return false
}

func sameStrings(a, b []string) bool {
	left := append([]string(nil), a...)
	right := append([]string(nil), b...)
	sort.Strings(left)
	sort.Strings(right)
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func Validate(capability Capability) error {
	if !capability.Enabled {
		return ErrRunNotAuthorized
	}
	if capability.Mode != ModeManaged && capability.Mode != ModeInstalled && capability.Mode != ModeAttached && capability.Mode != ModeRemote {
		return fmt.Errorf("unsupported browser mode %q", capability.Mode)
	}
	if capability.Permissions.Level != ProfileObserve && capability.Permissions.Level != ProfileAct && capability.Permissions.Level != ProfileUnrestricted {
		return fmt.Errorf("unsupported browser permission profile %q", capability.Permissions.Level)
	}
	if strings.TrimSpace(capability.Permissions.ProfileID) == "" {
		return errors.New("browser permission profile requires a stable profile ID")
	}
	budget := capability.Budget
	if budget.SessionMinutes <= 0 || budget.Actions <= 0 || budget.Bytes <= 0 || budget.InferenceTokens < 0 {
		return errors.New("browser capability requires positive session, action and byte limits; inference token limit cannot be negative")
	}
	if budget.SessionMinutes > MaxSessionMinutes || budget.Actions > MaxActions || budget.Bytes > MaxBytes || budget.InferenceTokens > MaxInferenceTokens {
		return errors.New("browser capability exceeds the hard session, action, byte or token ceiling")
	}
	for name, entries := range map[string][]string{
		"allowedOrigins":     capability.Permissions.AllowedOrigins,
		"credentialRefs":     capability.Permissions.CredentialRefs,
		"extensionIDs":       capability.Permissions.ExtensionIDs,
		"requireApprovalFor": capability.Permissions.RequireApprovalFor,
	} {
		if len(entries) > MaxPermissionEntries {
			return fmt.Errorf("browser permission %s exceeds %d entries", name, MaxPermissionEntries)
		}
	}
	for _, origin := range capability.Permissions.AllowedOrigins {
		parsed, err := url.Parse(strings.TrimSpace(origin))
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || (!strings.EqualFold(parsed.Scheme, "https") && !strings.EqualFold(parsed.Scheme, "http")) || parsed.Path != "" && parsed.Path != "/" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("browser allowed origin %q must be an http(s) origin", origin)
		}
	}
	if capability.Permissions.Level == ProfileObserve && capability.Permissions.AllowHistory {
		return errors.New("observe profile cannot grant browser history access")
	}
	if capability.Permissions.Control && !capability.Permissions.View {
		return errors.New("browser control requires explicit view permission")
	}
	if capability.Permissions.ExtensionInstall || capability.Permissions.ExtensionLoad {
		if capability.Mode != ModeManaged {
			return errors.New("extension install and load are supported only in managed browser mode")
		}
		if capability.Permissions.ProfileMode != SessionProfileFresh {
			return errors.New("extension install and load require the persisted fresh managed profile mode")
		}
		if len(capability.Permissions.ExtensionIDs) == 0 {
			return errors.New("extension install and load require an existing exact extension ID allowlist")
		}
	}
	if capability.Permissions.PerformanceTrace {
		if capability.Mode != ModeManaged {
			return errors.New("performance trace is supported only in managed browser mode")
		}
		if capability.Permissions.ProfileMode != SessionProfileFresh {
			return errors.New("performance trace requires the persisted fresh managed profile mode")
		}
	}
	if capability.Permissions.AppMonitorMetrics {
		if capability.Mode != ModeManaged {
			return errors.New("App Monitor metrics are supported only in managed browser mode")
		}
		if capability.Permissions.ProfileMode != SessionProfileFresh {
			return errors.New("App Monitor metrics require the persisted fresh managed profile mode")
		}
		if !capability.Permissions.View {
			return errors.New("App Monitor metrics require explicit browser view permission")
		}
	}
	trace := capability.Permissions.AppMonitorTrace
	traceDuration := capability.Permissions.AppMonitorTraceMaxDurationMS
	traceBytes := capability.Permissions.AppMonitorTraceMaxBytes
	if trace {
		if capability.Mode != ModeManaged || capability.Permissions.ProfileMode != SessionProfileFresh {
			return errors.New("App Monitor trace is supported only in a fresh managed browser profile")
		}
		if !capability.Permissions.View {
			return errors.New("App Monitor trace requires explicit browser view permission")
		}
		if traceDuration <= 0 || traceDuration > MaxAppMonitorTraceDurationMS || traceBytes <= 0 || traceBytes > MaxAppMonitorTraceBytes {
			return errors.New("App Monitor trace requires positive duration and byte limits within the hard capture ceiling")
		}
	} else if traceDuration != 0 || traceBytes != 0 {
		return errors.New("App Monitor trace budgets require the explicit trace permission")
	}
	switch capability.Permissions.AudioMode {
	case AudioProfileNone:
	case AudioProfileFakeMicrophone, AudioProfilePipeWire:
		if capability.Mode != ModeManaged || !capability.Permissions.View || !capability.Permissions.Control || capability.Permissions.Level == ProfileObserve || len(capability.Permissions.AllowedOrigins) == 0 {
			return errors.New("audio capture requires a managed, view-enabled, act-capable profile with an explicit origin scope")
		}
	default:
		return fmt.Errorf("unsupported browser audio profile %q", capability.Permissions.AudioMode)
	}
	if capability.Permissions.ProfileMode != "" {
		want, ok := ProfileModeForMode(capability.Mode)
		if !ok || capability.Permissions.ProfileMode != want {
			return fmt.Errorf("browser session profile mode %q does not match backend mode %q", capability.Permissions.ProfileMode, capability.Mode)
		}
	}
	if capability.Permissions.IdentityProfile != nil {
		want, ok := ProfileModeForMode(capability.Mode)
		if !ok || capability.Permissions.ProfileMode != want {
			return errors.New("identity profile requires an explicit persisted profile mode matching its browser backend")
		}
	}
	if err := ValidateIdentityProfile(capability.Mode, capability.Permissions.IdentityProfile); err != nil {
		return err
	}
	if len(capability.Candidates) == 0 {
		return ErrNoEligibleCandidate
	}
	if len(capability.Candidates) > MaxCandidates {
		return fmt.Errorf("browser capability exceeds the %d-candidate policy limit", MaxCandidates)
	}
	seen := make(map[string]struct{}, len(capability.Candidates))
	for _, candidate := range capability.Candidates {
		if strings.TrimSpace(candidate.Harness) == "" || strings.TrimSpace(candidate.ResourceRole) == "" {
			return errors.New("browser capability candidate requires harness and resource role")
		}
		if candidate.ResourceRole != ResourceRoleSession {
			return fmt.Errorf("unsupported browser resource role %q", candidate.ResourceRole)
		}
		key := strings.TrimSpace(candidate.Harness) + "\x00" + strings.TrimSpace(candidate.ResourceRole)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate browser capability candidate %s/%s", candidate.Harness, candidate.ResourceRole)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// ValidateIdentityProfile enforces the currently supported identity surface.
// Only managed Playwright contexts can apply fresh-session customization;
// attached tabs preserve their real identity and other modes remain unavailable.
func ValidateIdentityProfile(mode Mode, profile *IdentityProfile) error {
	if profile == nil {
		return nil
	}
	if mode == ModeInstalled {
		if profile.Preset != "" || profile.DeviceScaleFactor != 0 || profile.Locale != "" || profile.TimezoneID != "" || profile.ColorScheme != "" {
			return errors.New("installed-binary identity customization supports viewport only; preset, device scale, locale, timezone and color scheme are unsupported")
		}
		return validateIdentityViewport(profile)
	}
	if mode != ModeManaged {
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
		if len(profile.Locale) > 64 || strings.TrimSpace(profile.Locale) != profile.Locale || !validLocaleTag(profile.Locale) {
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

func validateIdentityViewport(profile *IdentityProfile) error {
	if profile.ViewportWidth < 0 || profile.ViewportWidth > 7680 || profile.ViewportHeight < 0 || profile.ViewportHeight > 4320 {
		return errors.New("identity viewport must be within 7680x4320")
	}
	if (profile.ViewportWidth == 0) != (profile.ViewportHeight == 0) {
		return errors.New("identity viewport width and height must be set together")
	}
	return nil
}

func validLocaleTag(value string) bool {
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

// ResolveCandidates intersects a requested run capability with the harnesses
// resolved from the owning Agent Manager execution role policy. An omitted
// candidate list means every eligible harness may use the one qualified BAS
// session role; supplied entries can only narrow that set.
func ResolveCandidates(requested []Candidate, eligibleHarnesses []string) ([]Candidate, error) {
	eligible := make(map[string]struct{}, len(eligibleHarnesses))
	for _, harness := range eligibleHarnesses {
		harness = strings.TrimSpace(harness)
		if harness != "" {
			eligible[harness] = struct{}{}
		}
	}
	if len(eligible) == 0 {
		return nil, ErrNoEligibleCandidate
	}
	requestedSet := make(map[string]struct{}, len(requested))
	for _, candidate := range requested {
		if strings.TrimSpace(candidate.Harness) == "" || candidate.ResourceRole != ResourceRoleSession {
			return nil, fmt.Errorf("browser candidate must use an eligible harness and resource role %q", ResourceRoleSession)
		}
		if _, ok := eligible[candidate.Harness]; !ok {
			return nil, fmt.Errorf("browser harness %q is not present in the resolved execution role", candidate.Harness)
		}
		requestedSet[candidate.Harness] = struct{}{}
	}
	resolved := make([]Candidate, 0, len(eligible))
	for _, harness := range eligibleHarnesses {
		harness = strings.TrimSpace(harness)
		if _, ok := eligible[harness]; !ok {
			continue
		}
		if len(requestedSet) > 0 {
			if _, ok := requestedSet[harness]; !ok {
				continue
			}
		}
		resolved = append(resolved, Candidate{Harness: harness, ResourceRole: ResourceRoleSession})
		delete(eligible, harness)
	}
	if len(resolved) == 0 {
		return nil, ErrNoEligibleCandidate
	}
	return resolved, nil
}

// Within reports whether child is no broader than the parent's run-scoped
// capability. Nil child means the child asks for no browser access.
func Within(parent, child *Capability) bool {
	if child == nil || !child.Enabled {
		return true
	}
	if parent == nil || !parent.Enabled || parent.Mode != child.Mode || Validate(*child) != nil {
		return false
	}
	if permissionRank(child.Permissions.Level) > permissionRank(parent.Permissions.Level) ||
		child.Permissions.ProfileMode != parent.Permissions.ProfileMode ||
		child.Permissions.AudioMode != AudioProfileNone && child.Permissions.AudioMode != parent.Permissions.AudioMode ||
		child.Budget.SessionMinutes > parent.Budget.SessionMinutes || child.Budget.Actions > parent.Budget.Actions ||
		child.Budget.Bytes > parent.Budget.Bytes || child.Budget.InferenceTokens > parent.Budget.InferenceTokens ||
		(child.Permissions.AllowHistory && !parent.Permissions.AllowHistory) ||
		(child.Permissions.View && !parent.Permissions.View) || (child.Permissions.Control && !parent.Permissions.Control) ||
		(child.Permissions.ExtensionInstall && !parent.Permissions.ExtensionInstall) ||
		(child.Permissions.ExtensionLoad && !parent.Permissions.ExtensionLoad) ||
		(child.Permissions.PerformanceTrace && !parent.Permissions.PerformanceTrace) ||
		(child.Permissions.AppMonitorMetrics && !parent.Permissions.AppMonitorMetrics) ||
		(child.Permissions.AppMonitorTrace && !parent.Permissions.AppMonitorTrace) ||
		!identityProfileWithin(parent.Permissions.IdentityProfile, child.Permissions.IdentityProfile) ||
		child.Permissions.AppMonitorTraceMaxDurationMS > parent.Permissions.AppMonitorTraceMaxDurationMS ||
		child.Permissions.AppMonitorTraceMaxBytes > parent.Permissions.AppMonitorTraceMaxBytes {
		return false
	}
	if !setWithin(parent.Permissions.AllowedOrigins, child.Permissions.AllowedOrigins) ||
		!setWithin(parent.Permissions.CredentialRefs, child.Permissions.CredentialRefs) ||
		!setWithin(parent.Permissions.ExtensionIDs, child.Permissions.ExtensionIDs) ||
		!setWithin(parent.Permissions.RequireApprovalFor, child.Permissions.RequireApprovalFor) {
		return false
	}
	for _, requested := range child.Candidates {
		if !candidateAllowed(parent.Candidates, requested.Harness, requested.ResourceRole) {
			return false
		}
	}
	return true
}

func identityProfileWithin(parent, child *IdentityProfile) bool {
	if child == nil {
		return true
	}
	if parent == nil {
		return false
	}
	return (child.Preset == "" || child.Preset == parent.Preset) &&
		(child.ViewportWidth == 0 || child.ViewportWidth == parent.ViewportWidth) &&
		(child.ViewportHeight == 0 || child.ViewportHeight == parent.ViewportHeight) &&
		(child.DeviceScaleFactor == 0 || child.DeviceScaleFactor == parent.DeviceScaleFactor) &&
		(child.Locale == "" || child.Locale == parent.Locale) &&
		(child.TimezoneID == "" || child.TimezoneID == parent.TimezoneID) &&
		(child.ColorScheme == "" || child.ColorScheme == parent.ColorScheme)
}

func permissionRank(level PermissionProfile) int {
	switch level {
	case ProfileObserve:
		return 0
	case ProfileAct:
		return 1
	case ProfileUnrestricted:
		return 2
	default:
		return 3
	}
}

func setWithin(parent, child []string) bool {
	allowed := make(map[string]struct{}, len(parent))
	for _, item := range parent {
		allowed[strings.TrimSpace(item)] = struct{}{}
	}
	for _, item := range child {
		if _, ok := allowed[strings.TrimSpace(item)]; !ok {
			return false
		}
	}
	return true
}
