import { LOCALE_CONFIG, type Locale } from "./locales";

/**
 * Isolation for values placed inside a translated sentence.
 *
 * A requester's words ("Set B, keep the character") dropped into an Arabic
 * sentence, or an Arabic name into an English one, take part in the sentence's
 * bidirectional reordering: punctuation jumps to the wrong side and neighbouring
 * words swap. Unicode first-strong isolates (FSI ... PDI, U+2068 and U+2069)
 * give the value its own direction and keep the sentence intact.
 *
 * Only values that run against the sentence are wrapped: Latin letters inside a
 * right-to-left locale, right-to-left letters inside a left-to-right one. A
 * value in the sentence's own script, a number or an id is left untouched, so
 * ordinary strings (and the text tests compare) stay exactly as written.
 */
const FSI = "\u2068";
const PDI = "\u2069";

const LATIN_LETTER = /[A-Za-z\u00C0-\u024F]/;
const RTL_LETTER = /[\u0590-\u08FF\uFB1D-\uFDFF\uFE70-\uFEFF]/;

const RTL_LOCALES: ReadonlySet<string> = new Set(
  (Object.keys(LOCALE_CONFIG) as Locale[]).filter((locale) => LOCALE_CONFIG[locale].dir === "rtl"),
);

/** i18next `interpolation.format` (with `alwaysFormat`): stringify, isolating a value that opposes the sentence. */
export function isolateInterpolation(value: unknown, _format?: string, lng?: string): string {
  if (typeof value !== "string") return String(value);
  const rtlSentence = RTL_LOCALES.has((lng ?? "").split("-")[0] ?? "");
  const opposes = rtlSentence ? LATIN_LETTER.test(value) : RTL_LETTER.test(value);
  return opposes ? `${FSI}${value}${PDI}` : value;
}

