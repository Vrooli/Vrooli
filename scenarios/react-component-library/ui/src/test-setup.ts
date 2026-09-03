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
import { cleanup } from "@testing-library/react";
import { afterEach, beforeEach, vi } from "vitest";
import { i18n } from "./i18n";
import { configureTestProviders } from "@vrooli/api-base/testing";
import { ThemeProvider } from "./components/theme/ThemeProvider";
import { createElement } from "react";

// jsdom does not implement matchMedia, while the released overlay core uses it
// to read reduced-motion preference during render. Install a neutral browser
// contract once; tests that exercise breakpoint changes replace it locally.
if (typeof window.matchMedia !== "function") {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: (query: string): MediaQueryList => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: () => undefined,
      removeEventListener: () => undefined,
      addListener: () => undefined,
      removeListener: () => undefined,
      dispatchEvent: () => false,
    }),
  });
}

// useScrollLock restores the captured offset on teardown. jsdom deliberately
// omits scrolling, so provide the browser seam without mutating layout state.
Object.defineProperty(window, "scrollTo", {
  configurable: true,
  writable: true,
  value: () => undefined,
});

configureTestProviders((children) => createElement(ThemeProvider, null, children));

let consoleError: ReturnType<typeof vi.spyOn>;
let consoleWarn: ReturnType<typeof vi.spyOn>;

beforeEach(async () => {
  window.localStorage.clear();
  await i18n.changeLanguage("cimode");
  consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
  consoleWarn = vi.spyOn(console, "warn").mockImplementation(() => {});
});

afterEach(() => {
  // Unmount every rendered tree before asserting on console output.
  //
  // The suite runs single-threaded, so one jsdom document is shared by every
  // test file in the run. Without this, a file that does not register its own
  // `cleanup()` leaks its DOM into every file after it, and the collision
  // surfaces as "Found multiple elements by ..." in an unrelated test —
  // 8 of 25 component files were relying on their neighbours to tidy up.
  // Cleaning here rather than in a separate hook keeps unmount-time console
  // errors inside the assertion below instead of escaping it.
  cleanup();

  const errorCalls = consoleError.mock.calls;
  const warnCalls = consoleWarn.mock.calls;
  consoleError.mockRestore();
  consoleWarn.mockRestore();

  // jsdom's CSS parser rejects the modern `color-mix()` expressions shipped
  // by the released library stylesheet. The browser still receives the
  // stylesheet unchanged; this is only the diagnostic emitted while jsdom
  // attaches the style element. Keep the strict console contract for every
  // other error and warning.
  const unexpectedErrorCalls = errorCalls.filter(
    (args) => !args.some((arg) => String(arg).includes("Could not parse CSS stylesheet")),
  );

  if (unexpectedErrorCalls.length > 0 || warnCalls.length > 0) {
    const formatted = [
      ...unexpectedErrorCalls.map((args) => `console.error: ${args.map(String).join(" ")}`),
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

// jsdom implements neither PointerEvent nor the pointer-capture methods, so
// `fireEvent.pointerMove` silently degrades to a bare Event carrying no
// coordinates and no pointerId — every pointer-driven assertion then passes
// vacuously or fails for the wrong reason. Gesture assets are tested through
// exactly that path, so the environment supplies the missing surface rather
// than each test faking it.
//
// `timeStamp` is writable here on purpose: it is read-only in a real browser,
// but velocity is a function of it, and a test that cannot control the clock
// cannot distinguish a flick from a slow drag.
if (typeof globalThis.PointerEvent === "undefined") {
  class PointerEventPolyfill extends MouseEvent {
    readonly pointerId: number;
    readonly pointerType: string;
    readonly isPrimary: boolean;
    readonly width: number;
    readonly height: number;
    readonly pressure: number;

    constructor(type: string, init: PointerEventInit & { timeStamp?: number } = {}) {
      super(type, init);
      this.pointerId = init.pointerId ?? 0;
      this.pointerType = init.pointerType ?? "";
      this.isPrimary = init.isPrimary ?? true;
      this.width = init.width ?? 1;
      this.height = init.height ?? 1;
      this.pressure = init.pressure ?? 0;
      if (init.timeStamp !== undefined) {
        Object.defineProperty(this, "timeStamp", { value: init.timeStamp, configurable: true });
      }
    }
  }
  Object.defineProperty(globalThis, "PointerEvent", {
    value: PointerEventPolyfill,
    configurable: true,
    writable: true,
  });
}

// Capture is a no-op rather than a throw: the library guards these calls, and a
// guard that is never exercised because the method is missing proves nothing.
for (const method of ["setPointerCapture", "releasePointerCapture"] as const) {
  if (typeof Element.prototype[method] !== "function") {
    Object.defineProperty(Element.prototype, method, {
      value: () => {},
      configurable: true,
      writable: true,
    });
  }
}
if (typeof Element.prototype.hasPointerCapture !== "function") {
  Object.defineProperty(Element.prototype, "hasPointerCapture", {
    value: () => false,
    configurable: true,
    writable: true,
  });
}
