import { NotificationState } from "@vrooli/proto-types/notification-hub/v1/shared/types_pb";

import { strings } from "../../consts/strings";

/** Every transport state a notification can be in, in the order they are explained. */
export const DELIVERY_STATES = ["pending", "held", "routed", "delivered", "failed", "unroutable", "suppressed"] as const;
export type DeliveryStateKey = typeof DELIVERY_STATES[number] | "unknown";

export const DELIVERY_STATE_LABEL = {
  pending: strings.pages.delivery.states.pending,
  held: strings.pages.delivery.states.held,
  routed: strings.pages.delivery.states.routed,
  delivered: strings.pages.delivery.states.delivered,
  failed: strings.pages.delivery.states.failed,
  unroutable: strings.pages.delivery.states.unroutable,
  suppressed: strings.pages.delivery.states.suppressed,
  unknown: strings.pages.delivery.states.unknown,
} as const satisfies Record<DeliveryStateKey, string>;

export const DELIVERY_STATE_MEANING = {
  pending: strings.pages.delivery.meanings.pending,
  held: strings.pages.delivery.meanings.held,
  routed: strings.pages.delivery.meanings.routed,
  delivered: strings.pages.delivery.meanings.delivered,
  failed: strings.pages.delivery.meanings.failed,
  unroutable: strings.pages.delivery.meanings.unroutable,
  suppressed: strings.pages.delivery.meanings.suppressed,
  unknown: strings.pages.delivery.meanings.unknown,
} as const satisfies Record<DeliveryStateKey, string>;

/** How loudly each transport state reads: reached is calm, a failure to reach is not. */
export const DELIVERY_STATE_TONE = {
  pending: "info",
  held: "neutral",
  routed: "info",
  delivered: "success",
  failed: "danger",
  unroutable: "danger",
  suppressed: "neutral",
  unknown: "warning",
} as const satisfies Record<DeliveryStateKey, "neutral" | "success" | "warning" | "danger" | "info">;

/** States whose meaning is worth a line of its own, because the notification may not have reached the owner. */
export const DELIVERY_STATE_EXCEPTIONAL: ReadonlySet<DeliveryStateKey> = new Set(["held", "failed", "unroutable", "suppressed", "unknown"]);

/** One canonical mapping for the seven native transport states and unknown values. */
export function deliveryStateKey(state: NotificationState | number): DeliveryStateKey {
  switch (state) {
    case 1: return "pending";
    case 2: return "held";
    case 3: return "routed";
    case 4: return "delivered";
    case 5: return "failed";
    case 6: return "unroutable";
    case 7: return "suppressed";
    default: return "unknown";
  }
}

/** Render source timestamps as absolute local date, time, and time zone. */
export function formatDeliveryTime(value: string): string {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) return "";
  return new Intl.DateTimeFormat(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    timeZoneName: "short",
  }).format(date);
}
