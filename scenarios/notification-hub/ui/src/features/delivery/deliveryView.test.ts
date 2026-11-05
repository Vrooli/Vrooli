import { describe, expect, it } from "vitest";

import { NotificationState } from "@vrooli/proto-types/notification-hub/v1/shared/types_pb";
import { deliveryStateKey, formatDeliveryTime } from "./deliveryView";

describe("delivery view mappings", () => {
  it("preserves each native notification state and does not turn unknown values into delivered", () => {
    expect([
      deliveryStateKey(NotificationState.PENDING),
      deliveryStateKey(NotificationState.HELD),
      deliveryStateKey(NotificationState.ROUTED),
      deliveryStateKey(NotificationState.DELIVERED),
      deliveryStateKey(NotificationState.FAILED),
      deliveryStateKey(NotificationState.UNROUTABLE),
      deliveryStateKey(NotificationState.SUPPRESSED),
    ]).toEqual(["pending", "held", "routed", "delivered", "failed", "unroutable", "suppressed"]);
    expect(deliveryStateKey(NotificationState.UNSPECIFIED)).toBe("unknown");
    expect(deliveryStateKey(99)).toBe("unknown");
  });

  it("does not invent a timestamp when the source value is empty or invalid", () => {
    expect(formatDeliveryTime("")).toBe("");
    expect(formatDeliveryTime("not-a-time")).toBe("");
    expect(formatDeliveryTime("2026-10-20T14:00:00Z")).toMatch(/2026/);
  });
});
