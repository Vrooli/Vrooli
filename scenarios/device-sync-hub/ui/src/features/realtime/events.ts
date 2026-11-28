import { fromJson, type JsonValue } from "@bufbuild/protobuf";
import {
  EventSchema,
  EventType,
  type Event,
  type PairingRequest,
} from "@vrooli/proto-types/device-sync-hub/v1/realtime/realtime_pb";

/**
 * Decode one SSE `data:` line into a typed realtime Event. The server emits
 * proto-JSON with `UseProtoNames` (snake_case field names); protobuf-es
 * `fromJson` accepts both proto and JSON field names, and `ignoreUnknownFields`
 * keeps an additive server change from breaking an older client. Returns null
 * for anything unparseable so a stray heartbeat / comment line is ignored.
 */
export function decodeEvent(line: string): Event | null {
  try {
    const json = JSON.parse(line) as JsonValue;
    return fromJson(EventSchema, json, { ignoreUnknownFields: true });
  } catch {
    return null;
  }
}

/** Online-state map keyed by device id, derived from PRESENCE_CHANGED events. */
export type PresenceMap = Readonly<Record<string, boolean>>;

/**
 * The slice of realtime state the UI renders directly. Item events don't live
 * here — they invalidate the react-query items cache instead — but presence and
 * the pending pairing request are pushed state the components read.
 */
export interface RealtimeState {
  presence: PresenceMap;
  pendingPairing: PairingRequest | null;
}

export const initialRealtimeState: RealtimeState = {
  presence: {},
  pendingPairing: null,
};

export type RealtimeAction =
  | { type: "event"; event: Event }
  | { type: "dismiss-pairing" };

/**
 * Pure reducer for realtime state. Kept separate from the hook so the routing
 * logic (which event mutates which slice) is unit-testable without a live
 * EventSource. ITEM_* events are intentionally inert here — the hook handles
 * them by invalidating the items query.
 */
export function realtimeReducer(state: RealtimeState, action: RealtimeAction): RealtimeState {
  if (action.type === "dismiss-pairing") {
    return state.pendingPairing ? { ...state, pendingPairing: null } : state;
  }
  const { event } = action;
  switch (event.type) {
    case EventType.PRESENCE_CHANGED: {
      const presence: Record<string, boolean> = {};
      for (const entry of event.presence) {
        presence[entry.deviceId] = entry.online;
      }
      return { ...state, presence };
    }
    case EventType.PAIRING_REQUESTED:
      return event.pairing ? { ...state, pendingPairing: event.pairing } : state;
    default:
      return state;
  }
}

export { EventType };
export type { Event, PairingRequest };
