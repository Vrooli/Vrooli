/**
 * Vrooli string registry — typed key paths
 *
 * Single source of truth for every user-facing string key in the UI. The
 * runtime values mirror the structure of `src/i18n/locales/en.json` and
 * each leaf is the dotted *key path* that gets passed to `t()`.
 *
 * ```tsx
 * import { useTranslation } from "../i18n";
 * import { strings } from "../consts/strings";
 *
 * const { t } = useTranslation();
 * return <h1>{t(strings.app.title)}</h1>;
 * // strings.app.title === "app.title"
 * // t("app.title") === "Security Health" / "シナリオテンプレート" / …
 * ```
 *
 * Why this shape (instead of inlining `t("app.title")` directly):
 *
 * 1. **Type safety.** `strings.app.titel` is a compile-time error.
 *    `t("app.titel")` is a runtime fallback to the key string.
 * 2. **Refactor safety.** Renaming a key in en.json triggers TypeScript
 *    errors at every callsite once the registry is regenerated.
 * 3. **Discoverability.** "Where is this string referenced?" is one grep
 *    for `strings.feature.key`, not a fuzzy search through `t("…")` calls.
 * 4. **Test stability.** Tests can compare against the same key paths the
 *    component uses, without ever asserting on translated copy.
 *
 * ## Implementation
 *
 * The actual `strings` const lives in `strings.generated.ts`, produced by
 * `scripts/gen-strings.mjs`. The Vite plugin in
 * `scripts/vite-plugin-strings-codegen.mjs` runs the codegen on every dev
 * start, on HMR of en.json, and on every build start, so the file stays in
 * sync without manual intervention. Vitest reads the same Vite config, so
 * tests are covered too.
 *
 * Why a generated file rather than a runtime traversal of en.json (the
 * pre-Option-C implementation): walking the catalog at module load forces
 * the bundler to ship en.json twice — once as i18next's resource and once
 * as the input to the registry. At ~500 strings that adds tens of KB to
 * every initial download. Codegen makes en.json bundled exactly once.
 *
 * ## Adding a string
 *
 *   1. Add the key to `i18n/locales/en.json` (and every other locale —
 *      `locales.test.ts` enforces parity).
 *   2. If `pnpm dev` (or `vitest`) is running, the plugin regenerates
 *      instantly. Otherwise run `pnpm strings:gen`.
 *   3. Reference it in JSX as `{t(strings.feature.key)}`.
 *   4. For interpolation, use i18next's `{{var}}` placeholders in the JSON
 *      and call `t(strings.feature.key, { var: value })`. Do NOT splice
 *      values with template literals — that hides the final shape from
 *      translators.
 *   5. Commit `en.json` AND `strings.generated.ts` together. CI runs
 *      `pnpm strings:check` and fails the PR if you forget.
 *
 * ## What NOT to put here
 *
 *   - Test IDs (those go in `selectors.ts`).
 *   - Non-user-facing strings (API URLs, log messages, internal enum codes).
 */
export { strings, type Strings } from "./strings.generated";
