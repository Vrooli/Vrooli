/**
 * Typed `t()` via react-i18next module augmentation.
 *
 * This file teaches TypeScript the shape of our i18n catalog so calls like
 * `t("not.a.real.key")` become compile-time errors instead of silent
 * runtime fallbacks (which would otherwise just render the key string back).
 *
 * The catalog shape comes from the generated `strings` registry (built from
 * `src/i18n/locales/en/*.json`), so a key added to an English namespace file is
 * known to `t()` as soon as the registry regenerates (the vite plugin does that
 * on dev start, on catalog change and on every build; `pnpm strings:gen` does it
 * by hand). Both ways of writing a key are then equally safe:
 *   - `t(strings.feature.key)` yields a literal type that is a key of the resource.
 *   - `t("feature.typo")` errors because the literal is not a key of the resource.
 *
 * The registry omits sentinel keys (any segment starting with `_`), so those can
 * never go through `t()` either.
 *
 * One namespace (`translation`) is kept: every UI scenario starts small, and
 * ownership is split by file, not by i18next namespace.
 */
import "i18next";
import type { Catalog } from "./i18n/catalog";

export type Translation = Catalog;

declare module "i18next" {
  interface CustomTypeOptions {
    defaultNS: "translation";
    // Mirrors the runtime `returnNull: false` set in `i18n/index.ts`. Without
    // this, `t()` returns `string | null` and consumers get spurious null-checks.
    returnNull: false;
    resources: { translation: Translation };
  }
}
