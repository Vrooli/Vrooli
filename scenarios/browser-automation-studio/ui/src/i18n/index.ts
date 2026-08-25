export {};

const i18n = { t: (key: string, options?: { defaultValue?: string }) => options?.defaultValue ?? key };

declare global {
  var __vrooliTranslate: (key: string, fallback: string) => string;
}

// vrooli:library-locale-bridge start
const vrooliLibraryTranslate = (key: string, fallback: string): string => i18n.t(key, { defaultValue: fallback });
globalThis.__vrooliTranslate = vrooliLibraryTranslate;
// vrooli:library-locale-bridge end
