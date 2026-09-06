package targetmodel

import (
	"fmt"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (r TargetRef) Proto() *commonv1.TargetRef {
	return &commonv1.TargetRef{OwnerScenario: r.OwnerScenario, ResourceId: r.ResourceID, HostNodeId: r.HostNodeID}
}

func TargetRefFromProto(r *commonv1.TargetRef) (TargetRef, error) {
	if r == nil {
		return TargetRef{}, fmt.Errorf("target reference is required")
	}
	result := TargetRef{OwnerScenario: r.OwnerScenario, ResourceID: r.ResourceId, HostNodeID: r.HostNodeId}
	return result, result.Validate()
}

func (r SurfaceRef) Proto() *commonv1.SurfaceRef {
	return &commonv1.SurfaceRef{Target: r.Target.Proto(), OwnerScenario: r.OwnerScenario, SurfaceId: r.SurfaceID}
}

func SurfaceRefFromProto(r *commonv1.SurfaceRef) (SurfaceRef, error) {
	if r == nil {
		return SurfaceRef{}, fmt.Errorf("surface reference is required")
	}
	target, err := TargetRefFromProto(r.Target)
	if err != nil {
		return SurfaceRef{}, err
	}
	result := SurfaceRef{Target: target, OwnerScenario: r.OwnerScenario, SurfaceID: r.SurfaceId}
	return result, result.Validate()
}

func (r SessionRef) Proto() *commonv1.SessionRef {
	return &commonv1.SessionRef{Surface: r.Surface.Proto(), SessionId: r.SessionID, DesktopSessionId: r.DesktopSessionID}
}

func SessionRefFromProto(r *commonv1.SessionRef) (SessionRef, error) {
	if r == nil {
		return SessionRef{}, fmt.Errorf("session reference is required")
	}
	surface, err := SurfaceRefFromProto(r.Surface)
	if err != nil {
		return SessionRef{}, err
	}
	result := SessionRef{Surface: surface, SessionID: r.SessionId, DesktopSessionID: r.DesktopSessionId}
	return result, result.Validate()
}

var kindToProto = map[SurfaceKind]commonv1.SurfaceKind{
	SurfaceScenario:    commonv1.SurfaceKind_SURFACE_KIND_SCENARIO,
	SurfaceTerminal:    commonv1.SurfaceKind_SURFACE_KIND_TERMINAL,
	SurfaceDesktop:     commonv1.SurfaceKind_SURFACE_KIND_DESKTOP,
	SurfaceBrowser:     commonv1.SurfaceKind_SURFACE_KIND_BROWSER,
	SurfaceDevicePanel: commonv1.SurfaceKind_SURFACE_KIND_DEVICE_PANEL,
}

var stateToProto = map[CapabilityState]commonv1.SurfaceCapabilityState{
	CapabilityReady:       commonv1.SurfaceCapabilityState_SURFACE_CAPABILITY_STATE_READY,
	CapabilityMissing:     commonv1.SurfaceCapabilityState_SURFACE_CAPABILITY_STATE_MISSING,
	CapabilityUnsupported: commonv1.SurfaceCapabilityState_SURFACE_CAPABILITY_STATE_UNSUPPORTED,
	CapabilityUnknown:     commonv1.SurfaceCapabilityState_SURFACE_CAPABILITY_STATE_UNKNOWN,
	CapabilityDenied:      commonv1.SurfaceCapabilityState_SURFACE_CAPABILITY_STATE_DENIED,
}

func (d SurfaceDescriptor) Proto() (*commonv1.SurfaceDescriptor, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	result := &commonv1.SurfaceDescriptor{
		Ref: d.Ref.Proto(), Kind: kindToProto[d.Kind], DisplayLabel: d.DisplayLabel,
		ProtocolVersions: append([]string{}, d.ProtocolVersions...),
		DesktopSessionId: d.DesktopSessionID, DisplayIds: append([]string{}, d.DisplayIDs...),
	}
	for _, fact := range d.Capabilities {
		observed, expires := timestamppb.New(fact.ObservedAt), timestamppb.New(fact.ExpiresAt)
		if observed.CheckValid() != nil || expires.CheckValid() != nil {
			return nil, fmt.Errorf("capability timestamp is outside the wire range")
		}
		result.Capabilities = append(result.Capabilities, &commonv1.SurfaceCapabilityFact{
			Capability: fact.Capability, State: stateToProto[fact.State], ReasonCode: fact.ReasonCode,
			EvidenceId: fact.EvidenceID, ObservedAt: observed, ExpiresAt: expires,
		})
	}
	return result, nil
}

func SurfaceDescriptorFromProto(d *commonv1.SurfaceDescriptor) (SurfaceDescriptor, error) {
	if d == nil {
		return SurfaceDescriptor{}, fmt.Errorf("surface descriptor is required")
	}
	ref, err := SurfaceRefFromProto(d.Ref)
	if err != nil {
		return SurfaceDescriptor{}, err
	}
	result := SurfaceDescriptor{
		Ref: ref, DisplayLabel: d.DisplayLabel, ProtocolVersions: append([]string{}, d.ProtocolVersions...),
		DesktopSessionID: d.DesktopSessionId, DisplayIDs: append([]string{}, d.DisplayIds...),
	}
	for kind, wire := range kindToProto {
		if d.Kind == wire {
			result.Kind = kind
			break
		}
	}
	if len(d.Capabilities) > 64 || len(d.ProtocolVersions) > 16 || len(d.DisplayIds) > 32 {
		return SurfaceDescriptor{}, fmt.Errorf("surface descriptor exceeds collection bounds")
	}
	for _, fact := range d.Capabilities {
		if fact == nil || fact.ObservedAt.CheckValid() != nil || fact.ExpiresAt.CheckValid() != nil {
			return SurfaceDescriptor{}, fmt.Errorf("capability timestamps are required and must be valid")
		}
		converted := CapabilityFact{
			Capability: fact.Capability, ReasonCode: fact.ReasonCode, EvidenceID: fact.EvidenceId,
			ObservedAt: fact.ObservedAt.AsTime(), ExpiresAt: fact.ExpiresAt.AsTime(),
		}
		for state, wire := range stateToProto {
			if fact.State == wire {
				converted.State = state
				break
			}
		}
		result.Capabilities = append(result.Capabilities, converted)
	}
	if err := result.Validate(); err != nil {
		return SurfaceDescriptor{}, err
	}
	return result, nil
}
