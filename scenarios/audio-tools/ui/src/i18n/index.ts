/**
 * i18n initialization
 *
 * Wires i18next + react-i18next with bundled English and on-demand locale resources. Components
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
 *   2. Append the code to `LOCALE_CODES` in `./locales.ts`.
 *   3. Add the matching entry to `LOCALE_CONFIG` below (TypeScript will fail
 *      the build if you forget — the Record is keyed by `Locale`).
 *   4. Add a loader case below so non-English catalogs remain route/runtime chunks.
 *   5. The language switcher reads `LOCALE_CONFIG` directly — no UI changes.
 *      The locales-parity test (`./locales/locales.test.ts`) verifies that
 *      `LOCALE_CODES`, `LOCALE_CONFIG`, and the catalog files all agree.
 */
import i18n from "i18next";
import { initReactI18next, useTranslation } from "react-i18next";
import { LOCALE_CODES, type Locale } from "./locales";
import en from "./locales/en.json";

interface LocaleConfig {
  /** Native-language label shown in switchers; never translated. */
  nativeLabel: string;
  /** HTML text direction. Set up now so adding RTL locales is a config change, not a code change. */
  dir: "ltr" | "rtl";
}

// `LOCALE_CODES` (in ./locales) is the canonical list of supported locales.
// `LOCALE_CONFIG` is keyed against it so adding a code without an entry — or
// vice versa — is a TypeScript error.
const LOCALE_CONFIG = {
  en: { nativeLabel: "English", dir: "ltr" },
  ja: { nativeLabel: "日本語", dir: "ltr" },
  ar: { nativeLabel: "العربية", dir: "rtl" },
} as const satisfies Record<Locale, LocaleConfig>;

// Re-export so callers that already import { Locale } from "./i18n" keep working.
export { LOCALE_CODES, type Locale };
export const SUPPORTED_LOCALES = LOCALE_CODES;

const STORAGE_KEY = "vrooli.locale";

const isSupported = (lng: string | null | undefined): lng is Locale =>
  Boolean(lng && (SUPPORTED_LOCALES as readonly string[]).includes(lng));

// Storage can be disabled by browser privacy settings or sandbox policy.
// Locale selection should still work for the current page in that case.
const readStoredLocale = (): string | null => {
  try {
    return window.localStorage.getItem(STORAGE_KEY);
  } catch {
    return null;
  }
};

const persistLocale = (lng: string) => {
  try {
    window.localStorage.setItem(STORAGE_KEY, lng);
  } catch {
    // Keep the selected locale in memory when storage is unavailable.
  }
};

const detectInitialLocale = (): Locale => {
  if (typeof window === "undefined") return "en";
  const stored = readStoredLocale();
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

i18n.on("languageChanged", (lng) => {
  if (typeof window !== "undefined" && isSupported(lng)) {
    persistLocale(lng);
  }
  applyDocumentLocale(lng);
});

const loadCatalog = async (lng: Locale): Promise<typeof en> => {
  switch (lng) {
    case "ja": return (await import("./locales/ja.json")).default;
    case "ar": return (await import("./locales/ar.json")).default;
    default: return en;
  }
};

export const i18nReady = (async () => {
  let lng = detectInitialLocale();
  let initialCatalog = en;
  if (lng !== "en") {
    try {
      initialCatalog = await loadCatalog(lng);
    } catch {
      // Keep startup usable if an optional locale chunk is unavailable.
      lng = "en";
    }
  }
  await i18n.use(initReactI18next).init({
    resources: {
      en: { translation: en },
      [lng]: { translation: initialCatalog },
    },
    lng,
    fallbackLng: "en",
    // React already escapes interpolated values; double-escaping mangles output.
    interpolation: { escapeValue: false },
    returnNull: false,
  });
  applyDocumentLocale(lng);
})();

export const setLocale = async (lng: Locale): Promise<void> => {
  await i18nReady;
  if (!i18n.hasResourceBundle(lng, "translation")) {
    try {
      i18n.addResourceBundle(lng, "translation", await loadCatalog(lng), true, true);
    } catch {
      // Keep the current language if the optional locale chunk cannot load.
      return;
    }
  }
  await i18n.changeLanguage(lng);
};

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
