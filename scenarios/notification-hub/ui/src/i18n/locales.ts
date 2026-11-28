/**
 * Side-effect-free locale code list.
 *
 * Lives in its own module (rather than inline in `i18n/index.ts`) so that
 * import-cheap modules — notably `consts/selectors.ts` — can pick up the
 * supported-locale tuple without pulling in i18next initialisation.
 *
 * `LOCALE_CODES` is a `readonly` tuple so consumers can derive narrow
 * literal-union types from it (`Locale = (typeof LOCALE_CODES)[number]`).
 * Labels and text direction live here too, so UI initialization and bidi
 * formatting share the same lightweight, side-effect-free metadata.
 *
 * Adding a locale: add its metadata here and a matching `<code>/` catalog
 * directory. The locales-parity test verifies the catalog directories.
 */
export const LOCALE_CODES = ["en", "ja", "ar"] as const;

export type Locale = (typeof LOCALE_CODES)[number];

export interface LocaleConfig {
  /** Native-language label shown in switchers; never translated. */
  nativeLabel: string;
  /** HTML text direction. */
  dir: "ltr" | "rtl";
}

export const LOCALE_CONFIG = {
  en: { nativeLabel: "English", dir: "ltr" },
  ja: { nativeLabel: "日本語", dir: "ltr" },
  ar: { nativeLabel: "العربية", dir: "rtl" },
} as const satisfies Record<Locale, LocaleConfig>;
