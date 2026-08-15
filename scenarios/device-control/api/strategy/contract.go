// Package strategy defines the deliberately small device-control adapter
// contract. Everything above the three floor operations is capability-gated.
package strategy

import (
	"context"
	"errors"
	"runtime"
	"time"
)

const (
	CapSemanticTree    = "semantic-tree"
	CapAppLifecycle    = "app-lifecycle"
	CapPermissions     = "permissions"
	CapInput           = "input"
	CapScreenshot      = "screenshot"
	CapScreenRecording = "screen-recording"
	CapNetworkControl  = "network-control"
	CapOrientation     = "orientation"
	CapClipboard       = "clipboard"
	CapFileTransfer    = "file-transfer"
	CapDeviceLogs      = "device-logs"
	CapWebViewAttach   = "webview-attach"
	CapNativeRecording = "native-recording"
	CapMultiTouch      = "multi-touch"
)

const (
	StatusAvailable    = "available"
	StatusUnavailable  = "unavailable"
	StatusUnsupported  = "unsupported"
	StatusDegraded     = "degraded"
	HealthUnreachable  = "unreachable"
	HealthUnauthorized = "unauthorized"
)

type PointerEvent struct {
	Kind       string  `json:"kind"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Button     string  `json:"button,omitempty"`
	Normalized bool    `json:"normalized,omitempty"`
	DurationMS int     `json:"duration_ms,omitempty"`
	Velocity   float64 `json:"velocity,omitempty"`
}

type KeyEvent struct {
	Kind string `json:"kind"`
	Key  string `json:"key"`
}

type Actuation struct {
	Pointer    *PointerEvent `json:"pointer,omitempty"`
	Key        *KeyEvent     `json:"key,omitempty"`
	Text       string        `json:"text,omitempty"`
	Action     string        `json:"action,omitempty"`
	Package    string        `json:"package,omitempty"`
	Permission string        `json:"permission,omitempty"`
	Value      string        `json:"value,omitempty"`
	Expected   string        `json:"expected,omitempty"`
	// Output receives bytes produced by output-bearing actions. It is an
	// in-process sink and is never serialized across the strategy boundary.
	Output *[]byte `json:"-"`
}

// ClaimClass describes the visual claim a recording is expected to support.
// It is deliberately part of the strategy contract so a recorder cannot
// silently treat a low-rate transition recording as equivalent to a still.
type ClaimClass string

const (
	ClaimStatic     ClaimClass = "static"
	ClaimTransition ClaimClass = "transition"
	ClaimAnimation  ClaimClass = "animation"
)

type RecordingHandle struct {
	ID         string     `json:"id"`
	ClaimClass ClaimClass `json:"claim_class"`
	StartedAt  time.Time  `json:"started_at"`
}

type RecordingArtifact struct {
	Bytes        []byte        `json:"-"`
	Method       string        `json:"recording_method"`
	ClaimClass   ClaimClass    `json:"claim_class"`
	FrameCount   int           `json:"frame_count"`
	Duration     time.Duration `json:"duration"`
	EffectiveFPS float64       `json:"effective_fps"`
}

// SessionRecorder is optional: strategies that cannot record declare that
// capability unavailable. Keeping it optional prevents a fake or a platform
// stub from claiming a recorder it cannot actually operate.
type SessionRecorder interface {
	StartRecording(context.Context, ClaimClass) (RecordingHandle, error)
	StopRecording(context.Context, RecordingHandle) (RecordingArtifact, error)
}

type Frame struct {
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Scale     float64   `json:"scale"`
	Timestamp time.Time `json:"timestamp"`
	MediaType string    `json:"media_type"`
	Bytes     []byte    `json:"-"`
}

// Device is a discovered target. Its identity is separate from the strategy
// implementation so reconnects do not reattribute audit history.
type Device struct {
	ID           string
	Serial       string
	Model        string
	OSVersion    string
	StrategyID   string
	Transport    string
	Health       string
	HealthReason string
	ObservedAt   time.Time
}

// Enumerator is optional so non-enumerating strategies remain compatible.
type Enumerator interface {
	Enumerate(context.Context) ([]Device, error)
}

// SemanticResolver is the deterministic view-hierarchy seam. Adapters own
// transport-specific tree capture, but the flow executor owns rung ordering,
// anchor fallback, and evidence. Bounds are normalized to the observed frame.
type SemanticResolver interface {
	ResolveSemantic(context.Context, string) (SemanticResult, error)
}

// WebViewAttacher exposes the device-owned debugging socket and port-forward
// seam used by a browser automation client. The client never receives an adb
// command or a device socket name that it can reinterpret; it receives only
// the bounded local CDP endpoint selected under the device lease.
type WebViewAttacher interface {
	AttachWebView(context.Context, string) (WebViewEndpoint, error)
}

type WebViewEndpoint struct {
	Package     string `json:"package"`
	Socket      string `json:"socket"`
	CDPEndpoint string `json:"cdp_endpoint"`
	RendererID  string `json:"renderer_id"`
	Transport   string `json:"transport"`
}

type SemanticResult struct {
	Bounds     []float64 `json:"bounds"`
	Confidence float64   `json:"confidence"`
}

type DeviceState struct {
	ForegroundPackage string            `json:"foreground_package,omitempty"`
	ScreenState       string            `json:"screen_state,omitempty"`
	LockState         string            `json:"lock_state,omitempty"`
	Orientation       string            `json:"orientation,omitempty"`
	AutoRotate        bool              `json:"auto_rotate"`
	BatteryLevel      int               `json:"battery_level,omitempty"`
	Charging          bool              `json:"charging"`
	ThermalStatus     string            `json:"thermal_status,omitempty"`
	DisplayWidth      int               `json:"display_width,omitempty"`
	DisplayHeight     int               `json:"display_height,omitempty"`
	DisplayDensity    int               `json:"display_density,omitempty"`
	Unavailable       map[string]string `json:"unavailable,omitempty"`
}

type StateReader interface {
	ReadState(context.Context) (DeviceState, error)
}

// UnlockRequest is deliberately not serializable. Secret is populated only
// after the credential authority resolves a profile reference and is consumed
// synchronously by the strategy adapter. Adapters must never place it in
// command arguments, logs, audit records, or evidence.
type UnlockRequest struct {
	Method       string
	Secret       []byte
	MaxAttempts  int
	AttemptLimit time.Duration
	Settle       time.Duration
}

type UnlockResult struct {
	Outcome  string `json:"outcome"`
	Attempts int    `json:"attempts"`
	Detail   string `json:"detail,omitempty"`
}

// Unlocker is the strategy boundary for device authentication. A strategy
// reports transport and postcondition facts; the control service owns profile
// policy, credential resolution, leases, and audit metadata.
type Unlocker interface {
	Unlock(context.Context, UnlockRequest) (UnlockResult, error)
}

type StateRestorer interface {
	RestoreState(context.Context, DeviceState) error
}

// DeviceScoped lets one strategy implementation bind commands to a specific
// physical target when several devices share the same transport adapter.
type DeviceScoped interface {
	ForDevice(serial string) Strategy
}

// WirelessReconnector re-establishes a previously promoted wireless
// transport, including endpoint discovery when Android rotates its wireless
// debugging address or port.
type WirelessReconnector interface {
	ReconnectWireless(context.Context) error
}

type Capability struct {
	Name          string `json:"name"`
	Status        string `json:"status"`
	Prerequisite  string `json:"prerequisite,omitempty"`
	NextAction    string `json:"next_action,omitempty"`
	ProbeEvidence string `json:"probe_evidence,omitempty"`
}

type Declaration struct {
	StrategyID       string                `json:"strategy_id"`
	Description      string                `json:"description"`
	SupportedHostOS  []string              `json:"supported_host_os"`
	Reason           string                `json:"reason,omitempty"`
	Status           string                `json:"status"`
	Capabilities     map[string]Capability `json:"capabilities"`
	Tiers            []string              `json:"tiers"`
	NextActions      []string              `json:"next_actions,omitempty"`
	Promotable       bool                  `json:"promotable"`
	EvidenceClass    string                `json:"evidence_class"`
	MinimumUsefulFPS float64               `json:"minimum_useful_fps"`
}

// HostOS is the host operating-system seam used by strategy resolution. It is
// runtime.GOOS in production and intentionally mutable in tests so every
// platform branch can be proven without pretending to be on another host.
var HostOS = runtime.GOOS

type Strategy interface {
	ID() string
	Observe(context.Context) (Frame, error)
	Actuate(context.Context, Actuation) error
	Describe(context.Context) (Declaration, error)
}

var (
	ErrUnavailable = errors.New("strategy unavailable")
	ErrUnsupported = errors.New("strategy capability unsupported")
)

type AvailabilityError struct {
	Reason     string
	NextAction string
}

func (e *AvailabilityError) Error() string { return e.Reason }

func Tiers(d Declaration) []string {
	if d.Status != StatusAvailable {
		return []string{}
	}
	available := func(name string) bool { return d.Capabilities[name].Status == StatusAvailable }
	tiers := []string{"floor"}
	if available(CapSemanticTree) {
		tiers = append(tiers, "semantic")
	}
	if available(CapScreenshot) && available(CapInput) {
		tiers = append(tiers, "observer")
	}
	if available(CapAppLifecycle) && available(CapPermissions) && available(CapSemanticTree) {
		tiers = append(tiers, "full")
	}
	return tiers
}

func StepKinds(d Declaration) []string {
	if d.Status != StatusAvailable {
		return []string{}
	}
	steps := []string{"observe", "tap", "key", "wait"}
	if d.Capabilities[CapInput].Status == StatusAvailable {
		steps = append(steps, "swipe", "long-press", "double-tap", "drag", "fling", "scroll-to", "text")
	}
	if d.Capabilities[CapMultiTouch].Status == StatusAvailable {
		steps = append(steps, "pinch")
	}
	if d.Capabilities[CapSemanticTree].Status == StatusAvailable {
		steps = append(steps, "semantic-target", "semantic-assert")
	}
	if d.Capabilities[CapAppLifecycle].Status == StatusAvailable {
		steps = append(steps, "install", "launch", "stop", "uninstall", "clear-data", "package-state")
	}
	if d.Capabilities[CapPermissions].Status == StatusAvailable {
		steps = append(steps, "grant-permission", "revoke-permission")
	}
	if d.Capabilities[CapDeviceLogs].Status == StatusAvailable {
		steps = append(steps, "device-logs")
	}
	if d.Capabilities[CapOrientation].Status == StatusAvailable {
		steps = append(steps, "rotate")
	}
	if d.Capabilities[CapNetworkControl].Status == StatusAvailable {
		steps = append(steps, "network")
	}
	if d.Capabilities[CapAppLifecycle].Status == StatusAvailable {
		steps = append(steps, "deep-link")
	}
	if d.Capabilities[CapScreenRecording].Status == StatusAvailable || d.Capabilities[CapNativeRecording].Status == StatusAvailable {
		steps = append(steps, "screenrecord", "recording-start", "recording-stop")
	}
	return steps
}
