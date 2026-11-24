// Package agentaccess exposes a small, permission-checked surface for agents
// and harnesses over BAS's existing session owner. It deliberately does not
// create a driver, CDP transport, or independent browser lifecycle.
package agentaccess

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/automation/harness"
	"github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

var (
	ErrAuthorizerRequired        = errors.New("browser access authorizer resolver is required")
	ErrSessionNotOwned           = errors.New("session is not owned by this BAS broker")
	ErrTraceNotFound             = errors.New("trace not found")
	ErrExtensionOwnerUnavailable = errors.New("BAS session extension controller unavailable")
)

// ProfileResolver is implemented by the existing permission owner. It
// resolves a caller-selected profile reference and policy on every operation
// so revokes and policy changes are observed without a broker-local grant store.
type ProfileResolver interface {
	Resolve(context.Context, string) (sessionpermission.Profile, sessionpermission.Authorizer, error)
}

type CallerResolver = sessionpermission.CallerResolver

// TrustedOpenContextResolver supplies the receipt and identity material from
// the authenticated Agent Manager run context. The values never come from JSON.
type TrustedOpenContextResolver interface {
	ResolveBrowserOpen(context.Context) (TrustedBrowserOpenContext, error)
}

type TrustedBrowserOpenContext struct {
	CallerID      string
	GrantReceipt  string
	IdentityToken string
	PermissionRef string
	RunID         string
}

// OpenRequest is intentionally explicit: callers choose a named permission
// profile and a fresh/reused BAS browser context. No real-user browser is
// silently attached.
type OpenRequest struct {
	PermissionRef        string `json:"permission_ref"`
	GrantReceipt         string `json:"-"`
	ConsentReceipt       string `json:"consent_receipt,omitempty"`
	TabID                string `json:"tab_id,omitempty"`
	Backend              string `json:"backend,omitempty"`
	StartURL             string `json:"start_url,omitempty"`
	ReuseMode            string `json:"reuse_mode,omitempty"` // fresh, clean, reuse
	ViewportWidth        int    `json:"viewport_width,omitempty"`
	ViewportHeight       int    `json:"viewport_height,omitempty"`
	AudioMode            string `json:"audio_mode,omitempty"`
	AudioPlaybackPauseMS int    `json:"audio_playback_pause_ms,omitempty"`
}

type SessionInfo struct {
	SessionID           string                                    `json:"session_id"`
	PermissionRef       string                                    `json:"permission_ref"`
	Mode                string                                    `json:"mode"`
	Role                string                                    `json:"role"`
	Capabilities        []string                                  `json:"capabilities"`
	PairingCode         string                                    `json:"pairing_code,omitempty"`
	PairingExpires      time.Time                                 `json:"pairing_expires,omitempty"`
	AudioDeviceEvidence *driver.AudioDeviceEvidence               `json:"audio_device_evidence,omitempty"`
	IdentityProfile     *sessionpermission.BrowserIdentityProfile `json:"identity_profile,omitempty"`
	ActualViewport      *driver.ActualViewport                    `json:"actual_viewport,omitempty"`
	IdentityEvidence    *driver.IdentityEvidence                  `json:"identity_evidence,omitempty"`
}

// BrowserInferenceBinding is an owner-resolved active browser run scope for
// one Gateway request. It contains correlation identifiers only, never the
// signed Agent Manager token.
type BrowserInferenceBinding struct {
	RunID        string
	SessionID    string
	GrantReceipt string
}

type AttachRequest struct {
	SessionID       string `json:"session_id"`
	PermissionRef   string `json:"permission_ref"`
	TabID           string `json:"tab_id,omitempty"`
	ConsentReceipt  string `json:"consent_receipt,omitempty"`
	ApprovalReceipt string `json:"approval_receipt,omitempty"`
}

type ObserveRequest struct {
	PageID    string `json:"page_id,omitempty"`
	TestID    string `json:"test_id,omitempty"`
	WaitUntil string `json:"wait_until,omitempty"`
	SettleMs  int    `json:"settle_ms,omitempty"`
}

type Observation struct {
	SessionID       string                            `json:"session_id"`
	ObservedAt      time.Time                         `json:"observed_at"`
	URL             string                            `json:"url"`
	Title           string                            `json:"title"`
	Text            string                            `json:"text,omitempty"`
	Screenshot      *driver.CaptureScreenshotResponse `json:"screenshot,omitempty"`
	EvidenceWarning string                            `json:"evidence_warning,omitempty"`
}

type ExtensionInventory struct {
	SessionID  string                                      `json:"session_id"`
	Binding    ExtensionSessionBinding                     `json:"binding"`
	ObservedAt time.Time                                   `json:"observed_at"`
	Extensions []sessionpermission.ExtensionInventoryEntry `json:"extensions"`
}

type ActRequest struct {
	Instruction     contracts.CompiledInstruction `json:"instruction"`
	TargetURL       string                        `json:"target_url,omitempty"`
	ApprovalReceipt string                        `json:"approval_receipt,omitempty"`
}

type ActionResult struct {
	SessionID       string                `json:"session_id"`
	Outcome         contracts.StepOutcome `json:"outcome"`
	EvidenceWarning string                `json:"evidence_warning,omitempty"`
}

// TraceEvent records a successfully accepted operation with enough typed
// material to promote an action trace into a deterministic BAS workflow.
// It intentionally excludes credentials, cookies, raw CDP endpoints and
// screenshot bytes.
type TraceEvent struct {
	Sequence           uint64                         `json:"sequence"`
	SessionID          string                         `json:"session_id"`
	Kind               string                         `json:"kind"`
	PermissionClass    string                         `json:"permission_class,omitempty"`
	EffectClass        string                         `json:"effect_class,omitempty"`
	OccurredAt         time.Time                      `json:"occurred_at"`
	URL                string                         `json:"url,omitempty"`
	TargetOrigin       string                         `json:"target_origin,omitempty"`
	Title              string                         `json:"title,omitempty"`
	Instruction        *contracts.CompiledInstruction `json:"instruction,omitempty"`
	Outcome            *contracts.StepOutcome         `json:"outcome,omitempty"`
	Parameters         []Parameter                    `json:"parameters,omitempty"`
	UnpromotableReason string                         `json:"unpromotable_reason,omitempty"`
}

type Trace struct {
	ID            string       `json:"id"`
	SessionID     string       `json:"session_id"`
	PermissionRef string       `json:"permission_ref"`
	Events        []TraceEvent `json:"events"`
}

// ExtensionController is a narrow adapter to scenario-to-extension's artifact
// owner and the BAS session owner's extension operations. Every call receives
// the exact admitted session, lease, profile and caller binding. Install must
// compare the staged artifact to the full owner-derived manifest before browser
// mutation; a receipt mismatch after installation is not a substitute for that
// precondition. Implementations must never launch a second browser owner or
// resolve arbitrary host filesystem paths.
type ExtensionController interface {
	Install(context.Context, ExtensionInstallRequest) (ExtensionReceipt, error)
	Load(context.Context, ExtensionLifecycleRequest) (ExtensionReceipt, error)
	Reload(context.Context, ExtensionLifecycleRequest) (ExtensionReceipt, error)
	Unload(context.Context, ExtensionLifecycleRequest) (ExtensionReceipt, error)
	Validate(context.Context, ExtensionLifecycleRequest) (ExtensionValidation, error)
	Inventory(context.Context, ExtensionSessionBinding) (ExtensionInventoryReceipt, error)
}

// ExtensionSessionFinalizer releases controller-owned staging resources after
// the canonical BAS session owner has closed the exact browser session.
type ExtensionSessionFinalizer interface {
	FinalizeClosedSession(context.Context, string) error
}

// ExtensionSessionBinding carries the current BAS lease and permission identity
// into every extension operation. Implementations must resolve SessionID through
// BAS's existing session owner and reject any mismatch; these fields are not a
// second session or permission authority.
type ExtensionSessionBinding struct {
	SessionID     string `json:"session_id"`
	Backend       string `json:"backend"`
	ProfileID     string `json:"profile_id"`
	PermissionRef string `json:"permission_ref"`
	OwnerID       string `json:"owner_id"`
	LeaseID       string `json:"lease_id"`
	CallerID      string `json:"caller_id"`
}

type ExtensionInstallRequest struct {
	Binding  ExtensionSessionBinding `json:"binding"`
	Artifact ExtensionArtifact       `json:"artifact"`
}

type ExtensionLifecycleRequest struct {
	Binding  ExtensionSessionBinding `json:"binding"`
	Artifact ExtensionArtifact       `json:"artifact"`
}

type ExtensionInventoryReceipt struct {
	Binding    ExtensionSessionBinding                     `json:"binding"`
	ObservedAt time.Time                                   `json:"observed_at"`
	Extensions []sessionpermission.ExtensionInventoryEntry `json:"extensions"`
}

// ExtensionArtifact is owner-derived metadata for one immutable build artifact.
// Permission values come from its verified manifest, never caller input.
type ExtensionArtifact struct {
	ArtifactRef     string
	ExtensionID     string
	Digest          string
	ManifestSHA256  string
	Version         string
	Permissions     []string
	HostPermissions []string
	PermissionProbe string
}

// ExtensionArtifactVerifier belongs at the scenario-to-extension boundary. It
// resolves opaque build IDs, verifies the artifact digest, derives the Chrome
// ID and manifest permission set, and returns only after binding all four to
// the requested install.
type ExtensionArtifactVerifier interface {
	VerifyBrowserExtensionArtifact(context.Context, string, string) (ExtensionArtifact, error)
}

// CapabilityReporter returns owner-backed runtime qualification receipts. A
// source-only capability list is never sufficient for an agent session.
type CapabilityReporter interface {
	BackendReports(context.Context) ([]sessionpermission.BackendReport, error)
}

type ExtensionReceipt struct {
	SessionID      string    `json:"session_id"`
	Backend        string    `json:"backend"`
	ProfileID      string    `json:"profile_id"`
	PermissionRef  string    `json:"permission_ref"`
	OwnerID        string    `json:"owner_id"`
	LeaseID        string    `json:"lease_id"`
	CallerID       string    `json:"caller_id"`
	ArtifactRef    string    `json:"artifact_ref"`
	Operation      string    `json:"operation"`
	ExtensionID    string    `json:"extension_id"`
	Version        string    `json:"version"`
	Digest         string    `json:"digest"`
	ManifestSHA256 string    `json:"manifest_sha256"`
	State          string    `json:"state"`
	OccurredAt     time.Time `json:"occurred_at"`
}

type ExtensionValidation struct {
	Receipt ExtensionReceipt `json:"receipt"`
	Valid   bool             `json:"valid"`
	Checks  map[string]bool  `json:"checks"`
	Message string           `json:"message,omitempty"`
}

type Broker struct {
	manager              *session.Manager
	profiles             ProfileResolver
	openAuthorizer       sessionpermission.OpenAuthorizer
	caller               CallerResolver
	openContext          TrustedOpenContextResolver
	capabilities         CapabilityReporter
	extensions           ExtensionController
	extensionVerifier    ExtensionArtifactVerifier
	bridge               SelectedTabBridge
	tabConsentIssuer     TabConsentIssuer
	streamVerifier       StreamReceiptVerifier
	streamCookies        SessionStreamCookieIssuer
	evidence             EvidenceRecorder
	audioFixtureResolver AudioFixtureResolver
	mu                   sync.RWMutex
	entries              map[string]*entry
}

func (b *Broker) SetEvidenceRecorder(recorder EvidenceRecorder) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.evidence = recorder
}

func (b *Broker) recordSessionEvidence(ctx context.Context, e *entry, state string) error {
	b.mu.RLock()
	recorder := b.evidence
	b.mu.RUnlock()
	if recorder == nil {
		return errors.New("Agent Manager browser evidence recorder is unavailable")
	}
	return recorder.RecordBrowserSession(ctx, e.sessionID, sessionpermission.Backend(e.backend), e.permissionRef, state)
}

func (b *Broker) recordActionEvidence(ctx context.Context, e *entry, action, outcome string) error {
	// This counter is broker-owned. It counts page observations/actions that
	// reached the evidence boundary, including denials when callers explicitly
	// report them. It is never read from a client-provided close summary.
	e.evidenceActions++
	b.mu.RLock()
	recorder := b.evidence
	b.mu.RUnlock()
	if recorder == nil {
		return errors.New("Agent Manager browser evidence recorder is unavailable")
	}
	return recorder.RecordBrowserAction(ctx, e.sessionID, sessionpermission.Backend(e.backend), e.permissionRef, action, outcome)
}

func (b *Broker) addEvidenceBytes(e *entry, count int) {
	if count <= 0 {
		return
	}
	if int64(count) > int64(^uint64(0)>>1)-e.evidenceBytes {
		e.evidenceBytes = int64(^uint64(0) >> 1)
		return
	}
	e.evidenceBytes += int64(count)
}

// ForwardControlInput is the existing Web Console stream-input adapter for a
// broker-owned managed session. It returns handled=false for BAS sessions
// outside this facade, so the existing recording owner remains the fallback.
// The identity proof comes from the server-held run context on this entry;
// callers cannot supply a caller, run, lease, or permission reference.
func (b *Broker) ForwardControlInput(sessionID string, input []byte) (*driver.ForwardInputResponse, bool, error) {
	e, err := b.get(sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotOwned) {
			return nil, false, nil
		}
		return nil, true, err
	}
	if err := b.lockEntry(e); err != nil {
		return nil, true, err
	}
	defer e.opMu.Unlock()
	if e.session == nil || e.identityToken == "" || e.grantReceipt == "" {
		return nil, true, errors.New("browser session has no managed control-input authority")
	}
	b.addEvidenceBytes(e, len(input))
	ctx := WithBrowserAgentCredentials(context.Background(), e.identityToken, e.grantReceipt)
	receipt, err := e.session.ForwardInput(ctx, input)
	category := inputEvidenceCategory(input)
	if err != nil {
		if category != "" {
			_ = b.recordActionEvidence(ctx, e, category, "denied")
		}
		return nil, true, err
	}
	b.addEvidenceBytes(e, evidencePayloadSize(receipt))
	if category != "" {
		traceOutcome := contracts.StepOutcome{Success: true, StepType: "live_input"}
		b.record(e, TraceEvent{Kind: "action", OccurredAt: time.Now().UTC(), Outcome: &traceOutcome, UnpromotableReason: "live input has no deterministic parameterized flow step"})
		_ = b.recordActionEvidence(ctx, e, category, "completed")
	}
	return receipt, true, nil
}

