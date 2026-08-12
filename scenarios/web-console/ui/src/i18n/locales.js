/**
 * Side-effect-free locale code list.
 *
 * Lives in its own module (rather than inline in `i18n/index.ts`) so that
 * import-cheap modules — notably `consts/selectors.ts` — can pick up the
 * supported-locale tuple without pulling in i18next initialisation.
 *
 * `LOCALE_CODES` is a `readonly` tuple so consumers can derive narrow
 * literal-union types from it (`Locale = (typeof LOCALE_CODES)[number]`).
 * `i18n/index.ts` re-exports `Locale` for backward compatibility.
 *
 * Adding a locale: append the new code here, then add the matching entry
 * to `LOCALE_CONFIG` in `i18n/index.ts` and drop a `<code>.json` next to
 * `en.json`. The locales-parity test asserts that all three stay in sync.
 */
export const LOCALE_CODES = ["en", "ja", "ar"];
