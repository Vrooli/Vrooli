package targetmodel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

// References identify owner resources, never a connection address or a grant.
// Hosts remain explicit for attached devices; the API host is not implicitly
// the user's machine. Owners resolve these references before admitting access.
type TargetRef struct {
	OwnerScenario string `json:"owner_scenario"`
	ResourceID    string `json:"resource_id"`
	HostNodeID    string `json:"host_node_id,omitempty"`
}

type SurfaceRef struct {
	Target        TargetRef `json:"target"`
	OwnerScenario string    `json:"owner_scenario"`
	SurfaceID     string    `json:"surface_id"`
}

// SessionRef identifies temporary access separately from the selected surface.
// DesktopSessionID is the OS login session, not the API process or lease ID.
// A reference is not evidence that its caller holds permission to use it.
type SessionRef struct {
	Surface          SurfaceRef `json:"surface"`
	SessionID        string     `json:"session_id"`
	DesktopSessionID string     `json:"desktop_session_id,omitempty"`
}

type SurfaceKind string

const (
	SurfaceScenario    SurfaceKind = "scenario"
	SurfaceTerminal    SurfaceKind = "terminal"
	SurfaceDesktop     SurfaceKind = "desktop"
	SurfaceBrowser     SurfaceKind = "browser"
	SurfaceDevicePanel SurfaceKind = "device-panel"
)

// CapabilityState does not collapse permission denial or unsupported APIs into
// observed absence. These facts describe a capability, not universal admission.
type CapabilityState string

const (
	CapabilityReady       CapabilityState = "ready"
	CapabilityMissing     CapabilityState = "missing"
	CapabilityUnsupported CapabilityState = "unsupported"
	CapabilityUnknown     CapabilityState = "unknown"
	CapabilityDenied      CapabilityState = "denied"
)

type CapabilityFact struct {
	Capability string          `json:"capability"`
	State      CapabilityState `json:"state"`
	ReasonCode string          `json:"reason_code,omitempty"`
	EvidenceID string          `json:"evidence_id,omitempty"`
	ObservedAt time.Time       `json:"observed_at"`
	ExpiresAt  time.Time       `json:"expires_at"`
}

// EffectiveState fails closed for expired, future-dated, or malformed facts.
// Cached readiness is never sufficient authority to perform an action.
func (f CapabilityFact) EffectiveState(now time.Time) CapabilityState {
	if f.Validate() != nil || now.Before(f.ObservedAt) || !now.Before(f.ExpiresAt) {
		return CapabilityUnknown
	}
	return f.State
}

// LegacyReadiness preserves the older four-state vocabulary conservatively.
// Neither denied nor unsupported means that a prerequisite is simply missing,
// or that it is irrelevant (not_applicable), so older consumers see unknown.
func (s CapabilityState) LegacyReadiness() ReadinessState {
	switch s {
	case CapabilityReady:
		return ReadinessReady
	case CapabilityMissing:
		return ReadinessMissing
	default:
		return ReadinessUnknown
	}
}

// SurfaceDescriptor is a safe projection, deliberately without arbitrary
// endpoint, credentials, connection offer, or extension metadata fields.
// Connection offers belong to the owner's separately authorized OpenSession.
type SurfaceDescriptor struct {
	Ref              SurfaceRef       `json:"ref"`
	Kind             SurfaceKind      `json:"kind"`
	DisplayLabel     string           `json:"display_label"`
	Capabilities     []CapabilityFact `json:"capabilities"`
	ProtocolVersions []string         `json:"protocol_versions"`
	DesktopSessionID string           `json:"desktop_session_id,omitempty"`
	DisplayIDs       []string         `json:"display_ids,omitempty"`
}