func inputEvidenceCategory(raw []byte) string {
	var input struct {
		Type   string `json:"type"`
		Action string `json:"action"`
		Text   string `json:"text"`
	}
	if json.Unmarshal(raw, &input) != nil {
		return ""
	}
	switch input.Type {
	case "keyboard":
		if input.Text != "" {
			return "type"
		}
	case "wheel":
		return "scroll"
	case "pointer":
		if input.Action == "click" || input.Action == "down" || input.Action == "up" {
			return "click"
		}
	}
	return ""
}

type entry struct {
	session                                                            *session.Session
	runtimeEvidence                                                    *BrowserRuntimeEvidence
	permissionRef                                                      string
	profile                                                            sessionpermission.Profile
	policy                                                             sessionpermission.Authorizer
	opMu                                                               sync.Mutex
	backend                                                            string
	trace                                                              Trace
	sessionID, leaseID, ownerID, callerID, grantReceipt, identityToken string
	runID                                                              string
	pairingCodeHash, pairingNonce                                      string
	pairingExpires                                                     time.Time
	openedAt                                                           time.Time
	pairingConsumed, closed                                            bool
	handedOff                                                          bool
	tabID, tabURL, tabTitle, consentReceipt                            string
	socketTicketHash                                                   string
	socketTicketExpiry                                                 time.Time
	socketConnected                                                    bool
	leaseReceipt, leaseCaller                                          string
	attachedSocket                                                     *websocket.Conn
	attachedTab                                                        AttachedTab
	closedAt                                                           time.Time
	evidenceActions, evidenceBytes                                     int64
	evidenceProof                                                      *closedBrowserEvidenceProof
	settlementResult                                                   BrowserSettlementResult
	revoking                                                           atomic.Bool
	consentRevokeMu                                                    sync.Mutex
	consentRevoked                                                     bool
	installedExtensions                                                map[string]ExtensionArtifact
	appMonitorCapture                                                  *appMonitorCaptureState
}

func NewBroker(manager *session.Manager, profiles ProfileResolver, openAuthorizer sessionpermission.OpenAuthorizer, caller CallerResolver, extensions ...ExtensionController) (*Broker, error) {
	if manager == nil {
		return nil, errors.New("BAS session manager is required")
	}
	if profiles == nil {
		return nil, ErrAuthorizerRequired
	}
	if openAuthorizer == nil {
		return nil, errors.New("browser session open authorizer is required")
	}
	if caller == nil {
		return nil, errors.New("trusted browser caller resolver is required")
	}
	b := &Broker{manager: manager, profiles: profiles, openAuthorizer: openAuthorizer, caller: caller, entries: make(map[string]*entry), bridge: WebSocketSelectedTabBridge{}}
	if len(extensions) > 0 {
		b.extensions = extensions[0]
	}
	return b, nil
}

// SetTrustedOpenContextResolver wires the Agent Manager adapter after broker
// construction. Until configured, Open fails closed.
func (b *Broker) SetTrustedOpenContextResolver(resolver TrustedOpenContextResolver) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.openContext = resolver
}

func (b *Broker) SetCapabilityReporter(reporter CapabilityReporter) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.capabilities = reporter
}

func (b *Broker) SetExtensionArtifactVerifier(verifier ExtensionArtifactVerifier) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.extensionVerifier = verifier
}

func (b *Broker) Capabilities(ctx context.Context, required []sessionpermission.Capability) ([]sessionpermission.CapabilityNegotiation, error) {
	b.mu.RLock()
	reporter := b.capabilities
	b.mu.RUnlock()
	reports := []sessionpermission.BackendReport{
		{Backend: sessionpermission.BackendManaged, Unavailable: map[sessionpermission.Capability]string{
			sessionpermission.CapabilityObserve: "managed session has no live qualification receipt", sessionpermission.CapabilityControl: "managed session has no live qualification receipt",
			sessionpermission.CapabilityAudioCapture: "managed microphone capture has no live qualification receipt", sessionpermission.CapabilityAudioDeviceEvidence: "PipeWire device evidence has no live qualification receipt",
			sessionpermission.CapabilityIdentityCustomization: "managed identity customization has no live qualification receipt",
			sessionpermission.CapabilityAppMonitorMetrics:     "App Monitor metrics read has no live owner qualification receipt",
			sessionpermission.CapabilityAppMonitorTrace:       "bounded App Monitor trace has no live owner qualification receipt",
		}},
		{Backend: sessionpermission.BackendInstalled, Unavailable: map[sessionpermission.Capability]string{
			sessionpermission.CapabilityObserve:                  "installed browser has no live qualification receipt",
			sessionpermission.CapabilityIdentityCustomization:    "installed-binary viewport identity has no live qualification receipt",
			sessionpermission.CapabilityExtensionInventory:       "installed-binary extension inventory adapter is unavailable",
			sessionpermission.CapabilityExtensionHostPermissions: "installed-binary extension permission evidence is unavailable",
			sessionpermission.CapabilityExtensionInstall:         "extension installation is supported only by the managed BAS session adapter",
			sessionpermission.CapabilityExtensionLoad:            "extension loading is supported only by the managed BAS session adapter",
			sessionpermission.CapabilityAppMonitorMetrics:        "App Monitor capture is managed-mode only",
			sessionpermission.CapabilityAppMonitorTrace:          "App Monitor trace is managed-mode only",
		}},
		{Backend: sessionpermission.BackendUserAttached, Unavailable: map[sessionpermission.Capability]string{
			sessionpermission.CapabilityObserve:               "selected-tab attachment has no live consent receipt",
			sessionpermission.CapabilityControl:               "selected-tab attachment has no live consent receipt",
			sessionpermission.CapabilityPreserveUserState:     "selected-tab preservation has not been qualified on a real signed-in browser",
			sessionpermission.CapabilityReconnect:             "selected-tab reconnect has no live qualification receipt",
			sessionpermission.CapabilityIdentityCustomization: "user-attached mode preserves the real tab identity; fresh-session customization is not applied",
			sessionpermission.CapabilityAppMonitorMetrics:     "App Monitor capture is managed-mode only",
			sessionpermission.CapabilityAppMonitorTrace:       "App Monitor trace is managed-mode only",
		}},
		{Backend: sessionpermission.BackendRemote, Unavailable: map[sessionpermission.Capability]string{sessionpermission.CapabilityObserve: "remote browser has no live qualification receipt", sessionpermission.CapabilityIdentityCustomization: "remote identity customization adapter is unavailable", sessionpermission.CapabilityAppMonitorMetrics: "App Monitor capture is managed-mode only", sessionpermission.CapabilityAppMonitorTrace: "App Monitor trace is managed-mode only"}},
	}
	if reporter != nil {
		provided, err := reporter.BackendReports(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve browser backend qualification: %w", err)
		}
		reports, err = mergeBackendQualificationReports(reports, provided)
		if err != nil {
			return nil, err
		}
	}
	reports = b.filterCapabilities(reports)
	rows := sessionpermission.NegotiateCapabilities(reports, required)
	for i := range rows {
		rows[i].Identity = identityCustomizationForBackend(rows[i].Backend)
	}
	return rows, nil
}

func identityCustomizationForBackend(backend sessionpermission.Backend) *sessionpermission.IdentityCustomization {
	unsupported := map[string]string{
		"preset":               "not exposed by the trusted Agent Manager browser profile",
		"viewport":             "not exposed by the selected backend's trusted identity profile",
		"device_scale_factor":  "not exposed by the trusted Agent Manager browser profile",
		"locale":               "not exposed by the trusted Agent Manager browser profile",
		"timezone":             "not exposed by the trusted Agent Manager browser profile",
		"color_scheme":         "not exposed by the trusted Agent Manager browser profile",
		"user_agent":           "not exposed by the trusted Agent Manager browser profile",
		"engine":               "engine identity is unavailable from this backend",
		"engine_version":       "engine version is unavailable from this backend",
		"hardware_concurrency": "not exposed by the trusted Agent Manager browser profile",
		"device_memory":        "not exposed by the trusted Agent Manager browser profile",
		"geolocation":          "not exposed by the trusted Agent Manager browser profile",
		"proxy":                "not exposed by the trusted Agent Manager browser profile",
		"raw_headers":          "not exposed by the trusted Agent Manager browser profile",
	}
	if backend == sessionpermission.BackendManaged {
		for _, field := range []string{"preset", "viewport", "device_scale_factor", "locale", "timezone", "color_scheme", "user_agent", "engine", "engine_version"} {
			delete(unsupported, field)
		}
		return &sessionpermission.IdentityCustomization{
			Configured:  []string{"preset", "viewport", "device_scale_factor", "locale", "timezone", "color_scheme"},
			Observed:    []string{"engine", "engine_version", "user_agent", "locale", "timezone", "color_scheme", "device_scale_factor", "viewport"},
			Unsupported: unsupported,
		}
	}
	if backend == sessionpermission.BackendUserAttached {
		for key := range unsupported {
			unsupported[key] = "user-attached mode preserves the real tab identity and does not apply fresh-session customization"
		}
		return &sessionpermission.IdentityCustomization{Observed: []string{}, Unsupported: unsupported}
	}
	if backend == sessionpermission.BackendInstalled {
		for _, field := range []string{"viewport", "user_agent", "engine", "engine_version"} {
			delete(unsupported, field)
		}
		unsupported["preset"] = "installed-binary mode uses the executable's native browser defaults"
		unsupported["device_scale_factor"] = "installed-binary context does not expose scale customization"
		unsupported["locale"] = "installed-binary context does not expose locale customization"
		unsupported["timezone"] = "installed-binary context does not expose timezone customization"
		unsupported["color_scheme"] = "installed-binary context does not expose color-scheme customization"
		return &sessionpermission.IdentityCustomization{
			Configured:  []string{"viewport"},
			Observed:    []string{"engine", "engine_version", "user_agent", "locale", "timezone", "color_scheme", "device_scale_factor", "viewport"},
			Unsupported: unsupported,
		}
	}
	for key := range unsupported {
		unsupported[key] = "remote backend has no implemented identity customization adapter"
	}
	return &sessionpermission.IdentityCustomization{Unsupported: unsupported}
}

// mergeBackendQualificationReports keeps the four mode rows visible when a
// live owner only qualifies a subset. A partial owner report must not make an
// unreported backend disappear or inherit another backend's evidence.
func mergeBackendQualificationReports(defaults, provided []sessionpermission.BackendReport) ([]sessionpermission.BackendReport, error) {
	indices := make(map[sessionpermission.Backend]int, len(defaults))
	for i, report := range defaults {
		indices[report.Backend] = i
	}
	seen := make(map[sessionpermission.Backend]bool, len(provided))
	out := make([]sessionpermission.BackendReport, len(defaults))
	copy(out, defaults)
	for _, report := range provided {
		index, exists := indices[report.Backend]
		if !exists {
			return nil, fmt.Errorf("browser capability reporter returned unsupported backend %q", report.Backend)
		}
		if seen[report.Backend] {
			return nil, fmt.Errorf("browser capability reporter returned duplicate backend %q", report.Backend)
		}
		seen[report.Backend] = true
		unavailable := cloneCapabilityReasons(out[index].Unavailable)
		for capability, reason := range report.Unavailable {
			if unavailable == nil {
				unavailable = make(map[sessionpermission.Capability]string)
			}
			unavailable[capability] = reason
		}
		report.Unavailable = unavailable
		report.Capabilities = append([]sessionpermission.Capability(nil), report.Capabilities...)
		out[index] = report
	}
	return out, nil
}

func (b *Broker) filterCapabilities(reports []sessionpermission.BackendReport) []sessionpermission.BackendReport {
	b.mu.RLock()
	hasController := b.extensions != nil
	hasVerifier := b.extensionVerifier != nil
	b.mu.RUnlock()
	out := make([]sessionpermission.BackendReport, len(reports))
	for i, report := range reports {
		report.Capabilities = append([]sessionpermission.Capability(nil), report.Capabilities...)
		report.Unavailable = cloneCapabilityReasons(report.Unavailable)
		available := report.Capabilities[:0]
		for _, capability := range report.Capabilities {
			reason := ""
			switch capability {
			case sessionpermission.CapabilityIdentityCustomization:
				if report.Backend != sessionpermission.BackendManaged && report.Backend != sessionpermission.BackendInstalled {
					reason = "fresh-session identity customization is unsupported for attached and remote modes"
				}
			case sessionpermission.CapabilityExtensionInstall:
				if report.Backend != sessionpermission.BackendManaged {
					reason = "extension installation is supported only by the managed BAS session adapter"
				} else if !hasController {
					reason = "BAS extension controller is not configured"
				} else if !hasVerifier {
					reason = "verified extension artifact authority is not configured"
				}
			case sessionpermission.CapabilityExtensionLoad:
				if report.Backend != sessionpermission.BackendManaged {
					reason = "extension loading is supported only by the managed BAS session adapter"
				} else if !hasController {
					reason = "BAS extension controller is not configured"
				} else if !hasVerifier {
					reason = "verified extension artifact authority is not configured"
				}
			case sessionpermission.CapabilityExtensionInventory, sessionpermission.CapabilityExtensionHostPermissions:
				if report.Backend == sessionpermission.BackendManaged && !hasController {
					reason = "BAS managed extension inventory controller is not configured"
				} else if report.Backend == sessionpermission.BackendInstalled && capability == sessionpermission.CapabilityExtensionInventory {
					reason = "installed-binary extension inventory adapter is unavailable"
				} else if report.Backend == sessionpermission.BackendInstalled && capability == sessionpermission.CapabilityExtensionHostPermissions {
					reason = "installed-binary extension permission evidence is unavailable"
				} else if report.Backend != sessionpermission.BackendUserAttached && report.Backend != sessionpermission.BackendManaged {
					reason = "no mode-specific extension inventory authority is configured"
				}
			}
			if reason == "" {
				available = append(available, capability)
			} else {
				report.Unavailable[capability] = reason
			}
		}
		report.Capabilities = available
		out[i] = report
	}
	return out
}

