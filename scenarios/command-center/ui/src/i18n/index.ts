import i18n from "i18next";
import { initReactI18next, useTranslation } from "react-i18next";

import en from "./locales/en.json";

// The board authors its own copy in en.json. React Component Library strings
// resolve through this same instance with the caller's defaultValue as the
// fallback, so LibraryStringsProvider is given a real translator rather than a
// passthrough shim — which is what the adoption contract requires.
export const defaultLocale = "en";

void i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
  },
  lng: defaultLocale,
  fallbackLng: defaultLocale,
  // React already escapes interpolated values; double-escaping mangles output.
  interpolation: { escapeValue: false },
  returnNull: false,
});

export { i18n, useTranslation };
