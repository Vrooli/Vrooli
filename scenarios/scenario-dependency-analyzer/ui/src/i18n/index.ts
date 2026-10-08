import i18n from "i18next";
import { initReactI18next, useTranslation } from "react-i18next";

import en from "./locales/en.json";

export const defaultLocale = "en";

void i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en }
  },
  lng: defaultLocale,
  fallbackLng: defaultLocale,
  interpolation: {
    escapeValue: false
  },
  returnNull: false
});

export function translate(key: string): string {
  return i18n.t(key);
}

export { i18n, useTranslation };
