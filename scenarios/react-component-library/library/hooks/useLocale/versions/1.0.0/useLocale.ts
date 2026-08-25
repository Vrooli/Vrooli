/** @vrooliComponentSource hooks.use-locale */

export function useLocale() {
  return typeof document !== "undefined"
    ? document.documentElement.lang || "en"
    : "en";
}

/**
 * Resolve library-owned copy through the adopting host's locale bridge. The
 * fallback is used only when the host has not installed a translator, which
 * keeps previews deterministic while preserving the adoption seam.
 */
export function translate(key: string, fallback: string): string {
  const host = globalThis as typeof globalThis & {
    __vrooliTranslate?: (key: string, fallback: string) => string;
  };
  return host.__vrooliTranslate?.(key, fallback) ?? fallback;
}
