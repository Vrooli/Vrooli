import { NotificationState } from "@vrooli/proto-types/notification-hub/v1/shared/types_pb";

import { strings } from "../../consts/strings";

export type DeliveryStateKey = keyof typeof strings.pages.delivery.states;

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