func cloneCapabilityReasons(reasons map[sessionpermission.Capability]string) map[sessionpermission.Capability]string {
	if reasons == nil {
		return make(map[sessionpermission.Capability]string)
	}
	cloned := make(map[sessionpermission.Capability]string, len(reasons))
	for capability, reason := range reasons {
		cloned[capability] = reason
	}
	return cloned
}

// InstallExtension installs an owner-resolved artifact into this session.
// artifactRef is an opaque scenario-to-extension reference, never a path.
func (b *Broker) InstallExtension(ctx context.Context, sessionID, extensionID, artifactRef, expectedDigest string) (ExtensionReceipt, error) {
	if strings.TrimSpace(artifactRef) == "" || strings.TrimSpace(expectedDigest) == "" {
		return ExtensionReceipt{}, errors.New("artifact_ref and expected_digest are required")
	}
	b.mu.RLock()
	controller, verifier := b.extensions, b.extensionVerifier
	b.mu.RUnlock()
	if controller == nil {
		return ExtensionReceipt{}, ErrExtensionOwnerUnavailable
	}
	if verifier == nil {
		return ExtensionReceipt{}, errors.New("verified extension artifact authority unavailable")
	}
	e, err := b.get(sessionID)
	if err != nil {
		return ExtensionReceipt{}, err
	}
	if err := ensureManagedExtensionSession(e); err != nil {
		return ExtensionReceipt{}, err
	}
	if err := b.lockEntry(e); err != nil {
		return ExtensionReceipt{}, err
	}
	defer e.opMu.Unlock()
	if err := e.session.AuthorizeActionRequest(ctx, sessionpermission.ActionRequest{Kind: "extension_artifact_read"}); err != nil {
		return ExtensionReceipt{}, err
	}
	artifact, err := verifier.VerifyBrowserExtensionArtifact(ctx, artifactRef, expectedDigest)
	if err != nil {
		return ExtensionReceipt{}, fmt.Errorf("verify extension artifact before install: %w", err)
	}
	if extensionID != "" && artifact.ExtensionID != extensionID {
		return ExtensionReceipt{}, errors.New("caller extension_id does not match artifact-derived extension id")
	}
	if err := verifyExtensionArtifactProfile(e.profile, artifact, artifact.ExtensionID, artifactRef, expectedDigest); err != nil {
		return ExtensionReceipt{}, err
	}
	if err := e.session.AuthorizeActionRequest(ctx, sessionpermission.ActionRequest{Kind: "extension_install", ExtensionID: artifact.ExtensionID}); err != nil {
		return ExtensionReceipt{}, err
	}
	binding, err := b.extensionSessionBinding(e)
	if err != nil {
		return ExtensionReceipt{}, err
	}
	receipt, err := controller.Install(ctx, ExtensionInstallRequest{Binding: binding, Artifact: artifact})
	if err != nil {
		return ExtensionReceipt{}, err
	}
	if receipt.Operation != "install" || receipt.State != "installed" || !extensionReceiptMatchesBinding(receipt, binding) || !extensionReceiptMatchesArtifact(receipt, sessionID, artifact) {
		return ExtensionReceipt{}, errors.New("extension owner returned a receipt not bound to the authorized session, profile, owner and artifact")
	}
	receipt.OccurredAt = time.Now().UTC()
	if e.installedExtensions == nil {
		e.installedExtensions = make(map[string]ExtensionArtifact)
	}
	e.installedExtensions[artifact.ExtensionID] = artifact
	return receipt, nil
}

// PreflightExtensionLifecycle verifies the explicit authority needed by the
// scenario-to-extension install/reload/validate/cleanup path before install
// can cause a browser effect. It does not stage or install the artifact.
func (b *Broker) PreflightExtensionLifecycle(ctx context.Context, sessionID, artifactRef, expectedDigest, expectedExtensionID string) error {
	if strings.TrimSpace(artifactRef) == "" || strings.TrimSpace(expectedDigest) == "" {
		return errors.New("artifact_ref and expected_digest are required")
	}
	b.mu.RLock()
	controller, verifier := b.extensions, b.extensionVerifier
	b.mu.RUnlock()
	if controller == nil {
		return ErrExtensionOwnerUnavailable
	}
	if verifier == nil {
		return errors.New("verified extension artifact authority unavailable")
	}
	e, err := b.get(sessionID)
	if err != nil {
		return err
	}
	if err := ensureManagedExtensionSession(e); err != nil {
		return err
	}
	if err := b.lockEntry(e); err != nil {
		return err
	}
	defer e.opMu.Unlock()
	if err := e.session.AuthorizeActionRequest(ctx, sessionpermission.ActionRequest{Kind: "extension_artifact_read"}); err != nil {
		return err
	}
	artifact, err := verifier.VerifyBrowserExtensionArtifact(ctx, artifactRef, expectedDigest)
	if err != nil {
		return fmt.Errorf("verify extension artifact before lifecycle preflight: %w", err)
	}
	if err := verifyExtensionArtifactProfile(e.profile, artifact, artifact.ExtensionID, artifactRef, expectedDigest); err != nil {
		return err
	}
	for _, kind := range []string{"extension_install", "extension_load", "extension_inventory"} {
		if err := e.session.AuthorizeActionRequest(ctx, sessionpermission.ActionRequest{Kind: kind, ExtensionID: artifact.ExtensionID}); err != nil {
			return err
		}
	}
	if _, err := b.extensionSessionBinding(e); err != nil {
		return err
	}
	if expectedExtensionID != "" && artifact.ExtensionID != expectedExtensionID {
		return errors.New("caller extension_id does not match artifact-derived extension id")
	}
	return nil
}

func (b *Broker) ReloadExtension(ctx context.Context, sessionID, extensionID string) (ExtensionReceipt, error) {
	return b.extensionOperation(ctx, sessionID, extensionID, "reload")
}

func (b *Broker) LoadExtension(ctx context.Context, sessionID, extensionID string) (ExtensionReceipt, error) {
	return b.extensionOperation(ctx, sessionID, extensionID, "load")
}

func (b *Broker) UnloadExtension(ctx context.Context, sessionID, extensionID string) (ExtensionReceipt, error) {
	return b.extensionOperation(ctx, sessionID, extensionID, "unload")
}

func (b *Broker) ValidateExtension(ctx context.Context, sessionID, extensionID string) (ExtensionValidation, error) {
	b.mu.RLock()
	controller := b.extensions
	b.mu.RUnlock()
	if controller == nil {
		return ExtensionValidation{}, errors.New("BAS session extension controller unavailable")
	}
	e, err := b.get(sessionID)
	if err != nil {
		return ExtensionValidation{}, err
	}
	if err := ensureManagedExtensionSession(e); err != nil {
		return ExtensionValidation{}, err
	}
	if err := b.lockEntry(e); err != nil {
		return ExtensionValidation{}, err
	}
	defer e.opMu.Unlock()
	if err := e.session.AuthorizeActionRequest(ctx, sessionpermission.ActionRequest{Kind: "extension_load", ExtensionID: extensionID}); err != nil {
		return ExtensionValidation{}, err
	}
	artifact, ok := e.installedExtensions[extensionID]
	if !ok {
		return ExtensionValidation{}, errors.New("extension has no broker-verified artifact install receipt")
	}
	binding, err := b.extensionSessionBinding(e)
	if err != nil {
		return ExtensionValidation{}, err
	}
	validation, err := controller.Validate(ctx, ExtensionLifecycleRequest{Binding: binding, Artifact: artifact})
	if err != nil {
		return ExtensionValidation{}, err
	}
	if !validation.Valid || validation.Receipt.Operation != "validate" || validation.Receipt.State != "valid" || !extensionReceiptMatchesBinding(validation.Receipt, binding) || !extensionReceiptMatchesArtifact(validation.Receipt, sessionID, artifact) {
		return ExtensionValidation{}, errors.New("extension validation is not bound to the authorized session, profile, owner and verified artifact")
	}
	return validation, nil
}

func (b *Broker) extensionOperation(ctx context.Context, sessionID, extensionID, operation string) (ExtensionReceipt, error) {
	if strings.TrimSpace(extensionID) == "" {
		return ExtensionReceipt{}, errors.New("extension_id is required")
	}
	b.mu.RLock()
	controller := b.extensions
	b.mu.RUnlock()
	if controller == nil {
		return ExtensionReceipt{}, errors.New("BAS session extension controller unavailable")
	}
	e, err := b.get(sessionID)
	if err != nil {
		return ExtensionReceipt{}, err
	}
	if err := ensureManagedExtensionSession(e); err != nil {
		return ExtensionReceipt{}, err
	}
	if err := b.lockEntry(e); err != nil {
		return ExtensionReceipt{}, err
	}
	defer e.opMu.Unlock()
	actionKind := "extension_load"
	if operation == "unload" {
		actionKind = "extension_unload"
	}
	if err := e.session.AuthorizeActionRequest(ctx, sessionpermission.ActionRequest{Kind: actionKind, ExtensionID: extensionID}); err != nil {
		return ExtensionReceipt{}, err
	}
	artifact, ok := e.installedExtensions[extensionID]
	if !ok {
		return ExtensionReceipt{}, errors.New("extension has no broker-verified artifact install receipt")
	}
	binding, err := b.extensionSessionBinding(e)
	if err != nil {
		return ExtensionReceipt{}, err
	}
	request := ExtensionLifecycleRequest{Binding: binding, Artifact: artifact}
	var receipt ExtensionReceipt
	switch operation {
	case "load":
		receipt, err = controller.Load(ctx, request)
	case "reload":
		receipt, err = controller.Reload(ctx, request)
	case "unload":
		receipt, err = controller.Unload(ctx, request)
	default:
		return ExtensionReceipt{}, errors.New("unsupported extension lifecycle operation")
	}
	if err != nil {
		return ExtensionReceipt{}, err
	}
	if receipt.Operation != operation || receipt.State != extensionReceiptState(operation) || !extensionReceiptMatchesBinding(receipt, binding) || !extensionReceiptMatchesArtifact(receipt, sessionID, artifact) {
		return ExtensionReceipt{}, errors.New("extension owner returned a receipt not bound to the authorized session, profile, owner and artifact")
	}
	receipt.OccurredAt = time.Now().UTC()
	return receipt, nil
}

func extensionReceiptMatchesArtifact(receipt ExtensionReceipt, sessionID string, artifact ExtensionArtifact) bool {
	return receipt.SessionID == sessionID && receipt.ExtensionID == artifact.ExtensionID &&
		receipt.ArtifactRef == artifact.ArtifactRef && receipt.Digest == artifact.Digest &&
		receipt.ManifestSHA256 == artifact.ManifestSHA256 && receipt.Version == artifact.Version
}

func extensionReceiptMatchesBinding(receipt ExtensionReceipt, binding ExtensionSessionBinding) bool {
	return receipt.SessionID == binding.SessionID && receipt.Backend == binding.Backend &&
		receipt.ProfileID == binding.ProfileID && receipt.PermissionRef == binding.PermissionRef &&
		receipt.OwnerID == binding.OwnerID && receipt.LeaseID == binding.LeaseID && receipt.CallerID == binding.CallerID
}

func extensionReceiptState(operation string) string {
	switch operation {
	case "load":
		return "loaded"
	case "reload":
		return "reloaded"
	case "unload":
		return "unloaded"
	default:
		return ""
	}
}

func (b *Broker) extensionSessionBinding(e *entry) (ExtensionSessionBinding, error) {
	if e == nil || e.session == nil || e.closed || e.revoking.Load() {
		return ExtensionSessionBinding{}, ErrSessionNotOwned
	}
	if e.backend != string(sessionpermission.BackendManaged) {
		return ExtensionSessionBinding{}, errors.New("extension controller supports only isolated managed BAS sessions")
	}
	owned, ok := b.manager.Get(e.sessionID)
	if !ok || owned != e.session {
		return ExtensionSessionBinding{}, ErrSessionNotOwned
	}
	ownerID, leaseID := e.session.OwnerIdentity()
	if ownerID == "" || leaseID == "" || ownerID != e.ownerID || leaseID != e.leaseID || e.callerID == "" || e.profile.ID == "" || e.profile.ProfileID == "" || e.permissionRef == "" {
		return ExtensionSessionBinding{}, errors.New("BAS session owner, lease, profile or caller binding is unavailable or changed")
	}
	return ExtensionSessionBinding{
		SessionID: e.sessionID, Backend: e.backend, ProfileID: e.profile.ProfileID,
		PermissionRef: e.permissionRef, OwnerID: ownerID, LeaseID: leaseID, CallerID: e.callerID,
	}, nil
}

func ensureManagedExtensionSession(e *entry) error {
	if e == nil || e.session == nil {
		return errors.New("extension lifecycle is unsupported on a user-attached browser")
	}
	if e.backend != string(sessionpermission.BackendManaged) {
		return errors.New("extension lifecycle requires an isolated managed BAS session")
	}
	return nil
}

