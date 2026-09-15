/**
 * Vrooli string registry — typed key paths
 *
 * Single source of truth for every user-facing string key in the UI. The
 * runtime values mirror the structure of `src/i18n/locales/en.json` and
 * each leaf is the dotted *key path* that gets passed to `t()`.
 *
 * Why this shape:
 *   1. Type safety: `strings.app.titel` is a compile-time error.
 *   2. Refactor safety: Renaming a key in en.json triggers TypeScript errors.
 *   3. Discoverability: "Where is this string referenced?" is one grep.
 *   4. Test stability: Tests compare keys, not translated copy.
 *
 * Do not edit `strings.generated.ts` by hand — it is regenerated from
 * `i18n/locales/en.json` by `scripts/gen-strings.mjs` (invoked automatically
 * by the Vite plugin on dev start, HMR of en.json, and build start; also
 * available as `pnpm strings:gen` and `pnpm strings:check`).
 */
export { strings, type Strings } from "./strings.generated";
