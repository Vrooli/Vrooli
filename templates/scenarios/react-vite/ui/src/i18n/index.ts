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
 *   1. Drop `./locales/<code>.json` next to en.json (same shape).
 *   2. Add the code to `SUPPORTED_LOCALES` and an entry to `LOCALE_CONFIG`.
 *   3. Import + register it in the `resources` block below.
 *   4. The language switcher reads `LOCALE_CONFIG` directly — no UI changes.
 */
import i18n from "i18next";
import { initReactI18next, useTranslation } from "react-i18next";
import en from "./locales/en.json";
import ja from "./locales/ja.json";

export const SUPPORTED_LOCALES = ["en", "ja"] as const;
export type Locale = (typeof SUPPORTED_LOCALES)[number];

interface LocaleConfig {
  /** Native-language label shown in switchers; never translated. */
  nativeLabel: string;
  /** HTML text direction. Set up now so adding RTL locales is a config change, not a code change. */
  dir: "ltr" | "rtl";
}

const LOCALE_CONFIG: Record<Locale, LocaleConfig> = {
  en: { nativeLabel: "English", dir: "ltr" },
  ja: { nativeLabel: "日本語", dir: "ltr" },
};

const STORAGE_KEY = "vrooli.locale";

const isSupported = (lng: string | null | undefined): lng is Locale =>
  Boolean(lng && (SUPPORTED_LOCALES as readonly string[]).includes(lng));

const detectInitialLocale = (): Locale => {
  if (typeof window === "undefined") return "en";
  const stored = window.localStorage.getItem(STORAGE_KEY);
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
  },
  lng: detectInitialLocale(),
  fallbackLng: "en",
  // React already escapes interpolated values; double-escaping mangles output.
  interpolation: { escapeValue: false },
  returnNull: false,
});

i18n.on("languageChanged", (lng) => {
  if (typeof window !== "undefined" && isSupported(lng)) {
    window.localStorage.setItem(STORAGE_KEY, lng);
  }
  applyDocumentLocale(lng);
});

applyDocumentLocale(i18n.language);

export const setLocale = (lng: Locale): Promise<unknown> =>
  i18n.changeLanguage(lng);

export const getLocaleConfig = (lng: Locale): LocaleConfig => LOCALE_CONFIG[lng];

export { i18n, useTranslation };
