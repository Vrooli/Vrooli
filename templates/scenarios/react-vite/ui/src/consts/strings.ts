/**
 * Vrooli Ascension string registry — typed key paths
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
 * // t("app.title") === "{{SCENARIO_DISPLAY_NAME}}" / "シナリオテンプレート" / …
 * ```
 *
 * Why this shape (instead of inlining `t("app.title")` directly):
 *
 * 1. **Type safety.** `strings.app.titel` is a compile-time error.
 *    `t("app.titel")` is a runtime fallback to the key string.
 * 2. **Refactor safety.** Renaming a key in en.json triggers TypeScript
 *    errors at every callsite when the registry is regenerated.
 * 3. **Discoverability.** "Where is this string referenced?" is one grep
 *    for `strings.feature.key`, not a fuzzy search through `t("…")` calls.
 * 4. **Test stability.** Tests can compare against the same key paths the
 *    component uses, without ever asserting on translated copy.
 *
 * Adding a string:
 *   1. Add the key to `i18n/locales/en.json` (and every other locale).
 *   2. Reference it in JSX as `{t(strings.feature.key)}`.
 *   3. For interpolation, use i18next's `{{var}}` placeholders in the JSON
 *      and call `t(strings.feature.key, { var: value })`. Do NOT splice
 *      values with template literals — that hides the final shape from
 *      translators.
 *
 * What NOT to put here:
 *   - Test IDs (those go in `selectors.ts`).
 *   - Non-user-facing strings (API URLs, log messages, internal enum codes).
 */
import en from "../i18n/locales/en.json";

type StringCatalog = Record<string, unknown>;

type KeyTree<T, Prefix extends string = ""> = {
  [K in keyof T & string]: T[K] extends string
    ? `${Prefix}${K}`
    : T[K] extends StringCatalog
      ? KeyTree<T[K], `${Prefix}${K}.`>
      : never;
};

const buildKeys = (
  catalog: StringCatalog,
  prefix = "",
): Record<string, unknown> => {
  const result: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(catalog)) {
    const path = prefix ? `${prefix}.${key}` : key;
    if (typeof value === "string") {
      result[key] = path;
    } else if (value && typeof value === "object") {
      result[key] = buildKeys(value as StringCatalog, path);
    }
  }
  return result;
};

export const strings = buildKeys(en) as KeyTree<typeof en>;
export type Strings = typeof strings;
