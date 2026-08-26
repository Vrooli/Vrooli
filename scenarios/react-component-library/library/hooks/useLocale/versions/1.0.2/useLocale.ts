/**
 * @libraryId react-component-library:useLocale
 * @displayName useLocale
 * @description Production-ready useLocale hook with SSR-safe lifecycle behavior.
 * @version 1.0.2
 * @tags ["runtime","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-locale */

declare global {
  var __vrooliTranslate: ((key: string, fallback: string) => string) | undefined;
}

export function useLocale() {
  return typeof document !== "undefined" ? document.documentElement.lang || "en" : "en";
}

export function translate(key: string, fallback: string): string {
  const bridge = globalThis.__vrooliTranslate;
  return typeof bridge === "function" ? bridge(key, fallback) : fallback;
}
