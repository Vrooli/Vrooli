/** @vrooliComponentSource hooks.use-locale */

export function useLocale() {
  return typeof document !== "undefined"
    ? document.documentElement.lang || "en"
    : "en";
}
