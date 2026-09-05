import en from "./locales/en.json";

type TranslationOptions = { defaultValue?: string };

const translations = en as Record<string, string>;

/** Minimal local translator used by the RCL library-string bridge. */
export const i18n = {
  t(key: string, options?: TranslationOptions): string {
    return translations[key] ?? options?.defaultValue ?? key;
  },
};
