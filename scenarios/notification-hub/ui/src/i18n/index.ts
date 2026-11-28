/**
 * i18n initialization
 *
 * Wires i18next + react-i18next with bundled JSON resources. Components
 * consume strings through `useTranslation()` and the typed key registry
 * in `src/consts/strings.ts`. Non-component code can use the exported
 * `i18n` singleton directly (`i18n.t(strings.x.y)`).
 *
 * Locale resolution order on first load:
 *   1. localStorage (returning visitor's last choice)
 *   2. navigator.language primary subtag (browser preference)
 *   3. "en" (fallback)
 *
 * On every `languageChanged` event we mirror the choice into `<html lang>`,
 * `<html dir>`, and localStorage so SSR-style consumers, RTL CSS, and
 * returning visits all stay in sync.
 *
 * Adding a locale:
 *   1. Add the locale's code, label, and direction to `./locales.ts`.
 *   2. Add a `./locales/<code>/` directory with the same namespace files as `en/`.
 *   3. Export the assembled catalog from `./catalog.ts` and register it in the
 *      `resources` block below.
 *   4. The language switcher reads `LOCALE_CONFIG` directly — no UI changes.
 *      The locales-parity test (`./locales/locales.test.ts`) verifies catalog
 *      directory parity; TypeScript checks metadata against `LOCALE_CODES`.
 */
import i18n from "i18next";
import { initReactI18next, useTranslation } from "react-i18next";
import { LOCALE_CODES, LOCALE_CONFIG, type Locale, type LocaleConfig } from "./locales";
import { readLocalPreference, writeLocalPreference } from "../theme/preferenceStorage";
import { isolateInterpolation } from "./bidi";
import { ar, en, ja } from "./catalog";

// Re-export so callers that already import { Locale } from "./i18n" keep working.
export { LOCALE_CODES, LOCALE_CONFIG, type Locale, type LocaleConfig };
export const SUPPORTED_LOCALES = LOCALE_CODES;

const STORAGE_KEY = "vrooli.locale";

const isSupported = (lng: string | null | undefined): lng is Locale =>
  Boolean(lng && (SUPPORTED_LOCALES as readonly string[]).includes(lng));

export const detectInitialLocale = (): Locale => {
  if (typeof window === "undefined") return "en";
  const stored = readLocalPreference("locale", STORAGE_KEY);
  if (isSupported(stored)) return stored;
  const primary = window.navigator.language.split("-")[0]?.toLowerCase();
  return isSupported(primary) ? primary : "en";
};

const applyDocumentLocale = (lng: string) => {
  if (typeof document === "undefined") return;
  const config = isSupported(lng) ? LOCALE_CONFIG[lng] : LOCALE_CONFIG.en;
  document.documentElement.lang = lng;
  document.documentElement.dir = config.dir;
};

void i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    ja: { translation: ja },
    ar: { translation: ar },
  },
  lng: detectInitialLocale(),
  fallbackLng: "en",
  // React already escapes interpolated values; double-escaping mangles output.
  // Interpolated values that run against the sentence's script get bidi isolation.
  interpolation: { escapeValue: false, format: isolateInterpolation, alwaysFormat: true },
  returnNull: false,
});

i18n.on("languageChanged", (lng) => {
  if (typeof window !== "undefined" && isSupported(lng)) {
    writeLocalPreference("locale", STORAGE_KEY, lng);
  }
  applyDocumentLocale(lng);
});

applyDocumentLocale(i18n.language);

export const setLocale = (lng: Locale): Promise<void> =>
  i18n.changeLanguage(lng).then(() => undefined);

export const getLocaleConfig = (lng: Locale): LocaleConfig => LOCALE_CONFIG[lng];

/**
 * Resolve the active locale as a typed `Locale`. `i18n.language` at runtime can
 * be `cimode` (test pseudo-locale), a region-tagged code like `en-US`, or a
 * fallback — all of which are typed as `string`. Use this helper anywhere the
 * caller needs to compare against `SUPPORTED_LOCALES` or index `LOCALE_CONFIG`.
 */
export const getCurrentLocale = (): Locale =>
  isSupported(i18n.language) ? i18n.language : "en";

export { i18n, useTranslation };
