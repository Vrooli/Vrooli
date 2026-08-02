/**
 * Typed `t()` via react-i18next module augmentation.
 *
 * This file teaches TypeScript the shape of our i18n catalog so calls like
 * `t("not.a.real.key")` become compile-time errors instead of silent
 * runtime fallbacks (which would otherwise just render the key string back).
 *
 * Why this is belt-and-suspenders:
 *   1. The typed `strings` registry in `src/consts/strings.generated.ts`
 *      already prevents bad paths from *entering* `t()` at the recommended
 *      callsite shape `t(strings.feature.key)`. The registry is
 *      auto-generated from `en.json` so its keys can't drift.
 *   2. But hand-written `t("feature.key")` (a string literal, not a
 *      registry reference) bypasses the registry. Without this
 *      augmentation, `t("feature.typo")` would type-check.
 *   3. With the augmentation below, both paths are equivalently safe:
 *      - `t(strings.feature.key)` works because the registry produces
 *        literal types like `"feature.key"` that satisfy the augmented
 *        key union.
 *      - `t("feature.typo")` errors because the literal isn't a key of
 *        the resource.
 *
 * If you add a new key, the workflow is:
 *   1. Edit `src/i18n/locales/en.json`.
 *   2. The vite plugin regenerates `strings.generated.ts` automatically.
 *   3. TypeScript instantly knows about the new key via this file (which
 *      imports `en.json` directly), so callsites can reference it.
 *
 * Adding more namespaces would mean:
 *   1. Add a separate JSON file (e.g., `errors.json`).
 *   2. Add another resource entry below.
 *   3. Update i18n/index.ts to register it as a separate namespace.
 *   4. Decide whether to keep `defaultNS: "translation"`.
 *
 * Right now we keep one namespace because every UI scenario starts small.
 */
import "i18next";
import en from "./i18n/locales/en.json";

/**
 * Strip catalog *sentinel* keys (any segment whose first character is `_`,
 * e.g. `_comment`) recursively. Sentinels live in catalogs as documentation
 * but never go through `t()` — `gen-strings.mjs` excludes them from the
 * typed `strings.*` registry and `eslint-rules/no-unused-keys.js` skips
 * them in the audit. Mirroring the rule in the augmentation makes
 * `t("_comment")` a TypeScript error too — closing the only path by which
 * sentinels could leak into runtime translation lookups.
 *
 * `K extends \`_${string}\`` only matches segments that *start* with `_`,
 * so CLDR plural variants like `refreshCount_one` (mid-key underscore) are
 * preserved correctly. The recursion handles future nested sentinels too.
 */
type StripSentinels<T> = T extends string
  ? T
  : T extends readonly unknown[]
    ? T
    : T extends object
      ? {
          [K in keyof T as K extends `_${string}` ? never : K]: StripSentinels<T[K]>;
        }
      : T;

export type Translation = StripSentinels<typeof en>;

declare module "i18next" {
  interface CustomTypeOptions {
    defaultNS: "translation";
    // Mirrors the runtime `returnNull: false` set in `i18n/index.ts`. Without
    // this, `t()` returns `string | null` and consumers get spurious null-checks.
    returnNull: false;
    resources: { translation: Translation };
  }
}