// DecodeSurfaceDescriptor is the boundary for untrusted JSON projections.
// Unknown fields are rejected rather than retained as connection authority.
func DecodeSurfaceDescriptor(data []byte) (SurfaceDescriptor, error) {
	var descriptor SurfaceDescriptor
	if len(data) > 64*1024 {
		return descriptor, fmt.Errorf("surface descriptor exceeds 64 KiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&descriptor); err != nil {
		return SurfaceDescriptor{}, fmt.Errorf("invalid surface descriptor")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return SurfaceDescriptor{}, fmt.Errorf("unexpected trailing descriptor content")
	}
	if err := descriptor.Validate(); err != nil {
		return SurfaceDescriptor{}, err
	}
	return descriptor, nil
}

var ownerName = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
var opaqueID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`)

func validateOwner(owner string) error {
	if len(owner) > 63 || !ownerName.MatchString(owner) {
		return fmt.Errorf("invalid owner scenario")
	}
	return nil
}

func validateID(id string) error {
	if len(id) > 255 || !opaqueID.MatchString(id) {
		return fmt.Errorf("expected a bounded opaque identifier, not a transport address")
	}
	return nil
}

func (r TargetRef) Validate() error {
	if err := validateOwner(r.OwnerScenario); err != nil {
		return err
	}
	if err := validateID(r.ResourceID); err != nil {
		return fmt.Errorf("target resource: %w", err)
	}
	if r.HostNodeID != "" {
		if err := validateID(r.HostNodeID); err != nil {
			return fmt.Errorf("host node: %w", err)
		}
	}
	return nil
}

func (r SurfaceRef) Validate() error {
	if err := r.Target.Validate(); err != nil {
		return err
	}
	if err := validateOwner(r.OwnerScenario); err != nil {
		return err
	}
	return validateID(r.SurfaceID)
}

func (r SessionRef) Validate() error {
	if err := r.Surface.Validate(); err != nil {
		return err
	}
	if err := validateID(r.SessionID); err != nil {
		return err
	}
	if r.DesktopSessionID != "" {
		return validateID(r.DesktopSessionID)
	}
	return nil
}

func (f CapabilityFact) Validate() error {
	if err := validateID(f.Capability); err != nil {
		return err
	}
	switch f.State {
	case CapabilityReady, CapabilityMissing, CapabilityUnsupported, CapabilityUnknown, CapabilityDenied:
	default:
		return fmt.Errorf("invalid capability state")
	}
	if f.State != CapabilityReady && f.ReasonCode == "" {
		return fmt.Errorf("non-ready capability requires a reason code")
	}
	for _, id := range []string{f.ReasonCode, f.EvidenceID} {
		if id != "" {
			if err := validateID(id); err != nil {
				return err
			}
		}
	}
	if f.ObservedAt.IsZero() || !f.ExpiresAt.After(f.ObservedAt) {
		return fmt.Errorf("capability requires a bounded observation interval")
	}
	if f.State == CapabilityReady && f.EvidenceID == "" {
		return fmt.Errorf("ready capability requires observation evidence")
	}
	return nil
}

func (d SurfaceDescriptor) Validate() error {
	if err := d.Ref.Validate(); err != nil {
		return err
	}
	switch d.Kind {
	case SurfaceScenario, SurfaceTerminal, SurfaceDesktop, SurfaceBrowser, SurfaceDevicePanel:
	default:
		return fmt.Errorf("invalid surface kind")
	}
	if strings.TrimSpace(d.DisplayLabel) == "" || len(d.DisplayLabel) > 256 || strings.ContainsAny(d.DisplayLabel, "\x00\r\n") {
		return fmt.Errorf("surface requires a bounded display label")
	}
	if d.Kind == SurfaceDesktop && d.DesktopSessionID == "" {
		return fmt.Errorf("desktop surface requires an explicit desktop session")
	}
	if d.DesktopSessionID != "" {
		if err := validateID(d.DesktopSessionID); err != nil {
			return err
		}
	}
	if len(d.Capabilities) > 64 {
		return fmt.Errorf("too many surface capabilities")
	}
	seen := make(map[string]bool)
	for _, fact := range d.Capabilities {
		if err := fact.Validate(); err != nil {
			return err
		}
		if seen[fact.Capability] {
			return fmt.Errorf("duplicate capability fact")
		}
		seen[fact.Capability] = true
	}
	if len(d.ProtocolVersions) == 0 || len(d.ProtocolVersions) > 16 || len(d.DisplayIDs) > 32 {
		return fmt.Errorf("invalid protocol or display list size")
	}
	for _, ids := range [][]string{d.ProtocolVersions, d.DisplayIDs} {
		seen = make(map[string]bool)
		for _, id := range ids {
			if err := validateID(id); err != nil {
				return err
			}
			if seen[id] {
				return fmt.Errorf("duplicate protocol or display identity")
			}
			seen[id] = true
		}
	}
	return nil
}

// NegotiateProtocol uses the consumer's preference order, with exact identities
// (including the protocol family). Desktop and terminal versions never alias.
func NegotiateProtocol(preferred, offered []string) (string, error) {
	if len(preferred) > 16 || len(offered) > 16 {
		return "", fmt.Errorf("too many protocol versions")
	}
	for _, versions := range [][]string{preferred, offered} {
		for _, version := range versions {
			if err := validateID(version); err != nil {
				return "", err
			}
		}
	}
	for _, candidate := range preferred {
		for _, version := range offered {
			if candidate == version {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("no compatible surface protocol")
}
