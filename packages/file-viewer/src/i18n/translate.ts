import type { LocaleMessages, MessageOverrides, StringKey } from "./catalog.js";
import { ar } from "./locales/ar.js";
import { en } from "./locales/en.js";
import { ja } from "./locales/ja.js";

export type TranslateParams = Readonly<Record<string, string | number | undefined>>;
export type Translate = (key: StringKey, params?: TranslateParams) => string;

/** Host adapter consulted before the package catalogs; return undefined to fall through. */
export type TranslateAdapter = (key: StringKey, params: TranslateParams | undefined) => string | undefined;

/** The pseudo-locale whose output is the key itself, so tests can assert structure instead of copy. */
export const KEYS_LOCALE = "keys";

const CATALOGS: Readonly<Record<string, LocaleMessages>> = { en, ar, ja };
const RTL_LANGUAGES: ReadonlySet<string> = new Set(["ar", "he", "fa", "ur"]);
const PLURAL_CATEGORIES = new Set(["zero", "one", "two", "few", "many", "other"]);

/** "ar-SA" -> "ar"; the language subtag decides catalog and direction. */
export function languageOf(locale: string): string {
  return (locale.split(/[-_]/)[0] ?? "en").toLowerCase();
}

/** The locale to hand to `Intl`: the pseudo-locale formats as English. */
export function formatLocaleOf(locale: string): string {
  return locale === KEYS_LOCALE || !locale ? "en" : locale;
}

export function directionOf(locale: string): "ltr" | "rtl" {
  return RTL_LANGUAGES.has(languageOf(locale)) ? "rtl" : "ltr";
}

function catalogFor(locale: string): LocaleMessages {
  return CATALOGS[locale] ?? CATALOGS[languageOf(locale)] ?? en;
}

function lookup(messages: LocaleMessages, key: string): string | undefined {
  let node: string | LocaleMessages | undefined = messages;
  for (const segment of key.split(".")) {
    if (node === undefined || typeof node === "string") return undefined;
    node = node[segment];
  }
  return typeof node === "string" ? node : undefined;
}

function pluralKey(key: string, locale: string, count: number): string {
  let category = "other";
  try { category = new Intl.PluralRules(formatLocaleOf(locale)).select(count); } catch { /* Unknown locale tag: use the base key. */ }
  return PLURAL_CATEGORIES.has(category) ? `${key}_${category}` : key;
}

function interpolate(template: string, params: TranslateParams | undefined): string {
  if (!params) return template;
  return template.replace(/\{\{\s*([\w.]+)\s*\}\}/g, (match, name: string) => {
    const value = params[name];
    return value === undefined ? match : String(value);
  });
}

export interface TranslatorOptions {
  overrides?: MessageOverrides;
  adapter?: TranslateAdapter;
}

/**
 * Build `t` for a locale. Resolution order: adapter, overrides, the locale's
 * catalog (exact tag, then language), English, and finally the key itself.
 * A `count` parameter selects a plural variant (`key_one`, `key_other`, ...)
 * by `Intl.PluralRules`, falling back to the base key.
 */
export function createTranslator(locale: string, options: TranslatorOptions = {}): Translate {
  if (locale === KEYS_LOCALE) return (key) => key;
  const overrides = options.overrides as Readonly<Record<string, string | undefined>> | undefined;
  const primary = catalogFor(locale);
  return (key, params) => {
    const adapted = options.adapter?.(key, params);
    if (adapted !== undefined) return adapted;
    const candidates = typeof params?.count === "number" ? [pluralKey(key, locale, params.count), key] : [key];
    for (const candidate of candidates) {
      const text = overrides?.[candidate] ?? lookup(primary, candidate) ?? lookup(en, candidate);
      if (text !== undefined) return interpolate(text, params);
    }
    return key;
  };
}
