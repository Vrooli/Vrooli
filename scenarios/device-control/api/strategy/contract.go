// Package strategy defines the deliberately small device-control adapter
// contract. Transport modalities are optional and capability-gated.
package strategy

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
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
	CapProperty        = "property"
	CapSensor          = "sensor"
	CapMedia           = "media"
	CapPairing         = "pairing"
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
	// CausationID links an operator request, the transport actuation, and any
	// state transition it produces. It is generated at the control boundary
	// when a caller does not provide one.
	CausationID string `json:"causation_id,omitempty"`
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

// PairRequest contains operator-provided pairing material. Secret is never
// serialized, logged, audited, or retained by a strategy after Pair returns.
type PairRequest struct {
	Secret []byte
}

type PairResult struct {
	Outcome   string `json:"outcome"`
	Transport string `json:"transport"`
	Detail    string `json:"detail,omitempty"`
}

// Pairer performs an interactive transport pairing exchange. The strategy is
// normally device-scoped before this method is called.
type Pairer interface {
	Pair(context.Context, PairRequest) (PairResult, error)
}

type Frame struct {
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	Scale     float64   `json:"scale"`
	Timestamp time.Time `json:"timestamp"`
	MediaType string    `json:"media_type"`
	Bytes     []byte    `json:"-"`
}

// Observer is an optional visual modality. Strategies without a screen do not
// implement it and must declare CapScreenshot unavailable with a reason.
type Observer interface {
	Observe(context.Context) (Frame, error)
}

// InputActuator is an optional input modality. It is intentionally separate
// from the strategy floor so property-only and sensor-only devices are valid.
type InputActuator interface {
	Actuate(context.Context, Actuation) error
}

// Device is a discovered target. Its identity is separate from the strategy
// implementation so reconnects do not reattribute audit history.
type Device struct {
	ID     string
	Name   string
	Serial string
	// Endpoint is the current transport address (for example an mDNS wireless
	// ADB endpoint). It is mutable and is never part of durable identity.
	Endpoint     string
	Model        string
	OSVersion    string
	StrategyID   string
	Transport    string
	Health       string
	HealthReason string
	ObservedAt   time.Time
	// IdentityKey is the durable reconciliation key. Serial remains populated
	// for compatibility with existing adapters and is the preferred identity
	// evidence when present.
	IdentityKey string `json:"identity_key,omitempty"`
	// IdentityKind identifies the hardware-grade claim represented by
	// IdentityKey. It is deliberately not inferred from an address or name.
	IdentityKind string            `json:"identity_kind,omitempty"`
	Transports   []DeviceTransport `json:"transports,omitempty"`
}

// DeviceTransport describes one independently reachable path to a device.
// Capability profiles belong here because the same device can expose a
// different operation set through each transport.
type DeviceTransport struct {
	StrategyID   string                `json:"strategy_id"`
	Name         string                `json:"name"`
	Endpoint     string                `json:"endpoint,omitempty"`
	Health       string                `json:"health"`
	HealthReason string                `json:"health_reason,omitempty"`
	Capabilities map[string]Capability `json:"capabilities"`
	Properties   []PropertyDescriptor  `json:"properties,omitempty"`
	ObservedAt   time.Time             `json:"observed_at,omitempty"`
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
	RendererURL string `json:"renderer_url"`
	Transport   string `json:"transport"`
}

type SemanticResult struct {
	Bounds     []float64 `json:"bounds"`
	Confidence float64   `json:"confidence"`
}

type DeviceState struct {
	ForegroundPackage string                   `json:"foreground_package,omitempty"`
	ScreenState       string                   `json:"screen_state,omitempty"`
	LockState         string                   `json:"lock_state,omitempty"`
	Orientation       string                   `json:"orientation,omitempty"`
	AutoRotate        bool                     `json:"auto_rotate"`
	BatteryLevel      int                      `json:"battery_level,omitempty"`
	Charging          bool                     `json:"charging"`
	ThermalStatus     string                   `json:"thermal_status,omitempty"`
	DisplayWidth      int                      `json:"display_width,omitempty"`
	DisplayHeight     int                      `json:"display_height,omitempty"`
	DisplayDensity    int                      `json:"display_density,omitempty"`
	Unavailable       map[string]string        `json:"unavailable,omitempty"`
	Properties        map[string]PropertyValue `json:"properties,omitempty"`
}

type PropertyValue struct {
	Value     any    `json:"value,omitempty"`
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
	Transport string `json:"transport,omitempty"`
}

type StateReader interface {
	ReadState(context.Context) (DeviceState, error)
}

// StateObserver runs until ctx is cancelled and publishes one event for each
// changed attribute. It is the producer-side contract for world-originated
// state changes; polling is a declared fallback for transports without push.
type StateObserver interface {
	ObserveState(context.Context, StateChangeSink) error
}

