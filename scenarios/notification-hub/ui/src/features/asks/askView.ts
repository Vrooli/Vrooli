import type {
  Ask,
  AskOption,
} from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { strings } from "../../consts/strings";

/** The words for each urgency the Hub knows. The server sends the token; a screen shows this, never the raw value. */
export const URGENCY_LABEL = {
  low: strings.asks.urgency.low,
  normal: strings.asks.urgency.normal,
  high: strings.asks.urgency.high,
  critical: strings.asks.urgency.critical,
} as const;

/** The catalog key naming a decision's urgency when it is worth saying (not normal, not unknown), else null. */
export function urgencyLabel(urgency: string): (typeof URGENCY_LABEL)[Exclude<keyof typeof URGENCY_LABEL, "normal">] | null {
  if (urgency === "low" || urgency === "high" || urgency === "critical") return URGENCY_LABEL[urgency];
  return null;
}

/** Options in display order: the recommended answer first, then as asked. */
export function orderedOptions(
  ask: Pick<Ask, "options" | "recommended">,
): AskOption[] {
  const recommended = ask.options.filter(
    (option) => option.key === ask.recommended,
  );
  return [
    ...recommended,
    ...ask.options.filter((option) => option.key !== ask.recommended),
  ];
}

export function optionLabel(ask: Pick<Ask, "options">, key: string): string {
  return ask.options.find((option) => option.key === key)?.label || key;
}

/** "5 h 12 min" until `iso`, or null when it is unknown or already past. */
export function remainingUntil(iso: string, now: number): string | null {
  const target = Date.parse(iso);
  if (!iso || Number.isNaN(target) || target <= now) return null;
  const minutes = Math.ceil((target - now) / 60_000);
  const hours = Math.floor(minutes / 60);
  return hours > 0 ? `${hours} h ${minutes % 60} min` : `${minutes} min`;
}

/** Only same-origin or http(s) links are offered as "more detail". */
export function safeContextUrl(value: string): string | null {
  try {
    const url = new URL(value, window.location.origin);
    return url.protocol === "https:" || url.protocol === "http:"
      ? url.href
      : null;
  } catch {
    return null;
  }
}

/**
 * The host a link leaves this site for, or null when it stays on the Hub's own
 * origin. `URL.host` is the punycode form, so a look-alike name reads as the
 * address the browser will actually open, and it keeps a port (a loopback review
 * page is `localhost:3000`, not `localhost`).
 */
export function externalHost(href: string): string | null {
  try {
    const url = new URL(href, window.location.origin);
    return url.origin === window.location.origin ? null : url.host;
  } catch {
    return null;
  }
}

/** The line that names a decision in a list or a link: its short title, else the whole question. */
export function askHeadline(ask: Pick<Ask, "title" | "question">): string {
  return ask.title && ask.title !== "Decision required" ? ask.title : ask.question;
}

/** How loudly a waiting decision should read in a list. Settled decisions are never loud. */
export function askTone(
  ask: Pick<Ask, "state" | "urgency" | "deadline">,
  now: number,
): "normal" | "attention" | "critical" {
  if (ask.state !== "pending" && ask.state !== "escalated") return "normal";
  if (ask.state === "escalated" || ask.urgency === "critical") return "critical";
  const deadline = Date.parse(ask.deadline);
  const dueSoon = !Number.isNaN(deadline) && deadline - now < 24 * 60 * 60_000;
  return ask.urgency === "high" || dueSoon ? "attention" : "normal";
}

/** True while the operator still owes an answer. */
export function isAwaiting(ask: Pick<Ask, "state">): boolean {
  return ask.state === "pending" || ask.state === "escalated";
}

/** A one-line preview for a list row: the big picture with its Markdown marks removed. */
export function askPreview(ask: Pick<Ask, "bigPicture">): string {
  return ask.bigPicture
    .replace(/```[\s\S]*?```/g, " ")
    .replace(/!?\[([^\]]*)\]\([^)]*\)/g, "$1")
    .replace(/[#*_`>~]/g, "")
    .replace(/\s+/g, " ")
    .trim();
}
