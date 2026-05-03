/**
 * Vitest setup file
 *
 * 1. Registers @testing-library/jest-dom matchers (`.toBeInTheDocument()`, …).
 * 2. Defaults i18next into `cimode` before every test so `t('app.title')`
 *    returns the *key* (`"app.title"`), not translated copy. This makes
 *    component tests robust against any wording change in any locale —
 *    assertions reference the typed `strings.*` registry, not brittle
 *    string literals.
 *
 * Tests that specifically need to validate translation behaviour (locale
 * switcher, real English/Japanese rendering, etc.) opt back via their own
 * `beforeEach`:
 *
 *   beforeEach(async () => { await setLocale("en"); });
 *
 * The setup-file `beforeEach` runs first, so per-file overrides win.
 */
import "@testing-library/jest-dom/vitest";
import { beforeEach, vi } from "vitest";
import { i18n } from "./i18n";

beforeEach(async () => {
  window.localStorage.clear();
  await i18n.changeLanguage("cimode");
});

// jsdom doesn't implement HTMLCanvasElement.getContext, which axe-core probes
// during color-contrast / icon-ligature checks. axe-core falls back gracefully,
// but the unhandled call spams stderr on every run. Returning null matches
// axe-core's "canvas unavailable" branch — same behavior, no noise.
vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);

/**
 * Substitute `{{name}}` placeholders in a catalog string with concrete values
 * for assertion. Centralised so an eventual ICU-MessageFormat migration is a
 * one-file change instead of touching every plural test.
 *
 * Use only in tests that explicitly verify the i18n pipeline end-to-end —
 * `cimode`-default tests assert on the raw key path via `strings.x.y` and
 * never need this helper.
 */
export const interp = (
  template: string,
  vars: Record<string, string | number>,
): string =>
  template.replace(/\{\{(\w+)\}\}/g, (_match, name: string) => {
    const value = vars[name];
    if (value === undefined) {
      throw new Error(
        `interp(): template expects '{{${name}}}' but no value was provided`,
      );
    }
    return String(value);
  });
