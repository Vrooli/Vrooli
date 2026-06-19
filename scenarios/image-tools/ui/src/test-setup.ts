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

// jsdom has no ResizeObserver — the WorkspaceCanvas overlay path (crop box /
// detection boxes) constructs one to keep the rendered-image box measured. A
// no-op stub keeps that effect from throwing; overlay-geometry tests pass the
// natural/client sizes directly, so a live observer isn't needed. Assigned on
// globalThis (not vi.stubGlobal) so per-test `vi.unstubAllGlobals()` calls
// don't strip it mid-suite.
class ResizeObserverStub {
  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
}
globalThis.ResizeObserver = ResizeObserverStub;

// jsdom has no URL.createObjectURL/revokeObjectURL — the Smart-Select surface
// (and any future image-loading feature) creates an object URL to preview the
// loaded File. Deterministic no-op stubs keep that effect from throwing; the
// blob-key→URL helper (api/client.blobUrl) is a plain string path and doesn't
// touch these. Assigned on the prototype (not vi.stubGlobal) so per-test
// `vi.unstubAllGlobals()` calls don't strip them mid-suite.
if (typeof URL.createObjectURL !== "function") {
  URL.createObjectURL = () => "blob:mock/object-url";
}
if (typeof URL.revokeObjectURL !== "function") {
  URL.revokeObjectURL = () => {};
}