// ConformanceTarget lets a transport provide a deterministic, non-network
// target for the contract suite. It is used only by strategy verification;
// device-control's live paths continue to use the real scoped adapter.
type ConformanceTarget interface {
	ConformanceTarget() Strategy
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
	Reason        string `json:"reason,omitempty"`
	Prerequisite  string `json:"prerequisite,omitempty"`
	NextAction    string `json:"next_action,omitempty"`
	ProbeEvidence string `json:"probe_evidence,omitempty"`
	// StateClass distinguishes persistent values from occurrence-only events.
	// It is optional for existing capabilities and required for new modality
	// declarations by the conformance helpers.
	StateClass string `json:"state_class,omitempty"`
}

const (
	StateBearing = "state-bearing"
	EventBearing = "event-bearing"
)

type PropertyDescriptor struct {
	Name        string   `json:"name"`
	ValueType   string   `json:"value_type"`
	Writable    bool     `json:"writable"`
	Minimum     *float64 `json:"minimum,omitempty"`
	Maximum     *float64 `json:"maximum,omitempty"`
	Enumeration []string `json:"enumeration,omitempty"`
	StateClass  string   `json:"state_class,omitempty"`
}

type PropertySet struct {
	Name        string `json:"name"`
	Value       any    `json:"value"`
	CausationID string `json:"causation_id,omitempty"`
}

type PropertyActuator interface {
	GetProperty(context.Context, string) (any, error)
	SetProperty(context.Context, PropertySet) error
}

type PropertyValidationError struct {
	Descriptor string
	Reason     string
}

func (e *PropertyValidationError) Error() string {
	return fmt.Sprintf("property %q value is invalid: %s", e.Descriptor, e.Reason)
}

func ValidatePropertyValue(descriptor PropertyDescriptor, value any) error {
	if descriptor.Name == "" {
		return &PropertyValidationError{Descriptor: "", Reason: "descriptor name is empty"}
	}
	if !descriptor.Writable {
		return &PropertyValidationError{Descriptor: descriptor.Name, Reason: "property is read-only"}
	}
	if len(descriptor.Enumeration) > 0 {
		text := fmt.Sprint(value)
		for _, candidate := range descriptor.Enumeration {
			if text == candidate {
				return nil
			}
		}
		return &PropertyValidationError{Descriptor: descriptor.Name, Reason: "value is outside the enumeration"}
	}
	if descriptor.Minimum == nil && descriptor.Maximum == nil {
		return nil
	}
	valueOf := reflect.ValueOf(value)
	if !valueOf.IsValid() || !valueOf.Type().ConvertibleTo(reflect.TypeOf(float64(0))) {
		return &PropertyValidationError{Descriptor: descriptor.Name, Reason: "value is not numeric"}
	}
	number := valueOf.Convert(reflect.TypeOf(float64(0))).Float()
	if descriptor.Minimum != nil && number < *descriptor.Minimum {
		return &PropertyValidationError{Descriptor: descriptor.Name, Reason: fmt.Sprintf("value is below minimum %v", *descriptor.Minimum)}
	}
	if descriptor.Maximum != nil && number > *descriptor.Maximum {
		return &PropertyValidationError{Descriptor: descriptor.Name, Reason: fmt.Sprintf("value is above maximum %v", *descriptor.Maximum)}
	}
	return nil
}

// ValidateObservedPropertyValue checks a value returned by a transport. It
// intentionally does not require Writable: read-only receiver state is valid
// state and must not be reported as unavailable merely because operators
// cannot set it.
func ValidateObservedPropertyValue(descriptor PropertyDescriptor, value any) error {
	if descriptor.Name == "" {
		return &PropertyValidationError{Descriptor: "", Reason: "descriptor name is empty"}
	}
	if len(descriptor.Enumeration) > 0 {
		text := fmt.Sprint(value)
		for _, candidate := range descriptor.Enumeration {
			if text == candidate {
				return nil
			}
		}
		return &PropertyValidationError{Descriptor: descriptor.Name, Reason: "value is outside the enumeration"}
	}
	if descriptor.ValueType != "" && !observedTypeMatches(descriptor.ValueType, value) {
		return &PropertyValidationError{Descriptor: descriptor.Name, Reason: "value does not match declared type " + descriptor.ValueType}
	}
	if descriptor.Minimum != nil || descriptor.Maximum != nil {
		valueOf := reflect.ValueOf(value)
		if !valueOf.IsValid() || !valueOf.Type().ConvertibleTo(reflect.TypeOf(float64(0))) {
			return &PropertyValidationError{Descriptor: descriptor.Name, Reason: "value is not numeric"}
		}
		number := valueOf.Convert(reflect.TypeOf(float64(0))).Float()
		if descriptor.Minimum != nil && number < *descriptor.Minimum {
			return &PropertyValidationError{Descriptor: descriptor.Name, Reason: fmt.Sprintf("value is below minimum %v", *descriptor.Minimum)}
		}
		if descriptor.Maximum != nil && number > *descriptor.Maximum {
			return &PropertyValidationError{Descriptor: descriptor.Name, Reason: fmt.Sprintf("value is above maximum %v", *descriptor.Maximum)}
		}
	}
	return nil
}

