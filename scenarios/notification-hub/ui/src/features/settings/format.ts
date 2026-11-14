/** Locale-aware formatting shared by the settings summaries and editors. */

/** The language to format with: `cimode` and unknown tags fall back to English. */
function formatLocale(lang: string): string {
  try {
    return Intl.DateTimeFormat.supportedLocalesOf(lang).length > 0 ? lang : "en";
  } catch {
    return "en";
  }
}

/** A 24-hour `HH:MM` value as the reader's clock shows it ("10:00 PM", "22:00"). */
export function formatClock(hhmm: string, lang: string): string {
  const [hours, minutes] = hhmm.split(":").map(Number);
  if (hours === undefined || minutes === undefined || Number.isNaN(hours) || Number.isNaN(minutes)) return hhmm;
  return new Intl.DateTimeFormat(formatLocale(lang), { hour: "numeric", minute: "2-digit", timeZone: "UTC" })
    .format(new Date(Date.UTC(2024, 0, 1, hours, minutes)));
}

/** A weekday (0 = Sunday) as a name in the reader's language. */
export function weekdayName(weekday: number, lang: string, style: "long" | "short"): string {
  // 7 January 2024 was a Sunday.
  return new Intl.DateTimeFormat(formatLocale(lang), { weekday: style, timeZone: "UTC" })
    .format(new Date(Date.UTC(2024, 0, 7 + weekday)));
}

/** "A, B and C" in the reader's language. */
export function formatList(items: readonly string[], lang: string): string {
  return new Intl.ListFormat(formatLocale(lang), { style: "long", type: "conjunction" }).format(items);
}

/** Time zones the device can name, with any extra zones that must stay selectable. */
export function timeZoneChoices(...extra: string[]): string[] {
  const zones = new Set<string>();
  // Not in every TypeScript lib or engine; the extras below keep the field usable without it.
  const supportedValuesOf = (Intl as unknown as { supportedValuesOf?: (key: "timeZone") => string[] }).supportedValuesOf;
  try {
    for (const zone of supportedValuesOf?.("timeZone") ?? []) zones.add(zone);
  } catch {
    // An engine that refuses the key is treated like one without the list.
  }
  zones.add("UTC");
  for (const zone of extra) if (zone) zones.add(zone);
  return [...zones].sort((a, b) => a.localeCompare(b));
}

/** The time zone this device is set to, or UTC when it will not say. */
export function deviceTimeZone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
  } catch {
    return "UTC";
  }
}
