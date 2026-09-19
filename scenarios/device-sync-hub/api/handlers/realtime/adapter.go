package realtime

import (
	internalrealtime "device-sync-hub/internal/realtime"

	realtimev1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/realtime"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// eventToProto converts a hub domain Event into the proto wire shape the SSE
// stream marshals. Mechanical, used only at the transport edge — the realtime
// hub never imports proto.
func eventToProto(e internalrealtime.Event) *realtimev1.Event {
	out := &realtimev1.Event{
		Type: kindToProto(e.Kind),
		At:   timestamppb.New(e.At.UTC()),
	}
	switch e.Kind {
	case internalrealtime.EventItemArrived, internalrealtime.EventItemDeleted:
		out.Item = &realtimev1.ItemRef{Id: e.ItemID, TargetDeviceId: e.TargetDeviceID}
	case internalrealtime.EventPresenceChanged:
		out.Presence = make([]*realtimev1.DevicePresence, 0, len(e.Presence))
		for _, p := range e.Presence {
			out.Presence = append(out.Presence, &realtimev1.DevicePresence{DeviceId: p.DeviceID, Online: p.Online})
		}
	case internalrealtime.EventPairingRequested:
		if e.Pairing != nil {
			out.Pairing = &realtimev1.PairingRequest{
				DeviceId: e.Pairing.DeviceID,
				Name:     e.Pairing.Name,
				Kind:     e.Pairing.Kind,
			}
		}
	}
	return out
}

func kindToProto(k internalrealtime.EventKind) realtimev1.EventType {
	switch k {
	case internalrealtime.EventItemArrived:
		return realtimev1.EventType_EVENT_TYPE_ITEM_ARRIVED
	case internalrealtime.EventItemDeleted:
		return realtimev1.EventType_EVENT_TYPE_ITEM_DELETED
	case internalrealtime.EventPresenceChanged:
		return realtimev1.EventType_EVENT_TYPE_PRESENCE_CHANGED
	case internalrealtime.EventPairingRequested:
		return realtimev1.EventType_EVENT_TYPE_PAIRING_REQUESTED
	default:
		return realtimev1.EventType_EVENT_TYPE_UNSPECIFIED
	}
}
