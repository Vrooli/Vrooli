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
 *
 * This file owns *setup-side-effects only*. Helpers tests reach for
 * (e.g. `interp`) live in `@/test-utils` so the setup-file remains a
 * pure registration surface — moving them out keeps test-setup.ts
 * single-purpose and makes the import graph for individual tests
 * easier to follow.
 */
import "@testing-library/jest-dom/vitest";
import { beforeEach, vi } from "vitest";
import { i18n } from "./i18n";

beforeEach(async () => {
  window.localStorage.clear();
  await i18n.changeLanguage("cimode");
});

// Process-wide spy by intent — axe-core probes canvas during every a11y
// run, and jsdom doesn't implement HTMLCanvasElement.getContext. We do
// NOT call vi.restoreAllMocks() between tests because the canvas mock
// is part of the test environment, not per-test arrangement. If a
// future test legitimately needs a real canvas surface, it should
// explicitly `vi.spyOn(HTMLCanvasElement.prototype, "getContext")` in
// its own beforeEach and restore it on teardown — opt-in override
// rather than process-wide unwiring.
vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
