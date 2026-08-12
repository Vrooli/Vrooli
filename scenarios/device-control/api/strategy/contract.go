// Package strategy defines the deliberately small device-control adapter
// contract. Everything above the three floor operations is capability-gated.
package strategy

import (
	"context"
	"errors"
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
	Kind   string  `json:"kind"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Button string  `json:"button,omitempty"`
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

// DeviceScoped lets one strategy implementation bind commands to a specific
// physical target when several devices share the same transport adapter.
type DeviceScoped interface {
	ForDevice(serial string) Strategy
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
	Status           string                `json:"status"`
	Capabilities     map[string]Capability `json:"capabilities"`
	Tiers            []string              `json:"tiers"`
	NextActions      []string              `json:"next_actions,omitempty"`
	Promotable       bool                  `json:"promotable"`
	EvidenceClass    string                `json:"evidence_class"`
	MinimumUsefulFPS float64               `json:"minimum_useful_fps"`
}

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
		steps = append(steps, "swipe", "text")
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
		steps = append(steps, "screenrecord")
	}
	return steps
}