// Open creates through the existing BAS session.Manager and only then makes
// the returned lease available through this access surface.
func (b *Broker) Open(ctx context.Context, req OpenRequest) (SessionInfo, error) {
	if req.StartURL == "" && req.Backend != string(sessionpermission.BackendUserAttached) {
		return SessionInfo{}, errors.New("start_url is required so the permission owner can check the target origin")
	}
	b.mu.RLock()
	openContext := b.openContext
	b.mu.RUnlock()
	if openContext == nil {
		return SessionInfo{}, errors.New("trusted Agent Manager open context resolver is required")
	}
	trusted, err := openContext.ResolveBrowserOpen(ctx)
	if err != nil {
		return SessionInfo{}, fmt.Errorf("resolve trusted Agent Manager browser grant: %w", err)
	}
	if trusted.CallerID == "" || trusted.GrantReceipt == "" || trusted.IdentityToken == "" {
		return SessionInfo{}, errors.New("trusted Agent Manager caller, signed identity token, and grant receipt are required")
	}
	if strings.TrimSpace(trusted.PermissionRef) == "" {
		return SessionInfo{}, errors.New("trusted Agent Manager browser profile reference is required")
	}
	if req.PermissionRef != "" && req.PermissionRef != trusted.PermissionRef {
		return SessionInfo{}, errors.New("requested permission profile does not match the trusted Agent Manager run context")
	}
	req.PermissionRef = trusted.PermissionRef
	profile, policy, err := b.profiles.Resolve(ctx, req.PermissionRef)
	if err != nil {
		return SessionInfo{}, fmt.Errorf("resolve browser permission profile: %w", err)
	}
	if profile.ID == "" {
		profile.ID = req.PermissionRef
	}
	if policy == nil {
		return SessionInfo{}, ErrAuthorizerRequired
	}
	if _, err := sessionpermission.New(profile, nil); err != nil {
		return SessionInfo{}, fmt.Errorf("resolved browser permission profile is invalid: %w", err)
	}
	reuseMode := req.ReuseMode
	if reuseMode == "" {
		reuseMode = "fresh"
	}
	if reuseMode != "fresh" && reuseMode != "clean" && reuseMode != "reuse" {
		return SessionInfo{}, fmt.Errorf("unsupported reuse_mode %q", reuseMode)
	}
	execID, workflowID := uuid.New(), uuid.New()
	backend := req.Backend
	if backend == "" {
		backend = "managed"
	}
	if backend != "managed" && backend != string(sessionpermission.BackendInstalled) && backend != string(sessionpermission.BackendUserAttached) && backend != string(sessionpermission.BackendRemote) {
		return SessionInfo{}, fmt.Errorf("unsupported browser backend %q", backend)
	}
	expectedProfileMode, profileModeSupported := profileModeForBackend(backend)
	if !profileModeSupported || profile.ProfileMode != expectedProfileMode {
		return SessionInfo{}, fmt.Errorf("browser backend %q requires an explicitly matching Agent Manager profile mode", backend)
	}
	if backend == string(sessionpermission.BackendInstalled) && reuseMode != "fresh" {
		return SessionInfo{}, errors.New("installed-binary sessions require a fresh isolated browser context")
	}
	if (profile.AppMonitorMetrics || profile.AppMonitorTrace) && backend != string(sessionpermission.BackendManaged) {
		return SessionInfo{}, errors.New("App Monitor capture requires the managed browser backend")
	}
	if req.AudioMode != "" {
		if string(profile.AudioMode) != req.AudioMode || backend != string(sessionpermission.BackendManaged) {
			return SessionInfo{}, errors.New("requested audio mode is not present in the Agent Manager grant for this managed session")
		}
		if req.AudioMode != string(sessionpermission.AudioModeFakeMicrophone) && req.AudioMode != string(sessionpermission.AudioModePipeWire) {
			return SessionInfo{}, fmt.Errorf("unsupported browser audio mode %q", req.AudioMode)
		}
		if req.AudioPlaybackPauseMS < 0 || req.AudioPlaybackPauseMS > 1000 || (req.AudioMode != string(sessionpermission.AudioModePipeWire) && req.AudioPlaybackPauseMS != 0) {
			return SessionInfo{}, errors.New("audio playback pause is out of range or unsupported for the granted mode")
		}
	} else if req.AudioPlaybackPauseMS != 0 {
		return SessionInfo{}, errors.New("audio playback controls require an explicitly granted audio mode")
	}
	required := requiredCapabilities(profile, backend)
	if req.AudioMode != "" {
		required = append(required, sessionpermission.CapabilityAudioCapture)
		if req.AudioMode == string(sessionpermission.AudioModePipeWire) {
			required = append(required, sessionpermission.CapabilityAudioDeviceEvidence)
		}
	}
	negotiated, err := b.Capabilities(ctx, required)
	if err != nil {
		return SessionInfo{}, err
	}
	accepted := false
	for _, candidate := range negotiated {
		if string(candidate.Backend) == backend {
			accepted = candidate.Accepted
			break
		}
	}
	if !accepted {
		return SessionInfo{}, fmt.Errorf("browser backend %q is not capability-qualified for the requested profile", backend)
	}
	if backend == string(sessionpermission.BackendRemote) {
		// No remote session owner is configured in BAS yet. Never represent the
		// local Manager as remote, even if an external qualification report is
		// accidentally present.
		return SessionInfo{}, errors.New("remote browser backend adapter is unavailable; refusing local-session fallback")
	}
	var runtimeEvidence *BrowserRuntimeEvidence
	if backend == string(sessionpermission.BackendManaged) {
		runtimeEvidence, err = verifiedBrowserRuntimeEvidence()
		if err != nil {
			return SessionInfo{}, fmt.Errorf("verify browser runtime provenance: %w", err)
		}
	}
	if backend == string(sessionpermission.BackendUserAttached) {
		if profile.ProfileMode != "user_attached" {
			return SessionInfo{}, errors.New("user_attached backend requires a user_attached permission profile")
		}
		if profile.Effect == sessionpermission.EffectUnrestricted {
			return SessionInfo{}, errors.New("selected-tab bridge does not support unrestricted or raw-protocol access")
		}
		if profile.HistoryRead || profile.HistoryExport || profile.UserStateExport || profile.ExtensionLoad || profile.ExtensionInstall || len(profile.CredentialIDs) > 0 {
			return SessionInfo{}, errors.New("selected-tab bridge does not support browser history, user-state export, credential use, or extension installation/loading")
		}
		code, err := randomBridgeToken()
		if err != nil {
			return SessionInfo{}, err
		}
		nonce, err := randomBridgeToken()
		if err != nil {
			return SessionInfo{}, err
		}
		sessionID := uuid.NewString()
		ownerID := execID.String()
		expires := time.Now().Add(5 * time.Minute)
		e := &entry{
			permissionRef: req.PermissionRef, profile: profile, policy: policy, backend: backend,
			trace:     Trace{ID: uuid.NewString(), SessionID: sessionID, PermissionRef: req.PermissionRef},
			sessionID: sessionID, leaseID: uuid.NewString(), ownerID: ownerID, runID: trusted.RunID,
			callerID: trusted.CallerID, grantReceipt: trusted.GrantReceipt, identityToken: trusted.IdentityToken, openedAt: time.Now(),
			pairingCodeHash: bridgeDigest(code), pairingNonce: nonce, pairingExpires: expires,
		}
		b.mu.Lock()
		b.entries[sessionID] = e
		b.mu.Unlock()
		time.AfterFunc(time.Until(expires), func() {
			b.mu.Lock()
			if b.entries[sessionID] == e && !e.socketConnected && e.attachedTab == nil {
				e.revoking.Store(true)
				e.closed = true
				e.closedAt = time.Now().UTC()
				e.identityToken = ""
				delete(b.entries, sessionID)
			}
			b.mu.Unlock()
		})
		info := b.info(e)
		info.PairingCode, info.PairingExpires, info.Role = code, expires, "pending_user_consent"
		return info, nil
	}
	open := &sessionpermission.OpenRequest{OwnerID: execID.String(), CallerID: trusted.CallerID, Backend: sessionpermission.Backend(backend), ProfileID: req.PermissionRef, InitialURL: req.StartURL, GrantReceipt: trusted.GrantReceipt, TabID: req.TabID, ConsentReceipt: req.ConsentReceipt}
	var audioFixturePath string
	if req.AudioMode != "" {
		b.mu.RLock()
		fixtureResolver := b.audioFixtureResolver
		b.mu.RUnlock()
		if fixtureResolver == nil {
			return SessionInfo{}, errors.New("owner-approved Audio Tools fixture resolver is unavailable")
		}
		audioFixturePath, err = fixtureResolver.ResolveAudioFixture(ctx, audioQualificationFixtureRef)
		if err != nil || strings.TrimSpace(audioFixturePath) == "" || !filepath.IsAbs(audioFixturePath) {
			return SessionInfo{}, errors.New("owner-approved Audio Tools fixture could not be resolved")
		}
	}
	sessionMode := session.ModeExecution
	if profile.Control {
		// Hybrid uses this same session lease for permitted Web Console input;
		// it does not start a second browser or recording owner.
		sessionMode = session.ModeHybrid
	}
	var frameStreaming *session.FrameStreamingConfig
	if profile.View {
		frameStreaming = &session.FrameStreamingConfig{Quality: 55, FPS: 6, Scale: "css"}
	}
	viewportWidth, viewportHeight := req.ViewportWidth, req.ViewportHeight
	if backend == string(sessionpermission.BackendInstalled) && profile.IdentityProfile != nil && profile.IdentityProfile.ViewportWidth > 0 {
		viewportWidth, viewportHeight = profile.IdentityProfile.ViewportWidth, profile.IdentityProfile.ViewportHeight
	}
	browserProfile := browserProfileFromIdentity(profile.IdentityProfile)
	if backend == string(sessionpermission.BackendInstalled) {
		// The installed driver accepts only its existing isolated viewport fields;
		// never send the managed BrowserProfile patch contract to this backend.
		browserProfile = nil
	}
	s, err := b.manager.Create(ctx, session.Spec{
		ExecutionID: execID, WorkflowID: workflowID, Mode: sessionMode,
		BaseURL: req.StartURL, ReuseMode: reuseMode, BrowserBackend: backend,
		ViewportWidth: viewportWidth, ViewportHeight: viewportHeight,
		FrameStreaming: frameStreaming,
		BrowserProfile: browserProfile,
		AudioMode:      sessionpermission.AudioMode(req.AudioMode), AudioPlaybackPauseMS: req.AudioPlaybackPauseMS,
		AudioPlaybackDeferStart: req.AudioMode == string(sessionpermission.AudioModePipeWire),
		AudioDeviceEvidence:     req.AudioMode == string(sessionpermission.AudioModePipeWire),
		FakeMicrophoneWav:       audioFixturePath,
		Labels:                  map[string]string{"source": "agentaccess", "permission_ref": req.PermissionRef, "browser_backend": backend},
		Authorizer:              &sessionAuthorizer{resolver: b.profiles, caller: b.caller, permissionRef: req.PermissionRef, profile: profile, policy: policy, expectedCallerID: trusted.CallerID},
		OpenAuthorizer:          b.openAuthorizer, OpenRequest: open, CallerResolver: b.caller, PermissionProfile: &profile,
	})
	if err != nil {
		return SessionInfo{}, fmt.Errorf("open BAS-managed browser session: %w", err)
	}
	executionOwner, actualLease := s.OwnerIdentity()
	e := &entry{session: s, permissionRef: req.PermissionRef, profile: profile, policy: policy, backend: backend, runtimeEvidence: runtimeEvidence, trace: Trace{ID: uuid.NewString(), SessionID: s.ID(), PermissionRef: req.PermissionRef}, sessionID: s.ID(), leaseID: actualLease, ownerID: executionOwner, callerID: trusted.CallerID, grantReceipt: trusted.GrantReceipt, identityToken: trusted.IdentityToken, tabURL: req.StartURL, openedAt: time.Now()}
	b.addEvidenceBytes(e, evidencePayloadSize(req.StartURL))
	if _, err := s.Navigate(ctx, req.StartURL); err != nil {
		_ = s.Close(ctx)
		return SessionInfo{}, fmt.Errorf("navigate new browser session: %w", err)
	}
	if req.AudioMode == string(sessionpermission.AudioModePipeWire) {
		evidence := s.AudioDeviceEvidence()
		if evidence == nil || !evidence.Enumerated {
			_ = s.Close(ctx)
			return SessionInfo{}, errors.New("PipeWire audio capability did not return owner-measured device evidence")
		}
	}
	instruction := contracts.CompiledInstruction{Index: 0, NodeID: "open-navigation", Action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_NAVIGATE, Params: &actionsv1.ActionDefinition_Navigate{Navigate: &actionsv1.NavigateParams{Url: req.StartURL}}}}
	redactedInstruction, parameters, redactErr := redactInstruction(instruction, 1)
	if redactErr != nil {
		_ = s.Close(ctx)
		return SessionInfo{}, redactErr
	}
	step := contracts.StepOutcome{Success: true, StepType: "navigate"}
	b.record(e, TraceEvent{Kind: "action", OccurredAt: time.Now().UTC(), URL: safeOrigin(req.StartURL), TargetOrigin: safeOrigin(req.StartURL), Instruction: &redactedInstruction, Outcome: &step, Parameters: parameters})
	if err := b.recordSessionEvidence(ctx, e, "opened"); err != nil {
		_ = s.Close(ctx)
		return SessionInfo{}, fmt.Errorf("record browser session open: %w", err)
	}
	if err := b.recordActionEvidence(ctx, e, "navigate", "completed"); err != nil {
		_ = s.Close(ctx)
		_ = b.recordSessionEvidence(ctx, e, "closed")
		return SessionInfo{}, fmt.Errorf("record initial browser navigation: %w", err)
	}
	b.mu.Lock()
	b.entries[s.ID()] = e
	b.mu.Unlock()
	return b.info(e), nil
}

func profileModeForBackend(backend string) (string, bool) {
	switch backend {
	case string(sessionpermission.BackendManaged):
		return "fresh", true
	case string(sessionpermission.BackendInstalled):
		return "isolated", true
	case string(sessionpermission.BackendUserAttached):
		return "user_attached", true
	case string(sessionpermission.BackendRemote):
		return "remote", true
	default:
		return "", false
	}
}

