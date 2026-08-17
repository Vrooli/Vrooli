/**
 * @vrooliComponentSource react-component-library:useLocale
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 5e22d9a4-a909-4a40-8ca2-f3092fa16ffa
 * @vrooliComponentAppliedAt 2026-08-17T08:31:55Z
 * @vrooliComponentSourceSha256 8dbc624b966f0ed254bcb1788b21d1baa836283c0c37f764f00a502e74757432
 * @vrooliComponentDriftHash 5989bb8de58b36603c8dfa71d40645803dafa384d8ff8f84aacdebf5785235e2
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
export function useLocale() {
  return typeof document !== "undefined"
    ? document.documentElement.lang || "en"
    : "en";
}