func observedTypeMatches(valueType string, value any) bool {
	switch strings.ToLower(strings.TrimSpace(valueType)) {
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean", "bool":
		_, ok := value.(bool)
		return ok
	case "number", "float", "float64":
		kind := reflect.ValueOf(value)
		return kind.IsValid() && kind.Type().ConvertibleTo(reflect.TypeOf(float64(0)))
	case "integer", "int":
		kind := reflect.ValueOf(value)
		return kind.IsValid() && kind.Kind() >= reflect.Int && kind.Kind() <= reflect.Int64
	default:
		return true
	}
}

type SensorReading struct {
	Name       string    `json:"name"`
	Value      any       `json:"value"`
	Unit       string    `json:"unit,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
	StateClass string    `json:"state_class,omitempty"`
}

type SensorReader interface {
	ReadSensors(context.Context) ([]SensorReading, error)
}

type MediaCommand struct {
	Action      string `json:"action"`
	Value       any    `json:"value,omitempty"`
	CausationID string `json:"causation_id,omitempty"`
}

type MediaController interface {
	ControlMedia(context.Context, MediaCommand) error
}

type Declaration struct {
	DeviceID            string                `json:"device_id,omitempty"`
	Transport           string                `json:"transport,omitempty"`
	StrategyID          string                `json:"strategy_id"`
	Description         string                `json:"description"`
	SupportedHostOS     []string              `json:"supported_host_os"`
	Reason              string                `json:"reason,omitempty"`
	Status              string                `json:"status"`
	Capabilities        map[string]Capability `json:"capabilities"`
	Tiers               []string              `json:"tiers"`
	NextActions         []string              `json:"next_actions,omitempty"`
	Promotable          bool                  `json:"promotable"`
	EvidenceClass       string                `json:"evidence_class"`
	MinimumUsefulFPS    float64               `json:"minimum_useful_fps"`
	Properties          []PropertyDescriptor  `json:"properties,omitempty"`
	ObservationMode     string                `json:"observation_mode,omitempty"`
	ObservationInterval time.Duration         `json:"observation_interval,omitempty"`
	StateObservation    StateObservation      `json:"state_observation,omitempty"`
}

type StateObservation struct {
	Mode     string        `json:"mode,omitempty"`
	Interval time.Duration `json:"interval,omitempty"`
}

// HostOS is the host operating-system seam used by strategy resolution. It is
// runtime.GOOS in production and intentionally mutable in tests so every
// platform branch can be proven without pretending to be on another host.
var HostOS = runtime.GOOS

type Strategy interface {
	ID() string
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

type UnsupportedCapabilityError struct {
	Capability string
	Operation  string
}

func (e *UnsupportedCapabilityError) Error() string {
	if e.Operation == "" {
		return fmt.Sprintf("strategy capability %q is unsupported", e.Capability)
	}
	return fmt.Sprintf("operation %q requires unsupported capability %q", e.Operation, e.Capability)
}

func (e *AvailabilityError) Error() string { return e.Reason }

func Tiers(d Declaration) []string {
	if d.Status != StatusAvailable {
		return []string{}
	}
	available := func(name string) bool { return d.Capabilities[name].Status == StatusAvailable }
	tiers := []string{"floor"}
	if available(CapProperty) {
		tiers = append(tiers, "property")
	}
	if available(CapSensor) {
		tiers = append(tiers, "sensor")
	}
	if available(CapMedia) {
		tiers = append(tiers, "media")
	}
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
	steps := []string{}
	if d.Capabilities[CapScreenshot].Status == StatusAvailable {
		steps = append(steps, "observe")
	}
	if d.Capabilities[CapInput].Status == StatusAvailable {
		steps = append(steps, "tap", "key", "wait", "swipe", "long-press", "double-tap", "drag", "fling", "scroll-to", "text", "screen")
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
		steps = append(steps, "device-logs", "logcat-start", "logcat-stop", "clock-sample")
	}
	if d.Capabilities[CapScreenshot].Status == StatusAvailable {
		steps = append(steps, "screenshot")
	}
	if d.Capabilities[CapClipboard].Status == StatusAvailable {
		steps = append(steps, "clipboard-read", "clipboard-write")
	}
	if d.Capabilities[CapOrientation].Status == StatusAvailable {
		steps = append(steps, "rotate")
	}
	if d.Capabilities[CapNetworkControl].Status == StatusAvailable {
		steps = append(steps, "network")
	}
	if d.Capabilities[CapAppLifecycle].Status == StatusAvailable {
		steps = append(steps, "deep-link", "share")
	}
	if d.Capabilities[CapScreenRecording].Status == StatusAvailable || d.Capabilities[CapNativeRecording].Status == StatusAvailable {
		steps = append(steps, "screenrecord", "recording-start", "recording-stop")
	}
	if d.Capabilities[CapProperty].Status == StatusAvailable {
		steps = append(steps, "property-get", "property-set")
	}
	if d.Capabilities[CapSensor].Status == StatusAvailable {
		steps = append(steps, "sensor-read")
	}
	if d.Capabilities[CapMedia].Status == StatusAvailable {
		steps = append(steps, "media-play", "media-pause", "media-stop", "media-next", "media-previous", "media-volume")
	}
	return steps
}