func requiredCapabilities(profile sessionpermission.Profile, backend string) []sessionpermission.Capability {
	required := []sessionpermission.Capability{sessionpermission.CapabilityObserve}
	if profile.Control {
		required = append(required, sessionpermission.CapabilityControl)
	}
	if profile.Approval != sessionpermission.ApprovalNone {
		required = append(required, sessionpermission.CapabilityApprovalReceipt)
	}
	if len(profile.CredentialIDs) > 0 {
		required = append(required, sessionpermission.CapabilityCredentialUse)
	}
	if profile.MaxCredits > 0 {
		required = append(required, sessionpermission.CapabilityCreditBudget)
	}
	if profile.MaxBytes > 0 {
		required = append(required, sessionpermission.CapabilityByteBudget)
	}
	if profile.HistoryRead {
		required = append(required, sessionpermission.CapabilityHistoryRead)
	}
	if profile.HistoryExport {
		required = append(required, sessionpermission.CapabilityHistoryExport)
	}
	if profile.HistoryRetention > 0 {
		required = append(required, sessionpermission.CapabilityHistoryRetention)
	}
	if profile.ExtensionAccess {
		required = append(required, sessionpermission.CapabilityExtensionInventory, sessionpermission.CapabilityExtensionHostPermissions)
	}
	if profile.ExtensionLoad {
		required = append(required, sessionpermission.CapabilityExtensionLoad)
	}
	if profile.ExtensionInstall {
		required = append(required, sessionpermission.CapabilityExtensionInstall)
	}
	if profile.IdentityProfile != nil {
		required = append(required, sessionpermission.CapabilityIdentityCustomization)
	}
	if profile.AppMonitorMetrics {
		required = append(required, sessionpermission.CapabilityAppMonitorMetrics)
	}
	if profile.AppMonitorTrace {
		required = append(required, sessionpermission.CapabilityAppMonitorTrace)
	}
	if profile.Handoff {
		required = append(required, sessionpermission.CapabilityHandoff)
	}
	if backend == string(sessionpermission.BackendUserAttached) {
		required = append(required, sessionpermission.CapabilityPreserveUserState, sessionpermission.CapabilityReconnect)
	}
	return required
}

// Attach borrows only a session already admitted by this process's canonical
// BAS manager. It never starts a second browser lifecycle.
func (b *Broker) Attach(ctx context.Context, req AttachRequest) (SessionInfo, error) {
	if req.SessionID == "" || req.PermissionRef == "" {
		return SessionInfo{}, errors.New("session_id and permission_ref are required")
	}
	b.mu.RLock()
	e := b.entries[req.SessionID]
	b.mu.RUnlock()
	if e == nil {
		return SessionInfo{}, ErrSessionNotOwned
	}
	if err := b.lockEntry(e); err != nil {
		return SessionInfo{}, err
	}
	defer e.opMu.Unlock()
	if e.permissionRef != req.PermissionRef {
		return SessionInfo{}, errors.New("permission profile does not match the admitted session")
	}
	if e.session == nil {
		if e.attachedTab == nil {
			return SessionInfo{}, errors.New("selected tab is awaiting explicit user consent and pairing")
		}
		if err := b.authorize(ctx, e, sessionpermission.ActionRequest{Kind: "attach", CurrentURL: e.tabURL, ApprovalReceipt: req.ApprovalReceipt}); err != nil {
			return SessionInfo{}, err
		}
		return b.info(e), nil
	}
	if err := e.session.AuthorizeAction(ctx, "observe", "", req.ApprovalReceipt); err != nil {
		return SessionInfo{}, err
	}
	owned, ok := b.manager.Get(req.SessionID)
	if !ok || owned != e.session {
		return SessionInfo{}, ErrSessionNotOwned
	}
	return b.info(e), nil
}

func (b *Broker) Observe(ctx context.Context, sessionID string, req ObserveRequest) (Observation, error) {
	if req.TestID != "" && !validScreenshotTestID(req.TestID) {
		return Observation{}, errors.New("screenshot test_id must be a bounded stable identifier")
	}
	if req.TestID == "" && (req.WaitUntil != "" || req.SettleMs != 0) || (req.WaitUntil != "" && req.WaitUntil != "networkidle") || req.SettleMs < 0 || req.SettleMs > 5000 {
		return Observation{}, errors.New("screenshot wait options require a target and bounded networkidle/settle values")
	}
	e, err := b.get(sessionID)
	if err != nil {
		return Observation{}, err
	}
	if err := b.lockEntry(e); err != nil {
		return Observation{}, err
	}
	defer e.opMu.Unlock()
	if e.session == nil {
		if req.TestID != "" {
			return Observation{}, errors.New("target screenshots require a BAS managed session")
		}
		if e.attachedTab == nil {
			return Observation{}, errors.New("selected tab is not connected")
		}
		currentURL, err := e.attachedTab.CurrentURL(ctx)
		if err != nil {
			return Observation{}, err
		}
		if !e.profile.AllowsURL(currentURL) {
			return Observation{}, sessionpermission.ErrDenied
		}
		if err := b.authorize(ctx, e, sessionpermission.ActionRequest{Kind: "observe", CurrentURL: currentURL, TargetURL: currentURL}); err != nil {
			return Observation{}, err
		}
		if err := ensureAttachedTabActive(e); err != nil {
			return Observation{}, err
		}
		tab, err := e.attachedTab.Observe(ctx)
		if err != nil {
			return Observation{}, err
		}
		if !e.profile.AllowsURL(tab.URL) {
			return Observation{}, sessionpermission.ErrDenied
		}
		if err := b.verifySelectedTabObservationResult(ctx, e, tab.URL); err != nil {
			return Observation{}, err
		}
		e.tabURL, e.tabTitle = tab.URL, tab.Title
		b.record(e, TraceEvent{Kind: "observation", OccurredAt: time.Now().UTC(), URL: safeOrigin(tab.URL), Title: tab.Title})
		observation := Observation{SessionID: sessionID, ObservedAt: time.Now().UTC(), URL: tab.URL, Title: tab.Title, Text: tab.Text}
		b.addEvidenceBytes(e, evidencePayloadSize(req)+evidencePayloadSize(observation))
		if evidenceErr := b.recordActionEvidence(ctx, e, "observe", "completed"); evidenceErr != nil {
			observation.EvidenceWarning = evidenceErr.Error()
		}
		return observation, nil
	}
	state, screenshot, err := e.session.CaptureObservationWithScreenshot(ctx, req.PageID, driver.CaptureScreenshotRequest{Format: "png", TestID: req.TestID, WaitUntil: req.WaitUntil, SettleMs: req.SettleMs})
	if err != nil {
		return Observation{}, err
	}
	obs := Observation{SessionID: sessionID, ObservedAt: time.Now().UTC(), URL: state.URL, Title: state.Title, Screenshot: screenshot}
	e.tabURL = state.URL
	b.addEvidenceBytes(e, evidencePayloadSize(req)+evidencePayloadSize(obs))
	b.record(e, TraceEvent{Kind: "observation", OccurredAt: obs.ObservedAt, URL: safeOrigin(obs.URL)})
	if evidenceErr := b.recordActionEvidence(ctx, e, "observe", "completed"); evidenceErr != nil {
		obs.EvidenceWarning = evidenceErr.Error()
	}
	return obs, nil
}

func (b *Broker) verifySelectedTabObservationResult(ctx context.Context, e *entry, observedURL string) error {
	callerID, err := b.caller.BrowserCallerID(ctx)
	if err != nil || callerID == "" || callerID != e.callerID {
		return ErrSessionNotOwned
	}
	profile, policy, err := b.profiles.Resolve(ctx, e.permissionRef)
	if err != nil || policy == nil {
		return fmt.Errorf("resolve selected-tab result authority: %w", sessionpermission.ErrDenied)
	}
	if profile.ID == "" {
		profile.ID = e.permissionRef
	}
	if !reflect.DeepEqual(profile, e.profile) {
		return errors.New("browser permission profile changed during selected-tab observation")
	}
	verifier, ok := policy.(sessionpermission.SelectedTabResultVerifier)
	if !ok {
		return errors.New("selected-tab result consent verification is unavailable")
	}
	return verifier.VerifySelectedTabResult(ctx, sessionpermission.ActionRequest{
		SessionID: e.sessionID, LeaseID: e.leaseID, OwnerID: e.ownerID, CallerID: callerID,
		Kind: "observe", CurrentURL: observedURL, TargetURL: observedURL,
		TabID: e.tabID, ConsentReceipt: e.consentReceipt,
	})
}

