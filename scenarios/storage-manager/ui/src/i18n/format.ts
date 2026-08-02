/**
 * Locale-aware formatters built on the platform `Intl` APIs.
 *
 * Always use these helpers instead of `Date.prototype.toLocaleString()` or
 * raw `Intl.NumberFormat` calls so number, date, currency, and relative-time
 * output follow the user's chosen locale automatically — even from non-React
 * code paths that don't have access to `useTranslation()`.
 *
 * The current locale is read from the i18next singleton at call time, so
 * formatters always reflect the *latest* `setLocale(...)` choice without
 * needing a re-render or React context.
 *
 * All helpers fall back gracefully: passing an unrecognised locale string
 * (e.g. test pseudo-locales like `cimode`) lets `Intl` itself fall back to
 * the runtime default rather than throwing.
 */
import { i18n } from "./index";

/**
 * Resolve the locale to pass to `Intl.*` constructors. We strip non-BCP-47
 * pseudo-locales (i18next's `cimode` / `dev`) so tests don't trip the
 * `RangeError: Incorrect locale information provided` thrown by older
 * runtimes when given unknown tags.
 */
const resolveIntlLocale = (override?: string): string | undefined => {
  const candidate = override ?? i18n.language;
  if (!candidate) return undefined;
  if (candidate === "cimode" || candidate === "dev") return undefined;
  return candidate;
};

export const formatDate = (
  value: Date | number,
  options?: Intl.DateTimeFormatOptions,
  localeOverride?: string,
): string =>
  new Intl.DateTimeFormat(resolveIntlLocale(localeOverride), options).format(value);

export const formatNumber = (
  value: number,
  options?: Intl.NumberFormatOptions,
  localeOverride?: string,
): string =>
  new Intl.NumberFormat(resolveIntlLocale(localeOverride), options).format(value);

export const formatCurrency = (
  value: number,
  currency: string,
  options?: Intl.NumberFormatOptions,
  localeOverride?: string,
): string =>
  new Intl.NumberFormat(resolveIntlLocale(localeOverride), {
    style: "currency",
    currency,
    ...options,
  }).format(value);

export const formatRelativeTime = (
  value: number,
  unit: Intl.RelativeTimeFormatUnit,
  options?: Intl.RelativeTimeFormatOptions,
  localeOverride?: string,
): string =>
  new Intl.RelativeTimeFormat(resolveIntlLocale(localeOverride), options).format(value, unit);

/**
 * Format a list of strings using locale-appropriate conjunctions ("a, b, and c"
 * vs "a、b、c"). Uses `Intl.ListFormat`; safe to call with an empty array.
 */
export const formatList = (
  items: readonly string[],
  options?: Intl.ListFormatOptions,
  localeOverride?: string,
): string =>
  new Intl.ListFormat(resolveIntlLocale(localeOverride), options).format(items);
