import { describe, expect, it } from "vitest";
import { create, toJson } from "@bufbuild/protobuf";
import {
  EventSchema,
  EventType,
  PairingRequestSchema,
} from "@vrooli/proto-types/device-sync-hub/v1/realtime/realtime_pb";

import {
  decodeEvent,
  initialRealtimeState,
  realtimeReducer,
} from "./events";

const eventJson = (init: Parameters<typeof create<typeof EventSchema>>[1]) =>
  // Server emits proto-JSON with snake_case (UseProtoNames); toJson with
  // useProtoFieldName mirrors that wire shape for the decode round-trip.
  JSON.stringify(toJson(EventSchema, create(EventSchema, init), { useProtoFieldName: true }));

describe("decodeEvent", () => {
  it("decodes a proto-JSON presence event (snake_case field names)", () => {
    const line = eventJson({
      type: EventType.PRESENCE_CHANGED,
      presence: [{ deviceId: "dev-a", online: true }],
    });
    const event = decodeEvent(line);
    expect(event?.type).toBe(EventType.PRESENCE_CHANGED);
    expect(event?.presence[0]?.deviceId).toBe("dev-a");
  });

  it("returns null for a non-JSON heartbeat/comment line", () => {
    expect(decodeEvent(":keep-alive")).toBeNull();
  });
});

describe("realtimeReducer", () => {
  it("replaces presence on PRESENCE_CHANGED", () => {
    const event = create(EventSchema, {
      type: EventType.PRESENCE_CHANGED,
      presence: [
        { deviceId: "a", online: true },
        { deviceId: "b", online: false },
      ],
    });
    const next = realtimeReducer(initialRealtimeState, { type: "event", event });
    expect(next.presence).toEqual({ a: true, b: false });
  });

  it("captures the pending pairing on PAIRING_REQUESTED", () => {
    const event = create(EventSchema, {
      type: EventType.PAIRING_REQUESTED,
      pairing: { deviceId: "p1", name: "Phone", kind: "phone" },
    });
    const next = realtimeReducer(initialRealtimeState, { type: "event", event });
    expect(next.pendingPairing?.deviceId).toBe("p1");
  });

  it("clears the pending pairing on dismiss", () => {
    const seeded = {
      ...initialRealtimeState,
      pendingPairing: create(PairingRequestSchema, { deviceId: "p1", name: "P", kind: "phone" }),
    };
    const next = realtimeReducer(seeded, { type: "dismiss-pairing" });
    expect(next.pendingPairing).toBeNull();
  });

  it("leaves state untouched for item events (handled via query invalidation)", () => {
    const event = create(EventSchema, {
      type: EventType.ITEM_ARRIVED,
      item: { id: "i1" },
    });
    const next = realtimeReducer(initialRealtimeState, { type: "event", event });
    expect(next).toBe(initialRealtimeState);
  });
});