func (b *Broker) SelectedTabExtensions(ctx context.Context, sessionID string) (ExtensionInventory, error) {
	e, err := b.get(sessionID)
	if err != nil {
		return ExtensionInventory{}, err
	}
	if err := b.lockEntry(e); err != nil {
		return ExtensionInventory{}, err
	}
	defer e.opMu.Unlock()
	if !e.profile.ExtensionAccess || len(e.profile.ExtensionPermissions) == 0 {
		return ExtensionInventory{}, sessionpermission.ErrDenied
	}
	var items []sessionpermission.ExtensionInventoryEntry
	observedAt := time.Now().UTC()
	if e.session != nil {
		if err := ensureManagedExtensionSession(e); err != nil {
			return ExtensionInventory{}, err
		}
		b.mu.RLock()
		controller := b.extensions
		b.mu.RUnlock()
		if controller == nil {
			return ExtensionInventory{}, errors.New("managed extension inventory controller unavailable")
		}
		if err := e.session.AuthorizeActionRequest(ctx, sessionpermission.ActionRequest{Kind: "extension_inventory"}); err != nil {
			return ExtensionInventory{}, err
		}
		binding, bindingErr := b.extensionSessionBinding(e)
		if bindingErr != nil {
			return ExtensionInventory{}, bindingErr
		}
		inventory, inventoryErr := controller.Inventory(ctx, binding)
		if inventoryErr != nil {
			return ExtensionInventory{}, inventoryErr
		}
		if inventory.Binding != binding || !extensionInventoryTimestampFresh(inventory.ObservedAt) {
			return ExtensionInventory{}, errors.New("extension inventory owner returned a mismatched binding or stale observation timestamp")
		}
		items = inventory.Extensions
		observedAt = inventory.ObservedAt
	} else {
		if e.attachedTab == nil {
			return ExtensionInventory{}, errors.New("selected-tab extension inventory is unavailable")
		}
		if err := b.authorize(ctx, e, sessionpermission.ActionRequest{Kind: "extension_inventory", CurrentURL: e.tabURL}); err != nil {
			return ExtensionInventory{}, err
		}
		if err := ensureAttachedTabActive(e); err != nil {
			return ExtensionInventory{}, err
		}
		snapshot, inventoryErr := e.attachedTab.ExtensionInventory(ctx)
		if inventoryErr != nil {
			return ExtensionInventory{}, inventoryErr
		}
		if !extensionInventoryTimestampFresh(snapshot.ObservedAt) {
			return ExtensionInventory{}, errors.New("selected-tab extension inventory owner omitted or returned a stale observation time")
		}
		// Inventory is a read, but it still carries sensitive host/API grant
		// information. Recheck the live bridge lease and trusted policy after
		// the owner response so mid-read revoke or scope loss cannot release it.
		if err := ensureAttachedTabActive(e); err != nil {
			return ExtensionInventory{}, err
		}
		if err := b.authorize(ctx, e, sessionpermission.ActionRequest{Kind: "extension_inventory", CurrentURL: e.tabURL}); err != nil {
			return ExtensionInventory{}, err
		}
		items = snapshot.Extensions
		observedAt = snapshot.ObservedAt
	}
	filtered, err := filterExtensionInventory(e.profile, items)
	if err != nil {
		return ExtensionInventory{}, err
	}
	if e.session != nil {
		if err := ensureManagedExtensionSession(e); err != nil {
			return ExtensionInventory{}, err
		}
		if err := e.session.AuthorizeActionRequest(ctx, sessionpermission.ActionRequest{Kind: "extension_inventory"}); err != nil {
			return ExtensionInventory{}, err
		}
	} else {
		if err := ensureAttachedTabActive(e); err != nil {
			return ExtensionInventory{}, err
		}
		if err := b.authorize(ctx, e, sessionpermission.ActionRequest{Kind: "extension_inventory", CurrentURL: e.tabURL}); err != nil {
			return ExtensionInventory{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return ExtensionInventory{}, err
	}
	for index := range filtered {
		if artifact, ok := e.installedExtensions[filtered[index].ID]; ok {
			if !inventoryPermissionsWithinArtifact(filtered[index], artifact) {
				return ExtensionInventory{}, fmt.Errorf("live extension permissions exceed artifact manifest for %s", filtered[index].ID)
			}
			filtered[index].DeclaredPermissions = append([]string(nil), artifact.Permissions...)
			filtered[index].DeclaredHostPermissions = append([]string(nil), artifact.HostPermissions...)
			filtered[index].ArtifactSHA256 = artifact.Digest
			filtered[index].ManifestSHA256 = artifact.ManifestSHA256
			filtered[index].ArtifactVerified = true
		} else {
			filtered[index].ArtifactSHA256 = ""
			filtered[index].ManifestSHA256 = ""
			filtered[index].ArtifactVerified = false
			filtered[index].DeclaredPermissions = nil
			filtered[index].DeclaredHostPermissions = nil
		}
	}
	binding, err := b.extensionSessionBinding(e)
	if err != nil {
		if e.session != nil {
			return ExtensionInventory{}, err
		}
		binding = ExtensionSessionBinding{SessionID: sessionID, Backend: e.backend, ProfileID: e.profile.ProfileID, PermissionRef: e.permissionRef, OwnerID: e.ownerID, LeaseID: e.leaseID, CallerID: e.callerID}
	}
	return ExtensionInventory{SessionID: sessionID, Binding: binding, ObservedAt: observedAt, Extensions: filtered}, nil
}

const extensionInventoryMaxAge = 30 * time.Second

func extensionInventoryTimestampFresh(observedAt time.Time) bool {
	if observedAt.IsZero() {
		return false
	}
	// Do not reject browsers on another host for clock skew. The WebSocket
	// request ID is the freshness correlation; this timestamp is display data.
	return time.Since(observedAt) <= extensionInventoryMaxAge
}

func filterExtensionInventory(profile sessionpermission.Profile, items []sessionpermission.ExtensionInventoryEntry) ([]sessionpermission.ExtensionInventoryEntry, error) {
	if !profile.ExtensionAccess || len(profile.ExtensionPermissions) == 0 {
		return nil, errors.New("extension inventory requires an explicit extension id allowlist")
	}
	filtered := make([]sessionpermission.ExtensionInventoryEntry, 0, len(items))
	for _, item := range items {
		allowedPermissions, allowed := profile.ExtensionPermissions[item.ID]
		if !allowed {
			continue
		}
		if err := validateExtensionPermissionEvidence(item); err != nil {
			return nil, fmt.Errorf("%w: extension %q permission evidence is invalid: %v", sessionpermission.ErrDenied, item.ID, err)
		}
		for _, permission := range item.Permissions {
			if !containsString(allowedPermissions, permission) {
				return nil, fmt.Errorf("%w: observed extension %q has unapproved permission %q", sessionpermission.ErrDenied, item.ID, permission)
			}
		}
		for _, permission := range item.HostPermissions {
			if !containsString(allowedPermissions, permission) {
				return nil, fmt.Errorf("%w: observed extension %q has unapproved host permission %q", sessionpermission.ErrDenied, item.ID, permission)
			}
		}
		filtered = append(filtered, item)
	}
	return filtered, nil
}

func validateExtensionPermissionEvidence(item sessionpermission.ExtensionInventoryEntry) error {
	evidence := item.PermissionEvidence
	if evidence.PerTabSiteAccess != sessionpermission.ExtensionPerTabSiteAccessNotMeasured {
		return errors.New("extension permission snapshot must state that per-tab site access was not measured")
	}
	switch evidence.Status {
	case sessionpermission.ExtensionPermissionEvidenceEffective:
		if evidence.Scope != sessionpermission.ExtensionPermissionScopeGrantSetOnly {
			return errors.New("effective extension permission snapshot must describe a granted permission set only")
		}
		switch evidence.Source {
		case sessionpermission.ExtensionPermissionSourceChromeManagement:
			if !item.Enabled {
				return errors.New("disabled extension cannot report effective permissions")
			}
		case sessionpermission.ExtensionPermissionSourceChromeGetAll:
			if !item.Enabled || !item.ArtifactVerified || !isSHA256Hex(item.ArtifactSHA256) || !isSHA256Hex(item.ManifestSHA256) {
				return errors.New("runtime permission probe is not bound to an enabled verified artifact")
			}
			if evidence.ObservedAt.IsZero() || !extensionInventoryTimestampFresh(evidence.ObservedAt) {
				return errors.New("runtime permission probe omitted a fresh browser observation timestamp")
			}
		default:
			return errors.New("unsupported effective-permission source")
		}
	case sessionpermission.ExtensionPermissionEvidenceUnavailable:
		if evidence.Scope != sessionpermission.ExtensionPermissionScopeUnavailable {
			return errors.New("unavailable extension permission snapshot must not claim a grant set")
		}
		if evidence.Source != sessionpermission.ExtensionPermissionSourceChromiumCDP && evidence.Source != sessionpermission.ExtensionPermissionSourceChromeManagement && evidence.Source != sessionpermission.ExtensionPermissionSourceChromeGetAll {
			return errors.New("unsupported unavailable-permission source")
		}
		if evidence.Source == sessionpermission.ExtensionPermissionSourceChromeManagement && item.Enabled {
			return errors.New("enabled extension cannot omit its management permission snapshot")
		}
		if evidence.Source == sessionpermission.ExtensionPermissionSourceChromeGetAll && (!item.ArtifactVerified || !isSHA256Hex(item.ArtifactSHA256) || !isSHA256Hex(item.ManifestSHA256)) {
			return errors.New("unavailable runtime permission probe is not bound to a verified artifact")
		}
		if len(item.Permissions) != 0 || len(item.HostPermissions) != 0 {
			return errors.New("unavailable permission evidence included permission values")
		}
	default:
		return errors.New("missing or unknown permission-evidence status")
	}
	return nil
}

func inventoryPermissionsWithinArtifact(item sessionpermission.ExtensionInventoryEntry, artifact ExtensionArtifact) bool {
	for _, permission := range item.Permissions {
		if !containsString(artifact.Permissions, permission) {
			return false
		}
	}
	for _, permission := range item.HostPermissions {
		if !containsString(artifact.HostPermissions, permission) {
			return false
		}
	}
	return item.Version == "" || item.Version == artifact.Version
}

func ensureAttachedTabActive(e *entry) error {
	if e == nil || e.revoking.Load() || e.attachedTab == nil || e.attachedTab.Revoked() {
		return ErrSessionNotOwned
	}
	return nil
}

func verifyExtensionArtifactProfile(profile sessionpermission.Profile, artifact ExtensionArtifact, requestedID, expectedRef, expectedDigest string) error {
	if artifact.ArtifactRef != expectedRef || artifact.ExtensionID != requestedID || artifact.Digest != expectedDigest ||
		!chromeExtensionIDPattern.MatchString(artifact.ExtensionID) || !isSHA256Hex(artifact.Digest) || !isSHA256Hex(artifact.ManifestSHA256) || strings.TrimSpace(artifact.Version) == "" {
		return errors.New("verified extension artifact reference, ID, digest, manifest or version does not match the requested install")
	}
	allowed, ok := profile.ExtensionPermissions[requestedID]
	if !profile.ExtensionAccess || !ok {
		return sessionpermission.ErrDenied
	}
	for _, permission := range append(append([]string(nil), artifact.Permissions...), artifact.HostPermissions...) {
		if !containsString(allowed, permission) {
			return fmt.Errorf("%w: extension artifact requests ungranted permission %q", sessionpermission.ErrDenied, permission)
		}
	}
	return nil
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

// Handoff permanently downgrades a selected-tab lease to observe-only. A new
// Open and fresh person consent are required to restore agent control.
func (b *Broker) Handoff(ctx context.Context, sessionID string) (SessionInfo, error) {
	e, err := b.get(sessionID)
	if err != nil {
		return SessionInfo{}, err
	}
	if err := b.lockEntry(e); err != nil {
		return SessionInfo{}, err
	}
	defer e.opMu.Unlock()
	if e.attachedTab == nil {
		return SessionInfo{}, errors.New("handoff is only supported for an attached selected tab")
	}
	if !e.profile.Handoff {
		return SessionInfo{}, sessionpermission.ErrDenied
	}
	if err := b.authorize(ctx, e, sessionpermission.ActionRequest{Kind: "handoff", CurrentURL: e.tabURL}); err != nil {
		return SessionInfo{}, err
	}
	if err := ensureAttachedTabActive(e); err != nil {
		return SessionInfo{}, err
	}
	if err := e.attachedTab.Handoff(ctx); err != nil {
		return SessionInfo{}, err
	}
	e.handedOff = true
	return b.info(e), nil
}

// AudioPlayback controls the deterministic fixture source attached to the
// existing BAS lease. Each command is authorized against the current Agent
// Manager grant and can only target this broker-owned session.
func (b *Broker) AudioPlayback(ctx context.Context, sessionID string, command session.AudioPlaybackCommand) error {
	e, err := b.get(sessionID)
	if err != nil {
		return err
	}
	if err := b.lockEntry(e); err != nil {
		return err
	}
	defer e.opMu.Unlock()
	if e.session == nil || e.backend != string(sessionpermission.BackendManaged) || e.profile.AudioMode != sessionpermission.AudioModePipeWire {
		return errors.New("audio playback controls require a PipeWire-qualified managed BAS session")
	}
	kind := ""
	switch command {
	case session.AudioPlaybackRestart:
		kind = "audio_playback_restart"
	case session.AudioPlaybackStop:
		kind = "audio_playback_stop"
	default:
		return errors.New("unsupported audio playback command")
	}
	if err := b.authorize(ctx, e, sessionpermission.ActionRequest{Kind: kind, CurrentURL: e.tabURL}); err != nil {
		return err
	}
	callerID, err := b.caller.BrowserCallerID(ctx)
	if err != nil || callerID != e.callerID {
		return ErrSessionNotOwned
	}
	ownerID, leaseID := e.session.OwnerIdentity()
	if err := e.session.AudioPlayback(ctx, ownerID, leaseID, e.profile.ProfileID, callerID, command); err != nil {
		return err
	}
	if recordErr := b.recordActionEvidence(ctx, e, kind, "completed"); recordErr != nil {
		return fmt.Errorf("audio command completed but owner evidence could not be recorded: %w", recordErr)
	}
	return nil
}

// AudioDiagnostics is the only audio-specific page observation supported by
// Agent Access. It delegates to a fixed Audio Tools surface script in the
// existing driver session and keeps transcript data out of traces/evidence.
func (b *Broker) AudioDiagnostics(ctx context.Context, sessionID, surface string) (session.AudioDiagnostics, error) {
	e, err := b.get(sessionID)
	if err != nil {
		return session.AudioDiagnostics{}, err
	}
	if err := b.lockEntry(e); err != nil {
		return session.AudioDiagnostics{}, err
	}
	defer e.opMu.Unlock()
	if e.session == nil || e.backend != string(sessionpermission.BackendManaged) || e.profile.AudioMode == sessionpermission.AudioModeNone {
		return session.AudioDiagnostics{}, errors.New("audio diagnostics require an audio-capable managed BAS session")
	}
	if surface != "audio-tools" && surface != "swarm-manager" {
		return session.AudioDiagnostics{}, errors.New("unsupported Audio Tools diagnostic surface")
	}
	if err := b.authorize(ctx, e, sessionpermission.ActionRequest{Kind: "audio_observe", CurrentURL: e.tabURL}); err != nil {
		return session.AudioDiagnostics{}, err
	}
	callerID, err := b.caller.BrowserCallerID(ctx)
	if err != nil || callerID != e.callerID {
		return session.AudioDiagnostics{}, ErrSessionNotOwned
	}
	ownerID, leaseID := e.session.OwnerIdentity()
	result, err := e.session.ReadAudioDiagnostics(ctx, ownerID, leaseID, e.profile.ProfileID, callerID, surface)
	if err != nil {
		return session.AudioDiagnostics{}, err
	}
	b.addEvidenceBytes(e, evidencePayloadSize(result))
	b.record(e, TraceEvent{Kind: "audio_observation", OccurredAt: time.Now().UTC(), URL: safeOrigin(e.tabURL)})
	if err := b.recordActionEvidence(ctx, e, "audio_observe", "completed"); err != nil {
		return session.AudioDiagnostics{}, fmt.Errorf("audio observation completed but owner evidence could not be recorded: %w", err)
	}
	return result, nil
}

func (b *Broker) Act(ctx context.Context, sessionID string, req ActRequest) (ActionResult, error) {
	e, err := b.get(sessionID)
	if err != nil {
		return ActionResult{}, err
	}
	if err := b.lockEntry(e); err != nil {
		return ActionResult{}, err
	}
	defer e.opMu.Unlock()
	if e.session == nil {
		if e.attachedTab == nil {
			return ActionResult{}, errors.New("selected tab is not connected")
		}
		if e.handedOff {
			return ActionResult{}, sessionpermission.ErrDenied
		}
		action, targetURL, err := bridgeAction(req.Instruction)
		if err != nil {
			return ActionResult{}, err
		}
		if targetURL != "" && req.TargetURL != "" && targetURL != req.TargetURL {
			return ActionResult{}, errors.New("target_url must match the typed navigation instruction")
		}
		if targetURL != "" && !e.profile.AllowsURL(targetURL) {
			return ActionResult{}, sessionpermission.ErrDenied
		}
		currentURL, err := e.attachedTab.CurrentURL(ctx)
		if err != nil {
			return ActionResult{}, err
		}
		if !e.profile.AllowsURL(currentURL) {
			return ActionResult{}, sessionpermission.ErrDenied
		}
		e.tabURL = currentURL
		kind := "act"
		if action.Type == "navigate" {
			kind = "navigate"
		}
		if err := b.authorize(ctx, e, sessionpermission.ActionRequest{Kind: kind, CurrentURL: e.tabURL, TargetURL: targetURL, ApprovalReceipt: req.ApprovalReceipt}); err != nil {
			return ActionResult{}, err
		}
		if err := ensureAttachedTabActive(e); err != nil {
			return ActionResult{}, err
		}
		if err := e.attachedTab.Act(ctx, action); err != nil {
			return ActionResult{}, err
		}
		if action.Type == "navigate" {
			e.tabURL = targetURL
		}
		result := ActionResult{SessionID: sessionID, Outcome: contracts.StepOutcome{Success: true, StepType: action.Type}}
		b.addEvidenceBytes(e, evidencePayloadSize(req)+evidencePayloadSize(result))
		if evidenceErr := b.recordActionEvidence(ctx, e, actionEvidenceCategory(action.Type), "completed"); evidenceErr != nil {
			result.EvidenceWarning = evidenceErr.Error()
		}
		return result, nil
	}
	if req.Instruction.Action == nil {
		return ActionResult{}, errors.New("instruction.action is required")
	}
	if req.ApprovalReceipt != "" {
		if req.Instruction.Metadata == nil {
			req.Instruction.Metadata = make(map[string]string)
		}
		if current := req.Instruction.Metadata["authorization_receipt"]; current != "" && current != req.ApprovalReceipt {
			return ActionResult{}, errors.New("approval_receipt must match the instruction authorization receipt")
		}
		req.Instruction.Metadata["authorization_receipt"] = req.ApprovalReceipt
	}
	if nav := req.Instruction.Action.GetNavigate(); nav != nil && req.TargetURL != "" && req.TargetURL != nav.GetUrl() {
		return ActionResult{}, errors.New("target_url must match the typed navigation instruction")
	}
	outcome, err := e.session.Run(ctx, req.Instruction)
	if err != nil {
		return ActionResult{}, err
	}
	traceSnapshot := b.traceSnapshot(e)
	copyInstruction, parameters, redactionErr := redactInstruction(req.Instruction, len(traceSnapshot.Events)+1)
	var traceInstruction *contracts.CompiledInstruction
	if redactionErr == nil {
		traceInstruction = &copyInstruction
	} else {
		parameters = nil
	}
	traceOutcome := contracts.StepOutcome{SchemaVersion: outcome.SchemaVersion, PayloadVersion: outcome.PayloadVersion, StepIndex: outcome.StepIndex, Attempt: outcome.Attempt, NodeID: outcome.NodeID, StepType: outcome.StepType, Success: outcome.Success, DurationMs: outcome.DurationMs}
	result := ActionResult{SessionID: sessionID, Outcome: outcome}
	b.addEvidenceBytes(e, evidencePayloadSize(req)+evidencePayloadSize(result))
	traceTargetOrigin := ""
	if navigate := req.Instruction.Action.GetNavigate(); navigate != nil {
		traceTargetOrigin = safeOrigin(navigate.GetUrl())
	}
	b.record(e, TraceEvent{Kind: "action", OccurredAt: time.Now().UTC(), URL: safeOrigin(e.tabURL), TargetOrigin: traceTargetOrigin, Instruction: traceInstruction, Outcome: &traceOutcome, Parameters: parameters, UnpromotableReason: func() string {
		if redactionErr != nil {
			return redactionErr.Error()
		}
		if instructionUsesEphemeralAccessibilityRef(req.Instruction) {
			return "ephemeral accessibility references cannot be promoted to a deterministic workflow selector"
		}
		return ""
	}()})
	if outcome.Success {
		if navigate := req.Instruction.Action.GetNavigate(); navigate != nil {
			e.tabURL = navigate.GetUrl()
		}
	}
	if evidenceErr := b.recordActionEvidence(ctx, e, instructionEvidenceCategory(req.Instruction.Action.Type), "completed"); evidenceErr != nil {
		result.EvidenceWarning = evidenceErr.Error()
	}
	return result, nil
}

func instructionUsesEphemeralAccessibilityRef(instruction contracts.CompiledInstruction) bool {
	if instruction.Action == nil {
		return false
	}
	selector := ""
	if click := instruction.Action.GetClick(); click != nil {
		selector = click.GetSelector()
	} else if input := instruction.Action.GetInput(); input != nil {
		selector = input.GetSelector()
	}
	return strings.HasPrefix(selector, "@bas-ref:")
}

func (b *Broker) Close(ctx context.Context, sessionID string) error {
	b.mu.RLock()
	e := b.entries[sessionID]
	b.mu.RUnlock()
	if e == nil {
		return ErrSessionNotOwned
	}
	callerID, err := b.caller.BrowserCallerID(ctx)
	b.mu.RLock()
	callerOwnsEntry := callerID != "" && (callerID == e.callerID || callerID == e.leaseCaller)
	b.mu.RUnlock()
	if err != nil || !callerOwnsEntry {
		return errors.New("only the trusted session owner may close this browser session")
	}
	// A caller that disconnects after issuing close must not leave its session
	// live. Identity values survive WithoutCancel, and the detached cleanup is
	// bounded so an abandoned HTTP request cannot hold the Broker indefinitely.
	closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if e.backend == string(sessionpermission.BackendUserAttached) {
		if err := b.authorize(closeCtx, e, sessionpermission.ActionRequest{Kind: "close"}); err != nil {
			return err
		}
		b.mu.Lock()
		if b.entries[sessionID] != e || e.closed || e.revoking.Load() {
			b.mu.Unlock()
			return ErrSessionNotOwned
		}
		e.revoking.Store(true)
		e.closed = true
		e.closedAt = time.Now().UTC()
		tab, conn := e.attachedTab, e.attachedSocket
		// Keep the closed entry in the broker map until its short-lived proof
		// expires. Agent Manager's verification callback resolves only through
		// this map, including for user-attached sessions.
		b.mu.Unlock()
		// Revoke outside the operation lock so a bridge command waiting for its
		// response is interrupted immediately instead of being allowed to finish.
		if tab != nil {
			_ = tab.Close()
		} else if conn != nil {
			_ = conn.Close()
		}
		var consentRevokeErr error
		if err := b.revokeTabConsent(e); err != nil {
			consentRevokeErr = fmt.Errorf("browser session closed locally but consent owner revocation failed: %w", err)
		}
		e.opMu.Lock()
		e.attachedTab, e.attachedSocket = nil, nil
		if err := b.issueClosedEvidenceProof(e, e.closedAt); err != nil {
			e.opMu.Unlock()
			return fmt.Errorf("browser session closed but broker evidence proof could not be prepared: %w", err)
		}
		proof := e.evidenceProof
		e.opMu.Unlock()
		return errors.Join(consentRevokeErr, b.finishCloseEvidence(closeCtx, sessionID, e, proof))
	}
	e.opMu.Lock()
	b.mu.RLock()
	owned := b.entries[sessionID] == e
	b.mu.RUnlock()
	if !owned {
		e.opMu.Unlock()
		return ErrSessionNotOwned
	}
	if !e.closed {
		if e.session != nil {
			managed, ok := b.manager.Get(sessionID)
			if !ok || managed != e.session {
				e.opMu.Unlock()
				return ErrSessionNotOwned
			}
			if err := e.session.Close(closeCtx); err != nil {
				e.opMu.Unlock()
				return err
			}
		} else if err := b.authorize(closeCtx, e, sessionpermission.ActionRequest{Kind: "close", CurrentURL: ""}); err != nil {
			e.opMu.Unlock()
			return err
		}
		e.closedAt = time.Now().UTC()
		b.mu.Lock()
		e.closed = true
		b.mu.Unlock()
		if e.attachedTab != nil {
			_ = e.attachedTab.Close()
		} else if e.attachedSocket != nil {
			_ = e.attachedSocket.Close()
		}
		if err := b.issueClosedEvidenceProof(e, e.closedAt); err != nil {
			e.opMu.Unlock()
			return fmt.Errorf("browser session closed but broker evidence proof could not be prepared: %w", err)
		}
	}
	var extensionCleanupErr error
	b.mu.RLock()
	extensions := b.extensions
	b.mu.RUnlock()
	if finalizer, ok := extensions.(ExtensionSessionFinalizer); ok {
		extensionCleanupErr = finalizer.FinalizeClosedSession(closeCtx, sessionID)
	}
	proof := e.evidenceProof
	e.opMu.Unlock()
	return errors.Join(extensionCleanupErr, b.finishCloseEvidence(closeCtx, sessionID, e, proof))
}

// SettlementReceipt returns only the LPBS result captured during this exact
// broker session's authenticated Agent Manager close callback. It never looks
// up a caller supplied reservation or account identifier.
func (b *Broker) SettlementReceipt(ctx context.Context, sessionID string) (BrowserSettlementResult, error) {
	b.mu.RLock()
	e := b.entries[sessionID]
	b.mu.RUnlock()
	if e == nil {
		return BrowserSettlementResult{}, ErrSessionNotOwned
	}
	callerID, err := b.caller.BrowserCallerID(ctx)
	credentials, credentialErr := credentialsFromContext(ctx)
	if err != nil || credentialErr != nil || callerID == "" || callerID != e.callerID ||
		credentials.identityToken != e.identityToken || credentials.grantReceipt != e.grantReceipt {
		return BrowserSettlementResult{}, errors.New("only the authorized session owner may read its settlement receipt")
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.entries[sessionID] != e || !e.closed {
		return BrowserSettlementResult{}, errors.New("browser session must be closed before its settlement receipt is available")
	}
	result := e.settlementResult
	if result.Status == "" {
		result.Status = "pending"
	}
	result.Receipt = append([]byte(nil), result.Receipt...)
	return result, nil
}

func (b *Broker) revokeTabConsent(e *entry) error {
	if e == nil || strings.TrimSpace(e.consentReceipt) == "" {
		return nil
	}
	e.consentRevokeMu.Lock()
	defer e.consentRevokeMu.Unlock()
	if e.consentRevoked {
		return nil
	}
	b.mu.RLock()
	issuer := b.tabConsentIssuer
	b.mu.RUnlock()
	revoker, ok := issuer.(TabConsentRevoker)
	if !ok || strings.TrimSpace(e.runID) == "" {
		return errors.New("selected-tab consent owner cannot revoke this lease")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := revoker.RevokeBrowserTabConsent(ctx, e.consentReceipt, e.runID, e.sessionID); err != nil {
		return err
	}
	e.consentRevoked = true
	return nil
}

func (b *Broker) finishCloseEvidence(ctx context.Context, sessionID string, e *entry, proof *closedBrowserEvidenceProof) error {
	b.RevokeStreamCookie(sessionID)

	// The AM append synchronously calls back to this broker for verification,
	// so no entry lock may be held across either recorder call.
	evidenceErr := b.recordSessionEvidence(ctx, e, "closed")
	var proofErr error
	if proof != nil {
		b.mu.RLock()
		evidence := b.evidence
		b.mu.RUnlock()
		if reporter, ok := evidence.(TrustedBrowserEvidenceReporter); ok {
			var result BrowserSettlementResult
			result, proofErr = reporter.RecordClosedBrowserSession(ctx, e.callerID, e.sessionID, e.grantReceipt, proof.Reference)
			if proofErr == nil {
				b.mu.Lock()
				if b.entries[sessionID] == e && e.closed && e.evidenceProof == proof {
					e.settlementResult = result
				}
				b.mu.Unlock()
			}
		}
	}
	if errors.Is(proofErr, ErrTrustedEvidenceUnavailable) {
		// A missing operator-issued BAS service credential must not turn a
		// successful browser close into a failed lifecycle call. The proof stays
		// bounded to this finalized entry and is never appended without auth.
		proofErr = nil
	}
	if evidenceErr != nil {
		return fmt.Errorf("browser session closed, but Agent Manager evidence recording failed: %w", evidenceErr)
	}
	if proofErr != nil {
		return fmt.Errorf("browser session closed, but trusted Agent Manager evidence is pending: %w", proofErr)
	}
	return nil
}

func instructionEvidenceCategory(kind actionsv1.ActionType) string {
	switch kind {
	case actionsv1.ActionType_ACTION_TYPE_NAVIGATE:
		return "navigate"
	case actionsv1.ActionType_ACTION_TYPE_CLICK:
		return "click"
	case actionsv1.ActionType_ACTION_TYPE_INPUT, actionsv1.ActionType_ACTION_TYPE_SELECT:
		return "type"
	case actionsv1.ActionType_ACTION_TYPE_SCROLL:
		return "scroll"
	case actionsv1.ActionType_ACTION_TYPE_SCREENSHOT:
		return "screenshot"
	default:
		return "observe"
	}
}

func actionEvidenceCategory(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "navigate":
		return "navigate"
	case "click":
		return "click"
	case "type", "input":
		return "type"
	case "scroll":
		return "scroll"
	case "screenshot":
		return "screenshot"
	default:
		return "observe"
	}
}

func (b *Broker) Trace(ctx context.Context, sessionID string) (Trace, error) {
	e, err := b.get(sessionID)
	if err != nil {
		return Trace{}, err
	}
	if err := b.lockEntry(e); err != nil {
		return Trace{}, err
	}
	defer e.opMu.Unlock()
	if e.session == nil {
		if err := b.authorize(ctx, e, sessionpermission.ActionRequest{Kind: "history_read", CurrentURL: e.tabURL}); err != nil {
			return Trace{}, err
		}
		return b.traceSnapshot(e), nil
	}
	if err := e.session.AuthorizeAction(ctx, "history_read", "", ""); err != nil {
		return Trace{}, err
	}
	return b.traceSnapshot(e), nil
}

// NavigationHistory reads the managed browser navigation stack under the existing
// session lease. History URLs outside the current permission profile are removed
// before the response leaves the broker. Selected-tab history remains unsupported.
func (b *Broker) NavigationHistory(ctx context.Context, sessionID string) (driver.NavigationStackResponse, error) {
	e, err := b.get(sessionID)
	if err != nil {
		return driver.NavigationStackResponse{}, err
	}
	if err := b.lockEntry(e); err != nil {
		return driver.NavigationStackResponse{}, err
	}
	defer e.opMu.Unlock()
	if e.session == nil {
		return driver.NavigationStackResponse{}, errors.New("browser navigation history is unsupported for user-attached sessions")
	}
	if !e.profile.HistoryRead {
		return driver.NavigationStackResponse{}, sessionpermission.ErrDenied
	}
	stack, err := e.session.GetNavigationStack(ctx, "")
	if err != nil {
		return driver.NavigationStackResponse{}, err
	}
	return filterNavigationHistory(e.profile, *stack)
}

func filterNavigationHistory(profile sessionpermission.Profile, history driver.NavigationStackResponse) (driver.NavigationStackResponse, error) {
	if history.Current == nil || !profile.AllowsURL(history.Current.URL) {
		return driver.NavigationStackResponse{}, sessionpermission.ErrDenied
	}
	filtered := driver.NavigationStackResponse{SessionID: history.SessionID, Current: history.Current}
	for _, item := range history.BackStack {
		if profile.AllowsURL(item.URL) {
			filtered.BackStack = append(filtered.BackStack, item)
		}
	}
	for _, item := range history.ForwardStack {
		if profile.AllowsURL(item.URL) {
			filtered.ForwardStack = append(filtered.ForwardStack, item)
		}
	}
	return filtered, nil
}

func (b *Broker) traceSnapshot(e *entry) Trace {
	b.mu.RLock()
	defer b.mu.RUnlock()
	trace := e.trace
	trace.Events = append([]TraceEvent(nil), e.trace.Events...)
	return trace
}

// ResolveInferenceBinding returns the signed run and grant already bound to
// an active broker session. The caller must present the same AM credentials
// used to open/authorize the session; no caller-supplied run or grant fields
// are accepted.
func (b *Broker) ResolveInferenceBinding(ctx context.Context, sessionID string) (BrowserInferenceBinding, error) {
	e, err := b.get(sessionID)
	if err != nil {
		return BrowserInferenceBinding{}, err
	}
	if err := b.lockEntry(e); err != nil {
		return BrowserInferenceBinding{}, err
	}
	defer e.opMu.Unlock()
	if e.session == nil && e.attachedTab == nil {
		return BrowserInferenceBinding{}, ErrSessionNotOwned
	}
	credentials, err := credentialsFromContext(ctx)
	if err != nil {
		return BrowserInferenceBinding{}, err
	}
	if e.grantReceipt == "" || subtle.ConstantTimeCompare([]byte(credentials.grantReceipt), []byte(e.grantReceipt)) != 1 {
		return BrowserInferenceBinding{}, ErrSessionNotOwned
	}
	if err := b.authorize(ctx, e, sessionpermission.ActionRequest{Kind: "observe", CurrentURL: e.tabURL, TargetURL: e.tabURL}); err != nil {
		return BrowserInferenceBinding{}, err
	}
	const runPrefix = "run:"
	if !strings.HasPrefix(e.callerID, runPrefix) {
		return BrowserInferenceBinding{}, errors.New("active browser session is not bound to an Agent Manager run")
	}
	runID, err := uuid.Parse(strings.TrimPrefix(e.callerID, runPrefix))
	if err != nil {
		return BrowserInferenceBinding{}, errors.New("active browser session has an invalid Agent Manager run identity")
	}
	return BrowserInferenceBinding{RunID: runID.String(), SessionID: e.sessionID, GrantReceipt: e.grantReceipt}, nil
}

func (b *Broker) get(sessionID string) (*entry, error) {
	b.mu.RLock()
	e := b.entries[sessionID]
	closed := e != nil && (e.closed || e.revoking.Load())
	b.mu.RUnlock()
	if e == nil || closed {
		return nil, ErrSessionNotOwned
	}
	if e.session == nil {
		if e.backend == string(sessionpermission.BackendUserAttached) {
			return e, nil
		}
		return nil, ErrSessionNotOwned
	}
	owned, ok := b.manager.Get(sessionID)
	if !ok || owned != e.session {
		return nil, ErrSessionNotOwned
	}
	return e, nil
}

func (b *Broker) lockEntry(e *entry) error {
	e.opMu.Lock()
	b.mu.Lock()
	owned := b.entries[e.sessionID] == e && !e.closed && !e.revoking.Load()
	expired := owned && e.backend == string(sessionpermission.BackendUserAttached) && !e.socketTicketExpiry.IsZero() && !time.Now().Before(e.socketTicketExpiry)
	var tab AttachedTab
	var conn *websocket.Conn
	if expired {
		e.revoking.Store(true)
		e.closed = true
		e.closedAt = time.Now().UTC()
		tab, conn = e.attachedTab, e.attachedSocket
		delete(b.entries, e.sessionID)
	}
	b.mu.Unlock()
	if expired {
		if tab != nil {
			_ = tab.Close()
		} else if conn != nil {
			_ = conn.Close()
		}
		e.opMu.Unlock()
		return ErrSessionNotOwned
	}
	if !owned {
		e.opMu.Unlock()
		return ErrSessionNotOwned
	}
	return nil
}

func (b *Broker) authorize(ctx context.Context, e *entry, request sessionpermission.ActionRequest) error {
	executionID, leaseID := e.ownerID, e.leaseID
	if e.session != nil {
		executionID, leaseID = e.session.OwnerIdentity()
	}
	request.SessionID, request.LeaseID, request.OwnerID = e.sessionID, leaseID, executionID
	if request.CurrentURL == "" {
		request.CurrentURL = e.tabURL
	}
	if request.TabID == "" {
		request.TabID = e.tabID
	}
	if request.ConsentReceipt == "" {
		request.ConsentReceipt = e.consentReceipt
	}
	if err := (&sessionAuthorizer{resolver: b.profiles, caller: b.caller, permissionRef: e.permissionRef, profile: e.profile, policy: e.policy, expectedCallerID: e.callerID}).Authorize(ctx, request); err != nil {
		return fmt.Errorf("authorize browser %s: %w", request.Kind, err)
	}
	return nil
}

func (b *Broker) info(e *entry) SessionInfo {
	role := string(e.profile.Effect)
	caps := capabilities(e.profile, e.backend)
	if e.handedOff {
		role = string(sessionpermission.EffectObserve)
		filtered := caps[:0]
		for _, capability := range caps {
			if capability != "act.selected-tab-typed-v1" {
				filtered = append(filtered, capability)
			}
		}
		caps = filtered
	}
	info := SessionInfo{SessionID: e.sessionID, PermissionRef: e.permissionRef, Mode: e.backend, Role: role, Capabilities: caps}
	if e.profile.View && (e.backend == string(sessionpermission.BackendManaged) || e.backend == string(sessionpermission.BackendInstalled)) && e.profile.IdentityProfile != nil {
		identity := *e.profile.IdentityProfile
		info.IdentityProfile = &identity
	}
	if e.session != nil {
		info.AudioDeviceEvidence = e.session.AudioDeviceEvidence()
		if e.profile.View {
			info.ActualViewport = e.session.ActualViewport()
			info.IdentityEvidence = e.session.IdentityEvidence()
		}
	}
	return info
}

func browserProfileFromIdentity(identity *sessionpermission.BrowserIdentityProfile) *sessionprofilepersistence.BrowserProfile {
	if identity == nil {
		return nil
	}
	return &sessionprofilepersistence.BrowserProfile{
		Preset: identity.Preset,
		Fingerprint: &sessionprofilepersistence.FingerprintSettings{
			ViewportWidth: identity.ViewportWidth, ViewportHeight: identity.ViewportHeight,
			DeviceScaleFactor: identity.DeviceScaleFactor, Locale: identity.Locale,
			TimezoneID: identity.TimezoneID, ColorScheme: identity.ColorScheme,
		},
	}
}

func (b *Broker) record(e *entry, event TraceEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	event.Sequence = uint64(len(e.trace.Events) + 1)
	event.SessionID = e.sessionID
	switch event.Kind {
	case "observation":
		event.PermissionClass = "observe"
		event.EffectClass = "none"
	case "action":
		event.PermissionClass = string(e.profile.Effect)
		if event.Instruction != nil && event.Instruction.Action != nil {
			switch event.Instruction.Action.Type {
			case actionsv1.ActionType_ACTION_TYPE_NAVIGATE:
				event.EffectClass = "navigation"
			case actionsv1.ActionType_ACTION_TYPE_CLICK, actionsv1.ActionType_ACTION_TYPE_INPUT:
				event.EffectClass = "ui"
			default:
				event.EffectClass = "unsupported"
			}
		} else {
			event.EffectClass = "unsupported"
		}
	default:
		event.PermissionClass = string(e.profile.Effect)
		event.EffectClass = "unsupported"
	}
	e.trace.Events = append(e.trace.Events, event)
}

func capabilities(profile sessionpermission.Profile, backend string) []string {
	result := []string{"trace.typed-v1"}
	if backend == string(sessionpermission.BackendUserAttached) {
		if profile.View {
			result = append(result, "observe.selected-tab-text", "observe.selected-tab-consent-bound-result-v1")
		}
		if profile.Control && profile.Effect != sessionpermission.EffectObserve {
			result = append(result, "act.selected-tab-typed-v1")
		}
		if profile.ExtensionAccess {
			result = append(result, "extensions.inventory-metadata")
		}
		return result
	}
	if profile.View {
		result = append(result, "observe.navigation-state", "observe.screenshot")
	}
	if profile.Control && profile.Effect != sessionpermission.EffectObserve {
		result = append(result, "act.compiled-instruction")
	}
	switch profile.AudioMode {
	case sessionpermission.AudioModeFakeMicrophone:
		result = append(result, "audio.fake-microphone")
	case sessionpermission.AudioModePipeWire:
		result = append(result, "audio.pipewire-qualification")
	}
	return result
}

// sessionAuthorizer resolves identity and profile policy afresh for each
// operation, allowing revocation and run-grant changes to take effect.
type sessionAuthorizer struct {
	resolver         ProfileResolver
	caller           CallerResolver
	permissionRef    string
	profile          sessionpermission.Profile
	policy           sessionpermission.Authorizer
	expectedCallerID string
}

func (a *sessionAuthorizer) ProfileSnapshot() sessionpermission.Profile {
	if a == nil {
		return sessionpermission.Profile{}
	}
	profile := a.profile
	profile.URLAllow = append([]string(nil), profile.URLAllow...)
	profile.URLDeny = append([]string(nil), profile.URLDeny...)
	profile.CredentialIDs = append([]string(nil), profile.CredentialIDs...)
	if profile.ExtensionPermissions != nil {
		permissions := make(map[string][]string, len(profile.ExtensionPermissions))
		for extensionID, values := range profile.ExtensionPermissions {
			permissions[extensionID] = append([]string(nil), values...)
		}
		profile.ExtensionPermissions = permissions
	}
	return profile
}

func (a *sessionAuthorizer) Authorize(ctx context.Context, request sessionpermission.ActionRequest) error {
	if a == nil || a.resolver == nil || a.caller == nil {
		return ErrAuthorizerRequired
	}
	if request.CallerID == "" {
		callerID, err := a.caller.BrowserCallerID(ctx)
		if err != nil || callerID == "" {
			if err == nil {
				err = errors.New("authenticated caller is missing")
			}
			return fmt.Errorf("resolve trusted browser caller: %w", err)
		}
		request.CallerID = callerID
	}
	if a.expectedCallerID == "" || request.CallerID != a.expectedCallerID {
		return errors.New("browser session belongs to a different trusted caller")
	}
	profile, currentPolicy, err := a.resolver.Resolve(ctx, a.permissionRef)
	if err != nil || currentPolicy == nil {
		if err == nil {
			err = ErrAuthorizerRequired
		}
		return fmt.Errorf("resolve browser permission profile: %w", err)
	}
	if !reflect.DeepEqual(profile, a.profile) {
		// A still-authenticated session owner must be able to release its
		// existing lease after the run's action profile is narrowed. This
		// exception applies only to session close; current caller/grant
		// resolution above remains mandatory, and every browser effect still
		// fails closed on profile drift.
		if request.Kind == "close" {
			return nil
		}
		return errors.New("browser permission profile changed during session; reopen under the updated grant")
	}
	policy := a.policy
	if policy == nil {
		policy = currentPolicy
	}
	return policy.Authorize(ctx, request)
}

// HarnessSession returns an adapter-safe borrowed view. It exposes only the
// observation and action vocabulary and carries no lifecycle or driver handle.
func (b *Broker) HarnessSession(sessionID string) (harness.BorrowedSession, error) {
	e, err := b.get(sessionID)
	if err != nil {
		return nil, err
	}
	if e.session == nil {
		return nil, errors.New("harness adapters are not qualified for a user-attached browser")
	}
	return &borrowedSession{broker: b, sessionID: sessionID}, nil
}

type borrowedSession struct {
	broker    *Broker
	sessionID string
}

func (s *borrowedSession) Observe(ctx context.Context, req harness.ObserveRequest) (harness.Observation, error) {
	switch req.Mode {
	case "navigation":
		e, err := s.broker.get(s.sessionID)
		if err != nil {
			return harness.Observation{}, err
		}
		state, err := e.session.ReadNavigationState(ctx, "observe", "", "")
		if err != nil {
			return harness.Observation{}, err
		}
		s.broker.record(e, TraceEvent{Kind: "observation", OccurredAt: time.Now().UTC(), URL: safeOrigin(state.URL)})
		data, _ := json.Marshal(map[string]string{"url": state.URL, "title": state.Title})
		return harness.Observation{Mode: "navigation", Data: data, TraceID: e.trace.ID}, nil
	case "screenshot":
		obs, err := s.broker.Observe(ctx, s.sessionID, ObserveRequest{})
		if err != nil {
			return harness.Observation{}, err
		}
		data, _ := json.Marshal(obs)
		return harness.Observation{Mode: "screenshot", Data: data}, nil
	case "accessibility-tree":
		tree, err := s.broker.AccessibilityTree(ctx, s.sessionID)
		if err != nil {
			return harness.Observation{}, err
		}
		data, err := json.Marshal(tree)
		if err != nil {
			return harness.Observation{}, fmt.Errorf("encode accessibility observation: %w", err)
		}
		e, err := s.broker.get(s.sessionID)
		if err != nil {
			return harness.Observation{}, err
		}
		return harness.Observation{Mode: "accessibility-tree", Data: data, TraceID: e.trace.ID}, nil
	default:
		return harness.Observation{}, fmt.Errorf("observation mode %q is not exposed by this BAS broker", req.Mode)
	}
}

func (s *borrowedSession) Act(ctx context.Context, req harness.ActRequest) (harness.ActionResult, error) {
	var instruction contracts.CompiledInstruction
	if len(req.Input) == 0 {
		return harness.ActionResult{}, errors.New("input must contain a typed BAS compiled instruction")
	}
	if err := json.Unmarshal(req.Input, &instruction); err != nil {
		return harness.ActionResult{}, fmt.Errorf("decode compiled instruction: %w", err)
	}
	if instruction.Action == nil || actionVocabularyName(instruction.Action.Type) != req.Action {
		return harness.ActionResult{}, errors.New("action must match input instruction action type")
	}
	result, err := s.broker.Act(ctx, s.sessionID, ActRequest{Instruction: instruction, TargetURL: req.Target})
	if err != nil {
		return harness.ActionResult{}, err
	}
	data, err := json.Marshal(result)
	if err != nil {
		return harness.ActionResult{}, err
	}
	e, _ := s.broker.get(s.sessionID)
	return harness.ActionResult{Accepted: result.Outcome.Success, Data: data, TraceID: e.trace.ID}, nil
}

func actionVocabularyName(kind actionsv1.ActionType) string {
	switch kind {
	case actionsv1.ActionType_ACTION_TYPE_NAVIGATE:
		return "navigate"
	case actionsv1.ActionType_ACTION_TYPE_CLICK:
		return "click"
	case actionsv1.ActionType_ACTION_TYPE_INPUT:
		return "type"
	case actionsv1.ActionType_ACTION_TYPE_SELECT:
		return "select"
	case actionsv1.ActionType_ACTION_TYPE_SCROLL:
		return "scroll"
	case actionsv1.ActionType_ACTION_TYPE_WAIT:
		return "wait"
	case actionsv1.ActionType_ACTION_TYPE_ASSERT:
		return "assert"
	case actionsv1.ActionType_ACTION_TYPE_EXTRACT:
		return "extract"
	case actionsv1.ActionType_ACTION_TYPE_SCREENSHOT:
		return "screenshot"
	default:
		return ""
	}
}

func safeOrigin(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
