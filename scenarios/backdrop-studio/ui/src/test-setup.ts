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
import { afterEach, beforeEach, vi } from "vitest";
import { i18n } from "./i18n";

let consoleError: ReturnType<typeof vi.spyOn>;
let consoleWarn: ReturnType<typeof vi.spyOn>;

beforeEach(async () => {
  window.localStorage.clear();
  await i18n.changeLanguage("cimode");
  consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
  consoleWarn = vi.spyOn(console, "warn").mockImplementation(() => {});
});

afterEach(() => {
  const errorCalls = consoleError.mock.calls;
  const warnCalls = consoleWarn.mock.calls;
  consoleError.mockRestore();
  consoleWarn.mockRestore();

  if (errorCalls.length > 0 || warnCalls.length > 0) {
    const formatted = [
      ...errorCalls.map((args) => `console.error: ${args.map(String).join(" ")}`),
      ...warnCalls.map((args) => `console.warn: ${args.map(String).join(" ")}`),
    ].join("\n");
    throw new Error(`Unexpected console output during test:\n${formatted}`);
  }
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

// jsdom implements neither `URL.createObjectURL` nor `URL.revokeObjectURL`.
//
// The studio pages hand candidate PNG bytes to an <img> through an object URL
// and revoke it on unmount, which is the correct production behaviour — a
// variation grid that leaked a megabyte-scale blob per re-render walks a
// comparison session into hundreds of megabytes with nothing visibly wrong. So
// the environment is given the two functions rather than the hook being taught
// to work without them: a guard in the hook would make the leak invisible in
// exactly the runtime that has the problem.
if (typeof URL.createObjectURL !== "function") {
  let issued = 0;
  URL.createObjectURL = () => `blob:test-${++issued}`;
  URL.revokeObjectURL = () => {};
}
