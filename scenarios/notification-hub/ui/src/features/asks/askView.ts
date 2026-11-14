import type {
  Ask,
  AskOption,
} from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

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
